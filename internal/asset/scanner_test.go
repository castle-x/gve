package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifySource(t *testing.T) {
	tests := []struct {
		source  string
		wantCls ImportClass
		wantKey string
	}{
		// Relative paths → skip
		{"./utils", ImportSkip, ""},
		{"../shared/helpers", ImportSkip, ""},

		// wk-ui peerDeps
		{"@/shared/wk/ui/spinner", ImportPeerDep, "ui/spinner"},
		{"@/shared/wk/ui/data-table/types", ImportPeerDep, "ui/data-table"},
		{"@/shared/wk/components/date-picker", ImportPeerDep, "components/date-picker"},
		{"@/shared/wk/components/modal/index", ImportPeerDep, "components/modal"},

		// @/shared/ non-wk → skip
		{"@/shared/lib/utils", ImportSkip, ""},
		{"@/shared/hooks/use-auth", ImportSkip, ""},
		{"@/shared/wk/types", ImportSkip, ""}, // not ui/ or components/

		// @/ project alias → skip
		{"@/components/layout", ImportSkip, ""},
		{"@/pages/home", ImportSkip, ""},

		// Static assets → skip
		{"./styles.css", ImportSkip, ""},
		{"../icon.svg", ImportSkip, ""},
		{"some-pkg/style.scss", ImportSkip, ""},

		// React host deps → skip
		{"react", ImportSkip, ""},
		{"react-dom", ImportSkip, ""},
		{"react/jsx-runtime", ImportSkip, ""},

		// npm deps
		{"lucide-react", ImportNPM, "lucide-react"},
		{"@tanstack/react-table", ImportNPM, "@tanstack/react-table"},
		{"@tanstack/react-table/utils", ImportNPM, "@tanstack/react-table"},
		{"class-variance-authority", ImportNPM, "class-variance-authority"},
		{"zod", ImportNPM, "zod"},
		{"clsx", ImportNPM, "clsx"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			gotCls, gotKey := classifySource(tt.source)
			if gotCls != tt.wantCls {
				t.Errorf("classifySource(%q) class = %d, want %d", tt.source, gotCls, tt.wantCls)
			}
			if gotKey != tt.wantKey {
				t.Errorf("classifySource(%q) key = %q, want %q", tt.source, gotKey, tt.wantKey)
			}
		})
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"lucide-react", "lucide-react"},
		{"lucide-react/icons/Arrow", "lucide-react"},
		{"@tanstack/react-table", "@tanstack/react-table"},
		{"@tanstack/react-table/utils", "@tanstack/react-table"},
		{"zod", "zod"},
		{"@scope/pkg", "@scope/pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractPackageName(tt.input)
			if got != tt.want {
				t.Errorf("extractPackageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractPeerDepKey(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantOK  bool
	}{
		{"@/shared/wk/ui/spinner", "ui/spinner", true},
		{"@/shared/wk/ui/data-table/types", "ui/data-table", true},
		{"@/shared/wk/components/date-picker", "components/date-picker", true},
		{"@/shared/wk/components/modal/index", "components/modal", true},
		{"@/shared/wk/types", "", false},       // no component name after category
		{"@/shared/lib/utils", "", false},       // not @/shared/wk/
		{"lucide-react", "", false},             // npm
		{"@/shared/wk/ui", "", false},           // no component name
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, ok := extractPeerDepKey(tt.input)
			if ok != tt.wantOK {
				t.Errorf("extractPeerDepKey(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if key != tt.wantKey {
				t.Errorf("extractPeerDepKey(%q) key = %q, want %q", tt.input, key, tt.wantKey)
			}
		})
	}
}

func TestScanFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tsx")

	content := `import { useState } from "react"
import { Button } from "@/shared/wk/ui/button"
import { cn } from "@/shared/lib/utils"
import { cva } from "class-variance-authority"
import type { VariantProps } from "class-variance-authority"
import * as Icons from "lucide-react"
export { Spinner } from "@/shared/wk/ui/spinner"
import "tailwindcss/base"
import "./local.css"
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	sources, err := ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should find all import sources
	want := map[string]bool{
		"react":                       true,
		"@/shared/wk/ui/button":       true,
		"@/shared/lib/utils":          true,
		"class-variance-authority":     true,
		"lucide-react":                true,
		"@/shared/wk/ui/spinner":      true,
		"tailwindcss/base":            true,
		"./local.css":                 true,
	}

	got := make(map[string]bool)
	for _, s := range sources {
		got[s] = true
	}

	for k := range want {
		if !got[k] {
			t.Errorf("expected source %q not found", k)
		}
	}
}

func TestScanFile_CommentsIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tsx")

	content := `import { real } from "real-pkg"
