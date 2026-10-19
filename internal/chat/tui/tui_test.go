package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_SplashView(t *testing.T) {
	m := model{
		width:            80,
		height:           25,
		splash:           true,
		splashSpinnerIdx: 0,
		splashStatus:     "Conectando à rede P2P...",
		localID:          "12D3KooWTestPeer123456789",
	}

	view := m.View()
	if !strings.Contains(view, "P 2 P") {
		t.Errorf("expected splash view to contain 'P 2 P', got:\n%s", view)
	}
	if !strings.Contains(view, "Conectando à rede P2P...") {
		t.Errorf("expected splash view to contain status, got:\n%s", view)
	}
	if !strings.Contains(view, "12D3KooWTestPeer") {
		t.Errorf("expected splash view to contain peer ID, got:\n%s", view)
	}
}

func TestModel_SplashTransition(t *testing.T) {
	m := model{
		width:        80,
		height:       25,
		splash:       true,
		splashStatus: "Conectando à rede P2P...",
	}

	// 1. Min duration fires before init is done -> should remain on splash
	newM, _ := m.Update(splashMinDurationMsg{})
	mod := newM.(model)
	if !mod.splash {
		t.Error("expected splash to remain true when init is not yet done")
	}

	// 2. Init finishes -> now both are done, splash should transition to false
	newM, cmd := mod.Update(splashInitResult{sender: nil, err: nil})
	mod = newM.(model)
	if mod.splash {
		t.Error("expected splash to transition to false when both min duration and init are done")
	}
	if cmd == nil {
		t.Error("expected refreshTickCmd after splash finishes")
	}
}

func TestModel_SplashError(t *testing.T) {
	m := model{
		width:        80,
		height:       25,
		splash:       true,
		splashStatus: "Conectando à rede P2P...",
	}

	expectedErr := errors.New("failed to start peer host")
	newM, _ := m.Update(splashInitResult{err: expectedErr})
	mod := newM.(model)

	if mod.splashErr != expectedErr {
		t.Errorf("expected splashErr to be %v, got %v", expectedErr, mod.splashErr)
	}

	view := mod.View()
	if !strings.Contains(view, "failed to start peer host") {
		t.Errorf("expected view to display error, got:\n%s", view)
	}
	if !strings.Contains(view, "pressione 'q' para sair") {
		t.Errorf("expected view to display quit hint, got:\n%s", view)
	}

	// Pressing 'q' should quit when in error state
	_, quitCmd := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Error("expected quitCmd when pressing 'q' on splash error")
	}
}
