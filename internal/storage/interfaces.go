package storage

import (
	"github.com/yourorg/zamm-mvp/internal/models"
)

// Storage defines the interface for data storage operations
type Storage interface {
	// Initialize storage
	InitializeStorage() error

	// Spec operations
	CreateSpecNode(spec *models.Spec) error
	GetSpecNode(id string) (*models.Spec, error)
	UpdateSpecNode(spec *models.Spec) error
	DeleteSpecNode(id string) error
	ListSpecNodes() ([]*models.Spec, error)

	// SpecCommitLink operations
	CreateSpecCommitLink(link *models.SpecCommitLink) error
	GetSpecCommitLinks(specID string) ([]*models.SpecCommitLink, error)
	DeleteSpecCommitLink(specID string) error
	DeleteSpecCommitLinkByFields(specID, commitID, repoPath string) error
	GetLinksByCommit(commitID, repoPath string) ([]*models.SpecCommitLink, error)
	GetLinksBySpec(specID string) ([]*models.SpecCommitLink, error)
	DeleteLink(specID string) error

	// SpecSpecLink operations
	CreateSpecSpecLink(link *models.SpecSpecLink) error
	GetSpecSpecLinks(specID string, direction models.Direction) ([]*models.SpecSpecLink, error)
	DeleteSpecSpecLink(fromSpecID, toSpecID string) error
	DeleteSpecLinkBySpecs(fromSpecID, toSpecID string) error

	// Hierarchical operations
	GetLinkedSpecs(specID string, direction models.Direction) ([]*models.Spec, error)
	GetOrphanSpecs() ([]*models.Spec, error)

	// ProjectMetadata operations
	GetProjectMetadata() (*models.ProjectMetadata, error)
	SetRootSpecID(specID *string) error
}
