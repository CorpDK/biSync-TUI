package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all application keybindings.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Select  key.Binding
	Sync    key.Binding
	SyncAll key.Binding
	DryRun  key.Binding
	Resync  key.Binding
	Logs    key.Binding
	Info    key.Binding
	Tab     key.Binding
	Quit    key.Binding
	Help    key.Binding
	Escape     key.Binding
	NewMapping key.Binding
	Diff       key.Binding
	Remotes    key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "actions"),
		),
		Sync: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sync"),
		),
		SyncAll: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "sync all"),
		),
		DryRun: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "dry-run"),
		),
		Resync: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "resync"),
		),
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs"),
		),
		Info: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "info"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch panel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		NewMapping: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new mapping"),
		),
		Diff: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "diff preview"),
		),
		Remotes: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "remotes"),
		),
	}
}

// MappingsHelp returns keybindings for the Mappings view status bar.
func (k KeyMap) MappingsHelp() []key.Binding {
	return []key.Binding{k.Sync, k.SyncAll, k.DryRun, k.Diff, k.NewMapping, k.Help, k.Quit}
}

// RemotesHelp returns keybindings for the Remotes view status bar.
func RemotesHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "create")),
		key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "delete")),
		key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "test connection")),
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

// DashboardHelp returns keybindings for the Dashboard view status bar.
func DashboardHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "mappings")),
		key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "remotes")),
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

// FullHelp returns the complete keybinding set.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Select, k.Tab},
		{k.Sync, k.SyncAll, k.DryRun, k.Resync},
		{k.Diff, k.NewMapping, k.Logs, k.Info},
		{k.Help, k.Quit},
	}
}
