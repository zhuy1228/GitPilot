package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/zhuy1228/GitPilot/config"
	"github.com/zhuy1228/GitPilot/internal/git"
)

// AppService 应用服务，暴露给前端调用
type AppService struct {
	app       *application.App
	config    *config.AppConfig
	gitClient *git.GitClient
}

func (a *AppService) SetApplication(app *application.App) {
	a.app = app
}

func New() *AppService {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置失败: %v, 使用默认配置", err)
		cfg = &config.AppConfig{
			Platforms: make(map[string]config.Platform),
			Settings:  config.Settings{Concurrency: 6, NetworkCheck: true, LogLevel: "info"},
		}
	}
	return &AppService{
		config:    cfg,
		gitClient: git.NewGitClient(),
	}
}

// SelectDirectory 打开系统文件夹选择器，返回选中的路径
func (s *AppService) SelectDirectory() (string, error) {
	if s.app == nil {
		return "", fmt.Errorf("应用未初始化")
	}
	path, err := s.app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("选择项目文件夹").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("打开文件夹选择器失败: %w", err)
	}
	return path, nil
}

// --- 平台/项目 树形结构 ---

// TreeNode 前端侧边栏树节点
type TreeNode struct {
	Key      string     `json:"key"`
	Label    string     `json:"label"`
	Type     string     `json:"type"` // platform, user, project
	Path     string     `json:"path,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// GetProjectTree 获取项目树，供前端侧边栏渲染
func (s *AppService) GetProjectTree() []TreeNode {
	var tree []TreeNode
	for platformName, platform := range s.config.Platforms {
		platformNode := TreeNode{
			Key:   platformName,
			Label: platformName,
			Type:  "platform",
		}
		for _, user := range platform.Users {
			userNode := TreeNode{
				Key:   platformName + "/" + user.Username,
				Label: user.Username,
				Type:  "user",
			}
			for _, proj := range user.Projects {
				userNode.Children = append(userNode.Children, TreeNode{
					Key:   platformName + "/" + user.Username + "/" + proj.Name,
					Label: proj.Name,
					Type:  "project",
					Path:  proj.Path,
				})
			}
			platformNode.Children = append(platformNode.Children, userNode)
		}
		tree = append(tree, platformNode)
	}
	return tree
}

// --- 项目管理 ---

// CloneProject 克隆远程仓库到本地目录，并添加到项目树
func (s *AppService) CloneProject(platform, username, repoURL, parentDir, name string) error {
	if repoURL == "" {
		return fmt.Errorf("仓库地址不能为空")
	}
	if parentDir == "" {
		return fmt.Errorf("目标目录不能为空")
	}
	if name == "" {
		return fmt.Errorf("项目名称不能为空")
	}

	// 目标路径: parentDir/name
	targetPath := parentDir + "/" + name

	// 执行 git clone
	_, err := s.gitClient.Clone(repoURL, targetPath)
	if err != nil {
		return fmt.Errorf("克隆失败: %w", err)
	}

	// 克隆成功后添加到项目树
	return s.AddProject(platform, username, name, targetPath)
}

// AddProject 添加项目到指定平台/用户下
func (s *AppService) AddProject(platform, username, name, path string) error {
	p, ok := s.config.Platforms[platform]
	if !ok {
		return fmt.Errorf("平台 %s 不存在", platform)
	}
	for i, user := range p.Users {
		if user.Username == username {
			// 检查重复
			for _, proj := range user.Projects {
				if proj.Name == name {
					return fmt.Errorf("项目 %s 已存在", name)
				}
			}
			s.config.Platforms[platform].Users[i].Projects = append(
				s.config.Platforms[platform].Users[i].Projects,
				config.Project{Name: name, Path: path},
			)
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("用户 %s 不存在于平台 %s", username, platform)
}

// RemoveProject 从指定平台/用户下删除项目
func (s *AppService) RemoveProject(platform, username, name string) error {
	p, ok := s.config.Platforms[platform]
	if !ok {
		return fmt.Errorf("平台 %s 不存在", platform)
	}
	for i, user := range p.Users {
		if user.Username == username {
			projects := user.Projects
			for j, proj := range projects {
				if proj.Name == name {
					s.config.Platforms[platform].Users[i].Projects = append(projects[:j], projects[j+1:]...)
					return config.SaveConfig(s.config)
				}
			}
			return fmt.Errorf("项目 %s 不存在", name)
		}
	}
	return fmt.Errorf("用户 %s 不存在于平台 %s", username, platform)
}

// --- 平台管理 ---

// AddPlatform 添加新平台
func (s *AppService) AddPlatform(name, baseURL string) error {
	if name == "" {
		return fmt.Errorf("平台名称不能为空")
	}
	if _, ok := s.config.Platforms[name]; ok {
		return fmt.Errorf("平台 %s 已存在", name)
	}
	s.config.Platforms[name] = config.Platform{
		BaseURL: baseURL,
		Users:   []config.User{},
	}
	return config.SaveConfig(s.config)
}

// UpdatePlatform 修改平台信息（base_url）
func (s *AppService) UpdatePlatform(name, baseURL string) error {
	p, ok := s.config.Platforms[name]
	if !ok {
		return fmt.Errorf("平台 %s 不存在", name)
	}
	p.BaseURL = baseURL
	s.config.Platforms[name] = p
	return config.SaveConfig(s.config)
}

// RemovePlatform 删除平台
func (s *AppService) RemovePlatform(name string) error {
	if _, ok := s.config.Platforms[name]; !ok {
		return fmt.Errorf("平台 %s 不存在", name)
	}
	delete(s.config.Platforms, name)
	return config.SaveConfig(s.config)
}

// --- 用户管理 ---

// AddUser 添加用户到指定平台
func (s *AppService) AddUser(platform, username, token string) error {
	p, ok := s.config.Platforms[platform]
	if !ok {
		return fmt.Errorf("平台 %s 不存在", platform)
	}
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	for _, user := range p.Users {
		if user.Username == username {
			return fmt.Errorf("用户 %s 已存在于平台 %s", username, platform)
		}
	}
	p.Users = append(p.Users, config.User{
		Username: username,
		Token:    token,
		Projects: []config.Project{},
	})
	s.config.Platforms[platform] = p
	return config.SaveConfig(s.config)
}

// UpdateUser 修改用户信息（用户名、token）
func (s *AppService) UpdateUser(platform, oldUsername, newUsername, token string) error {
	p, ok := s.config.Platforms[platform]
	if !ok {
		return fmt.Errorf("平台 %s 不存在", platform)
	}
	if newUsername == "" {
		return fmt.Errorf("新用户名不能为空")
	}
	for i, user := range p.Users {
		if user.Username == oldUsername {
			// 如果改了用户名，检查新名字是否冲突
			if oldUsername != newUsername {
				for _, u := range p.Users {
					if u.Username == newUsername {
						return fmt.Errorf("用户 %s 已存在于平台 %s", newUsername, platform)
					}
				}
			}
			s.config.Platforms[platform].Users[i].Username = newUsername
			s.config.Platforms[platform].Users[i].Token = token
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("用户 %s 不存在于平台 %s", oldUsername, platform)
}

// RemoveUser 从平台删除用户
func (s *AppService) RemoveUser(platform, username string) error {
	p, ok := s.config.Platforms[platform]
	if !ok {
		return fmt.Errorf("平台 %s 不存在", platform)
	}
	for i, user := range p.Users {
		if user.Username == username {
			s.config.Platforms[platform] = config.Platform{
				BaseURL: p.BaseURL,
				Users:   append(p.Users[:i], p.Users[i+1:]...),
			}
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("用户 %s 不存在于平台 %s", username, platform)
}

// GetUserInfo 获取用户信息
func (s *AppService) GetUserInfo(platform, username string) (*UserInfo, error) {
	p, ok := s.config.Platforms[platform]
	if !ok {
		return nil, fmt.Errorf("平台 %s 不存在", platform)
	}
	for _, user := range p.Users {
		if user.Username == username {
			return &UserInfo{
				Username: user.Username,
				Token:    user.Token,
			}, nil
		}
	}
	return nil, fmt.Errorf("用户 %s 不存在于平台 %s", username, platform)
}

// GetPlatformInfo 获取平台信息
func (s *AppService) GetPlatformInfo(name string) (*PlatformInfo, error) {
	p, ok := s.config.Platforms[name]
	if !ok {
		return nil, fmt.Errorf("平台 %s 不存在", name)
	}
	return &PlatformInfo{
		Name:    name,
		BaseURL: p.BaseURL,
	}, nil
}

// --- Git 操作 ---

// ProjectStatus 项目状态信息
type ProjectStatus struct {
	Branch       string     `json:"branch"`
	RemoteURL    string     `json:"remoteUrl"`
	ChangedFiles []FileInfo `json:"changedFiles"`
}

// FileInfo 文件信息
type FileInfo struct {
	Status     string `json:"status"`
	StatusText string `json:"statusText"`
	FilePath   string `json:"filePath"`
	Staged     bool   `json:"staged"`
}

// UserInfo 用户信息
type UserInfo struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

// PlatformInfo 平台信息
type PlatformInfo struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

// GetProjectStatus 获取项目 git 状态
func (s *AppService) GetProjectStatus(path string) (*ProjectStatus, error) {
	// 先校验路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}

	branch, err := s.gitClient.Branch(path)
	if err != nil {
		return nil, fmt.Errorf("获取分支失败: %w", err)
	}

	remoteURL, _ := s.gitClient.RemoteURL(path)

	return &ProjectStatus{
		Branch:       strings.TrimSpace(branch),
		RemoteURL:    remoteURL,
		ChangedFiles: []FileInfo{},
	}, nil
}

// GetProjectChangedFiles 获取项目变更文件列表（可能较慢）
func (s *AppService) GetProjectChangedFiles(path string) ([]FileInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}

	changes, err := s.gitClient.ChangedFiles(path)
	if err != nil {
		return nil, fmt.Errorf("获取变更文件失败: %w", err)
	}

	var files []FileInfo
	for _, c := range changes {
		files = append(files, FileInfo{
			Status:     c.Status,
			StatusText: c.StatusText(),
			FilePath:   c.FilePath,
			Staged:     c.Staged,
		})
	}

	return files, nil
}

// GetFileContent 获取文件内容
func (s *AppService) GetFileContent(projectPath, filePath string) (string, error) {
	fullPath := filepath.Join(projectPath, filePath)

	// 检查文件大小，超过 2MB 不读取
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	if info.Size() > 2*1024*1024 {
		return "", fmt.Errorf("文件过大 (%.1f MB)，不支持预览", float64(info.Size())/1024/1024)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}

	// 检测二进制文件：取前 8KB 检查是否含 NUL 字节
	checkLen := len(data)
	if checkLen > 8192 {
		checkLen = 8192
	}
	for _, b := range data[:checkLen] {
		if b == 0 {
			return "", fmt.Errorf("二进制文件，不支持预览")
		}
	}

	return string(data), nil
}

// GetFileDiff 获取文件 diff
func (s *AppService) GetFileDiff(projectPath, filePath string) (string, error) {
	return s.gitClient.DiffFile(projectPath, filePath)
}

// StageFiles 暂存指定文件
func (s *AppService) StageFiles(path string, files []string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if len(files) == 0 {
		return fmt.Errorf("未指定文件")
	}
	_, err := s.gitClient.Add(path, files...)
	return err
}

// UnstageFiles 取消暂存指定文件
func (s *AppService) UnstageFiles(path string, files []string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if len(files) == 0 {
		return fmt.Errorf("未指定文件")
	}
	_, err := s.gitClient.Reset(path, files...)
	return err
}

// StageAll 暂存所有变更文件
func (s *AppService) StageAll(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.Add(path, ".")
	return err
}

// UnstageAll 取消暂存所有文件
func (s *AppService) UnstageAll(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.Reset(path, ".")
	return err
}

// CommitChanges 提交已暂存的更改
func (s *AppService) CommitChanges(path, message string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("提交信息不能为空")
	}
	_, err := s.gitClient.Commit(path, message)
	return err
}

// DiscardFiles 丢弃工作区指定文件的更改
func (s *AppService) DiscardFiles(path string, files []string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if len(files) == 0 {
		return fmt.Errorf("未指定文件")
	}

	// 对每个文件判断：未跟踪的用 clean 删除，已跟踪的用 checkout 还原
	changes, err := s.gitClient.ChangedFiles(path)
	if err != nil {
		return fmt.Errorf("获取变更状态失败: %w", err)
	}
	untrackedMap := make(map[string]bool)
	for _, c := range changes {
		if c.Status == "?" {
			untrackedMap[c.FilePath] = true
		}
	}

	var trackedFiles, untrackedFiles []string
	for _, f := range files {
		if untrackedMap[f] {
			untrackedFiles = append(untrackedFiles, f)
		} else {
			trackedFiles = append(trackedFiles, f)
		}
	}
	if len(trackedFiles) > 0 {
		if _, err := s.gitClient.Restore(path, trackedFiles...); err != nil {
			return fmt.Errorf("还原文件失败: %w", err)
		}
	}
	if len(untrackedFiles) > 0 {
		if _, err := s.gitClient.CleanFiles(path, untrackedFiles...); err != nil {
			return fmt.Errorf("删除未跟踪文件失败: %w", err)
		}
	}
	return nil
}

// GetFileDiffStaged 获取已暂存文件的 diff
func (s *AppService) GetFileDiffStaged(projectPath, filePath string) (string, error) {
	return s.gitClient.DiffStagedFile(projectPath, filePath)
}

// PullProject 拉取项目（当前分支）
func (s *AppService) PullProject(path string) (string, error) {
	branch, err := s.gitClient.Branch(path)
	if err != nil {
		return "", fmt.Errorf("获取当前分支失败: %w", err)
	}
	return s.gitClient.Run(path, "pull", "origin", strings.TrimSpace(branch))
}

// PushProject 推送项目（当前分支）
func (s *AppService) PushProject(path string) (string, error) {
	branch, err := s.gitClient.Branch(path)
	if err != nil {
		return "", fmt.Errorf("获取当前分支失败: %w", err)
	}
	return s.gitClient.Run(path, "push", "origin", strings.TrimSpace(branch))
}

// GetCommitDiff 获取指定提交的 diff
func (s *AppService) GetCommitDiff(path, hash string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("项目路径不存在: %s", path)
	}
	return s.gitClient.CommitShow(path, hash)
}

// CommitFileInfo 提交中的文件变更信息
type CommitFileInfo struct {
	Status   string `json:"status"`
	FilePath string `json:"filePath"`
}

// GetCommitFiles 获取指定提交中变更的文件列表
func (s *AppService) GetCommitFiles(path, hash string) ([]CommitFileInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	out, err := s.gitClient.CommitFiles(path, hash)
	if err != nil {
		return nil, fmt.Errorf("获取提交文件列表失败: %w", err)
	}
	var files []CommitFileInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		files = append(files, CommitFileInfo{
			Status:   parts[0],
			FilePath: parts[len(parts)-1],
		})
	}
	return files, nil
}

// GetCommitFileDiff 获取指定提交中某个文件的 diff
func (s *AppService) GetCommitFileDiff(path, hash, filePath string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("项目路径不存在: %s", path)
	}
	return s.gitClient.CommitFileDiff(path, hash, filePath)
}

// FetchProject 拉取远程信息
func (s *AppService) FetchProject(path string) (string, error) {
	return s.gitClient.Fetch(path)
}

// CommitLog 提交记录
type CommitLog struct {
	Hash      string `json:"hash"`
	ShortHash string `json:"shortHash"`
	Author    string `json:"author"`
	Email     string `json:"email"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
	Pushed    bool   `json:"pushed"`
}

