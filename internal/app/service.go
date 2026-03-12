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
			Projects:    []config.ProjectItem{},
			Groups:      []config.Group{},
			Credentials: []config.Credential{},
			Settings:    config.Settings{Concurrency: 6, NetworkCheck: true, LogLevel: "info"},
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

// --- 项目/分组 树形结构 ---

// TreeNode 前端侧边栏树节点
type TreeNode struct {
	Key      string     `json:"key"`
	Label    string     `json:"label"`
	Type     string     `json:"type"` // group, project
	Path     string     `json:"path,omitempty"`
	Icon     string     `json:"icon,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// GetProjectTree 获取项目树，按分组组织
func (s *AppService) GetProjectTree() []TreeNode {
	// 按分组归类项目
	groupProjects := make(map[string][]config.ProjectItem)
	for _, proj := range s.config.Projects {
		g := proj.Group
		if g == "" {
			g = "未分组"
		}
		groupProjects[g] = append(groupProjects[g], proj)
	}

	// 构建分组 → icon 映射
	groupIcon := make(map[string]string)
	for _, g := range s.config.Groups {
		groupIcon[g.Name] = g.Icon
	}

	var tree []TreeNode
	// 先输出已定义的分组（保持顺序）
	rendered := make(map[string]bool)
	for _, g := range s.config.Groups {
		projects, ok := groupProjects[g.Name]
		if !ok {
			projects = nil
		}
		node := TreeNode{
			Key:   "group:" + g.Name,
			Label: g.Name,
			Type:  "group",
			Icon:  g.Icon,
		}
		for _, proj := range projects {
			node.Children = append(node.Children, TreeNode{
				Key:   "project:" + proj.Path,
				Label: proj.Name,
				Type:  "project",
				Path:  proj.Path,
			})
		}
		tree = append(tree, node)
		rendered[g.Name] = true
	}

	// 输出"未分组"或不在 groups 列表里的项目
	for groupName, projects := range groupProjects {
		if rendered[groupName] {
			continue
		}
		node := TreeNode{
			Key:   "group:" + groupName,
			Label: groupName,
			Type:  "group",
			Icon:  "folder",
		}
		for _, proj := range projects {
			node.Children = append(node.Children, TreeNode{
				Key:   "project:" + proj.Path,
				Label: proj.Name,
				Type:  "project",
				Path:  proj.Path,
			})
		}
		tree = append(tree, node)
	}

	return tree
}

// --- 项目管理 ---

// CloneProject 克隆远程仓库到本地目录，并添加到项目列表
func (s *AppService) CloneProject(repoURL, parentDir, name, group string) error {
	if repoURL == "" {
		return fmt.Errorf("仓库地址不能为空")
	}
	if parentDir == "" {
		return fmt.Errorf("目标目录不能为空")
	}
	if name == "" {
		return fmt.Errorf("项目名称不能为空")
	}

	targetPath := parentDir + "/" + name

	_, err := s.gitClient.Clone(repoURL, targetPath)
	if err != nil {
		return fmt.Errorf("克隆失败: %w", err)
	}

	return s.AddProject(name, targetPath, group)
}

// AddProject 添加项目
func (s *AppService) AddProject(name, path, group string) error {
	// 检查路径重复
	for _, proj := range s.config.Projects {
		if proj.Path == path {
			return fmt.Errorf("项目路径 %s 已存在", path)
		}
	}
	s.config.Projects = append(s.config.Projects, config.ProjectItem{
		Name:  name,
		Path:  path,
		Group: group,
	})
	// 如果分组不存在，自动创建
	if group != "" {
		found := false
		for _, g := range s.config.Groups {
			if g.Name == group {
				found = true
				break
			}
		}
		if !found {
			s.config.Groups = append(s.config.Groups, config.Group{Name: group, Icon: "folder"})
		}
	}
	return config.SaveConfig(s.config)
}

// RemoveProject 删除项目（按路径匹配）
func (s *AppService) RemoveProject(path string) error {
	for i, proj := range s.config.Projects {
		if proj.Path == path {
			s.config.Projects = append(s.config.Projects[:i], s.config.Projects[i+1:]...)
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("项目不存在: %s", path)
}

// MoveProjectToGroup 移动项目到指定分组
func (s *AppService) MoveProjectToGroup(path, group string) error {
	for i, proj := range s.config.Projects {
		if proj.Path == path {
			s.config.Projects[i].Group = group
			// 自动创建分组
			if group != "" {
				found := false
				for _, g := range s.config.Groups {
					if g.Name == group {
						found = true
						break
					}
				}
				if !found {
					s.config.Groups = append(s.config.Groups, config.Group{Name: group, Icon: "folder"})
				}
			}
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("项目不存在: %s", path)
}

// --- 分组管理 ---

// GetGroups 获取所有分组
func (s *AppService) GetGroups() []config.Group {
	return s.config.Groups
}

// AddGroup 添加分组
func (s *AppService) AddGroup(name, icon string) error {
	if name == "" {
		return fmt.Errorf("分组名称不能为空")
	}
	for _, g := range s.config.Groups {
		if g.Name == name {
			return fmt.Errorf("分组 %s 已存在", name)
		}
	}
	if icon == "" {
		icon = "folder"
	}
	s.config.Groups = append(s.config.Groups, config.Group{Name: name, Icon: icon})
	return config.SaveConfig(s.config)
}

// UpdateGroup 修改分组
func (s *AppService) UpdateGroup(oldName, newName, icon string) error {
	if newName == "" {
		return fmt.Errorf("分组名称不能为空")
	}
	for i, g := range s.config.Groups {
		if g.Name == oldName {
			// 如果改名，检查冲突并同步更新项目的分组引用
			if oldName != newName {
				for _, g2 := range s.config.Groups {
					if g2.Name == newName {
						return fmt.Errorf("分组 %s 已存在", newName)
					}
				}
				for j, proj := range s.config.Projects {
					if proj.Group == oldName {
						s.config.Projects[j].Group = newName
					}
				}
			}
			s.config.Groups[i].Name = newName
			if icon != "" {
				s.config.Groups[i].Icon = icon
			}
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("分组 %s 不存在", oldName)
}

// RemoveGroup 删除分组（分组下的项目变为"未分组"）
func (s *AppService) RemoveGroup(name string) error {
	for i, g := range s.config.Groups {
		if g.Name == name {
			s.config.Groups = append(s.config.Groups[:i], s.config.Groups[i+1:]...)
			// 将该分组下项目归入未分组
			for j, proj := range s.config.Projects {
				if proj.Group == name {
					s.config.Projects[j].Group = ""
				}
			}
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("分组 %s 不存在", name)
}

// --- 凭证管理 ---

// CredentialInfo 凭证信息（前端展示用）
type CredentialInfo struct {
	Platform string `json:"platform"`
	BaseURL  string `json:"baseUrl"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// GetCredentials 获取所有凭证
func (s *AppService) GetCredentials() []CredentialInfo {
	var result []CredentialInfo
	for _, c := range s.config.Credentials {
		result = append(result, CredentialInfo{
			Platform: c.Platform,
			BaseURL:  c.BaseURL,
			Username: c.Username,
			Token:    c.Token,
		})
	}
	return result
}

// AddCredential 添加凭证
func (s *AppService) AddCredential(platform, baseURL, username, token string) error {
	if platform == "" {
		return fmt.Errorf("平台不能为空")
	}
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	// 检查重复（同平台同用户名）
	for _, c := range s.config.Credentials {
		if c.Platform == platform && c.Username == username {
			return fmt.Errorf("凭证已存在: %s/%s", platform, username)
		}
	}
	s.config.Credentials = append(s.config.Credentials, config.Credential{
		Platform: platform,
		BaseURL:  baseURL,
		Username: username,
		Token:    token,
	})
	return config.SaveConfig(s.config)
}

// UpdateCredential 更新凭证
func (s *AppService) UpdateCredential(platform, username, baseURL, token string) error {
	for i, c := range s.config.Credentials {
		if c.Platform == platform && c.Username == username {
			s.config.Credentials[i].BaseURL = baseURL
			s.config.Credentials[i].Token = token
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("凭证不存在: %s/%s", platform, username)
}

// RemoveCredential 删除凭证
func (s *AppService) RemoveCredential(platform, username string) error {
	for i, c := range s.config.Credentials {
		if c.Platform == platform && c.Username == username {
			s.config.Credentials = append(s.config.Credentials[:i], s.config.Credentials[i+1:]...)
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("凭证不存在: %s/%s", platform, username)
}

// --- Git 操作 ---

// ProjectStatus 项目状态信息
type ProjectStatus struct {
	Branch        string       `json:"branch"`
	RemoteURL     string       `json:"remoteUrl"`
	Remotes       []RemoteItem `json:"remotes"`
	CurrentRemote string       `json:"currentRemote"`
	ChangedFiles  []FileInfo   `json:"changedFiles"`
	UseProxy      *bool        `json:"useProxy"` // nil=跟随全局, true=开启, false=关闭
}

// RemoteItem 远程仓库信息
type RemoteItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FileInfo 文件信息
type FileInfo struct {
	Status     string `json:"status"`
	StatusText string `json:"statusText"`
	FilePath   string `json:"filePath"`
	Staged     bool   `json:"staged"`
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

	// 获取所有 remote 列表
	var remotes []RemoteItem
	remoteList, remoteErr := s.gitClient.RemoteList(path)
	if remoteErr == nil {
		for _, r := range remoteList {
			remotes = append(remotes, RemoteItem{Name: r.Name, URL: r.URL})
		}
	}
	currentRemote := "origin"
	if len(remotes) > 0 {
		currentRemote = remotes[0].Name
	}

	// 获取项目代理设置
	var useProxy *bool
	for _, proj := range s.config.Projects {
		if proj.Path == path {
			useProxy = proj.UseProxy
			break
		}
	}

	return &ProjectStatus{
		Branch:        strings.TrimSpace(branch),
		RemoteURL:     remoteURL,
		Remotes:       remotes,
		CurrentRemote: currentRemote,
		ChangedFiles:  []FileInfo{},
		UseProxy:      useProxy,
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

// PullProject 拉取项目（当前分支，指定 remote）
func (s *AppService) PullProject(path, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	branch, err := s.gitClient.Branch(path)
	if err != nil {
		return "", fmt.Errorf("获取当前分支失败: %w", err)
	}
	proxy := s.GetProjectProxy(path)
	return s.gitClient.RunWithProxy(path, proxy, "pull", remote, strings.TrimSpace(branch))
}

// PushProject 推送项目（当前分支，指定 remote）
func (s *AppService) PushProject(path, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	branch, err := s.gitClient.Branch(path)
	if err != nil {
		return "", fmt.Errorf("获取当前分支失败: %w", err)
	}
	proxy := s.GetProjectProxy(path)
	return s.gitClient.RunWithProxy(path, proxy, "push", remote, strings.TrimSpace(branch))
}

// PushAllResult 一键多端推送结果
type PushAllResult struct {
	Remote  string `json:"remote"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PushToAllRemotes 一键推送到所有远程仓库（当前分支）
func (s *AppService) PushToAllRemotes(path string) ([]PushAllResult, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	branch, err := s.gitClient.Branch(path)
	if err != nil {
		return nil, fmt.Errorf("获取当前分支失败: %w", err)
	}
	branch = strings.TrimSpace(branch)

	remoteList, err := s.gitClient.RemoteList(path)
	if err != nil {
		return nil, fmt.Errorf("获取远程仓库列表失败: %w", err)
	}

	proxy := s.GetProjectProxy(path)
	var results []PushAllResult
	for _, r := range remoteList {
		result := PushAllResult{Remote: r.Name}
		_, pushErr := s.gitClient.RunWithProxy(path, proxy, "push", r.Name, branch)
		if pushErr != nil {
			result.Message = pushErr.Error()
		} else {
			result.Success = true
			result.Message = "推送成功"
		}
		results = append(results, result)
	}
	return results, nil
}

// PushTagToAllRemotes 一键推送标签到所有远程仓库
func (s *AppService) PushTagToAllRemotes(path, tagName string) ([]PushAllResult, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return nil, fmt.Errorf("标签名不能为空")
	}

	remoteList, err := s.gitClient.RemoteList(path)
	if err != nil {
		return nil, fmt.Errorf("获取远程仓库列表失败: %w", err)
	}

	proxy := s.GetProjectProxy(path)
	var results []PushAllResult
	for _, r := range remoteList {
		result := PushAllResult{Remote: r.Name}
		_, pushErr := s.gitClient.RunWithProxy(path, proxy, "push", r.Name, tagName)
		if pushErr != nil {
			result.Message = pushErr.Error()
		} else {
			result.Success = true
			result.Message = "推送成功"
		}
		results = append(results, result)
	}
	return results, nil
}

// --- 项目代理开关 ---

// GetProjectProxy 获取项目代理设置（nil 表示跟随全局）
func (s *AppService) GetProjectProxy(path string) *bool {
	for _, proj := range s.config.Projects {
		if proj.Path == path {
			return proj.UseProxy
		}
	}
	return nil
}

// SetProjectProxy 设置项目代理开关
// useProxy: true=强制启用, false=强制禁用, nil(传空)=跟随全局
func (s *AppService) SetProjectProxy(path string, useProxy *bool) error {
	for i, proj := range s.config.Projects {
		if proj.Path == path {
			s.config.Projects[i].UseProxy = useProxy
			return config.SaveConfig(s.config)
		}
	}
	return fmt.Errorf("项目不存在: %s", path)
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

// FetchProject 拉取远程信息（指定 remote，空则 fetch --all）
func (s *AppService) FetchProject(path, remote string) (string, error) {
	proxy := s.GetProjectProxy(path)
	if remote == "" {
		return s.gitClient.RunWithProxy(path, proxy, "fetch")
	}
	return s.gitClient.RunWithProxy(path, proxy, "fetch", remote)
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
func (s *AppService) DeleteTag(path, name, remote string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("标签名不能为空")
	}
	if remote == "" {
		remote = "origin"
	}
	// 删除本地标签
	if _, err := s.gitClient.DeleteTag(path, name); err != nil {
		return fmt.Errorf("删除本地标签失败: %w", err)
	}
	// 尝试删除远程标签（忽略错误，可能未推送过）
	s.gitClient.DeleteRemoteTagFrom(path, remote, name)
	return nil
}

// PushTag 推送标签到远程
func (s *AppService) PushTag(path, name, remote string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("标签名不能为空")
	}
	if remote == "" {
		remote = "origin"
	}
	_, err := s.gitClient.PushTagTo(path, remote, name)
	return err
}

// --- 分支管理 ---

// CreateBranch 创建新分支
func (s *AppService) CreateBranch(path, name string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("分支名不能为空")
	}
	_, err := s.gitClient.CreateBranch(path, name)
	return err
}

// DeleteBranch 删除本地分支
func (s *AppService) DeleteBranch(path, name string, force bool) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("分支名不能为空")
	}
	var err error
	if force {
		_, err = s.gitClient.ForceDeleteBranch(path, name)
	} else {
		_, err = s.gitClient.DeleteBranch(path, name)
	}
	return err
}

// MergeBranch 合并指定分支到当前分支
func (s *AppService) MergeBranch(path, branch string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("项目路径不存在: %s", path)
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("分支名不能为空")
	}
	return s.gitClient.MergeBranch(path, branch)
}

// --- 远程分支管理 ---

// GetRemoteBranches 获取远程分支列表
func (s *AppService) GetRemoteBranches(path, remote string) ([]BranchInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	var out string
	var err error
	if remote == "" {
		out, err = s.gitClient.RemoteBranchList(path)
	} else {
		out, err = s.gitClient.RemoteBranchListByRemote(path, remote)
	}
	if err != nil {
		return nil, fmt.Errorf("获取远程分支列表失败: %w", err)
	}
	var branches []BranchInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || strings.Contains(name, "HEAD") {
			continue
		}
		branches = append(branches, BranchInfo{Name: name, Current: false})
	}
	return branches, nil
}

// CheckoutRemoteBranch 检出远程分支到本地
func (s *AppService) CheckoutRemoteBranch(path, remoteBranch string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	remoteBranch = strings.TrimSpace(remoteBranch)
	if remoteBranch == "" {
		return fmt.Errorf("远程分支名不能为空")
	}
	// origin/feature -> feature
	localBranch := remoteBranch
	if idx := strings.Index(remoteBranch, "/"); idx != -1 {
		localBranch = remoteBranch[idx+1:]
	}
	_, err := s.gitClient.CheckoutNewBranch(path, localBranch, remoteBranch)
	return err
}

// DeleteRemoteBranch 删除远程分支
func (s *AppService) DeleteRemoteBranch(path, branch, remote string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("分支名不能为空")
	}
	if remote == "" {
		remote = "origin"
	}
	// origin/feature -> feature
	localName := branch
	if idx := strings.Index(branch, "/"); idx != -1 {
		localName = branch[idx+1:]
	}
	_, err := s.gitClient.DeleteRemoteBranchFrom(path, remote, localName)
	return err
}

// --- Stash 贮藏管理 ---

// StashInfo 贮藏信息
type StashInfo struct {
	Index     int    `json:"index"`
	Ref       string `json:"ref"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// StashSave 保存当前变更到贮藏
func (s *AppService) StashSave(path, message string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.StashSave(path, message)
	return err
}

// GetStashList 获取贮藏列表
func (s *AppService) GetStashList(path string) ([]StashInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	out, err := s.gitClient.StashList(path)
	if err != nil {
		return nil, fmt.Errorf("获取贮藏列表失败: %w", err)
	}
	var stashes []StashInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		ref := parts[0]
		message := parts[1]
		var ts int64
		if len(parts) >= 3 {
			fmt.Sscanf(parts[2], "%d", &ts)
		}
		var index int
		fmt.Sscanf(ref, "stash@{%d}", &index)
		stashes = append(stashes, StashInfo{
			Index:     index,
			Ref:       ref,
			Message:   message,
			Timestamp: ts,
		})
	}
	return stashes, nil
}

// StashApply 应用贮藏（不删除）
func (s *AppService) StashApply(path string, index int) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.StashApply(path, index)
	return err
}

// StashPop 应用贮藏并删除
func (s *AppService) StashPop(path string, index int) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.StashPop(path, index)
	return err
}

// StashDrop 删除贮藏
func (s *AppService) StashDrop(path string, index int) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.StashDrop(path, index)
	return err
}

// --- 设置管理 ---

// GitConfig Git 全局配置
type GitConfig struct {
	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
}

// GetGitGlobalConfig 获取 git 全局配置
func (s *AppService) GetGitGlobalConfig() (*GitConfig, error) {
	name, _ := s.gitClient.GetGitGlobalConfig("user.name")
	email, _ := s.gitClient.GetGitGlobalConfig("user.email")
	return &GitConfig{
		UserName:  name,
		UserEmail: email,
	}, nil
}

// SetGitGlobalConfig 设置 git 全局配置
func (s *AppService) SetGitGlobalConfig(name, email string) error {
	if name != "" {
		if _, err := s.gitClient.SetGitGlobalConfig("user.name", name); err != nil {
			return fmt.Errorf("设置 user.name 失败: %w", err)
		}
	}
	if email != "" {
		if _, err := s.gitClient.SetGitGlobalConfig("user.email", email); err != nil {
			return fmt.Errorf("设置 user.email 失败: %w", err)
		}
	}
	return nil
}

// GetAppSettings 获取应用设置
func (s *AppService) GetAppSettings() *config.Settings {
	return &s.config.Settings
}

// UpdateAppSettings 更新应用设置
func (s *AppService) UpdateAppSettings(logLevel string) error {
	s.config.Settings.LogLevel = logLevel
	return config.SaveConfig(s.config)
}

// --- 远程仓库管理 ---

// GetRemotes 获取项目所有远程仓库列表
func (s *AppService) GetRemotes(path string) ([]RemoteItem, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	remoteList, err := s.gitClient.RemoteList(path)
	if err != nil {
		return nil, fmt.Errorf("获取远程仓库列表失败: %w", err)
	}
	var remotes []RemoteItem
	for _, r := range remoteList {
		remotes = append(remotes, RemoteItem{Name: r.Name, URL: r.URL})
	}
	return remotes, nil
}

// AddRemote 添加远程仓库
func (s *AppService) AddRemote(path, name, url string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" {
		return fmt.Errorf("远程名称不能为空")
	}
	if url == "" {
		return fmt.Errorf("远程地址不能为空")
	}
	_, err := s.gitClient.AddRemote(path, name, url)
	return err
}

// RemoveRemote 删除远程仓库
func (s *AppService) RemoveRemote(path, name string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("远程名称不能为空")
	}
	_, err := s.gitClient.RemoveRemote(path, name)
	return err
}

// --- 冲突处理 ---

// ConflictFileInfo 冲突文件信息
type ConflictFileInfo struct {
	FilePath string `json:"filePath"`
}

// GetConflictFiles 获取冲突文件列表
func (s *AppService) GetConflictFiles(path string) ([]ConflictFileInfo, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	out, err := s.gitClient.ConflictFiles(path)
	if err != nil {
		// 没有冲突文件时命令可能返回错误，返回空列表
		return []ConflictFileInfo{}, nil
	}
	var files []ConflictFileInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, ConflictFileInfo{FilePath: line})
	}
	return files, nil
}

