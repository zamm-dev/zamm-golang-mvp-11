package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/zamm-mvp/internal/config"
)

// createInitCommand creates the init command
func (a *App) createInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize zamm in current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if .zamm directory already exists
			if _, err := os.Stat(a.config.Storage.Path); err == nil {
				fmt.Printf("ZAMM is already initialized in %s\n", a.config.Storage.Path)
				return nil
			}

			if err := config.WriteDefaultConfig(); err != nil {
				return err
			}

			// Perform complete initialization
			if err := a.InitializeZamm(); err != nil {
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
				// If storage doesn't exist, show uninitialized status
				if jsonOutput {
					status := map[string]interface{}{
						"config_path":  a.config.Storage.Path,
						"storage_path": a.config.Storage.Path,
						"spec_count":   0,
						"initialized":  false,
						"error":        err.Error(),
					}
					return a.outputJSON(status)
				}

				fmt.Printf("ZAMM Status\n")
				fmt.Printf("===========\n")
				fmt.Printf("Storage: %s (not initialized)\n", a.config.Storage.Path)
				fmt.Printf("Specifications: 0\n")
				fmt.Printf("Error: %s\n", err.Error())
				return nil
			}

			status := map[string]interface{}{
				"config_path":  a.config.Storage.Path,
				"storage_path": a.config.Storage.Path,
				"spec_count":   len(specs),
				"initialized":  true,
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

// createMigrateCommand creates the generic migration command
func (a *App) createMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database/data migrations (e.g., add missing fields)",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationsRun := 0

			// Run specs-to-nodes migration
			if err := a.migrateSpecsToNodes(); err != nil {
				// If specs directory doesn't exist, that's fine - migration already done
				if !strings.Contains(err.Error(), "specs directory does not exist") {
					return err
				}
			} else {
				fmt.Printf("[specs-to-nodes] Migration complete. Renamed specs folder to nodes.\n")
				migrationsRun++
			}

			if migrationsRun == 0 {
				fmt.Println("All migrations are up to date.")
			}
			return nil
		},
	}
}

// migrateSpecsToNodes renames the specs folder to nodes folder
func (a *App) migrateSpecsToNodes() error {
	specsDir := filepath.Join(a.config.Storage.Path, "specs")
	nodesDir := filepath.Join(a.config.Storage.Path, "nodes")

	// Check if specs directory exists
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		return fmt.Errorf("specs directory does not exist: %s", specsDir)
	}

	// Check if nodes directory already exists
	if _, err := os.Stat(nodesDir); err == nil {
		return fmt.Errorf("nodes directory already exists: %s", nodesDir)
	}

	// Rename specs directory to nodes
	if err := os.Rename(specsDir, nodesDir); err != nil {
		return fmt.Errorf("failed to rename specs directory to nodes: %w", err)
	}

	return nil
}
