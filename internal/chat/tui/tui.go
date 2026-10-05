// Package tui implements the interactive terminal interface for gochat.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JoaoVitor615/gochat/internal/chat/client"
	"github.com/JoaoVitor615/gochat/internal/chat/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// PeerAPI is the Discovery API functionality needed by the TUI.
type PeerAPI interface {
	CreateInvite(context.Context, string) (*client.CreateInviteResponse, error)
	AddPeer(context.Context, string) (*client.AddPeerResponse, error)
}

// PeerStore is the local contact persistence needed by the TUI.
type PeerStore interface {
	ListPeers(context.Context) ([]storage.Peer, error)
	SavePeer(context.Context, storage.Peer) error
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
	ctx       context.Context
	api       PeerAPI
	peerStore PeerStore
	localID   string
	width     int
	height    int
	selected  int
	active    int
	panel     panel
	code      string
	invite    string
	loading   bool
	errText   string
	peers     []peer
}

type inviteResult struct {
	code string
	err  error
}

type addPeerResult struct {
	peer           *client.AddPeerResponse
	err            error
	persistenceErr error
	lastSeenAt     time.Time
}

var (
	accent = lipgloss.Color("#7DD3FC")
	muted  = lipgloss.Color("#94A3B8")
	green  = lipgloss.Color("#4ADE80")
	white  = lipgloss.Color("#F8FAFC")

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(accent)
	mutedStyle    = lipgloss.NewStyle().Foreground(muted)
	peerStyle     = lipgloss.NewStyle().Foreground(white)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0F172A")).Background(accent)
	onlineStyle   = lipgloss.NewStyle().Foreground(green)
	borderStyle   = lipgloss.NewStyle().Foreground(muted)
)

// Run opens the chat TUI and keeps peer operations tied to the app context.
func Run(ctx context.Context, localPeerID string, api PeerAPI, peerStore PeerStore) error {
	savedPeers, err := peerStore.ListPeers(ctx)
	if err != nil {
		return fmt.Errorf("load saved peers: %w", err)
	}
	initial := model{ctx: ctx, api: api, peerStore: peerStore, localID: localPeerID, active: -1}
	for _, saved := range savedPeers {
		initial.peers = append(initial.peers, peer{
			id: saved.PeerID, displayName: saved.DisplayName,
			addresses: saved.Addresses, lastSeenAt: saved.LastSeenAt,
		})
	}
	_, err = tea.NewProgram(initial, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case inviteResult:
		m.loading = false
		m.invite, m.errText = msg.code, errorText(msg.err)
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

func (m model) createInviteCmd() tea.Cmd {
	return func() tea.Msg {
		response, err := m.api.CreateInvite(m.ctx, m.localID)
		if err != nil {
			return inviteResult{err: err}
		}
		if response == nil {
			return inviteResult{err: fmt.Errorf("a API não retornou um código de convite")}
		}
		return inviteResult{code: response.Code}
	}
}

func (m model) addPeerCmd(code string) tea.Cmd {
	return func() tea.Msg {
		response, err := m.api.AddPeer(m.ctx, code)
		result := addPeerResult{peer: response, err: err}
		if err != nil || response == nil || response.PeerID == "" {
			return result
		}
		result.lastSeenAt = time.Now().UTC()
		result.persistenceErr = m.peerStore.SavePeer(m.ctx, storage.Peer{
			PeerID: response.PeerID, AddedAt: result.lastSeenAt,
			LastSeenAt: result.lastSeenAt, Addresses: response.Addresses,
		})
		return result
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
	rows[2] = borderStyle.Render(strings.Repeat("─", width))
	rows[4] = "  " + mutedStyle.Render("Peer ID")
	rows[5] = "  " + peerStyle.Render(ansi.Truncate(p.id, width-4, "…"))
	if len(p.addresses) > 0 {
		rows[7] = "  " + mutedStyle.Render("Endereços anunciados")
		for i, address := range p.addresses {
			row := 8 + i
			if row >= height-2 {
				break
			}
			rows[row] = "  " + peerStyle.Render(ansi.Truncate(address, width-4, "…"))
		}
	}
	rows[height-2] = mutedStyle.Render("  Peer salvo localmente neste dispositivo.")
	return rows
}

func (m model) dialog(width, height int) []string {
	rows := make([]string, height)
	boxWidth := min(43, width-4)
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
		content = append(content, lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Render(ansi.Truncate(m.errText, boxWidth-6, "…")))
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
