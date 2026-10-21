// Package tui implements the interactive terminal interface for gochat.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/events"
	"github.com/JoaoVitor615/gochat/internal/chat/logger"
	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/messaging"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	"github.com/JoaoVitor615/gochat/internal/i18n"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// PeerAPI is the Discovery API functionality needed by the TUI.
type PeerAPI interface {
	CreateInvite(context.Context, string) (*client.CreateInviteResponse, error)
	AddPeer(context.Context, string, string) (*client.AddPeerResponse, error)
	ListAcceptances(context.Context, string) (*client.ListAcceptancesResponse, error)
	AcknowledgeAcceptances(context.Context, string, []string) error
}

// PeerStore is the local contact persistence needed by the TUI.
type PeerStore interface {
	ListPeers(context.Context) ([]storage.Peer, error)
	SavePeer(context.Context, storage.Peer) error
	SetPeerDisplayName(context.Context, string, string) error
	ListMessages(context.Context, string, *message.Message, int) ([]storage.StoredMessage, error)
	CryptographicErasure() error
}

type MessageSender interface {
	Send(context.Context, string, string) (messaging.SendResult, error)
	AddPeerAddresses(string, []string) error
	ConnectPeer(context.Context, string) error
}

type peer struct {
	id            string
	displayName   string
	addresses     []string
	addedAt       time.Time
	lastSeenAt    time.Time
	lastMessageAt time.Time
}

type panel uint8

const (
	noPanel panel = iota
	addPanel
	invitePanel
	nicknamePanel
	logsPanel
	settingsPanel
)

type model struct {
	ctx                    context.Context
	api                    PeerAPI
	peerStore              PeerStore
	sender                 MessageSender
	localID                string
	width                  int
	height                 int
	selected               int
	active                 int
	panel                  panel
	settingsSelected       int
	settingsConfirmWipe    bool
	code                   string
	invite                 string
	nicknameInput          string
	nicknamePeerID         string
	acceptancePollUntil    time.Time
	acceptancePollInFlight bool
	loading                bool
	errText                string
	peers                  []peer
	messages               []storage.StoredMessage
	hasOlder               bool
	loadingChat            bool
	sending                bool
	input                  string
	chatNotice             string
	scrollBack             int
	logsScroll             int
	busCh                  chan storage.StoredMessage
	presenceCh             chan events.PeerPresence
	splash                 bool
	splashSpinnerIdx       int
	splashStatus           string
	splashErr              error
	splashInitDone         bool
	splashMinDone          bool
	stopPresence           func()
	initFn                 func(context.Context) (MessageSender, func(), error)
}

type inviteResult struct {
	code      string
	createdAt time.Time
	err       error
}

type addPeerResult struct {
	peer           *client.AddPeerResponse
	err            error
	persistenceErr error
	connectionErr  error
	lastSeenAt     time.Time
}

type acceptancesResult struct {
	peers []acceptedPeerResult
	err   error
}

type acceptedPeerResult struct {
	acceptance     client.InviteAcceptance
	persistenceErr error
	connectionErr  error
}

const (
	messagePageSize = 50
	refreshInterval = 2 * time.Second
	inviteLifetime  = 10 * time.Minute
)

type historyResult struct {
	peerID   string
	messages []storage.StoredMessage
	older    bool
	hasOlder bool
	err      error
}

type sendResult struct {
	peerID string
	text   string
	result messaging.SendResult
	err    error
}

type refreshTick time.Time

var (
	accent = lipgloss.Color("#7DD3FC")
	muted  = lipgloss.Color("#94A3B8")
	white  = lipgloss.Color("#F8FAFC")

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	peerStyle     = lipgloss.NewStyle().Foreground(white)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0F172A")).Background(accent)
	borderStyle   = lipgloss.NewStyle().Foreground(muted)
)

// Run opens the chat TUI and keeps peer operations tied to the app context.
func Run(ctx context.Context, localPeerID string, api PeerAPI, peerStore PeerStore, sender MessageSender, bus *events.Bus) error {
	return RunWithSplash(ctx, localPeerID, api, peerStore, bus, func(context.Context) (MessageSender, func(), error) {
		return sender, nil, nil
	})
}

// RunWithSplash opens the chat TUI with an animated splash screen while executing the initialization function.
func RunWithSplash(ctx context.Context, localPeerID string, api PeerAPI, peerStore PeerStore, bus *events.Bus, initFn func(context.Context) (MessageSender, func(), error)) error {
	savedPeers, err := peerStore.ListPeers(ctx)
	if err != nil {
		return fmt.Errorf("load saved peers: %w", err)
	}
	var busCh chan storage.StoredMessage
	var presenceCh chan events.PeerPresence
	if bus != nil {
		busCh = bus.Subscribe()
		defer bus.Unsubscribe(busCh)
		presenceCh = bus.SubscribePresence()
		defer bus.UnsubscribePresence(presenceCh)
	}
	initial := model{
		ctx:          ctx,
		api:          api,
		peerStore:    peerStore,
		localID:      localPeerID,
		active:       -1,
		busCh:        busCh,
		presenceCh:   presenceCh,
		splash:       true,
		splashStatus: i18n.T.SplashConnecting(),
		initFn:       initFn,
	}
	for _, saved := range savedPeers {
		var lastMsgAt time.Time
		if msgs, err := peerStore.ListMessages(ctx, saved.PeerID, nil, 1); err == nil && len(msgs) > 0 {
			lastMsgAt = msgs[0].Envelope.CreatedAt
		}
		initial.peers = append(initial.peers, peer{
			id:            saved.PeerID,
			displayName:   saved.DisplayName,
			addresses:     saved.Addresses,
			addedAt:       saved.AddedAt,
			lastSeenAt:    saved.LastSeenAt,
			lastMessageAt: lastMsgAt,
		})
	}
	initial.sortPeers()
	p := tea.NewProgram(initial, tea.WithAltScreen(), tea.WithContext(ctx), tea.WithMouseCellMotion())
	finalModel, err := p.Run()
	if finalM, ok := finalModel.(model); ok && finalM.stopPresence != nil {
		finalM.stopPresence()
	}
	return err
}

type splashTickMsg time.Time
type splashMinDurationMsg struct{}
type splashInitResult struct {
	sender       MessageSender
	stopPresence func()
	err          error
}

func splashTickCmd() tea.Cmd {
	return tea.Tick(75*time.Millisecond, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

func splashMinDurationCmd() tea.Cmd {
	return tea.Tick(1200*time.Millisecond, func(_ time.Time) tea.Msg {
		return splashMinDurationMsg{}
	})
}

func (m model) runInitCmd() tea.Cmd {
	return func() tea.Msg {
		if m.initFn == nil {
			return splashInitResult{}
		}
		sender, stopPresence, err := m.initFn(m.ctx)
		return splashInitResult{
			sender:       sender,
			stopPresence: stopPresence,
			err:          err,
		}
	}
}

type messageEventMsg storage.StoredMessage

func waitForEventCmd(ch chan storage.StoredMessage) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return messageEventMsg(msg)
	}
}

