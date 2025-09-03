package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/storage"
)

const (
	DocumentationRoot = "docs"
	IndexFile         = "README.md"
)

// SpecService interface defines operations for managing specifications
type SpecService interface {
	CreateSpec(title, content string) (*models.Spec, error)
	CreateProject(title, content string) (*models.Project, error)
	CreateImplementation(title, content string, repoURL, branch, folderPath *string) (*models.Implementation, error)
	GetNode(id string) (models.Node, error)
	GetProject(id string) (*models.Project, error)
	UpdateSpec(id, title, content string) (*models.Spec, error)
	UpdateImplementation(id, title, content string, repoURL, branch, folderPath *string) (*models.Implementation, error)
	UpdateNode(id, title, content string) (models.Node, error)
	ListNodes() ([]models.Node, error)
	DeleteSpec(id string) error
	IsRootNode(node models.Node) bool

	// Hierarchical operations
	AddChildToParent(childSpecID, parentSpecID, label string) (*models.SpecSpecLink, error)
	RemoveChildFromParent(childSpecID, parentSpecID string) error
	GetParents(specID string) ([]models.Node, error)
	GetChildren(specID string) ([]models.Node, error)

	// Root spec operations
	InitializeRootSpec() error
	GetRootNode() (models.Node, error)
	GetOrphanSpecs() ([]*models.Spec, error)

	// Organization operations
	OrganizeNodes(nodeID string) error
}

// specService implements the SpecService interface
type specService struct {
	storage storage.Storage
}

// NewSpecService creates a new SpecService instance
func NewSpecService(storage storage.Storage) SpecService {
	return &specService{
		storage: storage,
	}
}

// CreateSpec creates a new specification
func (s *specService) CreateSpec(title, content string) (*models.Spec, error) {
	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	spec := models.NewSpec(strings.TrimSpace(title), strings.TrimSpace(content))

	if err := s.storage.CreateNode(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// CreateProject creates a new project
func (s *specService) CreateProject(title, content string) (*models.Project, error) {
	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	project := models.NewProject(strings.TrimSpace(title), strings.TrimSpace(content))

	if err := s.storage.CreateNode(project); err != nil {
		return nil, err
	}

	return project, nil
}

// CreateImplementation creates a new implementation node
func (s *specService) CreateImplementation(title, content string, repoURL, branch, folderPath *string) (*models.Implementation, error) {
	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	impl := models.NewImplementation(strings.TrimSpace(title), strings.TrimSpace(content))
	// Set optional fields if provided
	if repoURL != nil && strings.TrimSpace(*repoURL) != "" {
		v := strings.TrimSpace(*repoURL)
		impl.RepoURL = &v
	}
	if branch != nil && strings.TrimSpace(*branch) != "" {
		v := strings.TrimSpace(*branch)
		impl.Branch = &v
	}
	if folderPath != nil && strings.TrimSpace(*folderPath) != "" {
		v := strings.TrimSpace(*folderPath)
		impl.FolderPath = &v
	}

	if err := s.storage.CreateNode(impl); err != nil {
		return nil, err
	}

	return impl, nil
}

// GetProject retrieves a project by ID
func (s *specService) GetProject(id string) (*models.Project, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "project ID cannot be empty")
	}

	node, err := s.storage.GetNode(id)
	if err != nil {
		return nil, err
	}

	// Type assert to Project
	if project, ok := node.(*models.Project); ok {
		return project, nil
	}

	return nil, models.NewZammError(models.ErrTypeValidation, "node is not a project")
}

