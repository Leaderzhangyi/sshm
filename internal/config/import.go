package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanUserPath(p string) string {
	p = strings.TrimSpace(p)
	if len(p) >= 2 && (p[0] == '"' || p[0] == '\'') && p[len(p)-1] == p[0] {
		p = p[1 : len(p)-1]
	}
	p = ExpandPath(p)
	p = os.ExpandEnv(p)
	p = filepath.Clean(p)
	return p
}

func ImportFinalShell(rootDir string, cfg *Config) (imported int, skipped int, err error) {
	rootDir = CleanUserPath(rootDir)

	info, statErr := os.Stat(rootDir)
	if statErr != nil {
		return 0, 0, fmt.Errorf("目录不存在: %w", statErr)
	}
	if !info.IsDir() {
		return 0, 0, fmt.Errorf("路径不是目录: %s", rootDir)
	}

	walkErr := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		var node struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user_name"`
			Name     string `json:"name"`
			Password string `json:"password"`
			KeyPath  string `json:"secret_key_id"`
			AuthType int    `json:"authentication_type"`
		}
		if jsonErr := json.Unmarshal(raw, &node); jsonErr != nil {
			return nil
		}

		if node.Host == "" {
			return nil
		}

		group := ""
		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr == nil {
			dir := filepath.Dir(relPath)
			if dir != "." && dir != "" {
				group = filepath.ToSlash(dir)
			}
		}

		auth := "password"
		password := node.Password
		if node.AuthType == 0 && node.KeyPath != "" {
			auth = "key"
			password = ""
		}
		port := node.Port
		if port == 0 {
			port = 22
		}

		for _, c := range cfg.Connections {
			if c.Host == node.Host && c.Port == port && c.User == node.User {
				skipped++
				return nil
			}
		}

		cfg.Connections = append(cfg.Connections, Connection{
			Name:     node.Name,
			Host:     node.Host,
			Port:     port,
			User:     node.User,
			Auth:     auth,
			Password: password,
			KeyPath:  node.KeyPath,
			Group:    group,
		})
		imported++
		return nil
	})

	if walkErr != nil {
		return imported, skipped, walkErr
	}

	return imported, skipped, nil
}