// GetConflictFileContent 获取冲突文件内容（包含冲突标记）
func (s *AppService) GetConflictFileContent(projectPath, filePath string) (string, error) {
	fullPath := filepath.Join(projectPath, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("读取冲突文件失败: %w", err)
	}
	return string(data), nil
}

// ResolveConflictFile 将冲突文件标记为已解决
func (s *AppService) ResolveConflictFile(path string, files []string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	if len(files) == 0 {
		return fmt.Errorf("未指定文件")
	}
	_, err := s.gitClient.MarkResolved(path, files...)
	return err
}

// SaveConflictFile 保存冲突文件内容（手动解决冲突后保存）
func (s *AppService) SaveConflictFile(projectPath, filePath, content string) error {
	fullPath := filepath.Join(projectPath, filePath)
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

// AbortMerge 中止合并
func (s *AppService) AbortMerge(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", path)
	}
	_, err := s.gitClient.AbortMerge(path)
	return err
}

// IsMerging 检查是否处于合并状态
func (s *AppService) IsMerging(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return s.gitClient.MergeStatus(path)
}

// --- 提交搜索 ---

// SearchCommitLog 搜索提交历史
func (s *AppService) SearchCommitLog(path, keyword, author string, maxCount int) ([]CommitLog, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("项目路径不存在: %s", path)
	}
	if maxCount <= 0 {
		maxCount = 100
	}
	out, err := s.gitClient.SearchCommits(path, keyword, author, maxCount)
	if err != nil {
		return nil, fmt.Errorf("搜索提交历史失败: %w", err)
	}
	logs := parseCommitLog(out)
	// 标记推送状态
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

