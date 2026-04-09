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

// View renders the title bar.
func (m TitleBarModel) View() string {
	title := theme.TitleStyle.Render(fmt.Sprintf(" syncctl  v%s ", m.version))

	var status string
	if m.connected {
		status = theme.ConnectedStyle.Render("● connected")
	} else {
		status = theme.DisconnectedStyle.Render("○ disconnected")
	}

	gap := m.width - lipgloss.Width(title) - lipgloss.Width(status) - 2
	if gap < 0 {
		gap = 0
	}
	spacer := lipgloss.NewStyle().Width(gap).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Top, title, spacer, status)
	return lipgloss.NewStyle().Width(m.width).Render(bar)
}
