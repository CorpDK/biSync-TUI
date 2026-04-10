package tui

import (
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui/components"
)

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
	m.detailPanel.SetMode(components.DetailLogs)
	m.detailPanel.ClearLogs()
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
	m.syncPool.Submit(bisync.Job{Mapping: mapping, Options: bisync.SyncOptions{Resync: true}})
	return m, nil
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

func (m AppModel) fetchRemoteSize(mapping config.Mapping) tea.Cmd {
	return func() tea.Msg {
		about, err := m.engine.GetRemoteAbout(m.ctx, mapping.Remote)
		return RemoteAboutMsg{Remote: mapping.Remote, About: about, Err: err}
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

// launchRcloneConfig suspends the TUI and runs rclone config interactively.
func (m AppModel) launchRcloneConfig() (tea.Model, tea.Cmd) {
	c := exec.Command(m.engine.RclonePath(), "config")
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return RcloneConfigDoneMsg{Err: err}
	})
}

func (m AppModel) deleteRemote(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.engine.DeleteRemote(m.ctx, name)
		return RemoteDeletedMsg{Name: name, Err: err}
	}
}

// testAllRemotes tests connectivity to all configured remotes in parallel.
func (m AppModel) testAllRemotes() tea.Cmd {
	engine := m.engine
	ctx := m.ctx
	return func() tea.Msg {
		remotes, err := engine.ListRemotes(ctx)
		if err != nil {
			return AllRemotesTestedMsg{}
		}

		results := make([]components.RemoteHealth, len(remotes))
		var wg sync.WaitGroup
		for i, remote := range remotes {
			wg.Add(1)
			go func(idx int, r string) {
				defer wg.Done()
				rh := components.RemoteHealth{Name: strings.TrimSuffix(r, ":")}
				err := engine.CheckConnectivity(ctx, r)
				if err != nil {
					rh.Error = err.Error()
				} else {
					rh.Healthy = true
				}
				results[idx] = rh
			}(i, remote)
		}
		wg.Wait()
		return AllRemotesTestedMsg{Results: results}
	}
}

func (m AppModel) testRemoteConnection(name string) tea.Cmd {
	engine := m.engine
	ctx := m.ctx
	return func() tea.Msg {
		err := engine.CheckConnectivity(ctx, name+":")
		return RemoteTestMsg{Name: name, Err: err}
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
