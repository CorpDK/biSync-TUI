package tui

import (
	"context"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/logs"
	"github.com/CorpDK/bisync-tui/internal/notify"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
)

// FocusedPanel tracks which panel has input focus.
type FocusedPanel int

const (
	PanelList FocusedPanel = iota
	PanelDetail
)

// AppModel is the root Bubbletea model.
type AppModel struct {
	// Dependencies
	config        *config.Config
	stateStore    *state.Store
	historyStore  *state.HistoryStore
	logMgr        *logs.LogManager
	syncPool      *bisync.Pool
	engine        *bisync.Engine
	lockMgr       *bisync.LockManager
	healthChecker *bisync.HealthChecker

	// UI components
	mappingList components.MappingListModel
	detailPanel components.DetailPanelModel
	statusBar   components.StatusBarModel
	titleBar    components.TitleBarModel
	actionMenu  *components.ActionMenuModel
	modal       *components.ModalModel

	// State
	focusedPanel FocusedPanel
	states       map[string]*state.MappingState
	width        int
	height       int
	keys         KeyMap
	quitting     bool
	version      string
	ctx          context.Context
	cancelFunc   context.CancelFunc
}

// NewApp creates the root application model.
func NewApp(cfg *config.Config, stateStore *state.Store, engine *bisync.Engine, lockMgr *bisync.LockManager, version string) AppModel {
	ctx, cancel := context.WithCancel(context.Background())

	keys := DefaultKeyMap()
	states := stateStore.LoadAll(cfg.Mappings)

	// Create stores and worker pool
	historyStore := state.NewHistoryStore(filepath.Join(config.StateDir(), "history"), 500)
	logMgr := logs.NewLogManager(filepath.Join(config.StateDir(), "logs"))
	notifier := notify.NewNotifier(cfg.Global.Notifications)
	pool := bisync.NewPool(cfg.Global.MaxWorkers, engine, lockMgr, stateStore, historyStore, logMgr, notifier)
	pool.Start(ctx)

	// Create health checker
	hc := bisync.NewHealthChecker(engine, cfg.Mappings, 5*time.Minute)
	go hc.Start(ctx)

	return AppModel{
		config:        cfg,
		stateStore:    stateStore,
		historyStore:  historyStore,
		logMgr:        logMgr,
		syncPool:      pool,
		engine:        engine,
		lockMgr:       lockMgr,
		healthChecker: hc,
		mappingList:   components.NewMappingList(cfg.Mappings, states, 30, 20),
		detailPanel:   components.NewDetailPanel(50, 20),
		statusBar:     components.NewStatusBar(keys.ShortHelp(), 80),
		titleBar:      components.NewTitleBar(version, 80),
		states:        states,
		keys:          keys,
		version:       version,
		ctx:           ctx,
		cancelFunc:    cancel,
	}
}

// Init returns the initial command.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
		m.listenPoolOutput(),
		m.listenPoolResults(),
		m.listenHealthUpdates(),
	)
}

