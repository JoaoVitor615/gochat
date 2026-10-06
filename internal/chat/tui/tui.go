// Package tui implements the interactive terminal interface for gochat.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/message"
	"github.com/JoaoVitor615/gochat/internal/chat/messaging"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
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
	ListMessages(context.Context, string, *message.Message, int) ([]storage.StoredMessage, error)
}

type MessageSender interface {
	Send(context.Context, string, string) (messaging.SendResult, error)
	AddPeerAddresses(string, []string) error
	ConnectPeer(context.Context, string) error
}

type peer struct {
	id          string
	displayName string
	addresses   []string
	lastSeenAt  time.Time
}

type panel uint8

const (
	noPanel panel = iota
	addPanel
	invitePanel
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
	code                   string
	invite                 string
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
func Run(ctx context.Context, localPeerID string, api PeerAPI, peerStore PeerStore, sender MessageSender) error {
	savedPeers, err := peerStore.ListPeers(ctx)
	if err != nil {
		return fmt.Errorf("load saved peers: %w", err)
	}
	initial := model{ctx: ctx, api: api, peerStore: peerStore, sender: sender, localID: localPeerID, active: -1}
	for _, saved := range savedPeers {
		initial.peers = append(initial.peers, peer{
			id: saved.PeerID, displayName: saved.DisplayName,
			addresses: saved.Addresses, lastSeenAt: saved.LastSeenAt,
		})
	}
	_, err = tea.NewProgram(initial, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	return err
}

func (m model) Init() tea.Cmd { return refreshTickCmd() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case refreshTick:
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
	case historyResult:
		if msg.peerID != m.activePeerID() {
			return m, nil
		}
		m.loadingChat = false
		if msg.err != nil {
			m.chatNotice = "Falha ao carregar histórico: " + msg.err.Error()
			return m, nil
		}
		m.hasOlder = msg.hasOlder
		if msg.older {
			m.messages = prependMessages(msg.messages, m.messages)
			m.scrollBack += len(msg.messages) * 3
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
			m.chatNotice = "Não foi possível enfileirar: " + msg.err.Error()
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
		}
		if msg.result.DeliveryError != nil {
			m.chatNotice = "Salva localmente; aguardando entrega. " + msg.result.DeliveryError.Error()
		} else if msg.result.Status == message.DeliveryDelivered {
			m.chatNotice = "Entrega confirmada pelo peer."
		} else {
			m.chatNotice = "Mensagem salva na fila local."
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
			m.errText = "A API não retornou o ID do peer."
			return m, nil
		}
		m.upsertPeer(peer{
			id: msg.peer.PeerID, addresses: msg.peer.Addresses,
			lastSeenAt: msg.lastSeenAt,
		})
		m.selected = m.peerIndex(msg.peer.PeerID)
		m.active = m.selected
		if msg.persistenceErr != nil {
			m.errText = fmt.Sprintf("Peer adicionado nesta sessão, mas não salvo localmente: %v", msg.persistenceErr)
			return m, nil
		}
		m.panel, m.code, m.errText = noPanel, "", ""
		if msg.connectionErr != nil {
			m.chatNotice = "Peer salvo; conexão P2P direta falhou: " + msg.connectionErr.Error()
		} else {
			m.chatNotice = "Conexão P2P direta estabelecida."
		}
	case acceptancesResult:
		m.acceptancePollInFlight = false
		if msg.err != nil {
			if m.panel == invitePanel {
				m.errText = "Falha ao consultar convites aceitos: " + msg.err.Error()
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
			m.chatNotice = "Convite aceito, mas o contato não foi salvo localmente: " + accepted.persistenceErr.Error()
		} else if accepted.connectionErr != nil {
			m.chatNotice = "Convite aceito; conexão P2P direta falhou: " + accepted.connectionErr.Error()
		} else {
			m.chatNotice = "Convite aceito; conexão P2P direta estabelecida."
		}
		return m, m.loadHistoryCmd(m.activePeerID(), nil, false)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
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
		case "esc":
			m.active = -1
		case "a":
			m.panel, m.code, m.errText = addPanel, "", ""
		case "i":
			m.panel, m.invite, m.errText, m.loading = invitePanel, "", "", true
			return m, m.createInviteCmd()
		}
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
			m.errText = "O código deve conter 9 caracteres."
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
	case "enter":
		if m.sending || strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		if m.sender == nil {
			m.chatNotice = "O serviço de envio não está disponível."
			return m, nil
		}
		peerID, text := m.activePeerID(), m.input
		m.sending, m.chatNotice = true, "Enfileirando mensagem…"
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
		m.scrollBack += step
		if m.scrollBack >= len(m.messages)*3 && m.hasOlder && !m.loadingChat && len(m.messages) > 0 {
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
	for _, current := range existing {
		if current.Envelope.ID == added.Envelope.ID {
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
			if left.CreatedAt.Before(right.CreatedAt) || (left.CreatedAt.Equal(right.CreatedAt) && left.ID <= right.ID) {
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
		m.peers[index] = updated
		return
	}
	m.peers = append(m.peers, updated)
}

func (m model) peerIndex(peerID string) int {
	for index, existing := range m.peers {
		if existing.id == peerID {
			return index
		}
	}
	return -1
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Abrindo gochat..."
	}
	if m.width < 58 || m.height < 21 {
		return "Amplie o terminal para pelo menos 58 colunas e 21 linhas."
	}

	innerWidth := m.width - 2
	sideWidth := min(29, max(22, innerWidth/3))
	mainWidth := innerWidth - sideWidth - 1
	bodyHeight := m.height - 6

	var rows []string
	rows = append(rows, borderStyle.Render("┌─ gochat "+strings.Repeat("─", m.width-11)+"┐"))
	rows = append(rows, frameLine(" "+titleStyle.Render("gochat")+"   "+mutedStyle.Render("peers"), innerWidth))
	rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", sideWidth)+"┬"+strings.Repeat("─", mainWidth)+"┤"))

	side, main := m.sidebar(sideWidth, bodyHeight), m.mainArea(mainWidth, bodyHeight)
	for i := 0; i < bodyHeight; i++ {
		rows = append(rows, borderStyle.Render("│")+fit(side[i], sideWidth)+borderStyle.Render("│")+fit(main[i], mainWidth)+borderStyle.Render("│"))
	}

	rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", sideWidth)+"┴"+strings.Repeat("─", mainWidth)+"┤"))
	footer := " ↑↓/j k navegar  ·  enter abrir  ·  a adicionar  ·  i convidar  ·  q sair"
	if m.active >= 0 {
		footer = " enter enviar  ·  pgup histórico  ·  esc voltar  ·  ctrl+u limpar"
	}
	if m.panel != noPanel {
		footer = " esc fechar  ·  convites expiram após 10 minutos"
	}
	rows = append(rows, frameLine(mutedStyle.Render(footer), innerWidth))
	rows = append(rows, borderStyle.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
	return strings.Join(rows, "\n")
}

func (m model) sidebar(width, height int) []string {
	rows := make([]string, height)
	rows[0] = "  " + titleStyle.Render("PEERS")
	for i, p := range m.peers {
		row := 2 + i*3
		if row+1 >= height-2 {
			break
		}
		name := p.displayName
		if name == "" {
			name = shortPeerID(p.id)
		}
		label := "  " + mutedStyle.Render("○") + " " + name
		if m.selected == i {
			label = selectedStyle.Render(" › ○ " + name + " ")
		} else {
			label = peerStyle.Render(label)
		}
		rows[row] = label
		rows[row+1] = mutedStyle.Render("    " + peerLastSeen(p.lastSeenAt))
	}
	rows[height-2] = mutedStyle.Render("  + Adicionar [a]")
	return rows
}

func (m model) mainArea(width, height int) []string {
	if m.panel != noPanel {
		return m.dialog(width, height)
	}
	rows := make([]string, height)
	if m.active < 0 || m.active >= len(m.peers) {
		middle := height / 2
		rows[middle-2] = center(titleStyle.Render("Nenhum peer aberto"), width)
		rows[middle] = center(mutedStyle.Render("Use [i] para criar convite ou [a] para adicionar"), width)
		return rows
	}
	p := m.peers[m.active]
	rows[0] = "  " + titleStyle.Render(shortPeerID(p.id))
	rows[1] = "  " + mutedStyle.Render(peerLastSeen(p.lastSeenAt))
	messageBottom := height - 4
	messageRows := make([]string, 0, height)
	if m.loadingChat && len(m.messages) == 0 {
		messageRows = append(messageRows, mutedStyle.Render("  Carregando histórico…"))
	} else if len(m.messages) == 0 {
		messageRows = append(messageRows, mutedStyle.Render("  Ainda não há mensagens. Escreva abaixo."))
	} else {
		for _, stored := range m.messages {
			envelope := stored.Envelope
			isLocal := envelope.SenderPeerID == m.localID
			who, status, headingStyle := "Peer", "recebida", mutedStyle
			if isLocal {
				who, status, headingStyle = "Você", deliveryLabel(stored.Status), titleStyle
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
	end := max(0, len(messageRows)-m.scrollBack)
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
		rows[height-2] += mutedStyle.Render("  enviando…")
	}
	return rows
}

func deliveryLabel(status message.DeliveryStatus) string {
	switch status {
	case message.DeliveryPending:
		return "na fila"
	case message.DeliverySent:
		return "enviado · aguardando confirmação"
	case message.DeliveryDelivered:
		return "entregue"
	default:
		return string(status)
	}
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
		instruction := "digite o código · enter adicionar"
		if m.loading {
			instruction = "consultando convite…"
		}
		content = []string{titleStyle.Render("ADICIONAR PEER"), "", "Código de convite", "", lipgloss.NewStyle().Foreground(accent).Render("[ " + field + " ]"), "", mutedStyle.Render(instruction)}
	} else {
		code := "gerando convite…"
		if !m.loading {
			code = m.invite
			if code == "" && m.errText != "" {
				code = "—"
			}
		}
		instruction := "compartilhe o código · r novo · esc fechar"
		if m.loading {
			instruction = "aguarde a resposta da API…"
		}
		content = []string{titleStyle.Render("SEU CONVITE"), "", "Compartilhe este código:", "", center(titleStyle.Render(code), boxWidth-4), "", center(mutedStyle.Render("válido por 10 minutos"), boxWidth-4), "", mutedStyle.Render(instruction)}
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
	if lastSeenAt.IsZero() {
		return "presença não verificada"
	}
	return "visto via Discovery " + lastSeenAt.Local().Format("02/01 15:04")
}

func frameLine(value string, width int) string {
	return borderStyle.Render("│") + fit(value, width) + borderStyle.Render("│")
}

func fit(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = ansi.Truncate(value, width, "…")
	return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
}

func center(value string, width int) string {
	return strings.Repeat(" ", max(0, (width-lipgloss.Width(value))/2)) + value
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
