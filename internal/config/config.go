package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	UIRegistry  string
	APIRegistry string
	CacheDir    string
}

func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		UIRegistry:  "github.com/castle-x/wk-ui",
		APIRegistry: "github.com/castle-x/wk-api",
		CacheDir:    filepath.Join(home, ".gve", "cache"),
	}
}

func (c *Config) UICacheDir() string {
	return filepath.Join(c.CacheDir, "ui")
}

func (c *Config) APICacheDir() string {
	return filepath.Join(c.CacheDir, "api")
}
