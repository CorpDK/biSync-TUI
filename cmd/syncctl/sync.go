package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
)

var (
	syncAll    bool
	syncName   string
	syncDryRun bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run sync without TUI",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().BoolVarP(&syncAll, "all", "a", false, "Sync all initialized mappings")
	syncCmd.Flags().StringVarP(&syncName, "name", "n", "", "Sync a specific mapping by name")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "d", false, "Perform a dry run")
	syncCmd.RegisterFlagCompletionFunc("name", completeMappingNames)
}

func runSync(cmd *cobra.Command, args []string) error {
	if !syncAll && syncName == "" {
		return fmt.Errorf("specify --all or --name <mapping>")
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}

	cfg, err := config.LoadProfile(profileName)
	if err != nil {
		return err
	}

	engine, err := bisync.NewEngine()
	if err != nil {
		return err
	}

	stateStore := state.NewDefaultStore()
	lockMgr := bisync.NewLockManager(filepath.Join(config.CacheDir(), "locks"))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	opts := bisync.SyncOptions{DryRun: syncDryRun}

	var mappings []config.Mapping
	if syncAll {
		for _, m := range cfg.Mappings {
			ms, _ := stateStore.Load(m.Name)
			if ms.Initialized {
				mappings = append(mappings, m)
			}
		}
	} else {
		for _, m := range cfg.Mappings {
			if m.Name == syncName {
				mappings = append(mappings, m)
				break
			}
		}
		if len(mappings) == 0 {
			return fmt.Errorf("mapping %q not found", syncName)
		}
	}

	for _, m := range mappings {
		fmt.Printf("Syncing %s...\n", m.Name)

		lock, err := lockMgr.Acquire(m.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Skip %s: %v\n", m.Name, err)
			continue
		}

		outputCh := make(chan string, 128)
		go func() {
			for line := range outputCh {
				fmt.Printf("  %s\n", line)
			}
		}()

		// Merge per-mapping config into options
		mOpts := opts
		if mOpts.FiltersFile == "" {
			mOpts.FiltersFile = m.FiltersFile
		}
		if mOpts.BandwidthLimit == "" {
			mOpts.BandwidthLimit = m.BandwidthLimit
		}
		if mOpts.ConflictResolve == "" {
			mOpts.ConflictResolve = m.ConflictResolve
		}
		mOpts.ExtraFlags = append(mOpts.ExtraFlags, m.ExtraFlags...)

		result := engine.RunSync(ctx, m, mOpts, outputCh)
		close(outputCh)
		lockMgr.Release(lock)

		if result.Success {
			fmt.Printf("  Done (%s)\n", result.Duration.Truncate(1e9))
		} else {
			fmt.Fprintf(os.Stderr, "  Failed: %s\n", result.ErrorMsg)
		}
	}

	return nil
}
