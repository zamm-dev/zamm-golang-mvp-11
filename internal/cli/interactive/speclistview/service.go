package speclistview

import (
	"github.com/yourorg/zamm-mvp/internal/models"
)

// LinkService interface for data access
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
	GetChildSpecs(specID string) ([]*models.Spec, error)
	GetSpecByID(specID string) (*models.Spec, error)
	GetParentSpec(specID string) (*models.Spec, error)
	GetRootSpec() (*models.Spec, error)
}
