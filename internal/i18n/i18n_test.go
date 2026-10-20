package i18n

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestI18n_FallbackToEnglishFirst(t *testing.T) {
	// Empty candidate should return English
	if lang := parseCandidate(""); lang != "" {
		t.Fatalf("expected empty for empty input, got %s", lang)
	}
	if lang := parseCandidate("ja-JP"); lang != "" {
		t.Fatalf("expected empty for unsupported language, got %s", lang)
	}

	// Unknown locale should fallback to English in DetectLanguage when env is cleared
	origGochatLang := os.Getenv("GOCHAT_LANG")
	origLang := os.Getenv("LANG")
	origLCAll := os.Getenv("LC_ALL")
	defer func() {
		_ = os.Setenv("GOCHAT_LANG", origGochatLang)
		_ = os.Setenv("LANG", origLang)
		_ = os.Setenv("LC_ALL", origLCAll)
	}()

	_ = os.Setenv("GOCHAT_LANG", "fr-FR")
	if detected := DetectLanguage(); detected != LangEnUS {
		t.Fatalf("expected fallback to English for unsupported locale, got %s", detected)
	}
}

func TestI18n_DetectLanguage(t *testing.T) {
	origGochatLang := os.Getenv("GOCHAT_LANG")
	defer func() {
		_ = os.Setenv("GOCHAT_LANG", origGochatLang)
	}()

	_ = os.Setenv("GOCHAT_LANG", "pt_BR.UTF-8")
	if detected := DetectLanguage(); detected != LangPtBR {
		t.Fatalf("expected pt-BR, got %s", detected)
	}

	_ = os.Setenv("GOCHAT_LANG", "pt")
	if detected := DetectLanguage(); detected != LangPtBR {
		t.Fatalf("expected pt-BR for 'pt', got %s", detected)
	}

	_ = os.Setenv("GOCHAT_LANG", "es_ES.UTF-8")
	if detected := DetectLanguage(); detected != LangEsES {
		t.Fatalf("expected es-ES, got %s", detected)
	}

	_ = os.Setenv("GOCHAT_LANG", "es")
	if detected := DetectLanguage(); detected != LangEsES {
		t.Fatalf("expected es-ES for 'es', got %s", detected)
	}

	_ = os.Setenv("GOCHAT_LANG", "en_GB")
	if detected := DetectLanguage(); detected != LangEnUS {
		t.Fatalf("expected en-US for 'en_GB', got %s", detected)
	}
}

func TestI18n_SetLanguageAndBundles(t *testing.T) {
	// Set to Portuguese
	SetLanguage(LangPtBR)
	if CurrentLanguage() != LangPtBR {
		t.Fatalf("expected CurrentLanguage to be pt-BR, got %s", CurrentLanguage())
	}
	if T.SplashConnecting() != "Conectando à rede P2P..." {
		t.Fatalf("unexpected PT string: %s", T.SplashConnecting())
	}
	if T.StatusQueued() != "na fila" {
		t.Fatalf("unexpected PT status: %s", T.StatusQueued())
	}

	// Set to Spanish
	SetLanguage(LangEsES)
	if CurrentLanguage() != LangEsES {
		t.Fatalf("expected CurrentLanguage to be es-ES, got %s", CurrentLanguage())
	}
	if T.SplashConnecting() != "Conectando a la red P2P..." {
		t.Fatalf("unexpected ES string: %s", T.SplashConnecting())
	}
	if T.StatusQueued() != "en cola" {
		t.Fatalf("unexpected ES status: %s", T.StatusQueued())
	}

	// Set to English
	SetLanguage(LangEnUS)
	if CurrentLanguage() != LangEnUS {
		t.Fatalf("expected CurrentLanguage to be en-US, got %s", CurrentLanguage())
	}
	if T.SplashConnecting() != "Connecting to P2P network..." {
		t.Fatalf("unexpected EN string: %s", T.SplashConnecting())
	}
	if T.StatusQueued() != "queued" {
		t.Fatalf("unexpected EN status: %s", T.StatusQueued())
	}

	// Format last seen
	now := time.Date(2026, 10, 19, 14, 30, 0, 0, time.UTC)
	if seen := T.LastSeen(now); seen != "seen 10/19 14:30" {
		t.Fatalf("unexpected EN last seen format: %s", seen)
	}
	if zero := T.LastSeen(time.Time{}); zero != "presence unverified" {
		t.Fatalf("unexpected EN zero last seen: %s", zero)
	}

	SetLanguage(LangPtBR)
	if seen := T.LastSeen(now); seen != "visto 19/10 14:30" {
		t.Fatalf("unexpected PT last seen format: %s", seen)
	}
	if zero := T.LastSeen(time.Time{}); zero != "presença não verificada" {
		t.Fatalf("unexpected PT zero last seen: %s", zero)
	}

	SetLanguage(LangEsES)
	if seen := T.LastSeen(now); seen != "visto 19/10 14:30" {
		t.Fatalf("unexpected ES last seen format: %s", seen)
	}
	if zero := T.LastSeen(time.Time{}); zero != "presencia no verificada" {
		t.Fatalf("unexpected ES zero last seen: %s", zero)
	}

	// Check error parameter handling
	testErr := errors.New("timeout dialing")
	if notice := T.NoticeHistoryError(testErr); notice != "Error al cargar el historial: timeout dialing" {
		t.Fatalf("unexpected ES notice: %s", notice)
	}
	if notice := PtBR.NoticeHistoryError(testErr); notice != "Falha ao carregar histórico: timeout dialing" {
		t.Fatalf("unexpected PT notice: %s", notice)
	}
	if notice := EnUS.NoticeHistoryError(testErr); notice != "Failed to load history: timeout dialing" {
		t.Fatalf("unexpected EN notice: %s", notice)
	}
}