// Update handles all messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		// If modal is showing, it gets priority
		if m.modal != nil {
			modal, cmd := m.modal.Update(msg)
			m.modal = &modal
			return m, cmd
		}

		// If action menu is showing, it gets priority
		if m.actionMenu != nil {
			menu, cmd := m.actionMenu.Update(msg)
			m.actionMenu = &menu
			return m, cmd
		}

		return m.handleKeyPress(msg)

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
		line := bisync.OutputLine(msg)
		selected := m.mappingList.SelectedMapping()
		if selected != nil && selected.Mapping.Name == line.MappingName {
			m.detailPanel.AppendLog(line.Line)
		}
		return m, m.listenPoolOutput()

	case PoolResultMsg:
		result := bisync.JobResult(msg)
		// Refresh state
		ms, _ := m.stateStore.Load(result.MappingName)
		m.states[result.MappingName] = ms
		m.mappingList.UpdateState(m.states)
		selected := m.mappingList.SelectedMapping()
		if selected != nil && selected.Mapping.Name == result.MappingName {
			m.detailPanel.UpdateState(ms)
		}
		return m, m.listenPoolResults()

	case TickMsg:
		// Periodic state refresh
		m.mappingList.UpdateState(m.states)
		return m, m.tickCmd()

	case StateRefreshMsg:
		m.states = m.stateStore.LoadAll(m.config.Mappings)
		m.mappingList.UpdateState(m.states)
		return m, nil

	case HealthStatusMsg:
		allHealthy := true
		for _, s := range msg.Statuses {
			if !s.Healthy {
				allHealthy = false
				break
			}
		}
		m.titleBar.SetConnected(allHealthy)
		return m, m.listenHealthUpdates()

	case HistoryLoadedMsg:
		m.detailPanel.SetHistory(msg.Records)
		m.detailPanel.SetMode(components.DetailHistory)
		return m, nil

	case AggregatedLogsMsg:
		m.detailPanel.SetAllLogs(msg.Entries)
		m.detailPanel.SetMode(components.DetailAllLogs)
		return m, nil

	case RemoteAboutMsg:
		if msg.Err == nil && msg.About != nil {
			m.detailPanel.SetRemoteAbout(&components.RemoteAboutInfo{
				Total:   msg.About.Total,
				Used:    msg.About.Used,
				Free:    msg.About.Free,
				Trashed: msg.About.Trashed,
			})
		}
		return m, nil

	case ConflictsDetectedMsg:
		// Store conflicts for later viewing - handled in detail panel
		return m, nil
	}

	// Pass to focused panel
	if m.focusedPanel == PanelList {
		var cmd tea.Cmd
		m.mappingList, cmd = m.mappingList.Update(msg)
		cmds = append(cmds, cmd)
		// Update detail panel when selection changes
		if selected := m.mappingList.SelectedMapping(); selected != nil {
			m.detailPanel.SetMapping(&selected.Mapping, selected.State)
		}
	} else {
		var cmd tea.Cmd
		m.detailPanel, cmd = m.detailPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		m.cancelFunc()
		m.syncPool.Shutdown()
		return m, tea.Quit

	case key.Matches(msg, m.keys.Tab):
		if m.focusedPanel == PanelList {
			m.focusedPanel = PanelDetail
			m.mappingList.SetActive(false)
			m.detailPanel.SetActive(true)
		} else {
			m.focusedPanel = PanelList
			m.mappingList.SetActive(true)
			m.detailPanel.SetActive(false)
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if m.focusedPanel == PanelList {
			if selected := m.mappingList.SelectedMapping(); selected != nil {
				needsInit := selected.State.LastStatus == state.StatusNeedsInit
				menu := components.NewActionMenu(selected.Mapping.Name, needsInit, m.width, m.height)
				m.actionMenu = &menu
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Sync):
		return m.syncSelected(false, false)

	case key.Matches(msg, m.keys.SyncAll):
		return m.syncAll()

	case key.Matches(msg, m.keys.DryRun):
		return m.syncSelected(true, false)

	case key.Matches(msg, m.keys.Resync):
		if selected := m.mappingList.SelectedMapping(); selected != nil {
			modal := components.NewModal(
				"resync-"+selected.Mapping.Name,
				"Force Resync",
				"This will re-establish the baseline for '"+selected.Mapping.Name+"'.\nAny unsynced changes may be overwritten.",
				m.width, m.height,
			)
			m.modal = &modal
		}
		return m, nil

	case key.Matches(msg, m.keys.Logs):
		m.detailPanel.SetMode(components.DetailLogs)
		return m, nil

	case key.Matches(msg, m.keys.Info):
		m.detailPanel.SetMode(components.DetailInfo)
		return m, nil

	case key.Matches(msg, m.keys.Help):
		// Toggle help display - for now just show full keybindings in detail panel
		return m, nil
	}

	// Pass to focused panel
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

func (m AppModel) handleAction(msg components.ActionSelectedMsg) (tea.Model, tea.Cmd) {
	mapping := m.findMapping(msg.MappingName)
	if mapping == nil {
		return m, nil
	}

	switch msg.Action {
	case components.ActionSync:
		return m.submitSync(*mapping, bisync.SyncOptions{})
	case components.ActionDryRun:
		return m.submitSync(*mapping, bisync.SyncOptions{DryRun: true})
	case components.ActionResync:
		modal := components.NewModal(
			"resync-"+mapping.Name,
			"Force Resync",
			"This will re-establish the baseline.\nAny unsynced changes may be overwritten.",
			m.width, m.height,
		)
		m.modal = &modal
		return m, nil
	case components.ActionInit:
		return m.initMapping(*mapping)
	case components.ActionLogs:
		m.detailPanel.SetMode(components.DetailLogs)
		return m, nil
	case components.ActionInfo:
		m.detailPanel.SetMode(components.DetailInfo)
		return m, nil
	case components.ActionRemoteSize:
		return m, m.fetchRemoteSize(*mapping)
	case components.ActionHistory:
		return m, m.loadHistory(mapping.Name)
	case components.ActionAllLogs:
		return m, m.loadAllLogs()
	}
	return m, nil
}

func (m AppModel) handleModalConfirm(msg components.ModalConfirmMsg) (tea.Model, tea.Cmd) {
	// Extract mapping name from modal ID (format: "resync-{name}")
	if len(msg.ID) > 7 && msg.ID[:7] == "resync-" {
		name := msg.ID[7:]
		if mapping := m.findMapping(name); mapping != nil {
			return m.submitSync(*mapping, bisync.SyncOptions{Resync: true})
		}
	}
	return m, nil
}

func (m AppModel) syncSelected(dryRun, resync bool) (tea.Model, tea.Cmd) {
	selected := m.mappingList.SelectedMapping()
	if selected == nil {
		return m, nil
	}
	return m.submitSync(selected.Mapping, bisync.SyncOptions{DryRun: dryRun, Resync: resync})
}

func (m AppModel) syncAll() (tea.Model, tea.Cmd) {
	for _, mapping := range m.config.Mappings {
		ms := m.states[mapping.Name]
		if ms != nil && ms.Initialized {
			m.syncPool.Submit(bisync.Job{Mapping: mapping, Options: bisync.SyncOptions{}})
		}
	}
	return m, nil
}

func (m AppModel) submitSync(mapping config.Mapping, opts bisync.SyncOptions) (tea.Model, tea.Cmd) {
	// Switch detail to log view and clear
	m.detailPanel.SetMode(components.DetailLogs)
	m.detailPanel.ClearLogs()

	// Update state to syncing
	if ms, ok := m.states[mapping.Name]; ok {
		ms.LastStatus = state.StatusSyncing
		m.mappingList.UpdateState(m.states)
	}

	m.syncPool.Submit(bisync.Job{Mapping: mapping, Options: opts})
	return m, nil
}

func (m AppModel) initMapping(mapping config.Mapping) (tea.Model, tea.Cmd) {
	m.detailPanel.SetMode(components.DetailLogs)
	m.detailPanel.ClearLogs()

	// For initialization, we do a resync
	opts := bisync.SyncOptions{Resync: true}
	m.syncPool.Submit(bisync.Job{Mapping: mapping, Options: opts})

	return m, nil
}

func (m AppModel) fetchRemoteSize(mapping config.Mapping) tea.Cmd {
	return func() tea.Msg {
		about, err := m.engine.GetRemoteAbout(m.ctx, mapping.Remote)
		return RemoteAboutMsg{
			Remote: mapping.Remote,
			About:  about,
			Err:    err,
		}
	}
}

func (m AppModel) loadHistory(name string) tea.Cmd {
	return func() tea.Msg {
		records, _ := m.historyStore.Load(name, 100)
		return HistoryLoadedMsg{MappingName: name, Records: records}
	}
}

func (m AppModel) loadAllLogs() tea.Cmd {
	return func() tea.Msg {
		entries, _ := m.logMgr.ReadAll(500)
		return AggregatedLogsMsg{Entries: entries}
	}
}

func (m AppModel) findMapping(name string) *config.Mapping {
	for _, mapping := range m.config.Mappings {
		if mapping.Name == name {
			return &mapping
		}
	}
	return nil
}

func (m AppModel) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m AppModel) listenPoolOutput() tea.Cmd {
	return func() tea.Msg {
		line, ok := <-m.syncPool.Output()
		if !ok {
			return nil
		}
		return PoolOutputMsg(line)
	}
}

func (m AppModel) listenPoolResults() tea.Cmd {
	return func() tea.Msg {
		result, ok := <-m.syncPool.Results()
		if !ok {
			return nil
		}
		return PoolResultMsg(result)
	}
}

func (m AppModel) listenHealthUpdates() tea.Cmd {
	return func() tea.Msg {
		statuses, ok := <-m.healthChecker.Updates()
		if !ok {
			return nil
		}
		return HealthStatusMsg{Statuses: statuses}
	}
}

func (m *AppModel) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	// Title bar: 1 line, Status bar: 1 line, borders: 2 lines
	contentHeight := m.height - 4

	// Left panel: 40% width, Right panel: 60% width
	leftWidth := int(float64(m.width) * 0.4)
	rightWidth := m.width - leftWidth

	// Account for panel borders
	innerLeftW := leftWidth - 2
	innerRightW := rightWidth - 2
	innerH := contentHeight - 2

	if innerLeftW < 10 {
		innerLeftW = 10
	}
	if innerRightW < 10 {
		innerRightW = 10
	}
	if innerH < 5 {
		innerH = 5
	}

	m.mappingList.SetSize(innerLeftW, innerH)
	m.detailPanel.SetSize(innerRightW, innerH-2) // account for tabs
	m.statusBar.SetWidth(m.width)
	m.titleBar.SetWidth(m.width)
}

// View renders the full application.
func (m AppModel) View() string {
	if m.quitting {
		return "Shutting down...\n"
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Render overlay if present
	if m.modal != nil {
		return m.modal.View()
	}
	if m.actionMenu != nil {
		return m.actionMenu.View()
	}

	// Main layout
	titleBar := m.titleBar.View()
	statusBar := m.statusBar.View()

	leftPanel := m.mappingList.View()
	rightPanel := m.detailPanel.View()

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		panels,
		statusBar,
	)
}
