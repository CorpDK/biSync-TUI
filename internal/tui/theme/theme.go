package theme

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	ColorPrimary = lipgloss.Color("#7C3AED")
	ColorSuccess = lipgloss.Color("#10B981")
	ColorWarning = lipgloss.Color("#F59E0B")
	ColorError   = lipgloss.Color("#EF4444")
	ColorMuted   = lipgloss.Color("#6B7280")
	ColorWhite   = lipgloss.Color("#F9FAFB")
	ColorBg      = lipgloss.Color("#1F2937")
)

// Panel styles
var (
	PanelBorder   = lipgloss.RoundedBorder()
	ActivePanel   = lipgloss.NewStyle().Border(PanelBorder).BorderForeground(ColorPrimary)
	InactivePanel = lipgloss.NewStyle().Border(PanelBorder).BorderForeground(ColorMuted)
)

// Title bar
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 1)
	ConnectedStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)
	DisconnectedStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)
)

// Status bar
var (
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)
	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)
	StatusDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Mapping list
var (
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)
	NormalItemStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)
	StatusIdleStyle  = lipgloss.NewStyle().Foreground(ColorSuccess)
	StatusSyncStyle  = lipgloss.NewStyle().Foreground(ColorWarning)
	StatusErrorStyle = lipgloss.NewStyle().Foreground(ColorError)
	StatusInitStyle  = lipgloss.NewStyle().Foreground(ColorMuted)
)

// Detail panel
var (
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Width(10)
	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ColorWhite)
	DetailHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Underline(true)
)

// Modal
var (
	ModalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)
	ModalTitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)
)
