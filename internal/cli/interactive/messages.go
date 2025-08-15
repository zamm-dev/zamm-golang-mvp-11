package interactive

import (
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

type NodesLoadedMsg struct {
	nodes []Spec
	err   error
}

type LinksLoadedMsg struct {
	links []LinkItem
	err   error
}

type OperationCompleteMsg struct {
	message string
}

type ReturnToSpecListMsg struct{}

type NavigateToNodeMsg struct {
	nodeID string
}

type SetCurrentNodeMsg struct {
	node models.Node
}

type LinkItem struct {
	ID        string
	CommitID  string
	RepoPath  string
	LinkLabel string
}
