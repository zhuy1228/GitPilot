package git

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/zhuy1228/GitPilot/internal"
)

type GitClient struct {
	Enabled bool
	Proxy   string
	Timeout time.Duration
}

func NewGitClient() *GitClient {
	proxy, _ := internal.GetCurrentProxy()
	return &GitClient{
		Enabled: proxy.Enabled,
		Proxy:   proxy.Protocol + "://" + proxy.Server,
		Timeout: 30 * time.Second,
	}
}

// Run 执行 git 命令，支持超时和代理设置
func (g *GitClient) Run(path string, args ...string) (string, error) {
	return g.RunWithProxy(path, nil, args...)
}

// RunWithProxy 执行 git 命令，useProxy 可覆盖全局代理设置
// useProxy == nil 时跟随 GitClient 自身的 Enabled 设置
func (g *GitClient) RunWithProxy(path string, useProxy *bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()
	header := []string{"-C", path, "-c", "core.quotePath=false"}
	enableProxy := g.Enabled
	if useProxy != nil {
		enableProxy = *useProxy
	}
	if enableProxy {
		header = append(header, "-c", "http.proxy="+g.Proxy, "-c", "https.proxy="+g.Proxy)
	}
	argsArr := append(header, args...)
	log.Println(argsArr)
	cmd := exec.CommandContext(ctx, "git", argsArr...)
	hideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("git command timeout: git %v", args)
	}
	if err != nil {
		return "", fmt.Errorf("git command error: %v, stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (g *GitClient) Pull(path string) (string, error) {
	return g.Run(path, "pull")
}

func (g *GitClient) Push(path string) (string, error) {
	return g.Run(path, "push")
}

// PushTo 推送到指定远程
func (g *GitClient) PushTo(path, remote, branch string) (string, error) {
	return g.Run(path, "push", remote, branch)
}

// PullFrom 从指定远程拉取
func (g *GitClient) PullFrom(path, remote, branch string) (string, error) {
	return g.Run(path, "pull", remote, branch)
}

// FetchRemote 拉取指定远程信息
func (g *GitClient) FetchRemote(path, remote string) (string, error) {
	return g.Run(path, "fetch", remote)
}

func (g *GitClient) Status(path string) (string, error) {
	return g.Run(path, "status")
}

func (g *GitClient) Clone(repoURL, path string) (string, error) {
	// Clone 操作可能需要较长时间，使用独立的超时设置（10 分钟）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	args := []string{"-c", "core.quotePath=false"}
	if g.Enabled {
		args = append(args, "-c", "http.proxy="+g.Proxy, "-c", "https.proxy="+g.Proxy)
	}
	args = append(args, "clone", "--progress", repoURL, path)
	log.Println(args)
	cmd := exec.CommandContext(ctx, "git", args...)
	hideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("git clone timeout")
	}
	if err != nil {
		return "", fmt.Errorf("git clone error: %v, stderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (g *GitClient) Fetch(path string) (string, error) {
	return g.Run(path, "fetch", "--all")
}

// RemoteList 获取所有远程仓库列表 (git remote -v)
type RemoteInfo struct {
	Name string
	URL  string
}

func (g *GitClient) RemoteList(path string) ([]RemoteInfo, error) {
	out, err := g.Run(path, "remote", "-v")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var remotes []RemoteInfo
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, RemoteInfo{Name: name, URL: parts[1]})
	}
	return remotes, nil
}

// AddRemote 添加远程仓库
func (g *GitClient) AddRemote(path, name, url string) (string, error) {
	return g.Run(path, "remote", "add", name, url)
}

// RemoveRemote 删除远程仓库
func (g *GitClient) RemoveRemote(path, name string) (string, error) {
	return g.Run(path, "remote", "remove", name)
}

