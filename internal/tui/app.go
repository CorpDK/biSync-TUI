package tui

import (
	"context"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/logs"
	"github.com/CorpDK/bisync-tui/internal/notify"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
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
	showHelp         bool
}

// NewApp creates the root application model.
func NewApp(cfg *config.Config, stateStore *state.Store, engine *bisync.Engine, lockMgr *bisync.LockManager, version string) AppModel {
	ctx, cancel := context.WithCancel(context.Background())
	keys := DefaultKeyMap()
	states := stateStore.LoadAll(cfg.Mappings)

	historyStore := state.NewHistoryStore(filepath.Join(config.StateDir(), "history"), 500)
	logMgr := logs.NewLogManager(filepath.Join(config.StateDir(), "logs"))
	notifier := notify.NewNotifier(cfg.Global.Notifications)
	pool := bisync.NewPool(cfg.Global.MaxWorkers, engine, lockMgr, stateStore, historyStore, logMgr, notifier)
	pool.Start(ctx)

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
		statusBar:        components.NewStatusBar(keys.MappingsHelp(), 80),
		titleBar:         components.NewTitleBar(version, 80),
		states:           states,
		keys:             keys,
		version:          version,
		ctx:              ctx,
		cancelFunc:       cancel,
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
