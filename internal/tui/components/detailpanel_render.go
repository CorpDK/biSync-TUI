package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

func (m *DetailPanelModel) renderInfo() string {
	if m.mapping == nil {
		return "  Select a mapping to view details."
	}
	var b strings.Builder
	b.WriteString(theme.DetailHeaderStyle.Render("Mapping Details") + "\n\n")

	row := detailRow(&b)
	row("Name", m.mapping.Name)
	m.renderMappingPaths(&b, row)
	m.renderMappingConfig(&b, row)
	m.renderStorageInfo(&b, row)
	m.renderStateInfo(&b, row)
	return b.String()
}

func (m *DetailPanelModel) renderMappingPaths(b *strings.Builder, row func(string, string)) {
	if config.IsRemotePath(m.mapping.Local) {
		row("Path 1", m.mapping.Local+" (remote)")
	} else {
		row("Local", m.mapping.Local)
	}
	row("Remote", m.mapping.Remote)
}

func (m *DetailPanelModel) renderMappingConfig(b *strings.Builder, row func(string, string)) {
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
}

func (m *DetailPanelModel) renderStorageInfo(b *strings.Builder, row func(string, string)) {
	if m.remoteAbout == nil {
		return
	}
	b.WriteString("\n")
	totalGiB := float64(m.remoteAbout.Total) / (1024 * 1024 * 1024)
	usedGiB := float64(m.remoteAbout.Used) / (1024 * 1024 * 1024)
	pct := float64(0)
	if m.remoteAbout.Total > 0 {
		pct = float64(m.remoteAbout.Used) / float64(m.remoteAbout.Total) * 100
	}
	row("Storage", fmt.Sprintf("%.1f GiB / %.1f GiB (%.0f%% used)", usedGiB, totalGiB, pct))
	if m.remoteAbout.Trashed > 0 {
		row("Trashed", fmt.Sprintf("%.1f MiB", float64(m.remoteAbout.Trashed)/(1024*1024)))
	}
}

func (m *DetailPanelModel) renderStateInfo(b *strings.Builder, row func(string, string)) {
	if m.state == nil {
		return
	}
	b.WriteString("\n")
	row("Status", string(m.state.LastStatus))
	if m.state.LastSync != nil {
		row("Last sync", fmt.Sprintf("%s (%s)",
			m.state.LastSync.Format("2006-01-02 15:04:05"),
			timeAgo(*m.state.LastSync)))
	} else {
		row("Last sync", "never")
	}
	row("Syncs", fmt.Sprintf("%d", m.state.SyncCount))
	if m.state.LastDuration != "" {
		row("Duration", m.state.LastDuration)
	}
	if m.state.LastError != "" {
		b.WriteString("\n")
		fmt.Fprintf(b, "  %s\n", theme.StatusErrorStyle.Render("Error: "+m.state.LastError))
	}
}

func (m *DetailPanelModel) renderLogs() string {
	if len(m.logLines) == 0 {
		return "  No log output yet."
	}
	var b strings.Builder
	b.WriteString(theme.DetailHeaderStyle.Render("Sync Output") + "\n\n")
	maxW := max(m.width-4, 20)
	for _, line := range m.logLines {
		b.WriteString("  " + wrapLine(line, maxW) + "\n")
	}
	return b.String()
}

func (m *DetailPanelModel) renderHistory() string {
	if len(m.history) == 0 {
		return "  No sync history yet."
	}
	var b strings.Builder
	b.WriteString(theme.DetailHeaderStyle.Render("Sync History") + "\n\n")
	fmt.Fprintf(&b, "  %-20s  %-8s  %-10s  %-6s  %s\n",
		"Timestamp", "Status", "Duration", "Files", "Bytes")
	fmt.Fprintf(&b, "  %-20s  %-8s  %-10s  %-6s  %s\n",
		"--------------------", "--------", "----------", "------", "-----")

	for i := len(m.history) - 1; i >= 0; i-- {
		r := m.history[i]
		style := theme.StatusIdleStyle
		if r.Status == "error" {
			style = theme.StatusErrorStyle
		}
		fmt.Fprintf(&b, "  %-20s  %s  %-10s  %-6d  %d\n",
			r.Timestamp.Format("2006-01-02 15:04:05"),
			style.Render(fmt.Sprintf("%-8s", r.Status)),
			r.Duration.Truncate(time.Second).String(),
			r.FilesTransferred, r.BytesTransferred)
	}
	return b.String()
}

func (m *DetailPanelModel) renderAllLogs() string {
	if len(m.allLogEntries) == 0 {
		return "  No aggregated logs yet."
	}
	var b strings.Builder
	b.WriteString(theme.DetailHeaderStyle.Render("All Logs") + "\n\n")

	for _, e := range m.allLogEntries {
		levelStyle := theme.StatusInitStyle
		switch e.Level {
		case "ERROR":
			levelStyle = theme.StatusErrorStyle
		case "WARN", "NOTICE":
			levelStyle = theme.StatusSyncStyle
		}
		fmt.Fprintf(&b, "  %s %s %s %s\n",
			theme.DetailLabelStyle.Render(e.Timestamp.Format("15:04:05")),
			theme.StatusSyncStyle.Render(fmt.Sprintf("[%-10s]", e.MappingName)),
			levelStyle.Render(fmt.Sprintf("[%s]", e.Level)),
			e.Message)
	}
	return b.String()
}

// detailRow returns a row-writing closure for consistent label/value formatting.
func detailRow(b *strings.Builder) func(string, string) {
	return func(label, value string) {
		fmt.Fprintf(b, "  %s %s\n",
			theme.DetailLabelStyle.Render(label+":"),
			theme.DetailValueStyle.Render(value))
	}
}

// wrapLine soft-wraps a line to fit within maxWidth columns.
func wrapLine(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	var b strings.Builder
	for len(s) > maxWidth {
		b.WriteString(s[:maxWidth])
		b.WriteByte('\n')
		b.WriteString("  ")
		s = s[maxWidth:]
	}
	b.WriteString(s)
	return b.String()
}
