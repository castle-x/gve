package i18n

import (
	"os"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMustInit_DefaultZh(t *testing.T) {
	os.Unsetenv("GVE_LANG")
	os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")

	MustInit()

	if Lang() != "zh" {
		t.Fatalf("expected default lang zh, got %s", Lang())
	}
	// Verify a known zh key
	got := T("root_short")
	if got == "root_short" {
		t.Fatal("expected zh translation for root_short, got key itself")
	}
}

func TestSetLang_En(t *testing.T) {
	MustInit()
	SetLang("en")

	if Lang() != "en" {
		t.Fatalf("expected lang en, got %s", Lang())
	}
	got := T("root_short")
	if got == "root_short" {
		t.Fatal("expected en translation for root_short, got key itself")
	}
	if got == T("root_short") {
		// Just ensure it's not the key
	}

	// Reset
	SetLang("zh")
}

func TestTf_Format(t *testing.T) {
	SetLang("en")
	defer SetLang("zh")

	got := Tf("init_creating", "myapp")
	expected := "Creating project myapp..."
	if got != expected {
		t.Fatalf("Tf: expected %q, got %q", expected, got)
	}
}

func TestDetectLang_GVE_LANG(t *testing.T) {
	os.Setenv("GVE_LANG", "en")
	defer os.Unsetenv("GVE_LANG")

	got := detectLang()
	if got != "en" {
		t.Fatalf("expected en from GVE_LANG, got %s", got)
	}
}

func TestDetectLang_LANG(t *testing.T) {
	os.Unsetenv("GVE_LANG")
	os.Setenv("LANG", "en_US.UTF-8")
	defer os.Unsetenv("LANG")

	got := detectLang()
	if got != "en" {
		t.Fatalf("expected en from LANG=en_US.UTF-8, got %s", got)
	}
}

func TestDetectLang_ZhCN(t *testing.T) {
	os.Unsetenv("GVE_LANG")
	os.Setenv("LANG", "zh_CN.UTF-8")
	defer os.Unsetenv("LANG")

	got := detectLang()
	if got != "zh" {
		t.Fatalf("expected zh from LANG=zh_CN.UTF-8, got %s", got)
	}
}

func TestT_MissingKey_ReturnsKey(t *testing.T) {
	MustInit()
	key := "nonexistent_key_12345"
	if T(key) != key {
		t.Fatalf("expected missing key to return itself, got %q", T(key))
	}
}

func TestT_NilMessages_ReturnsKey(t *testing.T) {
	old := messages
	messages = nil
	defer func() { messages = old }()

	key := "any_key"
	if T(key) != key {
		t.Fatalf("expected key when messages is nil, got %q", T(key))
	}
}

func TestAllKeysMatch(t *testing.T) {
	zhData, err := localeFS.ReadFile("locales/zh.yaml")
	if err != nil {
		t.Fatalf("read zh.yaml: %v", err)
	}
	enData, err := localeFS.ReadFile("locales/en.yaml")
	if err != nil {
		t.Fatalf("read en.yaml: %v", err)
	}

	zhMap := make(map[string]string)
	enMap := make(map[string]string)
	if err := yaml.Unmarshal(zhData, &zhMap); err != nil {
		t.Fatalf("parse zh.yaml: %v", err)
	}
	if err := yaml.Unmarshal(enData, &enMap); err != nil {
		t.Fatalf("parse en.yaml: %v", err)
	}

	zhKeys := sortedKeys(zhMap)
	enKeys := sortedKeys(enMap)

	// Check zh has all en keys
	for _, k := range enKeys {
		if _, ok := zhMap[k]; !ok {
			t.Errorf("key %q in en.yaml but missing in zh.yaml", k)
		}
	}

	// Check en has all zh keys
	for _, k := range zhKeys {
		if _, ok := enMap[k]; !ok {
			t.Errorf("key %q in zh.yaml but missing in en.yaml", k)
		}
	}
}

func TestNormalizeLang(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"en", "en"},
		{"en_US", "en"},
		{"en_US.UTF-8", "en"},
		{"zh", "zh"},
		{"zh_CN", "zh"},
		{"zh_CN.UTF-8", "zh"},
		{"fr_FR.UTF-8", "zh"}, // unknown defaults to zh
		{"", "zh"},
	}
	for _, tt := range tests {
		got := normalizeLang(tt.input)
		if got != tt.want {
			t.Errorf("normalizeLang(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
