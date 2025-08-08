package storage

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
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

// BaseDir returns the base directory path
func (fs *FileStorage) BaseDir() string {
	return fs.baseDir
}

// InitializeStorage creates the necessary directory structure
func (fs *FileStorage) InitializeStorage() error {
	dirs := []string{
		fs.baseDir,
		filepath.Join(fs.baseDir, "nodes"),
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
			{"from_spec_id", "to_spec_id", "link_label"},
		})
	case "commit-links.csv":
		return fs.writeCSVFile(path, [][]string{
			{"spec_id", "commit_id", "repo_path", "link_label"},
		})
	case "project_metadata.json":
		metadata := models.ProjectMetadata{}
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

// Node operations
// CreateNode creates a new node
func (fs *FileStorage) CreateNode(node models.Node) error {
	if node.GetID() == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	path := fs.getNodeFilePath(node.GetID())
	return fs.writeJSONFile(path, node)
}

// GetNode retrieves a node by ID
func (fs *FileStorage) GetNode(id string) (models.Node, error) {
	path := fs.getNodeFilePath(id)

	// First read as NodeBase to determine the type
	var nodeBase models.NodeBase
	if err := fs.readJSONFile(path, &nodeBase); err != nil {
		if os.IsNotExist(err) {
			return nil, models.NewZammError(models.ErrTypeNotFound, "node not found")
		}
		return nil, err
	}

	// Based on the type, create the appropriate node
	switch nodeBase.Type {
	case "specification":
		var spec models.Spec
		if err := fs.readJSONFile(path, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	case "project":
		var project models.Project
		if err := fs.readJSONFile(path, &project); err != nil {
			return nil, err
		}
		return &project, nil
	case "implementation":
		var implementation models.Implementation
		if err := fs.readJSONFile(path, &implementation); err != nil {
			return nil, err
		}
		return &implementation, nil
	default:
		return &nodeBase, nil
	}
}

// UpdateNode updates an existing node
func (fs *FileStorage) UpdateNode(node models.Node) error {
	if node.GetID() == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	// Check if node exists
	_, err := fs.GetNode(node.GetID())
	if err != nil {
		return err
	}

	path := fs.getNodeFilePath(node.GetID())
	return fs.writeJSONFile(path, node)
}

// DeleteNode deletes a node
func (fs *FileStorage) DeleteNode(id string) error {
	path := fs.getNodeFilePath(id)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return models.NewZammError(models.ErrTypeNotFound, "node not found")
	}

	return os.Remove(path)
}

// ListNodes returns all nodes
func (fs *FileStorage) ListNodes() ([]models.Node, error) {
	nodesDir := filepath.Join(fs.baseDir, "nodes")
	entries, err := os.ReadDir(nodesDir)
	if err != nil {
		return nil, err
	}

	var nodes []models.Node
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		node, err := fs.GetNode(id)
		if err != nil {
			continue // Skip invalid files
		}
		nodes = append(nodes, node)
	}

	// Sort by ID for consistent ordering
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].GetID() < nodes[j].GetID()
	})

	return nodes, nil
}

// SpecCommitLink operations

