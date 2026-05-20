package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"github.com/alvarorichard/Goanime/internal/tracking"
)

const dashboardLogo = `   ██████╗  ██████╗  █████╗ ███╗   ██╗██╗███╗   ███╗███████╗
  ██╔════╝ ██╔═══██╗██╔══██╗████╗  ██║██║████╗ ████║██╔════╝
  ██║  ███╗██║   ██║███████║██╔██╗ ██║██║██╔████╔██║█████╗
  ██║   ██║██║   ██║██╔══██║██║╚██╗██║██║██║╚██╔╝██║██╔══╝
  ╚██████╔╝╚██████╔╝██║  ██║██║ ╚████║██║██║ ╚═╝ ██║███████╗
   ╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝╚═╝     ╚═╝╚══════╝`

type dashboardMode int

const (
	modeSearch  dashboardMode = iota
	modeHistory
)

type historyEntry struct {
	title   string
	episode int
	elapsed time.Duration
}

type dashboardModel struct {
	searchQuery string
	history     []historyEntry
	cursor      int
	mode        dashboardMode
	choice      string
	done        bool
	width       int
	height      int
}

func loadHistory() []historyEntry {
	t := tracking.GetGlobalTracker()
	if t == nil {
		return nil
	}
	all, err := t.GetAllAnime()
	if err != nil || len(all) == 0 {
		return nil
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].LastUpdated.After(all[j].LastUpdated)
	})
	max := 8
	if len(all) < max {
		max = len(all)
	}
	entries := make([]historyEntry, max)
	for i := 0; i < max; i++ {
		entries[i] = historyEntry{
			title:   all[i].Title,
			episode: all[i].EpisodeNumber,
			elapsed: time.Since(all[i].LastUpdated),
		}
	}
	return entries
}

func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "agora"
	case d < time.Hour:
		return fmt.Sprintf("%dm atrás", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh atrás", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd atrás", int(d.Hours()/24))
	}
}

func (m dashboardModel) Init() tea.Cmd { return nil }

func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.done = true
			return m, tea.Quit

		case "tab":
			if m.mode == modeSearch {
				if len(m.history) > 0 {
					m.mode = modeHistory
				}
			} else {
				m.mode = modeSearch
			}

		case "up", "ctrl+p":
			if m.mode == modeHistory && m.cursor > 0 {
				m.cursor--
			}

		case "down", "ctrl+n":
			if m.mode == modeHistory && m.cursor < len(m.history)-1 {
				m.cursor++
			}

		case "enter":
			if m.mode == modeHistory && len(m.history) > 0 {
				m.choice = m.history[m.cursor].title
				m.done = true
				return m, tea.Quit
			}
			q := strings.TrimSpace(m.searchQuery)
			if len(q) >= 2 {
				m.choice = q
				m.done = true
				return m, tea.Quit
			}

		case "esc":
			if m.mode == modeHistory {
				m.mode = modeSearch
			}

		case "backspace", "ctrl+h":
			if m.mode == modeSearch && len(m.searchQuery) > 0 {
				runes := []rune(m.searchQuery)
				m.searchQuery = string(runes[:len(runes)-1])
			}

		default:
			if m.mode == modeSearch {
				s := msg.String()
				if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
					m.searchQuery += s
				}
			}
		}
	}
	return m, nil
}

func (m dashboardModel) View() tea.View {
	w := m.width
	if w <= 0 {
		w = 80
	}
	inner := w - 4
	if inner < 50 {
		inner = 50
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(inner)

	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Render(strings.Repeat("─", inner))

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	queryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FCD34D"))

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	histTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true)

	selStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#5B21B6")).
		Bold(true).
		Width(inner)

	normStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D1D5DB")).
		Width(inner)

	footStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Width(inner).
		Align(lipgloss.Center)

	var body strings.Builder

	body.WriteString(logoStyle.Render(dashboardLogo) + "\n")
	body.WriteString(divider + "\n")

	// Search row
	cursor := ""
	if m.mode == modeSearch {
		cursor = promptStyle.Render("█")
	}
	body.WriteString(promptStyle.Render(" ❯ ") + queryStyle.Render(m.searchQuery) + cursor + "\n")

	if m.mode == modeSearch {
		if len(strings.TrimSpace(m.searchQuery)) < 2 {
			body.WriteString(hintStyle.Render(" mínimo 2 caracteres — tab para histórico") + "\n")
		} else {
			body.WriteString(hintStyle.Render(" enter para buscar — tab para histórico") + "\n")
		}
	} else {
		body.WriteString(hintStyle.Render(" tab para voltar à busca — esc cancelar") + "\n")
	}

	// History section
	if len(m.history) > 0 {
		body.WriteString(divider + "\n")
		body.WriteString(histTitleStyle.Render(" Continue Assistindo") + "\n")

		for i, h := range m.history {
			ep := fmt.Sprintf("ep %d", h.episode)
			when := formatElapsed(h.elapsed)
			line := fmt.Sprintf(" %-34s  %-8s  %s", truncate(h.title, 34), ep, when)

			if m.mode == modeHistory && i == m.cursor {
				body.WriteString(selStyle.Render("▶"+line))
			} else {
				body.WriteString(normStyle.Render(" "+line))
			}
			body.WriteString("\n")
		}
	}

	body.WriteString(divider + "\n")
	if m.mode == modeSearch {
		body.WriteString(footStyle.Render("enter buscar  •  tab histórico  •  ctrl+c sair"))
	} else {
		body.WriteString(footStyle.Render("↑↓ navegar  •  enter selecionar  •  tab buscar  •  esc cancelar"))
	}

	pad := lipgloss.NewStyle().Padding(1, 2)
	v := tea.NewView(pad.Render(borderStyle.Render(body.String())))
	v.AltScreen = true
	return v
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s + strings.Repeat(" ", max-len(runes))
	}
	return string(runes[:max-1]) + "…"
}

// RunDashboard shows the home dashboard and returns the search query the user chose.
// Returns empty string if the user quits without selecting.
func RunDashboard() string {
	m := dashboardModel{
		history: loadHistory(),
	}
	p := NewSelectorProgram(m)
	result, err := p.Run()
	if err != nil {
		return ""
	}
	if final, ok := result.(dashboardModel); ok {
		return final.choice
	}
	return ""
}