func (g *GitClient) RemoteURL(path string) (string, error) {
	out, err := g.Run(path, "config", "--get", "remote.origin.url")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (g *GitClient) Branch(path string) (string, error) {
	return g.Run(path, "rev-parse", "--abbrev-ref", "HEAD")
}

func (g *GitClient) Add(path string, files ...string) (string, error) {
	args := append([]string{"add"}, files...)
	return g.Run(path, args...)
}

// Reset 取消暂存指定文件 (git reset HEAD -- files...)
func (g *GitClient) Reset(path string, files ...string) (string, error) {
	args := append([]string{"reset", "HEAD", "--"}, files...)
	return g.Run(path, args...)
}

// Commit 提交暂存区的更改
func (g *GitClient) Commit(path, message string) (string, error) {
	return g.Run(path, "commit", "-m", message)
}

// Restore 丢弃工作区指定文件的更改 (git checkout -- files...)
func (g *GitClient) Restore(path string, files ...string) (string, error) {
	args := append([]string{"checkout", "--"}, files...)
	return g.Run(path, args...)
}

// CleanFiles 删除未跟踪的文件 (git clean -f -- files...)
func (g *GitClient) CleanFiles(path string, files ...string) (string, error) {
	args := append([]string{"clean", "-f", "--"}, files...)
	return g.Run(path, args...)
}

// Log 获取提交历史 (git log --oneline --format=... -n count)
func (g *GitClient) Log(path string, count int) (string, error) {
	return g.Run(path, "log",
		fmt.Sprintf("--max-count=%d", count),
		"--format=%H%n%h%n%an%n%ae%n%at%n%s%n---END---",
	)
}

// UnpushedCommits 获取当前分支上未推送到远程的提交哈希列表（默认 origin）
func (g *GitClient) UnpushedCommits(path, branch string) (string, error) {
	return g.Run(path, "rev-list", "origin/"+branch+"..HEAD")
}

// UnpushedCommitsTo 获取当前分支上未推送到指定远程的提交哈希列表
func (g *GitClient) UnpushedCommitsTo(path, remote, branch string) (string, error) {
	return g.Run(path, "rev-list", remote+"/"+branch+"..HEAD")
}

// RevertCommit 撤回指定提交（创建一个反向提交）
func (g *GitClient) RevertCommit(path, hash string) (string, error) {
	return g.Run(path, "revert", "--no-edit", hash)
}

// TagList 获取所有标签（按版本号降序，带创建时间和提交哈希）
func (g *GitClient) TagList(path string) (string, error) {
	return g.Run(path, "tag", "-l", "--sort=-version:refname",
		"--format=%(refname:short)\t%(objectname:short)\t%(creatordate:unix)\t%(*objectname:short)\t%(contents:subject)")
}

// CreateTag 创建注释标签
func (g *GitClient) CreateTag(path, name, message string) (string, error) {
	return g.Run(path, "tag", "-a", name, "-m", message)
}

// DeleteTag 删除本地标签
func (g *GitClient) DeleteTag(path, name string) (string, error) {
	return g.Run(path, "tag", "-d", name)
}

// PushTag 推送标签到远程（默认 origin）
func (g *GitClient) PushTag(path, name string) (string, error) {
	return g.Run(path, "push", "origin", name)
}

// PushTagTo 推送标签到指定远程
func (g *GitClient) PushTagTo(path, remote, name string) (string, error) {
	return g.Run(path, "push", remote, name)
}

// DeleteRemoteTag 删除远程标签（默认 origin）
func (g *GitClient) DeleteRemoteTag(path, name string) (string, error) {
	return g.Run(path, "push", "origin", "--delete", name)
}

// DeleteRemoteTagFrom 删除指定远程的标签
func (g *GitClient) DeleteRemoteTagFrom(path, remote, name string) (string, error) {
	return g.Run(path, "push", remote, "--delete", name)
}

// BranchList 获取所有本地分支
func (g *GitClient) BranchList(path string) (string, error) {
	return g.Run(path, "branch", "--format=%(refname:short)\t%(HEAD)")
}

// Checkout 切换分支
func (g *GitClient) Checkout(path, branch string) (string, error) {
	return g.Run(path, "checkout", branch)
}

// ResetToCommit 将 HEAD 重置到指定提交 (git reset --<mode> <hash>)
// mode: hard / soft / mixed
func (g *GitClient) ResetToCommit(path, hash, mode string) (string, error) {
	if mode == "" {
		mode = "hard"
	}
	return g.Run(path, "reset", "--"+mode, hash)
}

// CommitShow 获取指定提交的详细 diff
func (g *GitClient) CommitShow(path, hash string) (string, error) {
	return g.Run(path, "show", "--format=%b", hash)
}

// CommitFiles 获取指定提交中变更的文件列表 (git diff-tree --root --no-commit-id -r --name-status <hash>)
// --root 使根提交（第一次提交）也能与空树对比，列出所有新增文件
func (g *GitClient) CommitFiles(path, hash string) (string, error) {
	return g.Run(path, "diff-tree", "--root", "--no-commit-id", "-r", "--name-status", hash)
}

// CommitFileDiff 获取指定提交中某个文件的 diff (git show <hash> -- <file>)
func (g *GitClient) CommitFileDiff(path, hash, filePath string) (string, error) {
	return g.Run(path, "show", "--format=", hash, "--", filePath)
}

// FileChange 表示单个文件的变更信息
type FileChange struct {
	// 变更状态: M(修改), A(新增), D(删除), R(重命名), C(复制), U(未合并), ?(未跟踪)
	Status string
	// 文件路径
	FilePath string
	// 重命名/复制时的原始路径
	OrigPath string
	// 是否为暂存区变更
	Staged bool
}

// ChangedFiles 获取工作区中所有变更的文件列表（包含新增、修改、删除、重命名、未跟踪等）
// 同一文件在暂存区和工作区都有变更时，会拆分为两条记录
func (g *GitClient) ChangedFiles(path string) ([]FileChange, error) {
	out, err := g.Run(path, "status", "--porcelain", "-uall")
	if err != nil {
		return nil, err
	}
	return parsePorcelainStatus(out), nil
}

// StagedFiles 获取已暂存的文件变更列表
func (g *GitClient) StagedFiles(path string) ([]FileChange, error) {
	out, err := g.Run(path, "diff", "--cached", "--name-status")
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

// DiffStat 获取工作区文件变更的统计信息（增删行数）
func (g *GitClient) DiffStat(path string) (string, error) {
	return g.Run(path, "diff", "--stat")
}

// DiffFile 获取指定文件的详细差异内容
func (g *GitClient) DiffFile(path, filePath string) (string, error) {
	return g.Run(path, "diff", "--", filePath)
}

// DiffStagedFile 获取指定已暂存文件的详细差异内容
func (g *GitClient) DiffStagedFile(path, filePath string) (string, error) {
	return g.Run(path, "diff", "--cached", "--", filePath)
}

// DiffCommit 获取两个提交之间的文件变更列表
func (g *GitClient) DiffCommit(path, from, to string) ([]FileChange, error) {
	out, err := g.Run(path, "diff", "--name-status", from, to)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

// parsePorcelainStatus 解析 git status --porcelain 的输出
// porcelain 格式: XY filename, X=暂存区状态, Y=工作区状态
// 同一文件若暂存区和工作区都有变更，会拆分为两条记录
func parsePorcelainStatus(output string) []FileChange {
	var changes []FileChange
	// 注意: 不能用 TrimSpace，porcelain 格式中行首空格表示"暂存区无变更"，TrimSpace 会错误地吃掉第一行的前导空格
	lines := strings.Split(strings.TrimRight(output, "\n\r "), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		indexStatus := string(line[0])
		workTreeStatus := string(line[1])
		filePath := strings.TrimSpace(line[2:])

		// 处理重命名情况 "R  old -> new"
		var origPath string
		if strings.Contains(filePath, " -> ") {
			parts := strings.SplitN(filePath, " -> ", 2)
			origPath = parts[0]
			filePath = parts[1]
		}

		// 未跟踪文件 (??) 只产生一条记录
		if indexStatus == "?" {
			changes = append(changes, FileChange{
				Status:   "?",
				FilePath: filePath,
			})
			continue
		}

		// 暂存区有变更 (X 不为空格)
		if indexStatus != " " {
			changes = append(changes, FileChange{
				Status:   indexStatus,
				FilePath: filePath,
				OrigPath: origPath,
				Staged:   true,
			})
		}

		// 工作区有变更 (Y 不为空格)
		if workTreeStatus != " " {
			changes = append(changes, FileChange{
				Status:   workTreeStatus,
				FilePath: filePath,
				OrigPath: origPath,
				Staged:   false,
			})
		}
	}
	return changes
}

// parseNameStatus 解析 git diff --name-status 的输出
func parseNameStatus(output string) []FileChange {
	var changes []FileChange
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		change := FileChange{
			Status:   parts[0],
			FilePath: parts[len(parts)-1],
		}
		// 重命名/复制时有原路径: R100 old_path new_path
		if len(parts) == 3 && (strings.HasPrefix(parts[0], "R") || strings.HasPrefix(parts[0], "C")) {
			change.OrigPath = parts[1]
			change.FilePath = parts[2]
		}
		changes = append(changes, change)
	}
	return changes
}

// CreateBranch 创建新分支
func (g *GitClient) CreateBranch(path, name string) (string, error) {
	return g.Run(path, "branch", name)
}

// DeleteBranch 删除本地分支 (-d 安全删除)
func (g *GitClient) DeleteBranch(path, name string) (string, error) {
	return g.Run(path, "branch", "-d", name)
}

// ForceDeleteBranch 强制删除本地分支 (-D)
func (g *GitClient) ForceDeleteBranch(path, name string) (string, error) {
	return g.Run(path, "branch", "-D", name)
}

// MergeBranch 合并指定分支到当前分支
func (g *GitClient) MergeBranch(path, branch string) (string, error) {
	return g.Run(path, "merge", branch)
}

// DeleteRemoteBranch 删除远程分支（默认 origin）
func (g *GitClient) DeleteRemoteBranch(path, branch string) (string, error) {
	return g.Run(path, "push", "origin", "--delete", branch)
}

// DeleteRemoteBranchFrom 删除指定远程的分支
func (g *GitClient) DeleteRemoteBranchFrom(path, remote, branch string) (string, error) {
	return g.Run(path, "push", remote, "--delete", branch)
}

// RemoteBranchList 获取所有远程分支
func (g *GitClient) RemoteBranchList(path string) (string, error) {
	return g.Run(path, "branch", "-r", "--format=%(refname:short)")
}

// RemoteBranchListByRemote 获取指定远程的分支
func (g *GitClient) RemoteBranchListByRemote(path, remote string) (string, error) {
	return g.Run(path, "branch", "-r", "--list", remote+"/*", "--format=%(refname:short)")
}

// CheckoutNewBranch 从远程分支检出新本地分支
func (g *GitClient) CheckoutNewBranch(path, localBranch, remoteBranch string) (string, error) {
	return g.Run(path, "checkout", "-b", localBranch, remoteBranch)
}

// StashSave 保存当前工作区变更到贮藏
func (g *GitClient) StashSave(path, message string) (string, error) {
	if message != "" {
		return g.Run(path, "stash", "push", "-m", message)
	}
	return g.Run(path, "stash", "push")
}

// StashList 获取贮藏列表
func (g *GitClient) StashList(path string) (string, error) {
	return g.Run(path, "stash", "list", "--format=%gd\t%s\t%at")
}

// StashApply 应用贮藏（不删除）
func (g *GitClient) StashApply(path string, index int) (string, error) {
	return g.Run(path, "stash", "apply", fmt.Sprintf("stash@{%d}", index))
}

// StashPop 应用贮藏并删除
func (g *GitClient) StashPop(path string, index int) (string, error) {
	return g.Run(path, "stash", "pop", fmt.Sprintf("stash@{%d}", index))
}

// StashDrop 删除贮藏
func (g *GitClient) StashDrop(path string, index int) (string, error) {
	return g.Run(path, "stash", "drop", fmt.Sprintf("stash@{%d}", index))
}

// GetGitGlobalConfig 获取 git 全局配置
func (g *GitClient) GetGitGlobalConfig(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "--get", key)
	hideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", nil // key 不存在不算错误
	}
	return strings.TrimSpace(stdout.String()), nil
}

