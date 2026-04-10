package components

import (
	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// FormSubmittedMsg is sent when a form overlay completes successfully.
type FormSubmittedMsg struct {
	ID     string
	Values map[string]string
}

// FormCancelledMsg is sent when a form overlay is dismissed.
type FormCancelledMsg struct {
	ID string
}

// FormOverlayModel hosts a huh.Form as a centered modal overlay.
type FormOverlayModel struct {
	form   *huh.Form
	id     string
	keys   []string // ordered field keys for value extraction
	width  int
	height int
}

// NewFormOverlay creates a form overlay with the given ID and huh.Form.
// keys lists the form field keys in order, used to extract values on submit.
func NewFormOverlay(id string, form *huh.Form, keys []string, width, height int) FormOverlayModel {
	form.WithTheme(huh.ThemeCatppuccin())
	form.WithWidth(min(60, width-8))

	return FormOverlayModel{
		form:   form,
		id:     id,
		keys:   keys,
		width:  width,
		height: height,
	}
}

// Init initialises the embedded form.
func (m FormOverlayModel) Init() tea.Cmd {
	return m.form.Init()
}

// Update delegates input to the embedded form.
func (m FormOverlayModel) Update(msg tea.Msg) (FormOverlayModel, tea.Cmd) {
	// Allow Esc to cancel
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "esc" {
		return m, func() tea.Msg { return FormCancelledMsg{ID: m.id} }
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	// Check completion
	if m.form.State == huh.StateCompleted {
		values := make(map[string]string, len(m.keys))
		for _, k := range m.keys {
			values[k] = m.form.GetString(k)
		}
		return m, func() tea.Msg {
			return FormSubmittedMsg{ID: m.id, Values: values}
		}
	}

	return m, cmd
}

// View renders the form inside a styled, centered overlay.
func (m FormOverlayModel) View() string {
	content := theme.ModalStyle.Render(m.form.View())

	menuW := lipgloss.Width(content)
	menuH := lipgloss.Height(content)

	x := (m.width - menuW) / 2
	y := (m.height - menuH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return lipgloss.NewStyle().
		MarginLeft(x).
		MarginTop(y).
		Render(content)
}
