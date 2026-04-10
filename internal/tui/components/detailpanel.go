package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/logs"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// RemoteDisplayInfo holds remote config info for display.
type RemoteDisplayInfo struct {
	Name    string
	Type    string
	Details map[string]string
}

// RemoteAboutInfo holds storage space data for display.
type RemoteAboutInfo struct {
	Total, Used, Free, Trashed int64
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

	diffView       DiffViewModel
	mapping        *config.Mapping
	state          *state.MappingState
	logLines       []string
	history        []state.HistoryRecord
	allLogEntries  []logs.LogEntry
	remoteAbout *RemoteAboutInfo
	autoScroll  bool
}

// NewDetailPanel creates a new detail panel.
func NewDetailPanel(width, height int) DetailPanelModel {
	vp := viewport.New(width, height)
	vp.SetContent("Select a mapping to view details.")
	return DetailPanelModel{
		viewport: vp, mode: DetailInfo,
		width: width, height: height, autoScroll: true,
	}
}

func (m *DetailPanelModel) SetActive(active bool) { m.active = active }

// Reset clears the panel back to its empty state.
func (m *DetailPanelModel) Reset() {
	m.mapping = nil
	m.state = nil
	m.logLines = nil
	m.mode = DetailInfo
	m.viewport.SetContent("Select a mapping to view details.")
}
func (m DetailPanelModel) Mode() DetailMode          { return m.mode }
func (m *DetailPanelModel) SetMode(mode DetailMode)  { m.mode = mode; m.refreshContent() }
func (m *DetailPanelModel) ClearLogs()               { m.logLines = nil; m.refreshContent() }
func (m *DetailPanelModel) UpdateState(st *state.MappingState) {
	m.state = st
	if m.mode == DetailInfo { m.refreshContent() }
}

func (m *DetailPanelModel) SetSize(w, h int) {
	m.width, m.height = w, h
	m.viewport.Width = w
	m.viewport.Height = h - 1
	m.refreshContent()
}

func (m *DetailPanelModel) SetMapping(mapping *config.Mapping, st *state.MappingState) {
	m.mapping, m.state = mapping, st
	m.mode = DetailInfo
	m.logLines = nil
	m.refreshContent()
}

func (m *DetailPanelModel) AppendLog(line string) {
	m.logLines = append(m.logLines, line)
	if m.mode == DetailLogs {
		m.refreshContent()
		if m.autoScroll { m.viewport.GotoBottom() }
	}
}

func (m *DetailPanelModel) SetHistory(records []state.HistoryRecord) {
	m.history = records
	if m.mode == DetailHistory { m.refreshContent() }
}

func (m *DetailPanelModel) SetAllLogs(entries []logs.LogEntry) {
	m.allLogEntries = entries
	if m.mode == DetailAllLogs { m.refreshContent() }
}

func (m *DetailPanelModel) SetRemoteAbout(about *RemoteAboutInfo) {
	m.remoteAbout = about
	if m.mode == DetailInfo { m.refreshContent() }
}

func (m *DetailPanelModel) SetDiffEntries(entries []bisync.DiffEntry) {
	m.diffView.SetEntries(entries)
	if m.mode == DetailDiff { m.refreshContent() }
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

// Update handles input for the detail panel.
func (m DetailPanelModel) Update(msg tea.Msg) (DetailPanelModel, tea.Cmd) {
	if !m.active { return m, nil }
	if km, ok := msg.(tea.KeyMsg); ok {
		if handled, cmd := m.handleDetailKey(km); handled {
			return m, cmd
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *DetailPanelModel) handleDetailKey(km tea.KeyMsg) (bool, tea.Cmd) {
	switch km.String() {
	case "h", "left":
		if m.mode > DetailInfo { m.mode--; m.refreshContent() }
		return true, nil
	case "l", "right":
		if m.mode < DetailAllLogs { m.mode++; m.refreshContent() }
		return true, nil
	}
	return false, nil
}

// View renders the detail panel.
func (m DetailPanelModel) View() string {
	style := theme.InactivePanel
	if m.active { style = theme.ActivePanel }

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.renderTabs(), m.viewport.View())

	return style.Width(m.width).Height(m.height).Render(content)
}

func (m DetailPanelModel) renderTabs() string {
	tabs := []struct{ label string; mode DetailMode }{
		{"Info", DetailInfo}, {"Logs", DetailLogs}, {"Diff", DetailDiff},
		{"History", DetailHistory}, {"All Logs", DetailAllLogs},
	}
	activeTab := lipgloss.NewStyle().Foreground(theme.ColorPrimary).Bold(true).Padding(0, 1)
	inactiveTab := lipgloss.NewStyle().Foreground(theme.ColorMuted).Padding(0, 1)

	var parts []string
	for _, t := range tabs {
		s := inactiveTab
		if t.mode == m.mode { s = activeTab }
		parts = append(parts, s.Render(t.label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
