package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
	"github.com/CorpDK/bisync-tui/internal/tui/forms"
	"github.com/CorpDK/bisync-tui/internal/tui/theme"
)

// ViewMode tracks which top-level view is active.
type ViewMode int

const (
	ViewMappings  ViewMode = iota // 1 - Sync mappings
	ViewRemotes                   // 2 - Rclone remotes
	ViewDashboard                 // 3 - Dashboard overview
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

	// UI components — Mappings view
	mappingList components.MappingListModel
	detailPanel components.DetailPanelModel

	// UI components — Remotes view
	remoteList   components.RemoteListModel
	remoteDetail components.RemoteDetailModel

	// UI components — Dashboard view
	dashboard components.DashboardModel

	// UI components — shared
	statusBar   components.StatusBarModel
	titleBar    components.TitleBarModel
	actionMenu  *components.ActionMenuModel
	modal       *components.ModalModel
	formOverlay *components.FormOverlayModel

	// State
	viewMode         ViewMode
	focusedPanel     FocusedPanel
	states           map[string]*state.MappingState
	width            int
	height           int
	keys             KeyMap
	quitting         bool
	version          string
	ctx              context.Context
	cancelFunc       context.CancelFunc
	pendingFirstForm bool // show new-mapping form after first WindowSizeMsg
	showHelp         bool
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
		config:           cfg,
		stateStore:       stateStore,
		historyStore:     historyStore,
		logMgr:           logMgr,
		syncPool:         pool,
		engine:           engine,
		lockMgr:          lockMgr,
		healthChecker:    hc,
		mappingList:      components.NewMappingList(cfg.Mappings, states, 30, 20),
		detailPanel:      components.NewDetailPanel(50, 20),
		remoteList:       components.NewRemoteList(nil, 30, 20),
		remoteDetail:     components.NewRemoteDetail(50, 20),
		dashboard:        components.NewDashboard(80, 20),
		statusBar:        components.NewStatusBar(keys.ShortHelp(), 80),
		titleBar:         components.NewTitleBar(version, 80),
		states:           states,
		keys:             keys,
		version:          version,
		ctx:              ctx,
		cancelFunc:       cancel,
		pendingFirstForm: len(cfg.Mappings) == 0,
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
		if m.pendingFirstForm {
			m.pendingFirstForm = false
			return m.showNewMappingForm()
		}
		return m, nil

	case tea.KeyMsg:
		// Dismiss help overlay on any key
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// If modal is showing, it gets priority
		if m.modal != nil {
			modal, cmd := m.modal.Update(msg)
			m.modal = &modal
			return m, cmd
		}

		// If form overlay is showing, it gets priority
		if m.formOverlay != nil {
			overlay, cmd := m.formOverlay.Update(msg)
			m.formOverlay = &overlay
			return m, cmd
		}

		// If action menu is showing, it gets priority
		if m.actionMenu != nil {
			menu, cmd := m.actionMenu.Update(msg)
			m.actionMenu = &menu
			return m, cmd
		}

		return m.handleKeyPress(msg)

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

	case DiffResultMsg:
		if msg.Error != "" {
			m.detailPanel.AppendLog("Diff error: " + msg.Error)
		}
		m.detailPanel.SetDiffEntries(msg.Entries)
		m.detailPanel.SetMode(components.DetailDiff)
		return m, nil

	case ConflictsDetectedMsg:
		// Store conflicts for later viewing - handled in detail panel
		return m, nil

	case RemotesLoadedMsg:
		if msg.Err != nil {
			m.detailPanel.AppendLog("Error loading remotes: " + msg.Err.Error())
			return m, nil
		}
		infos := make([]components.RemoteDisplayInfo, len(msg.Remotes))
		for i, r := range msg.Remotes {
			infos[i] = components.RemoteDisplayInfo{
				Name:    r.Name,
				Type:    r.Type,
				Details: r.Details,
			}
		}
		// Update both old detail-panel remotes tab and new remotes view
		m.detailPanel.SetRemotes(infos)
		m.remoteList.SetItems(infos)
		if m.viewMode == ViewRemotes {
			if selected := m.remoteList.SelectedRemote(); selected != nil {
				m.remoteDetail.SetRemote(selected)
			}
		} else {
			m.detailPanel.SetMode(components.DetailRemotes)
		}
		return m, nil

	case RemoteCreatedMsg:
		if msg.Err != nil {
			m.detailPanel.AppendLog("Error creating remote: " + msg.Err.Error())
		}
		return m, m.loadRemotes()

	case RemoteDeletedMsg:
		if msg.Err != nil {
			m.detailPanel.AppendLog("Error deleting remote: " + msg.Err.Error())
		}
		return m, m.loadRemotes()

	case RemoteTestMsg:
		if msg.Err != nil {
			m.detailPanel.AppendLog(fmt.Sprintf("Remote '%s': connection failed: %v", msg.Name, msg.Err))
		} else {
			m.detailPanel.AppendLog(fmt.Sprintf("Remote '%s': connection OK", msg.Name))
		}
		return m, nil
	}

	// Forward unhandled messages to the form overlay when active
	if m.formOverlay != nil {
		overlay, cmd := m.formOverlay.Update(msg)
		m.formOverlay = &overlay
		return m, cmd
	}

	// Route unhandled messages to the active view's panels
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
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m AppModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys (work in all views)
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

	switch mode {
	case ViewRemotes:
		m.remoteList.SetActive(true)
		m.remoteDetail.SetActive(false)
		m.mappingList.SetActive(false)
		m.detailPanel.SetActive(false)
		return m, m.loadRemotes()
	case ViewDashboard:
		m.mappingList.SetActive(false)
		m.detailPanel.SetActive(false)
		m.remoteList.SetActive(false)
		m.remoteDetail.SetActive(false)
		m.dashboard.SetData(m.config.Mappings, m.states, m.remoteList.ItemCount())
		return m, nil
	default: // ViewMappings
		m.mappingList.SetActive(true)
		m.detailPanel.SetActive(false)
		m.remoteList.SetActive(false)
		m.remoteDetail.SetActive(false)
		return m, nil
	}
}

