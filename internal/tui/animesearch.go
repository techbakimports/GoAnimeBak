package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"github.com/alvarorichard/Goanime/internal/models"
)

// ErrAnimeBack is returned when the user goes back from anime selection.
var ErrAnimeBack = errors.New("back requested from anime selection")

type animeEntry struct {
	anime  *models.Anime
	label  string
	isBack bool
}

type animeSearchModel struct {
	all      []animeEntry
	filtered []animeEntry
	filter   string
	cursor   int
	choice   *models.Anime
	back     bool
	done     bool
	width    int
	height   int
}

func (m animeSearchModel) applyFilter() animeSearchModel {
	if m.filter == "" {
		cp := make([]animeEntry, len(m.all))
		copy(cp, m.all)
		m.filtered = cp
		return m
	}
	q := strings.ToLower(m.filter)
	var out []animeEntry
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

func (m animeSearchModel) Init() tea.Cmd { return nil }

func (m animeSearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "enter":
			if len(m.filtered) > 0 {
				sel := m.filtered[m.cursor]
				if sel.isBack {
					m.back = true
				} else {
					m.choice = sel.anime
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

func (m animeSearchModel) sourceTag(a *models.Anime) string {
	switch a.Source {
	case "AnimeFire", "Goyabu", "SuperFlix":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Render(" [PT-BR]")
	case "AllAnime":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(" [EN/JP]")
	default:
		return ""
	}
}

func (m animeSearchModel) View() tea.View {
	w := m.width
	if w <= 0 {
		w = 80
	}
	inner := w - 4
	if inner < 50 {
		inner = 50
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
	body.WriteString(titleStyle.Render("Selecionar Anime ") +
		countStyle.Render(fmt.Sprintf("(%d/%d)", found, total)) + "\n")
	body.WriteString(divider + "\n")
	body.WriteString(promptStyle.Render(" ❯ ") + filterStyle.Render(m.filter) +
		promptStyle.Render("█") + "\n")
	body.WriteString(divider + "\n")

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	if len(m.filtered) == 0 {
		body.WriteString(countStyle.Render(" Nenhum resultado") + "\n")
	} else {
		for i := start; i < end; i++ {
			e := m.filtered[i]
			switch {
			case i == m.cursor && e.isBack:
				body.WriteString(selStyle.Render(" ▶ " + e.label))
			case i == m.cursor:
				tag := m.sourceTag(e.anime)
				line := " ▶ " + e.label
				body.WriteString(selStyle.Render(line) + tag)
			case e.isBack:
				body.WriteString(backStyle.Render("   " + e.label))
			default:
				tag := m.sourceTag(e.anime)
				body.WriteString(normStyle.Render("   "+e.label) + tag)
			}
			body.WriteString("\n")
		}
		if len(m.filtered) > maxRows {
			body.WriteString(countStyle.Render(fmt.Sprintf(
				" ... %d de %d — continue digitando para filtrar", m.cursor+1, len(m.filtered))) + "\n")
		}
	}

	body.WriteString(divider + "\n")
	body.WriteString(footStyle.Render("↑↓ navegar  •  letras filtrar  •  ⌫ apagar  •  enter ok  •  esc nova busca"))

	pad := lipgloss.NewStyle().Padding(1, 2)
	v := tea.NewView(pad.Render(borderStyle.Render(body.String())))
	v.AltScreen = true
	return v
}

// RunAnimeList shows a full-screen Bubble Tea anime selector and returns the chosen anime.
// Returns ErrAnimeBack when the user goes back or presses esc.
func RunAnimeList(animes []*models.Anime) (*models.Anime, error) {
	if len(animes) == 0 {
		return nil, errors.New("no results provided")
	}

	all := make([]animeEntry, 0, len(animes)+1)
	all = append(all, animeEntry{isBack: true, label: "← Nova Busca"})
	for _, a := range animes {
		label := a.Name
		if a.Year != "" && !strings.Contains(label, "("+a.Year+")") {
			label += " (" + a.Year + ")"
		}
		all = append(all, animeEntry{anime: a, label: label})
	}

	m := animeSearchModel{all: all}
	m = m.applyFilter()

	p := NewSelectorProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, ErrAnimeBack
	}
	if final, ok := result.(animeSearchModel); ok {
		if final.back || !final.done {
			return nil, ErrAnimeBack
		}
		return final.choice, nil
	}
	return nil, ErrAnimeBack
}
