package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// Update handles all messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case components.FormSubmittedMsg:
		m.formOverlay = nil
		return m.handleFormSubmit(msg)
	case components.FormCancelledMsg:
		m.formOverlay = nil
		return m, nil

	case components.ActionSelectedMsg:
		m.actionMenu = nil
		return m.handleAction(msg)
	case components.ActionCancelledMsg:
		m.actionMenu = nil
		return m, nil

	case components.ModalConfirmMsg:
		m.modal = nil
		return m.handleModalConfirm(msg)
	case components.ModalCancelMsg:
		m.modal = nil
		return m, nil

	case PoolOutputMsg:
		return m.handlePoolOutput(msg)
	case PoolResultMsg:
		return m.handlePoolResult(msg)

	case TickMsg:
		m.mappingList.UpdateState(m.states)
		return m, m.tickCmd()
	case StateRefreshMsg:
		m.states = m.stateStore.LoadAll(m.config.Mappings)
		m.mappingList.UpdateState(m.states)
		return m, nil
	case HealthStatusMsg:
		return m.handleHealthStatus(msg)

	case HistoryLoadedMsg:
		m.detailPanel.SetHistory(msg.Records)
		m.detailPanel.SetMode(components.DetailHistory)
		return m, nil
	case AggregatedLogsMsg:
		m.detailPanel.SetAllLogs(msg.Entries)
		m.detailPanel.SetMode(components.DetailAllLogs)
		return m, nil

	case RemoteAboutMsg:
		return m.handleRemoteAbout(msg)
	case DiffResultMsg:
		return m.handleDiffResult(msg)
	case ConflictsDetectedMsg:
		return m, nil

	case RemotesLoadedMsg:
		return m.handleRemotesLoaded(msg)
	case RemoteDeletedMsg:
		if msg.Err != nil {
			m.remoteDetail.SetStatus(
				theme.StatusErrorStyle.Render("Error deleting remote: " + msg.Err.Error()))
			return m, nil
		}
		m.remoteDetail.SetStatus("")
		return m, m.loadRemotes()
	case RemoteTestMsg:
		return m.handleRemoteTest(msg)
	case RcloneConfigDoneMsg:
		return m, m.loadRemotes()
	}

	// Forward unhandled messages to the form overlay when active
	if m.formOverlay != nil {
		overlay, cmd := m.formOverlay.Update(msg)
		m.formOverlay = &overlay
		return m, cmd
	}

	// Route unhandled messages to the active view's panels
	cmd := m.routeToActivePanel(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.layout()
	return m, nil
}

func (m AppModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}
	if m.modal != nil {
		modal, cmd := m.modal.Update(msg)
		m.modal = &modal
		return m, cmd
	}
	if m.formOverlay != nil {
		overlay, cmd := m.formOverlay.Update(msg)
		m.formOverlay = &overlay
		return m, cmd
	}
	if m.actionMenu != nil {
		menu, cmd := m.actionMenu.Update(msg)
		m.actionMenu = &menu
		return m, cmd
	}
	return m.handleKeyPress(msg)
}

func (m AppModel) handlePoolOutput(msg PoolOutputMsg) (tea.Model, tea.Cmd) {
	line := bisync.OutputLine(msg)
	selected := m.mappingList.SelectedMapping()
	if selected != nil && selected.Mapping.Name == line.MappingName {
		m.detailPanel.AppendLog(line.Line)
	}
	return m, m.listenPoolOutput()
}

func (m AppModel) handlePoolResult(msg PoolResultMsg) (tea.Model, tea.Cmd) {
	result := bisync.JobResult(msg)
	ms, _ := m.stateStore.Load(result.MappingName)
	m.states[result.MappingName] = ms
	m.mappingList.UpdateState(m.states)
	selected := m.mappingList.SelectedMapping()
	if selected != nil && selected.Mapping.Name == result.MappingName {
		m.detailPanel.UpdateState(ms)
	}
	return m, m.listenPoolResults()
}

func (m AppModel) handleHealthStatus(msg HealthStatusMsg) (tea.Model, tea.Cmd) {
	allHealthy := true
	for _, s := range msg.Statuses {
		if !s.Healthy {
			allHealthy = false
			break
		}
	}
	m.titleBar.SetConnected(allHealthy)
	return m, m.listenHealthUpdates()
}

func (m AppModel) handleRemoteAbout(msg RemoteAboutMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil && msg.About != nil {
		m.detailPanel.SetRemoteAbout(&components.RemoteAboutInfo{
			Total:   msg.About.Total,
			Used:    msg.About.Used,
			Free:    msg.About.Free,
			Trashed: msg.About.Trashed,
		})
	}
	return m, nil
}

func (m AppModel) handleDiffResult(msg DiffResultMsg) (tea.Model, tea.Cmd) {
	if msg.Error != "" {
		m.detailPanel.AppendLog("Diff error: " + msg.Error)
	}
	m.detailPanel.SetDiffEntries(msg.Entries)
	m.detailPanel.SetMode(components.DetailDiff)
	return m, nil
}

func (m AppModel) handleRemotesLoaded(msg RemotesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.detailPanel.AppendLog("Error loading remotes: " + msg.Err.Error())
		return m, nil
	}
	infos := make([]components.RemoteDisplayInfo, len(msg.Remotes))
	for i, r := range msg.Remotes {
		infos[i] = components.RemoteDisplayInfo{
			Name: r.Name, Type: r.Type, Details: r.Details,
		}
	}
	// Rebuild the list to ensure the UI reflects the new items
	m.remoteList = components.NewRemoteList(infos,
		m.remoteList.Width(), m.remoteList.Height())
	m.remoteList.SetActive(m.viewMode == ViewRemotes && m.focusedPanel == PanelList)
	if selected := m.remoteList.SelectedRemote(); selected != nil {
		m.remoteDetail.SetRemote(selected)
	}
	return m, nil
}

func (m AppModel) handleRemoteTest(msg RemoteTestMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.remoteDetail.SetStatus(
			theme.StatusErrorStyle.Render(fmt.Sprintf("✗ Connection failed: %v", msg.Err)))
	} else {
		m.remoteDetail.SetStatus(
			theme.StatusIdleStyle.Render("✓ Connection OK"))
	}
	return m, nil
}

func (m AppModel) routeToActivePanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.viewMode {
	case ViewMappings:
		if m.focusedPanel == PanelList {
			m.mappingList, cmd = m.mappingList.Update(msg)
			if selected := m.mappingList.SelectedMapping(); selected != nil {
				m.detailPanel.SetMapping(&selected.Mapping, selected.State)
			}
		} else {
			m.detailPanel, cmd = m.detailPanel.Update(msg)
		}
	case ViewRemotes:
		if m.focusedPanel == PanelList {
			m.remoteList, cmd = m.remoteList.Update(msg)
			if selected := m.remoteList.SelectedRemote(); selected != nil {
				m.remoteDetail.SetRemote(selected)
			}
		} else {
			m.remoteDetail, cmd = m.remoteDetail.Update(msg)
		}
	case ViewDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	}
	return cmd
}
