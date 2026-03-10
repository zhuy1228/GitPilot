package internal

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// readSystemProxy 从 Windows 注册表读取系统代理设置
func readSystemProxy(info *ProxyInfo) error {
	key, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return fmt.Errorf("无法打开注册表键: %w", err)
	}
	defer key.Close()

	// 读取代理是否启用 (ProxyEnable: 0=关闭, 1=开启)
	proxyEnable, _, err := key.GetIntegerValue("ProxyEnable")
	if err != nil {
		return fmt.Errorf("无法读取 ProxyEnable: %w", err)
	}
	info.Enabled = proxyEnable == 1

	// 读取代理服务器地址
	proxyServer, _, err := key.GetStringValue("ProxyServer")
	if err == nil && proxyServer != "" {
		info.Server = proxyServer
		// 根据端口推断代理协议
		info.Protocol = guessProtocol(proxyServer)
	}

	// 读取代理绕过列表
	bypass, _, err := key.GetStringValue("ProxyOverride")
	if err == nil {
		info.Bypass = bypass
	}

	return nil
}