// BranchInfo 分支信息
type BranchInfo struct {
	Name    string `json:"name"`
	Current bool   `json:"current"`
}

// GetCommitLog 获取提交历史
func (s *AppService) GetCommitLog(path string, count int) ([]CommitLog, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	if count <= 0 {
		count = 50
	}
	out, err := s.gitClient.Log(path, count)
	if err != nil {
		return nil, fmt.Errorf("获取提交历史失败: %w", err)
	}
	logs := parseCommitLog(out)

	// 获取当前分支名，查询未推送的提交
	branch, brErr := s.gitClient.Branch(path)
	if brErr == nil {
		branch = strings.TrimSpace(branch)
		unpushedOut, upErr := s.gitClient.UnpushedCommits(path, branch)
		unpushedSet := make(map[string]bool)
		if upErr == nil {
			for _, h := range strings.Split(strings.TrimSpace(unpushedOut), "\n") {
				if h != "" {
					unpushedSet[h] = true
				}
			}
		}
		for i := range logs {
			logs[i].Pushed = !unpushedSet[logs[i].Hash]
		}
	}

	return logs, nil
}

// RevertCommit 撤回指定提交（生成一个反向提交）
func (s *AppService) RevertCommit(path, hash string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if strings.TrimSpace(hash) == "" {
		return fmt.Errorf("提交哈希不能为空")
	}
	_, err := s.gitClient.RevertCommit(path, strings.TrimSpace(hash))
	return err
}