type presenceEventMsg events.PeerPresence

func waitForPresenceCmd(ch chan events.PeerPresence) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return presenceEventMsg(event)
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{}
	if m.busCh != nil {
		cmds = append(cmds, waitForEventCmd(m.busCh))
	}
	if m.presenceCh != nil {
		cmds = append(cmds, waitForPresenceCmd(m.presenceCh))
	}
	if m.splash {
		cmds = append(cmds, splashTickCmd(), splashMinDurationCmd())
		if m.initFn != nil {
			cmds = append(cmds, m.runInitCmd())
		} else {
			m.splashInitDone = true
		}
	} else {
		cmds = append(cmds, refreshTickCmd())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case splashTickMsg:
		if m.splash {
			m.splashSpinnerIdx = (m.splashSpinnerIdx + 1) % len(spinnerFrames)
			return m, splashTickCmd()
		}
		return m, nil
	case splashMinDurationMsg:
		m.splashMinDone = true
		if m.splashInitDone && m.splashErr == nil {
			m.splash = false
			return m, refreshTickCmd()
		}
		return m, nil
	case splashInitResult:
		m.splashInitDone = true
		if msg.err != nil {
			m.splashErr = msg.err
			m.splashStatus = i18n.T.SplashError()
			return m, nil
		}
		m.sender = msg.sender
		m.stopPresence = msg.stopPresence
		m.splashStatus = i18n.T.SplashConnected()
		if m.splashMinDone {
			m.splash = false
			return m, refreshTickCmd()
		}
		return m, nil
	case refreshTick:
		if m.splash {
			return m, nil
		}
		commands := []tea.Cmd{refreshTickCmd()}
		if !m.acceptancePollUntil.IsZero() && time.Now().Before(m.acceptancePollUntil) && !m.acceptancePollInFlight {
			m.acceptancePollInFlight = true
			commands = append(commands, m.listAcceptancesCmd())
		}
		if peerID := m.activePeerID(); peerID != "" && m.scrollBack == 0 && !m.loadingChat {
			m.loadingChat = true
			commands = append(commands, m.loadHistoryCmd(peerID, nil, false))
		}
		return m, tea.Batch(commands...)
	case messageEventMsg:
		stored := storage.StoredMessage(msg)
		if stored.ConversationPeerID == m.activePeerID() {
			m.messages = appendMessage(m.messages, stored)
			if m.scrollBack == 0 {
				m.scrollBack = 0
			}
			if stored.Envelope.SenderPeerID == m.localID && stored.Status == message.DeliveryDelivered {
				m.chatNotice = i18n.T.NoticeDeliveredAck()
			}
		}
		if idx := m.peerIndex(stored.ConversationPeerID); idx >= 0 {
			if stored.Envelope.CreatedAt.After(m.peers[idx].lastSeenAt) {
				m.peers[idx].lastSeenAt = stored.Envelope.CreatedAt
			}
			if stored.Envelope.CreatedAt.After(m.peers[idx].lastMessageAt) {
				m.peers[idx].lastMessageAt = stored.Envelope.CreatedAt
			}
			m.sortPeers()
		}
		return m, waitForEventCmd(m.busCh)
	case presenceEventMsg:
		event := events.PeerPresence(msg)
		if idx := m.peerIndex(event.PeerID); idx >= 0 {
			if event.LastSeenAt.After(m.peers[idx].lastSeenAt) {
				m.peers[idx].lastSeenAt = event.LastSeenAt
			}
		}
		return m, waitForPresenceCmd(m.presenceCh)
	case historyResult:
		if msg.peerID != m.activePeerID() {
			return m, nil
		}
		m.loadingChat = false
		if msg.err != nil {
			m.chatNotice = i18n.T.NoticeHistoryError(msg.err)
			return m, nil
		}
		m.hasOlder = msg.hasOlder
		if msg.older {
			m.messages = prependMessages(msg.messages, m.messages)
			m.scrollBack += len(msg.messages) * 3
			maxScroll := m.maxScrollBack()
			if m.scrollBack > maxScroll && !m.hasOlder {
				m.scrollBack = maxScroll
			}
		} else {
			m.messages = msg.messages
			m.scrollBack = 0
			m.chatNotice = ""
		}
	case sendResult:
		m.sending = false
		if msg.peerID != m.activePeerID() {
			return m, nil
		}
		if msg.err != nil {
			m.chatNotice = i18n.T.NoticeQueueError(msg.err)
			return m, nil
		}
		if msg.result.Message.ID != "" {
			m.messages = appendMessage(m.messages, storage.StoredMessage{
				ConversationPeerID: msg.peerID,
				Envelope:           msg.result.Message,
				Status:             msg.result.Status,
			})
			m.scrollBack = 0
			if m.input == msg.text {
				m.input = ""
			}
			if idx := m.peerIndex(msg.peerID); idx >= 0 {
				if msg.result.Message.CreatedAt.After(m.peers[idx].lastMessageAt) {
					m.peers[idx].lastMessageAt = msg.result.Message.CreatedAt
				}
				m.sortPeers()
			}
		}
		if msg.result.DeliveryError != nil {
			m.chatNotice = i18n.T.NoticeSavedWaitingDelivery(msg.result.DeliveryError)
		} else if msg.result.Status == message.DeliveryDelivered {
			m.chatNotice = i18n.T.NoticeDeliveredAck()
		} else {
			m.chatNotice = i18n.T.NoticeSavedQueue()
		}
	case inviteResult:
		m.loading = false
		m.invite, m.errText = msg.code, errorText(msg.err)
		if msg.err == nil && msg.code != "" {
			m.acceptancePollUntil = msg.createdAt.Add(inviteLifetime)
			m.acceptancePollInFlight = false
		}
	case addPeerResult:
		m.loading = false
		if msg.err != nil {
			m.errText = msg.err.Error()
			return m, nil
		}
		if msg.peer == nil || msg.peer.PeerID == "" {
			m.errText = i18n.T.AddPanelErrorNoPeerID()
			return m, nil
		}
		m.upsertPeer(peer{
			id: msg.peer.PeerID, addresses: msg.peer.Addresses,
			addedAt:    time.Now().UTC(),
			lastSeenAt: msg.lastSeenAt,
		})
		m.selected = m.peerIndex(msg.peer.PeerID)
		m.active = m.selected
		if msg.persistenceErr != nil {
			m.errText = i18n.T.NoticePeerNotSaved(msg.persistenceErr)
			return m, nil
		}
		m.panel, m.code, m.errText = noPanel, "", ""
		if msg.connectionErr != nil {
			m.chatNotice = i18n.T.NoticePeerSavedDialFailed(msg.connectionErr)
		} else {
			m.chatNotice = i18n.T.NoticeP2PConnected()
		}
	case acceptancesResult:
		m.acceptancePollInFlight = false
		if msg.err != nil {
			if m.panel == invitePanel {
				m.errText = i18n.T.InvitePanelError(msg.err)
			}
			return m, nil
		}
		if len(msg.peers) == 0 {
			return m, nil
		}
		m.acceptancePollUntil = time.Time{}
		for _, accepted := range msg.peers {
			m.upsertPeer(peer{
				id:         accepted.acceptance.PeerID,
				addresses:  accepted.acceptance.Addresses,
				addedAt:    accepted.acceptance.AcceptedAt,
				lastSeenAt: accepted.acceptance.AcceptedAt,
			})
		}
		accepted := msg.peers[len(msg.peers)-1]
		m.selected = m.peerIndex(accepted.acceptance.PeerID)
		m.active = m.selected
		m.messages = nil
		m.hasOlder = false
		m.scrollBack = 0
		m.loadingChat = true
		m.panel, m.invite, m.errText = noPanel, "", ""
		if accepted.persistenceErr != nil {
			m.chatNotice = i18n.T.NoticeInviteAcceptedNotSaved(accepted.persistenceErr)
		} else if accepted.connectionErr != nil {
			m.chatNotice = i18n.T.NoticeInviteAcceptedDialFailed(accepted.connectionErr)
		} else {
			m.chatNotice = i18n.T.NoticeInviteAcceptedConnected()
		}
		return m, m.loadHistoryCmd(m.activePeerID(), nil, false)
	case tea.MouseMsg:
		if m.splash {
			return m, nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.panel == logsPanel {
				m.logsScroll += 3
				return m, nil
			}
			if m.active >= 0 {
				step := 3
				maxScroll := m.maxScrollBack()
				m.scrollBack += step
				if m.scrollBack > maxScroll && !m.hasOlder {
					m.scrollBack = maxScroll
				}
				if m.hasOlder && !m.loadingChat && len(m.messages) > 0 && m.scrollBack >= maxScroll {
					m.loadingChat = true
					before := m.messages[0].Envelope
					return m, m.loadHistoryCmd(m.activePeerID(), &before, true)
				}
				return m, nil
			}
			if m.selected > 0 {
				m.selected--
				return m, nil
			}
		case tea.MouseButtonWheelDown:
			if m.panel == logsPanel {
				if m.logsScroll > 3 {
					m.logsScroll -= 3
				} else {
					m.logsScroll = 0
				}
				return m, nil
			}
			if m.active >= 0 {
				m.scrollBack = max(0, m.scrollBack-3)
				return m, nil
			}
			if m.selected < len(m.peers)-1 {
				m.selected++
				return m, nil
			}
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			if m.stopPresence != nil {
				m.stopPresence()
			}
			return m, tea.Quit
		}
		if m.splash {
			switch msg.String() {
			case "q", "esc":
				if m.splashErr != nil {
					if m.stopPresence != nil {
						m.stopPresence()
					}
					return m, tea.Quit
				}
			case "enter", "space":
				if m.splashInitDone && m.splashErr == nil {
					m.splash = false
					return m, refreshTickCmd()
				}
			}
			return m, nil
		}
		if msg.String() == "ctrl+l" {
			if m.panel == logsPanel {
				m.panel = noPanel
			} else {
				m.panel = logsPanel
				m.logsScroll = 0
			}
			return m, nil
		}
		if m.panel == logsPanel {
			return m.updateLogsPanel(msg)
		}
		if m.panel == settingsPanel {
			return m.updateSettingsPanel(msg)
		}
		if m.panel == addPanel {
			return m.updateAddPanel(msg)
		}
		if m.panel == invitePanel {
			switch msg.String() {
			case "esc":
				m.panel = noPanel
			case "r":
				m.loading, m.errText, m.invite = true, "", ""
				return m, m.createInviteCmd()
			}
			return m, nil
		}
		if m.panel == nicknamePanel {
			return m.updateNicknamePanel(msg)
		}
		if m.active >= 0 {
			return m.updateChat(msg)
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.peers)-1 {
				m.selected++
			}
		case "enter":
			if m.selected >= 0 && m.selected < len(m.peers) {
				m.active = m.selected
				m.messages = nil
				m.hasOlder = false
				m.loadingChat = true
				m.chatNotice = ""
				m.scrollBack = 0
				return m, m.loadHistoryCmd(m.activePeerID(), nil, false)
			}
		case "esc", "s":
			m.panel = settingsPanel
			m.settingsSelected = m.currentLanguageIndex()
			m.settingsConfirmWipe = false
		case "a":
			m.panel, m.code, m.errText = addPanel, "", ""
		case "i":
			m.panel, m.invite, m.errText, m.loading = invitePanel, "", "", true
			return m, m.createInviteCmd()
		case "n":
			if m.selected >= 0 && m.selected < len(m.peers) {
				m.panel = nicknamePanel
				m.nicknamePeerID = m.peers[m.selected].id
				m.nicknameInput = m.peers[m.selected].displayName
				m.errText = ""
			}
		}
	}
	return m, nil
}

