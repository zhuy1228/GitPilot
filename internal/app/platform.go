package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// --- 平台 API 通用 ---

// PlatformAPI 统一平台接口
type PlatformAPI interface {
	// CreateRepo 创建仓库，返回 clone URL
	CreateRepo(name, description string, private bool) (string, error)
	// ListRepos 列出当前用户的所有仓库
	ListRepos() ([]RepoInfo, error)
	// ListReleases 获取所有 Release
	ListReleases(owner, repo string) ([]ReleaseInfo, error)
	// CreateRelease 创建 Release
	CreateRelease(owner, repo string, release ReleaseCreate) (*ReleaseInfo, error)
	// UploadAsset 上传附件到指定 Release
	UploadAsset(owner, repo string, releaseID int64, fileName string, data io.Reader) error
	// ListLabels 获取仓库标签
	ListLabels(owner, repo string) ([]LabelInfo, error)
	// CreateLabel 创建标签
	CreateLabel(owner, repo string, label LabelInfo) error
	// ListMilestones 获取里程碑
	ListMilestones(owner, repo string) ([]MilestoneInfo, error)
	// CreateMilestone 创建里程碑
	CreateMilestone(owner, repo string, ms MilestoneInfo) (*MilestoneInfo, error)
	// ListIssues 获取工单
	ListIssues(owner, repo string) ([]IssueInfo, error)
	// CreateIssue 创建工单
	CreateIssue(owner, repo string, issue IssueInfo, milestoneMap map[string]int64) error
	// ListPullRequests 获取合并请求
	ListPullRequests(owner, repo string) ([]PullRequestInfo, error)
	// GetPlatformName 平台名称
	GetPlatformName() string
}

// RepoInfo 仓库信息
type RepoInfo struct {
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Description string `json:"description"`
	CloneURL    string `json:"cloneUrl"`
	SSHURL      string `json:"sshUrl"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	Empty       bool   `json:"empty"`
	UpdatedAt   string `json:"updatedAt"`
}

// ReleaseInfo 通用 Release 信息
type ReleaseInfo struct {
	ID          int64       `json:"id"`
	TagName     string      `json:"tagName"`
	Name        string      `json:"name"`
	Body        string      `json:"body"`
	Draft       bool        `json:"draft"`
	Prerelease  bool        `json:"prerelease"`
	CreatedAt   string      `json:"createdAt"`
	PublishedAt string      `json:"publishedAt"`
	Assets      []AssetInfo `json:"assets"`
}

// AssetInfo Release 附件
type AssetInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"downloadUrl"`
}

// ReleaseCreate 创建 Release 的参数
type ReleaseCreate struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// --- GitHub API ---

type GitHubAPI struct {
	Token    string
	Username string
	client   *http.Client
}

