package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
	}
}

// Run 执行 git 命令，支持超时和代理设置
func (g *GitClient) Run(path string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), g.Timeout)
	defer cancel()
	header := []string{"-C", path, "git"}
	if g.Enabled {
		header = append(header, "-c", "http.proxy="+g.Proxy, "-c", "https.proxy="+g.Proxy)
	}
	argsArr := append(header, args...)
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