func (m model) currentLanguageIndex() int {
	cur := i18n.CurrentLanguage()
	for i, l := range i18n.AvailableLanguages() {
		if l.Tag == cur {
			return i
		}
	}
	return 0
}

func (m model) updateSettingsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.settingsConfirmWipe {
		switch msg.String() {
		case "y", "Y", "s", "S":
			return m.performCryptographicErasure()
		case "n", "N", "esc":
			m.settingsConfirmWipe = false
		}
		return m, nil
	}

	langs := i18n.AvailableLanguages()
	totalOptions := len(langs) + 1

	switch msg.String() {
	case "esc":
		m.panel = noPanel
		m.settingsConfirmWipe = false
	case "up", "k":
		if m.settingsSelected > 0 {
			m.settingsSelected--
		}
	case "down", "j":
		if m.settingsSelected < totalOptions-1 {
			m.settingsSelected++
		}
	case "enter", " ":
		if m.settingsSelected >= 0 && m.settingsSelected < len(langs) {
			i18n.SetLanguage(langs[m.settingsSelected].Tag)
		} else if m.settingsSelected == len(langs) {
			m.settingsConfirmWipe = true
		}
	}
	return m, nil
}

func (m model) performCryptographicErasure() (model, tea.Cmd) {
	if m.peerStore != nil {
		if err := m.peerStore.CryptographicErasure(); err != nil {
			m.chatNotice = i18n.T.NoticeWipeError(err)
			m.settingsConfirmWipe = false
			return m, nil
		}
		newStore, err := storage.OpenDefault()
		if err == nil {
			m.peerStore = newStore
		}
	}

	m.peers = nil
	m.messages = nil
	m.active = -1
	m.selected = 0
	m.code = ""
	m.invite = ""
	m.input = ""
	m.scrollBack = 0
	m.chatNotice = i18n.T.NoticeWipeSuccess()
	m.panel = noPanel
	m.settingsConfirmWipe = false

	return m, nil
}

