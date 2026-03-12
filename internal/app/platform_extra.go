package app

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// --- 迁移相关扩展类型 ---

// LabelInfo 标签
type LabelInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"` // 不含 # 前缀，如 "fc2929"
	Desc  string `json:"description,omitempty"`
}

// MilestoneInfo 里程碑
type MilestoneInfo struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"` // open / closed
	DueOn       string `json:"dueOn,omitempty"`
}

// IssueInfo 工单
type IssueInfo struct {
	ID             int64         `json:"id"`
	Number         int           `json:"number"`
	Title          string        `json:"title"`
	Body           string        `json:"body"`
	State          string        `json:"state"` // open / closed
	Labels         []string      `json:"labels,omitempty"`
	MilestoneTitle string        `json:"milestoneTitle,omitempty"`
	MilestoneID    int64         `json:"milestoneId,omitempty"`
	Comments       []CommentInfo `json:"comments,omitempty"`
	CreatedAt      string        `json:"createdAt,omitempty"`
}

// CommentInfo 评论
type CommentInfo struct {
	Body      string `json:"body"`
	User      string `json:"user,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// PullRequestInfo 合并请求（简化信息，用于记录式迁移）
type PullRequestInfo struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"` // open / closed / merged
	Head      string `json:"head"`
	Base      string `json:"base"`
	User      string `json:"user,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// ================ GitHub ================

func (g *GitHubAPI) ListLabels(owner, repo string) ([]LabelInfo, error) {
	var all []LabelInfo
	page := 1
	for {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/labels?per_page=100&page=%d", owner, repo, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		var items []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
			Desc  string `json:"description"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			all = append(all, LabelInfo{ID: it.ID, Name: it.Name, Color: it.Color, Desc: it.Desc})
		}
		if len(items) < 100 {
			break
		}
		page++
	}
	return all, nil
}

