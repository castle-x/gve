package semver

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.0.0", Version{1, 0, 0}, false},
		{"0.1.2", Version{0, 1, 2}, false},
		{"v2.3.4", Version{2, 3, 4}, false},
		{"12.34.56", Version{12, 34, 56}, false},
		{"invalid", Version{}, true},
		{"1.2", Version{}, true},
		{"a.b.c", Version{}, true},
	}

	for _, tt := range tests {
		got, err := Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b Version
		want int
	}{
		{Version{1, 0, 0}, Version{1, 0, 0}, 0},
		{Version{2, 0, 0}, Version{1, 0, 0}, 1},
		{Version{1, 2, 0}, Version{1, 1, 0}, 1},
		{Version{1, 0, 1}, Version{1, 0, 0}, 1},
		{Version{0, 9, 0}, Version{1, 0, 0}, -1},
	}

	for _, tt := range tests {
		got := Compare(tt.a, tt.b)
		if (tt.want > 0 && got <= 0) || (tt.want < 0 && got >= 0) || (tt.want == 0 && got != 0) {
			t.Errorf("Compare(%v, %v) = %d, want sign of %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSortVersions(t *testing.T) {
	input := []string{"2.0.0", "1.0.0", "1.2.0", "1.1.0"}
	got := SortVersions(input)
	want := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0"}

	if len(got) != len(want) {
		t.Fatalf("SortVersions length = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("SortVersions[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestBumpPatch(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"1.2.3", "1.2.4", false},
		{"0.0.0", "0.0.1", false},
		{"v2.5.9", "2.5.10", false},
		{"0.1.0", "0.1.1", false},
		{"invalid", "", true},
		{"1.2", "", true},
	}

	for _, tt := range tests {
		got, err := BumpPatch(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("BumpPatch(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("BumpPatch(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	available := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0"}

	tests := []struct {
		constraint string
		want       string
		wantErr    bool
	}{
		{"latest", "2.0.0", false},
		{"", "2.0.0", false},
		{"1.1.0", "1.1.0", false},
		{"^1.0.0", "1.2.0", false},
		{"~1.0.0", "1.0.0", false},
		{"~1.1.0", "1.1.0", false},
		{"3.0.0", "", true},
		{"^3.0.0", "", true},
	}

	for _, tt := range tests {
		got, err := ResolveVersion(tt.constraint, available)
		if (err != nil) != tt.wantErr {
			t.Errorf("ResolveVersion(%q) error = %v, wantErr %v", tt.constraint, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ResolveVersion(%q) = %q, want %q", tt.constraint, got, tt.want)
		}
	}
}
