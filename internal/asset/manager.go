package asset

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager struct {
	CacheDir string
}

func NewManager(cacheDir string) *Manager {
	return &Manager{CacheDir: cacheDir}
}

// EnsureCache clones or pulls the asset registry into the local cache.
// registryURL is like "github.com/castle-x/wk-ui".
// subDir is "ui" or "api" — determines the cache subdirectory.
func (m *Manager) EnsureCache(registryURL, subDir string) error {
	cacheDir := filepath.Join(m.CacheDir, subDir)

	gitDir := filepath.Join(cacheDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		cmd := exec.Command("git", "-C", cacheDir, "pull", "--ff-only")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git pull %s: %w", cacheDir, err)
		}
		return nil
	}

	gitURL := toGitURL(registryURL)

	if err := os.MkdirAll(filepath.Dir(cacheDir), 0755); err != nil {
		return fmt.Errorf("create cache parent: %w", err)
	}

	cmd := exec.Command("git", "clone", gitURL, cacheDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", gitURL, err)
	}
	return nil
}

// GetRegistry loads the registry.json from the cached asset repository.
func (m *Manager) GetRegistry(subDir string) (Registry, error) {
	regPath := filepath.Join(m.CacheDir, subDir, "registry.json")
	return LoadRegistry(regPath)
}

// GetAssetDir returns the full path to an asset version in the cache.
func (m *Manager) GetAssetDir(subDir, assetPath string) string {
	return filepath.Join(m.CacheDir, subDir, assetPath)
}

func toGitURL(registryURL string) string {
	if strings.HasPrefix(registryURL, "git@") || strings.HasPrefix(registryURL, "https://") {
		return registryURL
	}
	return "git@" + strings.Replace(registryURL, "/", ":", 1) + ".git"
}
