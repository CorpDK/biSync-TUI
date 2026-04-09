package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check connectivity to all remotes",
	RunE:  runHealth,
}

func runHealth(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	engine, err := bisync.NewEngine()
	if err != nil {
		return err
	}

	remotes := bisync.ExtractRemotes(cfg.Mappings)
	if len(remotes) == 0 {
		fmt.Println("No remotes configured.")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "REMOTE\tSTATUS")
	fmt.Fprintln(w, "------\t------")

	for _, remote := range remotes {
		err := engine.CheckConnectivity(ctx, remote+":")
		status := "healthy"
		if err != nil {
			status = fmt.Sprintf("unreachable: %v", err)
		}
		fmt.Fprintf(w, "%s\t%s\n", remote, status)
	}

	return w.Flush()
}
