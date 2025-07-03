package storage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/yourorg/zamm-mvp/internal/models"
)

// FileStorage implements file-based storage for ZAMM
type FileStorage struct {
	baseDir string
}

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(baseDir string) *FileStorage {
	return &FileStorage{
		baseDir: baseDir,
	}
}

// InitializeStorage creates the necessary directory structure
func (fs *FileStorage) InitializeStorage() error {
	dirs := []string{
		fs.baseDir,
		filepath.Join(fs.baseDir, "specs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create empty files if they don't exist
	files := []string{"spec-links.csv", "commit-links.csv", "project_metadata.json"}
	for _, file := range files {
		path := filepath.Join(fs.baseDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := fs.createEmptyFile(path, file); err != nil {
				return err
			}
		}
	}

	return nil
}

// createEmptyFile creates empty files with appropriate headers
func (fs *FileStorage) createEmptyFile(path, filename string) error {
	switch filename {
	case "spec-links.csv":
		return fs.writeCSVFile(path, [][]string{
			{"id", "from_spec_id", "to_spec_id", "link_type", "created_at"},
		})
	case "commit-links.csv":
		return fs.writeCSVFile(path, [][]string{
			{"id", "spec_id", "commit_id", "repo_path", "link_type", "created_at"},
		})
	case "project_metadata.json":
		metadata := models.ProjectMetadata{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		return fs.writeJSONFile(path, metadata)
	default:
		// Create empty file
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		return file.Close()
	}
}

// SpecNode operations

// CreateSpecNode creates a new spec node
func (fs *FileStorage) CreateSpecNode(spec *models.SpecNode) error {
	if spec.ID == "" {
		return fmt.Errorf("spec ID cannot be empty")
	}

	now := time.Now()
	spec.CreatedAt = now
	spec.UpdatedAt = now

	path := filepath.Join(fs.baseDir, "specs", spec.ID+".json")
	return fs.writeJSONFile(path, spec)
}

// GetSpecNode retrieves a spec node by ID
func (fs *FileStorage) GetSpecNode(id string) (*models.SpecNode, error) {
	path := filepath.Join(fs.baseDir, "specs", id+".json")
	
	var spec models.SpecNode
	if err := fs.readJSONFile(path, &spec); err != nil {
		if os.IsNotExist(err) {
			return nil, models.NewZammError(models.ErrTypeNotFound, "spec not found")
		}
		return nil, err
	}

	return &spec, nil
}

// UpdateSpecNode updates an existing spec node
func (fs *FileStorage) UpdateSpecNode(spec *models.SpecNode) error {
	if spec.ID == "" {
		return fmt.Errorf("spec ID cannot be empty")
	}

	// Check if spec exists
	existing, err := fs.GetSpecNode(spec.ID)
	if err != nil {
		return err
	}

	spec.CreatedAt = existing.CreatedAt
	spec.UpdatedAt = time.Now()

	path := filepath.Join(fs.baseDir, "specs", spec.ID+".json")
	return fs.writeJSONFile(path, spec)
}

// DeleteSpecNode deletes a spec node
func (fs *FileStorage) DeleteSpecNode(id string) error {
	path := filepath.Join(fs.baseDir, "specs", id+".json")
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return models.NewZammError(models.ErrTypeNotFound, "spec not found")
	}

	return os.Remove(path)
}

// ListSpecNodes returns all spec nodes
func (fs *FileStorage) ListSpecNodes() ([]*models.SpecNode, error) {
	specsDir := filepath.Join(fs.baseDir, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return nil, err
	}

	var specs []*models.SpecNode
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		spec, err := fs.GetSpecNode(id)
		if err != nil {
			continue // Skip invalid files
		}
		specs = append(specs, spec)
	}

	// Sort by creation time
	sort.Slice(specs, func(i, j int) bool {
		return specs[i].CreatedAt.Before(specs[j].CreatedAt)
	})

	return specs, nil
}

// SpecCommitLink operations

// CreateSpecCommitLink creates a new spec-commit link
func (fs *FileStorage) CreateSpecCommitLink(link *models.SpecCommitLink) error {
	if link.ID == "" {
		return fmt.Errorf("link ID cannot be empty")
	}

	link.CreatedAt = time.Now()

	links, err := fs.getAllSpecCommitLinks()
	if err != nil {
		return err
	}

	links = append(links, link)
	return fs.writeSpecCommitLinks(links)
}

// GetSpecCommitLinks retrieves all spec-commit links for a spec
func (fs *FileStorage) GetSpecCommitLinks(specID string) ([]*models.SpecCommitLink, error) {
	allLinks, err := fs.getAllSpecCommitLinks()
	if err != nil {
		return nil, err
	}

	var links []*models.SpecCommitLink
	for _, link := range allLinks {
		if link.SpecID == specID {
			links = append(links, link)
		}
	}

	return links, nil
}

// DeleteSpecCommitLink deletes a spec-commit link
func (fs *FileStorage) DeleteSpecCommitLink(id string) error {
	links, err := fs.getAllSpecCommitLinks()
	if err != nil {
		return err
	}

	found := false
	var filtered []*models.SpecCommitLink
	for _, link := range links {
		if link.ID != id {
			filtered = append(filtered, link)
		} else {
			found = true
		}
	}

	if !found {
		return models.NewZammError(models.ErrTypeNotFound, "spec-commit link not found")
	}

	return fs.writeSpecCommitLinks(filtered)
}

// GetLinksByCommit retrieves all spec-commit links for a commit
func (fs *FileStorage) GetLinksByCommit(commitID, repoPath string) ([]*models.SpecCommitLink, error) {
	allLinks, err := fs.getAllSpecCommitLinks()
	if err != nil {
		return nil, err
	}

	var links []*models.SpecCommitLink
	for _, link := range allLinks {
		if link.CommitID == commitID && link.RepoPath == repoPath {
			links = append(links, link)
		}
	}

	return links, nil
}

// GetLinksBySpec retrieves all spec-commit links for a spec (alias for GetSpecCommitLinks)
func (fs *FileStorage) GetLinksBySpec(specID string) ([]*models.SpecCommitLink, error) {
	return fs.GetSpecCommitLinks(specID)
}

// DeleteLink deletes a spec-commit link by ID (alias for DeleteSpecCommitLink)
func (fs *FileStorage) DeleteLink(id string) error {
	return fs.DeleteSpecCommitLink(id)
}

// SpecSpecLink operations

// CreateSpecSpecLink creates a new spec-spec link
func (fs *FileStorage) CreateSpecSpecLink(link *models.SpecSpecLink) error {
	if link.ID == "" {
		return fmt.Errorf("link ID cannot be empty")
	}

	link.CreatedAt = time.Now()

	links, err := fs.getAllSpecSpecLinks()
	if err != nil {
		return err
	}

	links = append(links, link)
	return fs.writeSpecSpecLinks(links)
}

// GetSpecSpecLinks retrieves spec-spec links
func (fs *FileStorage) GetSpecSpecLinks(specID string, direction models.Direction) ([]*models.SpecSpecLink, error) {
	allLinks, err := fs.getAllSpecSpecLinks()
	if err != nil {
		return nil, err
	}

	var links []*models.SpecSpecLink
	for _, link := range allLinks {
		if direction == models.Outgoing && link.FromSpecID == specID {
			links = append(links, link)
		} else if direction == models.Incoming && link.ToSpecID == specID {
			links = append(links, link)
		}
	}

	return links, nil
}

// DeleteSpecSpecLink deletes a spec-spec link
func (fs *FileStorage) DeleteSpecSpecLink(id string) error {
	links, err := fs.getAllSpecSpecLinks()
	if err != nil {
		return err
	}

	found := false
	var filtered []*models.SpecSpecLink
	for _, link := range links {
		if link.ID != id {
			filtered = append(filtered, link)
		} else {
			found = true
		}
	}

	if !found {
		return models.NewZammError(models.ErrTypeNotFound, "spec-spec link not found")
	}

	return fs.writeSpecSpecLinks(filtered)
}

// DeleteSpecLinkBySpecs deletes a spec-spec link by source and target spec IDs
func (fs *FileStorage) DeleteSpecLinkBySpecs(fromSpecID, toSpecID string) error {
	links, err := fs.getAllSpecSpecLinks()
	if err != nil {
		return err
	}

	found := false
	var filtered []*models.SpecSpecLink
	for _, link := range links {
		if link.FromSpecID != fromSpecID || link.ToSpecID != toSpecID {
			filtered = append(filtered, link)
		} else {
			found = true
		}
	}

	if !found {
		return models.NewZammError(models.ErrTypeNotFound, "spec-spec link not found")
	}

	return fs.writeSpecSpecLinks(filtered)
}

// GetLinkedSpecs retrieves specs linked to a given spec
func (fs *FileStorage) GetLinkedSpecs(specID string, direction models.Direction) ([]*models.SpecNode, error) {
	links, err := fs.GetSpecSpecLinks(specID, direction)
	if err != nil {
		return nil, err
	}

	var specs []*models.SpecNode
	for _, link := range links {
		var targetSpecID string
		if direction == models.Outgoing {
			targetSpecID = link.ToSpecID
		} else {
			targetSpecID = link.FromSpecID
		}

		spec, err := fs.GetSpecNode(targetSpecID)
		if err != nil {
			continue // Skip if spec not found
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// GetOrphanSpecs retrieves all specs that don't have any parent links
func (fs *FileStorage) GetOrphanSpecs() ([]*models.SpecNode, error) {
	allSpecs, err := fs.ListSpecNodes()
	if err != nil {
		return nil, err
	}

	allLinks, err := fs.getAllSpecSpecLinks()
	if err != nil {
		return nil, err
	}

	// Build a set of spec IDs that have parents
	hasParents := make(map[string]bool)
	for _, link := range allLinks {
		hasParents[link.FromSpecID] = true
	}

	var orphans []*models.SpecNode
	for _, spec := range allSpecs {
		if !hasParents[spec.ID] {
			orphans = append(orphans, spec)
		}
	}

	return orphans, nil
}

// ProjectMetadata operations

// GetProjectMetadata retrieves project metadata
func (fs *FileStorage) GetProjectMetadata() (*models.ProjectMetadata, error) {
	path := filepath.Join(fs.baseDir, "project_metadata.json")
	
	var metadata models.ProjectMetadata
	if err := fs.readJSONFile(path, &metadata); err != nil {
		if os.IsNotExist(err) {
			// Create default metadata
			metadata = models.ProjectMetadata{
				ID:        1,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := fs.writeJSONFile(path, metadata); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return &metadata, nil
}

// SetRootSpecID sets the root spec ID
func (fs *FileStorage) SetRootSpecID(specID *string) error {
	metadata, err := fs.GetProjectMetadata()
	if err != nil {
		return err
	}

	metadata.RootSpecID = specID
	metadata.UpdatedAt = time.Now()

	path := filepath.Join(fs.baseDir, "project_metadata.json")
	return fs.writeJSONFile(path, metadata)
}

// Helper methods

// getAllSpecCommitLinks reads all spec-commit links from CSV
func (fs *FileStorage) getAllSpecCommitLinks() ([]*models.SpecCommitLink, error) {
	path := filepath.Join(fs.baseDir, "commit-links.csv")
	records, err := fs.readCSVFile(path)
	if err != nil {
		return nil, err
	}

	var links []*models.SpecCommitLink
	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}

		if len(record) < 6 {
			continue // Skip invalid records
		}

		createdAt, _ := time.Parse(time.RFC3339, record[5])
		link := &models.SpecCommitLink{
			ID:        record[0],
			SpecID:    record[1],
			CommitID:  record[2],
			RepoPath:  record[3],
			LinkType:  record[4],
			CreatedAt: createdAt,
		}
		links = append(links, link)
	}

	return links, nil
}

// writeSpecCommitLinks writes spec-commit links to CSV
func (fs *FileStorage) writeSpecCommitLinks(links []*models.SpecCommitLink) error {
	path := filepath.Join(fs.baseDir, "commit-links.csv")
	
	records := [][]string{
		{"id", "spec_id", "commit_id", "repo_path", "link_type", "created_at"},
	}

	for _, link := range links {
		records = append(records, []string{
			link.ID,
			link.SpecID,
			link.CommitID,
			link.RepoPath,
			link.LinkType,
			link.CreatedAt.Format(time.RFC3339),
		})
	}

	return fs.writeCSVFile(path, records)
}

// getAllSpecSpecLinks reads all spec-spec links from CSV
func (fs *FileStorage) getAllSpecSpecLinks() ([]*models.SpecSpecLink, error) {
	path := filepath.Join(fs.baseDir, "spec-links.csv")
	records, err := fs.readCSVFile(path)
	if err != nil {
		return nil, err
	}

	var links []*models.SpecSpecLink
	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}

		if len(record) < 5 {
			continue // Skip invalid records
		}

		createdAt, _ := time.Parse(time.RFC3339, record[4])
		link := &models.SpecSpecLink{
			ID:         record[0],
			FromSpecID: record[1],
			ToSpecID:   record[2],
			LinkType:   record[3],
			CreatedAt:  createdAt,
		}
		links = append(links, link)
	}

	return links, nil
}

// writeSpecSpecLinks writes spec-spec links to CSV
func (fs *FileStorage) writeSpecSpecLinks(links []*models.SpecSpecLink) error {
	path := filepath.Join(fs.baseDir, "spec-links.csv")
	
	records := [][]string{
		{"id", "from_spec_id", "to_spec_id", "link_type", "created_at"},
	}

	for _, link := range links {
		records = append(records, []string{
			link.ID,
			link.FromSpecID,
			link.ToSpecID,
			link.LinkType,
			link.CreatedAt.Format(time.RFC3339),
		})
	}

	return fs.writeCSVFile(path, records)
}

// File I/O helpers

// readJSONFile reads JSON data from a file
func (fs *FileStorage) readJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// writeJSONFile writes JSON data to a file
func (fs *FileStorage) writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// readCSVFile reads CSV data from a file
func (fs *FileStorage) readCSVFile(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	return reader.ReadAll()
}

// writeCSVFile writes CSV data to a file
func (fs *FileStorage) writeCSVFile(path string, records [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	return writer.WriteAll(records)
}