// GetNode retrieves a node by ID
func (s *specService) GetNode(id string) (models.Node, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	node, err := s.storage.GetNode(id)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// UpdateSpec updates an existing specification
func (s *specService) UpdateSpec(id, title, content string) (*models.Spec, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	// Get existing spec
	node, err := s.storage.GetNode(id)
	if err != nil {
		return nil, err
	}

	// Type assert to Spec
	spec, ok := node.(*models.Spec)
	if !ok {
		return nil, models.NewZammError(models.ErrTypeValidation, "node is not a spec")
	}

	// Update fields
	spec.SetTitle(strings.TrimSpace(title))
	spec.SetContent(strings.TrimSpace(content))
	spec.SetType("specification")

	// Save changes
	if err := s.storage.UpdateNode(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// UpdateImplementation updates an existing implementation node
func (s *specService) UpdateImplementation(id, title, content string, repoURL, branch, folderPath *string) (*models.Implementation, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "implementation ID cannot be empty")
	}

	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	// Get existing implementation
	node, err := s.storage.GetNode(id)
	if err != nil {
		return nil, err
	}

	// Type assert to Implementation
	impl, ok := node.(*models.Implementation)
	if !ok {
		return nil, models.NewZammError(models.ErrTypeValidation, "node is not an implementation")
	}

	// Update basic fields
	impl.SetTitle(strings.TrimSpace(title))
	impl.SetContent(strings.TrimSpace(content))
	impl.SetType("implementation")

	// Update optional fields if provided
	if repoURL != nil && strings.TrimSpace(*repoURL) != "" {
		v := strings.TrimSpace(*repoURL)
		impl.RepoURL = &v
	} else {
		impl.RepoURL = nil
	}
	if branch != nil && strings.TrimSpace(*branch) != "" {
		v := strings.TrimSpace(*branch)
		impl.Branch = &v
	} else {
		impl.Branch = nil
	}
	if folderPath != nil && strings.TrimSpace(*folderPath) != "" {
		v := strings.TrimSpace(*folderPath)
		impl.FolderPath = &v
	} else {
		impl.FolderPath = nil
	}

	// Save changes
	if err := s.storage.UpdateNode(impl); err != nil {
		return nil, err
	}

	return impl, nil
}

func isImplementationNode(node models.Node) bool {
	return node.Type() == "implementation"
}

func (s *specService) GetOrganizedChildren(node models.Node) (models.ChildGroup, error) {
	cg := node.GetChildGrouping()
	allChildren, err := s.GetChildren(node.ID())
	if err != nil {
		return cg, err
	}

	cg.AppendUnmatched(allChildren)
	cg.UngroupedLabel = "Children"

	if node.Type() == "project" {
		cg.Regroup("Implementations", isImplementationNode)
	}

	return cg, nil
}

// UpdateNode updates an existing node regardless of its type
func (s *specService) UpdateNode(id, title, content string) (models.Node, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "node ID cannot be empty")
	}

	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	// Get existing node
	node, err := s.storage.GetNode(id)
	if err != nil {
		return nil, err
	}

	// Update fields based on node type
	switch n := node.(type) {
	case *models.Spec:
		n.SetTitle(strings.TrimSpace(title))
		n.SetContent(strings.TrimSpace(content))
		n.SetType("specification")
	case *models.Project:
		n.SetTitle(strings.TrimSpace(title))
		n.SetContent(strings.TrimSpace(content))
		n.SetType("project")
	case *models.Implementation:
		n.SetTitle(strings.TrimSpace(title))
		n.SetContent(strings.TrimSpace(content))
		n.SetType("implementation")
	default:
		return nil, models.NewZammError(models.ErrTypeValidation, "unknown node type")
	}

	// Save changes with children links if any exist
	children, err := s.GetOrganizedChildren(node)
	if err != nil {
		return nil, err
	}

	err = s.storage.WriteNodeWithChildren(node, children)
	return node, err
}

// ListNodes retrieves all nodes regardless of type
func (s *specService) ListNodes() ([]models.Node, error) {
	nodes, err := s.storage.ListNodes()
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// DeleteSpec deletes a specification
func (s *specService) DeleteSpec(id string) error {
	if id == "" {
		return models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.DeleteNode(id)
}

// AddChildToParent adds a parent-child relationship by specifying the child and parent
func (s *specService) AddChildToParent(childSpecID, parentSpecID, label string) (*models.SpecSpecLink, error) {
	// Validate input
	if childSpecID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "child spec ID cannot be empty")
	}
	if parentSpecID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "parent spec ID cannot be empty")
	}
	if childSpecID == parentSpecID {
		return nil, models.NewZammError(models.ErrTypeValidation, "cannot link a node to itself")
	}

	// Verify both nodes exist
	_, err := s.storage.GetNode(childSpecID)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeValidation, "child node not found")
	}

	_, err = s.storage.GetNode(parentSpecID)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeValidation, "parent node not found")
	}

	// Use provided link type or default to "child"
	if label == "" {
		label = "child"
	}

	link := &models.SpecSpecLink{
		FromSpecID: childSpecID,
		ToSpecID:   parentSpecID,
		LinkLabel:  label,
	}

	if err := s.storage.CreateSpecSpecLink(link); err != nil {
		return nil, err
	}

	return link, nil
}

// RemoveChildFromParent removes a parent-child relationship by specifying the child and parent
func (s *specService) RemoveChildFromParent(childSpecID, parentSpecID string) error {
	if childSpecID == "" {
		return models.NewZammError(models.ErrTypeValidation, "child spec ID cannot be empty")
	}
	if parentSpecID == "" {
		return models.NewZammError(models.ErrTypeValidation, "parent spec ID cannot be empty")
	}

	return s.storage.DeleteSpecLinkBySpecs(childSpecID, parentSpecID)
}

