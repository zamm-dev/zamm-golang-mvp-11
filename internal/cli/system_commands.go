package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yourorg/zamm-mvp/internal/config"
	"github.com/yourorg/zamm-mvp/internal/models"
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

			// Run database migrations
			if err := a.runMigrations(); err != nil {
				return err
			}

			fmt.Println("Initialized zamm successfully")
			configPath, _ := config.GetConfigPath()
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Printf("Database: %s\n", a.config.Database.Path)
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
				"config_path":   a.config.Database.Path,
				"database_path": a.config.Database.Path,
				"spec_count":    len(specs),
			}

			if jsonOutput {
				return a.outputJSON(status)
			}

			fmt.Printf("ZAMM Status\n")
			fmt.Printf("===========\n")
			fmt.Printf("Database: %s\n", a.config.Database.Path)
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

// runMigrations runs database migrations
func (a *App) runMigrations() error {
	// Read migration file
	migrationPath := "migrations/001_initial.sql"
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to read migration file", err)
	}

	// Execute migration using the storage interface
	if err := a.storage.RunMigration(string(migrationSQL)); err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to execute migration", err)
	}

	fmt.Printf("Migration executed successfully: %s\n", migrationPath)
	return nil
}