// import { fake } from "fake-pkg"
/* import { blocked } from "blocked-pkg" */
/*
import { multi } from "multi-line-blocked"
*/
import { also_real } from "also-real-pkg"
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	sources, err := ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool)
	for _, s := range sources {
		got[s] = true
	}

	if !got["real-pkg"] {
		t.Error("expected real-pkg")
	}
	if !got["also-real-pkg"] {
		t.Error("expected also-real-pkg")
	}
	if got["fake-pkg"] {
		t.Error("fake-pkg should be excluded (line comment)")
	}
	if got["blocked-pkg"] {
		t.Error("blocked-pkg should be excluded (block comment)")
	}
	if got["multi-line-blocked"] {
		t.Error("multi-line-blocked should be excluded (multiline block comment)")
	}
}

func TestScanFile_CSSSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tsx")

	content := `import { Button } from "some-lib"
import "./styles.css"
import "../global.scss"
import "some-pkg/dist/style.css"
`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Use ScanDir to get classified results
	result, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Only "some-lib" should appear as npm dep
	if len(result.Deps) != 1 || result.Deps[0] != "some-lib" {
		t.Errorf("Deps = %v, want [some-lib]", result.Deps)
	}
	if len(result.PeerDeps) != 0 {
		t.Errorf("PeerDeps = %v, want []", result.PeerDeps)
	}
}

func TestScanDir(t *testing.T) {
	dir := t.TempDir()

	// File 1: has npm deps and peer deps
	f1 := filepath.Join(dir, "button.tsx")
	if err := os.WriteFile(f1, []byte(`
import { cva } from "class-variance-authority"
import { Spinner } from "@/shared/wk/ui/spinner"
import { cn } from "@/shared/lib/utils"
`), 0644); err != nil {
		t.Fatal(err)
	}

	// File 2: has overlapping deps (test dedup)
	f2 := filepath.Join(dir, "variant.tsx")
	if err := os.WriteFile(f2, []byte(`
import { cva } from "class-variance-authority"
import { Icon } from "lucide-react"
import { DatePicker } from "@/shared/wk/components/date-picker"
`), 0644); err != nil {
		t.Fatal(err)
	}

	// File 3: .d.ts should be excluded
	f3 := filepath.Join(dir, "types.d.ts")
	if err := os.WriteFile(f3, []byte(`
import { ShouldBeIgnored } from "ignored-pkg"
`), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Check deps (sorted, deduped)
	wantDeps := []string{"class-variance-authority", "lucide-react"}
	if len(result.Deps) != len(wantDeps) {
		t.Fatalf("Deps = %v, want %v", result.Deps, wantDeps)
	}
	for i, d := range wantDeps {
		if result.Deps[i] != d {
			t.Errorf("Deps[%d] = %q, want %q", i, result.Deps[i], d)
		}
	}

	// Check peer deps (sorted, deduped)
	wantPeers := []string{"components/date-picker", "ui/spinner"}
	if len(result.PeerDeps) != len(wantPeers) {
		t.Fatalf("PeerDeps = %v, want %v", result.PeerDeps, wantPeers)
	}
	for i, p := range wantPeers {
		if result.PeerDeps[i] != p {
			t.Errorf("PeerDeps[%d] = %q, want %q", i, result.PeerDeps[i], p)
		}
	}

	// .d.ts should not be scanned
	if len(result.ScannedFiles) != 2 {
		t.Errorf("ScannedFiles = %v, want 2 files", result.ScannedFiles)
	}

	// "ignored-pkg" should not appear
	for _, d := range result.Deps {
		if d == "ignored-pkg" {
			t.Error("ignored-pkg should not appear (.d.ts file)")
		}
	}
}