// GetParents retrieves all parent nodes for a given node
func (s *specService) GetParents(specID string) ([]models.Node, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "node ID cannot be empty")
	}

	return s.storage.GetLinkedNodes(specID, models.Outgoing)
}

// GetChildren retrieves all child nodes for a given node
func (s *specService) GetChildren(specID string) ([]models.Node, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "node ID cannot be empty")
	}

	return s.storage.GetLinkedNodes(specID, models.Incoming)
}

// InitializeRootSpec creates the root specification if it doesn't exist
// and links all orphaned specs to it. On interactive mode startup, converts
// the root node to a Project node if it isn't already one.
func (s *specService) InitializeRootSpec() error {
	metadata, err := s.storage.GetProjectMetadata()
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get project metadata", err)
	}

	if metadata.RootSpecID == nil {
		// Create root project
		newRootProject, err := s.CreateProject("New Project", "Requirement: This project should exist.")
		if err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create root project", err)
		}

		// Set it as the root spec in metadata
		rootId := newRootProject.ID()
		err = s.storage.SetRootSpecID(&rootId)
		if err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to set root spec ID", err)
		}
		return nil
	}

	// Root exists, check if it's a Project
	rootNode, err := s.storage.GetNode(*metadata.RootSpecID)
	if err != nil {
		return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to get root node", err)
	}

	// If it's already a Project, we're done
	if rootNode.Type() == "project" {
		return nil
	} else { // otherwise, convert it to a Project
		rootNode.SetType("project")

		// Update the node in storage
		if err := s.storage.UpdateNode(rootNode); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to convert root spec to project", err)
		}
	}

	return nil
}

// GetRootNode retrieves the root node without type conversion
func (s *specService) GetRootNode() (models.Node, error) {
	metadata, err := s.storage.GetProjectMetadata()
	if err != nil {
		return nil, err
	}

	if metadata.RootSpecID == nil {
		return nil, models.NewZammError(models.ErrTypeNotFound, "root spec ID not set in project metadata")
	}

	node, err := s.storage.GetNode(*metadata.RootSpecID)
	if err != nil {
		return nil, err
	}

	return node, nil
}

// GetOrphanSpecs retrieves all specs that don't have any parents
func (s *specService) GetOrphanSpecs() ([]*models.Spec, error) {
	return s.storage.GetOrphanSpecs()
}

// validateSpecInput validates specification input data
func (s *specService) validateSpecInput(title, content string) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)

	if title == "" {
		return models.NewZammError(models.ErrTypeValidation, "title cannot be empty")
	}

	if len(title) > 200 {
		return models.NewZammError(models.ErrTypeValidation, "title cannot exceed 200 characters")
	}

	if content == "" {
		return models.NewZammError(models.ErrTypeValidation, "content cannot be empty")
	}

	if len(content) > 50*1024 { // 50KB limit
		return models.NewZammError(models.ErrTypeValidation, "content cannot exceed 50KB")
	}

	return nil
}

// OrganizeNodes moves nodes from generic locations to hierarchical paths
func (s *specService) OrganizeNodes(nodeID string) error {
	if nodeID != "" {
		// Organize specific node only (not its subtree)
		node, err := s.storage.GetNode(nodeID)
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", nodeID, err)
		}

		// Generate slug only for this node and its ancestors if needed
		if err := s.generateSlugForNodeAndAncestors(node); err != nil {
			return fmt.Errorf("failed to generate slugs for node and ancestors: %w", err)
		}

		basePath, err := s.computeNodeBasePath(node)
		if err != nil {
			return fmt.Errorf("failed to compute base path for node %s: %w", nodeID, err)
		}

		return s.organizeSingleNode(node, basePath)
	}

	// Organize all nodes starting from root - generate all missing slugs first
	if err := s.generateMissingSlugs(); err != nil {
		return fmt.Errorf("failed to generate slugs: %w", err)
	}

	rootNode, err := s.GetRootNode()
	if err != nil {
		return fmt.Errorf("failed to get root node: %w", err)
	}

	return s.organizeNodeRecursively(rootNode, DocumentationRoot)
}

func (s *specService) generateMissingSlugs() error {
	nodes, err := s.storage.ListNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if node.GetSlug() == "" && !s.IsRootNode(node) {
			slug := s.sanitizeSlug(node.Title())
			node.SetSlug(slug)
			if err := s.storage.UpdateNode(node); err != nil {
				return fmt.Errorf("failed to update node %s: %w", node.ID(), err)
			}
		}
	}

	return nil
}

