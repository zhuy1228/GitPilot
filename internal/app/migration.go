package app

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
)

// --- 迁移相关类型 ---

// MigrateResult 迁移结果
type MigrateResult struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	CloneURL   string `json:"cloneUrl,omitempty"`
	RemoteName string `json:"remoteName,omitempty"`
}

// --- Release 同步相关类型 ---

// ReleaseSyncResult 单条 Release 同步结果
type ReleaseSyncResult struct {
	TagName string `json:"tagName"`
	Name    string `json:"name"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// --- 辅助：根据 remote URL 匹配 credential ---

// detectPlatformFromURL 从 remote URL 推断平台类型
func detectPlatformFromURL(remoteURL string) string {
	u := strings.ToLower(remoteURL)
	if strings.Contains(u, "github.com") {
		return "github"
	}
	if strings.Contains(u, "gitee.com") {
		return "gitee"
	}
	if strings.Contains(u, "gitlab.com") {
		return "gitlab"
	}
	// 默认当作 Gitea（自建）
	return "gitea"
}

// extractBaseURL 从 remote URL 提取 base URL（用于 Gitea 自建平台匹配）
func extractBaseURL(remoteURL string) string {
	// https://host:port/owner/repo.git => https://host:port
	// git@host:owner/repo.git => 不适用
	if strings.HasPrefix(remoteURL, "git@") {
		return ""
	}
	idx := strings.Index(remoteURL, "://")
	if idx < 0 {
		return ""
	}
	rest := remoteURL[idx+3:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		return ""
	}
	return remoteURL[:idx+3+slashIdx]
}

// findCredentialForRemote 根据 remote URL 自动查找匹配的凭证
func (s *AppService) findCredentialForRemote(remoteURL string) (platform, baseURL, username, token string, found bool) {
	plat := detectPlatformFromURL(remoteURL)
	remoteBase := extractBaseURL(remoteURL)

	// 1. 对于自建平台（gitea/gitlab 或未知），优先按 BaseURL 精确匹配
	if plat == "gitea" || plat == "gitlab" {
		if remoteBase != "" {
			for _, c := range s.config.Credentials {
				cp := strings.ToLower(c.Platform)
				if (cp == "gitea" || cp == "gitlab") && c.BaseURL != "" {
					if strings.EqualFold(strings.TrimRight(c.BaseURL, "/"), strings.TrimRight(remoteBase, "/")) {
						return c.Platform, c.BaseURL, c.Username, c.Token, true
					}
				}
			}
		}
	}

	// 2. 按平台名称匹配（github / gitee / gitlab.com 等公有平台）
	for _, c := range s.config.Credentials {
		if strings.ToLower(c.Platform) == plat {
			return c.Platform, c.BaseURL, c.Username, c.Token, true
		}
	}
	return "", "", "", "", false
}

// --- 迁移服务方法 ---

// MigrateProject 将项目迁移到目标平台
// targetCredPlatform + targetCredUsername 用于定位目标凭证
// repoName: 目标仓库名（空则用项目名）
// remoteName: 新增的远程名（空则用平台名）
// private: 是否私有仓库
// description: 仓库描述
func (s *AppService) MigrateProject(projectPath, targetCredPlatform, targetCredUsername, repoName, remoteName, description string, private bool) (*MigrateResult, error) {
	result := &MigrateResult{Path: projectPath}

	// 校验项目路径
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", projectPath)
	}

	// 查找目标凭证
	var cred *CredentialInfo
	for _, c := range s.config.Credentials {
		if strings.EqualFold(c.Platform, targetCredPlatform) && c.Username == targetCredUsername {
			cred = &CredentialInfo{Platform: c.Platform, BaseURL: c.BaseURL, Username: c.Username, Token: c.Token}
			break
		}
	}
	if cred == nil {
		return nil, fmt.Errorf("未找到目标凭证: %s/%s", targetCredPlatform, targetCredUsername)
	}
	if cred.Token == "" {
		return nil, fmt.Errorf("目标凭证缺少 Token: %s/%s", targetCredPlatform, targetCredUsername)
	}

	// 创建平台 API 客户端
	api, err := NewPlatformAPI(cred.Platform, cred.BaseURL, cred.Username, cred.Token)
	if err != nil {
		return nil, fmt.Errorf("创建平台 API 失败: %w", err)
	}

	// 确定仓库名
	if repoName == "" {
		// 从项目名或路径推断
		for _, p := range s.config.Projects {
			if p.Path == projectPath {
				repoName = p.Name
				break
			}
		}
		if repoName == "" {
			parts := strings.Split(strings.ReplaceAll(projectPath, "\\", "/"), "/")
			repoName = parts[len(parts)-1]
		}
	}

	// 确定远程名
	if remoteName == "" {
		remoteName = strings.ToLower(cred.Platform)
		// 如果已存在同名 remote，加上用户名
		remoteList, _ := s.gitClient.RemoteList(projectPath)
		for _, r := range remoteList {
			if r.Name == remoteName {
				remoteName = strings.ToLower(cred.Platform) + "-" + cred.Username
				break
			}
		}
	}

	result.Name = repoName
	result.RemoteName = remoteName

	// 1. 通过 API 在目标平台创建仓库
	cloneURL, err := api.CreateRepo(repoName, description, private)
	if err != nil {
		result.Message = fmt.Sprintf("创建仓库失败: %v", err)
		return result, nil
	}
	result.CloneURL = cloneURL
	log.Printf("迁移: 在 %s 上创建仓库 %s => %s", cred.Platform, repoName, cloneURL)

	// 2. 添加 remote
	_, err = s.gitClient.AddRemote(projectPath, remoteName, cloneURL)
	if err != nil {
		result.Message = fmt.Sprintf("仓库已创建(%s)，但添加远程仓库失败: %v", cloneURL, err)
		return result, nil
	}

	// 3. 推送所有分支到新 remote
	proxy := s.GetProjectProxy(projectPath)
	_, err = s.gitClient.RunWithProxy(projectPath, proxy, "push", remoteName, "--all")
	if err != nil {
		result.Message = fmt.Sprintf("推送分支失败: %v", err)
		return result, nil
	}

	// 4. 推送所有标签到新 remote
	_, err = s.gitClient.RunWithProxy(projectPath, proxy, "push", remoteName, "--tags")
	if err != nil {
		result.Message = fmt.Sprintf("分支已推送，但推送标签失败: %v", err)
		return result, nil
	}

	result.Success = true
	result.Message = "迁移成功"
	return result, nil
}

// BatchMigrateProjects 批量迁移项目到目标平台
func (s *AppService) BatchMigrateProjects(projectPaths []string, targetCredPlatform, targetCredUsername, description string, private bool) []MigrateResult {
	var results []MigrateResult
	for _, path := range projectPaths {
		result, err := s.MigrateProject(path, targetCredPlatform, targetCredUsername, "", "", description, private)
		if err != nil {
			results = append(results, MigrateResult{
				Path:    path,
				Message: err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}
	return results
}

// --- Release 同步服务方法 ---

// GetRemoteReleases 获取指定远程仓库的 Release 列表
// overrideToken: 前端传入的 Token，优先于凭证中的 Token
func (s *AppService) GetRemoteReleases(projectPath, remoteName, overrideToken string) ([]ReleaseInfo, error) {
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", projectPath)
	}

	// 获取 remote URL
	remoteList, err := s.gitClient.RemoteList(projectPath)
	if err != nil {
		return nil, fmt.Errorf("获取远程仓库列表失败: %w", err)
	}

	var remoteURL string
	for _, r := range remoteList {
		if r.Name == remoteName {
			remoteURL = r.URL
			break
		}
	}
	if remoteURL == "" {
		return nil, fmt.Errorf("未找到远程仓库: %s", remoteName)
	}

	// 查找凭证
	plat, baseURL, username, token, found := s.findCredentialForRemote(remoteURL)
	if !found {
		return nil, fmt.Errorf("未找到匹配的凭证，请先配置匹配 %s 的凭证", remoteURL)
	}
	if overrideToken != "" {
		token = overrideToken
	}

	// 创建 API 客户端
	api, err := NewPlatformAPI(plat, baseURL, username, token)
	if err != nil {
		return nil, fmt.Errorf("创建平台 API 失败: %w", err)
	}

	// 解析 owner/repo
	owner, repo, err := ParseOwnerRepo(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("解析远程地址失败: %w", err)
	}

	return api.ListReleases(owner, repo)
}

// SyncReleases 将源远程仓库的 Release 同步到目标远程仓库
// 仅同步目标不存在的 Release，包括附件
func (s *AppService) SyncReleases(projectPath, sourceRemote, targetRemote, srcOverrideToken, tgtOverrideToken string) ([]ReleaseSyncResult, error) {
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", projectPath)
	}

	// 获取 remote URLs
	remoteList, err := s.gitClient.RemoteList(projectPath)
	if err != nil {
		return nil, fmt.Errorf("获取远程仓库列表失败: %w", err)
	}

	var sourceURL, targetURL string
	for _, r := range remoteList {
		if r.Name == sourceRemote {
			sourceURL = r.URL
		}
		if r.Name == targetRemote {
			targetURL = r.URL
		}
	}
	if sourceURL == "" {
		return nil, fmt.Errorf("未找到源远程仓库: %s", sourceRemote)
	}
	if targetURL == "" {
		return nil, fmt.Errorf("未找到目标远程仓库: %s", targetRemote)
	}

	// 查找源和目标凭证
	srcPlat, srcBase, srcUser, srcToken, srcFound := s.findCredentialForRemote(sourceURL)
	if !srcFound {
		return nil, fmt.Errorf("未找到源平台的凭证，请先配置匹配 %s 的凭证", sourceURL)
	}
	if srcOverrideToken != "" {
		srcToken = srcOverrideToken
	}
	tgtPlat, tgtBase, tgtUser, tgtToken, tgtFound := s.findCredentialForRemote(targetURL)
	if !tgtFound {
		return nil, fmt.Errorf("未找到目标平台的凭证，请先配置匹配 %s 的凭证", targetURL)
	}
	if tgtOverrideToken != "" {
		tgtToken = tgtOverrideToken
	}

	// 创建 API 客户端
	srcAPI, err := NewPlatformAPI(srcPlat, srcBase, srcUser, srcToken)
	if err != nil {
		return nil, fmt.Errorf("创建源平台 API 失败: %w", err)
	}
	tgtAPI, err := NewPlatformAPI(tgtPlat, tgtBase, tgtUser, tgtToken)
	if err != nil {
		return nil, fmt.Errorf("创建目标平台 API 失败: %w", err)
	}

	// 解析 owner/repo
	srcOwner, srcRepo, err := ParseOwnerRepo(sourceURL)
	if err != nil {
		return nil, fmt.Errorf("解析源远程地址失败: %w", err)
	}
	tgtOwner, tgtRepo, err := ParseOwnerRepo(targetURL)
	if err != nil {
		return nil, fmt.Errorf("解析目标远程地址失败: %w", err)
	}

	// 获取源 Release 列表
	srcReleases, err := srcAPI.ListReleases(srcOwner, srcRepo)
	if err != nil {
		return nil, fmt.Errorf("获取源 Release 列表失败: %w", err)
	}

	// 获取目标 Release 列表，用于跳过已存在的
	tgtReleases, err := tgtAPI.ListReleases(tgtOwner, tgtRepo)
	if err != nil {
		// 目标可能还没有 Release，不报错
		tgtReleases = []ReleaseInfo{}
	}
	existingTags := make(map[string]bool)
	for _, r := range tgtReleases {
		existingTags[r.TagName] = true
	}

	var results []ReleaseSyncResult

	// 先确保所有 tags 已推送到目标
	proxy := s.GetProjectProxy(projectPath)
	s.gitClient.RunWithProxy(projectPath, proxy, "push", targetRemote, "--tags")

	// 逐个同步 Release
	for _, srcRel := range srcReleases {
		result := ReleaseSyncResult{
			TagName: srcRel.TagName,
			Name:    srcRel.Name,
		}

		// 跳过已存在的
		if existingTags[srcRel.TagName] {
			result.Success = true
			result.Message = "已存在，跳过"
			results = append(results, result)
			continue
		}

		// 创建 Release
		created, err := tgtAPI.CreateRelease(tgtOwner, tgtRepo, ReleaseCreate{
			TagName:    srcRel.TagName,
			Name:       srcRel.Name,
			Body:       srcRel.Body,
			Draft:      srcRel.Draft,
			Prerelease: srcRel.Prerelease,
		})
		if err != nil {
			result.Message = fmt.Sprintf("创建 Release 失败: %v", err)
			results = append(results, result)
			continue
		}

		// 同步附件
		assetErrors := []string{}
		for _, asset := range srcRel.Assets {
			if asset.DownloadURL == "" {
				continue
			}
			// 下载附件
			data, err := DownloadAsset(asset.DownloadURL, srcToken, srcPlat)
			if err != nil {
				assetErrors = append(assetErrors, fmt.Sprintf("%s: 下载失败(%v)", asset.Name, err))
				continue
			}
			// 上传到目标
			err = tgtAPI.UploadAsset(tgtOwner, tgtRepo, created.ID, asset.Name, bytes.NewReader(data))
			if err != nil {
				assetErrors = append(assetErrors, fmt.Sprintf("%s: 上传失败(%v)", asset.Name, err))
			}
		}

		if len(assetErrors) > 0 {
			result.Success = true
			result.Message = "Release 已创建，但部分附件同步失败: " + strings.Join(assetErrors, "; ")
		} else {
			result.Success = true
			if len(srcRel.Assets) > 0 {
				result.Message = fmt.Sprintf("同步成功（含 %d 个附件）", len(srcRel.Assets))
			} else {
				result.Message = "同步成功"
			}
		}
		results = append(results, result)
	}

	return results, nil
}