// SetGitGlobalConfig 设置 git 全局配置
func (g *GitClient) SetGitGlobalConfig(key, value string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "config", "--global", key, value)
	hideWindow(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git config error: %v, stderr: %s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// --- 冲突处理 ---

// ConflictFiles 获取冲突文件列表 (git diff --name-only --diff-filter=U)
func (g *GitClient) ConflictFiles(path string) (string, error) {
	return g.Run(path, "diff", "--name-only", "--diff-filter=U")
}

// GetConflictContent 获取冲突文件的完整内容（含冲突标记）
func (g *GitClient) GetConflictContent(path, filePath string) (string, error) {
	return g.Run(path, "show", ":0:"+filePath)
}

// MarkResolved 将冲突文件标记为已解决 (git add <file>)
func (g *GitClient) MarkResolved(path string, files ...string) (string, error) {
	args := append([]string{"add"}, files...)
	return g.Run(path, args...)
}

// AbortMerge 中止合并 (git merge --abort)
func (g *GitClient) AbortMerge(path string) (string, error) {
	return g.Run(path, "merge", "--abort")
}

// MergeStatus 检查是否处于合并状态
func (g *GitClient) MergeStatus(path string) bool {
	// 检查 .git/MERGE_HEAD 文件是否存在
	_, err := g.Run(path, "rev-parse", "--verify", "MERGE_HEAD")
	return err == nil
}

// --- 提交搜索 ---

// SearchCommits 搜索提交历史 (git log --grep / --author / --after / --before)
func (g *GitClient) SearchCommits(path string, keyword, author string, maxCount int) (string, error) {
	args := []string{"log"}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", maxCount))
	}
	if keyword != "" {
		args = append(args, "--grep="+keyword, "-i")
	}
	if author != "" {
		args = append(args, "--author="+author)
	}
	args = append(args, "--format=%H%n%h%n%an%n%ae%n%at%n%s%n---END---")
	return g.Run(path, args...)
}

