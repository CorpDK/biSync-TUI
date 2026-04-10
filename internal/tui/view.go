package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// View renders the full application.
func (m AppModel) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}
	if m.width == 0 {
		return "Loading..."
	}
	if m.showHelp {
		return m.renderHelp()
	}
	if m.formOverlay != nil {
		return m.formOverlay.View()
	}
	if m.modal != nil {
		return m.modal.View()
	}
	if m.actionMenu != nil {
		return m.actionMenu.View()
	}
	return m.renderMainLayout()
}

func (m AppModel) renderMainLayout() string {
	titleBar := m.titleBar.View()
	statusBar := m.statusBar.View()

	var content string
	switch m.viewMode {
	case ViewMappings:
		content = lipgloss.JoinHorizontal(lipgloss.Top,
			m.mappingList.View(), m.detailPanel.View())
	case ViewRemotes:
		content = lipgloss.JoinHorizontal(lipgloss.Top,
			m.remoteList.View(), m.remoteDetail.View())
	case ViewDashboard:
		content = m.dashboard.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, content, statusBar)
}

func (m *AppModel) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	contentHeight := m.height - 4 // title + status + borders
	leftWidth := int(float64(m.width) * 0.4)
	rightWidth := m.width - leftWidth

	innerLeftW := max(leftWidth-2, 10)
	innerRightW := max(rightWidth-2, 10)
	innerH := max(contentHeight-2, 5)

	m.mappingList.SetSize(innerLeftW, innerH)
	m.detailPanel.SetSize(innerRightW, innerH)
	m.remoteList.SetSize(innerLeftW, innerH)
	m.remoteDetail.SetSize(innerRightW, innerH)
	m.dashboard.SetSize(max(m.width-2, 10), max(contentHeight-2, 5))

	m.statusBar.SetWidth(m.width)
	m.titleBar.SetWidth(m.width)
}

func (m AppModel) renderHelp() string {
	title := theme.ModalTitleStyle.Render("Keybindings")

	common := []struct{ key, desc string }{
		{"1 / 2 / 3", "Switch view (Mappings / Remotes / Dashboard)"},
		{"j/k, Up/Down", "Navigate list"},
		{"Tab", "Switch panel focus"},
		{"?", "This help"},
		{"q / Ctrl+C", "Quit"},
		{"Esc", "Back / dismiss"},
	}

	var viewBindings []struct{ key, desc string }
	switch m.viewMode {
	case ViewMappings:
		viewBindings = []struct{ key, desc string }{
			{"Enter", "Open actions menu"},
			{"h/l, Left/Right", "Switch detail tab"},
			{"s", "Sync selected mapping"},
			{"S", "Sync all mappings"},
			{"d", "Dry-run (preview only)"},
			{"D", "Diff preview"},
			{"r", "Force resync"},
			{"n", "New mapping"},
			{"Enter > E", "Edit mapping"},
			{"Enter > X", "Delete mapping"},
		}
	case ViewRemotes:
		viewBindings = []struct{ key, desc string }{
			{"C", "Create remote (launches rclone config)"},
			{"X", "Delete selected remote"},
			{"t", "Test remote connection"},
		}
	case ViewDashboard:
		viewBindings = []struct{ key, desc string }{
			{"t", "Test all remote connections"},
		}
	}


	var b strings.Builder
	b.WriteString(title + "\n\n")
	for _, bind := range viewBindings {
		fmt.Fprintf(&b, "  %s  %s\n",
			theme.StatusKeyStyle.Render(fmt.Sprintf("%-20s", bind.key)),
			theme.StatusDescStyle.Render(bind.desc),
		)
	}
	b.WriteString("\n  " + theme.StatusDescStyle.Render("── Common ──") + "\n\n")
	for _, bind := range common {
		fmt.Fprintf(&b, "  %s  %s\n",
			theme.StatusKeyStyle.Render(fmt.Sprintf("%-20s", bind.key)),
			theme.StatusDescStyle.Render(bind.desc),
		)
	}
	viewNames := []string{"Mappings", "Remotes", "Dashboard"}
	b.WriteString("\n" + theme.StatusDescStyle.Render(
		fmt.Sprintf("  Showing: %s view — switch views (1/2/3) for different keys", viewNames[m.viewMode])))
	b.WriteString("\n" + theme.StatusDescStyle.Render("  Press any key to dismiss"))

	return m.centerOverlay(theme.ModalStyle.Render(b.String()))
}

func (m AppModel) centerOverlay(content string) string {
	w := lipgloss.Width(content)
	h := lipgloss.Height(content)
	x := max((m.width-w)/2, 0)
	y := max((m.height-h)/2, 0)
	return lipgloss.NewStyle().MarginLeft(x).MarginTop(y).Render(content)
}