func parseCommitLog(output string) []CommitLog {
	var logs []CommitLog
	entries := strings.Split(output, "---END---")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		lines := strings.SplitN(entry, "\n", 6)
		if len(lines) < 6 {
			continue
		}
		var ts int64
		fmt.Sscanf(lines[4], "%d", &ts)
		logs = append(logs, CommitLog{
			Hash:      lines[0],
			ShortHash: lines[1],
			Author:    lines[2],
			Email:     lines[3],
			Timestamp: ts,
			Message:   lines[5],
		})
	}
	return logs
}

// GetBranches 获取所有本地分支
func (s *AppService) GetBranches(path string) ([]BranchInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	out, err := s.gitClient.BranchList(path)
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %w", err)
	}
	var branches []BranchInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		name := parts[0]
		current := len(parts) > 1 && strings.TrimSpace(parts[1]) == "*"
		branches = append(branches, BranchInfo{Name: name, Current: current})
	}
	return branches, nil
}

// ResetProject 版本回滚（git reset）
// mode: "hard"(丢弃所有更改), "soft"(保留更改到暂存区), "mixed"(保留更改到工作区)
func (s *AppService) ResetProject(path, hash, mode string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if strings.TrimSpace(hash) == "" {
		return fmt.Errorf("提交哈希不能为空")
	}
	allowed := map[string]bool{"hard": true, "soft": true, "mixed": true}
	if !allowed[mode] {
		mode = "hard"
	}
	_, err := s.gitClient.ResetToCommit(path, strings.TrimSpace(hash), mode)
	return err
}