func (m model) updateLogsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.panel = noPanel
	case "up", "k":
		m.logsScroll++
	case "down", "j":
		if m.logsScroll > 0 {
			m.logsScroll--
		}
	case "pgup":
		m.logsScroll += 10
	case "pgdown":
		if m.logsScroll > 10 {
			m.logsScroll -= 10
		} else {
			m.logsScroll = 0
		}
	case "home":
		m.logsScroll = 999999
	case "end":
		m.logsScroll = 0
	}
	return m, nil
}

func (m model) updateAddPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.panel, m.code, m.errText = noPanel, "", ""
	case "enter":
		code := strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(m.code)), "-", "")
		if len(code) != 9 {
			m.errText = i18n.T.AddPanelErrorLength()
			return m, nil
		}
		m.loading, m.errText = true, ""
		return m, m.addPeerCmd(code)
	case "backspace", "ctrl+h":
		if len(m.code) > 0 && !m.loading {
			m.code = m.code[:len(m.code)-1]
		}
	default:
		if !m.loading && msg.Type == tea.KeyRunes {
			for _, char := range strings.ToUpper(string(msg.Runes)) {
				if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
					if len(m.code) < 9 {
						m.code += string(char)
					}
				}
			}
		}
	}
	return m, nil
}

func (m model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.active, m.input, m.chatNotice = -1, "", ""
	case "ctrl+n":
		peerID := m.activePeerID()
		if peerID != "" {
			m.panel = nicknamePanel
			m.nicknamePeerID = peerID
			m.nicknameInput = m.peers[m.active].displayName
			m.errText = ""
			return m, nil
		}
	case "enter":
		if m.sending || strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		rawInput := strings.TrimSpace(m.input)
		if isNicknameCommand(rawInput) {
			return m.handleNicknameCommand(rawInput)
		}
		if m.sender == nil {
			m.chatNotice = i18n.T.NoticeSenderUnavailable()
			return m, nil
		}
		peerID, text := m.activePeerID(), m.input
		m.sending, m.chatNotice = true, i18n.T.NoticeEnqueuing()
		return m, func() tea.Msg {
			result, err := m.sender.Send(m.ctx, peerID, text)
			return sendResult{peerID: peerID, text: text, result: result, err: err}
		}
	case "backspace", "ctrl+h":
		if runes := []rune(m.input); len(runes) > 0 {
			m.input = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.input = ""
	case "pgup":
		step := max(1, m.height/3)
		maxScroll := m.maxScrollBack()
		m.scrollBack += step
		if m.scrollBack > maxScroll && !m.hasOlder {
			m.scrollBack = maxScroll
		}
		if m.hasOlder && !m.loadingChat && len(m.messages) > 0 && m.scrollBack >= maxScroll {
			m.loadingChat = true
			before := m.messages[0].Envelope
			return m, m.loadHistoryCmd(m.activePeerID(), &before, true)
		}
	case "pgdown":
		m.scrollBack = max(0, m.scrollBack-max(1, m.height/3))
	default:
		if msg.Type == tea.KeyRunes && !m.sending {
			m.input += string(msg.Runes)
		}
	}
	return m, nil
}

func isNicknameCommand(input string) bool {
	return input == "/nick" || strings.HasPrefix(input, "/nick ") ||
		input == "/apelido" || strings.HasPrefix(input, "/apelido ") ||
		input == "/nickname" || strings.HasPrefix(input, "/nickname ")
}

func (m model) handleNicknameCommand(input string) (tea.Model, tea.Cmd) {
	peerID := m.activePeerID()
	if peerID == "" {
		m.input = ""
		return m, nil
	}
	var newName string
	var hasArg bool
	if strings.HasPrefix(input, "/nick ") {
		newName, hasArg = strings.TrimSpace(strings.TrimPrefix(input, "/nick ")), true
	} else if strings.HasPrefix(input, "/apelido ") {
		newName, hasArg = strings.TrimSpace(strings.TrimPrefix(input, "/apelido ")), true
	} else if strings.HasPrefix(input, "/nickname ") {
		newName, hasArg = strings.TrimSpace(strings.TrimPrefix(input, "/nickname ")), true
	}

	if !hasArg {
		// Open nickname dialog when command is invoked without argument
		m.panel = nicknamePanel
		m.nicknamePeerID = peerID
		m.nicknameInput = m.peers[m.active].displayName
		m.errText = ""
		m.input = ""
		return m, nil
	}

	if newName == "-" || newName == `""` || newName == "''" {
		newName = ""
	}

	if err := m.peerStore.SetPeerDisplayName(m.ctx, peerID, newName); err != nil {
		m.chatNotice = i18n.T.NicknamePanelError(err)
	} else {
		if idx := m.peerIndex(peerID); idx >= 0 {
			m.peers[idx].displayName = newName
		}
		if newName != "" {
			m.chatNotice = i18n.T.NoticeNicknameUpdated()
		} else {
			m.chatNotice = i18n.T.NoticeNicknameRemoved()
		}
	}
	m.input = ""
	return m, nil
}

func (m model) updateNicknamePanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.panel, m.nicknameInput, m.nicknamePeerID, m.errText = noPanel, "", "", ""
	case "enter":
		targetPeerID := m.nicknamePeerID
		if targetPeerID == "" {
			if m.active >= 0 && m.active < len(m.peers) {
				targetPeerID = m.peers[m.active].id
			} else if m.selected >= 0 && m.selected < len(m.peers) {
				targetPeerID = m.peers[m.selected].id
			}
		}
		if targetPeerID == "" {
			m.panel = noPanel
			return m, nil
		}
		newName := strings.TrimSpace(m.nicknameInput)
		if err := m.peerStore.SetPeerDisplayName(m.ctx, targetPeerID, newName); err != nil {
			m.errText = i18n.T.NicknamePanelError(err)
			return m, nil
		}
		if idx := m.peerIndex(targetPeerID); idx >= 0 {
			m.peers[idx].displayName = newName
		}
		m.panel, m.nicknameInput, m.nicknamePeerID, m.errText = noPanel, "", "", ""
		if newName != "" {
			m.chatNotice = i18n.T.NoticeNicknameUpdated()
		} else {
			m.chatNotice = i18n.T.NoticeNicknameRemoved()
		}
	case "backspace", "ctrl+h":
		if runes := []rune(m.nicknameInput); len(runes) > 0 {
			m.nicknameInput = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.nicknameInput = ""
	case "space", " ":
		if len([]rune(m.nicknameInput)) < 30 {
			m.nicknameInput += " "
		}
	default:
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if len([]rune(m.nicknameInput)) < 30 {
					m.nicknameInput += string(r)
				}
			}
		}
	}
	return m, nil
}

