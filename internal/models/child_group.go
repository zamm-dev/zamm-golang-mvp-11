package models

type ChildGroup struct {
	Label          string // can be empty for root node
	Children       []Node
	Groups         []ChildGroup
	UngroupedLabel string
}

func (cg *ChildGroup) Contains(node Node) bool {
	for _, child := range cg.Children {
		if child.GetID() == node.GetID() {
			return true
		}
	}
	for _, group := range cg.Groups {
		if group.Contains(node) {
			return true
		}
	}
	return false
}

func (cg *ChildGroup) AllNodes() []Node {
	var allNodes []Node
	allNodes = append(allNodes, cg.Children...)
	for _, group := range cg.Groups {
		allNodes = append(allNodes, group.AllNodes()...)
	}
	return allNodes
}

func (cg *ChildGroup) IsEmpty() bool {
	return len(cg.Children) == 0 && len(cg.Groups) == 0
}

func (cg *ChildGroup) AppendUnmatched(nodes []Node) {
	for _, node := range nodes {
		if !cg.Contains(node) {
			cg.Children = append(cg.Children, node)
		}
	}
}

func (cg *ChildGroup) Remove(predicate func(Node) bool) []Node {
	var removed []Node
	removed, cg.Children = partitionNodes(cg.Children, predicate)
	for i := 0; i < len(cg.Groups); i++ {
		removed = append(removed, cg.Groups[i].Remove(predicate)...)
		if len(cg.Groups[i].Children) == 0 {
			cg.Groups = append(cg.Groups[:i], cg.Groups[i+1:]...)
			i--
		}
	}
	return removed
}

func partitionNodes(nodes []Node, predicate func(Node) bool) ([]Node, []Node) {
	var matching []Node
	var unmatching []Node
	for _, node := range nodes {
		if predicate(node) {
			matching = append(matching, node)
		} else {
			unmatching = append(unmatching, node)
		}
	}
	return matching, unmatching
}

func (cg *ChildGroup) Regroup(label string, filter func(Node) bool) {
	removed := cg.Remove(filter)
	if len(removed) > 0 {
		cg.Groups = append([]ChildGroup{{Label: label, Children: removed}}, cg.Groups...)
	}
}

func (cg *ChildGroup) Render(renderer ChildGroupRenderer) {
	cg.recursivelyRender(-1, renderer)
}

func (cg *ChildGroup) recursivelyRender(nestingLevel int, renderer ChildGroupRenderer) {
	renderOverallEnclosure := nestingLevel >= 0
	renderUngroupedEnclosure := cg.UngroupedLabel != "" && len(cg.Children) > 0

	if renderOverallEnclosure {
		renderer.RenderGroupStart(nestingLevel, cg.Label)
	}

	for _, subGroup := range cg.Groups {
		subGroup.recursivelyRender(nestingLevel+1, renderer)
	}
	if renderUngroupedEnclosure {
		ungroupedLevel := nestingLevel + 1
		renderer.RenderGroupStart(ungroupedLevel, cg.UngroupedLabel)
		for _, child := range cg.Children {
			renderer.RenderNode(ungroupedLevel, child)
		}
		if renderUngroupedEnclosure {
			renderer.RenderGroupEnd(ungroupedLevel)
		}
	} else {
		if nestingLevel < 0 {
			nestingLevel = 0 // Ensure we start at level 0 for the root group
		}
		for _, child := range cg.Children {
			renderer.RenderNode(nestingLevel, child)
		}
	}

	if renderOverallEnclosure {
		renderer.RenderGroupEnd(nestingLevel)
	}
}

type ChildGroupRenderer interface {
	RenderGroupStart(nestingLevel int, label string)
	RenderGroupEnd(nestingLevel int)
	RenderNode(nestingLevel int, node Node)
}
