//go:build darwin

package internal

import (
	"fmt"
	"os/exec"
	"strings"
)

// readSystemProxy 从 macOS 系统偏好设置读取代理配置
// 使用 scutil --proxy 命令获取系统代理信息
func readSystemProxy(info *ProxyInfo) error {
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return fmt.Errorf("执行 scutil --proxy 失败: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	settings := make(map[string]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, " : ", 2)
		if len(parts) == 2 {
			settings[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	// 检查 HTTP 代理
	if settings["HTTPEnable"] == "1" {
		host := settings["HTTPProxy"]
		port := settings["HTTPPort"]
		if host != "" {
			info.Enabled = true
			info.Server = host
			if port != "" && port != "0" {
				info.Server = host + ":" + port
			}
			info.Protocol = "http"
		}
	}

	// 检查 HTTPS 代理
	if !info.Enabled && settings["HTTPSEnable"] == "1" {
		host := settings["HTTPSProxy"]
		port := settings["HTTPSPort"]
		if host != "" {
			info.Enabled = true
			info.Server = host
			if port != "" && port != "0" {
				info.Server = host + ":" + port
			}
			info.Protocol = "http"
		}
	}

	// 检查 SOCKS 代理
	if !info.Enabled && settings["SOCKSEnable"] == "1" {
		host := settings["SOCKSProxy"]
		port := settings["SOCKSPort"]
		if host != "" {
			info.Enabled = true
			info.Server = host
			if port != "" && port != "0" {
				info.Server = host + ":" + port
			}
			info.Protocol = "socks5"
		}
	}

	// 如果通过以上方式检测到了代理，根据端口再次猜测协议
	if info.Enabled && info.Server != "" {
		info.Protocol = guessProtocol(info.Server)
	}

	// 读取绕过列表
	if exceptions, ok := settings["ExceptionsList"]; ok {
		info.Bypass = exceptions
	}

	return nil
}
