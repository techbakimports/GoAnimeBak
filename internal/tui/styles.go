package tui

import "charm.land/lipgloss/v2"

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A78BFA")).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F9FAFB")).
				Background(lipgloss.Color("#5B21B6")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B"))

	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563"))
)
