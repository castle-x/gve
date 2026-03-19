// Package i18n provides internationalization support for the GVE CLI.
// Translations are embedded from YAML locale files at compile time.
package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

var (
	messages map[string]string
	lang     string
)

// MustInit detects the user language and loads the corresponding locale file.
// Detection priority: GVE_LANG > LANG > LC_ALL > default "zh".
func MustInit() {
	l := detectLang()
	lang = l
	loadLang(l)
}

// T returns the translated string for key. If key is not found, key itself is returned.
func T(key string) string {
	if messages == nil {
		return key
	}
	if v, ok := messages[key]; ok {
		return v
	}
	return key
}

// Tf returns a formatted translated string using fmt.Sprintf.
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// Lang returns the current language code ("zh" or "en").
func Lang() string {
	return lang
}

// SetLang switches the active language (mainly for testing).
func SetLang(l string) {
	l = normalizeLang(l)
	lang = l
	loadLang(l)
}

// detectLang reads environment variables to determine the user's language.
func detectLang() string {
	for _, env := range []string{"GVE_LANG", "LANG", "LC_ALL"} {
		if v := os.Getenv(env); v != "" {
			return normalizeLang(v)
		}
	}
	return "zh"
}

// normalizeLang converts locale strings like "en_US.UTF-8" to "en".
func normalizeLang(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	// Strip encoding suffix (e.g. ".UTF-8")
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		s = s[:idx]
	}

	// Extract language part (e.g. "en_US" -> "en")
	if idx := strings.IndexByte(s, '_'); idx >= 0 {
		s = s[:idx]
	}

	switch s {
	case "en":
		return "en"
	case "zh":
		return "zh"
	default:
		return "zh"
	}
}

// loadLang loads the locale YAML for the given language code.
func loadLang(l string) {
	data, err := localeFS.ReadFile("locales/" + l + ".yaml")
	if err != nil {
		// Fallback to zh
		data, err = localeFS.ReadFile("locales/zh.yaml")
		if err != nil {
			messages = make(map[string]string)
			return
		}
	}

	m := make(map[string]string)
	if err := yaml.Unmarshal(data, &m); err != nil {
		messages = make(map[string]string)
		return
	}
	messages = m
}
