package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"github.com/alvarorichard/Goanime/internal/models"
)

// ErrEpisodeBack is returned when the user requests to go back from episode selection.
var ErrEpisodeBack = errors.New("back requested from episode list")

type episodeEntry struct {
	url    string
	number string
	label  string
	isBack bool
}

type episodeListModel struct {
	all      []episodeEntry
	filtered []episodeEntry
	filter   string
	cursor   int
	choice   *episodeEntry
	back     bool
	done     bool
	width    int
	height   int
}

func (m episodeListModel) applyFilter() episodeListModel {
	if m.filter == "" {
		cp := make([]episodeEntry, len(m.all))
		copy(cp, m.all)
		m.filtered = cp
		return m
	}
	q := strings.ToLower(m.filter)
	var out []episodeEntry
	if len(m.all) > 0 && m.all[0].isBack {
		out = append(out, m.all[0])
	}
	for _, e := range m.all {
		if !e.isBack && strings.Contains(strings.ToLower(e.label), q) {
			out = append(out, e)
		}
	}
	m.filtered = out
	return m
}

func (m episodeListModel) Init() tea.Cmd { return nil }

func (m episodeListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "pgup":
			page := m.height - 12
			if page < 5 {
				page = 5
			}
			m.cursor -= page
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "pgdown":
			page := m.height - 12
			if page < 5 {
				page = 5
			}
			m.cursor += page
			if m.cursor > len(m.filtered)-1 {
				m.cursor = len(m.filtered) - 1
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len(m.filtered) - 1
		case "enter":
			if len(m.filtered) > 0 {
				sel := m.filtered[m.cursor]
				if sel.isBack {
					m.back = true
				} else {
					m.choice = &sel
				}
				m.done = true
				return m, tea.Quit
			}
		case "esc", "ctrl+c":
			m.back = true
			m.done = true
			return m, tea.Quit
		case "backspace", "ctrl+h":
			if len(m.filter) > 0 {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.cursor = 0
				m = m.applyFilter()
			}
		default:
			s := msg.String()
			if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
				m.filter += s
				m.cursor = 0
				m = m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m episodeListModel) View() tea.View {
	w := m.width
	if w <= 0 {
		w = 80
	}
	inner := w - 6
	if inner < 40 {
		inner = 40
	}

	maxRows := m.height - 12
	if maxRows < 5 {
		maxRows = 10
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(inner)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true)

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Render(strings.Repeat("─", inner))

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FCD34D"))

	selStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#5B21B6")).
		Bold(true).
		Width(inner)

	normStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D1D5DB")).
		Width(inner)

	backStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Width(inner)

	footStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		Width(inner).
		Align(lipgloss.Center)

	total := len(m.all) - 1
	found := len(m.filtered)
	if len(m.filtered) > 0 && m.filtered[0].isBack {
		found--
	}

	var body strings.Builder
	body.WriteString(titleStyle.Render("Selecionar Episódio ") +
		countStyle.Render(fmt.Sprintf("(%d/%d)", found, total)) + "\n")
	body.WriteString(divider + "\n")
	body.WriteString(promptStyle.Render(" ❯ ") + filterStyle.Render(m.filter) +
		promptStyle.Render("█") + "\n")
	body.WriteString(divider + "\n")

	half := maxRows / 2
	start := m.cursor - half
	if start < 0 {
		start = 0
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
		start = end - maxRows
		if start < 0 {
			start = 0
		}
	}

	if len(m.filtered) == 0 {
		body.WriteString(countStyle.Render(" Nenhum resultado") + "\n")
	} else {
		for i := start; i < end; i++ {
			e := m.filtered[i]
			switch {
			case i == m.cursor && e.isBack:
				body.WriteString(selStyle.Render(truncateLine(" ▶ "+e.label, inner)))
			case i == m.cursor:
				body.WriteString(selStyle.Render(truncateLine(" ▶ "+e.label, inner)))
			case e.isBack:
				body.WriteString(backStyle.Render(truncateLine("   "+e.label, inner)))
			default:
				body.WriteString(normStyle.Render(truncateLine("   "+e.label, inner)))
			}
			body.WriteString("\n")
		}
		if len(m.filtered) > maxRows {
			body.WriteString(countStyle.Render(fmt.Sprintf(
				" ... %d de %d — continue digitando para filtrar", m.cursor+1, len(m.filtered))) + "\n")
		}
	}

	body.WriteString(divider + "\n")
	body.WriteString(footStyle.Render("↑↓ navegar  •  letras filtrar  •  ⌫ apagar  •  enter ok  •  esc voltar"))

	pad := lipgloss.NewStyle().Padding(1, 2)
	v := tea.NewView(pad.Render(borderStyle.Render(body.String())))
	v.AltScreen = true
	return v
}

// RunEpisodeList shows a full-screen Bubble Tea episode list and returns (url, number, error).
// Returns ErrEpisodeBack when the user presses esc or selects "← Voltar".
func RunEpisodeList(episodes []models.Episode) (string, string, error) {
	if len(episodes) == 0 {
		return "", "", errors.New("no episodes provided")
	}

	all := make([]episodeEntry, 0, len(episodes)+1)
	all = append(all, episodeEntry{isBack: true, label: "← Voltar"})
	for _, ep := range episodes {
		label := ep.Number
		title := ep.Title.Romaji
		if title == "" {
			title = ep.Title.English
		}
		if title != "" {
			label = fmt.Sprintf("%s — %s", ep.Number, title)
		}
		all = append(all, episodeEntry{
			url:    ep.URL,
			number: ep.Number,
			label:  label,
		})
	}

	m := episodeListModel{all: all}
	m = m.applyFilter()

	p := NewSelectorProgram(m)
	result, err := p.Run()
	if err != nil {
		return "", "", ErrEpisodeBack
	}
	if final, ok := result.(episodeListModel); ok {
		if final.back || !final.done {
			return "", "", ErrEpisodeBack
		}
		if final.choice != nil {
			return final.choice.url, final.choice.number, nil
		}
	}
	return "", "", ErrEpisodeBack
}
