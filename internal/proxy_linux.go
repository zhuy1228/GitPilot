//go:build linux

package internal

import (
	"fmt"
	"os/exec"
	"strings"
)

// readSystemProxy 从 Linux 系统读取代理配置
// 优先尝试 GNOME (gsettings)，失败则回退到环境变量
func readSystemProxy(info *ProxyInfo) error {
	// 尝试通过 gsettings 读取 GNOME 系统代理
	if err := readGnomeProxy(info); err == nil && info.Enabled {
		return nil
	}

	// 尝试通过 KDE 配置读取代理
	if err := readKDEProxy(info); err == nil && info.Enabled {
		return nil
	}

	// Linux 上环境变量已由 GetCurrentProxy 中读取，此处无需额外操作
	return nil
}

// readGnomeProxy 通过 gsettings 读取 GNOME 桌面环境的代理配置
func readGnomeProxy(info *ProxyInfo) error {
	// 检查 gsettings 是否可用
	if _, err := exec.LookPath("gsettings"); err != nil {
		return fmt.Errorf("gsettings 不可用: %w", err)
	}

	// 读取代理模式
	mode, err := gsettingsGet("org.gnome.system.proxy", "mode")
	if err != nil {
		return err
	}

	if mode != "'manual'" {
		return nil // 代理未启用
	}

	// 读取 HTTP 代理
	host, err := gsettingsGet("org.gnome.system.proxy.http", "host")
	if err == nil && host != "" && host != "''" {
		host = strings.Trim(host, "'")
		port, _ := gsettingsGet("org.gnome.system.proxy.http", "port")
		port = strings.TrimSpace(port)
		if port != "" && port != "0" {
			info.Enabled = true
			info.Server = host + ":" + port
			info.Protocol = guessProtocol(info.Server)
			return nil
		}
	}

	// 读取 SOCKS 代理
	host, err = gsettingsGet("org.gnome.system.proxy.socks", "host")
	if err == nil && host != "" && host != "''" {
		host = strings.Trim(host, "'")
		port, _ := gsettingsGet("org.gnome.system.proxy.socks", "port")
		port = strings.TrimSpace(port)
		if port != "" && port != "0" {
			info.Enabled = true
			info.Server = host + ":" + port
			info.Protocol = "socks5"
			return nil
		}
	}

	return nil
}

// readKDEProxy 通过 kreadconfig5/kreadconfig6 读取 KDE 代理配置
func readKDEProxy(info *ProxyInfo) error {
	// 尝试 kreadconfig5 或 kreadconfig6
	var cmd string
	if _, err := exec.LookPath("kreadconfig6"); err == nil {
		cmd = "kreadconfig6"
	} else if _, err := exec.LookPath("kreadconfig5"); err == nil {
		cmd = "kreadconfig5"
	} else {
		return fmt.Errorf("kreadconfig 不可用")
	}

	// 读取代理类型
	proxyType, err := kdeGet(cmd, "Proxy Settings", "ProxyType")
	if err != nil || proxyType != "1" {
		return nil // 0 = 无代理, 1 = 手动
	}

	// 读取 HTTP 代理
	httpProxy, err := kdeGet(cmd, "Proxy Settings", "httpProxy")
	if err == nil && httpProxy != "" {
		info.Enabled = true
		info.Server = extractServerFromURL(httpProxy)
		info.Protocol = guessProtocol(info.Server)
		return nil
	}

	return nil
}

// gsettingsGet 执行 gsettings get 命令
func gsettingsGet(schema, key string) (string, error) {
	out, err := exec.Command("gsettings", "get", schema, key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// kdeGet 执行 kreadconfig 命令读取 KDE 配置
func kdeGet(cmd, group, key string) (string, error) {
	out, err := exec.Command(cmd, "--file", "kioslaverc", "--group", group, "--key", key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
