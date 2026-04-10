package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// TitleBarModel renders the top bar with app title and connectivity status.
type TitleBarModel struct {
	version   string
	connected bool
	width     int
	viewMode  int // 0=Mappings, 1=Remotes, 2=Dashboard
}

// NewTitleBar creates a new title bar.
func NewTitleBar(version string, width int) TitleBarModel {
	return TitleBarModel{
		version:   version,
		connected: false,
		width:     width,
	}
}

// SetConnected updates the connectivity indicator.
func (m *TitleBarModel) SetConnected(connected bool) {
	m.connected = connected
}

// SetWidth updates the bar width.
func (m *TitleBarModel) SetWidth(w int) {
	m.width = w
}

// SetViewMode updates the active view indicator.
func (m *TitleBarModel) SetViewMode(mode int) {
	m.viewMode = mode
}

// View renders the title bar.
func (m TitleBarModel) View() string {
	title := theme.TitleStyle.Render(fmt.Sprintf(" syncctl  v%s ", m.version))

	// View tabs
	tabs := m.renderViewTabs()

	var status string
	if m.connected {
		status = theme.ConnectedStyle.Render("● connected")
	} else {
		status = theme.DisconnectedStyle.Render("○ disconnected")
	}

	gap := m.width - lipgloss.Width(title) - lipgloss.Width(tabs) - lipgloss.Width(status) - 2
	if gap < 0 {
		gap = 0
	}
	leftGap := gap / 2
	rightGap := gap - leftGap

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		lipgloss.NewStyle().Width(leftGap).Render(""),
		tabs,
		lipgloss.NewStyle().Width(rightGap).Render(""),
		status,
	)
	return lipgloss.NewStyle().Width(m.width).Render(bar)
}

func (m TitleBarModel) renderViewTabs() string {
	views := []struct {
		key   string
		label string
	}{
		{"1", "Mappings"},
		{"2", "Remotes"},
		{"3", "Dashboard"},
	}

	activeTab := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		Padding(0, 1)
	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Padding(0, 1)

	var parts []string
	for i, v := range views {
		style := inactiveTab
		if i == m.viewMode {
			style = activeTab
		}
		parts = append(parts, style.Render(fmt.Sprintf("[%s] %s", v.key, v.label)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