func (m model) loadHistoryCmd(peerID string, before *message.Message, older bool) tea.Cmd {
	var cursor *message.Message
	if before != nil {
		copy := *before
		cursor = &copy
	}
	return func() tea.Msg {
		messages, err := m.peerStore.ListMessages(m.ctx, peerID, cursor, messagePageSize+1)
		if err != nil {
			return historyResult{peerID: peerID, older: older, err: err}
		}
		hasOlder := len(messages) > messagePageSize
		if hasOlder {
			messages = messages[len(messages)-messagePageSize:]
		}
		return historyResult{peerID: peerID, messages: messages, older: older, hasOlder: hasOlder}
	}
}

func refreshTickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return refreshTick(t) })
}

func (m model) activePeerID() string {
	if m.active < 0 || m.active >= len(m.peers) {
		return ""
	}
	return m.peers[m.active].id
}

func appendMessage(existing []storage.StoredMessage, added storage.StoredMessage) []storage.StoredMessage {
	for i, current := range existing {
		if current.Envelope.ID == added.Envelope.ID {
			existing[i] = added
			return existing
		}
	}
	return sortedMessages(append(existing, added))
}

func prependMessages(older, current []storage.StoredMessage) []storage.StoredMessage {
	combined := make([]storage.StoredMessage, 0, len(older)+len(current))
	combined = append(combined, older...)
	combined = append(combined, current...)
	return sortedMessages(combined)
}

func sortedMessages(messages []storage.StoredMessage) []storage.StoredMessage {
	for i := 1; i < len(messages); i++ {
		for j := i; j > 0; j-- {
			left, right := messages[j-1].Envelope, messages[j].Envelope
			if message.Compare(left, right) <= 0 {
				break
			}
			messages[j-1], messages[j] = messages[j], messages[j-1]
		}
	}
	unique := messages[:0]
	seen := make(map[string]struct{}, len(messages))
	for _, stored := range messages {
		if _, ok := seen[stored.Envelope.ID]; ok {
			continue
		}
		seen[stored.Envelope.ID] = struct{}{}
		unique = append(unique, stored)
	}
	return unique
}

func (m model) createInviteCmd() tea.Cmd {
	return func() tea.Msg {
		createdAt := time.Now()
		response, err := m.api.CreateInvite(m.ctx, m.localID)
		if err != nil {
			return inviteResult{createdAt: createdAt, err: err}
		}
		if response == nil {
			return inviteResult{createdAt: createdAt, err: fmt.Errorf("a API não retornou um código de convite")}
		}
		return inviteResult{code: response.Code, createdAt: createdAt}
	}
}

func (m model) addPeerCmd(code string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.api.AddPeer(m.ctx, code, m.localID)
		result := addPeerResult{peer: response, err: err}
		if err != nil || response == nil || response.PeerID == "" {
			return result
		}
		if m.sender == nil {
			result.err = fmt.Errorf("serviço de conexão P2P não está disponível")
			return result
		}
		if err := m.sender.AddPeerAddresses(response.PeerID, response.Addresses); err != nil {
			result.err = fmt.Errorf("registrar endereços do peer: %w", err)
			return result
		}
		result.lastSeenAt = time.Now().UTC()
		result.persistenceErr = m.peerStore.SavePeer(m.ctx, storage.Peer{
			PeerID: response.PeerID, AddedAt: result.lastSeenAt,
			LastSeenAt: result.lastSeenAt, Addresses: response.Addresses,
		})
		if result.persistenceErr == nil {
			result.connectionErr = m.sender.ConnectPeer(m.ctx, response.PeerID)
		}
		return result
	}
}

func (m model) listAcceptancesCmd() tea.Cmd {
	return func() tea.Msg {
		response, err := m.api.ListAcceptances(m.ctx, m.localID)
		if err != nil {
			return acceptancesResult{err: err}
		}
		if response == nil || len(response.Acceptances) == 0 {
			return acceptancesResult{}
		}

		results := make([]acceptedPeerResult, 0, len(response.Acceptances))
		acknowledge := make([]string, 0, len(response.Acceptances))
		for _, acceptance := range response.Acceptances {
			if acceptance.InviteID == "" || acceptance.PeerID == "" {
				return acceptancesResult{err: fmt.Errorf("Discovery retornou uma aceitação inválida")}
			}
			if err := m.sender.AddPeerAddresses(acceptance.PeerID, acceptance.Addresses); err != nil {
				return acceptancesResult{err: fmt.Errorf("registrar endereços do peer aceito: %w", err)}
			}
			acceptedAt := acceptance.AcceptedAt
			if acceptedAt.IsZero() {
				acceptedAt = time.Now().UTC()
			}
			if err := m.peerStore.SavePeer(m.ctx, storage.Peer{
				PeerID: acceptance.PeerID, AddedAt: acceptedAt,
				LastSeenAt: acceptedAt, Addresses: acceptance.Addresses,
			}); err != nil {
				return acceptancesResult{err: fmt.Errorf("salvar peer aceito localmente: %w", err)}
			}
			acknowledge = append(acknowledge, acceptance.InviteID)
			results = append(results, acceptedPeerResult{
				acceptance:    acceptance,
				connectionErr: m.sender.ConnectPeer(m.ctx, acceptance.PeerID),
			})
		}
		if err := m.api.AcknowledgeAcceptances(m.ctx, m.localID, acknowledge); err != nil {
			return acceptancesResult{err: err}
		}
		return acceptancesResult{peers: results}
	}
}

func (m *model) upsertPeer(updated peer) {
	if index := m.peerIndex(updated.id); index >= 0 {
		if updated.displayName == "" {
			updated.displayName = m.peers[index].displayName
		}
		if updated.addedAt.IsZero() {
			updated.addedAt = m.peers[index].addedAt
		}
		if updated.lastMessageAt.IsZero() {
			updated.lastMessageAt = m.peers[index].lastMessageAt
		}
		m.peers[index] = updated
		m.sortPeers()
		return
	}
	m.peers = append(m.peers, updated)
	m.sortPeers()
}

func (m model) peerIndex(peerID string) int {
	for index, existing := range m.peers {
		if existing.id == peerID {
			return index
		}
	}
	return -1
}

