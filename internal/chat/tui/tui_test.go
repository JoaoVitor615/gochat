package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

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

func TestPeerLastSeen(t *testing.T) {
	// Zero time
	if got := peerLastSeen(time.Time{}); got != "presença não verificada" {
		t.Errorf("got %q, want 'presença não verificada'", got)
	}

	// Given timestamp
	sampleTime := time.Date(2026, 10, 18, 14, 30, 0, 0, time.Local)
	expected := "visto 18/10 14:30"
	if got := peerLastSeen(sampleTime); got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
	if strings.Contains(peerLastSeen(sampleTime), "Discovery") {
		t.Errorf("expected no mention of Discovery, got %q", peerLastSeen(sampleTime))
	}
}

func TestIsPeerOnline(t *testing.T) {
	if isPeerOnline(time.Time{}) {
		t.Error("zero time should be offline")
	}
	if !isPeerOnline(time.Now().Add(-10 * time.Second)) {
		t.Error("10 seconds ago should be online")
	}
	if !isPeerOnline(time.Now().Add(-25 * time.Second)) {
		t.Error("25 seconds ago should be online")
	}
	if isPeerOnline(time.Now().Add(-35 * time.Second)) {
		t.Error("35 seconds ago should be offline")
	}
	if isPeerOnline(time.Now().Add(-1 * time.Hour)) {
		t.Error("1 hour ago should be offline")
	}
}

func TestModel_PresenceOnlineAndOffline(t *testing.T) {
	m := model{
		width:   80,
		height:  24,
		splash:  false,
		active:  0,
		peers: []peer{
			{
				id:          "peer-1",
				displayName: "Alice",
				lastSeenAt:  time.Now().Add(-2 * time.Hour), // offline initially
			},
		},
	}

	// Offline check
	sideOffline := strings.Join(m.sidebar(25, 20), "\n")
	if !strings.Contains(sideOffline, "○") {
		t.Errorf("expected offline circle ○, got:\n%s", sideOffline)
	}
	if !strings.Contains(sideOffline, "visto") {
		t.Errorf("expected visto in offline sidebar, got:\n%s", sideOffline)
	}

	// Now send presence event saying Alice was seen right now
	newM, _ := m.Update(presenceEventMsg{PeerID: "peer-1", LastSeenAt: time.Now()})
	mod := newM.(model)

	// Online check
	sideOnline := strings.Join(mod.sidebar(25, 20), "\n")
	if !strings.Contains(sideOnline, "●") {
		t.Errorf("expected online circle ●, got:\n%s", sideOnline)
	}
	if !strings.Contains(sideOnline, "online") {
		t.Errorf("expected online in sidebar, got:\n%s", sideOnline)
	}
}


