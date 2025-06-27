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
