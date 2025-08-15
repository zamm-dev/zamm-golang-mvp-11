package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/config"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
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
		Short: "Run system migrations for node format and mappings",
		RunE: func(cmd *cobra.Command, args []string) error {
			migrationsRun := 0

			// Run title-to-heading migration
			if err := a.migrateTitlesToHeadings(); err != nil {
				return err
			}
			fmt.Printf("[title-to-heading] Migrated node titles to level 1 headings.\n")
			migrationsRun++

			if migrationsRun == 0 {
				fmt.Println("All migrations are up to date.")
			}
			return nil
		},
	}
}

// migrateTitlesToHeadings migrates all node files to use level 1 headings for titles
func (a *App) migrateTitlesToHeadings() error {
	// Cast storage to FileStorage to access getAllNodeFileLinks and GetNodeFilePath
	fileStorage, ok := a.storage.(*storage.FileStorage)
	if !ok {
		return fmt.Errorf("storage is not FileStorage type")
	}

	// Get all node IDs from both regular storage and custom file paths
	migratedNodeIDs := make(map[string]bool)
	migratedCount := 0

	// First, migrate all nodes from regular storage (.zamm/nodes/)
	nodes, err := a.specService.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes {
		nodeID := node.GetID()
		// Get the file path for this node using storage's GetNodeFilePath
		filePath := fileStorage.GetNodeFilePath(nodeID)

		if err := a.migrateNodeFileToHeading(filePath); err != nil {
			return fmt.Errorf("failed to migrate node %s: %w", nodeID, err)
		}
		migratedNodeIDs[nodeID] = true
		migratedCount++
	}

	// Second, get all custom file paths from node-files.csv and migrate those too
	nodeFileLinks, err := fileStorage.GetAllNodeFileLinks()
	if err == nil { // Don't fail if the CSV doesn't exist or can't be read
		for nodeID, customPath := range nodeFileLinks {
			// Skip if we already migrated this node from regular storage
			if migratedNodeIDs[nodeID] {
				continue
			}

			// Build the full path if it's relative
			var filePath string
			if filepath.IsAbs(customPath) {
				filePath = customPath
			} else {
				filePath = filepath.Join(filepath.Dir(fileStorage.BaseDir()), customPath)
			}

			if err := a.migrateNodeFileToHeading(filePath); err != nil {
				return fmt.Errorf("failed to migrate custom path node %s at %s: %w", nodeID, filePath, err)
			}
			migratedCount++
		}
	}

	fmt.Printf("Migrated %d node files to use level 1 headings.\n", migratedCount)
	return nil
}

// migrateNodeFileToHeading migrates a single node file to use level 1 heading for title
func (a *App) migrateNodeFileToHeading(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(data)

	// Check if already has frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("invalid markdown format: missing frontmatter")
	}

	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid markdown format: malformed frontmatter")
	}

	yamlContent := parts[0]
	markdownContent := strings.TrimSpace(parts[1])

	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
		return fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Extract title from frontmatter
	title, hasTitle := frontmatter["title"].(string)
	if !hasTitle || title == "" {
		return fmt.Errorf("no title found in frontmatter")
	}

	// Check if content already starts with level 1 heading
	if strings.HasPrefix(markdownContent, "# "+title+"\n") {
		// Already migrated, skip
		return nil
	}

	// Remove title from frontmatter
	delete(frontmatter, "title")

	// Rebuild YAML frontmatter
	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML frontmatter: %w", err)
	}

	// Build new content with title as level 1 heading
	var newContent strings.Builder
	newContent.WriteString("---\n")
	newContent.Write(yamlData)
	newContent.WriteString("---\n\n")
	newContent.WriteString("# ")
	newContent.WriteString(title)
	newContent.WriteString("\n\n")
	if markdownContent != "" {
		newContent.WriteString(markdownContent)
		newContent.WriteString("\n")
	}

	return os.WriteFile(filePath, []byte(newContent.String()), 0644)
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