// SwitchBranch 切换分支
func (s *AppService) SwitchBranch(path, branch string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if strings.TrimSpace(branch) == "" {
		return fmt.Errorf("分支名不能为空")
	}
	_, err := s.gitClient.Checkout(path, strings.TrimSpace(branch))
	return err
}

// TagInfo 标签信息
type TagInfo struct {
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// GetTags 获取所有标签
func (s *AppService) GetTags(path string) ([]TagInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	out, err := s.gitClient.TagList(path)
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %w", err)
	}
	var tags []TagInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 3 {
			continue
		}
		var ts int64
		fmt.Sscanf(parts[2], "%d", &ts)
		hash := parts[1]
		// 注释标签的实际提交哈希在 *objectname
		if len(parts) > 3 && parts[3] != "" {
			hash = parts[3]
		}
		message := ""
		if len(parts) > 4 {
			message = parts[4]
		}
		tags = append(tags, TagInfo{
			Name:      parts[0],
			Hash:      hash,
			Timestamp: ts,
			Message:   message,
		})
	}
	return tags, nil
}

// CreateTag 创建标签
func (s *AppService) CreateTag(path, name, message string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("标签名不能为空")
	}
	if strings.TrimSpace(message) == "" {
		message = name
	}
	_, err := s.gitClient.CreateTag(path, name, message)
	return err
}

// DeleteTag 删除标签（本地+远程）
func (s *AppService) DeleteTag(path, name string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("标签名不能为空")
	}
	// 删除本地标签
	if _, err := s.gitClient.DeleteTag(path, name); err != nil {
		return fmt.Errorf("删除本地标签失败: %w", err)
	}
	// 尝试删除远程标签（忽略错误，可能未推送过）
	s.gitClient.DeleteRemoteTag(path, name)
	return nil
}

// PushTag 推送标签到远程
func (s *AppService) PushTag(path, name string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("标签名不能为空")
	}
	_, err := s.gitClient.PushTag(path, name)
	return err
}
