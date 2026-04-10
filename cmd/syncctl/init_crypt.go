package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
	bisync "github.com/CorpDK/bisync-tui/internal/sync"
)

var initCryptName string

var initCryptCmd = &cobra.Command{
	Use:   "init-crypt",
	Short: "Set up encryption for a mapping",
	Long: `Guides you through configuring a rclone crypt remote for encrypted sync.

If you already have a crypt remote configured in rclone, this command
will validate it and update your syncctl config.`,
	RunE: runInitCrypt,
}

func init() {
	initCryptCmd.Flags().StringVarP(&initCryptName, "name", "n", "", "Mapping name to configure encryption for (required)")
	initCryptCmd.MarkFlagRequired("name")
	initCryptCmd.RegisterFlagCompletionFunc("name", completeMappingNames)
}

func runInitCrypt(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadProfile(profileName)
	if err != nil {
		return err
	}

	var mapping *config.Mapping
	for _, m := range cfg.Mappings {
		if m.Name == initCryptName {
			mapping = &m
			break
		}
	}
	if mapping == nil {
		return fmt.Errorf("mapping %q not found", initCryptName)
	}

	engine, err := bisync.NewEngine()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List available remotes
	remotes, err := engine.ListRemotes(ctx)
	if err != nil {
		return fmt.Errorf("listing remotes: %w", err)
	}

	// Find crypt remotes
	fmt.Println("Available rclone remotes:")
	fmt.Println()
	cryptRemotes := []string{}
	for _, r := range remotes {
		isCrypt, _ := engine.IsCryptRemote(ctx, r[:len(r)-1])
		marker := ""
		if isCrypt {
			marker = " (crypt)"
			cryptRemotes = append(cryptRemotes, r)
		}
		fmt.Printf("  %s%s\n", r, marker)
	}
	fmt.Println()

	if len(cryptRemotes) == 0 {
		printCryptSetupGuide()
		return nil
	}

	fmt.Printf("Found %d crypt remote(s). To use one, update your config:\n\n", len(cryptRemotes))
	fmt.Printf("  [mapping.%s.encryption]\n", mapping.Name)
	fmt.Printf("  enabled = true\n")
	fmt.Printf("  crypt_remote = %q\n", cryptRemotes[0])
	fmt.Println()
	fmt.Println("Or use the TUI (Enter → Encryption setup) for interactive configuration.")

	return nil
}

func printCryptSetupGuide() {
	fmt.Println("No crypt remotes found. To create one:")
	fmt.Println()
	fmt.Println("  1. Run: rclone config")
	fmt.Println("  2. Choose 'n' for new remote")
	fmt.Println("  3. Name it (e.g., 'mycrypt')")
	fmt.Println("  4. Choose 'crypt' as the type")
	fmt.Println("  5. Set the underlying remote (e.g., 'gdrive:encrypted')")
	fmt.Println("  6. Configure passwords when prompted")
	fmt.Println()
	fmt.Println("Then run this command again to validate and configure.")
}
