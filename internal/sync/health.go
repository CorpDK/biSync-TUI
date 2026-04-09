package sync

import (
	"context"
	"strings"
	gosync "sync"
	"time"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// HealthStatus tracks the health of a single remote.
type HealthStatus struct {
	Remote    string
	Healthy   bool
	LastCheck time.Time
	Error     string
}

// HealthChecker periodically checks connectivity to all remotes.
type HealthChecker struct {
	engine   *Engine
	interval time.Duration
	statuses map[string]*HealthStatus
	mu       gosync.RWMutex
	updates  chan map[string]*HealthStatus
}

// NewHealthChecker creates a health checker for the given mappings.
func NewHealthChecker(engine *Engine, mappings []config.Mapping, interval time.Duration) *HealthChecker {
	remotes := ExtractRemotes(mappings)
	statuses := make(map[string]*HealthStatus, len(remotes))
	for _, r := range remotes {
		statuses[r] = &HealthStatus{Remote: r}
	}
	return &HealthChecker{
		engine:   engine,
		interval: interval,
		statuses: statuses,
		updates:  make(chan map[string]*HealthStatus, 1),
	}
}

// Start begins periodic health checks. Blocks until context is cancelled.
func (h *HealthChecker) Start(ctx context.Context) {
	h.checkAll(ctx)
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkAll(ctx)
		}
	}
}

// Updates returns the channel that receives health status updates.
func (h *HealthChecker) Updates() <-chan map[string]*HealthStatus {
	return h.updates
}

// Statuses returns a snapshot of current health statuses.
func (h *HealthChecker) Statuses() map[string]*HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	cp := make(map[string]*HealthStatus, len(h.statuses))
	for k, v := range h.statuses {
		s := *v
		cp[k] = &s
	}
	return cp
}

func (h *HealthChecker) checkAll(ctx context.Context) {
	h.mu.Lock()
	for remote, status := range h.statuses {
		checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := h.engine.CheckConnectivity(checkCtx, remote+":")
		cancel()

		status.LastCheck = time.Now()
		if err != nil {
			status.Healthy = false
			status.Error = err.Error()
		} else {
			status.Healthy = true
			status.Error = ""
		}
	}
	h.mu.Unlock()

	// Send update (non-blocking)
	snapshot := h.Statuses()
	select {
	case h.updates <- snapshot:
	default:
	}
}

// ExtractRemotes returns deduplicated remote names from mappings.
func ExtractRemotes(mappings []config.Mapping) []string {
	seen := make(map[string]bool)
	var remotes []string
	for _, m := range mappings {
		for _, path := range []string{m.Local, m.Remote} {
			if parts := strings.SplitN(path, ":", 2); len(parts) == 2 && parts[0] != "" {
				remote := parts[0]
				if !seen[remote] {
					seen[remote] = true
					remotes = append(remotes, remote)
				}
			}
		}
	}
	return remotes
}
