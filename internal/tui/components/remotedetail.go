package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// RemoteDetailModel displays details for the selected rclone remote.
type RemoteDetailModel struct {
	viewport viewport.Model
	active   bool
	width    int
	height   int
	remote   *RemoteItem
}

// NewRemoteDetail creates a new remote detail panel.
func NewRemoteDetail(width, height int) RemoteDetailModel {
	vp := viewport.New(width, height)
	vp.SetContent("Select a remote to view details.")
	return RemoteDetailModel{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// SetActive sets whether this panel has focus.
func (m *RemoteDetailModel) SetActive(active bool) {
	m.active = active
}

// SetSize updates the panel dimensions.
func (m *RemoteDetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	m.refreshContent()
}

// SetRemote updates the displayed remote.
func (m *RemoteDetailModel) SetRemote(r *RemoteItem) {
	m.remote = r
	m.refreshContent()
}

func (m *RemoteDetailModel) refreshContent() {
	if m.remote == nil {
		m.viewport.SetContent("  Select a remote to view details.")
		return
	}
	m.viewport.SetContent(m.renderDetail())
}

func (m *RemoteDetailModel) renderDetail() string {
	r := m.remote
	var b strings.Builder

	header := theme.DetailHeaderStyle.Render("Remote: " + r.Name)
	b.WriteString(header + "\n\n")

	row := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			theme.DetailLabelStyle.Render(label+":"),
			theme.DetailValueStyle.Render(value),
		))
	}

	row("Name", r.Name)
	row("Type", r.Type)
	b.WriteString("\n")

	// Sort keys for consistent display
	keys := make([]string, 0, len(r.Details))
	for k := range r.Details {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sensitive := map[string]bool{
		"token": true, "password": true, "password2": true,
		"client_secret": true, "service_account_credentials": true,
	}

	for _, k := range keys {
		if k == "type" {
			continue
		}
		v := r.Details[k]
		if sensitive[k] {
			v = "********"
		} else if len(v) > 70 {
			v = v[:67] + "..."
		}
		row(k, v)
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s create  %s delete  %s test connection",
		theme.StatusKeyStyle.Render("C"),
		theme.StatusKeyStyle.Render("X"),
		theme.StatusKeyStyle.Render("t"),
	))

	return b.String()
}

// Update handles input for the remote detail panel.
func (m RemoteDetailModel) Update(msg tea.Msg) (RemoteDetailModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the remote detail panel.
func (m RemoteDetailModel) View() string {
	style := theme.InactivePanel
	if m.active {
		style = theme.ActivePanel
	}
	return style.
		Width(m.width).
		Height(m.height).
		Render(m.viewport.View())
}

// SelectedRemoteName returns the name of the displayed remote.
func (m RemoteDetailModel) SelectedRemoteName() string {
	if m.remote == nil {
		return ""
	}
	return m.remote.Name
}

// Placeholder for empty right panel in full-width mode.
func renderRemotePlaceholder() string {
	return lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Padding(2, 4).
		Render("Select a remote from the list to view its configuration.")
}
