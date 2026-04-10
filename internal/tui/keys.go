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

// ShortHelp returns a minimal set of keybindings for the status bar.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Sync, k.SyncAll, k.DryRun, k.Diff, k.NewMapping, k.Remotes, k.Help, k.Quit}
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