func (m *model) sortPeers() {
	if len(m.peers) <= 1 {
		return
	}
	activeID := ""
	if m.active >= 0 && m.active < len(m.peers) {
		activeID = m.peers[m.active].id
	}
	selectedID := ""
	if m.selected >= 0 && m.selected < len(m.peers) {
		selectedID = m.peers[m.selected].id
	}

	sort.SliceStable(m.peers, func(i, j int) bool {
		pi, pj := m.peers[i], m.peers[j]
		if !pi.lastMessageAt.IsZero() && !pj.lastMessageAt.IsZero() {
			if !pi.lastMessageAt.Equal(pj.lastMessageAt) {
				return pi.lastMessageAt.After(pj.lastMessageAt)
			}
		}
		if !pi.lastMessageAt.IsZero() && pj.lastMessageAt.IsZero() {
			return true
		}
		if pi.lastMessageAt.IsZero() && !pj.lastMessageAt.IsZero() {
			return false
		}
		if !pi.addedAt.IsZero() && !pj.addedAt.IsZero() {
			if !pi.addedAt.Equal(pj.addedAt) {
				return pi.addedAt.After(pj.addedAt)
			}
		}
		if !pi.addedAt.IsZero() && pj.addedAt.IsZero() {
			return true
		}
		if pi.addedAt.IsZero() && !pj.addedAt.IsZero() {
			return false
		}
		return pi.id < pj.id
	})

	if activeID != "" {
		m.active = m.peerIndex(activeID)
	}
	if selectedID != "" {
		m.selected = m.peerIndex(selectedID)
	}
}