func NewGitHubAPI(username, token string) *GitHubAPI {
	return &GitHubAPI{
		Token:    token,
		Username: username,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (g *GitHubAPI) GetPlatformName() string { return "github" }

func (g *GitHubAPI) doRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	return g.client.Do(req)
}

func (g *GitHubAPI) CreateRepo(name, description string, private bool) (string, error) {
	payload := map[string]interface{}{
		"name":        name,
		"description": description,
		"private":     private,
	}
	resp, err := g.doRequest("POST", "https://api.github.com/user/repos", payload)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("创建仓库失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		CloneURL string `json:"clone_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	return result.CloneURL, nil
}

func (g *GitHubAPI) ListReleases(owner, repo string) ([]ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=100", owner, repo)
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取 Releases 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var ghReleases []struct {
		ID          int64  `json:"id"`
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		CreatedAt   string `json:"created_at"`
		PublishedAt string `json:"published_at"`
		Assets      []struct {
			ID                 int64  `json:"id"`
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ghReleases); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var releases []ReleaseInfo
	for _, r := range ghReleases {
		rel := ReleaseInfo{
			ID:          r.ID,
			TagName:     r.TagName,
			Name:        r.Name,
			Body:        r.Body,
			Draft:       r.Draft,
			Prerelease:  r.Prerelease,
			CreatedAt:   r.CreatedAt,
			PublishedAt: r.PublishedAt,
		}
		for _, a := range r.Assets {
			rel.Assets = append(rel.Assets, AssetInfo{
				ID:          a.ID,
				Name:        a.Name,
				Size:        a.Size,
				DownloadURL: a.BrowserDownloadURL,
			})
		}
		releases = append(releases, rel)
	}
	return releases, nil
}

func (g *GitHubAPI) CreateRelease(owner, repo string, release ReleaseCreate) (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	resp, err := g.doRequest("POST", url, release)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建 Release 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID      int64  `json:"id"`
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &ReleaseInfo{ID: result.ID, TagName: result.TagName, Name: result.Name}, nil
}

func (g *GitHubAPI) UploadAsset(owner, repo string, releaseID int64, fileName string, data io.Reader) error {
	url := fmt.Sprintf("https://uploads.github.com/repos/%s/%s/releases/%d/assets?name=%s", owner, repo, releaseID, fileName)

	// GitHub upload API 使用 raw binary body
	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("上传附件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传附件失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (g *GitHubAPI) ListRepos() ([]RepoInfo, error) {
	var allRepos []RepoInfo
	page := 1
	for {
		url := fmt.Sprintf("https://api.github.com/user/repos?per_page=100&page=%d&affiliation=owner", page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		var repos []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			CloneURL    string `json:"clone_url"`
			SSHURL      string `json:"ssh_url"`
			Private     bool   `json:"private"`
			Fork        bool   `json:"fork"`
			UpdatedAt   string `json:"updated_at"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("获取仓库列表失败 (HTTP %d): %s", resp.StatusCode, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		resp.Body.Close()
		if len(repos) == 0 {
			break
		}
		for _, r := range repos {
			allRepos = append(allRepos, RepoInfo{
				Name:        r.Name,
				FullName:    r.FullName,
				Description: r.Description,
				CloneURL:    r.CloneURL,
				SSHURL:      r.SSHURL,
				Private:     r.Private,
				Fork:        r.Fork,
				UpdatedAt:   r.UpdatedAt,
			})
		}
		if len(repos) < 100 {
			break
		}
		page++
	}
	return allRepos, nil
}

// --- Gitea API ---

type GiteaAPI struct {
	BaseURL  string
	Token    string
	Username string
	client   *http.Client
}

func NewGiteaAPI(baseURL, username, token string) *GiteaAPI {
	baseURL = strings.TrimRight(baseURL, "/")
	return &GiteaAPI{
		BaseURL:  baseURL,
		Token:    token,
		Username: username,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (g *GiteaAPI) GetPlatformName() string { return "gitea" }

func (g *GiteaAPI) doRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}
	return g.client.Do(req)
}

func (g *GiteaAPI) CreateRepo(name, description string, private bool) (string, error) {
	payload := map[string]interface{}{
		"name":        name,
		"description": description,
		"private":     private,
	}
	url := g.BaseURL + "/api/v1/user/repos"
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("创建仓库失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		CloneURL string `json:"clone_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	return result.CloneURL, nil
}

func (g *GiteaAPI) ListReleases(owner, repo string) ([]ReleaseInfo, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases?limit=50", g.BaseURL, owner, repo)
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取 Releases 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var giteaReleases []struct {
		ID          int64  `json:"id"`
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		Draft       bool   `json:"draft"`
		Prerelease  bool   `json:"prerelease"`
		CreatedAt   string `json:"created_at"`
		PublishedAt string `json:"published_at"`
		Assets      []struct {
			ID                 int64  `json:"id"`
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&giteaReleases); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var releases []ReleaseInfo
	for _, r := range giteaReleases {
		rel := ReleaseInfo{
			ID:          r.ID,
			TagName:     r.TagName,
			Name:        r.Name,
			Body:        r.Body,
			Draft:       r.Draft,
			Prerelease:  r.Prerelease,
			CreatedAt:   r.CreatedAt,
			PublishedAt: r.PublishedAt,
		}
		for _, a := range r.Assets {
			rel.Assets = append(rel.Assets, AssetInfo{
				ID:          a.ID,
				Name:        a.Name,
				Size:        a.Size,
				DownloadURL: a.BrowserDownloadURL,
			})
		}
		releases = append(releases, rel)
	}
	return releases, nil
}

func (g *GiteaAPI) CreateRelease(owner, repo string, release ReleaseCreate) (*ReleaseInfo, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases", g.BaseURL, owner, repo)
	resp, err := g.doRequest("POST", url, release)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建 Release 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID      int64  `json:"id"`
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &ReleaseInfo{ID: result.ID, TagName: result.TagName, Name: result.Name}, nil
}

func (g *GiteaAPI) UploadAsset(owner, repo string, releaseID int64, fileName string, data io.Reader) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/releases/%d/assets?name=%s", g.BaseURL, owner, repo, releaseID, fileName)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("attachment", fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, data); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if g.Token != "" {
		req.Header.Set("Authorization", "token "+g.Token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("上传附件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传附件失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (g *GiteaAPI) ListRepos() ([]RepoInfo, error) {
	var allRepos []RepoInfo
	page := 1
	for {
		url := fmt.Sprintf("%s/api/v1/user/repos?page=%d&limit=50", g.BaseURL, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		var repos []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			CloneURL    string `json:"clone_url"`
			SSHURL      string `json:"ssh_url"`
			Private     bool   `json:"private"`
			Fork        bool   `json:"fork"`
			Empty       bool   `json:"empty"`
			UpdatedAt   string `json:"updated_at"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("获取仓库列表失败 (HTTP %d): %s", resp.StatusCode, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		resp.Body.Close()
		if len(repos) == 0 {
			break
		}
		for _, r := range repos {
			allRepos = append(allRepos, RepoInfo{
				Name:        r.Name,
				FullName:    r.FullName,
				Description: r.Description,
				CloneURL:    r.CloneURL,
				SSHURL:      r.SSHURL,
				Private:     r.Private,
				Fork:        r.Fork,
				Empty:       r.Empty,
				UpdatedAt:   r.UpdatedAt,
			})
		}
		if len(repos) < 50 {
			break
		}
		page++
	}
	return allRepos, nil
}

// --- Gitee API ---

type GiteeAPI struct {
	Token    string
	Username string
	client   *http.Client
}

func NewGiteeAPI(username, token string) *GiteeAPI {
	return &GiteeAPI{
		Token:    token,
		Username: username,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (g *GiteeAPI) GetPlatformName() string { return "gitee" }

func (g *GiteeAPI) doRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return g.client.Do(req)
}

func (g *GiteeAPI) CreateRepo(name, description string, private bool) (string, error) {
	payload := map[string]interface{}{
		"access_token": g.Token,
		"name":         name,
		"description":  description,
		"private":      private,
	}
	resp, err := g.doRequest("POST", "https://gitee.com/api/v5/user/repos", payload)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("创建仓库失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		HtmlURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	// Gitee clone URL = html_url + ".git"
	return result.HtmlURL + ".git", nil
}

func (g *GiteeAPI) ListReleases(owner, repo string) ([]ReleaseInfo, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/releases?access_token=%s&per_page=100", owner, repo, g.Token)
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取 Releases 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var giteeReleases []struct {
		ID         int64  `json:"id"`
		TagName    string `json:"tag_name"`
		Name       string `json:"name"`
		Body       string `json:"body"`
		Prerelease bool   `json:"prerelease"`
		CreatedAt  string `json:"created_at"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&giteeReleases); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var releases []ReleaseInfo
	for _, r := range giteeReleases {
		rel := ReleaseInfo{
			ID:         r.ID,
			TagName:    r.TagName,
			Name:       r.Name,
			Body:       r.Body,
			Prerelease: r.Prerelease,
			CreatedAt:  r.CreatedAt,
		}
		for _, a := range r.Assets {
			rel.Assets = append(rel.Assets, AssetInfo{
				Name:        a.Name,
				DownloadURL: a.BrowserDownloadURL,
			})
		}
		releases = append(releases, rel)
	}
	return releases, nil
}

func (g *GiteeAPI) CreateRelease(owner, repo string, release ReleaseCreate) (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/releases", owner, repo)
	payload := map[string]interface{}{
		"access_token": g.Token,
		"tag_name":     release.TagName,
		"name":         release.Name,
		"body":         release.Body,
		"prerelease":   release.Prerelease,
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建 Release 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID      int64  `json:"id"`
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &ReleaseInfo{ID: result.ID, TagName: result.TagName, Name: result.Name}, nil
}

func (g *GiteeAPI) UploadAsset(owner, repo string, releaseID int64, fileName string, data io.Reader) error {
	// Gitee 目前不支持通过 API 上传 Release 附件
	return fmt.Errorf("Gitee 暂不支持通过 API 上传 Release 附件")
}

func (g *GiteeAPI) ListRepos() ([]RepoInfo, error) {
	var allRepos []RepoInfo
	page := 1
	for {
		url := fmt.Sprintf("https://gitee.com/api/v5/user/repos?access_token=%s&type=personal&per_page=100&page=%d", g.Token, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		var repos []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			HtmlURL     string `json:"html_url"`
			SSHURL      string `json:"ssh_url"`
			Private     bool   `json:"private"`
			Fork        bool   `json:"fork"`
			UpdatedAt   string `json:"updated_at"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("获取仓库列表失败 (HTTP %d): %s", resp.StatusCode, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		resp.Body.Close()
		if len(repos) == 0 {
			break
		}
		for _, r := range repos {
			allRepos = append(allRepos, RepoInfo{
				Name:        r.Name,
				FullName:    r.FullName,
				Description: r.Description,
				CloneURL:    r.HtmlURL + ".git",
				SSHURL:      r.SSHURL,
				Private:     r.Private,
				Fork:        r.Fork,
				UpdatedAt:   r.UpdatedAt,
			})
		}
		if len(repos) < 100 {
			break
		}
		page++
	}
	return allRepos, nil
}

// --- GitLab API ---

type GitLabAPI struct {
	BaseURL  string
	Token    string
	Username string
	client   *http.Client
}

func NewGitLabAPI(baseURL, username, token string) *GitLabAPI {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &GitLabAPI{
		BaseURL:  baseURL,
		Token:    token,
		Username: username,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (g *GitLabAPI) GetPlatformName() string { return "gitlab" }

// projectPath 返回 URL 编码的 owner/repo 路径，用于 GitLab API
func (g *GitLabAPI) projectPath(owner, repo string) string {
	return strings.ReplaceAll(owner+"/"+repo, "/", "%2F")
}

func (g *GitLabAPI) doRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if g.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.Token)
	}
	return g.client.Do(req)
}

func (g *GitLabAPI) CreateRepo(name, description string, private bool) (string, error) {
	visibility := "public"
	if private {
		visibility = "private"
	}
	payload := map[string]interface{}{
		"name":        name,
		"description": description,
		"visibility":  visibility,
	}
	url := g.BaseURL + "/api/v4/projects"
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("创建仓库失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		HTTPURLToRepo string `json:"http_url_to_repo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	return result.HTTPURLToRepo, nil
}

func (g *GitLabAPI) ListReleases(owner, repo string) ([]ReleaseInfo, error) {
	pp := g.projectPath(owner, repo)
	url := fmt.Sprintf("%s/api/v4/projects/%s/releases?per_page=100", g.BaseURL, pp)
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取 Releases 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var glReleases []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
		ReleasedAt  string `json:"released_at"`
		Assets      struct {
			Links []struct {
				ID       int64  `json:"id"`
				Name     string `json:"name"`
				URL      string `json:"url"`
				LinkType string `json:"link_type"`
			} `json:"links"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&glReleases); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var releases []ReleaseInfo
	for _, r := range glReleases {
		rel := ReleaseInfo{
			TagName:     r.TagName,
			Name:        r.Name,
			Body:        r.Description,
			CreatedAt:   r.CreatedAt,
			PublishedAt: r.ReleasedAt,
		}
		for _, link := range r.Assets.Links {
			rel.Assets = append(rel.Assets, AssetInfo{
				ID:          link.ID,
				Name:        link.Name,
				DownloadURL: link.URL,
			})
		}
		releases = append(releases, rel)
	}
	return releases, nil
}

func (g *GitLabAPI) CreateRelease(owner, repo string, release ReleaseCreate) (*ReleaseInfo, error) {
	pp := g.projectPath(owner, repo)
	url := fmt.Sprintf("%s/api/v4/projects/%s/releases", g.BaseURL, pp)
	payload := map[string]interface{}{
		"tag_name":    release.TagName,
		"name":        release.Name,
		"description": release.Body,
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("创建 Release 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	return &ReleaseInfo{TagName: result.TagName, Name: result.Name}, nil
}

func (g *GitLabAPI) UploadAsset(owner, repo string, releaseID int64, fileName string, data io.Reader) error {
	// GitLab Release 附件通过 project uploads + release link 方式实现
	pp := g.projectPath(owner, repo)

	// 1. 上传文件到项目
	uploadURL := fmt.Sprintf("%s/api/v4/projects/%s/uploads", g.BaseURL, pp)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", fileName)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, data); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", uploadURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if g.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", g.Token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("上传附件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传附件失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var uploadResult struct {
		URL      string `json:"url"`
		Markdown string `json:"markdown"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResult); err != nil {
		return fmt.Errorf("解析上传结果失败: %w", err)
	}

	// 注意：GitLab uploads API 返回的 URL 是相对路径，不自动关联到 Release
	// 如需关联到 Release，需要调用 Release Links API（需要 tag_name 而非 releaseID）
	return nil
}

func (g *GitLabAPI) ListRepos() ([]RepoInfo, error) {
	var allRepos []RepoInfo
	page := 1
	for {
		url := fmt.Sprintf("%s/api/v4/projects?membership=true&owned=true&per_page=100&page=%d", g.BaseURL, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %w", err)
		}
		var repos []struct {
			ID                int64  `json:"id"`
			Name              string `json:"name"`
			PathWithNamespace string `json:"path_with_namespace"`
			Description       string `json:"description"`
			HTTPURLToRepo     string `json:"http_url_to_repo"`
			SSHURLToRepo      string `json:"ssh_url_to_repo"`
			Visibility        string `json:"visibility"`
			ForkedFromProject *struct {
				ID int64 `json:"id"`
			} `json:"forked_from_project"`
			Empty     bool   `json:"empty_repo"`
			UpdatedAt string `json:"last_activity_at"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("获取仓库列表失败 (HTTP %d): %s", resp.StatusCode, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		resp.Body.Close()
		if len(repos) == 0 {
			break
		}
		for _, r := range repos {
			allRepos = append(allRepos, RepoInfo{
				Name:        r.Name,
				FullName:    r.PathWithNamespace,
				Description: r.Description,
				CloneURL:    r.HTTPURLToRepo,
				SSHURL:      r.SSHURLToRepo,
				Private:     r.Visibility == "private",
				Fork:        r.ForkedFromProject != nil,
				Empty:       r.Empty,
				UpdatedAt:   r.UpdatedAt,
			})
		}
		if len(repos) < 100 {
			break
		}
		page++
	}
	return allRepos, nil
}

// --- 辅助方法 ---

// NewPlatformAPI 根据凭证创建平台 API 客户端
func NewPlatformAPI(platform, baseURL, username, token string) (PlatformAPI, error) {
	switch strings.ToLower(platform) {
	case "github":
		return NewGitHubAPI(username, token), nil
	case "gitea":
		if baseURL == "" {
			return nil, fmt.Errorf("Gitea 需要提供 Base URL")
		}
		return NewGiteaAPI(baseURL, username, token), nil
	case "gitee":
		return NewGiteeAPI(username, token), nil
	case "gitlab":
		return NewGitLabAPI(baseURL, username, token), nil
	default:
		return nil, fmt.Errorf("不支持的平台: %s", platform)
	}
}

// ParseOwnerRepo 从 remote URL 中解析 owner 和 repo
// 支持 HTTPS 和 SSH 格式
func ParseOwnerRepo(remoteURL string) (owner, repo string, err error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", "", fmt.Errorf("远程地址为空")
	}

	// SSH 格式: git@github.com:owner/repo.git
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("无法解析 SSH 地址: %s", remoteURL)
		}
		ownerRepo := strings.TrimSuffix(parts[1], ".git")
		segments := strings.SplitN(ownerRepo, "/", 2)
		if len(segments) != 2 {
			return "", "", fmt.Errorf("无法解析 owner/repo: %s", remoteURL)
		}
		return segments[0], segments[1], nil
	}

	// HTTPS 格式: https://github.com/owner/repo.git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	// 去掉协议
	urlPath := remoteURL
	if idx := strings.Index(urlPath, "://"); idx >= 0 {
		urlPath = urlPath[idx+3:]
	}
	// 去掉 host
	if idx := strings.Index(urlPath, "/"); idx >= 0 {
		urlPath = urlPath[idx+1:]
	}
	segments := strings.SplitN(urlPath, "/", 2)
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
		return "", "", fmt.Errorf("无法解析 owner/repo: %s", remoteURL)
	}
	return segments[0], segments[1], nil
}

// DownloadAsset 下载远程附件到内存
func DownloadAsset(url, token, platform string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// 某些平台私有仓库的附件需要认证
	if token != "" {
		switch strings.ToLower(platform) {
		case "github":
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "application/octet-stream")
		case "gitea":
			req.Header.Set("Authorization", "token "+token)
		case "gitlab":
			req.Header.Set("PRIVATE-TOKEN", token)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("下载失败 (HTTP %d)", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
