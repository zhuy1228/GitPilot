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
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()
	header := []string{"-C", path, "-c", "core.quotePath=false"}
	if g.Enabled {
		header = append(header, "-c", "http.proxy="+g.Proxy, "-c", "https.proxy="+g.Proxy)
	}
	argsArr := append(header, args...)
	log.Println(argsArr)
	cmd := exec.CommandContext(ctx, "git", argsArr...)
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

func (g *GitClient) Status(path string) (string, error) {
	return g.Run(path, "status")
}

func (g *GitClient) Clone(repoURL, path string) (string, error) {
	return g.Run(".", "clone", repoURL, path)
}

func (g *GitClient) Fetch(path string) (string, error) {
	return g.Run(path, "fetch", "--all")
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
