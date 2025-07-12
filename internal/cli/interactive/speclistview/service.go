package speclistview

import (
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// LinkService interface for data access
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
	GetChildSpecs(specID string) ([]*models.Spec, error)
	GetSpecByID(specID string) (*interactive.Spec, error)
	GetParentSpec(specID string) (*interactive.Spec, error)
	GetRootSpec() (*interactive.Spec, error)
}
