package app

import (
	"os/exec"
	"syscall"
)

// hideWindowCmd 在 Windows 上隐藏子进程的控制台窗口
func hideWindowCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}
