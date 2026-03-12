package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectItem 项目配置（以路径为核心）
type ProjectItem struct {
	Name     string `yaml:"name"`
	Path     string `yaml:"path"`
	Group    string `yaml:"group,omitempty"`     // 所属分组名，可为空表示"未分组"
	UseProxy *bool  `yaml:"use_proxy,omitempty"` // 是否走系统代理，nil 表示跟随全局
}

// Group 项目分组
type Group struct {
	Name string `yaml:"name"`
	Icon string `yaml:"icon,omitempty"` // 可选图标标识
}

// Credential 凭证信息（独立于项目）
type Credential struct {
	Platform string `yaml:"platform"`           // github / gitee / gitea / ...
	BaseURL  string `yaml:"base_url,omitempty"` // 自建平台地址
	Username string `yaml:"username"`
	Token    string `yaml:"token,omitempty"`
}

// Settings 应用设置
type Settings struct {
	Concurrency  int    `yaml:"concurrency"`
	NetworkCheck bool   `yaml:"network_check"`
	LogLevel     string `yaml:"log_level"`
}

// AppConfig 应用配置（扁平化结构）
type AppConfig struct {
	Projects    []ProjectItem `yaml:"projects"`
	Groups      []Group       `yaml:"groups"`
	Credentials []Credential  `yaml:"credentials"`
	Settings    Settings      `yaml:"settings"`
}

// configPath 返回 config.yaml 的绝对路径（与可执行文件同目录）
func configPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(filepath.Dir(exe), "config.yaml")
}

// LoadConfig 从 config.yaml 文件加载配置
func LoadConfig() (*AppConfig, error) {
	file, err := os.Open(configPath())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var appConfig AppConfig
	if err := yaml.NewDecoder(file).Decode(&appConfig); err != nil {
		return nil, err
	}

	if appConfig.Projects == nil {
		appConfig.Projects = []ProjectItem{}
	}
	if appConfig.Groups == nil {
		appConfig.Groups = []Group{}
	}
	if appConfig.Credentials == nil {
		appConfig.Credentials = []Credential{}
	}

	return &appConfig, nil
}

// SaveConfig 将配置保存到 config.yaml 文件
func SaveConfig(cfg *AppConfig) error {
	file, err := os.OpenFile(configPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return enc.Close()
}
