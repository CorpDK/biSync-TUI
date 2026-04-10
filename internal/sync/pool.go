package sync

import (
	"context"
	"sync"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/logs"
	"github.com/CorpDK/bisync-tui/internal/notify"
	"github.com/CorpDK/bisync-tui/internal/state"
)

// Job represents a sync operation to be executed.
type Job struct {
	Mapping config.Mapping
	Options SyncOptions
}

// JobResult holds the result of a completed job.
type JobResult struct {
	MappingName string
	Result      SyncResult
}

// OutputLine represents a line of output from a running job.
type OutputLine struct {
	MappingName string
	Line        string
}

// Pool manages concurrent sync workers.
type Pool struct {
	maxWorkers   int
	engine       *Engine
	lockMgr      *LockManager
	stateStore   *state.Store
	historyStore *state.HistoryStore
	logMgr       *logs.LogManager
	notifier     *notify.Notifier

	jobs    chan Job
	results chan JobResult
	output  chan OutputLine
	wg      sync.WaitGroup
}

// NewPool creates a worker pool with the given concurrency limit.
func NewPool(maxWorkers int, engine *Engine, lockMgr *LockManager, stateStore *state.Store, historyStore *state.HistoryStore, logMgr *logs.LogManager, notifier *notify.Notifier) *Pool {
	return &Pool{
		maxWorkers:   maxWorkers,
		engine:       engine,
		lockMgr:      lockMgr,
		stateStore:   stateStore,
		historyStore: historyStore,
		logMgr:       logMgr,
		notifier:     notifier,
		jobs:         make(chan Job, 64),
		results:      make(chan JobResult, 64),
		output:       make(chan OutputLine, 256),
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

func (p *Pool) Submit(job Job)              { p.jobs <- job }
func (p *Pool) Results() <-chan JobResult    { return p.results }
func (p *Pool) Output() <-chan OutputLine    { return p.output }

func (p *Pool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
	close(p.output)
}

func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()
	for job := range p.jobs {
		if ctx.Err() != nil {
			return
		}
		p.executeJob(ctx, job)
	}
}

func (p *Pool) executeJob(ctx context.Context, job Job) {
	name := job.Mapping.Name

	lock, err := p.lockMgr.Acquire(name)
	if err != nil {
		p.results <- JobResult{MappingName: name, Result: SyncResult{ErrorMsg: err.Error()}}
		return
	}
	defer p.lockMgr.Release(lock)

	opts := p.mergeOptions(job)
	p.markSyncing(name)
	outputCh := p.startOutputForwarder(name)

	if job.Mapping.BackupEnabled {
		bm := NewBackupManager(p.engine)
		opts.ExtraFlags = append(opts.ExtraFlags, bm.BuildBackupFlags(job.Mapping)...)
	}

	result := p.engine.RunSync(ctx, job.Mapping, opts, outputCh)
	close(outputCh)

	p.cleanupBackups(ctx, job.Mapping, result.Success)
	p.updateState(name, result)
	p.recordHistory(name, result)
	p.sendNotification(name, result)

	p.results <- JobResult{MappingName: name, Result: result}
}

func (p *Pool) mergeOptions(job Job) SyncOptions {
	opts := job.Options
	if opts.FiltersFile == "" {
		opts.FiltersFile = job.Mapping.FiltersFile
	}
	if opts.BandwidthLimit == "" {
		opts.BandwidthLimit = job.Mapping.BandwidthLimit
	}
	if opts.ConflictResolve == "" {
		opts.ConflictResolve = job.Mapping.ConflictResolve
	}
	opts.ExtraFlags = append(opts.ExtraFlags, job.Mapping.ExtraFlags...)
	return opts
}

func (p *Pool) markSyncing(name string) {
	ms, _ := p.stateStore.Load(name)
	ms.LastStatus = state.StatusSyncing
	p.stateStore.Save(name, ms)
}

func (p *Pool) startOutputForwarder(name string) chan string {
	outputCh := make(chan string, 128)
	go func() {
		for line := range outputCh {
			p.output <- OutputLine{MappingName: name, Line: line}
			if p.logMgr != nil {
				p.logMgr.Write(name, line)
			}
		}
	}()
	return outputCh
}

func (p *Pool) cleanupBackups(ctx context.Context, mapping config.Mapping, success bool) {
	if success && mapping.BackupEnabled {
		bm := NewBackupManager(p.engine)
		go bm.CleanupOldBackups(ctx, mapping)
	}
}

func (p *Pool) updateState(name string, result SyncResult) {
	ms, _ := p.stateStore.Load(name)
	now := time.Now()
	ms.LastSync = &now
	ms.LastDuration = result.Duration.Truncate(time.Second).String()
	ms.SyncCount++
	if result.Success {
		ms.LastStatus = state.StatusIdle
		ms.LastError = ""
	} else {
		ms.LastStatus = state.StatusError
		ms.LastError = result.ErrorMsg
	}
	p.stateStore.Save(name, ms)
}

func (p *Pool) recordHistory(name string, result SyncResult) {
	if p.historyStore == nil {
		return
	}
	status, errMsg := "success", ""
	if !result.Success {
		status, errMsg = "error", result.ErrorMsg
	}
	files, bytes := ParseTransferSummary(result.Output)
	p.historyStore.Append(name, state.HistoryRecord{
		Timestamp:        time.Now(),
		Duration:         result.Duration,
		Status:           status,
		FilesTransferred: files,
		BytesTransferred: bytes,
		Error:            errMsg,
	})
}

func (p *Pool) sendNotification(name string, result SyncResult) {
	if p.notifier != nil {
		p.notifier.NotifySyncResult(name, result.Success, result.Duration, result.ErrorMsg)
	}
}