// CreateSpecCommitLink creates a new spec-commit link
func (fs *FileStorage) CreateSpecCommitLink(link *models.SpecCommitLink) error {
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

// DeleteSpecCommitLink deletes a spec-commit link by matching fields
func (fs *FileStorage) DeleteSpecCommitLink(specID string) error {
	links, err := fs.getAllSpecCommitLinks()
	if err != nil {
		return err
	}

	found := false
	var filtered []*models.SpecCommitLink
	for _, link := range links {
		if link.SpecID != specID {
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

// DeleteSpecCommitLinkByFields deletes a spec-commit link by matching all fields
func (fs *FileStorage) DeleteSpecCommitLinkByFields(specID, commitID, repoPath string) error {
	links, err := fs.getAllSpecCommitLinks()
	if err != nil {
		return err
	}

	found := false
	var filtered []*models.SpecCommitLink
	for _, link := range links {
		if link.SpecID != specID || link.CommitID != commitID || link.RepoPath != repoPath {
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

// DeleteLink deletes a spec-commit link by specID (alias for DeleteSpecCommitLink)
func (fs *FileStorage) DeleteLink(specID string) error {
	return fs.DeleteSpecCommitLink(specID)
}

// SpecSpecLink operations

// CreateSpecSpecLink creates a new spec-spec link
func (fs *FileStorage) CreateSpecSpecLink(link *models.SpecSpecLink) error {
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

// DeleteSpecSpecLink deletes a spec-spec link by matching fields
func (fs *FileStorage) DeleteSpecSpecLink(fromSpecID, toSpecID string) error {
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

// Hierarchical operations
// GetLinkedSpecs retrieves specs linked to a given spec
func (fs *FileStorage) GetLinkedSpecs(specID string, direction models.Direction) ([]*models.Spec, error) {
	links, err := fs.GetSpecSpecLinks(specID, direction)
	if err != nil {
		return nil, err
	}

	var specs []*models.Spec
	for _, link := range links {
		var targetSpecID string
		if direction == models.Outgoing {
			targetSpecID = link.ToSpecID
		} else {
			targetSpecID = link.FromSpecID
		}

		node, err := fs.GetNode(targetSpecID)
		if err != nil {
			continue // Skip if spec not found
		}

		// Type assert to Spec
		if spec, ok := node.(*models.Spec); ok {
			specs = append(specs, spec)
		}
	}

	return specs, nil
}

// GetLinkedNodes retrieves nodes linked to a given node
func (fs *FileStorage) GetLinkedNodes(nodeID string, direction models.Direction) ([]models.Node, error) {
	links, err := fs.GetSpecSpecLinks(nodeID, direction)
	if err != nil {
		return nil, err
	}

	var nodes []models.Node
	for _, link := range links {
		var targetNodeID string
		if direction == models.Outgoing {
			targetNodeID = link.ToSpecID
		} else {
			targetNodeID = link.FromSpecID
		}

		node, err := fs.GetNode(targetNodeID)
		if err != nil {
			continue // Skip if node not found
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// GetOrphanSpecs retrieves all specs that don't have any parent links
func (fs *FileStorage) GetOrphanSpecs() ([]*models.Spec, error) {
	allNodes, err := fs.ListNodes()
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

	var orphans []*models.Spec
	for _, node := range allNodes {
		// Only include Spec nodes
		if spec, ok := node.(*models.Spec); ok {
			if !hasParents[spec.ID] {
				orphans = append(orphans, spec)
			}
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
			metadata = models.ProjectMetadata{}
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

	path := filepath.Join(fs.baseDir, "project_metadata.json")
	return fs.writeJSONFile(path, metadata)
}

// Helper methods

// getNodeFilePath returns the file path for a node
func (fs *FileStorage) getNodeFilePath(nodeID string) string {
	return filepath.Join(fs.baseDir, "nodes", nodeID+".json")
}

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

		if len(record) < 4 {
			continue // Skip invalid records
		}

		link := &models.SpecCommitLink{
			SpecID:    record[0],
			CommitID:  record[1],
			RepoPath:  record[2],
			LinkLabel: record[3],
		}
		links = append(links, link)
	}

	return links, nil
}

// writeSpecCommitLinks writes spec-commit links to CSV
func (fs *FileStorage) writeSpecCommitLinks(links []*models.SpecCommitLink) error {
	path := filepath.Join(fs.baseDir, "commit-links.csv")

	records := [][]string{
		{"spec_id", "commit_id", "repo_path", "link_label"},
	}

	for _, link := range links {
		records = append(records, []string{
			link.SpecID,
			link.CommitID,
			link.RepoPath,
			link.LinkLabel,
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

		if len(record) < 3 {
			continue // Skip invalid records
		}

		link := &models.SpecSpecLink{
			FromSpecID: record[0],
			ToSpecID:   record[1],
			LinkLabel:  record[2],
		}
		links = append(links, link)
	}

	return links, nil
}

// writeSpecSpecLinks writes spec-spec links to CSV
func (fs *FileStorage) writeSpecSpecLinks(links []*models.SpecSpecLink) error {
	path := filepath.Join(fs.baseDir, "spec-links.csv")

	records := [][]string{
		{"from_spec_id", "to_spec_id", "link_label"},
	}

	for _, link := range links {
		records = append(records, []string{
			link.FromSpecID,
			link.ToSpecID,
			link.LinkLabel,
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
	defer func() {
		_ = file.Close() // Explicitly ignore error in defer
	}()

	reader := csv.NewReader(file)
	return reader.ReadAll()
}

// writeCSVFile writes CSV data to a file
func (fs *FileStorage) writeCSVFile(path string, records [][]string) error {
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