// --- 批量操作 ---

// QuickStatus 快速获取分支名和是否有变更（轻量级）
func (g *GitClient) QuickStatus(path string) (branch string, hasChanges bool, unpushed int, err error) {
	branchOut, err := g.Branch(path)
	if err != nil {
		return "", false, 0, err
	}
	branch = strings.TrimSpace(branchOut)

	// 检查是否有变更
	statusOut, err := g.Run(path, "status", "--porcelain")
	if err == nil {
		hasChanges = strings.TrimSpace(statusOut) != ""
	}

	// 检查未推送提交数
	unpushedOut, err := g.Run(path, "rev-list", "--count", "origin/"+branch+"..HEAD")
	if err == nil {
		fmt.Sscanf(strings.TrimSpace(unpushedOut), "%d", &unpushed)
	}

	return branch, hasChanges, unpushed, nil
}

// StatusText 返回变更状态的中文描述
func (fc FileChange) StatusText() string {
	switch {
	case strings.HasPrefix(fc.Status, "R"):
		return "重命名"
	case strings.HasPrefix(fc.Status, "C"):
		return "复制"
	default:
		statusMap := map[string]string{
			"M": "已修改",
			"A": "新增",
			"D": "已删除",
			"U": "未合并",
			"?": "未跟踪",
		}
		if text, ok := statusMap[fc.Status]; ok {
			return text
		}
		return fc.Status
	}
}

// String 返回文件变更的可读字符串
func (fc FileChange) String() string {
	area := "工作区"
	if fc.Staged {
		area = "暂存区"
	}
	if fc.Status == "?" {
		return fmt.Sprintf("[%s] %s", fc.StatusText(), fc.FilePath)
	}
	if fc.OrigPath != "" {
		return fmt.Sprintf("[%s] (%s) %s → %s", fc.StatusText(), area, fc.OrigPath, fc.FilePath)
	}
	return fmt.Sprintf("[%s] (%s) %s", fc.StatusText(), area, fc.FilePath)
}