// --- 批量操作 ---

// ProjectOverview 项目概览信息（轻量级）
type ProjectOverview struct {
	Key        string `json:"key"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Branch     string `json:"branch"`
	HasChanges bool   `json:"hasChanges"`
	Unpushed   int    `json:"unpushed"`
	Error      string `json:"error,omitempty"`
}

// GetAllProjectOverview 获取所有项目的概览状态
func (s *AppService) GetAllProjectOverview() []ProjectOverview {
	var results []ProjectOverview
	for _, proj := range s.config.Projects {
		overview := ProjectOverview{
			Key:  proj.Path,
			Name: proj.Name,
			Path: proj.Path,
		}
		if _, err := os.Stat(proj.Path); os.IsNotExist(err) {
			overview.Error = "路径不存在"
			results = append(results, overview)
			continue
		}
		branch, hasChanges, unpushed, err := s.gitClient.QuickStatus(proj.Path)
		if err != nil {
			overview.Error = err.Error()
		} else {
			overview.Branch = branch
			overview.HasChanges = hasChanges
			overview.Unpushed = unpushed
		}
		results = append(results, overview)
	}
	return results
}

// BatchPullResult 批量 pull 结果
type BatchPullResult struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// BatchPull 批量拉取指定项目
func (s *AppService) BatchPull(paths []string, remote string) []BatchPullResult {
	if remote == "" {
		remote = "origin"
	}
	var results []BatchPullResult
	for _, path := range paths {
		result := BatchPullResult{Path: path}
		// 从路径推断名称
		result.Name = filepath.Base(path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.Message = "路径不存在"
			results = append(results, result)
			continue
		}
		branch, err := s.gitClient.Branch(path)
		if err != nil {
			result.Message = "获取分支失败: " + err.Error()
			results = append(results, result)
			continue
		}
		_, err = s.gitClient.PullFrom(path, remote, strings.TrimSpace(branch))
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = "拉取成功"
		}
		results = append(results, result)
	}
	return results
}

// BatchPush 批量推送指定项目
func (s *AppService) BatchPush(paths []string, remote string) []BatchPullResult {
	if remote == "" {
		remote = "origin"
	}
	var results []BatchPullResult
	for _, path := range paths {
		result := BatchPullResult{Path: path}
		result.Name = filepath.Base(path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			result.Message = "路径不存在"
			results = append(results, result)
			continue
		}
		branch, err := s.gitClient.Branch(path)
		if err != nil {
			result.Message = "获取分支失败: " + err.Error()
			results = append(results, result)
			continue
		}
		_, err = s.gitClient.PushTo(path, remote, strings.TrimSpace(branch))
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = "推送成功"
		}
		results = append(results, result)
	}
	return results
}