func (m AppModel) handleMappingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Tab):
		return m.togglePanelFocus()

	case key.Matches(msg, m.keys.Select) && m.focusedPanel == PanelList:
		if selected := m.mappingList.SelectedMapping(); selected != nil {
			needsInit := selected.State.LastStatus == state.StatusNeedsInit
			menu := components.NewActionMenu(selected.Mapping.Name, needsInit, m.width, m.height)
			m.actionMenu = &menu
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

func (m AppModel) handleRemotesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Tab):
		return m.togglePanelFocus()
	}

	switch msg.String() {
	case "C":
		return m.showNewRemoteForm()
	case "X":
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
	case "t":
		if selected := m.remoteList.SelectedRemote(); selected != nil {
			return m, m.testRemoteConnection(selected.Name)
		}
		return m, nil
	}

	// Pass to focused panel
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

func (m AppModel) testRemoteConnection(name string) tea.Cmd {
	engine := m.engine
	ctx := m.ctx
	return func() tea.Msg {
		err := engine.CheckConnectivity(ctx, name+":")
		if err != nil {
			return RemoteTestMsg{Name: name, Err: err}
		}
		return RemoteTestMsg{Name: name}
	}
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
	case components.ActionDiff:
		return m.runDiffPreview()
	case components.ActionEncryption:
		return m.showEncryptionForm(msg.MappingName)
	case components.ActionEditMapping:
		return m.showEditMappingForm(*mapping)
	case components.ActionDeleteMapping:
		modal := components.NewModal(
			"delete-mapping-"+mapping.Name,
			"Delete Mapping",
			"Are you sure you want to delete mapping '"+mapping.Name+"'?\nThis removes it from your config.",
			m.width, m.height,
		)
		m.modal = &modal
		return m, nil
	}
	return m, nil
}

