package app

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// --- 在线迁移相关类型 ---

// OnlineMigrateCompareRequest 在线迁移对比请求
type OnlineMigrateCompareRequest struct {
	SrcPlatform string `json:"srcPlatform"` // github / gitee / gitea
	SrcBaseURL  string `json:"srcBaseUrl"`
	SrcUsername string `json:"srcUsername"`
	SrcToken    string `json:"srcToken"`
	TgtPlatform string `json:"tgtPlatform"`
	TgtBaseURL  string `json:"tgtBaseUrl"`
	TgtUsername string `json:"tgtUsername"`
	TgtToken    string `json:"tgtToken"`
}

// MigrateOptions 迁移选项
type MigrateOptions struct {
	Labels       bool `json:"labels"`
	Issues       bool `json:"issues"`
	PullRequests bool `json:"pullRequests"`
	Releases     bool `json:"releases"`
	Milestones   bool `json:"milestones"`
}

// OnlineMigrateCompareResult 在线迁移对比结果
type OnlineMigrateCompareResult struct {
	SourceRepos  []RepoInfo `json:"sourceRepos"`  // 源平台所有仓库
	TargetRepos  []RepoInfo `json:"targetRepos"`  // 目标平台所有仓库
	MissingRepos []RepoInfo `json:"missingRepos"` // 目标平台缺少的仓库
}

