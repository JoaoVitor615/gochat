package i18n

import (
	"os"
	"strings"
	"sync"
)

// Language represents a supported language code.
type Language string

const (
	// LangEnUS represents English (United States) - Default fallback.
	LangEnUS Language = "en-US"
	// LangPtBR represents Portuguese (Brazil).
	LangPtBR Language = "pt-BR"
	// LangEsES represents Spanish.
	LangEsES Language = "es-ES"
)

// LanguageOption represents a selectable language in the settings menu.
type LanguageOption struct {
	Tag  Language
	Name string
}

// AvailableLanguages returns all supported languages for user selection.
func AvailableLanguages() []LanguageOption {
	return []LanguageOption{
		{Tag: LangPtBR, Name: PtBR.LanguageName()},
		{Tag: LangEnUS, Name: EnUS.LanguageName()},
		{Tag: LangEsES, Name: EsES.LanguageName()},
	}
}

var (
	mu          sync.RWMutex
	currentLang = LangEnUS
	// T is the globally active Messages bundle.
	T Messages = EnUS
)

func init() {
	SetLanguage(DetectLanguage())
}

// CurrentLanguage returns the currently active language.
func CurrentLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// SetLanguage activates the specified language bundle.
// If an unsupported language is provided, it falls back to English (LangEnUS).
func SetLanguage(lang Language) {
	mu.Lock()
	defer mu.Unlock()

	norm := normalizeLangString(string(lang))
	switch {
	case strings.HasPrefix(norm, "pt"):
		currentLang = LangPtBR
		T = PtBR
	case strings.HasPrefix(norm, "es"):
		currentLang = LangEsES
		T = EsES
	default:
		currentLang = LangEnUS
		T = EnUS
	}
}

// DetectLanguage inspects environment variables and OS locale settings
// to determine the preferred language.
//
// Evaluation order:
// 1. GOCHAT_LANG environment variable
// 2. LC_ALL environment variable
// 3. LC_MESSAGES environment variable
// 4. LANG environment variable
// 5. Operating System locale (e.g., Windows UI locale)
// 6. Fallback to English (LangEnUS)
func DetectLanguage() Language {
	if custom := os.Getenv("GOCHAT_LANG"); custom != "" {
		if lang := parseCandidate(custom); lang != "" {
			return lang
		}
		return LangEnUS
	}

	candidates := []string{
		os.Getenv("LC_ALL"),
		os.Getenv("LC_MESSAGES"),
		os.Getenv("LANG"),
		detectOSLocale(),
	}

	for _, cand := range candidates {
		lang := parseCandidate(cand)
		if lang != "" {
			return lang
		}
	}

	// Default fallback is English first
	return LangEnUS
}

func parseCandidate(raw string) Language {
	trimmed := normalizeLangString(raw)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "pt") {
		return LangPtBR
	}
	if strings.HasPrefix(trimmed, "es") {
		return LangEsES
	}
	if strings.HasPrefix(trimmed, "en") {
		return LangEnUS
	}

	return ""
}

func normalizeLangString(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	// Strip encoding like .UTF-8 or @euro
	if idx := strings.IndexAny(s, ".@"); idx != -1 {
		s = s[:idx]
	}
	s = strings.ReplaceAll(s, "_", "-")
	return s
}