// createRecacheCommand creates the recache command
func (a *App) createRecacheCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recache",
		Short: "Update node-files.csv with any missing nodes from the .zamm/nodes directory",
		Long: `Scan the .zamm/nodes directory for node files and ensure all nodes are properly 
tracked in node-files.csv. This fixes issues where new nodes weren't being added to the index.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Cast storage to FileStorage to access file system operations
			fileStorage, ok := a.storage.(*storage.FileStorage)
			if !ok {
				return fmt.Errorf("storage is not FileStorage type")
			}

			// Get existing node-file mappings
			existingMappings, err := fileStorage.GetAllNodeFileLinks()
			if err != nil {
				// If file doesn't exist, start with empty map
				existingMappings = make(map[string]string)
			}

			// Scan .zamm/nodes directory for all .md files
			nodesDir := filepath.Join(fileStorage.BaseDir(), "nodes")
			entries, err := os.ReadDir(nodesDir)
			if err != nil {
				return fmt.Errorf("failed to read nodes directory: %w", err)
			}

			newCount := 0
			updatedMappings := make(map[string]string)

			// Copy existing mappings
			for nodeID, filePath := range existingMappings {
				updatedMappings[nodeID] = filePath
			}

			// Process each .md file in the nodes directory
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}

				filePath := filepath.Join(nodesDir, entry.Name())

				// Extract node ID from the file
				nodeID, err := a.extractNodeIDFromFile(filePath)
				if err != nil {
					fmt.Printf("Warning: failed to extract node ID from %s: %v\n", entry.Name(), err)
					continue
				}

				// Create relative path from project root for storage
				projectRoot := filepath.Dir(fileStorage.BaseDir())
				relPath, err := filepath.Rel(projectRoot, filePath)
				if err != nil {
					// If we can't make it relative, use absolute path
					relPath = filePath
				}

				// Check if this node is already tracked
				if _, exists := existingMappings[nodeID]; !exists {
					updatedMappings[nodeID] = relPath
					newCount++
					fmt.Printf("Added missing node: %s -> %s\n", nodeID, relPath)
				}
			}

			// Write updated mappings back to CSV
			if err := a.writeNodeFileLinksMap(fileStorage, updatedMappings); err != nil {
				return fmt.Errorf("failed to update node-files.csv: %w", err)
			}

			if newCount == 0 {
				fmt.Println("No missing nodes found. node-files.csv is up to date.")
			} else {
				fmt.Printf("Successfully added %d missing nodes to node-files.csv\n", newCount)
			}

			return nil
		},
	}
}

// extractNodeIDFromFile reads a node file and extracts the node ID from frontmatter
func (a *App) extractNodeIDFromFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return "", fmt.Errorf("invalid markdown format: missing frontmatter")
	}

	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid markdown format: malformed frontmatter")
	}

	yamlContent := parts[0]

	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
		return "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	nodeID, ok := frontmatter["id"].(string)
	if !ok || nodeID == "" {
		return "", fmt.Errorf("missing or invalid id field in frontmatter")
	}

	return nodeID, nil
}

// writeNodeFileLinksMap writes the node-file mappings to CSV using the same format as FileStorage
func (a *App) writeNodeFileLinksMap(fileStorage *storage.FileStorage, nodeFiles map[string]string) error {
	path := filepath.Join(fileStorage.BaseDir(), "node-files.csv")

	records := [][]string{
		{"node_id", "file_path"},
	}

	// Create a slice of node IDs and sort them alphabetically for consistent git diffs
	nodeIDs := make([]string, 0, len(nodeFiles))
	for nodeID := range nodeFiles {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)

	// Add records in sorted order
	for _, nodeID := range nodeIDs {
		records = append(records, []string{nodeID, nodeFiles[nodeID]})
	}

	// Write CSV file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close() // Explicitly ignore error in defer
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer.WriteAll(records)
}
