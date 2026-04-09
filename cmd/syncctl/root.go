package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
	"github.com/CorpDK/bisync-tui/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "syncctl",
	Short: "A rich terminal dashboard for rclone bisync",
	Long:  "syncctl provides a lazygit-style interface for managing rclone bidirectional sync operations.",
	RunE:  runTUI,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("syncctl %s\n", version)
		},
	})
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Ensure directories exist
	if err := config.EnsureDirs(); err != nil {
		return fmt.Errorf("create dirs: %w", err)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w\n\nCreate %s with your mappings.\nSee README for TOML format.",
			err, filepath.Join(config.ConfigDir(), "settings.toml"))
	}

	// Validate config
	if errs := cfg.Validate(); len(errs) > 0 {
		fmt.Fprintln(os.Stderr, "Config warnings:")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}

	// Init engine
	engine, err := bisync.NewEngine()
	if err != nil {
		return err
	}

	// Init state store and lock manager
	stateStore := state.NewDefaultStore()
	lockMgr := bisync.NewLockManager(filepath.Join(config.CacheDir(), "locks"))

	// Launch TUI
	app := tui.NewApp(cfg, stateStore, engine, lockMgr, version)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
