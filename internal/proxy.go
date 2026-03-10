package internal

import (
	"fmt"
	"os"
	"strings"
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
// 优先读取系统代理（Windows 注册表 / macOS scutil / Linux 环境变量），
// 同时也读取环境变量中的代理配置
func GetCurrentProxy() (*ProxyInfo, error) {
	info := &ProxyInfo{}

	// 1. 从环境变量读取代理
	info.HTTPProxy = getEnvProxy("HTTP_PROXY", "http_proxy")
	info.HTTPSProxy = getEnvProxy("HTTPS_PROXY", "https_proxy")

	// 2. 从系统设置读取代理（平台相关）
	if err := readSystemProxy(info); err != nil {
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
	if idx := strings.Index(s, "://"); idx != -1 {
		s = s[idx+3:]
	}
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
		"7890":  "http",
		"7891":  "socks5",
		"7892":  "http",
		"10808": "socks5",
		"10809": "http",
		"1080":  "socks5",
		"1081":  "http",
		"8080":  "http",
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
