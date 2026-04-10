package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/logs"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// RemoteAboutInfo holds storage space data for display.
type RemoteAboutInfo struct {
	Total   int64
	Used    int64
	Free    int64
	Trashed int64
}

// DetailMode selects what the detail panel shows.
type DetailMode int

const (
	DetailInfo DetailMode = iota
	DetailLogs
	DetailDiff
	DetailHistory
	DetailAllLogs
)

// DetailPanelModel displays info/logs for the selected mapping.
type DetailPanelModel struct {
	viewport viewport.Model
	mode     DetailMode
	active   bool
	width    int
	height   int

	// Sub-components
	diffView DiffViewModel

	// Current content
	mapping       *config.Mapping
	state         *state.MappingState
	logLines      []string
	history       []state.HistoryRecord
	allLogEntries []logs.LogEntry
	remoteAbout   *RemoteAboutInfo
	autoScroll    bool
}

// NewDetailPanel creates a new detail panel.
func NewDetailPanel(width, height int) DetailPanelModel {
	vp := viewport.New(width, height)
	vp.SetContent("Select a mapping to view details.")

	return DetailPanelModel{
		viewport:   vp,
		mode:       DetailInfo,
		width:      width,
		height:     height,
		autoScroll: true,
	}
}

// SetActive sets whether this panel has focus.
func (m *DetailPanelModel) SetActive(active bool) {
	m.active = active
}

// SetSize updates the panel dimensions.
func (m *DetailPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h
	m.refreshContent()
}

// SetMapping updates the displayed mapping.
func (m *DetailPanelModel) SetMapping(mapping *config.Mapping, st *state.MappingState) {
	m.mapping = mapping
	m.state = st
	m.mode = DetailInfo
	m.logLines = nil
	m.refreshContent()
}

// SetMode switches the display mode.
func (m *DetailPanelModel) SetMode(mode DetailMode) {
	m.mode = mode
	m.refreshContent()
}

// AppendLog adds a line to the log view.
func (m *DetailPanelModel) AppendLog(line string) {
	m.logLines = append(m.logLines, line)
	if m.mode == DetailLogs {
		m.refreshContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
	}
}

// ClearLogs resets the log buffer.
func (m *DetailPanelModel) ClearLogs() {
	m.logLines = nil
	if m.mode == DetailLogs {
		m.refreshContent()
	}
}

// UpdateState refreshes the state display.
func (m *DetailPanelModel) UpdateState(st *state.MappingState) {
	m.state = st
	if m.mode == DetailInfo {
		m.refreshContent()
	}
}

// SetHistory sets the sync history records for display.
func (m *DetailPanelModel) SetHistory(records []state.HistoryRecord) {
	m.history = records
	if m.mode == DetailHistory {
		m.refreshContent()
	}
}

// SetAllLogs sets aggregated log entries for display.
func (m *DetailPanelModel) SetAllLogs(entries []logs.LogEntry) {
	m.allLogEntries = entries
	if m.mode == DetailAllLogs {
		m.refreshContent()
	}
}

// SetRemoteAbout sets storage space info for display in the info panel.
func (m *DetailPanelModel) SetRemoteAbout(about *RemoteAboutInfo) {
	m.remoteAbout = about
	if m.mode == DetailInfo {
		m.refreshContent()
	}
}

// SetDiffEntries updates the diff preview entries.
func (m *DetailPanelModel) SetDiffEntries(entries []bisync.DiffEntry) {
	m.diffView.SetEntries(entries)
	if m.mode == DetailDiff {
		m.refreshContent()
	}
}

func (m *DetailPanelModel) refreshContent() {
	switch m.mode {
	case DetailInfo:
		m.viewport.SetContent(m.renderInfo())
	case DetailLogs:
		m.viewport.SetContent(m.renderLogs())
	case DetailDiff:
		m.viewport.SetContent(m.diffView.View())
	case DetailHistory:
		m.viewport.SetContent(m.renderHistory())
	case DetailAllLogs:
		m.viewport.SetContent(m.renderAllLogs())
	}
}

