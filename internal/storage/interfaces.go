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
	GetLinkedSpecs(specID string, direction models.Direction) ([]*models.SpecNode, error)
	DeleteSpecLink(id string) error
	DeleteSpecLinkBySpecs(fromSpecID, toSpecID string) error
	// DAG validation
	WouldCreateCycle(fromSpecID, toSpecID string) (bool, error)

	// Utility
	BackupDatabase(backupPath string) error
	Close() error

	// Migration operations
	RunMigrationsIfNeeded() error
	GetMigrationVersion() (uint, bool, error)
	ForceMigrationVersion(version uint) error
}
