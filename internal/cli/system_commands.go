package cli

import (
	"fmt"
	"os"
	"path/filepath"

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

// createBackupCommand creates the backup command
func (a *App) createBackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup <destination>",
		Short: "Backup the database to a specified location",
		Long:  "Creates a backup of the ZAMM database to the specified file path. The backup is a complete copy of the database that can be restored later.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupPath := args[0]

			// Expand relative paths and handle ~ for home directory
			if !filepath.IsAbs(backupPath) {
				cwd, err := os.Getwd()
				if err != nil {
					return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to get current directory", err)
				}
				backupPath = filepath.Join(cwd, backupPath)
			}

			// Create directory if it doesn't exist
			backupDir := filepath.Dir(backupPath)
			if err := os.MkdirAll(backupDir, 0755); err != nil {
				return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to create backup directory", err)
			}

			// Check if file already exists and warn user
			if _, err := os.Stat(backupPath); err == nil {
				fmt.Printf("Warning: File %s already exists and will be overwritten.\n", backupPath)
			}

			// Perform the backup
			if err := a.storage.BackupDatabase(backupPath); err != nil {
				return err
			}

			// Get file info to show backup size
			fileInfo, err := os.Stat(backupPath)
			if err != nil {
				fmt.Printf("Database successfully backed up to: %s\n", backupPath)
			} else {
				size := fileInfo.Size()
				sizeStr := formatBytes(size)
				fmt.Printf("Database successfully backed up to: %s (%s)\n", backupPath, sizeStr)
			}

			return nil
		},
	}
}

// createMigrationCommand creates the migration management command
func (a *App) createMigrationCommand() *cobra.Command {
	migrationCmd := &cobra.Command{
		Use:   "migration",
		Short: "Manage database migrations",
		Long:  "Commands to manage database migrations including status and force operations",
	}

	// Migration status subcommand
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show current migration status",
		Long:  "Display the current migration version and whether the database is in a dirty state",
		RunE: func(cmd *cobra.Command, args []string) error {
			version, dirty, err := a.storage.GetMigrationVersion()
			if err != nil {
				return err
			}

			if version == 0 {
				fmt.Println("No migrations have been applied")
			} else {
				fmt.Printf("Current migration version: %d\n", version)
			}

			if dirty {
				fmt.Println("Database is in dirty state - manual intervention may be required")
			} else {
				fmt.Println("Database is clean")
			}

			return nil
		},
	}

	// Migration force subcommand
	forceCmd := &cobra.Command{
		Use:   "force <version>",
		Short: "Force migration version (for recovery)",
		Long:  "Force the migration version to a specific value. Use with caution - this is for recovery purposes only.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := uint(0)
			if _, err := fmt.Sscanf(args[0], "%d", &version); err != nil {
				return fmt.Errorf("invalid version number: %s", args[0])
			}

			fmt.Printf("Forcing migration version to %d...\n", version)
			if err := a.storage.ForceMigrationVersion(version); err != nil {
				return err
			}

			fmt.Println("Migration version forced successfully")
			return nil
		},
	}

	// Migration up subcommand
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Run pending migrations",
		Long:  "Check for and run any pending database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.storage.RunMigrationsIfNeeded()
		},
	}

	migrationCmd.AddCommand(statusCmd, forceCmd, upCmd)
	return migrationCmd
}

// formatBytes formats byte count as human readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// runMigrations runs database migrations
func (a *App) runMigrations() error {
	// Use the new migration system
	if err := a.storage.RunMigrationsIfNeeded(); err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to run migrations", err)
	}

	return nil
}
