package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	"github.com/CorpDK/bisync-tui/internal/state"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all mappings",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
}

// StatusEntry is the JSON representation of a mapping's status.
type StatusEntry struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	LastSync    *string `json:"last_sync"`
	SyncCount   int     `json:"sync_count"`
	Duration    string  `json:"last_duration"`
	LastError   string  `json:"last_error,omitempty"`
	Initialized bool    `json:"initialized"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadProfile(profileName)
	if err != nil {
		return err
	}

	stateStore := state.NewDefaultStore()
	states := stateStore.LoadAll(cfg.Mappings)

	if statusJSON {
		return runStatusJSON(cfg, states)
	}
	return runStatusTable(cfg, states)
}

func runStatusJSON(cfg *config.Config, states map[string]*state.MappingState) error {
	entries := make([]StatusEntry, 0, len(cfg.Mappings))
	for _, m := range cfg.Mappings {
		ms := states[m.Name]
		entry := StatusEntry{
			Name:        m.Name,
			Status:      string(ms.LastStatus),
			SyncCount:   ms.SyncCount,
			Duration:    ms.LastDuration,
			LastError:   ms.LastError,
			Initialized: ms.Initialized,
		}
		if ms.LastSync != nil {
			ts := ms.LastSync.Format("2006-01-02T15:04:05Z07:00")
			entry.LastSync = &ts
		}
		entries = append(entries, entry)
	}

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func runStatusTable(cfg *config.Config, states map[string]*state.MappingState) error {
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
