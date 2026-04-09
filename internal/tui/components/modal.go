package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// ModalConfirmMsg is sent when the user confirms a modal action.
type ModalConfirmMsg struct {
	ID string
}

// ModalCancelMsg is sent when the user cancels a modal.
type ModalCancelMsg struct {
	ID string
}

// ModalModel displays a confirmation dialog.
type ModalModel struct {
	ID     string
	Title  string
	Body   string
	width  int
	height int
}

// NewModal creates a confirmation dialog.
func NewModal(id, title, body string, width, height int) ModalModel {
	return ModalModel{
		ID:     id,
		Title:  title,
		Body:   body,
		width:  width,
		height: height,
	}
}

// Update handles input for the modal.
func (m ModalModel) Update(msg tea.Msg) (ModalModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			return m, func() tea.Msg { return ModalConfirmMsg{ID: m.ID} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return ModalCancelMsg{ID: m.ID} }
		}
	}
	return m, nil
}

// View renders the modal as a centered overlay.
func (m ModalModel) View() string {
	var b strings.Builder
	b.WriteString(theme.ModalTitleStyle.Render(m.Title))
	b.WriteString("\n\n")
	b.WriteString(m.Body)
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s to confirm, %s to cancel",
		theme.StatusKeyStyle.Render("y"),
		theme.StatusKeyStyle.Render("n/esc"),
	))

	content := theme.ModalStyle.Render(b.String())

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
