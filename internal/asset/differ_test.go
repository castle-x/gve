package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiffAsset_Unchanged(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()

	content := []byte("const x = 1;\n")
	os.WriteFile(filepath.Join(local, "index.ts"), content, 0644)
	os.WriteFile(filepath.Join(cache, "index.ts"), content, 0644)

	diffs, err := DiffAsset(local, cache, []string{"index.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if diffs[0].Status != "unchanged" {
		t.Errorf("status = %q, want unchanged", diffs[0].Status)
	}
}

func TestDiffAsset_Modified(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()

	os.WriteFile(filepath.Join(cache, "index.ts"), []byte("const x = 1;\n"), 0644)
	os.WriteFile(filepath.Join(local, "index.ts"), []byte("const x = 2;\n"), 0644)

	diffs, err := DiffAsset(local, cache, []string{"index.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if diffs[0].Status != "modified" {
		t.Errorf("status = %q, want modified", diffs[0].Status)
	}
	if diffs[0].Diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestDiffAsset_Deleted(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()

	os.WriteFile(filepath.Join(cache, "index.ts"), []byte("const x = 1;\n"), 0644)

	diffs, err := DiffAsset(local, cache, []string{"index.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if diffs[0].Status != "deleted" {
		t.Errorf("status = %q, want deleted", diffs[0].Status)
	}
}

func TestDiffAsset_Added(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()

	os.WriteFile(filepath.Join(local, "extra.ts"), []byte("const y = 1;\n"), 0644)

	diffs, err := DiffAsset(local, cache, []string{"extra.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if diffs[0].Status != "added" {
		t.Errorf("status = %q, want added", diffs[0].Status)
	}
}

func TestHasLocalChanges_True(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()

	os.WriteFile(filepath.Join(cache, "a.ts"), []byte("original\n"), 0644)
	os.WriteFile(filepath.Join(local, "a.ts"), []byte("modified\n"), 0644)

	if !HasLocalChanges(local, cache, []string{"a.ts"}) {
		t.Error("expected HasLocalChanges to return true")
	}
}

func TestHasLocalChanges_False(t *testing.T) {
	local := t.TempDir()
	cache := t.TempDir()

	content := []byte("same\n")
	os.WriteFile(filepath.Join(cache, "a.ts"), content, 0644)
	os.WriteFile(filepath.Join(local, "a.ts"), content, 0644)

	if HasLocalChanges(local, cache, []string{"a.ts"}) {
		t.Error("expected HasLocalChanges to return false")
	}
}
