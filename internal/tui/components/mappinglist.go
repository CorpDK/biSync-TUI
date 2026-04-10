package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// MappingItem implements list.Item for the mapping list.
type MappingItem struct {
	Mapping config.Mapping
	State   *state.MappingState
}

func (i MappingItem) Title() string       { return i.Mapping.Name }
func (i MappingItem) FilterValue() string { return i.Mapping.Name }
func (i MappingItem) Description() string {
	status := statusIcon(i.State.LastStatus)
	ago := "never"
	if i.State.LastSync != nil {
		ago = timeAgo(*i.State.LastSync)
	}
	return fmt.Sprintf("%s  %s", status, ago)
}

func statusIcon(s state.SyncStatus) string {
	switch s {
	case state.StatusIdle:
		return theme.StatusIdleStyle.Render("✓ idle")
	case state.StatusSyncing:
		return theme.StatusSyncStyle.Render("◐ syncing")
	case state.StatusError:
		return theme.StatusErrorStyle.Render("✗ error")
	case state.StatusNeedsInit:
		return theme.StatusInitStyle.Render("○ init")
	default:
		return theme.StatusInitStyle.Render("? unknown")
	}
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// MappingListModel wraps a bubbles/list for mapping display.
type MappingListModel struct {
	list   list.Model
	active bool
}

// NewMappingList creates a new mapping list component.
func NewMappingList(mappings []config.Mapping, states map[string]*state.MappingState, width, height int) MappingListModel {
	items := make([]list.Item, len(mappings))
	for i, m := range mappings {
		s := states[m.Name]
		if s == nil {
			s = &state.MappingState{Name: m.Name, LastStatus: state.StatusNeedsInit}
		}
		items[i] = MappingItem{Mapping: m, State: s}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.ColorPrimary).
		BorderLeftForeground(theme.ColorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.ColorMuted).
		BorderLeftForeground(theme.ColorPrimary)

	l := list.New(items, delegate, width, height)
	l.Title = "Mappings"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = theme.DetailHeaderStyle
	l.DisableQuitKeybindings()

	return MappingListModel{list: l, active: true}
}

// SetActive sets whether this panel has focus.
func (m *MappingListModel) SetActive(active bool) {
	m.active = active
}

// SetSize updates the list dimensions.
func (m *MappingListModel) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

// Width returns the current list width.
func (m MappingListModel) Width() int { return m.list.Width() }

// Height returns the current list height.
func (m MappingListModel) Height() int { return m.list.Height() }

// SelectedMapping returns the currently selected mapping, or nil.
func (m MappingListModel) SelectedMapping() *MappingItem {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	mi := item.(MappingItem)
	return &mi
}

// UpdateState refreshes the state for all items.
func (m *MappingListModel) UpdateState(states map[string]*state.MappingState) {
	items := m.list.Items()
	for i, item := range items {
		mi := item.(MappingItem)
		if s, ok := states[mi.Mapping.Name]; ok {
			mi.State = s
			items[i] = mi
		}
	}
	m.list.SetItems(items)
}

// Update handles input for the mapping list.
func (m MappingListModel) Update(msg tea.Msg) (MappingListModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the mapping list.
func (m MappingListModel) View() string {
	style := theme.InactivePanel
	if m.active {
		style = theme.ActivePanel
	}
	return style.Render(m.list.View())
}

// ShortHelp returns keybindings for the help display.
func (m MappingListModel) ShortHelp() []key.Binding {
	return m.list.ShortHelp()
}

// FullHelp returns keybindings for the full help display.
func (m MappingListModel) FullHelp() [][]key.Binding {
	return m.list.FullHelp()
}
