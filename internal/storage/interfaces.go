package storage

import (
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

// Storage defines the interface for data storage operations
type Storage interface {
	// Initialize storage
	InitializeStorage() error

	// Get base directory
	BaseDir() string

	// Node operations
	WriteNode(node models.Node) error
	GetNode(id string) (models.Node, error)
	WriteNodeWithChildren(node models.Node, childGrouping models.ChildGroup) error
	DeleteNode(id string) error
	ListNodes() ([]models.Node, error)

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
	GetLinkedNodes(nodeID string, direction models.Direction) ([]models.Node, error)
	GetOrphanSpecs() ([]*models.Spec, error)

	// ProjectMetadata operations
	GetProjectMetadata() (*models.ProjectMetadata, error)
	SetRootSpecID(specID *string) error
}
