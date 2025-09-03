package storage

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"gopkg.in/yaml.v3"
)

// FileStorage implements file-based storage for ZAMM
type FileStorage struct {
	baseDir string
}

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(baseDir string) *FileStorage {
	fs := &FileStorage{
		baseDir: baseDir,
	}

	if _, err := os.Stat(fs.nodesDir()); errors.Is(err, os.ErrNotExist) {
		if err := fs.InitializeStorage(); err != nil {
			return nil
		}
	}
	return fs
}

// BaseDir returns the base directory path
func (fs *FileStorage) BaseDir() string {
	return fs.baseDir
}

func (fs *FileStorage) nodesDir() string {
	return filepath.Join(fs.baseDir, "nodes")
}

// InitializeStorage creates the necessary directory structure
func (fs *FileStorage) InitializeStorage() error {
	dirs := []string{
		fs.baseDir,
		fs.nodesDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create empty files if they don't exist
	files := []string{"spec-links.csv", "commit-links.csv", "node-files.csv", "project_metadata.json"}
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
	case "node-files.csv":
		return fs.writeCSVFile(path, [][]string{
			{"node_id", "file_path"},
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
	if node.ID() == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	path := fs.GetNodeFilePath(node.ID())

	// Write the node file
	if err := fs.writeMarkdownFile(path, node); err != nil {
		return err
	}

	// Ensure the node is tracked in node-files.csv
	// Get relative path from the project root for storage
	projectRoot := filepath.Dir(fs.baseDir)
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		// If we can't make it relative, use absolute path
		relPath = path
	}

	return fs.UpdateNodeFilePath(node.ID(), relPath)
}

// GetNode retrieves a node by ID
func (fs *FileStorage) GetNode(id string) (models.Node, error) {
	path := fs.GetNodeFilePath(id)

	// First read as NodeBase to determine the type
	var nodeBase models.NodeBase
	if err := fs.readMarkdownFile(path, &nodeBase); err != nil {
		if os.IsNotExist(err) {
			return nil, models.NewZammError(models.ErrTypeNotFound, "node not found")
		}
		return nil, err
	}

	// Based on the type, create the appropriate node
	switch nodeBase.Type() {
	case "specification":
		var spec models.Spec
		if err := fs.readMarkdownFile(path, &spec); err != nil {
			return nil, err
		}
		return &spec, nil
	case "project":
		var project models.Project
		if err := fs.readMarkdownFile(path, &project); err != nil {
			return nil, err
		}
		return &project, nil
	case "implementation":
		var implementation models.Implementation
		if err := fs.readMarkdownFile(path, &implementation); err != nil {
			return nil, err
		}
		return &implementation, nil
	default:
		return &nodeBase, nil
	}
}

// UpdateNode updates an existing node
func (fs *FileStorage) UpdateNode(node models.Node) error {
	if node.ID() == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	// Check if node exists
	_, err := fs.GetNode(node.ID())
	if err != nil {
		return err
	}

	path := fs.GetNodeFilePath(node.ID())
	return fs.writeMarkdownFile(path, node)
}

// DeleteNode deletes a node
func (fs *FileStorage) DeleteNode(id string) error {
	path := fs.GetNodeFilePath(id)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return models.NewZammError(models.ErrTypeNotFound, "node not found")
	}

	return os.Remove(path)
}

// ListNodes returns all nodes
func (fs *FileStorage) ListNodes() ([]models.Node, error) {
	nodes := make([]models.Node, 0)

	// Get all nodes from node-files.csv
	nodeFiles, err := fs.getAllNodeFileLinks()
	if err != nil {
		return nil, err
	}

	for nodeID := range nodeFiles {
		node, err := fs.GetNode(nodeID)
		if err != nil {
			continue // Skip invalid files
		}
		nodes = append(nodes, node)
	}

	// Sort by ID for consistent ordering
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID() < nodes[j].ID()
	})

	return nodes, nil
}

func (fs *FileStorage) ReadNode(id string) (models.Node, error) {
	return fs.GetNode(id)
}

