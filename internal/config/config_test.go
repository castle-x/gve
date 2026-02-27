package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	c := Default()
	if c.UIRegistry != "github.com/castle-x/wk-ui" {
		t.Errorf("UIRegistry = %q", c.UIRegistry)
	}
	if c.APIRegistry != "github.com/castle-x/wk-api" {
		t.Errorf("APIRegistry = %q", c.APIRegistry)
	}
	if c.CacheDir == "" {
		t.Error("CacheDir is empty")
	}
	if !strings.Contains(c.CacheDir, ".gve") {
		t.Errorf("CacheDir should contain .gve, got %q", c.CacheDir)
	}
}

func TestUICacheDir(t *testing.T) {
	c := Default()
	ui := c.UICacheDir()
	if ui == "" {
		t.Error("UICacheDir is empty")
	}
	if filepath.Base(ui) != "ui" {
		t.Errorf("UICacheDir should end with ui, got %q", ui)
	}
}

func TestAPICacheDir(t *testing.T) {
	c := Default()
	api := c.APICacheDir()
	if api == "" {
		t.Error("APICacheDir is empty")
	}
	if filepath.Base(api) != "api" {
		t.Errorf("APICacheDir should end with api, got %q", api)
	}
}
