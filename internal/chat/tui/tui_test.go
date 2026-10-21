package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	"github.com/JoaoVitor615/gochat/internal/i18n"
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
	if !strings.Contains(view, i18n.T.SplashQuitHint()) {
		t.Errorf("expected view to display quit hint %q, got:\n%s", i18n.T.SplashQuitHint(), view)
	}

	// Pressing 'q' should quit when in error state
	_, quitCmd := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Error("expected quitCmd when pressing 'q' on splash error")
	}
}

func TestPeerLastSeen(t *testing.T) {
	orig := i18n.CurrentLanguage()
	defer i18n.SetLanguage(orig)

	sampleTime := time.Date(2026, 10, 18, 14, 30, 0, 0, time.Local)

	// Test pt-BR
	i18n.SetLanguage(i18n.LangPtBR)
	if got := peerLastSeen(time.Time{}); got != "presença não verificada" {
		t.Errorf("pt-BR got %q, want 'presença não verificada'", got)
	}
	expectedPt := "visto 18/10 14:30"
	if got := peerLastSeen(sampleTime); got != expectedPt {
		t.Errorf("pt-BR got %q, want %q", got, expectedPt)
	}

	// Test es-ES
	i18n.SetLanguage(i18n.LangEsES)
	if got := peerLastSeen(time.Time{}); got != "presencia no verificada" {
		t.Errorf("es-ES got %q, want 'presencia no verificada'", got)
	}
	expectedEs := "visto 18/10 14:30"
	if got := peerLastSeen(sampleTime); got != expectedEs {
		t.Errorf("es-ES got %q, want %q", got, expectedEs)
	}

	// Test en-US
	i18n.SetLanguage(i18n.LangEnUS)
	if got := peerLastSeen(time.Time{}); got != "presence unverified" {
		t.Errorf("en-US got %q, want 'presence unverified'", got)
	}
	expectedEn := "seen 10/18 14:30"
	if got := peerLastSeen(sampleTime); got != expectedEn {
		t.Errorf("en-US got %q, want %q", got, expectedEn)
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
	orig := i18n.CurrentLanguage()
	defer i18n.SetLanguage(orig)
	i18n.SetLanguage(i18n.LangPtBR)

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

func TestModel_PeerSortingByLastMessage(t *testing.T) {
	now := time.Now()
	aliceTime := now.Add(-10 * time.Minute)
	bobTime := now.Add(-5 * time.Minute)

	m := model{
		width:   80,
		height:  24,
		splash:  false,
		active:  0,
		selected: 0,
		peers: []peer{
			{
				id:            "alice-id",
				displayName:   "Alice",
				lastMessageAt: aliceTime,
			},
			{
				id:            "bob-id",
				displayName:   "Bob",
				lastMessageAt: bobTime,
			},
			{
				id:          "charlie-id",
				displayName: "Charlie",
				addedAt:     now.Add(-1 * time.Hour),
				// no messages
			},
		},
	}

	m.sortPeers()

	// Bob should be first (most recent message at -5m)
	// Alice should be second (at -10m)
	// Charlie should be third (no messages)
	if m.peers[0].id != "bob-id" {
		t.Fatalf("expected Bob at index 0, got %s", m.peers[0].displayName)
	}
	if m.peers[1].id != "alice-id" {
		t.Fatalf("expected Alice at index 1, got %s", m.peers[1].displayName)
	}
	if m.peers[2].id != "charlie-id" {
		t.Fatalf("expected Charlie at index 2, got %s", m.peers[2].displayName)
	}

	// Active peer was index 0 (Alice before sort). After sort, active should track Alice to index 1!
	if m.active != 1 || m.peers[m.active].id != "alice-id" {
		t.Fatalf("expected active to track Alice at index 1, got %d", m.active)
	}

	// Now Charlie sends a brand new message (now)
	charlieMsg := storage.StoredMessage{
		ConversationPeerID: "charlie-id",
		Envelope: message.Message{
			ID:              "msg-charlie-1",
			SenderPeerID:    "charlie-id",
			RecipientPeerID: "my-id",
			CreatedAt:       now,
			Content:         "Hello from Charlie!",
		},
		Status: message.DeliveryDelivered,
	}

	newM, _ := m.Update(messageEventMsg(charlieMsg))
	mod := newM.(model)

	// Charlie should now be at the very top (index 0) like WhatsApp!
	if mod.peers[0].id != "charlie-id" {
		t.Fatalf("expected Charlie to move to index 0 after message, got %s", mod.peers[0].displayName)
	}
	if mod.peers[1].id != "bob-id" {
		t.Fatalf("expected Bob at index 1, got %s", mod.peers[1].displayName)
	}
	if mod.peers[2].id != "alice-id" {
		t.Fatalf("expected Alice at index 2, got %s", mod.peers[2].displayName)
	}

	// Active was Alice (index 1 before Charlie's message). Now Alice is at index 2!
	if mod.active != 2 || mod.peers[mod.active].id != "alice-id" {
		t.Fatalf("expected active to track Alice at index 2, got %d", mod.active)
	}
}


func TestModel_SettingsPanel(t *testing.T) {
	orig := i18n.CurrentLanguage()
	defer i18n.SetLanguage(orig)
	i18n.SetLanguage(i18n.LangPtBR)

	m := model{
		width:  80,
		height: 24,
		splash: false,
		active: -1,
		panel:  noPanel,
	}

	// 1. Pressing 'esc' on home screen should open settingsPanel
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mod := newM.(model)
	if mod.panel != settingsPanel {
		t.Fatalf("expected settingsPanel on esc, got %v", mod.panel)
	}

	// View should render settings dialog with options
	view := mod.View()
	if !strings.Contains(view, "CONFIGURAÇÕES") {
		t.Errorf("expected view to contain 'CONFIGURAÇÕES', got:\n%s", view)
	}
	if !strings.Contains(view, "Español") || !strings.Contains(view, "English") || !strings.Contains(view, "Português") {
		t.Errorf("expected view to list all languages, got:\n%s", view)
	}

	// 2. Select Spanish
	langs := i18n.AvailableLanguages()
	esIdx := -1
	for i, l := range langs {
		if l.Tag == i18n.LangEsES {
			esIdx = i
			break
		}
	}
	if esIdx < 0 {
		t.Fatal("es-ES not found in AvailableLanguages")
	}

	mod.settingsSelected = esIdx
	newM, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mod = newM.(model)

	if i18n.CurrentLanguage() != i18n.LangEsES {
		t.Fatalf("expected current language to change to es-ES, got %s", i18n.CurrentLanguage())
	}

	// Now the view should show Spanish title
	viewEs := mod.View()
	if !strings.Contains(viewEs, "CONFIGURACIÓN") {
		t.Errorf("expected view to show Spanish title 'CONFIGURACIÓN', got:\n%s", viewEs)
	}

	// 3. Pressing 'esc' in settings closes the panel
	newM, _ = mod.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mod = newM.(model)
	if mod.panel != noPanel {
		t.Fatalf("expected noPanel after esc in settings, got %v", mod.panel)
	}
}


type mockWipeStore struct {
	wiped bool
}

func (m *mockWipeStore) ListPeers(ctx context.Context) ([]storage.Peer, error) { return nil, nil }
func (m *mockWipeStore) SavePeer(ctx context.Context, p storage.Peer) error     { return nil }
func (m *mockWipeStore) SetPeerDisplayName(ctx context.Context, id, name string) error {
	return nil
}
func (m *mockWipeStore) ListMessages(ctx context.Context, peerID string, before *message.Message, limit int) ([]storage.StoredMessage, error) {
	return nil, nil
}
func (m *mockWipeStore) CryptographicErasure() error {
	m.wiped = true
	return nil
}

func TestModel_SettingsCryptographicErasure(t *testing.T) {
	orig := i18n.CurrentLanguage()
	defer i18n.SetLanguage(orig)
	i18n.SetLanguage(i18n.LangPtBR)

	mockStore := &mockWipeStore{}
	m := model{
		width:     80,
		height:    24,
		splash:    false,
		active:    0,
		peerStore: mockStore,
		peers: []peer{
			{id: "peer-1", displayName: "Alice"},
		},
		messages: []storage.StoredMessage{
			{ConversationPeerID: "peer-1"},
		},
		panel: settingsPanel,
	}

	langs := i18n.AvailableLanguages()
	wipeIdx := len(langs)

	// 1. Navigate to Wipe option (index len(langs))
	m.settingsSelected = wipeIdx
	view := m.View()
	if !strings.Contains(view, "Apagar todas as mensagens") {
		t.Errorf("expected view to contain wipe option, got:\n%s", view)
	}

	// 2. Press Enter on Wipe option -> should enter confirmation mode
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mod := newM.(model)
	if !mod.settingsConfirmWipe {
		t.Fatal("expected settingsConfirmWipe to be true")
	}

	confirmView := mod.View()
	if !strings.Contains(confirmView, "IRREVERSÍVEL") {
		t.Errorf("expected confirmation view to warn about IRREVERSÍVEL, got:\n%s", confirmView)
	}
	if !strings.Contains(confirmView, "AES-256") {
		t.Errorf("expected confirmation view to mention AES-256, got:\n%s", confirmView)
	}

	// 3. Test Cancel with 'n'
	cancelM, _ := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	cancelMod := cancelM.(model)
	if cancelMod.settingsConfirmWipe {
		t.Error("expected settingsConfirmWipe to be false after 'n'")
	}
	if mockStore.wiped {
		t.Error("expected store NOT to be wiped after cancel")
	}

	// 4. Enter confirmation mode again and confirm with 'y'
	mod.settingsConfirmWipe = true
	wipeM, _ := mod.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	wipedMod := wipeM.(model)

	if !mockStore.wiped {
		t.Error("expected mockStore to be wiped after 'y'")
	}
	if len(wipedMod.peers) != 0 {
		t.Errorf("expected peers to be empty after wipe, got %d", len(wipedMod.peers))
	}
	if len(wipedMod.messages) != 0 {
		t.Errorf("expected messages to be empty after wipe, got %d", len(wipedMod.messages))
	}
	if wipedMod.active != -1 {
		t.Errorf("expected active to be -1, got %d", wipedMod.active)
	}
	if wipedMod.panel != noPanel {
		t.Errorf("expected settings panel to close after wipe, got %v", wipedMod.panel)
	}
	if !strings.Contains(wipedMod.chatNotice, "sucesso") {
		t.Errorf("expected chatNotice to indicate success, got %q", wipedMod.chatNotice)
	}
}
