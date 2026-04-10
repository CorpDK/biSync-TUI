package sync

import (
	"context"
	gosync "sync"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// RunParallelDiffs runs dry-run diffs for multiple mappings in parallel.
// maxWorkers limits concurrency. Results are returned in mapping order.
func RunParallelDiffs(ctx context.Context, engine *Engine, mappings []config.Mapping, maxWorkers int) []DiffResult {
	if maxWorkers <= 0 {
		maxWorkers = 3
	}

	results := make([]DiffResult, len(mappings))
	sem := make(chan struct{}, maxWorkers)
	var wg gosync.WaitGroup

	for i, m := range mappings {
		wg.Add(1)
		go func(idx int, mapping config.Mapping) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			results[idx] = engine.RunDiff(ctx, mapping, SyncOptions{})
		}(i, m)
	}

	wg.Wait()
	return results
}
