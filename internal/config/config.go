// Package config 持久化应用配置（游戏 exe 路径等）。
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const appDirName = "ModManager"

// Config 与磁盘 config.json 字段对应。
type Config struct {
	GameExePath string `json:"game_exe_path"`
}

func filePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDirName, "config.json"), nil
}

// Load 读取配置；文件不存在时返回零值。
func Load() (Config, error) {
	var c Config
	p, err := filePath()
	if err != nil {
		return c, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

// Save 写入配置。
func Save(c Config) error {
	p, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o644)
}
