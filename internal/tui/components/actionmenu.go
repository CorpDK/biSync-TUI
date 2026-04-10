package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// Action represents a sync action the user can perform.
type Action string

const (
	ActionSync       Action = "sync"
	ActionDryRun     Action = "dry-run"
	ActionResync     Action = "force-resync"
	ActionInit       Action = "initialize"
	ActionLogs       Action = "view-logs"
	ActionInfo       Action = "info"
	ActionRemoteSize Action = "remote-size"
	ActionHistory    Action = "view-history"
	ActionAllLogs    Action = "all-logs"
	ActionDiff       Action = "diff-preview"
	ActionEncryption Action = "encryption"
)

// ActionItem implements list.Item for the action menu.
type ActionItem struct {
	action Action
	key    string
	label  string
}

func (a ActionItem) Title() string       { return fmt.Sprintf("%s  %s", a.key, a.label) }
func (a ActionItem) Description() string { return "" }
func (a ActionItem) FilterValue() string { return a.label }

// ActionSelectedMsg is sent when the user picks an action.
type ActionSelectedMsg struct {
	MappingName string
	Action      Action
}

// ActionCancelledMsg is sent when the user cancels the menu.
type ActionCancelledMsg struct{}

// ActionMenuModel displays an overlay action selector.
type ActionMenuModel struct {
	list        list.Model
	mappingName string
	width       int
	height      int
}

// NewActionMenu creates an action menu for the given mapping.
func NewActionMenu(mappingName string, needsInit bool, width, height int) ActionMenuModel {
	var items []list.Item

	if needsInit {
		items = []list.Item{
			ActionItem{action: ActionInit, key: "b", label: "Initialize (bootstrap)"},
			ActionItem{action: ActionInfo, key: "i", label: "Info / details"},
			ActionItem{action: ActionRemoteSize, key: "z", label: "Show remote size"},
		}
	} else {
		items = []list.Item{
			ActionItem{action: ActionSync, key: "s", label: "Sync"},
			ActionItem{action: ActionDryRun, key: "d", label: "Dry run"},
			ActionItem{action: ActionResync, key: "r", label: "Force resync"},
			ActionItem{action: ActionLogs, key: "l", label: "View logs"},
			ActionItem{action: ActionHistory, key: "h", label: "View history"},
			ActionItem{action: ActionAllLogs, key: "L", label: "All logs"},
			ActionItem{action: ActionDiff, key: "D", label: "Diff preview"},
			ActionItem{action: ActionEncryption, key: "e", label: "Encryption setup"},
			ActionItem{action: ActionInfo, key: "i", label: "Info / details"},
			ActionItem{action: ActionRemoteSize, key: "z", label: "Show remote size"},
		}
	}

	menuW := min(40, width-4)
	menuH := min(len(items)+4, height-4)

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.ColorPrimary).
		BorderLeftForeground(theme.ColorPrimary)

	l := list.New(items, delegate, menuW, menuH)
	l.Title = fmt.Sprintf("Actions: %s", mappingName)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = theme.ModalTitleStyle
	l.DisableQuitKeybindings()

	return ActionMenuModel{
		list:        l,
		mappingName: mappingName,
		width:       width,
		height:      height,
	}
}

// Update handles input for the action menu.
func (m ActionMenuModel) Update(msg tea.Msg) (ActionMenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return ActionCancelledMsg{} }
		case "enter":
			item := m.list.SelectedItem()
			if item != nil {
				ai := item.(ActionItem)
				return m, func() tea.Msg {
					return ActionSelectedMsg{
						MappingName: m.mappingName,
						Action:      ai.action,
					}
				}
			}
		case "s":
			return m, m.selectByAction(ActionSync)
		case "d":
			return m, m.selectByAction(ActionDryRun)
		case "r":
			return m, m.selectByAction(ActionResync)
		case "l":
			return m, m.selectByAction(ActionLogs)
		case "i":
			return m, m.selectByAction(ActionInfo)
		case "z":
			return m, m.selectByAction(ActionRemoteSize)
		case "h":
			return m, m.selectByAction(ActionHistory)
		case "L":
			return m, m.selectByAction(ActionAllLogs)
		case "D":
			return m, m.selectByAction(ActionDiff)
		case "e":
			return m, m.selectByAction(ActionEncryption)
		case "b":
			return m, m.selectByAction(ActionInit)
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ActionMenuModel) selectByAction(action Action) tea.Cmd {
	return func() tea.Msg {
		return ActionSelectedMsg{
			MappingName: m.mappingName,
			Action:      action,
		}
	}
}

// View renders the action menu as a centered overlay.
func (m ActionMenuModel) View() string {
	menu := theme.ModalStyle.Render(m.list.View())

	// Center the menu
	menuW := lipgloss.Width(menu)
	menuH := lipgloss.Height(menu)

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
		Render(menu)
}
