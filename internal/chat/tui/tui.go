// Package tui contains the terminal-only chat mock. No peer or message data is
// loaded from the network yet.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type message struct {
	author string
	time   string
	text   string
	owned  bool
}

type peer struct {
	name     string
	code     string
	online   bool
	unread   string
	messages []message
}

var peers = []peer{
	{
		name: "Maria", code: "12D-7AF-P92", online: true, unread: "2 mensagens",
		messages: []message{
			{author: "Maria", time: "14:28", text: "Oi! Conseguiu testar o projeto?"},
			{author: "Você", time: "14:30", text: "Sim, está funcionando!", owned: true},
			{author: "Maria", time: "14:31", text: "Perfeito 👌"},
		},
	},
	{
		name: "João", code: "7AA-45C-N21", online: false,
		messages: []message{
			{author: "João", time: "Ontem", text: "Vamos conversar depois?"},
			{author: "Você", time: "Ontem", text: "Claro!", owned: true},
		},
	},
	{
		name: "Pedro", code: "90K-2BB-L83", online: true,
		messages: []message{
			{author: "Pedro", time: "13:05", text: "Oi! Tudo bem por aí?"},
		},
	},
}

type panel uint8

const (
	noPanel panel = iota
	addPanel
	invitePanel
)

type model struct {
	width    int
	height   int
	selected int
	active   int
	panel    panel
	code     string
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

// Run opens the visual mock in the terminal alternate screen.
func Run() error {
	_, err := tea.NewProgram(model{active: -1}, tea.WithAltScreen()).Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.panel == addPanel {
			switch msg.String() {
			case "esc":
				m.panel, m.code = noPanel, ""
			case "backspace", "ctrl+h":
				if len(m.code) > 0 {
					m.code = m.code[:len(m.code)-1]
				}
			default:
				if msg.Type == tea.KeyRunes {
					for _, char := range strings.ToUpper(string(msg.Runes)) {
						if len(m.code) < 24 && ((char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-') {
							m.code += string(char)
						}
					}
				}
			}
			return m, nil
		}
		if m.panel == invitePanel {
			if msg.String() == "esc" {
				m.panel = noPanel
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
			if m.selected < len(peers)-1 {
				m.selected++
			}
		case "enter":
			m.active = m.selected
		case "esc":
			m.active = -1
		case "a":
			m.panel, m.code = addPanel, ""
		case "i":
			m.panel = invitePanel
		}
	}
	return m, nil
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
	rows = append(rows, frameLine(" "+titleStyle.Render("gochat")+"   "+mutedStyle.Render("conversas"), innerWidth))
	rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", sideWidth)+"┬"+strings.Repeat("─", mainWidth)+"┤"))

	side := m.sidebar(sideWidth, bodyHeight)
	main := m.mainArea(mainWidth, bodyHeight)
	for i := 0; i < bodyHeight; i++ {
		rows = append(rows, borderStyle.Render("│")+fit(side[i], sideWidth)+borderStyle.Render("│")+fit(main[i], mainWidth)+borderStyle.Render("│"))
	}

	rows = append(rows, borderStyle.Render("├"+strings.Repeat("─", sideWidth)+"┴"+strings.Repeat("─", mainWidth)+"┤"))
	footer := " ↑↓/j k navegar  ·  enter abrir  ·  a adicionar  ·  i convite  ·  q sair"
	if m.panel != noPanel {
		footer = " esc fechar  ·  interface demonstrativa"
	}
	rows = append(rows, frameLine(mutedStyle.Render(footer), innerWidth))
	rows = append(rows, borderStyle.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
	return strings.Join(rows, "\n")
}

func (m model) sidebar(width, height int) []string {
	rows := make([]string, height)
	rows[0] = "  " + titleStyle.Render("PEERS")
	for i, peer := range peers {
		row := 2 + i*3
		if row+1 >= height-2 {
			break
		}
		marker := onlineStyle.Render("●")
		if !peer.online {
			marker = mutedStyle.Render("○")
		}
		label := "  " + marker + " " + peer.name
		if m.selected == i {
			label = selectedStyle.Render(" › " + symbol(peer.online) + " " + peer.name + " ")
		} else {
			label = peerStyle.Render(label)
		}
		rows[row] = label
		detail := peer.unread
		if detail == "" {
			if peer.online {
				detail = "online"
			} else {
				detail = "offline"
			}
		}
		rows[row+1] = mutedStyle.Render("    " + detail)
	}
	rows[height-2] = mutedStyle.Render("  + Adicionar peer  [a]")
	return rows
}

func (m model) mainArea(width, height int) []string {
	if m.panel != noPanel {
		return m.dialog(width, height)
	}

	rows := make([]string, height)
	if m.active < 0 {
		middle := height / 2
		rows[middle-1] = center(titleStyle.Render("Selecione um peer"), width)
		rows[middle+1] = center(mutedStyle.Render("↑↓ para navegar · enter para abrir"), width)
		return rows
	}

	peer := peers[m.active]
	rows[0] = "  " + titleStyle.Render(peer.name)
	status := "offline"
	if peer.online {
		status = onlineStyle.Render("● online")
	}
	rows[1] = "  " + status + mutedStyle.Render("  ·  "+peer.code)
	rows[2] = borderStyle.Render(strings.Repeat("─", width))
	rows[height-3] = borderStyle.Render(strings.Repeat("─", width))
	rows[height-2] = mutedStyle.Render("  Digite uma mensagem...  (mock)")

	row := 4
	for _, message := range peer.messages {
		if row+2 >= height-3 {
			break
		}
		heading := message.author + "  " + message.time
		body := message.text
		if message.owned {
			rows[row] = right(mutedStyle.Render(heading), width, 2)
			rows[row+1] = right(peerStyle.Render(body), width, 2)
		} else {
			rows[row] = "  " + mutedStyle.Render(heading)
			rows[row+1] = "  " + peerStyle.Render(body)
		}
		row += 3
	}
	return rows
}

func (m model) dialog(width, height int) []string {
	rows := make([]string, height)
	boxWidth := min(43, width-4)
	var content []string
	if m.panel == addPanel {
		field := m.code
		if field == "" {
			field = "88B-11A-KXL"
		}
		field = ansi.Truncate(field, boxWidth-8, "…")
		instructions := "enter adicionar · esc fechar"
		if width < 45 {
			instructions = "esc fechar · mock"
		}
		content = []string{
			titleStyle.Render("ADICIONAR PEER"), "", "Código do peer", "",
			lipgloss.NewStyle().Foreground(accent).Render("[ " + field + " ]"), "",
			mutedStyle.Render(instructions),
		}
	} else {
		instructions := "c copiar · r gerar outro · esc fechar"
		if width < 45 {
			instructions = "esc fechar · mock"
		}
		content = []string{
			titleStyle.Render("SEU CONVITE"), "", "Compartilhe este código:", "",
			center(titleStyle.Render("3FK-92Q-M1Z"), boxWidth-4), "",
			center(mutedStyle.Render("Expira em 09:42"), boxWidth-4), "",
			mutedStyle.Render(instructions),
		}
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

func symbol(online bool) string {
	if online {
		return "●"
	}
	return "○"
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

func right(value string, width, padding int) string {
	return strings.Repeat(" ", max(0, width-padding-lipgloss.Width(value))) + value
}
