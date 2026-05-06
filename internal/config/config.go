// Package config defines connection data structures, file I/O, and path utilities.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Connection struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Auth      string `json:"auth"`
	Password  string `json:"password"`
	KeyPath   string `json:"key_path"`
	ProxyJump string `json:"proxy_jump"`
	Group     string `json:"group"`
}

type Config struct {
	Connections []Connection `json:"connections"`
}

var defaultConfig = Config{
	Connections: []Connection{
		{
			Name:  "示例-Web01",
			Host:  "192.168.1.10",
			Port:  22,
			User:  "root",
			Auth:  "password",
			Group: "隐私计算平台/stg1",
		},
		{
			Name:      "示例-DB01",
			Host:      "192.168.1.20",
			Port:      22,
			User:      "dbadmin",
			Auth:      "key",
			KeyPath:   "/home/user/.ssh/db01.pem",
			ProxyJump: "root@192.168.1.1:22",
			Group:     "隐私计算平台/stg2",
		},
	},
}

func Path() string {
	exe, err := os.Executable()
	if err != nil {
		return "connections.json"
	}
	return filepath.Join(filepath.Dir(exe), "connections.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			if werr := Save(&defaultConfig); werr != nil {
				return nil, fmt.Errorf("无法创建配置文件: %w", werr)
			}
			return &defaultConfig, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(Path(), data, 0600)
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func ParseJumpHost(jump string) (user, host string, port int) {
	port = 22
	parts := strings.SplitN(jump, "@", 2)
	if len(parts) == 2 {
		user = parts[0]
		hostPort := parts[1]
		hostParts := strings.SplitN(hostPort, ":", 2)
		host = hostParts[0]
		if len(hostParts) == 2 {
			fmt.Sscanf(hostParts[1], "%d", &port)
		}
	} else {
		host = jump
	}
	return
}
