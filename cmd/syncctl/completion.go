package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/CorpDK/bisync-tui/internal/config"
)

// completeMappingNames provides dynamic completion for --name flags.
func completeMappingNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, m := range cfg.Mappings {
		names = append(names, m.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

var completionCmd = &cobra.Command{
	Use:       "completion [bash|zsh|fish]",
	Short:     "Generate shell completion scripts",
	Long:      `Generate shell completion scripts for syncctl.`,
	ValidArgs: []string{"bash", "zsh", "fish"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}
