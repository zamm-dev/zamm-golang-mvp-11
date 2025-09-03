package models

type ChildGroup struct {
	Children       []Node
	Groups         map[string]*ChildGroup
	UngroupedLabel string
}

func (cg *ChildGroup) Contains(node Node) bool {
	for _, child := range cg.Children {
		if child.ID() == node.ID() {
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

func (cg *ChildGroup) Size() int {
	size := len(cg.Children)
	for _, group := range cg.Groups {
		size += group.Size()
	}
	return size
}

func (cg *ChildGroup) NodeAt(index int) Node {
	if index < 0 || index >= cg.Size() {
		return nil
	}
	for _, group := range cg.Groups {
		if index < group.Size() {
			return group.NodeAt(index)
		}
		index -= group.Size()
	}
	if index < len(cg.Children) {
		return cg.Children[index]
	}
	return nil
}

func (cg *ChildGroup) AllNodes() []Node {
	var allNodes []Node
	for _, group := range cg.Groups {
		allNodes = append(allNodes, group.AllNodes()...)
	}
	allNodes = append(allNodes, cg.Children...)
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
	for label, group := range cg.Groups {
		removed = append(removed, group.Remove(predicate)...)
		if len(group.Children) == 0 && len(group.Groups) == 0 {
			delete(cg.Groups, label)
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
		if cg.Groups == nil {
			cg.Groups = make(map[string]*ChildGroup)
		}
		cg.Groups[label] = &ChildGroup{Children: removed}
	}
}

func (cg *ChildGroup) Render(renderer ChildGroupRenderer) {
	cg.recursivelyRender(0, renderer)
}

func (cg *ChildGroup) recursivelyRender(nestingLevel int, renderer ChildGroupRenderer) {
	renderUngroupedEnclosure := cg.UngroupedLabel != "" && len(cg.Children) > 0

	for groupLabel, subGroup := range cg.Groups {
		renderer.RenderGroupStart(nestingLevel, groupLabel)
		subGroup.recursivelyRender(nestingLevel+1, renderer)
		renderer.RenderGroupEnd(nestingLevel)
	}

	if renderUngroupedEnclosure {
		renderer.RenderGroupStart(nestingLevel, cg.UngroupedLabel)
		for _, child := range cg.Children {
			renderer.RenderNode(nestingLevel+1, child)
		}
		renderer.RenderGroupEnd(nestingLevel)
	} else {
		for _, child := range cg.Children {
			renderer.RenderNode(nestingLevel, child)
		}
	}
}

type ChildGroupRenderer interface {
	RenderGroupStart(nestingLevel int, label string)
	RenderGroupEnd(nestingLevel int)
	RenderNode(nestingLevel int, node Node)
}
