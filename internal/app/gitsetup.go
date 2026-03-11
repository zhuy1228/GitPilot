package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zhuy1228/GitPilot/internal/git"
)

// GitStatus Git 安装状态
type GitStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Path      string `json:"path"`
}

// GitInstallProgress 安装进度
type GitInstallProgress struct {
	Phase   string  `json:"phase"`   // downloading, installing, done, error
	Percent float64 `json:"percent"` // 下载百分比 0-100
	Message string  `json:"message"`
}

// CheckGitInstalled 检查 Git 是否已安装
func (s *AppService) CheckGitInstalled() GitStatus {
	path, err := exec.LookPath("git")
	if err != nil {
		log.Println("Git 未安装:", err)
		return GitStatus{Installed: false}
	}

	// 获取版本号
	cmd := exec.Command("git", "--version")
	hideWindowCmd(cmd)
	output, err := cmd.Output()
	if err != nil {
		log.Println("获取 Git 版本失败:", err)
		return GitStatus{Installed: true, Path: path}
	}

	version := strings.TrimSpace(string(output))
	// "git version 2.43.0.windows.1" -> "2.43.0"
	version = strings.TrimPrefix(version, "git version ")
	if idx := strings.Index(version, ".windows"); idx > 0 {
		version = version[:idx]
	}

	return GitStatus{
		Installed: true,
		Version:   version,
		Path:      path,
	}
}

// SelectGitInstallDir 打开文件夹选择器让用户选择 Git 安装路径
func (s *AppService) SelectGitInstallDir() (string, error) {
	if s.app == nil {
		return "", fmt.Errorf("应用未初始化")
	}
	path, err := s.app.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle("选择 Git 安装路径").
		PromptForSingleSelection()
	if err != nil {
		return "", fmt.Errorf("打开文件夹选择器失败: %w", err)
	}
	return path, nil
}

// getGitDownloadURL 根据平台返回 Git 下载地址
func getGitDownloadURL() string {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "arm64" {
			return "https://github.com/git-for-windows/git/releases/download/v2.49.0.windows.1/Git-2.49.0-arm64.exe"
		}
		return "https://github.com/git-for-windows/git/releases/download/v2.49.0.windows.1/Git-2.49.0-64-bit.exe"
	case "darwin":
		return "" // macOS 建议使用 brew 或 Xcode CLI
	default:
		return "" // Linux 建议使用包管理器
	}
}

// InstallGit 下载并安装 Git
// installDir: 用户选择的安装目录（Windows 有效）
func (s *AppService) InstallGit(installDir string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("自动安装仅支持 Windows，请使用系统包管理器安装 Git")
	}

	url := getGitDownloadURL()
	if url == "" {
		return fmt.Errorf("无法获取 Git 下载地址")
	}

	// 1. 下载安装包
	s.emitInstallProgress("downloading", 0, "正在下载 Git 安装包...")
	log.Println("开始下载 Git:", url)

	tmpDir := os.TempDir()
	installerPath := filepath.Join(tmpDir, "Git-Installer.exe")

	err := s.downloadFile(url, installerPath)
	if err != nil {
		s.emitInstallProgress("error", 0, "下载失败: "+err.Error())
		return fmt.Errorf("下载 Git 安装包失败: %w", err)
	}
	defer os.Remove(installerPath)

	// 2. 静默安装
	s.emitInstallProgress("installing", 100, "正在安装 Git，请稍候...")
	log.Println("开始安装 Git 到:", installDir)

	// Git for Windows 静默安装参数
	args := []string{
		"/VERYSILENT",
		"/NORESTART",
		"/NOCANCEL",
		"/SP-",
		"/CLOSEAPPLICATIONS",
		"/RESTARTAPPLICATIONS",
		"/COMPONENTS=icons,ext,ext\\shellhere,ext\\guihere,gitlfs,assoc,assoc_sh,autoupdate",
	}
	if installDir != "" {
		args = append(args, "/DIR="+installDir)
	}

	cmd := exec.Command(installerPath, args...)
	hideWindowCmd(cmd)
	err = cmd.Run()
	if err != nil {
		s.emitInstallProgress("error", 0, "安装失败: "+err.Error())
		return fmt.Errorf("安装 Git 失败: %w", err)
	}

	// 3. 验证安装
	time.Sleep(2 * time.Second) // 等待 PATH 更新
	status := s.CheckGitInstalled()
	if !status.Installed {
		// 尝试在指定目录查找
		gitExe := filepath.Join(installDir, "bin", "git.exe")
		if _, err := os.Stat(gitExe); err == nil {
			s.emitInstallProgress("done", 100, "Git 安装成功！路径: "+gitExe)
			// 重新初始化 gitClient
			s.reinitGitClient()
			return nil
		}
		s.emitInstallProgress("error", 0, "安装完成但未检测到 Git，请重启应用重试")
		return fmt.Errorf("安装完成但未检测到 Git")
	}

	s.emitInstallProgress("done", 100, "Git 安装成功！版本: "+status.Version)
	s.reinitGitClient()
	return nil
}

// reinitGitClient 重新初始化 Git 客户端
func (s *AppService) reinitGitClient() {
	s.gitClient = git.NewGitClient()
}

// emitInstallProgress 向前端发送安装进度事件
func (s *AppService) emitInstallProgress(phase string, percent float64, message string) {
	if s.app != nil {
		s.app.Event.Emit("git-install-progress", GitInstallProgress{
			Phase:   phase,
			Percent: percent,
			Message: message,
		})
	}
}

// downloadFile 下载文件并通过事件报告进度
func (s *AppService) downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastReport := time.Now()

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := out.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)

			// 每 300ms 报告一次进度
			if time.Since(lastReport) > 300*time.Millisecond {
				var pct float64
				if totalSize > 0 {
					pct = float64(downloaded) / float64(totalSize) * 100
				}
				sizeMB := float64(downloaded) / 1024 / 1024
				msg := fmt.Sprintf("正在下载... %.1f MB", sizeMB)
				if totalSize > 0 {
					totalMB := float64(totalSize) / 1024 / 1024
					msg = fmt.Sprintf("正在下载... %.1f / %.1f MB", sizeMB, totalMB)
				}
				s.emitInstallProgress("downloading", pct, msg)
				lastReport = time.Now()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	return nil
}
