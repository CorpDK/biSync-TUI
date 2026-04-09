package main

import (
	"context"
	"fmt"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
)

var initName string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize (bootstrap) a mapping",
	Long:  "Pulls remote content to local, then establishes the bisync baseline with --resync.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVarP(&initName, "name", "n", "", "Mapping name to initialize (required)")
	initCmd.MarkFlagRequired("name")
	initCmd.RegisterFlagCompletionFunc("name", completeMappingNames)
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var mapping *config.Mapping
	for _, m := range cfg.Mappings {
		if m.Name == initName {
			mapping = &m
			break
		}
	}
	if mapping == nil {
		return fmt.Errorf("mapping %q not found", initName)
	}

	engine, err := bisync.NewEngine()
	if err != nil {
		return err
	}

	stateStore := state.NewDefaultStore()
	lockMgr := bisync.NewLockManager(filepath.Join(config.CacheDir(), "locks"))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lock, err := lockMgr.Acquire(mapping.Name)
	if err != nil {
		return err
	}
	defer lockMgr.Release(lock)

	outputCh := make(chan string, 128)
	go func() {
		for line := range outputCh {
			fmt.Printf("  %s\n", line)
		}
	}()

	// Step 1: Copy remote → local (skip if both sides are remote)
	if !config.IsRemotePath(mapping.Local) {
		fmt.Printf("Pulling remote → local for %q...\n", mapping.Name)
		result := engine.RunCopy(ctx, mapping.Remote, mapping.Local, outputCh)
		if !result.Success {
			close(outputCh)
			return fmt.Errorf("copy failed: %s", result.ErrorMsg)
		}
	} else {
		fmt.Printf("Both paths are remote for %q, skipping initial copy...\n", mapping.Name)
	}

	// Step 2: Establish baseline
	fmt.Printf("Establishing bisync baseline...\n")
	result := engine.RunSync(ctx, *mapping, bisync.SyncOptions{Resync: true}, outputCh)
	close(outputCh)

	if !result.Success {
		return fmt.Errorf("resync failed: %s", result.ErrorMsg)
	}

	// Update state
	ms, _ := stateStore.Load(mapping.Name)
	ms.Initialized = true
	ms.LastStatus = state.StatusIdle
	stateStore.Save(mapping.Name, ms)

	fmt.Printf("Initialized %q successfully.\n", mapping.Name)
	return nil
}
