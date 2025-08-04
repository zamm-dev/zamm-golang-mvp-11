package cli

import (
	"github.com/spf13/cobra"
)

// CreateRootCommand creates the root command for the CLI
func (a *App) CreateRootCommand() *cobra.Command {
	var jsonOutput bool
	var quiet bool

	rootCmd := &cobra.Command{
		Use:   "zamm",
		Short: "ZAMM - Zen and the Automation of Metaprogramming for the Masses",
		Long:  "ZAMM is a tool for linking specifications to Git commits, enabling traceability between requirements and implementation.",
	}

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Quiet output")

	// Add subcommands
	rootCmd.AddCommand(a.createSpecCommand(jsonOutput, quiet))
	rootCmd.AddCommand(a.createLinkCommand(jsonOutput, quiet))
	rootCmd.AddCommand(a.createInitCommand())
	rootCmd.AddCommand(a.createStatusCommand(jsonOutput))
	rootCmd.AddCommand(a.createVersionCommand())
	rootCmd.AddCommand(a.createInteractiveCommand())
	rootCmd.AddCommand(a.createMigrateCommand())

	return rootCmd
}
