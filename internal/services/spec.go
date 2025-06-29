package services

import (
	"strings"

	"github.com/google/uuid"
	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// SpecService interface defines operations for managing specifications
type SpecService interface {
	CreateSpec(title, content string) (*models.SpecNode, error)
	GetSpec(id string) (*models.SpecNode, error)
	UpdateSpec(id, title, content string) (*models.SpecNode, error)
	ListSpecs() ([]*models.SpecNode, error)
	DeleteSpec(id string) error

	// Hierarchical operations
	LinkSpecs(parentSpecID, childSpecID, linkType string) (*models.SpecSpecLink, error)
	UnlinkSpecs(parentSpecID, childSpecID string) error
	GetParentSpecs(specID string) ([]*models.SpecSpecLink, error)
	GetChildSpecs(specID string) ([]*models.SpecSpecLink, error)
	GetSpecsWithHierarchy() ([]*SpecWithHierarchy, error)
}

// SpecWithHierarchy represents a spec with its hierarchical relationships
type SpecWithHierarchy struct {
	*models.SpecNode
	Parents  []*models.SpecSpecLink `json:"parents"`
	Children []*models.SpecSpecLink `json:"children"`
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
func (s *specService) CreateSpec(title, content string) (*models.SpecNode, error) {
	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	spec := &models.SpecNode{
		ID:       uuid.New().String(),
		StableID: uuid.New().String(),
		Version:  1,
		Title:    strings.TrimSpace(title),
		Content:  strings.TrimSpace(content),
		NodeType: "spec",
	}

	if err := s.storage.CreateSpec(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// GetSpec retrieves a specification by ID
func (s *specService) GetSpec(id string) (*models.SpecNode, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.GetSpec(id)
}

// UpdateSpec updates an existing specification
func (s *specService) UpdateSpec(id, title, content string) (*models.SpecNode, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	// Validate input
	if err := s.validateSpecInput(title, content); err != nil {
		return nil, err
	}

	// Get existing spec
	spec, err := s.storage.GetSpec(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	spec.Title = strings.TrimSpace(title)
	spec.Content = strings.TrimSpace(content)

	// Save changes
	if err := s.storage.UpdateSpec(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// ListSpecs retrieves all specifications
func (s *specService) ListSpecs() ([]*models.SpecNode, error) {
	return s.storage.ListSpecs()
}

// DeleteSpec deletes a specification
func (s *specService) DeleteSpec(id string) error {
	if id == "" {
		return models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.DeleteSpec(id)
}

// LinkSpecs creates a hierarchical link between two specifications
func (s *specService) LinkSpecs(parentSpecID, childSpecID, linkType string) (*models.SpecSpecLink, error) {
	// Validate input
	if parentSpecID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "parent spec ID cannot be empty")
	}
	if childSpecID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "child spec ID cannot be empty")
	}
	if parentSpecID == childSpecID {
		return nil, models.NewZammError(models.ErrTypeValidation, "cannot link a spec to itself")
	}
	if linkType == "" {
		linkType = "child"
	}

	// Verify both specs exist
	_, err := s.storage.GetSpec(parentSpecID)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeValidation, "parent spec not found")
	}
	_, err = s.storage.GetSpec(childSpecID)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeValidation, "child spec not found")
	}

	link := &models.SpecSpecLink{
		FromSpecID: parentSpecID,
		ToSpecID:   childSpecID,
		LinkType:   linkType,
	}

	if err := s.storage.CreateSpecLink(link); err != nil {
		return nil, err
	}

	return link, nil
}

// UnlinkSpecs removes a hierarchical link between two specifications
func (s *specService) UnlinkSpecs(parentSpecID, childSpecID string) error {
	if parentSpecID == "" {
		return models.NewZammError(models.ErrTypeValidation, "parent spec ID cannot be empty")
	}
	if childSpecID == "" {
		return models.NewZammError(models.ErrTypeValidation, "child spec ID cannot be empty")
	}

	return s.storage.DeleteSpecLinkBySpecs(parentSpecID, childSpecID)
}

// GetParentSpecs retrieves all parent specs for a given spec
func (s *specService) GetParentSpecs(specID string) ([]*models.SpecSpecLink, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.GetParentSpecs(specID)
}

// GetChildSpecs retrieves all child specs for a given spec
func (s *specService) GetChildSpecs(specID string) ([]*models.SpecSpecLink, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.GetChildSpecs(specID)
}

// GetSpecsWithHierarchy retrieves all specs with their hierarchical relationships
func (s *specService) GetSpecsWithHierarchy() ([]*SpecWithHierarchy, error) {
	specs, err := s.storage.ListSpecs()
	if err != nil {
		return nil, err
	}

	result := make([]*SpecWithHierarchy, len(specs))
	for i, spec := range specs {
		parents, err := s.storage.GetParentSpecs(spec.ID)
		if err != nil {
			return nil, err
		}

		children, err := s.storage.GetChildSpecs(spec.ID)
		if err != nil {
			return nil, err
		}

		result[i] = &SpecWithHierarchy{
			SpecNode: spec,
			Parents:  parents,
			Children: children,
		}
	}

	return result, nil
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
