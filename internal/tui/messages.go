package tui

import (
	"time"

	"github.com/CorpDK/bisync-tui/internal/logs"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
)

// SyncStartedMsg signals that a sync has begun.
type SyncStartedMsg struct {
	MappingName string
}

// SyncOutputMsg carries a line of output from a running sync.
type SyncOutputMsg struct {
	MappingName string
	Line        string
}

// SyncCompleteMsg signals that a sync has finished.
type SyncCompleteMsg struct {
	MappingName string
	Result      bisync.SyncResult
}

// RemoteSizeMsg carries the result of a remote size query.
type RemoteSizeMsg struct {
	MappingName string
	Size        string
	Err         error
}

// TickMsg is sent periodically for state refresh.
type TickMsg time.Time

// PoolOutputMsg wraps an output line from the worker pool.
type PoolOutputMsg bisync.OutputLine

// PoolResultMsg wraps a job result from the worker pool.
type PoolResultMsg bisync.JobResult

// StateRefreshMsg triggers a reload of mapping states.
type StateRefreshMsg struct{}

// HealthStatusMsg carries health check results for all remotes.
type HealthStatusMsg struct {
	Statuses map[string]*bisync.HealthStatus
}

// HistoryLoadedMsg carries loaded sync history for a mapping.
type HistoryLoadedMsg struct {
	MappingName string
	Records     []state.HistoryRecord
}

// AggregatedLogsMsg carries log entries across all mappings.
type AggregatedLogsMsg struct {
	Entries []logs.LogEntry
}

// RemoteAboutMsg carries storage space info for a remote.
type RemoteAboutMsg struct {
	Remote string
	About  *bisync.RemoteAbout
	Err    error
}

// ConflictsDetectedMsg carries parsed conflicts from sync output.
type ConflictsDetectedMsg struct {
	MappingName string
	Conflicts   []bisync.Conflict
}

// DiffResultMsg carries diff preview results for a mapping.
type DiffResultMsg struct {
	MappingName string
	Entries     []bisync.DiffEntry
	Error       string
}

// RemotesLoadedMsg carries loaded remote config info.
type RemotesLoadedMsg struct {
	Remotes []bisync.RemoteInfo
	Err     error
}

// RemoteDeletedMsg signals that a remote was deleted.
type RemoteDeletedMsg struct {
	Name string
	Err  error
}

// RemoteCreatedMsg signals that a remote was created.
type RemoteCreatedMsg struct {
	Err error
}

// RemoteTestMsg carries the result of a remote connectivity test.
type RemoteTestMsg struct {
	Name string
	Err  error
}