package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/storage"
)

// SpecService interface defines operations for managing specifications
type SpecService interface {
	CreateSpec(title, content string) (*models.Spec, error)
	GetSpec(id string) (*models.Spec, error)
	UpdateSpec(id, title, content string) (*models.Spec, error)
	ListSpecs() ([]*models.Spec, error)
	DeleteSpec(id string) error

	// Hierarchical operations
	AddChildToParent(childSpecID, parentSpecID, label string) (*models.SpecSpecLink, error)
	RemoveChildFromParent(childSpecID, parentSpecID string) error
	GetParents(specID string) ([]*models.Spec, error)
	GetChildren(specID string) ([]*models.Spec, error)

	// Root spec operations
	InitializeRootSpec() error
	GetRootSpec() (*models.Spec, error)
	GetOrphanSpecs() ([]*models.Spec, error)

	// Migration operations
	MigrateSpecTypeField() (int, error)
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

	if err := s.storage.CreateSpecNode(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// GetSpec retrieves a specification by ID
func (s *specService) GetSpec(id string) (*models.Spec, error) {
	if id == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.GetSpecNode(id)
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
	spec, err := s.storage.GetSpecNode(id)
	if err != nil {
		return nil, err
	}

	// Update fields
	spec.Title = strings.TrimSpace(title)
	spec.Content = strings.TrimSpace(content)
	spec.Type = "specification"

	// Save changes
	if err := s.storage.UpdateSpecNode(spec); err != nil {
		return nil, err
	}

	return spec, nil
}

// ListSpecs retrieves all specifications
func (s *specService) ListSpecs() ([]*models.Spec, error) {
	return s.storage.ListSpecNodes()
}

// DeleteSpec deletes a specification
func (s *specService) DeleteSpec(id string) error {
	if id == "" {
		return models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.DeleteSpecNode(id)
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
		return nil, models.NewZammError(models.ErrTypeValidation, "cannot link a spec to itself")
	}

	// Verify both specs exist
	_, err := s.storage.GetSpecNode(childSpecID)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeValidation, "child spec not found")
	}
	_, err = s.storage.GetSpecNode(parentSpecID)
	if err != nil {
		return nil, models.NewZammError(models.ErrTypeValidation, "parent spec not found")
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

// GetParents retrieves all parent specs for a given spec
func (s *specService) GetParents(specID string) ([]*models.Spec, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.GetLinkedSpecs(specID, models.Outgoing)
}

// GetChildren retrieves all child specs for a given spec
func (s *specService) GetChildren(specID string) ([]*models.Spec, error) {
	if specID == "" {
		return nil, models.NewZammError(models.ErrTypeValidation, "spec ID cannot be empty")
	}

	return s.storage.GetLinkedSpecs(specID, models.Incoming)
}

// InitializeRootSpec creates the root specification if it doesn't exist
// and links all orphaned specs to it
func (s *specService) InitializeRootSpec() error {
	// Check if root spec already exists
	rootSpec, err := s.GetRootSpec()

	if err != nil || rootSpec == nil {
		// Create root spec
		newRootSpec, err := s.CreateSpec("New Project", "Requirement: This project should exist.")
		if err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to create root spec", err)
		}

		// Set it as the root spec in metadata
		err = s.storage.SetRootSpecID(&newRootSpec.ID)
		if err != nil {
			return models.NewZammErrorWithCause(models.ErrTypeStorage, "failed to set root spec ID", err)
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

	return s.storage.GetSpecNode(*metadata.RootSpecID)
}

// GetOrphanSpecs retrieves all specs that don't have any parents
func (s *specService) GetOrphanSpecs() ([]*models.Spec, error) {
	return s.storage.GetOrphanSpecs()
}

// MigrateSpecTypeField migrates the Type field of all specs to "specification"
func (s *specService) MigrateSpecTypeField() (int, error) {
	specs, err := s.ListSpecs()
	if err != nil {
		return 0, err
	}

	updatedCount := 0
	for _, spec := range specs {
		// Check if the actual JSON file has a type field by reading it directly
		path := filepath.Join(s.storage.BaseDir(), "specs", spec.ID+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable files
		}

		var jsonSpec map[string]interface{}
		if err := json.Unmarshal(data, &jsonSpec); err != nil {
			continue // skip invalid JSON
		}

		// Only update if the JSON file doesn't have a type field
		if _, ok := jsonSpec["type"]; !ok {
			// Set the type field in the spec object
			spec.Type = "specification"
			// Use the storage layer to write it back
			if err := s.storage.UpdateSpecNode(spec); err != nil {
				continue // skip if can't write
			}
			updatedCount++
		}
	}
	return updatedCount, nil
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