func (g *GitHubAPI) CreateLabel(owner, repo string, label LabelInfo) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/labels", owner, repo)
	payload := map[string]interface{}{
		"name":        label.Name,
		"color":       strings.TrimPrefix(label.Color, "#"),
		"description": label.Desc,
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (g *GitHubAPI) ListMilestones(owner, repo string) ([]MilestoneInfo, error) {
	var all []MilestoneInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("https://api.github.com/repos/%s/%s/milestones?state=%s&per_page=100&page=%d", owner, repo, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID          int64  `json:"id"`
				Number      int    `json:"number"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       string `json:"state"`
				DueOn       string `json:"due_on"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				all = append(all, MilestoneInfo{
					ID: it.ID, Title: it.Title, Description: it.Description, State: it.State, DueOn: it.DueOn,
				})
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GitHubAPI) CreateMilestone(owner, repo string, ms MilestoneInfo) (*MilestoneInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/milestones", owner, repo)
	payload := map[string]interface{}{
		"title":       ms.Title,
		"description": ms.Description,
		"state":       ms.State,
	}
	if ms.DueOn != "" {
		payload["due_on"] = ms.DueOn
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	var result struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return &MilestoneInfo{ID: int64(result.Number), Title: result.Title, State: ms.State}, nil
}

func (g *GitHubAPI) ListIssues(owner, repo string) ([]IssueInfo, error) {
	var all []IssueInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=%s&per_page=100&page=%d&direction=asc", owner, repo, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID     int64  `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				State  string `json:"state"`
				Labels []struct {
					Name string `json:"name"`
				} `json:"labels"`
				Milestone *struct {
					Title string `json:"title"`
				} `json:"milestone"`
				PullRequest *struct {
					URL string `json:"url"`
				} `json:"pull_request"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				// GitHub Issues API 也返回 PR，通过 pull_request 字段过滤
				if it.PullRequest != nil {
					continue
				}
				issue := IssueInfo{
					ID: it.ID, Number: it.Number, Title: it.Title, Body: it.Body,
					State: it.State, CreatedAt: it.CreatedAt,
				}
				for _, l := range it.Labels {
					issue.Labels = append(issue.Labels, l.Name)
				}
				if it.Milestone != nil {
					issue.MilestoneTitle = it.Milestone.Title
				}
				all = append(all, issue)
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GitHubAPI) CreateIssue(owner, repo string, issue IssueInfo, milestoneMap map[string]int64) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", owner, repo)
	payload := map[string]interface{}{
		"title":  issue.Title,
		"body":   issue.Body,
		"labels": issue.Labels,
	}
	if issue.MilestoneTitle != "" {
		if msID, ok := milestoneMap[issue.MilestoneTitle]; ok {
			payload["milestone"] = msID
		}
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	// 如果需要关闭 issue
	if issue.State == "closed" {
		var created struct {
			Number int `json:"number"`
		}
		json.NewDecoder(resp.Body).Decode(&created)
		closeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, created.Number)
		closeResp, err := g.doRequest("PATCH", closeURL, map[string]string{"state": "closed"})
		if err == nil {
			closeResp.Body.Close()
		}
	}
	return nil
}

func (g *GitHubAPI) ListPullRequests(owner, repo string) ([]PullRequestInfo, error) {
	var all []PullRequestInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls?state=%s&per_page=100&page=%d&direction=asc", owner, repo, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID     int64  `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				State  string `json:"state"`
				Merged bool   `json:"merged"`
				Head   struct {
					Ref string `json:"ref"`
				} `json:"head"`
				Base struct {
					Ref string `json:"ref"`
				} `json:"base"`
				User struct {
					Login string `json:"login"`
				} `json:"user"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				st := it.State
				if it.Merged {
					st = "merged"
				}
				all = append(all, PullRequestInfo{
					ID: it.ID, Number: it.Number, Title: it.Title, Body: it.Body,
					State: st, Head: it.Head.Ref, Base: it.Base.Ref,
					User: it.User.Login, CreatedAt: it.CreatedAt,
				})
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

// ================ Gitea ================

func (g *GiteaAPI) ListLabels(owner, repo string) ([]LabelInfo, error) {
	var all []LabelInfo
	page := 1
	for {
		url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels?page=%d&limit=50", g.BaseURL, owner, repo, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		var items []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
			Desc  string `json:"description"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			all = append(all, LabelInfo{ID: it.ID, Name: it.Name, Color: strings.TrimPrefix(it.Color, "#"), Desc: it.Desc})
		}
		if len(items) < 50 {
			break
		}
		page++
	}
	return all, nil
}

func (g *GiteaAPI) CreateLabel(owner, repo string, label LabelInfo) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", g.BaseURL, owner, repo)
	color := label.Color
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}
	payload := map[string]interface{}{
		"name":        label.Name,
		"color":       color,
		"description": label.Desc,
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (g *GiteaAPI) ListMilestones(owner, repo string) ([]MilestoneInfo, error) {
	var all []MilestoneInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/api/v1/repos/%s/%s/milestones?state=%s&page=%d&limit=50", g.BaseURL, owner, repo, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID          int64  `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       string `json:"state"`
				DueOn       string `json:"due_on"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				all = append(all, MilestoneInfo{
					ID: it.ID, Title: it.Title, Description: it.Description, State: it.State, DueOn: it.DueOn,
				})
			}
			if len(items) < 50 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GiteaAPI) CreateMilestone(owner, repo string, ms MilestoneInfo) (*MilestoneInfo, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/milestones", g.BaseURL, owner, repo)
	payload := map[string]interface{}{
		"title":       ms.Title,
		"description": ms.Description,
		"state":       ms.State,
	}
	if ms.DueOn != "" {
		payload["due_on"] = ms.DueOn
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	var result struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return &MilestoneInfo{ID: result.ID, Title: result.Title, State: ms.State}, nil
}

func (g *GiteaAPI) ListIssues(owner, repo string) ([]IssueInfo, error) {
	var all []IssueInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?state=%s&type=issues&page=%d&limit=50", g.BaseURL, owner, repo, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID     int64  `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				State  string `json:"state"`
				Labels []struct {
					Name string `json:"name"`
				} `json:"labels"`
				Milestone *struct {
					Title string `json:"title"`
				} `json:"milestone"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				issue := IssueInfo{
					ID: it.ID, Number: it.Number, Title: it.Title, Body: it.Body,
					State: it.State, CreatedAt: it.CreatedAt,
				}
				for _, l := range it.Labels {
					issue.Labels = append(issue.Labels, l.Name)
				}
				if it.Milestone != nil {
					issue.MilestoneTitle = it.Milestone.Title
				}
				all = append(all, issue)
			}
			if len(items) < 50 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GiteaAPI) CreateIssue(owner, repo string, issue IssueInfo, milestoneMap map[string]int64) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues", g.BaseURL, owner, repo)
	payload := map[string]interface{}{
		"title":  issue.Title,
		"body":   issue.Body,
		"labels": []int64{}, // Gitea 需要 label ID，后面处理
	}
	if issue.MilestoneTitle != "" {
		if msID, ok := milestoneMap[issue.MilestoneTitle]; ok {
			payload["milestone"] = msID
		}
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	// 关闭 issue
	if issue.State == "closed" {
		var created struct {
			Number int `json:"number"`
		}
		json.NewDecoder(resp.Body).Decode(&created)
		closeURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d", g.BaseURL, owner, repo, created.Number)
		closeResp, err := g.doRequest("PATCH", closeURL, map[string]string{"state": "closed"})
		if err == nil {
			closeResp.Body.Close()
		}
	}
	return nil
}

func (g *GiteaAPI) ListPullRequests(owner, repo string) ([]PullRequestInfo, error) {
	var all []PullRequestInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=%s&page=%d&limit=50", g.BaseURL, owner, repo, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID     int64  `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				State  string `json:"state"`
				Merged bool   `json:"merged"`
				Head   struct {
					Ref string `json:"ref"`
				} `json:"head"`
				Base struct {
					Ref string `json:"ref"`
				} `json:"base"`
				User struct {
					Username string `json:"username"`
				} `json:"user"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				st := it.State
				if it.Merged {
					st = "merged"
				}
				all = append(all, PullRequestInfo{
					ID: it.ID, Number: it.Number, Title: it.Title, Body: it.Body,
					State: st, Head: it.Head.Ref, Base: it.Base.Ref,
					User: it.User.Username, CreatedAt: it.CreatedAt,
				})
			}
			if len(items) < 50 {
				break
			}
			page++
		}
	}
	return all, nil
}

// Gitea 迁移 API — 当目标平台是 Gitea 时可直接使用
func (g *GiteaAPI) MigrateRepo(cloneAddr, repoName, token, srcPlatform string, opts MigrateOptions) (string, error) {
	url := g.BaseURL + "/api/v1/repos/migrate"

	serviceType := "git"
	switch strings.ToLower(srcPlatform) {
	case "github":
		serviceType = "github"
	case "gitea":
		serviceType = "gitea"
	case "gitlab":
		serviceType = "gitlab"
	case "gitee":
		serviceType = "gitee"
	}

	payload := map[string]interface{}{
		"clone_addr":    cloneAddr,
		"auth_token":    token,
		"repo_name":     repoName,
		"repo_owner":    g.Username,
		"service":       serviceType,
		"mirror":        false,
		"issues":        opts.Issues,
		"labels":        opts.Labels,
		"milestones":    opts.Milestones,
		"releases":      opts.Releases,
		"pull_requests": opts.PullRequests,
		"wiki":          false,
	}

	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("迁移失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		CloneURL string `json:"clone_url"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.CloneURL, nil
}

// ================ GitLab ================

func (g *GitLabAPI) ListLabels(owner, repo string) ([]LabelInfo, error) {
	var all []LabelInfo
	pp := g.projectPath(owner, repo)
	page := 1
	for {
		url := fmt.Sprintf("%s/api/v4/projects/%s/labels?per_page=100&page=%d", g.BaseURL, pp, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		var items []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
			Desc  string `json:"description"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			all = append(all, LabelInfo{ID: it.ID, Name: it.Name, Color: strings.TrimPrefix(it.Color, "#"), Desc: it.Desc})
		}
		if len(items) < 100 {
			break
		}
		page++
	}
	return all, nil
}

func (g *GitLabAPI) CreateLabel(owner, repo string, label LabelInfo) error {
	pp := g.projectPath(owner, repo)
	url := fmt.Sprintf("%s/api/v4/projects/%s/labels", g.BaseURL, pp)
	color := label.Color
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}
	payload := map[string]interface{}{
		"name":        label.Name,
		"color":       color,
		"description": label.Desc,
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (g *GitLabAPI) ListMilestones(owner, repo string) ([]MilestoneInfo, error) {
	var all []MilestoneInfo
	pp := g.projectPath(owner, repo)
	for _, state := range []string{"active", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/api/v4/projects/%s/milestones?state=%s&per_page=100&page=%d", g.BaseURL, pp, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID          int64  `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       string `json:"state"`
				DueDate     string `json:"due_date"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				msState := it.State
				if msState == "active" {
					msState = "open"
				}
				all = append(all, MilestoneInfo{
					ID: it.ID, Title: it.Title, Description: it.Description, State: msState, DueOn: it.DueDate,
				})
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GitLabAPI) CreateMilestone(owner, repo string, ms MilestoneInfo) (*MilestoneInfo, error) {
	pp := g.projectPath(owner, repo)
	url := fmt.Sprintf("%s/api/v4/projects/%s/milestones", g.BaseURL, pp)
	state := ms.State
	if state == "open" {
		state = "activate"
	} else if state == "closed" {
		state = "close"
	}
	payload := map[string]interface{}{
		"title":       ms.Title,
		"description": ms.Description,
		"state_event": state,
	}
	if ms.DueOn != "" {
		payload["due_date"] = ms.DueOn
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	var result struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return &MilestoneInfo{ID: result.ID, Title: result.Title, State: ms.State}, nil
}

func (g *GitLabAPI) ListIssues(owner, repo string) ([]IssueInfo, error) {
	var all []IssueInfo
	pp := g.projectPath(owner, repo)
	for _, state := range []string{"opened", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/api/v4/projects/%s/issues?state=%s&per_page=100&page=%d&order_by=created_at&sort=asc", g.BaseURL, pp, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID        int64    `json:"id"`
				IID       int      `json:"iid"`
				Title     string   `json:"title"`
				Desc      string   `json:"description"`
				State     string   `json:"state"`
				Labels    []string `json:"labels"`
				Milestone *struct {
					Title string `json:"title"`
				} `json:"milestone"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				issueState := it.State
				if issueState == "opened" {
					issueState = "open"
				}
				issue := IssueInfo{
					ID: it.ID, Number: it.IID, Title: it.Title, Body: it.Desc,
					State: issueState, Labels: it.Labels, CreatedAt: it.CreatedAt,
				}
				if it.Milestone != nil {
					issue.MilestoneTitle = it.Milestone.Title
				}
				all = append(all, issue)
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GitLabAPI) CreateIssue(owner, repo string, issue IssueInfo, milestoneMap map[string]int64) error {
	pp := g.projectPath(owner, repo)
	url := fmt.Sprintf("%s/api/v4/projects/%s/issues", g.BaseURL, pp)
	payload := map[string]interface{}{
		"title":       issue.Title,
		"description": issue.Body,
		"labels":      strings.Join(issue.Labels, ","),
	}
	if issue.MilestoneTitle != "" {
		if msID, ok := milestoneMap[issue.MilestoneTitle]; ok {
			payload["milestone_id"] = msID
		}
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	// 如果需要关闭 issue
	if issue.State == "closed" {
		var created struct {
			IID int `json:"iid"`
		}
		json.NewDecoder(resp.Body).Decode(&created)
		closeURL := fmt.Sprintf("%s/api/v4/projects/%s/issues/%d", g.BaseURL, pp, created.IID)
		closeResp, err := g.doRequest("PUT", closeURL, map[string]string{"state_event": "close"})
		if err == nil {
			closeResp.Body.Close()
		}
	}
	return nil
}

func (g *GitLabAPI) ListPullRequests(owner, repo string) ([]PullRequestInfo, error) {
	var all []PullRequestInfo
	pp := g.projectPath(owner, repo)
	for _, state := range []string{"opened", "closed", "merged"} {
		page := 1
		for {
			url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests?state=%s&per_page=100&page=%d&order_by=created_at&sort=asc", g.BaseURL, pp, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID           int64  `json:"id"`
				IID          int    `json:"iid"`
				Title        string `json:"title"`
				Description  string `json:"description"`
				State        string `json:"state"`
				SourceBranch string `json:"source_branch"`
				TargetBranch string `json:"target_branch"`
				Author       struct {
					Username string `json:"username"`
				} `json:"author"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				st := it.State
				if st == "opened" {
					st = "open"
				}
				all = append(all, PullRequestInfo{
					ID: it.ID, Number: it.IID, Title: it.Title, Body: it.Description,
					State: st, Head: it.SourceBranch, Base: it.TargetBranch,
					User: it.Author.Username, CreatedAt: it.CreatedAt,
				})
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

// ================ Gitee ================

func (g *GiteeAPI) ListLabels(owner, repo string) ([]LabelInfo, error) {
	var all []LabelInfo
	page := 1
	for {
		url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/labels?access_token=%s&per_page=100&page=%d", owner, repo, g.Token, page)
		resp, err := g.doRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		var items []struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Color string `json:"color"`
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
		}
		json.NewDecoder(resp.Body).Decode(&items)
		resp.Body.Close()
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			all = append(all, LabelInfo{ID: it.ID, Name: it.Name, Color: strings.TrimPrefix(it.Color, "#")})
		}
		if len(items) < 100 {
			break
		}
		page++
	}
	return all, nil
}

func (g *GiteeAPI) CreateLabel(owner, repo string, label LabelInfo) error {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/labels", owner, repo)
	color := label.Color
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}
	payload := map[string]interface{}{
		"access_token": g.Token,
		"name":         label.Name,
		"color":        color,
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (g *GiteeAPI) ListMilestones(owner, repo string) ([]MilestoneInfo, error) {
	var all []MilestoneInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/milestones?access_token=%s&state=%s&per_page=100&page=%d", owner, repo, g.Token, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID          int64  `json:"id"`
				Number      int    `json:"number"`
				Title       string `json:"title"`
				Description string `json:"description"`
				State       string `json:"state"`
				DueOn       string `json:"due_on"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				all = append(all, MilestoneInfo{
					ID: it.ID, Title: it.Title, Description: it.Description, State: it.State, DueOn: it.DueOn,
				})
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GiteeAPI) CreateMilestone(owner, repo string, ms MilestoneInfo) (*MilestoneInfo, error) {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/milestones", owner, repo)
	payload := map[string]interface{}{
		"access_token": g.Token,
		"title":        ms.Title,
		"description":  ms.Description,
		"state":        ms.State,
	}
	if ms.DueOn != "" {
		payload["due_on"] = ms.DueOn
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	var result struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return &MilestoneInfo{ID: result.ID, Title: result.Title, State: ms.State}, nil
}

func (g *GiteeAPI) ListIssues(owner, repo string) ([]IssueInfo, error) {
	var all []IssueInfo
	for _, state := range []string{"open", "closed"} {
		page := 1
		for {
			url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/issues?access_token=%s&state=%s&per_page=100&page=%d&direction=asc", owner, repo, g.Token, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID     int64  `json:"id"`
				Number string `json:"number"` // Gitee uses string
				Title  string `json:"title"`
				Body   string `json:"body"`
				State  string `json:"state"`
				Labels []struct {
					Name string `json:"name"`
				} `json:"labels"`
				Milestone *struct {
					Title string `json:"title"`
				} `json:"milestone"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				issue := IssueInfo{
					ID: it.ID, Title: it.Title, Body: it.Body,
					State: it.State, CreatedAt: it.CreatedAt,
				}
				for _, l := range it.Labels {
					issue.Labels = append(issue.Labels, l.Name)
				}
				if it.Milestone != nil {
					issue.MilestoneTitle = it.Milestone.Title
				}
				all = append(all, issue)
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}

func (g *GiteeAPI) CreateIssue(owner, repo string, issue IssueInfo, milestoneMap map[string]int64) error {
	url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/issues", owner)
	payload := map[string]interface{}{
		"access_token": g.Token,
		"repo":         repo,
		"title":        issue.Title,
		"body":         issue.Body,
	}
	if len(issue.Labels) > 0 {
		payload["labels"] = strings.Join(issue.Labels, ",")
	}
	if issue.MilestoneTitle != "" {
		if msID, ok := milestoneMap[issue.MilestoneTitle]; ok {
			payload["milestone"] = msID
		}
	}
	resp, err := g.doRequest("POST", url, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	// Gitee 关闭 issue 需要 PATCH
	if issue.State == "closed" {
		var created struct {
			Number string `json:"number"`
		}
		json.NewDecoder(resp.Body).Decode(&created)
		closeURL := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/issues/%s", owner, repo, created.Number)
		closeResp, err := g.doRequest("PATCH", closeURL, map[string]interface{}{
			"access_token": g.Token,
			"state":        "closed",
		})
		if err == nil {
			closeResp.Body.Close()
		}
	}
	return nil
}

func (g *GiteeAPI) ListPullRequests(owner, repo string) ([]PullRequestInfo, error) {
	var all []PullRequestInfo
	for _, state := range []string{"open", "closed", "merged"} {
		page := 1
		for {
			url := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/pulls?access_token=%s&state=%s&per_page=100&page=%d", owner, repo, g.Token, state, page)
			resp, err := g.doRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			var items []struct {
				ID     int64  `json:"id"`
				Number int    `json:"number"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				State  string `json:"state"`
				Head   struct {
					Ref string `json:"ref"`
				} `json:"head"`
				Base struct {
					Ref string `json:"ref"`
				} `json:"base"`
				User struct {
					Login string `json:"login"`
				} `json:"user"`
				CreatedAt string `json:"created_at"`
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
			}
			json.NewDecoder(resp.Body).Decode(&items)
			resp.Body.Close()
			if len(items) == 0 {
				break
			}
			for _, it := range items {
				all = append(all, PullRequestInfo{
					ID: it.ID, Number: it.Number, Title: it.Title, Body: it.Body,
					State: it.State, Head: it.Head.Ref, Base: it.Base.Ref,
					User: it.User.Login, CreatedAt: it.CreatedAt,
				})
			}
			if len(items) < 100 {
				break
			}
			page++
		}
	}
	return all, nil
}
