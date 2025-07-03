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
				"config_path":   a.config.Storage.Path,
				"storage_path":  a.config.Storage.Path,
				"spec_count":    len(specs),
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

// createBackupCommand creates the backup command
func (a *App) createBackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "backup <destination>",
		Short: "Backup the storage directory to a specified location",
		Long:  "Creates a backup of the ZAMM storage directory to the specified file path. The backup is a complete copy of all specs and links.",
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

			// Copy storage directory to backup location
			if err := copyDir(a.config.Storage.Path, backupPath); err != nil {
				return models.NewZammErrorWithCause(models.ErrTypeSystem, "failed to backup storage directory", err)
			}

			fmt.Printf("Storage directory successfully backed up to: %s\n", backupPath)
			return nil
		},
	}
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

// copyDir copies a directory from src to dst
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the relative path from src
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Create the destination path
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, info.Mode())
		} else {
			// Copy file
			return copyFile(path, destPath)
		}
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	_, err = dstFile.ReadFrom(srcFile)
	return err
}
