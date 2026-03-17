package semver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semver: %q (expected x.y.z)", s)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %q", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %q", parts[1])
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %q", parts[2])
	}

	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func Compare(a, b Version) int {
	if a.Major != b.Major {
		return a.Major - b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor - b.Minor
	}
	return a.Patch - b.Patch
}

func SortVersions(versions []string) []string {
	type tagged struct {
		raw string
		ver Version
	}
	var items []tagged
	for _, v := range versions {
		parsed, err := Parse(v)
		if err != nil {
			continue
		}
		items = append(items, tagged{raw: v, ver: parsed})
	}

	sort.Slice(items, func(i, j int) bool {
		return Compare(items[i].ver, items[j].ver) < 0
	})

	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.raw
	}
	return result
}

// BumpPatch increments the patch component of a semver string.
// "1.2.3" → "1.2.4", "v0.0.0" → "0.0.1".
func BumpPatch(v string) (string, error) {
	parsed, err := Parse(v)
	if err != nil {
		return "", err
	}
	parsed.Patch++
	return parsed.String(), nil
}

// ResolveVersion resolves a version constraint against available versions.
// Supports: "latest", exact match ("1.2.0"), caret ("^1.0.0"), tilde ("~1.2.0").
func ResolveVersion(constraint string, available []string) (string, error) {
	if len(available) == 0 {
		return "", fmt.Errorf("no versions available")
	}

	sorted := SortVersions(available)

	if constraint == "latest" || constraint == "" {
		return sorted[len(sorted)-1], nil
	}

	if strings.HasPrefix(constraint, "^") {
		target, err := Parse(strings.TrimPrefix(constraint, "^"))
		if err != nil {
			return "", fmt.Errorf("invalid caret constraint: %w", err)
		}
		var best string
		for _, v := range sorted {
			parsed, _ := Parse(v)
			if parsed.Major == target.Major && Compare(parsed, target) >= 0 {
				best = v
			}
		}
		if best == "" {
			return "", fmt.Errorf("no version matching %s", constraint)
		}
		return best, nil
	}

	if strings.HasPrefix(constraint, "~") {
		target, err := Parse(strings.TrimPrefix(constraint, "~"))
		if err != nil {
			return "", fmt.Errorf("invalid tilde constraint: %w", err)
		}
		var best string
		for _, v := range sorted {
			parsed, _ := Parse(v)
			if parsed.Major == target.Major && parsed.Minor == target.Minor && Compare(parsed, target) >= 0 {
				best = v
			}
		}
		if best == "" {
			return "", fmt.Errorf("no version matching %s", constraint)
		}
		return best, nil
	}

	// Exact match
	for _, v := range available {
		if v == constraint {
			return v, nil
		}
	}
	return "", fmt.Errorf("version %s not found", constraint)
}
