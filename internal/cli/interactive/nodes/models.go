package nodes

import (
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
)

func isImplementationNode(node models.Node) bool {
	return node.GetType() == "implementation"
}

func GetOrganizedChildren(ss services.SpecService, node models.Node) (models.ChildGroup, error) {
	cg := node.GetChildGrouping()
	allChildren, err := ss.GetChildren(node.GetID())
	if err != nil {
		return cg, err
	}

	cg.AppendUnmatched(allChildren)
	cg.UngroupedLabel = "Children"

	if node.GetType() == "project" {
		cg.Regroup("Implementations", isImplementationNode)
	}

	return cg, nil
}
