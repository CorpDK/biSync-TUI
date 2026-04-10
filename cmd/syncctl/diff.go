package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
)

var (
	diffAll  bool
	diffName string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Preview changes before syncing (dry-run)",
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().BoolVarP(&diffAll, "all", "a", false, "Diff all initialized mappings")
	diffCmd.Flags().StringVarP(&diffName, "name", "n", "", "Diff a specific mapping by name")
	diffCmd.RegisterFlagCompletionFunc("name", completeMappingNames)
}

// Styles for colored diff output.
var (
	diffAddStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	diffDelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	diffModStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
)

func runDiff(cmd *cobra.Command, args []string) error {
	if !diffAll && diffName == "" {
		return fmt.Errorf("specify --all or --name <mapping>")
	}

	cfg, err := config.LoadProfile(profileName)
	if err != nil {
		return err
	}

	engine, err := bisync.NewEngine()
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var mappings []config.Mapping
	if diffAll {
		stateStore := state.NewDefaultStore()
		for _, m := range cfg.Mappings {
			ms, _ := stateStore.Load(m.Name)
			if ms.Initialized {
				mappings = append(mappings, m)
			}
		}
	} else {
		for _, m := range cfg.Mappings {
			if m.Name == diffName {
				mappings = append(mappings, m)
				break
			}
		}
		if len(mappings) == 0 {
			return fmt.Errorf("mapping %q not found", diffName)
		}
	}

	results := bisync.RunParallelDiffs(ctx, engine, mappings, cfg.Global.MaxWorkers)

	for _, r := range results {
		if len(results) > 1 {
			fmt.Printf("\n=== %s ===\n", r.MappingName)
		}

		if r.Error != "" {
			fmt.Fprintf(os.Stderr, "  Error: %s\n", r.Error)
			continue
		}

		if len(r.Entries) == 0 {
			fmt.Println("  No changes detected.")
			continue
		}

		printDiffEntries(r.Entries)
		fmt.Printf("  (%d changes, took %s)\n", len(r.Entries), r.Duration.Truncate(1e8))
	}

	return nil
}

func printDiffEntries(entries []bisync.DiffEntry) {
	for _, e := range entries {
		var prefix, styled string
		switch e.Type {
		case bisync.DiffAdded:
			prefix = "+"
			styled = diffAddStyle.Render(fmt.Sprintf("  %s [%s] %s", prefix, e.Side, e.Path))
		case bisync.DiffDeleted:
			prefix = "-"
			styled = diffDelStyle.Render(fmt.Sprintf("  %s [%s] %s", prefix, e.Side, e.Path))
		case bisync.DiffModified:
			prefix = "~"
			styled = diffModStyle.Render(fmt.Sprintf("  %s [%s] %s", prefix, e.Side, e.Path))
		}
		fmt.Println(styled)
	}
}
