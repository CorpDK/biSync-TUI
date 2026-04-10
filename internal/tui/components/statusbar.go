package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// StatusBarModel renders the bottom keybinding hints.
type StatusBarModel struct {
	bindings []key.Binding
	width    int
}

// NewStatusBar creates a new status bar with keybinding hints.
func NewStatusBar(bindings []key.Binding, width int) StatusBarModel {
	return StatusBarModel{bindings: bindings, width: width}
}

// SetWidth updates the bar width.
func (m *StatusBarModel) SetWidth(w int) {
	m.width = w
}

// SetBindings replaces the displayed keybinding hints.
func (m *StatusBarModel) SetBindings(bindings []key.Binding) {
	m.bindings = bindings
}

// View renders the status bar.
func (m StatusBarModel) View() string {
	var parts []string

	for _, b := range m.bindings {
		keys := b.Help().Key
		desc := b.Help().Desc
		part := theme.StatusKeyStyle.Render(keys) + " " + theme.StatusDescStyle.Render(desc)
		parts = append(parts, part)
	}

	bar := strings.Join(parts, "  ")
	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Foreground(theme.ColorMuted).
		Render(bar)
}
