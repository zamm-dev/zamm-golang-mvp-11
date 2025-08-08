package speclistview

import (
	"github.com/yourorg/zamm-mvp/internal/models"
)

// LinkService interface for data access
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
	GetChildNodes(specID string) ([]models.Node, error)
	GetNodeByID(specID string) (models.Node, error)
	GetParentNode(specID string) (models.Node, error)
	GetRootNode() (models.Node, error)
}
