package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage config profiles",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available config profiles",
	RunE:  runProfileList,
}

var profileCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new config profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileCreate,
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileCreateCmd)
}

func runProfileList(cmd *cobra.Command, args []string) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}

	fmt.Println("Available profiles:")
	fmt.Println()

	// Default profile
	defaultPath := config.ProfilePath("")
	marker := ""
	if _, err := os.Stat(defaultPath); err == nil {
		marker = " (exists)"
	}
	fmt.Printf("  default%s\n", marker)

	// Scan profiles directory
	profilesDir := filepath.Join(config.ConfigDir(), "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		fmt.Printf("  %s\n", name)
	}

	return nil
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}

	name := args[0]
	path := config.ProfilePath(name)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("profile %q already exists at %s", name, path)
	}

	if err := config.CreateDefaultConfig(path); err != nil {
		return err
	}

	fmt.Printf("Created profile %q at %s\n", name, path)
	fmt.Println("Edit this file to add your mappings.")
	return nil
}
