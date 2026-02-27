package version

import (
	"strings"
	"testing"
)

func TestFull(t *testing.T) {
	out := Full()
	if !strings.HasPrefix(out, "gve v") {
		t.Errorf("Full() should start with 'gve v', got %q", out)
	}
	if !strings.Contains(out, Version) {
		t.Errorf("Full() should contain Version %q, got %q", Version, out)
	}
	if !strings.Contains(out, GitCommit) {
		t.Errorf("Full() should contain GitCommit %q, got %q", GitCommit, out)
	}
	if !strings.Contains(out, BuildDate) {
		t.Errorf("Full() should contain BuildDate %q, got %q", BuildDate, out)
	}
}