func (fs *FileStorage) WriteNode(node models.Node) error {
	path, exists := fs.getNodeFilePathIfExists(node.ID())
	if !exists {
		return fs.CreateNode(node)
	}
	return fs.writeMarkdownFile(path, node)
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

	links := make([]*models.SpecCommitLink, 0, len(allLinks))
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
	filtered := make([]*models.SpecCommitLink, 0, len(links))
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
	filtered := make([]*models.SpecCommitLink, 0, len(links))
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

	links := make([]*models.SpecCommitLink, 0, len(allLinks))
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

	links := make([]*models.SpecSpecLink, 0, len(allLinks))
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
	filtered := make([]*models.SpecSpecLink, 0, len(links))
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
	filtered := make([]*models.SpecSpecLink, 0, len(links))
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

	specs := make([]*models.Spec, 0, len(links))
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

	nodes := make([]models.Node, 0, len(links))
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

	orphans := make([]*models.Spec, 0, len(allNodes))
	for _, node := range allNodes {
		// Only include Spec nodes
		if spec, ok := node.(*models.Spec); ok {
			if !hasParents[spec.ID()] {
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

func (fs *FileStorage) getNodeFilePathIfExists(nodeID string) (string, bool) {
	nodeFiles, err := fs.getAllNodeFileLinks()
	if err == nil {
		if customPath, exists := nodeFiles[nodeID]; exists {
			if !filepath.IsAbs(customPath) {
				return filepath.Join(filepath.Dir(fs.baseDir), customPath), true
			}
			return customPath, true
		}
	}
	return "", false
}

// GetNodeFilePath returns the file path for a node from node-files.csv or default location
func (fs *FileStorage) GetNodeFilePath(nodeID string) string {
	path, exists := fs.getNodeFilePathIfExists(nodeID)
	if exists {
		return path
	}
	return filepath.Join(fs.baseDir, "nodes", nodeID+".md")
}

// getAllSpecCommitLinks reads all spec-commit links from CSV
func (fs *FileStorage) getAllSpecCommitLinks() ([]*models.SpecCommitLink, error) {
	path := filepath.Join(fs.baseDir, "commit-links.csv")
	records, err := fs.readCSVFile(path)
	if err != nil {
		return nil, err
	}

	links := make([]*models.SpecCommitLink, 0, len(records))
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

	links := make([]*models.SpecSpecLink, 0, len(records))
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

// readMarkdownFile reads markdown data with YAML frontmatter from a file
func (fs *FileStorage) readMarkdownFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("invalid markdown format: missing frontmatter")
	}

	parts := strings.SplitN(content[4:], "\n---\n", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid markdown format: malformed frontmatter")
	}

	yamlContent := parts[0]
	markdownContent := strings.TrimSpace(parts[1])

	// Remove content after the last horizontal divider (child links section)
	if lastDividerIndex := strings.LastIndex(markdownContent, "\n---\n"); lastDividerIndex != -1 {
		markdownContent = strings.TrimSpace(markdownContent[:lastDividerIndex])
	}

	var frontmatter map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
		return fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Extract title from level 1 heading if present
	if strings.HasPrefix(markdownContent, "# ") {
		lines := strings.SplitN(markdownContent, "\n", 2)
		title := strings.TrimPrefix(lines[0], "# ")
		frontmatter["title"] = title

		// Remove title heading from content
		if len(lines) > 1 {
			markdownContent = strings.TrimSpace(lines[1])
		} else {
			markdownContent = ""
		}
	}

	frontmatter["content"] = markdownContent

	jsonData, err := json.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	return json.Unmarshal(jsonData, v)
}

// generateMarkdownString generates markdown content with YAML frontmatter
func (fs *FileStorage) generateMarkdownString(v interface{}) (string, error) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	var nodeData map[string]interface{}
	if err := json.Unmarshal(jsonData, &nodeData); err != nil {
		return "", fmt.Errorf("failed to unmarshal data: %w", err)
	}

	content, hasContent := nodeData["content"].(string)
	if !hasContent {
		content = ""
	}

	title, hasTitle := nodeData["title"].(string)

	// Create frontmatter map with all fields except content and title
	frontmatter := make(map[string]interface{})
	for key, value := range nodeData {
		if key != "content" && key != "title" {
			frontmatter[key] = value
		}
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("failed to marshal YAML frontmatter: %w", err)
	}

	var mdContent strings.Builder
	mdContent.WriteString("---\n")
	mdContent.Write(yamlData)
	mdContent.WriteString("---\n\n")

	// Add title as level 1 heading
	if hasTitle && title != "" {
		mdContent.WriteString("# ")
		mdContent.WriteString(title)
		mdContent.WriteString("\n\n")
	}

	if content != "" {
		mdContent.WriteString(content)
		mdContent.WriteString("\n")
	}

	return mdContent.String(), nil
}

func (fs *FileStorage) WriteNodeWithExtraData(node models.Node, extraData string) error {
	path, exists := fs.getNodeFilePathIfExists(node.ID())
	if !exists {
		path = fs.GetNodeFilePath(node.ID())

		// Ensure the node is tracked in node-files.csv
		// Get relative path from the project root for storage
		projectRoot := filepath.Dir(fs.baseDir)
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			// If we can't make it relative, use absolute path
			relPath = path
		}

		err = fs.UpdateNodeFilePath(node.ID(), relPath)
		if err != nil {
			return err
		}
	}

	content, err := fs.generateMarkdownString(node)
	if err != nil {
		return err
	}

	content += extraData
	return os.WriteFile(path, []byte(content), 0644)
}

// generateChildrenString generates child links for the markdown
func (fs *FileStorage) generateChildrenString(v interface{}, children models.ChildGroup) (string, error) {
	node, ok := v.(models.Node)
	if !ok {
		return "", fmt.Errorf("invalid node type")
	}

	// Append children section
	var childrenSection strings.Builder
	childrenSection.WriteString("\n---\n\n")
	childrenSection.WriteString("## Child Specifications\n\n")

	renderer := &markdownChildrenRenderer{
		sb:                &childrenSection,
		nodePathRetriever: fs.GetNodeFilePath,
		originPath:        filepath.Dir(fs.GetNodeFilePath(node.ID())),
	}
	children.Render(renderer)

	return childrenSection.String(), nil
}

// writeMarkdownFile writes markdown data with YAML frontmatter to a file
func (fs *FileStorage) writeMarkdownFile(path string, v interface{}) error {
	content, err := fs.generateMarkdownString(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// writeMarkdownFileWithChildren writes markdown data with YAML frontmatter and optional child links
func (fs *FileStorage) writeMarkdownFileWithChildren(path string, v interface{}, children models.ChildGroup) error {
	childrenContent, err := fs.generateChildrenString(v, children)
	if err != nil {
		return err
	}
	return fs.WriteNodeWithExtraData(v.(models.Node), childrenContent)
}

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

// getAllNodeFileLinks reads all node-file mappings from CSV
func (fs *FileStorage) getAllNodeFileLinks() (map[string]string, error) {
	path := filepath.Join(fs.baseDir, "node-files.csv")
	records, err := fs.readCSVFile(path)
	if err != nil {
		return nil, err
	}

	nodeFiles := make(map[string]string)
	for i, record := range records {
		if i == 0 {
			continue
		}

		if len(record) < 2 {
			continue
		}

		nodeFiles[record[0]] = record[1]
	}

	return nodeFiles, nil
}

// GetAllNodeFileLinks returns all node-file mappings (public wrapper for getAllNodeFileLinks)
func (fs *FileStorage) GetAllNodeFileLinks() (map[string]string, error) {
	return fs.getAllNodeFileLinks()
}

func (fs *FileStorage) writeNodeFileLinks(nodeFiles map[string]string) error {
	path := filepath.Join(fs.baseDir, "node-files.csv")

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

	return fs.writeCSVFile(path, records)
}

func (fs *FileStorage) WriteNodeWithChildren(node models.Node, childGrouping models.ChildGroup) error {
	if node.ID() == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	path := fs.GetNodeFilePath(node.ID())

	// Write the node file with children
	if err := fs.writeMarkdownFileWithChildren(path, node, childGrouping); err != nil {
		return err
	}

	// Ensure the node is tracked in node-files.csv
	projectRoot := filepath.Dir(fs.baseDir)
	relPath, err := filepath.Rel(projectRoot, path)
	if err != nil {
		relPath = path
	}

	return fs.UpdateNodeFilePath(node.ID(), relPath)
}

// UpdateNodeFilePath updates a single node's file path in the CSV
func (fs *FileStorage) UpdateNodeFilePath(nodeID, newPath string) error {
	nodeFiles, err := fs.getAllNodeFileLinks()
	if err != nil {
		return err
	}

	nodeFiles[nodeID] = newPath
	return fs.writeNodeFileLinks(nodeFiles)
}

type markdownChildrenRenderer struct {
	sb                *strings.Builder
	nodePathRetriever func(nodeID string) string
	originPath        string
}

func (r *markdownChildrenRenderer) RenderGroupStart(nestingLevel int, label string) {
	fmt.Fprintf(r.sb, "%*s- %s\n", nestingLevel*2, "", label)
}

func (r *markdownChildrenRenderer) RenderGroupEnd(nestingLevel int) {}

func (r *markdownChildrenRenderer) RenderNode(nestingLevel int, node models.Node) {
	childPath := r.nodePathRetriever(node.ID())
	relNodePath, err := filepath.Rel(r.originPath, childPath)
	if err != nil {
		relNodePath = childPath
	}
	fmt.Fprintf(r.sb, "%*s- [%s](%s)\n", nestingLevel*2, "", node.Title(), relNodePath)
}
