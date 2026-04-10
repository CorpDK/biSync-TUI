package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/state"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
)

func (m AppModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		m.cancelFunc()
		m.syncPool.Shutdown()
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
	}

	// View switching via number keys
	switch msg.String() {
	case "1":
		return m.switchView(ViewMappings)
	case "2":
		return m.switchView(ViewRemotes)
	case "3":
		return m.switchView(ViewDashboard)
	}

	// Delegate to active view
	switch m.viewMode {
	case ViewMappings:
		return m.handleMappingsKey(msg)
	case ViewRemotes:
		return m.handleRemotesKey(msg)
	case ViewDashboard:
		return m.handleDashboardKey(msg)
	}
	return m, nil
}

func (m AppModel) switchView(mode ViewMode) (tea.Model, tea.Cmd) {
	m.viewMode = mode
	m.focusedPanel = PanelList
	m.titleBar.SetViewMode(int(mode))

	// Deactivate all panels
	m.mappingList.SetActive(false)
	m.detailPanel.SetActive(false)
	m.remoteList.SetActive(false)
	m.remoteDetail.SetActive(false)

	switch mode {
	case ViewRemotes:
		m.remoteList.SetActive(true)
		m.statusBar.SetBindings(RemotesHelp())
		return m, m.loadRemotes()
	case ViewDashboard:
		m.dashboard.SetData(m.config.Mappings, m.states, m.remoteList.ItemCount())
		m.statusBar.SetBindings(DashboardHelp())
		return m, nil
	default:
		m.mappingList.SetActive(true)
		m.statusBar.SetBindings(m.keys.MappingsHelp())
		return m, nil
	}
}

func (m AppModel) togglePanelFocus() (tea.Model, tea.Cmd) {
	if m.focusedPanel == PanelList {
		m.focusedPanel = PanelDetail
	} else {
		m.focusedPanel = PanelList
	}
	switch m.viewMode {
	case ViewMappings:
		m.mappingList.SetActive(m.focusedPanel == PanelList)
		m.detailPanel.SetActive(m.focusedPanel == PanelDetail)
	case ViewRemotes:
		m.remoteList.SetActive(m.focusedPanel == PanelList)
		m.remoteDetail.SetActive(m.focusedPanel == PanelDetail)
	}
	return m, nil
}

func (m AppModel) handleMappingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Tab):
		return m.togglePanelFocus()
	case key.Matches(msg, m.keys.Select) && m.focusedPanel == PanelList:
		return m.openMappingActionMenu()
	case key.Matches(msg, m.keys.Sync):
		return m.syncSelected(false, false)
	case key.Matches(msg, m.keys.SyncAll):
		return m.syncAll()
	case key.Matches(msg, m.keys.DryRun):
		return m.syncSelected(true, false)
	case key.Matches(msg, m.keys.Resync):
		return m.confirmResync()
	case key.Matches(msg, m.keys.Logs) && m.focusedPanel == PanelList:
		m.detailPanel.SetMode(components.DetailLogs)
		return m, nil
	case key.Matches(msg, m.keys.Info) && m.focusedPanel == PanelList:
		m.detailPanel.SetMode(components.DetailInfo)
		return m, nil
	case key.Matches(msg, m.keys.Diff):
		return m.runDiffPreview()
	case key.Matches(msg, m.keys.NewMapping):
		return m.showNewMappingForm()
	}
	return m.routeMappingsPanel(msg)
}

func (m AppModel) openMappingActionMenu() (tea.Model, tea.Cmd) {
	if selected := m.mappingList.SelectedMapping(); selected != nil {
		needsInit := selected.State.LastStatus == state.StatusNeedsInit
		menu := components.NewActionMenu(selected.Mapping.Name, needsInit, m.width, m.height)
		m.actionMenu = &menu
	}
	return m, nil
}

func (m AppModel) confirmResync() (tea.Model, tea.Cmd) {
	if selected := m.mappingList.SelectedMapping(); selected != nil {
		modal := components.NewModal(
			"resync-"+selected.Mapping.Name,
			"Force Resync",
			"This will re-establish the baseline for '"+selected.Mapping.Name+
				"'.\nAny unsynced changes may be overwritten.",
			m.width, m.height,
		)
		m.modal = &modal
	}
	return m, nil
}

func (m AppModel) routeMappingsPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focusedPanel == PanelList {
		m.mappingList, cmd = m.mappingList.Update(msg)
		if selected := m.mappingList.SelectedMapping(); selected != nil {
			m.detailPanel.SetMapping(&selected.Mapping, selected.State)
		}
	} else {
		m.detailPanel, cmd = m.detailPanel.Update(msg)
	}
	return m, cmd
}

func (m AppModel) handleRemotesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Tab):
		return m.togglePanelFocus()
	}
	switch msg.String() {
	case "C":
		return m.launchRcloneConfig()
	case "X":
		return m.confirmDeleteRemote()
	case "t":
		return m.testSelectedRemote()
	}
	return m.routeRemotesPanel(msg)
}

func (m AppModel) confirmDeleteRemote() (tea.Model, tea.Cmd) {
	if name := m.remoteDetail.SelectedRemoteName(); name != "" {
		modal := components.NewModal(
			"delete-remote-"+name,
			"Delete Remote",
			"Are you sure you want to delete remote '"+name+"'?\nThis cannot be undone.",
			m.width, m.height,
		)
		m.modal = &modal
	}
	return m, nil
}

func (m AppModel) testSelectedRemote() (tea.Model, tea.Cmd) {
	if selected := m.remoteList.SelectedRemote(); selected != nil {
		m.remoteDetail.SetStatus("◐ Testing connection...")
		return m, m.testRemoteConnection(selected.Name)
	}
	return m, nil
}

func (m AppModel) routeRemotesPanel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.focusedPanel == PanelList {
		m.remoteList, cmd = m.remoteList.Update(msg)
		if selected := m.remoteList.SelectedRemote(); selected != nil {
			m.remoteDetail.SetRemote(selected)
		}
	} else {
		m.remoteDetail, cmd = m.remoteDetail.Update(msg)
	}
	return m, cmd
}

func (m AppModel) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.dashboard, cmd = m.dashboard.Update(msg)
	return m, cmd
}
