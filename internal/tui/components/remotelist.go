package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// RemoteItem implements list.Item for the remote list.
type RemoteItem struct {
	Name    string
	Type    string
	Details map[string]string
}

func (i RemoteItem) Title() string       { return i.Name }
func (i RemoteItem) FilterValue() string { return i.Name }
func (i RemoteItem) Description() string {
	return theme.StatusDescStyle.Render(i.Type)
}

// RemoteListModel wraps a bubbles/list for remote display.
type RemoteListModel struct {
	list   list.Model
	active bool
}

// NewRemoteList creates a new remote list component.
func NewRemoteList(remotes []RemoteDisplayInfo, width, height int) RemoteListModel {
	items := make([]list.Item, len(remotes))
	for i, r := range remotes {
		items[i] = RemoteItem{Name: r.Name, Type: r.Type, Details: r.Details}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.ColorPrimary).
		BorderLeftForeground(theme.ColorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.ColorMuted).
		BorderLeftForeground(theme.ColorPrimary)

	l := list.New(items, delegate, width, height)
	l.Title = "Remotes"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = theme.DetailHeaderStyle
	l.DisableQuitKeybindings()

	return RemoteListModel{list: l, active: true}
}

// SetActive sets whether this panel has focus.
func (m *RemoteListModel) SetActive(active bool) {
	m.active = active
}

// SetSize updates the list dimensions.
func (m *RemoteListModel) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

// Width returns the current list width.
func (m RemoteListModel) Width() int { return m.list.Width() }

// Height returns the current list height.
func (m RemoteListModel) Height() int { return m.list.Height() }

// SelectedRemote returns the currently selected remote, or nil.
func (m RemoteListModel) SelectedRemote() *RemoteItem {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	ri := item.(RemoteItem)
	return &ri
}

// SetItems replaces the remote list items.
func (m *RemoteListModel) SetItems(remotes []RemoteDisplayInfo) {
	items := make([]list.Item, len(remotes))
	for i, r := range remotes {
		items[i] = RemoteItem{Name: r.Name, Type: r.Type, Details: r.Details}
	}
	m.list.SetItems(items)
}

// Update handles input for the remote list.
func (m RemoteListModel) Update(msg tea.Msg) (RemoteListModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the remote list.
func (m RemoteListModel) View() string {
	style := theme.InactivePanel
	if m.active {
		style = theme.ActivePanel
	}
	return style.
		Width(m.list.Width()).
		Height(m.list.Height()).
		Render(m.list.View())
}

// ItemCount returns the number of items.
func (m RemoteListModel) ItemCount() int {
	return len(m.list.Items())
}

// StatusHint returns context-aware hints for the status bar.
func (m RemoteListModel) StatusHint() string {
	if m.ItemCount() == 0 {
		return fmt.Sprintf("No remotes configured. Press %s to create one.",
			theme.StatusKeyStyle.Render("C"))
	}
	return ""
}