// OnlineMigrateItemResult 单个仓库在线迁移结果
type OnlineMigrateItemResult struct {
	Name     string `json:"name"`
	CloneURL string `json:"cloneUrl,omitempty"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}

// OnlineMigrateProgress 迁移进度事件数据
type OnlineMigrateProgress struct {
	Current  int     `json:"current"`  // 当前正在处理第几个仓库（从 1 开始）
	Total    int     `json:"total"`    // 仓库总数
	RepoName string  `json:"repoName"` // 当前仓库名
	Phase    string  `json:"phase"`    // clone | create | push | labels | milestones | issues | releases | pullRequests | done | error
	Percent  float64 `json:"percent"`  // 整体进度百分比 0-100
	Message  string  `json:"message"`  // 可读状态描述
}

// emitMigrateProgress 向前端发送迁移进度
func (s *AppService) emitMigrateProgress(current, total int, repoName, phase, message string) {
	pct := float64(0)
	if total > 0 {
		phaseWeights := map[string]float64{
			"clone":        0.3,
			"create":       0.05,
			"push":         0.3,
			"labels":       0.05,
			"milestones":   0.05,
			"issues":       0.1,
			"releases":     0.1,
			"pullRequests": 0.05,
			"done":         1.0,
			"error":        0.0,
		}
		w := phaseWeights[phase]
		pct = (float64(current-1) + w) / float64(total) * 100
		if pct > 100 {
			pct = 100
		}
	}
	if s.app != nil {
		s.app.Event.Emit("online-migrate-progress", OnlineMigrateProgress{
			Current:  current,
			Total:    total,
			RepoName: repoName,
			Phase:    phase,
			Percent:  pct,
			Message:  message,
		})
	}
}

// --- 在线迁移服务方法 ---

// OnlineMigrateCompare 对比两个平台的仓库列表
func (s *AppService) OnlineMigrateCompare(
	srcPlatform, srcBaseURL, srcUsername, srcToken,
	tgtPlatform, tgtBaseURL, tgtUsername, tgtToken string,
) (*OnlineMigrateCompareResult, error) {
	srcAPI, err := NewPlatformAPI(srcPlatform, srcBaseURL, srcUsername, srcToken)
	if err != nil {
		return nil, fmt.Errorf("创建源平台 API 失败: %w", err)
	}
	tgtAPI, err := NewPlatformAPI(tgtPlatform, tgtBaseURL, tgtUsername, tgtToken)
	if err != nil {
		return nil, fmt.Errorf("创建目标平台 API 失败: %w", err)
	}

	srcRepos, err := srcAPI.ListRepos()
	if err != nil {
		return nil, fmt.Errorf("获取源平台仓库列表失败: %w", err)
	}
	tgtRepos, err := tgtAPI.ListRepos()
	if err != nil {
		return nil, fmt.Errorf("获取目标平台仓库列表失败: %w", err)
	}

	tgtNames := make(map[string]bool)
	for _, r := range tgtRepos {
		tgtNames[strings.ToLower(r.Name)] = true
	}

	var missing []RepoInfo
	for _, r := range srcRepos {
		if !tgtNames[strings.ToLower(r.Name)] {
			missing = append(missing, r)
		}
	}

	return &OnlineMigrateCompareResult{
		SourceRepos:  srcRepos,
		TargetRepos:  tgtRepos,
		MissingRepos: missing,
	}, nil
}

// OnlineMigrateExecute 执行在线迁移
func (s *AppService) OnlineMigrateExecute(
	srcPlatform, srcBaseURL, srcUsername, srcToken string,
	tgtPlatform, tgtBaseURL, tgtUsername, tgtToken string,
	repoNames []string,
	srcUseProxy, tgtUseProxy bool,
	opts MigrateOptions,
) []OnlineMigrateItemResult {

	total := len(repoNames)

	srcAPI, err := NewPlatformAPI(srcPlatform, srcBaseURL, srcUsername, srcToken)
	if err != nil {
		return []OnlineMigrateItemResult{{Name: "(全局)", Message: fmt.Sprintf("创建源平台 API 失败: %v", err)}}
	}

	tgtAPI, err := NewPlatformAPI(tgtPlatform, tgtBaseURL, tgtUsername, tgtToken)
	if err != nil {
		return []OnlineMigrateItemResult{{Name: "(全局)", Message: fmt.Sprintf("创建目标平台 API 失败: %v", err)}}
	}

	srcRepos, err := srcAPI.ListRepos()
	if err != nil {
		return []OnlineMigrateItemResult{{Name: "(全局)", Message: fmt.Sprintf("获取源平台仓库列表失败: %v", err)}}
	}
	repoMap := make(map[string]RepoInfo)
	for _, r := range srcRepos {
		repoMap[strings.ToLower(r.Name)] = r
	}

	// 检查目标是否是 Gitea（可以用原生迁移 API）
	// 但如果源平台需要代理，说明 Gitea 服务器可能无法直接访问源平台（如国内服务器访问 GitHub），
	// 此时应跳过原生迁移，改用本地中转（本地代理克隆 → 推送到 Gitea）
	giteaTgt, isGiteaTarget := tgtAPI.(*GiteaAPI)
	useNativeGiteaMigrate := isGiteaTarget && !srcUseProxy

	// 创建临时目录（非原生 Gitea 迁移时使用）
	var tmpDir string
	if !useNativeGiteaMigrate {
		tmpDir, err = os.MkdirTemp("", "gitpilot-migrate-*")
		if err != nil {
			return []OnlineMigrateItemResult{{Name: "(全局)", Message: fmt.Sprintf("创建临时目录失败: %v", err)}}
		}
		defer os.RemoveAll(tmpDir)
	}

	srcProxy := srcUseProxy
	tgtProxy := tgtUseProxy

	var results []OnlineMigrateItemResult

	for idx, name := range repoNames {
		current := idx + 1
		result := OnlineMigrateItemResult{Name: name}

		srcRepo, ok := repoMap[strings.ToLower(name)]
		if !ok {
			result.Message = "在源平台未找到该仓库"
			s.emitMigrateProgress(current, total, name, "error", result.Message)
			results = append(results, result)
			continue
		}

		cloneURL := srcRepo.CloneURL
		if cloneURL == "" {
			result.Message = "源仓库无 clone URL"
			s.emitMigrateProgress(current, total, name, "error", result.Message)
			results = append(results, result)
			continue
		}

		// 解析源仓库的 owner/repo
		srcOwner, srcRepoName, parseErr := ParseOwnerRepo(cloneURL)
		if parseErr != nil {
			srcOwner = srcUsername
			srcRepoName = name
		}

		// --- Gitea 目标且源不需代理: 使用原生迁移 API ---
		if useNativeGiteaMigrate {
			s.emitMigrateProgress(current, total, name, "clone", fmt.Sprintf("正在迁移 %s（Gitea 原生迁移）...", name))

			targetCloneURL, migrateErr := giteaTgt.MigrateRepo(cloneURL, name, srcToken, srcPlatform, opts)
			if migrateErr != nil {
				result.Message = fmt.Sprintf("Gitea 迁移失败: %v", migrateErr)
				s.emitMigrateProgress(current, total, name, "error", result.Message)
				results = append(results, result)
				continue
			}

			result.CloneURL = targetCloneURL
			result.Success = true
			result.Message = "迁移成功（含选中的标签/工单/发布/里程碑/合并请求）"
			s.emitMigrateProgress(current, total, name, "done", result.Message)
			results = append(results, result)
			log.Printf("在线迁移(Gitea): %s 完成 → %s", name, targetCloneURL)
			continue
		}

		// --- 非 Gitea 目标: 手动迁移 ---
		authCloneURL := injectTokenToURL(cloneURL, srcUsername, srcToken, srcPlatform)
		localPath := filepath.Join(tmpDir, name)

		// 1. Clone
		s.emitMigrateProgress(current, total, name, "clone", fmt.Sprintf("正在克隆 %s...", name))
		_, err = s.gitClient.CloneWithProxy(authCloneURL, localPath, &srcProxy)
		if err != nil {
			result.Message = fmt.Sprintf("克隆失败: %v", err)
			s.emitMigrateProgress(current, total, name, "error", result.Message)
			results = append(results, result)
			continue
		}

		// 2. 创建目标仓库
		s.emitMigrateProgress(current, total, name, "create", fmt.Sprintf("正在创建目标仓库 %s...", name))
		targetCloneURL, err := tgtAPI.CreateRepo(name, srcRepo.Description, srcRepo.Private)
		if err != nil {
			result.Message = fmt.Sprintf("创建目标仓库失败: %v", err)
			s.emitMigrateProgress(current, total, name, "error", result.Message)
			results = append(results, result)
			continue
		}
		result.CloneURL = targetCloneURL

		tgtOwner, tgtRepoName, tgtParseErr := ParseOwnerRepo(targetCloneURL)
		if tgtParseErr != nil {
			tgtOwner = tgtUsername
			tgtRepoName = name
		}

		pushURL := injectTokenToURL(targetCloneURL, tgtUsername, tgtToken, tgtPlatform)

		// 3. 添加 remote
		_, err = s.gitClient.Run(localPath, "remote", "add", "target", pushURL)
		if err != nil {
			result.Message = fmt.Sprintf("添加 remote 失败: %v", err)
			s.emitMigrateProgress(current, total, name, "error", result.Message)
			results = append(results, result)
			continue
		}

		// 4. Push 分支和标签
		s.emitMigrateProgress(current, total, name, "push", fmt.Sprintf("正在推送 %s 的分支和标签...", name))
		pushFailed := false

		// 先尝试 --all 一次性推送
		_, err = s.gitClient.RunWithProxyTimeout(localPath, &tgtProxy, 10*time.Minute,
			"-c", "http.postBuffer=524288000", "push", "target", "--all")
		if err != nil {
			// --all 推送失败（可能 HTTP 413），改为逐分支推送
			log.Printf("push --all 失败，改为逐分支推送: %v", err)
			s.emitMigrateProgress(current, total, name, "push", fmt.Sprintf("正在逐分支推送 %s...", name))

			branchOut, brErr := s.gitClient.Run(localPath, "branch", "-a", "--format=%(refname:short)")
			if brErr != nil {
				result.Message = fmt.Sprintf("获取分支列表失败: %v", brErr)
				s.emitMigrateProgress(current, total, name, "error", result.Message)
				results = append(results, result)
				continue
			}
			branches := strings.Split(strings.TrimSpace(branchOut), "\n")
			var pushErrs []string
			for _, br := range branches {
				br = strings.TrimSpace(br)
				if br == "" || strings.HasPrefix(br, "origin/HEAD") {
					continue
				}
				// 将 origin/xxx 转换为本地 refspec
				refspec := br
				if strings.HasPrefix(br, "origin/") {
					refspec = "refs/remotes/" + br + ":refs/heads/" + strings.TrimPrefix(br, "origin/")
				}
				_, pushErr := s.gitClient.RunWithProxyTimeout(localPath, &tgtProxy, 10*time.Minute,
					"-c", "http.postBuffer=524288000", "push", "target", refspec)
				if pushErr != nil {
					pushErrs = append(pushErrs, fmt.Sprintf("%s(%v)", br, pushErr))
				}
			}
			if len(pushErrs) > 0 {
				// 如果所有分支都失败，才算整体失败
				if len(pushErrs) == len(branches) {
					result.Message = fmt.Sprintf("推送分支全部失败: %s", strings.Join(pushErrs, "; "))
					s.emitMigrateProgress(current, total, name, "error", result.Message)
					results = append(results, result)
					pushFailed = true
				} else {
					log.Printf("部分分支推送失败: %s", strings.Join(pushErrs, "; "))
				}
			}
		}
		if pushFailed {
			continue
		}

		// 推送标签
		s.gitClient.RunWithProxyTimeout(localPath, &tgtProxy, 10*time.Minute,
			"-c", "http.postBuffer=524288000", "push", "target", "--tags")

		// --- 迁移扩展数据 ---
		var migrateNotes []string

		// 5. 标签
		if opts.Labels {
			s.emitMigrateProgress(current, total, name, "labels", fmt.Sprintf("正在迁移 %s 的标签...", name))
			if labErr := s.migrateLabels(srcAPI, tgtAPI, srcOwner, srcRepoName, tgtOwner, tgtRepoName); labErr != nil {
				migrateNotes = append(migrateNotes, fmt.Sprintf("标签: %v", labErr))
			} else {
				migrateNotes = append(migrateNotes, "标签 ✓")
			}
		}

		// 6. 里程碑
		var milestoneMap map[string]int64
		if opts.Milestones {
			s.emitMigrateProgress(current, total, name, "milestones", fmt.Sprintf("正在迁移 %s 的里程碑...", name))
			var msErr error
			milestoneMap, msErr = s.migrateMilestones(srcAPI, tgtAPI, srcOwner, srcRepoName, tgtOwner, tgtRepoName)
			if msErr != nil {
				migrateNotes = append(migrateNotes, fmt.Sprintf("里程碑: %v", msErr))
			} else {
				migrateNotes = append(migrateNotes, "里程碑 ✓")
			}
		}

		// 7. 工单
		if opts.Issues {
			s.emitMigrateProgress(current, total, name, "issues", fmt.Sprintf("正在迁移 %s 的工单...", name))
			if issErr := s.migrateIssues(srcAPI, tgtAPI, srcOwner, srcRepoName, tgtOwner, tgtRepoName, milestoneMap); issErr != nil {
				migrateNotes = append(migrateNotes, fmt.Sprintf("工单: %v", issErr))
			} else {
				migrateNotes = append(migrateNotes, "工单 ✓")
			}
		}

		// 8. 发布
		if opts.Releases {
			s.emitMigrateProgress(current, total, name, "releases", fmt.Sprintf("正在迁移 %s 的发布...", name))
			if relErr := s.migrateReleases(srcAPI, tgtAPI, srcOwner, srcRepoName, tgtOwner, tgtRepoName, srcToken, srcPlatform); relErr != nil {
				migrateNotes = append(migrateNotes, fmt.Sprintf("发布: %v", relErr))
			} else {
				migrateNotes = append(migrateNotes, "发布 ✓")
			}
		}

		// 9. 合并请求（以 Issue 方式记录）
		if opts.PullRequests {
			s.emitMigrateProgress(current, total, name, "pullRequests", fmt.Sprintf("正在迁移 %s 的合并请求...", name))
			if prErr := s.migratePullRequests(srcAPI, tgtAPI, srcOwner, srcRepoName, tgtOwner, tgtRepoName, milestoneMap); prErr != nil {
				migrateNotes = append(migrateNotes, fmt.Sprintf("合并请求: %v", prErr))
			} else {
				migrateNotes = append(migrateNotes, "合并请求 ✓")
			}
		}

		result.Success = true
		if len(migrateNotes) > 0 {
			result.Message = "迁移完成 | " + strings.Join(migrateNotes, " | ")
		} else {
			result.Message = "迁移成功"
		}
		s.emitMigrateProgress(current, total, name, "done", result.Message)
		results = append(results, result)
		log.Printf("在线迁移: %s 完成 → %s", name, targetCloneURL)

		os.RemoveAll(localPath)
	}

	return results
}

// --- 迁移子步骤 ---

func (s *AppService) migrateLabels(srcAPI, tgtAPI PlatformAPI, srcOwner, srcRepo, tgtOwner, tgtRepo string) error {
	srcLabels, err := srcAPI.ListLabels(srcOwner, srcRepo)
	if err != nil {
		return fmt.Errorf("获取源标签失败: %w", err)
	}
	if len(srcLabels) == 0 {
		return nil
	}
	tgtLabels, _ := tgtAPI.ListLabels(tgtOwner, tgtRepo)
	existing := make(map[string]bool)
	for _, l := range tgtLabels {
		existing[strings.ToLower(l.Name)] = true
	}
	var errs []string
	for _, label := range srcLabels {
		if existing[strings.ToLower(label.Name)] {
			continue
		}
		if err := tgtAPI.CreateLabel(tgtOwner, tgtRepo, label); err != nil {
			errs = append(errs, fmt.Sprintf("%s(%v)", label.Name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *AppService) migrateMilestones(srcAPI, tgtAPI PlatformAPI, srcOwner, srcRepo, tgtOwner, tgtRepo string) (map[string]int64, error) {
	msMap := make(map[string]int64)
	srcMS, err := srcAPI.ListMilestones(srcOwner, srcRepo)
	if err != nil {
		return msMap, fmt.Errorf("获取源里程碑失败: %w", err)
	}
	if len(srcMS) == 0 {
		return msMap, nil
	}
	tgtMS, _ := tgtAPI.ListMilestones(tgtOwner, tgtRepo)
	for _, ms := range tgtMS {
		msMap[ms.Title] = ms.ID
	}
	var errs []string
	for _, ms := range srcMS {
		if _, exists := msMap[ms.Title]; exists {
			continue
		}
		created, err := tgtAPI.CreateMilestone(tgtOwner, tgtRepo, ms)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s(%v)", ms.Title, err))
		} else if created != nil {
			msMap[created.Title] = created.ID
		}
	}
	if len(errs) > 0 {
		return msMap, fmt.Errorf("部分失败: %s", strings.Join(errs, "; "))
	}
	return msMap, nil
}

func (s *AppService) migrateIssues(srcAPI, tgtAPI PlatformAPI, srcOwner, srcRepo, tgtOwner, tgtRepo string, milestoneMap map[string]int64) error {
	srcIssues, err := srcAPI.ListIssues(srcOwner, srcRepo)
	if err != nil {
		return fmt.Errorf("获取源工单失败: %w", err)
	}
	if len(srcIssues) == 0 {
		return nil
	}
	if milestoneMap == nil {
		milestoneMap = make(map[string]int64)
	}
	var errs []string
	for _, issue := range srcIssues {
		if err := tgtAPI.CreateIssue(tgtOwner, tgtRepo, issue, milestoneMap); err != nil {
			errs = append(errs, fmt.Sprintf("#%d(%v)", issue.Number, err))
			if len(errs) >= 5 {
				errs = append(errs, "...更多错误已省略")
				break
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *AppService) migrateReleases(srcAPI, tgtAPI PlatformAPI, srcOwner, srcRepo, tgtOwner, tgtRepo, srcToken, srcPlatform string) error {
	srcRel, err := srcAPI.ListReleases(srcOwner, srcRepo)
	if err != nil {
		return fmt.Errorf("获取源发布失败: %w", err)
	}
	if len(srcRel) == 0 {
		return nil
	}
	tgtRel, _ := tgtAPI.ListReleases(tgtOwner, tgtRepo)
	existing := make(map[string]bool)
	for _, r := range tgtRel {
		existing[r.TagName] = true
	}
	var errs []string
	for _, rel := range srcRel {
		if existing[rel.TagName] {
			continue
		}
		created, err := tgtAPI.CreateRelease(tgtOwner, tgtRepo, ReleaseCreate{
			TagName: rel.TagName, Name: rel.Name, Body: rel.Body, Draft: rel.Draft, Prerelease: rel.Prerelease,
		})
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s(%v)", rel.TagName, err))
			continue
		}
		for _, asset := range rel.Assets {
			if asset.DownloadURL == "" {
				continue
			}
			data, dlErr := DownloadAsset(asset.DownloadURL, srcToken, srcPlatform)
			if dlErr != nil {
				continue
			}
			tgtAPI.UploadAsset(tgtOwner, tgtRepo, created.ID, asset.Name, bytes.NewReader(data))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (s *AppService) migratePullRequests(srcAPI, tgtAPI PlatformAPI, srcOwner, srcRepo, tgtOwner, tgtRepo string, milestoneMap map[string]int64) error {
	srcPRs, err := srcAPI.ListPullRequests(srcOwner, srcRepo)
	if err != nil {
		return fmt.Errorf("获取源合并请求失败: %w", err)
	}
	if len(srcPRs) == 0 {
		return nil
	}
	if milestoneMap == nil {
		milestoneMap = make(map[string]int64)
	}
	var errs []string
	for _, pr := range srcPRs {
		stateEmoji := "🟢"
		if pr.State == "closed" {
			stateEmoji = "🔴"
		} else if pr.State == "merged" {
			stateEmoji = "🟣"
		}
		body := fmt.Sprintf(
			"%s **[迁移的合并请求]** `%s` → `%s` | 状态: %s %s | 作者: @%s\n\n---\n\n%s",
			stateEmoji, pr.Head, pr.Base, stateEmoji, pr.State, pr.User, pr.Body,
		)
		issue := IssueInfo{
			Title: fmt.Sprintf("[PR#%d] %s", pr.Number, pr.Title),
			Body:  body,
			State: "closed",
		}
		if err := tgtAPI.CreateIssue(tgtOwner, tgtRepo, issue, milestoneMap); err != nil {
			errs = append(errs, fmt.Sprintf("PR#%d(%v)", pr.Number, err))
			if len(errs) >= 5 {
				errs = append(errs, "...更多错误已省略")
				break
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("部分失败: %s", strings.Join(errs, "; "))
	}
	return nil
}

// injectTokenToURL 将 token 注入到 HTTPS clone URL 中用于认证
func injectTokenToURL(cloneURL, username, token, platform string) string {
	if token == "" {
		return cloneURL
	}
	if !strings.HasPrefix(cloneURL, "https://") && !strings.HasPrefix(cloneURL, "http://") {
		return cloneURL
	}
	idx := strings.Index(cloneURL, "://")
	if idx < 0 {
		return cloneURL
	}
	scheme := cloneURL[:idx+3]
	rest := cloneURL[idx+3:]
	return fmt.Sprintf("%s%s:%s@%s", scheme, username, token, rest)
}
