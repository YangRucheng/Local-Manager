// Package config 提供本地电气台账的运行时配置。
package config

import (
	"os"
	"path/filepath"
)

// Config 描述应用的运行配置。
type Config struct {
	// Port 监听端口，默认 5288。
	Port string
	// DataDir 数据目录（sqlite 与 annex 子目录均位于其下），默认 ./data。
	DataDir string
}

// FromEnv 从环境变量读取配置并填充默认值。
//
//	PORT     监听端口，默认 5288
//	DATA_DIR 数据目录，默认 ./data
func FromEnv() Config {
	return Config{
		Port:    envOr("PORT", "5288"),
		DataDir: envOr("DATA_DIR", "./data"),
	}
}

// AnnexDir 返回附件存储目录（DATA_DIR/annex）。
func (c Config) AnnexDir() string {
	return filepath.Join(c.DataDir, "annex")
}

// DBPath 返回 sqlite 数据库文件路径。
func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "app.db")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
