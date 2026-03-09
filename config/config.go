package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Project struct {
	Name    string `yaml:"name"`
	Path    string `yaml:"path"`
	Enabled bool   `yaml:"enabled"` // 可选，默认 true
}

type User struct {
	Username string    `yaml:"username"`
	Token    string    `yaml:"token"` // 可选：未来扩展 API 功能
	Projects []Project `yaml:"projects"`
}

type Platform struct {
	BaseURL string `yaml:"base_url"` // GitHub/Gitee 可为空，Gitea 需要
	Users   []User `yaml:"users"`
}

type Settings struct {
	Concurrency  int    `yaml:"concurrency"`
	NetworkCheck bool   `yaml:"network_check"`
	LogLevel     string `yaml:"log_level"`
}

type AppConfig struct {
	Platforms map[string]Platform `yaml:"platforms"`
	Settings  Settings            `yaml:"settings"`
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

	var appConfig *AppConfig

	err = yaml.NewDecoder(file).Decode(&appConfig)
	if err != nil {
		return nil, err
	}

	return appConfig, nil
}

// SaveConfig 将配置保存到 config.yaml 文件
func SaveConfig(config *AppConfig) error {
	file, err := os.OpenFile(configPath(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := yaml.NewEncoder(file)
	enc.SetIndent(2)
	if err := enc.Encode(config); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return enc.Close()

}
