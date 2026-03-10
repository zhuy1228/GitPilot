package git

import (
	"os/exec"
	"syscall"
)

// hideWindow 在 Windows 上隐藏子进程的控制台窗口
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
