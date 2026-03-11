//go:build !windows

package app

import "os/exec"

// hideWindowCmd 非 Windows 平台无需处理
func hideWindowCmd(cmd *exec.Cmd) {}
