package storage

import "github.com/yourorg/zamm-mvp/internal/models"

// Storage defines the interface for all storage operations
type Storage interface {
	// Spec operations
	CreateSpec(spec *models.SpecNode) error
	GetSpec(id string) (*models.SpecNode, error)
	GetSpecByStableID(stableID string, version int) (*models.SpecNode, error)
	GetLatestSpecByStableID(stableID string) (*models.SpecNode, error)
	ListSpecs() ([]*models.SpecNode, error)
	UpdateSpec(spec *models.SpecNode) error
	DeleteSpec(id string) error

	// Link operations
	CreateLink(link *models.SpecCommitLink) error
	GetLink(id string) (*models.SpecCommitLink, error)
	GetLinksBySpec(specID string) ([]*models.SpecCommitLink, error)
	GetLinksByCommit(commitID, repoPath string) ([]*models.SpecCommitLink, error)
	DeleteLink(id string) error

	// Spec hierarchy operations (DAG)
	CreateSpecLink(link *models.SpecSpecLink) error
	GetSpecLink(id string) (*models.SpecSpecLink, error)
	GetParentSpecs(specID string) ([]*models.SpecSpecLink, error)
	GetChildSpecs(specID string) ([]*models.SpecSpecLink, error)
	DeleteSpecLink(id string) error
	DeleteSpecLinkBySpecs(parentSpecID, childSpecID string) error
	// DAG validation
	WouldCreateCycle(parentSpecID, childSpecID string) (bool, error)

	// Utility
	RunMigration(migrationSQL string) error
	BackupDatabase(backupPath string) error
	Close() error
}
