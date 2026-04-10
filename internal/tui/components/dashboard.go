package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// RemoteHealth holds the connectivity test result for one remote.
type RemoteHealth struct {
	Name    string
	Healthy bool
	Error   string
}

// DashboardModel shows an overview across all mappings.
type DashboardModel struct {
	viewport viewport.Model
	width    int
	height   int

	mappings      []config.Mapping
	states        map[string]*state.MappingState
	remoteCount   int
	remoteHealth  []RemoteHealth
	healthTesting bool
}

// NewDashboard creates a new dashboard view.
func NewDashboard(width, height int) DashboardModel {
	vp := viewport.New(width, height)
	return DashboardModel{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// SetSize updates the dashboard dimensions.
func (m *DashboardModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	m.refreshContent()
}

// SetData sets the mapping/state data and refreshes.
func (m *DashboardModel) SetData(mappings []config.Mapping, states map[string]*state.MappingState, remoteCount int) {
	m.mappings = mappings
	m.states = states
	m.remoteCount = remoteCount
	m.refreshContent()
}

// SetHealthTesting marks that connectivity tests are in progress.
func (m *DashboardModel) SetHealthTesting() {
	m.healthTesting = true
	m.remoteHealth = nil
	m.refreshContent()
}

// SetRemoteHealth sets the connectivity test results and refreshes.
func (m *DashboardModel) SetRemoteHealth(results []RemoteHealth) {
	m.remoteHealth = results
	m.healthTesting = false
	m.refreshContent()
}

func (m *DashboardModel) refreshContent() {
	m.viewport.SetContent(m.render())
}

func (m *DashboardModel) render() string {
	var b strings.Builder

	header := theme.DetailHeaderStyle.Render("Dashboard")
	b.WriteString(header + "\n\n")

	row := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			lipgloss.NewStyle().Foreground(theme.ColorMuted).Width(22).Render(label+":"),
			theme.DetailValueStyle.Render(value),
		))
	}

	// Summary counts
	total := len(m.mappings)
	healthy := 0
	errored := 0
	needsInit := 0
	syncing := 0
	var lastSyncTime *time.Time

	for _, mapping := range m.mappings {
		st := m.states[mapping.Name]
		if st == nil {
			needsInit++
			continue
		}
		switch st.LastStatus {
		case state.StatusIdle:
			healthy++
		case state.StatusError:
			errored++
		case state.StatusSyncing:
			syncing++
		case state.StatusNeedsInit:
			needsInit++
		}
		if st.LastSync != nil {
			if lastSyncTime == nil || st.LastSync.After(*lastSyncTime) {
				lastSyncTime = st.LastSync
			}
		}
	}

	row("Total mappings", fmt.Sprintf("%d", total))
	row("Remotes configured", fmt.Sprintf("%d", m.remoteCount))

	b.WriteString("\n")
	sectionHeader := theme.DetailHeaderStyle.Render("Mapping Status")
	b.WriteString(sectionHeader + "\n\n")

	row("Healthy (idle)",
		theme.StatusIdleStyle.Render(fmt.Sprintf("%d", healthy)))
	row("Error",
		theme.StatusErrorStyle.Render(fmt.Sprintf("%d", errored)))
	row("Syncing",
		theme.StatusSyncStyle.Render(fmt.Sprintf("%d", syncing)))
	row("Needs initialization",
		theme.StatusInitStyle.Render(fmt.Sprintf("%d", needsInit)))

	if lastSyncTime != nil {
		b.WriteString("\n")
		row("Last sync across all", fmt.Sprintf("%s (%s)",
			lastSyncTime.Format("2006-01-02 15:04:05"),
			timeAgo(*lastSyncTime),
		))
	}

	// Per-mapping breakdown
	if total > 0 {
		b.WriteString("\n")
		sectionHeader = theme.DetailHeaderStyle.Render("Mappings")
		b.WriteString(sectionHeader + "\n\n")

		b.WriteString(fmt.Sprintf("  %-20s  %-12s  %-10s  %s\n",
			theme.StatusKeyStyle.Render("Name"),
			theme.StatusKeyStyle.Render("Status"),
			theme.StatusKeyStyle.Render("Syncs"),
			theme.StatusKeyStyle.Render("Last Sync"),
		))
		b.WriteString(fmt.Sprintf("  %-20s  %-12s  %-10s  %s\n",
			"--------------------", "------------", "----------", "---------"))

		for _, mapping := range m.mappings {
			st := m.states[mapping.Name]
			status := "unknown"
			syncs := "0"
			lastSync := "never"
			statusStyle := theme.StatusInitStyle

			if st != nil {
				status = string(st.LastStatus)
				syncs = fmt.Sprintf("%d", st.SyncCount)
				if st.LastSync != nil {
					lastSync = timeAgo(*st.LastSync)
				}
				switch st.LastStatus {
				case state.StatusIdle:
					statusStyle = theme.StatusIdleStyle
				case state.StatusError:
					statusStyle = theme.StatusErrorStyle
				case state.StatusSyncing:
					statusStyle = theme.StatusSyncStyle
				}
			}

			name := mapping.Name
			if len(name) > 20 {
				name = name[:17] + "..."
			}

			b.WriteString(fmt.Sprintf("  %-20s  %s  %-10s  %s\n",
				name,
				statusStyle.Render(fmt.Sprintf("%-12s", status)),
				syncs,
				lastSync,
			))
		}
	}

	// Remote health section
	b.WriteString("\n")
	b.WriteString(theme.DetailHeaderStyle.Render("Remote Connectivity") + "\n\n")
	if m.healthTesting {
		b.WriteString("  " + theme.StatusSyncStyle.Render("◐ Testing all remotes...") + "\n")
	} else if len(m.remoteHealth) == 0 {
		b.WriteString("  Press " + theme.StatusKeyStyle.Render("t") +
			" to test all remote connections\n")
	} else {
		for _, rh := range m.remoteHealth {
			if rh.Healthy {
				fmt.Fprintf(&b, "  %s  %s\n",
					theme.StatusIdleStyle.Render("✓"),
					rh.Name)
			} else {
				fmt.Fprintf(&b, "  %s  %s  %s\n",
					theme.StatusErrorStyle.Render("✗"),
					rh.Name,
					theme.StatusErrorStyle.Render(rh.Error))
			}
		}
	}

	return b.String()
}

// Update handles input for the dashboard.
func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the dashboard.
func (m DashboardModel) View() string {
	style := theme.ActivePanel
	return style.
		Width(m.width).
		Height(m.height).
		Render(m.viewport.View())
}