func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return i18n.T.OpeningApp()
	}
	if m.width < 58 || m.height < 21 {
		return i18n.T.TerminalTooSmall(58, 21)
	}
	if m.splash {
		return m.splashView()
	}

	innerWidth := m.width - 2
	sideWidth := min(29, max(22, innerWidth/3))
	mainWidth := innerWidth - sideWidth - 1
	bodyHeight := m.height - 6

	if m.panel == settingsPanel {
		var rows []string
		rows = append(rows, borderStyle.Render("┌─ gochat "+strings.Repeat("─", m.width-11)+"┐"))
		rows = append(rows, frameLine(" "+titleStyle.Render("gochat")+"   "+mutedStyle.Render("settings"), innerWidth))
		rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", innerWidth)+"┤"))

		settingsRows := m.settingsView(innerWidth, bodyHeight)
		for i := 0; i < bodyHeight; i++ {
			rows = append(rows, frameLine(settingsRows[i], innerWidth))
		}

		rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", innerWidth)+"┤"))
		footer := i18n.T.FooterSettings()
		rows = append(rows, frameLine(mutedStyle.Render(footer), innerWidth))
		rows = append(rows, borderStyle.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
		return strings.Join(rows, "\n")
	}

	var rows []string
	rows = append(rows, borderStyle.Render("┌─ gochat "+strings.Repeat("─", m.width-11)+"┐"))
	rows = append(rows, frameLine(" "+titleStyle.Render("gochat")+"   "+mutedStyle.Render("peers"), innerWidth))
	rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", sideWidth)+"┬"+strings.Repeat("─", mainWidth)+"┤"))

	side, main := m.sidebar(sideWidth, bodyHeight), m.mainArea(mainWidth, bodyHeight)
	for i := 0; i < bodyHeight; i++ {
		rows = append(rows, borderStyle.Render("│")+fit(side[i], sideWidth)+borderStyle.Render("│")+fit(main[i], mainWidth)+borderStyle.Render("│"))
	}

	rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", sideWidth)+"┴"+strings.Repeat("─", mainWidth)+"┤"))
	footer := i18n.T.FooterHome()
	if m.active >= 0 {
		footer = i18n.T.FooterChat()
	}
	if m.panel == nicknamePanel {
		footer = i18n.T.FooterNickname()
	} else if m.panel == logsPanel {
		footer = i18n.T.FooterLogs()
	} else if m.panel == invitePanel {
		footer = i18n.T.FooterInviteNew()
	} else if m.panel != noPanel {
		footer = i18n.T.FooterCancel()
	}
	rows = append(rows, frameLine(mutedStyle.Render(footer), innerWidth))
	rows = append(rows, borderStyle.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
	return strings.Join(rows, "\n")
}

func (m model) sidebar(width, height int) []string {
	rows := make([]string, height)
	rows[0] = "  " + titleStyle.Render(i18n.T.PeersTitle())
	for i, p := range m.peers {
		row := 2 + i*3
		if row+1 >= height-2 {
			break
		}
		name := p.displayName
		if name == "" {
			name = shortPeerID(p.id)
		}
		online := isPeerOnline(p.lastSeenAt)
		var circle string
		if online {
			circle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("●")
		} else {
			circle = mutedStyle.Render("○")
		}
		var label string
		if m.selected == i {
			label = fmt.Sprintf("  %s %s", circle, selectedStyle.Render(" "+name+" "))
		} else {
			label = fmt.Sprintf("  %s %s", circle, peerStyle.Render(name))
		}
		rows[row] = label
		if online {
			rows[row+1] = "    " + lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(i18n.T.OnlineStatus())
		} else {
			rows[row+1] = mutedStyle.Render("    " + peerLastSeen(p.lastSeenAt))
		}
	}
	rows[height-2] = mutedStyle.Render("  " + i18n.T.AddContactButton())
	return rows
}

func (m model) messageRowCount(width int) int {
	if len(m.messages) == 0 || m.active < 0 || m.active >= len(m.peers) {
		return 0
	}
	wrapWidth := max(1, width-6)
	p := m.peers[m.active]
	count := 0
	for _, stored := range m.messages {
		who, status := "Peer", i18n.T.StatusReceived()
		if stored.Envelope.SenderPeerID == m.localID {
			who, status = i18n.T.LabelYou(), deliveryLabel(stored.Status)
		} else if p.displayName != "" {
			who = p.displayName
		}
		heading := fmt.Sprintf("%s · %s · %s", who, stored.Envelope.CreatedAt.Local().Format("15:04"), status)
		count += len(wrapChatText(heading, wrapWidth))
		count += len(wrapChatText(stored.Envelope.Content, wrapWidth))
		count++
	}
	return count
}

func (m model) maxScrollBack() int {
	if m.width <= 0 || m.height <= 0 || len(m.messages) == 0 || m.active < 0 || m.active >= len(m.peers) {
		return 0
	}
	innerWidth := m.width - 2
	sideWidth := min(29, max(22, innerWidth/3))
	mainWidth := innerWidth - sideWidth - 1
	bodyHeight := m.height - 6
	messageBottom := bodyHeight - 4
	visibleRows := max(0, messageBottom-3)

	totalRows := m.messageRowCount(mainWidth)
	return max(0, totalRows-visibleRows)
}

func (m model) mainArea(width, height int) []string {
	if m.panel == logsPanel {
		return m.logsView(width, height)
	}
	if m.panel != noPanel {
		return m.dialog(width, height)
	}
	rows := make([]string, height)
	if m.active < 0 || m.active >= len(m.peers) {
		middle := height / 2
		rows[middle-2] = center(titleStyle.Render(i18n.T.NoChatOpenTitle()), width)
		rows[middle] = center(mutedStyle.Render(i18n.T.NoChatOpenHint()), width)
		return rows
	}
	p := m.peers[m.active]
	online := isPeerOnline(p.lastSeenAt)
	var circle string
	if online {
		circle = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render("●")
	} else {
		circle = mutedStyle.Render("○")
	}
	if p.displayName != "" {
		rows[0] = fmt.Sprintf("  %s %s  %s", circle, titleStyle.Render(p.displayName), mutedStyle.Render("("+shortPeerID(p.id)+")"))
	} else {
		rows[0] = fmt.Sprintf("  %s %s", circle, titleStyle.Render(shortPeerID(p.id)))
	}
	if online {
		rows[1] = "    " + lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Render(i18n.T.OnlineStatus())
	} else {
		rows[1] = "  " + mutedStyle.Render(peerLastSeen(p.lastSeenAt))
	}
	messageBottom := height - 4
	messageRows := make([]string, 0, height)
	if m.loadingChat && len(m.messages) == 0 {
		messageRows = append(messageRows, mutedStyle.Render("  "+i18n.T.NoticeLoadingOlder()))
	} else if len(m.messages) == 0 {
		messageRows = append(messageRows, mutedStyle.Render("  "+i18n.T.EmptyChatPrompt()))
	} else {
		for _, stored := range m.messages {
			envelope := stored.Envelope
			isLocal := envelope.SenderPeerID == m.localID
			who, status, headingStyle := "Peer", i18n.T.StatusReceived(), mutedStyle
			if isLocal {
				who, status, headingStyle = i18n.T.LabelYou(), deliveryLabel(stored.Status), titleStyle
			} else if p.displayName != "" {
				who = p.displayName
			}
			wrapWidth := max(1, width-6)
			heading := fmt.Sprintf("%s · %s · %s", who, envelope.CreatedAt.Local().Format("15:04"), status)
			for _, line := range wrapChatText(heading, wrapWidth) {
				messageRows = append(messageRows, alignChatLine(headingStyle.Render(line), width, isLocal))
			}
			for _, line := range wrapChatText(envelope.Content, wrapWidth) {
				messageRows = append(messageRows, alignChatLine(line, width, isLocal))
			}
			messageRows = append(messageRows, "")
		}
	}
	visibleRows := max(0, messageBottom-3)
	maxScroll := max(0, len(messageRows)-visibleRows)
	scroll := min(m.scrollBack, maxScroll)
	end := max(0, len(messageRows)-scroll)
	start := max(0, end-visibleRows)
	for i, line := range messageRows[start:end] {
		row := 3 + i
		if row >= messageBottom {
			break
		}
		rows[row] = line
	}
	rows[height-4] = borderStyle.Render(strings.Repeat("─", width))
	if m.chatNotice != "" {
		rows[height-3] = mutedStyle.Render("  " + ansi.Truncate(m.chatNotice, width-4, "…"))
	}
	rows[height-2] = "  > " + ansi.Truncate(m.input, max(0, width-8), "…")
	if m.sending {
		rows[height-2] += mutedStyle.Render("  " + i18n.T.SendingStatus())
	}
	return rows
}

func deliveryLabel(status message.DeliveryStatus) string {
	switch status {
	case message.DeliveryPending:
		return i18n.T.StatusQueued()
	case message.DeliverySent:
		return i18n.T.StatusSent()
	case message.DeliveryDelivered:
		return i18n.T.StatusDelivered()
	default:
		return string(status)
	}
}

func (m model) settingsView(width, height int) []string {
	rows := make([]string, height)

	if m.settingsConfirmWipe {
		cardWidth := min(72, width-6)
		warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
		bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FCA5A5")).Width(cardWidth - 6)
		promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FDE047")).Bold(true)

		boxContent := []string{
			warnStyle.Render("⚠  " + i18n.T.SettingsWipeConfirmTitle()),
			"",
			bodyStyle.Render(i18n.T.SettingsWipeWarning()),
			"",
			promptStyle.Render(i18n.T.SettingsWipePrompt()),
		}
		box := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#EF4444")).
			Padding(1, 3).
			Width(cardWidth).
			Render(strings.Join(boxContent, "\n"))

		boxLines := strings.Split(box, "\n")
		startRow := max(0, (height-len(boxLines))/2)
		for i, line := range boxLines {
			if startRow+i < height {
				rows[startRow+i] = center(line, width)
			}
		}
		return rows
	}

	var content []string
	content = append(content,
		"",
		"   "+titleStyle.Bold(true).Render(i18n.T.SettingsPanelTitle()),
		"   "+mutedStyle.Render(strings.Repeat("─", min(width-6, 68))),
		"",
		"   "+titleStyle.Render(i18n.T.SettingsLanguageSection()),
		"",
	)

	langs := i18n.AvailableLanguages()
	current := i18n.CurrentLanguage()
	for i, l := range langs {
		radio := "( )"
		if l.Tag == current {
			radio = "(*)"
		}
		label := fmt.Sprintf("%s %s (%s)", radio, l.Name, l.Tag)
		if m.settingsSelected == i {
			content = append(content, "   "+selectedStyle.Render(" > "+label+" "))
		} else {
			content = append(content, "     "+mutedStyle.Render(label))
		}
	}

	content = append(content,
		"",
		"   "+lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true).Render(i18n.T.SettingsDangerSection()),
		"",
	)

	wipeLabel := i18n.T.SettingsWipeOption()
	if m.settingsSelected == len(langs) {
		dangerSelectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#DC2626"))
		content = append(content, "   "+dangerSelectedStyle.Render(" > "+wipeLabel+" "))
	} else {
		dangerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
		content = append(content, "     "+dangerStyle.Render(wipeLabel))
	}

	content = append(content,
		"",
		"   "+mutedStyle.Render(i18n.T.InstructionSettings()),
	)

	if m.errText != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")).
			Width(max(1, width-8))
		content = append(content, "", "   "+errorStyle.Render(m.errText))
	}

	for i, line := range content {
		if i < height {
			rows[i] = line
		}
	}
	return rows
}

