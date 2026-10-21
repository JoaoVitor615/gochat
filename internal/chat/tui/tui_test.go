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
	"github.com/charmbracelet/lipgloss"
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
	if isPeerOnline(peer{online: false}) {
		t.Error("peer with online=false should be offline")
	}
	if !isPeerOnline(peer{online: true}) {
		t.Error("peer with online=true should be online")
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
				online:      false,
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
	newM, _ := m.Update(presenceEventMsg{PeerID: "peer-1", LastSeenAt: time.Now(), Online: true})
	mod := newM.(model)

	// Online check
	sideOnline := strings.Join(mod.sidebar(25, 20), "\n")
	if !strings.Contains(sideOnline, "●") {
		t.Errorf("expected online circle ●, got:\n%s", sideOnline)
	}
	if !strings.Contains(sideOnline, "online") {
		t.Errorf("expected online in sidebar, got:\n%s", sideOnline)
	}

	// Now send disconnect event: Alice should be marked offline IMMEDIATELY with zero delay!
	offM, _ := mod.Update(presenceEventMsg{PeerID: "peer-1", Online: false})
	offMod := offM.(model)

	sideOfflineNow := strings.Join(offMod.sidebar(25, 20), "\n")
	if !strings.Contains(sideOfflineNow, "○") {
		t.Errorf("expected offline circle ○ immediately on disconnect, got:\n%s", sideOfflineNow)
	}
	if !strings.Contains(sideOfflineNow, "visto") {
		t.Errorf("expected visto in offline sidebar immediately on disconnect, got:\n%s", sideOfflineNow)
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

	// View should render full-screen settings without contacts sidebar
	view := mod.View()
	if !strings.Contains(view, "CONFIGURAÇÕES") {
		t.Errorf("expected view to contain 'CONFIGURAÇÕES', got:\n%s", view)
	}
	if !strings.Contains(view, "Español") || !strings.Contains(view, "English") || !strings.Contains(view, "Português") {
		t.Errorf("expected view to list all languages, got:\n%s", view)
	}
	if strings.Contains(view, "CONTATOS") || strings.Contains(view, "PEERS") {
		t.Errorf("expected settings view to NOT show contacts sidebar, got:\n%s", view)
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
	if strings.Contains(view, "Alice") {
		t.Errorf("expected contacts (Alice) to be hidden in full-screen settings, got:\n%s", view)
	}
	for i, line := range strings.Split(view, "\n") {
		if w := lipgloss.Width(line); w != m.width {
			t.Errorf("row %d width mismatch: expected %d, got %d for line:\n%s", i, m.width, w, line)
		}
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

func TestModel_UnreadBadgeLifecycle(t *testing.T) {
	orig := i18n.CurrentLanguage()
	defer i18n.SetLanguage(orig)
	i18n.SetLanguage(i18n.LangPtBR)

	mockStore := &mockWipeStore{}
	m := model{
		width:     80,
		height:    24,
		splash:    false,
		active:    -1,
		localID:   "local-peer",
		peerStore: mockStore,
		peers: []peer{
			{id: "peer-joao", displayName: "João"},
			{id: "peer-maria", displayName: "Maria"},
		},
		panel: noPanel,
	}

	// 1. Initial state: no unread messages
	view := m.View()
	if !strings.Contains(view, "João") {
		t.Fatalf("expected view to contain João")
	}
	if strings.Contains(view, "(1)") || strings.Contains(view, "(2)") {
		t.Errorf("expected no unread badge initially, got:\n%s", view)
	}

	// 2. Incoming message from João while no chat is open -> unreadCount becomes 1
	msg1 := storage.StoredMessage{
		ConversationPeerID: "peer-joao",
		Envelope: message.Message{
			ID:           "msg-1",
			SenderPeerID: "peer-joao",
			Content:      "Olá, tudo bem?",
			CreatedAt:    time.Now().UTC(),
		},
	}
	newM, _ := m.Update(messageEventMsg(msg1))
	m = newM.(model)

	joaoIdx := m.peerIndex("peer-joao")
	if joaoIdx < 0 || m.peers[joaoIdx].unreadCount != 1 {
		t.Fatalf("expected João unreadCount to be 1, got %v", m.peers[joaoIdx].unreadCount)
	}
	view = m.View()
	if !strings.Contains(view, "(1)") {
		t.Errorf("expected view to contain '(1)' for João, got:\n%s", view)
	}

	// 3. Second incoming message from João -> unreadCount becomes 2
	msg2 := storage.StoredMessage{
		ConversationPeerID: "peer-joao",
		Envelope: message.Message{
			ID:           "msg-2",
			SenderPeerID: "peer-joao",
			Content:      "Viu minha mensagem anterior?",
			CreatedAt:    time.Now().UTC().Add(time.Second),
		},
	}
	newM, _ = m.Update(messageEventMsg(msg2))
	m = newM.(model)

	joaoIdx = m.peerIndex("peer-joao")
	if m.peers[joaoIdx].unreadCount != 2 {
		t.Fatalf("expected João unreadCount to be 2, got %v", m.peers[joaoIdx].unreadCount)
	}
	view = m.View()
	if !strings.Contains(view, "(2)") {
		t.Errorf("expected view to contain '(2)' for João, got:\n%s", view)
	}

	// 4. User selects João and presses Enter to open conversation -> unreadCount resets to 0
	m.selected = joaoIdx
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newM.(model)

	if m.active != joaoIdx {
		t.Fatalf("expected active conversation to be João (%d), got %d", joaoIdx, m.active)
	}
	if m.peers[joaoIdx].unreadCount != 0 {
		t.Errorf("expected unreadCount to reset to 0 upon opening chat, got %d", m.peers[joaoIdx].unreadCount)
	}
	view = m.View()
	if strings.Contains(view, "(2)") || strings.Contains(view, "(1)") {
		t.Errorf("expected unread badge to disappear after opening conversation, got:\n%s", view)
	}

	// 5. Incoming message from João while chat is ALREADY open -> unreadCount stays 0
	msg3 := storage.StoredMessage{
		ConversationPeerID: "peer-joao",
		Envelope: message.Message{
			ID:           "msg-3",
			SenderPeerID: "peer-joao",
			Content:      "Mensagem com chat aberto",
			CreatedAt:    time.Now().UTC().Add(2 * time.Second),
		},
	}
	newM, _ = m.Update(messageEventMsg(msg3))
	m = newM.(model)

	if m.peers[joaoIdx].unreadCount != 0 {
		t.Errorf("expected unreadCount to remain 0 while chat is open, got %d", m.peers[joaoIdx].unreadCount)
	}

	// 6. Incoming message from Maria while João is active -> Maria's unreadCount becomes 1
	msgMaria := storage.StoredMessage{
		ConversationPeerID: "peer-maria",
		Envelope: message.Message{
			ID:           "msg-4",
			SenderPeerID: "peer-maria",
			Content:      "Oi do outro contato",
			CreatedAt:    time.Now().UTC().Add(3 * time.Second),
		},
	}
	newM, _ = m.Update(messageEventMsg(msgMaria))
	m = newM.(model)

	mariaIdx := m.peerIndex("peer-maria")
	if m.peers[mariaIdx].unreadCount != 1 {
		t.Errorf("expected Maria unreadCount to be 1, got %d", m.peers[mariaIdx].unreadCount)
	}
	view = m.View()
	if !strings.Contains(view, "(1)") {
		t.Errorf("expected view to show badge for Maria, got:\n%s", view)
	}
}
