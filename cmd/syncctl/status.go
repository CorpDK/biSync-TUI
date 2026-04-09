package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all mappings",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	stateStore := state.NewDefaultStore()
	states := stateStore.LoadAll(cfg.Mappings)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tLAST SYNC\tSYNCS\tDURATION")
	fmt.Fprintln(w, "----\t------\t---------\t-----\t--------")

	for _, m := range cfg.Mappings {
		ms := states[m.Name]
		lastSync := "never"
		if ms.LastSync != nil {
			lastSync = ms.LastSync.Format("2006-01-02 15:04")
		}
		duration := "-"
		if ms.LastDuration != "" {
			duration = ms.LastDuration
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
			m.Name, ms.LastStatus, lastSync, ms.SyncCount, duration)
	}

	return w.Flush()
}