func (m AppModel) handleModalConfirm(msg components.ModalConfirmMsg) (tea.Model, tea.Cmd) {
	switch {
	case strings.HasPrefix(msg.ID, "resync-"):
		name := strings.TrimPrefix(msg.ID, "resync-")
		if mapping := m.findMapping(name); mapping != nil {
			return m.submitSync(*mapping, bisync.SyncOptions{Resync: true})
		}
	case strings.HasPrefix(msg.ID, "delete-remote-"):
		name := strings.TrimPrefix(msg.ID, "delete-remote-")
		return m, m.deleteRemote(name)
	case strings.HasPrefix(msg.ID, "delete-mapping-"):
		name := strings.TrimPrefix(msg.ID, "delete-mapping-")
		return m.handleDeleteMapping(name)
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

func (m AppModel) loadRemotes() tea.Cmd {
	return func() tea.Msg {
		remotes, err := m.engine.ConfigDump(m.ctx)
		return RemotesLoadedMsg{Remotes: remotes, Err: err}
	}
}

func (m AppModel) deleteRemote(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.engine.DeleteRemote(m.ctx, name)
		return RemoteDeletedMsg{Name: name, Err: err}
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

func (m AppModel) runDiffPreview() (tea.Model, tea.Cmd) {
	selected := m.mappingList.SelectedMapping()
	if selected == nil {
		return m, nil
	}
	mapping := selected.Mapping
	engine := m.engine
	ctx := m.ctx

	return m, func() tea.Msg {
		result := engine.RunDiff(ctx, mapping, bisync.SyncOptions{})
		return DiffResultMsg{
			MappingName: result.MappingName,
			Entries:     result.Entries,
			Error:       result.Error,
		}
	}
}

func (m AppModel) showNewMappingForm() (tea.Model, tea.Cmd) {
	form, keys := forms.NewMappingForm()
	overlay := components.NewFormOverlay("create-mapping", form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) showEncryptionForm(mappingName string) (tea.Model, tea.Cmd) {
	remotes, err := m.engine.ListRemotes(m.ctx)
	if err != nil {
		m.detailPanel.AppendLog("Error listing remotes: " + err.Error())
		return m, nil
	}
	if len(remotes) == 0 {
		m.detailPanel.AppendLog("No rclone remotes configured. Run 'rclone config' first.")
		return m, nil
	}

	form, keys := forms.NewEncryptionForm(mappingName, remotes)
	overlay := components.NewFormOverlay("setup-encryption-"+mappingName, form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) showEditMappingForm(mapping config.Mapping) (tea.Model, tea.Cmd) {
	form, keys := forms.NewEditMappingForm(forms.MappingValues{
		Name:            mapping.Name,
		Local:           mapping.Local,
		Remote:          mapping.Remote,
		FiltersFile:     mapping.FiltersFile,
		BandwidthLimit:  mapping.BandwidthLimit,
		ConflictResolve: mapping.ConflictResolve,
		BackupEnabled:   mapping.BackupEnabled,
	})
	overlay := components.NewFormOverlay("edit-mapping-"+mapping.Name, form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) showNewRemoteForm() (tea.Model, tea.Cmd) {
	form, keys := forms.NewRemoteForm()
	overlay := components.NewFormOverlay("create-remote", form, keys, m.width, m.height)
	m.formOverlay = &overlay
	return m, overlay.Init()
}

func (m AppModel) handleFormSubmit(msg components.FormSubmittedMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.ID == "create-mapping":
		return m.handleCreateMapping(msg.Values)
	case msg.ID == "create-profile":
		return m.handleCreateProfile(msg.Values)
	case msg.ID == "create-remote":
		return m.handleCreateRemote(msg.Values)
	case strings.HasPrefix(msg.ID, "edit-mapping-"):
		mappingName := strings.TrimPrefix(msg.ID, "edit-mapping-")
		return m.handleEditMapping(mappingName, msg.Values)
	case strings.HasPrefix(msg.ID, "setup-encryption-"):
		mappingName := strings.TrimPrefix(msg.ID, "setup-encryption-")
		return m.handleSetupEncryption(mappingName, msg.Values)
	}
	return m, nil
}

func (m AppModel) handleCreateMapping(values map[string]string) (tea.Model, tea.Cmd) {
	mapping := config.Mapping{
		Name:            values["name"],
		Local:           values["local"],
		Remote:          values["remote"],
		FiltersFile:     values["filters_file"],
		BandwidthLimit:  values["bandwidth_limit"],
		ConflictResolve: values["conflict_resolve"],
		BackupEnabled:   values["backup_enabled"] == "true",
	}
	if mapping.BackupEnabled {
		mapping.BackupRetention = 7
	}

	cfgPath := config.ProfilePath("")
	if err := config.AddMapping(cfgPath, mapping); err != nil {
		// Show error in detail panel
		m.detailPanel.AppendLog("Error adding mapping: " + err.Error())
		return m, nil
	}

	// Reload config and refresh UI
	cfg, err := config.LoadProfile("")
	if err != nil {
		m.detailPanel.AppendLog("Error reloading config: " + err.Error())
		return m, nil
	}
	m.config = cfg
	m.states = m.stateStore.LoadAll(cfg.Mappings)
	m.mappingList = components.NewMappingList(cfg.Mappings, m.states, m.mappingList.Width(), m.mappingList.Height())
	return m, nil
}

func (m AppModel) handleEditMapping(originalName string, values map[string]string) (tea.Model, tea.Cmd) {
	mapping := config.Mapping{
		Name:            values["name"],
		Local:           values["local"],
		Remote:          values["remote"],
		FiltersFile:     values["filters_file"],
		BandwidthLimit:  values["bandwidth_limit"],
		ConflictResolve: values["conflict_resolve"],
		BackupEnabled:   values["backup_enabled"] == "true",
	}

	// Preserve existing fields not in the form
	if orig := m.findMapping(originalName); orig != nil {
		mapping.Encryption = orig.Encryption
		if mapping.BackupEnabled && orig.BackupRetention > 0 {
			mapping.BackupRetention = orig.BackupRetention
		} else if mapping.BackupEnabled {
			mapping.BackupRetention = 7
		}
	}

	cfgPath := config.ProfilePath("")

	// If name changed, remove old and add new
	if originalName != mapping.Name {
		if err := config.RemoveMapping(cfgPath, originalName); err != nil {
			m.detailPanel.AppendLog("Error removing old mapping: " + err.Error())
			return m, nil
		}
		if err := config.AddMapping(cfgPath, mapping); err != nil {
			m.detailPanel.AppendLog("Error adding renamed mapping: " + err.Error())
			return m, nil
		}
	} else {
		if err := config.UpdateMapping(cfgPath, mapping); err != nil {
			m.detailPanel.AppendLog("Error updating mapping: " + err.Error())
			return m, nil
		}
	}

	return m.reloadConfig()
}

func (m AppModel) handleDeleteMapping(name string) (tea.Model, tea.Cmd) {
	cfgPath := config.ProfilePath("")
	if err := config.RemoveMapping(cfgPath, name); err != nil {
		m.detailPanel.AppendLog("Error deleting mapping: " + err.Error())
		return m, nil
	}
	return m.reloadConfig()
}

func (m AppModel) reloadConfig() (tea.Model, tea.Cmd) {
	cfg, err := config.LoadProfile("")
	if err != nil {
		m.detailPanel.AppendLog("Error reloading config: " + err.Error())
		return m, nil
	}
	m.config = cfg
	m.states = m.stateStore.LoadAll(cfg.Mappings)
	m.mappingList = components.NewMappingList(cfg.Mappings, m.states, m.mappingList.Width(), m.mappingList.Height())
	return m, nil
}

func (m AppModel) handleCreateRemote(values map[string]string) (tea.Model, tea.Cmd) {
	name := values["name"]
	remoteType := values["type"]

	// Parse extra params (key=value per line)
	params := make(map[string]string)
	for _, line := range strings.Split(values["params"], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			params[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	engine := m.engine
	ctx := m.ctx
	return m, func() tea.Msg {
		err := engine.CreateRemote(ctx, name, remoteType, params)
		return RemoteCreatedMsg{Err: err}
	}
}

func (m AppModel) handleSetupEncryption(mappingName string, values map[string]string) (tea.Model, tea.Cmd) {
	mapping := m.findMapping(mappingName)
	if mapping == nil {
		return m, nil
	}

	mapping.Encryption = config.EncryptionConfig{
		Enabled:     true,
		CryptRemote: values["crypt_remote"],
	}

	cfgPath := config.ProfilePath("")
	if err := config.UpdateMapping(cfgPath, *mapping); err != nil {
		m.detailPanel.AppendLog("Error saving encryption config: " + err.Error())
		return m, nil
	}

	// Reload config
	cfg, err := config.LoadProfile("")
	if err != nil {
		m.detailPanel.AppendLog("Error reloading config: " + err.Error())
		return m, nil
	}
	m.config = cfg
	m.detailPanel.AppendLog("Encryption enabled for " + mappingName + " using " + values["crypt_remote"])

	// Refresh detail panel
	for _, mp := range cfg.Mappings {
		if mp.Name == mappingName {
			ms := m.states[mappingName]
			m.detailPanel.SetMapping(&mp, ms)
			break
		}
	}

	return m, nil
}

func (m AppModel) handleCreateProfile(values map[string]string) (tea.Model, tea.Cmd) {
	name := values["name"]
	path := config.ProfilePath(name)
	if err := config.CreateDefaultConfig(path); err != nil {
		m.detailPanel.AppendLog("Error creating profile: " + err.Error())
		return m, nil
	}
	m.detailPanel.AppendLog("Created profile: " + name)
	return m, nil
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
	m.detailPanel.SetSize(innerRightW, innerH)
	m.remoteList.SetSize(innerLeftW, innerH)
	m.remoteDetail.SetSize(innerRightW, innerH)

	// Dashboard is full width
	fullW := m.width - 2 // border
	m.dashboard.SetSize(fullW, contentHeight-2)

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
	if m.showHelp {
		return m.renderHelp()
	}
	if m.formOverlay != nil {
		return m.formOverlay.View()
	}
	if m.modal != nil {
		return m.modal.View()
	}
	if m.actionMenu != nil {
		return m.actionMenu.View()
	}

	// Main layout
	titleBar := m.titleBar.View()
	statusBar := m.statusBar.View()

	var content string
	switch m.viewMode {
	case ViewMappings:
		content = lipgloss.JoinHorizontal(lipgloss.Top,
			m.mappingList.View(), m.detailPanel.View())
	case ViewRemotes:
		content = lipgloss.JoinHorizontal(lipgloss.Top,
			m.remoteList.View(), m.remoteDetail.View())
	case ViewDashboard:
		content = m.dashboard.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		content,
		statusBar,
	)
}

func (m AppModel) renderHelp() string {
	title := theme.ModalTitleStyle.Render("Keybindings")

	bindings := []struct{ key, desc string }{
		{"j/k, Up/Down", "Navigate mappings"},
		{"Enter", "Open actions menu"},
		{"Tab", "Switch panel focus"},
		{"h/l, Left/Right", "Switch detail tab (when detail focused)"},
		{"s", "Sync selected mapping"},
		{"S", "Sync all mappings"},
		{"d", "Dry-run (preview only)"},
		{"D", "Diff preview"},
		{"r", "Force resync"},
		{"l", "View logs (from list panel)"},
		{"i", "View info (from list panel)"},
		{"n", "New mapping"},
		{"Enter > E", "Edit mapping (via actions menu)"},
		{"Enter > X", "Delete mapping (via actions menu)"},
		{"R", "Remotes settings"},
		{"C / X", "Create / delete remote (on Remotes tab)"},
		{"?", "This help"},
		{"q / Ctrl+C", "Quit"},
		{"Esc", "Back / dismiss"},
	}

	var b strings.Builder
	b.WriteString(title + "\n\n")
	for _, bind := range bindings {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			theme.StatusKeyStyle.Render(fmt.Sprintf("%-20s", bind.key)),
			theme.StatusDescStyle.Render(bind.desc),
		))
	}
	b.WriteString("\n" + theme.StatusDescStyle.Render("  Press any key to dismiss"))

	content := theme.ModalStyle.Render(b.String())

	menuW := lipgloss.Width(content)
	menuH := lipgloss.Height(content)
	x := (m.width - menuW) / 2
	y := (m.height - menuH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	return lipgloss.NewStyle().
		MarginLeft(x).
		MarginTop(y).
		Render(content)
}
