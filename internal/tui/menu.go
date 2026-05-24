package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
)

// MenuItem maps a display label to the value returned by RunMenu.
type MenuItem struct {
	Label string
	Value string
}

type menuModel struct {
	title  string
	items  []MenuItem
	cursor int
	choice string
	done   bool
	width  int
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j", "ctrl+n":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.choice = m.items[m.cursor].Value
			m.done = true
			return m, tea.Quit
		case "esc":
			m.choice = "back"
			m.done = true
			return m, tea.Quit
		case "ctrl+c":
			m.choice = "q"
			m.done = true
			return m, tea.Quit
		default:
			s := msg.String()
			if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
				idx := int(s[0]-'1')
				if idx < len(m.items) {
					m.cursor = idx
					m.choice = m.items[idx].Value
					m.done = true
					return m, tea.Quit
				}
			}
		}
	}
	return m, nil
}

func (m menuModel) View() tea.View {
	w := m.width
	if w <= 0 {
		w = 60
	}
	inner := w - 6
	if inner < 40 {
		inner = 40
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(inner)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A78BFA")).
		Bold(true).
		Width(inner).
		Align(lipgloss.Center)

	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Render(strings.Repeat("─", inner))

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
	body.WriteString(titleStyle.Render(m.title) + "\n")
	body.WriteString(divider + "\n")

	for i, item := range m.items {
		num := fmt.Sprintf("%d. ", i+1)
		label := num + item.Label
		if i == m.cursor {
			body.WriteString(selStyle.Render(" ▶ " + label))
		} else {
			body.WriteString(normStyle.Render("   " + label))
		}
		body.WriteString("\n")
	}

	body.WriteString(divider + "\n")
	body.WriteString(footStyle.Render("↑↓ navegar  •  enter selecionar  •  esc voltar"))

	box := borderStyle.Render(body.String())

	pad := lipgloss.NewStyle().Padding(1, 2)
	v := tea.NewView(pad.Render(box))
	v.AltScreen = true
	return v
}

// RunMenu shows a full-screen styled Bubble Tea selection menu and returns the chosen value.
// Returns "q" on ctrl+c, "back" on esc.
func RunMenu(title string, items []MenuItem) string {
	m := menuModel{title: title, items: items}
	p := NewSelectorProgram(m)
	result, err := p.Run()
	if err != nil {
		return "q"
	}
	if final, ok := result.(menuModel); ok {
		return final.choice
	}
	return "q"
}
