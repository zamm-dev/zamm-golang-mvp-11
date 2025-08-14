package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/config"
	"gopkg.in/yaml.v3"
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
			nodes, err := a.specService.ListNodes()
			if err != nil {
				// If storage doesn't exist, show uninitialized status
				if jsonOutput {
					status := map[string]interface{}{
						"config_path":  a.config.Storage.Path,
						"storage_path": a.config.Storage.Path,
						"node_count":   0,
						"initialized":  false,
						"error":        err.Error(),
					}
					return a.outputJSON(status)
				}

				fmt.Printf("ZAMM Status\n")
				fmt.Printf("===========\n")
				fmt.Printf("Storage: %s (not initialized)\n", a.config.Storage.Path)
				fmt.Printf("Nodes: 0\n")
				fmt.Printf("Error: %s\n", err.Error())
				return nil
			}

			status := map[string]interface{}{
				"config_path":  a.config.Storage.Path,
				"storage_path": a.config.Storage.Path,
				"node_count":   len(nodes),
				"initialized":  true,
			}

			if jsonOutput {
				return a.outputJSON(status)
			}

			fmt.Printf("ZAMM Status\n")
			fmt.Printf("===========\n")
			fmt.Printf("Storage: %s\n", a.config.Storage.Path)
			fmt.Printf("Nodes: %d\n", len(nodes))
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
		Short: "Convert JSON node files to Markdown files with YAML frontmatter",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationsRun := 0

			// Run JSON-to-Markdown migration
			if err := a.migrateSpecsToNodes(); err != nil {
				// If nodes directory doesn't exist, that's fine - migration already done
				if !strings.Contains(err.Error(), "nodes directory does not exist") &&
					!strings.Contains(err.Error(), "no JSON files found to migrate") {
					return err
				}
			} else {
				fmt.Printf("[json-to-markdown] Migration complete. Converted JSON files to Markdown.\n")
				migrationsRun++
			}

			if migrationsRun == 0 {
				fmt.Println("All migrations are up to date.")
			}
			return nil
		},
	}
}

// migrateSpecsToNodes converts JSON node files to Markdown files with YAML frontmatter
func (a *App) migrateSpecsToNodes() error {
	nodesDir := filepath.Join(a.config.Storage.Path, "nodes")

	// Check if nodes directory exists
	if _, err := os.Stat(nodesDir); os.IsNotExist(err) {
		return fmt.Errorf("nodes directory does not exist: %s", nodesDir)
	}

	// Read all JSON files in nodes directory
	entries, err := os.ReadDir(nodesDir)
	if err != nil {
		return fmt.Errorf("failed to read nodes directory: %w", err)
	}

	migratedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		jsonPath := filepath.Join(nodesDir, entry.Name())
		mdPath := strings.TrimSuffix(jsonPath, ".json") + ".md"

		// Skip if markdown file already exists
		if _, err := os.Stat(mdPath); err == nil {
			continue
		}

		if err := a.convertJSONToMarkdown(jsonPath, mdPath); err != nil {
			return fmt.Errorf("failed to convert %s: %w", entry.Name(), err)
		}

		migratedCount++
	}

	if migratedCount == 0 {
		return fmt.Errorf("no JSON files found to migrate")
	}

	return nil
}

func (a *App) convertJSONToMarkdown(jsonPath, mdPath string) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var nodeData map[string]interface{}
	if err := json.Unmarshal(data, &nodeData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	content, hasContent := nodeData["content"].(string)
	if !hasContent {
		content = ""
	}

	// Create frontmatter map with all fields except content
	frontmatter := make(map[string]interface{})
	for key, value := range nodeData {
		if key != "content" {
			frontmatter[key] = value
		}
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML frontmatter: %w", err)
	}

	var mdContent strings.Builder
	mdContent.WriteString("---\n")
	mdContent.Write(yamlData)
	mdContent.WriteString("---\n")
	if content != "" {
		mdContent.WriteString("\n")
		mdContent.WriteString(content)
		mdContent.WriteString("\n")
	}

	if err := os.WriteFile(mdPath, []byte(mdContent.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	return nil
}

// createRedirectCommand creates the redirect command
func (a *App) createRedirectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "redirect [directory]",
		Short: "Set up data redirection to another directory",
		Long: `Configure ZAMM to read data from a different directory by creating a local-metadata.json file.
The specified directory will be used instead of the local .zamm directory for all data storage.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := args[0]

			// Get current working directory
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			// Convert relative paths to absolute
			if !filepath.IsAbs(targetDir) {
				targetDir = filepath.Join(workingDir, targetDir)
			}

			// Verify the target directory exists
			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				return fmt.Errorf("target directory does not exist: %s", targetDir)
			}

			// Ensure local .zamm directory exists
			localZammDir := filepath.Join(workingDir, ".zamm")
			if err := os.MkdirAll(localZammDir, 0755); err != nil {
				return fmt.Errorf("failed to create .zamm directory: %w", err)
			}

			// Create local-metadata.json
			metadata := config.LocalMetadata{
				DataRedirect: targetDir,
			}

			metadataPath := filepath.Join(localZammDir, "local-metadata.json")
			jsonData, err := json.MarshalIndent(metadata, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			if err := os.WriteFile(metadataPath, jsonData, 0644); err != nil {
				return fmt.Errorf("failed to write metadata file: %w", err)
			}

			fmt.Printf("Successfully configured data redirection\n")
			fmt.Printf("Local metadata file: %s\n", metadataPath)
			fmt.Printf("Data will be redirected to: %s\n", targetDir)

			return nil
		},
	}
}