// generateSlugForNodeAndAncestors generates slugs only for the specified node and its ancestors
func (s *specService) generateSlugForNodeAndAncestors(node models.Node) error {
	// Generate slug for the current node if it doesn't have one
	if err := s.generateSlugForSingleNode(node); err != nil {
		return err
	}

	// Generate slugs for all ancestors (needed for path computation)
	currentNode := node
	for {
		parents, err := s.GetParents(currentNode.ID())
		if err != nil {
			return fmt.Errorf("failed to get parents for node %s: %w", currentNode.ID(), err)
		}

		if len(parents) == 0 {
			break // Reached the top
		}

		parent := parents[0]
		if err := s.generateSlugForSingleNode(parent); err != nil {
			return err
		}
		currentNode = parent
	}

	return nil
}

// generateSlugForSingleNode generates a slug for a single node if it doesn't already have one
func (s *specService) generateSlugForSingleNode(node models.Node) error {
	if node.GetSlug() == "" && !s.IsRootNode(node) {
		slug := s.sanitizeSlug(node.Title())
		node.SetSlug(slug)
		if err := s.storage.UpdateNode(node); err != nil {
			return fmt.Errorf("failed to update node %s: %w", node.ID(), err)
		}
	}
	return nil
}

func (s *specService) organizeNodeRecursively(node models.Node, basePath string) error {
	// First organize this node
	if err := s.organizeSingleNode(node, basePath); err != nil {
		return err
	}

	// Then recursively organize its children
	children, err := s.GetChildren(node.ID())
	if err != nil {
		return fmt.Errorf("failed to get children for node %s: %w", node.ID(), err)
	}

	if len(children) > 0 {
		slug := s.getNodeSlug(node)
		var childBasePath string

		// Handle root node specially - its children go directly under basePath
		if s.IsRootNode(node) {
			childBasePath = basePath
		} else {
			childBasePath = filepath.Join(basePath, slug)
		}

		for _, child := range children {
			if err := s.organizeNodeRecursively(child, childBasePath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *specService) organizeSingleNode(node models.Node, basePath string) error {
	children, err := s.GetChildren(node.ID())
	if err != nil {
		return fmt.Errorf("failed to get children for node %s: %w", node.ID(), err)
	}

	var newPath string
	slug := s.getNodeSlug(node)

	// Handle root node specially
	if s.IsRootNode(node) {
		if len(children) > 0 {
			// Root node with children goes to docs/README.md
			newPath = filepath.Join(basePath, IndexFile)
		} else {
			// Root node without children goes to docs/README.md
			newPath = filepath.Join(basePath, IndexFile)
		}
	} else {
		// Non-root nodes follow the regular logic
		if len(children) > 0 {
			newPath = filepath.Join(basePath, slug, IndexFile)
		} else {
			newPath = filepath.Join(basePath, slug+".md")
		}
	}

	return s.moveNodeToPath(node, newPath)
}

func (s *specService) moveNodeToPath(node models.Node, newPath string) error {
	fileStorage, ok := s.storage.(*storage.FileStorage)
	if !ok {
		return fmt.Errorf("storage is not FileStorage type")
	}

	currentPath := fileStorage.GetNodeFilePath(node.ID())

	fullNewPath := filepath.Join(filepath.Dir(fileStorage.BaseDir()), newPath)

	if err := os.MkdirAll(filepath.Dir(fullNewPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.Rename(currentPath, fullNewPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return fileStorage.UpdateNodeFilePath(node.ID(), newPath)
}

func (s *specService) IsRootNode(node models.Node) bool {
	metadata, err := s.storage.GetProjectMetadata()
	if err != nil || metadata.RootSpecID == nil {
		return false
	}
	return node.ID() == *metadata.RootSpecID
}

func (s *specService) getNodeSlug(node models.Node) string {
	if slug := node.GetSlug(); slug != "" || s.IsRootNode(node) {
		return slug
	}
	// todo: don't auto-sanitize if missing
	return s.sanitizeSlug(node.Title())
}

func (s *specService) computeNodeBasePath(node models.Node) (string, error) {
	var pathSegments []string
	currentNode := node

	for {
		parents, err := s.GetParents(currentNode.ID())
		if err != nil {
			return "", fmt.Errorf("failed to get parents for node %s: %w", currentNode.ID(), err)
		}

		if len(parents) == 0 {
			// If this is the root node, its base path is just "docs"
			// If this is an orphan (non-root) node, it goes under "docs" too
			pathSegments = append([]string{DocumentationRoot}, pathSegments...)
			break
		}

		parent := parents[0]
		parentSlug := s.getNodeSlug(parent)

		// Only add parent slug to path if it's not empty (i.e., not the root)
		if parentSlug != "" {
			pathSegments = append([]string{parentSlug}, pathSegments...)
		}
		currentNode = parent
	}

	return filepath.Join(pathSegments...), nil
}

func (s *specService) sanitizeSlug(title string) string {
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "untitled"
	}
	return slug
}
