package internal

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ProxyInfo 保存当前系统代理信息
type ProxyInfo struct {
	// 是否启用了系统代理
	Enabled bool
	// 系统代理服务器地址 (例如 127.0.0.1:7890)
	Server string
	// 代理协议 (http, https, socks5)
	Protocol string
	// HTTP_PROXY 环境变量
	HTTPProxy string
	// HTTPS_PROXY 环境变量
	HTTPSProxy string
	// 代理绕过列表
	Bypass string
}

// GetCurrentProxy 获取本机当前的代理设置
// 优先读取 Windows 系统代理（注册表），同时也读取环境变量中的代理配置
// 支持检测 Clash、V2Ray 等常见代理工具的配置
func GetCurrentProxy() (*ProxyInfo, error) {
	info := &ProxyInfo{}

	// 1. 从环境变量读取代理
	info.HTTPProxy = getEnvProxy("HTTP_PROXY", "http_proxy")
	info.HTTPSProxy = getEnvProxy("HTTPS_PROXY", "https_proxy")

	// 2. 从 Windows 注册表读取系统代理设置
	if err := readWindowsSystemProxy(info); err != nil {
		// 注册表读取失败不算致命错误，环境变量中可能仍有代理信息
		fmt.Printf("读取系统代理设置时出错: %v\n", err)
	}

	// 3. 如果系统代理未启用，尝试从环境变量推断
	if !info.Enabled && (info.HTTPProxy != "" || info.HTTPSProxy != "") {
		info.Enabled = true
		proxy := info.HTTPProxy
		if proxy == "" {
			proxy = info.HTTPSProxy
		}
		info.Server = extractServerFromURL(proxy)
		info.Protocol = extractProtocolFromURL(proxy)
	}

	return info, nil
}

// readWindowsSystemProxy 从 Windows 注册表读取系统代理设置
func readWindowsSystemProxy(info *ProxyInfo) error {
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

// getEnvProxy 依次检查多个环境变量名，返回第一个非空值
func getEnvProxy(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

// extractServerFromURL 从代理 URL 中提取 host:port 部分
// 例如 "http://127.0.0.1:7890" -> "127.0.0.1:7890"
func extractServerFromURL(proxyURL string) string {
	s := proxyURL
	// 去掉协议前缀
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
	// 去掉尾部斜杠
	s = strings.TrimRight(s, "/")
	return s
}

// extractProtocolFromURL 从代理 URL 中提取协议
// 例如 "socks5://127.0.0.1:7891" -> "socks5"
func extractProtocolFromURL(proxyURL string) string {
	if idx := strings.Index(proxyURL, "://"); idx != -1 {
		return proxyURL[:idx]
	}
	return "http"
}

// guessProtocol 根据代理服务器地址猜测代理协议
// Clash 默认: 7890(http), 7891(socks5)
// V2Ray 默认: 10808(socks5), 10809(http)
func guessProtocol(server string) string {
	knownPorts := map[string]string{
		"7890":  "http",   // Clash HTTP
		"7891":  "socks5", // Clash SOCKS5
		"7892":  "http",   // Clash mixed
		"10808": "socks5", // V2Ray SOCKS5
		"10809": "http",   // V2Ray HTTP
		"1080":  "socks5", // 通用 SOCKS5
		"1081":  "http",   // 通用 HTTP
		"8080":  "http",   // 通用 HTTP
	}

	parts := strings.Split(server, ":")
	if len(parts) >= 2 {
		port := parts[len(parts)-1]
		if protocol, ok := knownPorts[port]; ok {
			return protocol
		}
	}
	return "http"
}

// String 返回代理信息的可读字符串
func (p *ProxyInfo) String() string {
	if !p.Enabled {
		return "当前未检测到系统代理"
	}

	var sb strings.Builder
	sb.WriteString("当前系统代理信息:\n")
	sb.WriteString("  状态:   已启用\n")
	fmt.Fprintf(&sb, "  服务器: %s\n", p.Server)
	fmt.Fprintf(&sb, "  协议:   %s\n", p.Protocol)

	if p.Bypass != "" {
		fmt.Fprintf(&sb, "  绕过:   %s\n", p.Bypass)
	}
	if p.HTTPProxy != "" {
		fmt.Fprintf(&sb, "  HTTP_PROXY:  %s\n", p.HTTPProxy)
	}
	if p.HTTPSProxy != "" {
		fmt.Fprintf(&sb, "  HTTPS_PROXY: %s\n", p.HTTPSProxy)
	}

	return sb.String()
}
