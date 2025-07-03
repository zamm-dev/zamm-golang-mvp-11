package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// createExportCommand creates the export command
func (a *App) createExportCommand() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export all data to .zamm folder",
		Long:  "Exports all specifications, links, and metadata to a .zamm folder structure",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runExport(outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", ".zamm", "Output directory")

	return cmd
}

// runExport executes the export functionality
func (a *App) runExport(outputDir string) error {
	// Create output directory structure
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	specsDir := filepath.Join(outputDir, "specs")
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return fmt.Errorf("failed to create specs directory: %w", err)
	}

	// Export specs to individual JSON files
	if err := a.exportSpecs(specsDir); err != nil {
		return fmt.Errorf("failed to export specs: %w", err)
	}

	// Export spec-links.csv
	if err := a.exportSpecLinks(outputDir); err != nil {
		return fmt.Errorf("failed to export spec links: %w", err)
	}

	// Export commit-links.csv
	if err := a.exportCommitLinks(outputDir); err != nil {
		return fmt.Errorf("failed to export commit links: %w", err)
	}

	// Export project_metadata.json
	if err := a.exportProjectMetadata(outputDir); err != nil {
		return fmt.Errorf("failed to export project metadata: %w", err)
	}

	fmt.Printf("Successfully exported data to %s\n", outputDir)
	return nil
}

// exportSpecs exports individual spec files to JSON
func (a *App) exportSpecs(specsDir string) error {
	specs, err := a.storage.ListSpecNodes()
	if err != nil {
		return err
	}

	for _, spec := range specs {
		// Create export struct without timestamps
		exportSpec := struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Content string `json:"content"`
		}{
			ID:      spec.ID,
			Title:   spec.Title,
			Content: spec.Content,
		}

		filename := fmt.Sprintf("%s.json", spec.ID)
		filepath := filepath.Join(specsDir, filename)

		data, err := json.MarshalIndent(exportSpec, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal spec %s: %w", spec.ID, err)
		}

		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return fmt.Errorf("failed to write spec file %s: %w", filepath, err)
		}
	}

	return nil
}

// exportSpecLinks exports spec-spec links to CSV
func (a *App) exportSpecLinks(outputDir string) error {
	filepath := filepath.Join(outputDir, "spec-links.csv")
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create spec-links.csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"from_spec_id", "to_spec_id", "link_type"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Get all spec-spec links directly from database
	links, err := a.getAllSpecSpecLinks()
	if err != nil {
		return fmt.Errorf("failed to get spec-spec links: %w", err)
	}

	// Write all links to CSV
	for _, link := range links {
		if err := writer.Write([]string{link.FromSpecID, link.ToSpecID, link.LinkType}); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// exportCommitLinks exports spec-commit links to CSV
func (a *App) exportCommitLinks(outputDir string) error {
	// Get all specs first to find all their commit links
	specs, err := a.storage.ListSpecNodes()
	if err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, "commit-links.csv")
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create commit-links.csv: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"spec_id", "commit_id", "repo_path", "link_type"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Get commit links for each spec
	for _, spec := range specs {
		links, err := a.storage.GetLinksBySpec(spec.ID)
		if err != nil {
			return fmt.Errorf("failed to get links for spec %s: %w", spec.ID, err)
		}

		for _, link := range links {
			if err := writer.Write([]string{link.SpecID, link.CommitID, link.RepoPath, link.LinkType}); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}
	}

	return nil
}

// exportProjectMetadata exports project metadata to JSON
func (a *App) exportProjectMetadata(outputDir string) error {
	metadata, err := a.storage.GetProjectMetadata()
	if err != nil {
		// If metadata doesn't exist, create empty metadata
		if zammErr, ok := err.(*models.ZammError); ok && zammErr.Type == models.ErrTypeNotFound {
			metadata = &models.ProjectMetadata{}
		} else {
			return err
		}
	}

	// Create export struct without timestamps and ID
	exportMetadata := struct {
		RootSpecID *string `json:"root_spec_id"`
	}{
		RootSpecID: metadata.RootSpecID,
	}

	filepath := filepath.Join(outputDir, "project_metadata.json")
	data, err := json.MarshalIndent(exportMetadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project metadata: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project metadata file: %w", err)
	}

	return nil
}

// getAllSpecSpecLinks retrieves all spec-spec links from the database
func (a *App) getAllSpecSpecLinks() ([]*models.SpecSpecLink, error) {
	// Since there's no direct method to get all spec-spec links, we'll use the underlying storage
	// We need to add a method to the storage interface or use raw SQL
	// For now, let's implement this using the existing GetLinkedSpecs method
	
	specs, err := a.storage.ListSpecNodes()
	if err != nil {
		return nil, err
	}

	var allLinks []*models.SpecSpecLink
	linkMap := make(map[string]bool) // to avoid duplicates

	for _, spec := range specs {
		// Get outgoing links for this spec
		children, err := a.storage.GetLinkedSpecs(spec.ID, models.Outgoing)
		if err != nil {
			return nil, fmt.Errorf("failed to get linked specs for %s: %w", spec.ID, err)
		}

		for _, child := range children {
			linkKey := fmt.Sprintf("%s->%s", spec.ID, child.ID)
			if !linkMap[linkKey] {
				linkMap[linkKey] = true
				// Create a SpecSpecLink struct
				link := &models.SpecSpecLink{
					FromSpecID: spec.ID,
					ToSpecID:   child.ID,
					LinkType:   "child", // We know this from the GetLinkedSpecs query
				}
				allLinks = append(allLinks, link)
			}
		}
	}

	return allLinks, nil
}
