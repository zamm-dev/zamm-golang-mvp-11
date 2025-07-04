package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourorg/zamm-mvp/internal/config"
)

// createInitCommand creates the init command
func (a *App) createInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize zamm in current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.WriteDefaultConfig(); err != nil {
				return err
			}

			// Initialize file-based storage
			if err := a.storage.InitializeStorage(); err != nil {
				return err
			}

			fmt.Println("Initialized zamm successfully")
			configPath, _ := config.GetConfigPath()
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Printf("Storage directory: %s\n", a.config.Storage.Path)
			return nil
		},
	}
}

// createStatusCommand creates the status command
func (a *App) createStatusCommand(jsonOutput bool) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status and statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			specs, err := a.specService.ListSpecs()
			if err != nil {
				return err
			}

			status := map[string]interface{}{
				"config_path":  a.config.Storage.Path,
				"storage_path": a.config.Storage.Path,
				"spec_count":   len(specs),
			}

			if jsonOutput {
				return a.outputJSON(status)
			}

			fmt.Printf("ZAMM Status\n")
			fmt.Printf("===========\n")
			fmt.Printf("Storage: %s\n", a.config.Storage.Path)
			fmt.Printf("Specifications: %d\n", len(specs))
			return nil
		},
	}
}

// createVersionCommand creates the version command
func (a *App) createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ZAMM MVP v0.1.0")
		},
	}
}