func (m model) dialog(width, height int) []string {
	rows := make([]string, height)
	boxWidth := min(76, width-4)
	var content []string
	if m.panel == addPanel {
		field := m.code
		if field == "" {
			field = "ABC123XYZ"
		}
		if len(field) > 6 {
			field = field[:3] + "-" + field[3:6] + "-" + field[6:]
		}
		field = ansi.Truncate(field, boxWidth-8, "…")
		instruction := i18n.T.InstructionAddPeer()
		if m.loading {
			instruction = i18n.T.InstructionQueryingInvite()
		}
		content = []string{titleStyle.Render(i18n.T.AddPanelTitle()), "", i18n.T.AddPanelPrompt(), "", lipgloss.NewStyle().Foreground(accent).Render("[ " + field + " ]"), "", mutedStyle.Render(instruction)}
	} else if m.panel == nicknamePanel {
		targetPeerID := m.nicknamePeerID
		displayName := ""
		if idx := m.peerIndex(targetPeerID); idx >= 0 {
			displayName = m.peers[idx].displayName
		}
		contactDesc := shortPeerID(targetPeerID)
		if displayName != "" {
			contactDesc += " (" + displayName + ")"
		}
		field := m.nicknameInput
		if field == "" {
			field = i18n.T.NoNicknamePlaceholder()
		}
		field = ansi.Truncate(field, boxWidth-8, "…")
		instruction := i18n.T.InstructionNickname()
		content = []string{
			titleStyle.Render(i18n.T.NicknamePanelTitle()),
			"",
			mutedStyle.Render(i18n.T.ContactLabel() + contactDesc),
			"",
			lipgloss.NewStyle().Foreground(accent).Render("[ " + field + " ]"),
			"",
			mutedStyle.Render(instruction),
		}
	} else {
		code := i18n.T.InvitePanelGenerating()
		if !m.loading {
			code = m.invite
			if code == "" && m.errText != "" {
				code = "—"
			}
		}
		instruction := i18n.T.InstructionInvite()
		if m.loading {
			instruction = i18n.T.InstructionWaitingAPI()
		}
		content = []string{titleStyle.Render(i18n.T.InvitePanelTitle()), "", i18n.T.InvitePanelPrompt(), "", center(titleStyle.Render(code), boxWidth-4), "", center(mutedStyle.Render(i18n.T.InvitePanelNotice()), boxWidth-4), "", mutedStyle.Render(instruction)}
	}
	if m.errText != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171")).
			Width(max(1, boxWidth-6))
		content = append(content, strings.Split(errorStyle.Render(m.errText), "\n")...)
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).
		Padding(1, 2).Width(boxWidth - 2).Render(strings.Join(content, "\n"))
	boxRows := strings.Split(box, "\n")
	start := max(0, (height-len(boxRows))/2)
	for i, line := range boxRows {
		if start+i >= height {
			break
		}
		rows[start+i] = center(line, width)
	}
	return rows
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func shortPeerID(id string) string {
	if len(id) <= 16 {
		return id
	}
	return id[:8] + "…" + id[len(id)-6:]
}

func peerLastSeen(lastSeenAt time.Time) string {
	return i18n.T.LastSeen(lastSeenAt)
}

const presenceOnlineThreshold = 30 * time.Second

func isPeerOnline(lastSeenAt time.Time) bool {
	if lastSeenAt.IsZero() {
		return false
	}
	return time.Since(lastSeenAt) < presenceOnlineThreshold
}

func frameLine(value string, width int) string {
	return borderStyle.Render("│") + fit(value, width) + borderStyle.Render("│")
}

func fit(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.ReplaceAll(value, "\ufe0f", "")
	value = ansi.Truncate(value, width, "…")
	return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
}

func center(value string, width int) string {
	clean := strings.ReplaceAll(value, "\ufe0f", "")
	return strings.Repeat(" ", max(0, (width-lipgloss.Width(clean))/2)) + value
}

func alignChatLine(value string, width int, right bool) string {
	if right {
		return strings.Repeat(" ", max(2, width-lipgloss.Width(value)-2)) + value
	}
	return "  " + value
}

func wrapChatText(value string, width int) []string {
	wrapped := lipgloss.NewStyle().Width(width).Render(value)
	lines := strings.Split(wrapped, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return lines
}

func (m model) logsView(width, height int) []string {
	rows := make([]string, height)
	lines := logger.Lines()

	countText := fmt.Sprintf("(%d logs)", len(lines))
	rows[0] = "  " + titleStyle.Render(i18n.T.LogsPanelTitle()) + "  " + mutedStyle.Render(countText)
	rows[1] = "  " + mutedStyle.Render(i18n.T.LogsPanelSubtitle())
	rows[2] = borderStyle.Render(strings.Repeat("─", width))

	bodyHeight := height - 4
	if bodyHeight <= 0 {
		return rows
	}

	if len(lines) == 0 {
		rows[4] = "  " + mutedStyle.Render(i18n.T.LogsPanelEmpty())
		return rows
	}

	wrapWidth := max(1, width-4)
	var formatted []string
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399"))

	for _, line := range lines {
		for _, w := range wrapChatText(line, wrapWidth) {
			lower := strings.ToLower(w)
			switch {
			case strings.Contains(lower, "fail"), strings.Contains(lower, "err"), strings.Contains(lower, "timeout"):
				formatted = append(formatted, "  "+errStyle.Render(w))
			case strings.Contains(lower, "ready"), strings.Contains(lower, "connected"), strings.Contains(lower, "confirmed"), strings.Contains(lower, "listening"):
				formatted = append(formatted, "  "+okStyle.Render(w))
			default:
				formatted = append(formatted, "  "+mutedStyle.Render(w))
			}
		}
	}

	maxScroll := max(0, len(formatted)-bodyHeight)
	scroll := min(m.logsScroll, maxScroll)

	end := max(0, len(formatted)-scroll)
	start := max(0, end-bodyHeight)

	for i, l := range formatted[start:end] {
		r := 3 + i
		if r >= height-1 {
			break
		}
		rows[r] = l
	}

	return rows
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const splashLogo = "   ______      ________          __ \n" +
	"  / ____/___  / ____/ /_  ____ _/ /_\n" +
	" / / __/ __ \\/ /   / __ \\/ __ `/ __/\n" +
	"/ /_/ / /_/ / /___/ / / / /_/ / /_  \n" +
	"\\____/\\____/\\____/_/ /_/\\__,_/\\__/  "

func (m model) splashView() string {
	logo := titleStyle.Render(splashLogo)
	tagline := mutedStyle.Render("P 2 P   C H A T")

	var statusLine string
	if m.splashErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Bold(true)
		statusLine = errStyle.Render("✗ "+m.splashErr.Error()) + "\n\n" + mutedStyle.Render(i18n.T.SplashQuitHint())
	} else {
		spinner := lipgloss.NewStyle().Foreground(accent).Bold(true).Render(spinnerFrames[m.splashSpinnerIdx])
		statusLine = fmt.Sprintf("%s  %s", spinner, lipgloss.NewStyle().Foreground(white).Render(m.splashStatus))
	}

	var peerLine string
	if m.localID != "" {
		peerLine = mutedStyle.Render("Peer ID: " + ansi.Truncate(m.localID, 26, "…"))
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logo,
		"",
		tagline,
		"",
		"",
		statusLine,
		"",
		peerLine,
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