func (m *DetailPanelModel) renderInfo() string {
	if m.mapping == nil {
		return "  Select a mapping to view details."
	}

	var b strings.Builder
	header := theme.DetailHeaderStyle.Render("Mapping Details")
	b.WriteString(header + "\n\n")

	row := func(label, value string) {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			theme.DetailLabelStyle.Render(label+":"),
			theme.DetailValueStyle.Render(value),
		))
	}

	row("Name", m.mapping.Name)

	// Multi-remote aware labels
	if config.IsRemotePath(m.mapping.Local) {
		row("Path 1", m.mapping.Local+" (remote)")
	} else {
		row("Local", m.mapping.Local)
	}
	row("Remote", m.mapping.Remote)

	// Per-mapping config
	if m.mapping.FiltersFile != "" {
		row("Filters", m.mapping.FiltersFile)
	}
	if m.mapping.BandwidthLimit != "" {
		row("BW Limit", m.mapping.BandwidthLimit)
	}
	if m.mapping.ConflictResolve != "" {
		row("Conflicts", m.mapping.ConflictResolve)
	}
	if m.mapping.BackupEnabled {
		row("Backup", fmt.Sprintf("enabled (%d day retention)", m.mapping.BackupRetention))
	}
	if m.mapping.Encryption.Enabled {
		row("Encryption", "enabled")
		row("Crypt", m.mapping.Encryption.CryptRemote)
	}

	// Remote storage info
	if m.remoteAbout != nil {
		b.WriteString("\n")
		totalGiB := float64(m.remoteAbout.Total) / (1024 * 1024 * 1024)
		usedGiB := float64(m.remoteAbout.Used) / (1024 * 1024 * 1024)
		pct := float64(0)
		if m.remoteAbout.Total > 0 {
			pct = float64(m.remoteAbout.Used) / float64(m.remoteAbout.Total) * 100
		}
		row("Storage", fmt.Sprintf("%.1f GiB / %.1f GiB (%.0f%% used)", usedGiB, totalGiB, pct))
		if m.remoteAbout.Trashed > 0 {
			trashedMiB := float64(m.remoteAbout.Trashed) / (1024 * 1024)
			row("Trashed", fmt.Sprintf("%.1f MiB", trashedMiB))
		}
	}

	if m.state != nil {
		b.WriteString("\n")
		row("Status", string(m.state.LastStatus))
		if m.state.LastSync != nil {
			row("Last sync", fmt.Sprintf("%s (%s)",
				m.state.LastSync.Format("2006-01-02 15:04:05"),
				timeAgo(*m.state.LastSync),
			))
		} else {
			row("Last sync", "never")
		}
		row("Syncs", fmt.Sprintf("%d", m.state.SyncCount))
		if m.state.LastDuration != "" {
			row("Duration", m.state.LastDuration)
		}
		if m.state.LastError != "" {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  %s\n",
				theme.StatusErrorStyle.Render("Error: "+m.state.LastError),
			))
		}
	}

	return b.String()
}

func (m *DetailPanelModel) renderLogs() string {
	if len(m.logLines) == 0 {
		return "  No log output yet."
	}
	var b strings.Builder
	header := theme.DetailHeaderStyle.Render("Sync Output")
	b.WriteString(header + "\n\n")
	for _, line := range m.logLines {
		b.WriteString("  " + line + "\n")
	}
	return b.String()
}

func (m *DetailPanelModel) renderHistory() string {
	if len(m.history) == 0 {
		return "  No sync history yet."
	}
	var b strings.Builder
	header := theme.DetailHeaderStyle.Render("Sync History")
	b.WriteString(header + "\n\n")
	b.WriteString(fmt.Sprintf("  %-20s  %-8s  %-10s  %-6s  %s\n",
		"Timestamp", "Status", "Duration", "Files", "Bytes"))
	b.WriteString(fmt.Sprintf("  %-20s  %-8s  %-10s  %-6s  %s\n",
		"--------------------", "--------", "----------", "------", "-----"))

	// Show most recent first
	for i := len(m.history) - 1; i >= 0; i-- {
		r := m.history[i]
		ts := r.Timestamp.Format("2006-01-02 15:04:05")
		dur := r.Duration.Truncate(time.Second).String()
		style := theme.StatusIdleStyle
		if r.Status == "error" {
			style = theme.StatusErrorStyle
		}
		b.WriteString(fmt.Sprintf("  %-20s  %s  %-10s  %-6d  %d\n",
			ts, style.Render(fmt.Sprintf("%-8s", r.Status)), dur, r.FilesTransferred, r.BytesTransferred))
	}
	return b.String()
}

func (m *DetailPanelModel) renderAllLogs() string {
	if len(m.allLogEntries) == 0 {
		return "  No aggregated logs yet."
	}
	var b strings.Builder
	header := theme.DetailHeaderStyle.Render("All Logs")
	b.WriteString(header + "\n\n")

	for _, e := range m.allLogEntries {
		ts := e.Timestamp.Format("15:04:05")
		levelStyle := theme.StatusInitStyle
		switch e.Level {
		case "ERROR":
			levelStyle = theme.StatusErrorStyle
		case "WARN", "NOTICE":
			levelStyle = theme.StatusSyncStyle
		}
		b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
			theme.DetailLabelStyle.Render(ts),
			theme.StatusSyncStyle.Render(fmt.Sprintf("[%-10s]", e.MappingName)),
			levelStyle.Render(fmt.Sprintf("[%s]", e.Level)),
			e.Message,
		))
	}
	return b.String()
}

// Update handles input for the detail panel.
func (m DetailPanelModel) Update(msg tea.Msg) (DetailPanelModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the detail panel.
func (m DetailPanelModel) View() string {
	style := theme.InactivePanel
	if m.active {
		style = theme.ActivePanel
	}

	// Mode indicator tabs
	tabs := m.renderTabs()

	content := lipgloss.JoinVertical(lipgloss.Left,
		tabs,
		m.viewport.View(),
	)

	return style.Render(content)
}

func (m DetailPanelModel) renderTabs() string {
	activeTab := lipgloss.NewStyle().
		Foreground(theme.ColorPrimary).
		Bold(true).
		Padding(0, 1)
	inactiveTab := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Padding(0, 1)

	tabs := []struct {
		label string
		mode  DetailMode
	}{
		{"Info", DetailInfo},
		{"Logs", DetailLogs},
		{"Diff", DetailDiff},
		{"History", DetailHistory},
		{"All Logs", DetailAllLogs},
	}

	var parts []string
	for _, t := range tabs {
		style := inactiveTab
		if t.mode == m.mode {
			style = activeTab
		}
		parts = append(parts, style.Render(t.label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
