package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAssetArg(t *testing.T) {
	tests := []struct {
		arg       string
		wantName  string
		wantVer   string
	}{
		{"button", "button", ""},
		{"button@1.2.0", "button", "1.2.0"},
		{"data-table@^2.0.0", "data-table", "^2.0.0"},
		{"pkg@latest", "pkg", "latest"},
	}

	for _, tt := range tests {
		name, ver := parseAssetArg(tt.arg)
		if name != tt.wantName || ver != tt.wantVer {
			t.Errorf("parseAssetArg(%q) = (%q, %q), want (%q, %q)", tt.arg, name, ver, tt.wantName, tt.wantVer)
		}
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Not in a gve project
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	_, err := findProjectRoot()
	if err == nil {
		t.Error("findProjectRoot should fail when gve.lock not found")
	}

	// Create gve.lock in current dir
	os.WriteFile(filepath.Join(dir, "gve.lock"), []byte("{}"), 0644)
	got, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot: %v", err)
	}
	if got != dir {
		t.Errorf("findProjectRoot = %q, want %q", got, dir)
	}

	// From subdir should still find root
	sub := filepath.Join(dir, "site", "src")
	os.MkdirAll(sub, 0755)
	os.Chdir(sub)
	got2, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot from subdir: %v", err)
	}
	if got2 != dir {
		t.Errorf("findProjectRoot from subdir = %q, want %q", got2, dir)
	}
}
