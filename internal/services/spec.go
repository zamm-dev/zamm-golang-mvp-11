package services

import (
	"strings"

	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// SpecService interface defines operations for managing specifications
type SpecService interface {
	CreateSpec(title, content string) (*models.Spec, error)
	CreateProject(title, content string) (*models.Project, error)
	GetNode(id string) (models.Node, error)
	GetProject(id string) (*models.Project, error)
	UpdateSpec(id, title, content string) (*models.Spec, error)
	ListNodes() ([]models.Node, error)
	DeleteSpec(id string) error

	// Hierarchical operations
	AddChildToParent(childSpecID, parentSpecID, label string) (*models.SpecSpecLink, error)
	RemoveChildFromParent(childSpecID, parentSpecID string) error
	GetParents(specID string) ([]models.Node, error)
	GetChildren(specID string) ([]models.Node, error)

	// Root spec operations
	InitializeRootSpec() error
	GetRootSpec() (*models.Spec, error)
	GetOrphanSpecs() ([]*models.Spec, error)
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
	spec.Title = strings.TrimSpace(title)
	spec.Content = strings.TrimSpace(content)
	spec.Type = "specification"

	// Save changes
	if err := s.storage.UpdateNode(spec); err != nil {
		return nil, err
	}

	return spec, nil
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
		err = s.storage.SetRootSpecID(&newRootProject.ID)
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
	if rootNode.GetType() == "project" {
		return nil
	} else { // otherwise, convert it to a Project
		// Create a new Project with the same content
		newProject := models.NewProject(rootNode.GetTitle(), rootNode.GetContent())
		// Keep the same ID to maintain references
		newProject.ID = rootNode.GetID()

		// Update the node in storage
		if err := s.storage.UpdateNode(newProject); err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to convert root spec to project", err)
		}
	}

	return nil
}

// GetRootSpec retrieves the root specification
func (s *specService) GetRootSpec() (*models.Spec, error) {
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

	// Handle both Spec and Project types for backward compatibility
	switch n := node.(type) {
	case *models.Spec:
		return n, nil
	case *models.Project:
		// Convert Project to Spec for backward compatibility
		spec := &models.Spec{
			NodeBase: models.NodeBase{
				ID:      n.ID,
				Title:   n.Title,
				Content: n.Content,
				Type:    "specification", // Present as spec for compatibility
			},
		}
		return spec, nil
	default:
		return nil, models.NewZammError(models.ErrTypeValidation, "root node is not a spec or project")
	}
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
