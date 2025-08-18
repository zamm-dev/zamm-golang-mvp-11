package nodes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/common"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
)

type GroupedChildren struct {
	Groups    []ChildGroup
	Ungrouped []models.Node
}

type ChildGroup struct {
	Label    string
	Children []GroupedChild
}

type GroupedChild struct {
	Node     models.Node
	IsGroup  bool
	Label    string
	Children []GroupedChild
}

// NodeDetail encapsulates all state and logic for a spec detail
// (separated from the viewport logic)
type NodeDetail struct {
	node                models.Node
	links               []*models.SpecCommitLink
	implementationNodes []models.Node
	groupedChildren     GroupedChildren
	cursor              int
	table               table.Model
	width               int
	height              int
	linkService         LinkService
}

func NewNodeDetail(linkService LinkService) *NodeDetail {
	if linkService == nil {
		panic("linkService cannot be nil in NewNodeDetail")
	}

	columns := []table.Column{
		{Title: "TYPE", Width: 6},
		{Title: "COMMIT", Width: 8},
		{Title: "REPO", Width: 16},
	}
	commitsTable := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithHeight(5),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.NoColor{}).
		Bold(false)
	commitsTable.SetStyles(s)
	return &NodeDetail{
		table:       commitsTable,
		cursor:      -1,
		linkService: linkService,
	}
}

func (d *NodeDetail) SetSize(width, height int) {
	d.width = width
	d.height = height
	extraPadding := 7
	columns := []table.Column{
		{Title: "TYPE", Width: 6},
		{Title: "COMMIT", Width: 8},
		{Title: "REPO", Width: width - 6 - 8 - extraPadding},
	}
	d.table.SetColumns(columns)
}

func (d *NodeDetail) SetSpec(node models.Node) {
	d.node = node

	// Retrieve links for this node
	if d.linkService != nil && node != nil {
		links, err := d.linkService.GetCommitsForSpec(node.GetID())
		if err != nil {
			d.links = nil
		} else {
			d.links = links
		}

		// Retrieve child nodes for this node
		childNodes, err := d.linkService.GetChildNodes(node.GetID())
		if err != nil {
			childNodes = nil
		}
		d.categorizeChildren(childNodes)
	} else {
		d.links = nil
		d.categorizeChildren(nil)
	}

	d.updateCommitsTable()
	d.cursor = -1
}

func (d *NodeDetail) categorizeChildren(childNodes []models.Node) {
	d.groupedChildren = d.organizeChildrenByGroups(childNodes)

	if d.node != nil && d.node.GetType() == "project" {
		implementations := make([]models.Node, 0)
		others := make([]models.Node, 0)

		for _, child := range d.groupedChildren.Ungrouped {
			if child.GetType() == "implementation" {
				implementations = append(implementations, child)
			} else {
				others = append(others, child)
			}
		}

		d.implementationNodes = implementations
		d.groupedChildren.Ungrouped = others
	} else {
		d.implementationNodes = make([]models.Node, 0)
	}
}

func (d *NodeDetail) organizeChildrenByGroups(allChildren []models.Node) GroupedChildren {
	if d.node == nil || d.node.GetChildGroups() == nil {
		return GroupedChildren{Ungrouped: allChildren}
	}

	childMap := make(map[string]models.Node)
	for _, child := range allChildren {
		childMap[child.GetID()] = child
	}

	groups := []ChildGroup{}
	usedChildIDs := make(map[string]bool)

	for label, value := range *d.node.GetChildGroups() {
		group := ChildGroup{Label: label}
		group.Children = d.processTreeNode(value, childMap, usedChildIDs)
		if len(group.Children) > 0 {
			groups = append(groups, group)
		}
	}

	ungrouped := []models.Node{}
	for _, child := range allChildren {
		if !usedChildIDs[child.GetID()] {
			ungrouped = append(ungrouped, child)
		}
	}

	return GroupedChildren{Groups: groups, Ungrouped: ungrouped}
}

func (d *NodeDetail) processTreeNode(value interface{}, childMap map[string]models.Node, usedChildIDs map[string]bool) []GroupedChild {
	result := []GroupedChild{}

	switch v := value.(type) {
	case string:
		if child, exists := childMap[v]; exists {
			usedChildIDs[v] = true
			result = append(result, GroupedChild{Node: child, IsGroup: false})
		}
	case map[string]interface{}:
		for label, subValue := range v {
			subChildren := d.processTreeNode(subValue, childMap, usedChildIDs)
			if len(subChildren) > 0 {
				result = append(result, GroupedChild{
					IsGroup:  true,
					Label:    label,
					Children: subChildren,
				})
			}
		}
	case []interface{}:
		for _, item := range v {
			subChildren := d.processTreeNode(item, childMap, usedChildIDs)
			result = append(result, subChildren...)
		}
	}

	return result
}

func (d *NodeDetail) GetSelectedChild() models.Node {
	allSelectableChildren := d.getAllSelectableChildren()
	if d.cursor >= 0 && d.cursor < len(allSelectableChildren) {
		return allSelectableChildren[d.cursor]
	}
	return nil
}

func (d *NodeDetail) SelectNextChild() {
	allSelectableChildren := d.getAllSelectableChildren()
	if len(allSelectableChildren) == 0 {
		d.cursor = -1
		return
	}
	d.cursor++
	if d.cursor >= len(allSelectableChildren) {
		d.cursor = len(allSelectableChildren) - 1
	}
}

func (d *NodeDetail) SelectPrevChild() {
	allSelectableChildren := d.getAllSelectableChildren()
	if len(allSelectableChildren) == 0 {
		d.cursor = -1
		return
	}
	d.cursor--
	if d.cursor < 0 {
		d.cursor = 0
	}
}

func (d *NodeDetail) getAllSelectableChildren() []models.Node {
	result := make([]models.Node, 0)
	
	result = append(result, d.implementationNodes...)
	
	result = append(result, d.flattenGroupedChildren(d.groupedChildren.Groups)...)
	
	result = append(result, d.groupedChildren.Ungrouped...)
	
	return result
}

func (d *NodeDetail) flattenGroupedChildren(groups []ChildGroup) []models.Node {
	result := make([]models.Node, 0)
	for _, group := range groups {
		result = append(result, d.flattenGroupedChildList(group.Children)...)
	}
	return result
}

func (d *NodeDetail) flattenGroupedChildList(children []GroupedChild) []models.Node {
	result := make([]models.Node, 0)
	for _, child := range children {
		if child.IsGroup {
			result = append(result, d.flattenGroupedChildList(child.Children)...)
		} else {
			result = append(result, child.Node)
		}
	}
	return result
}

func (d *NodeDetail) ResetCursor() {
	d.cursor = -1
}

func (d *NodeDetail) updateCommitsTable() {
	if d.links == nil {
		d.table.SetRows([]table.Row{})
		return
	}
	rows := make([]table.Row, len(d.links))
	for i, link := range d.links {
		commitID := link.CommitID
		repo := link.RepoPath
		var label string
		switch link.LinkLabel {
		case "implements":
			label = "IMPL"
		case "updates":
			label = "UPDATE"
		case "fixes":
			label = "FIX"
		case "refactors":
			label = "CLEAN"
		case "documents":
			label = "DOC"
		case "tests":
			label = "TEST"
		default:
			label = link.LinkLabel
		}
		rows[i] = table.Row{label, commitID, repo}
	}
	d.table.SetRows(rows)
	d.table.SetHeight(len(rows) + 2)
}


// tea.Model interface implementation
func (d *NodeDetail) Init() tea.Cmd {
	return nil
}

func (d *NodeDetail) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return d, nil
}

func (d *NodeDetail) View() string {
	// Handle case where node hasn't been set yet
	if d.node == nil {
		return "No specification selected"
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString(fmt.Sprintf("%s\n%s\n\n%s\n\n", d.node.GetTitle(), strings.Repeat("=", d.width), d.node.GetContent()))
	if len(d.links) == 0 {
		contentBuilder.WriteString("[No linked commits found]\n")
	} else {
		contentBuilder.WriteString(d.table.View())
	}

	contentBuilder.WriteString("\n\n")

	// For project nodes, always show implementations section
	if d.node.GetType() == "project" {
		contentBuilder.WriteString("Implementations:\n")
		if len(d.implementationNodes) == 0 {
			contentBuilder.WriteString("  - no implementations\n")
		} else {
			for i, childNode := range d.implementationNodes {
				d.renderChildNode(&contentBuilder, childNode, i)
			}
		}
		contentBuilder.WriteString("\n")
	}

	// Display regular children section
	contentBuilder.WriteString("Child Nodes:\n")
	
	cursorIndex := len(d.implementationNodes)
	
	for _, group := range d.groupedChildren.Groups {
		contentBuilder.WriteString(fmt.Sprintf("  %s:\n", group.Label))
		cursorIndex = d.renderGroupedChildren(&contentBuilder, group.Children, cursorIndex, 2)
	}
	
	if len(d.groupedChildren.Ungrouped) == 0 && len(d.groupedChildren.Groups) == 0 {
		contentBuilder.WriteString("  -\n")
	} else {
		for _, childNode := range d.groupedChildren.Ungrouped {
			d.renderChildNode(&contentBuilder, childNode, cursorIndex)
			cursorIndex++
		}
	}

	// Use lipgloss to constrain the entire output to the component width
	style := lipgloss.NewStyle().Width(d.width)
	return style.Render(contentBuilder.String())
}

func (d *NodeDetail) renderGroupedChildren(contentBuilder *strings.Builder, children []GroupedChild, startCursor int, indent int) int {
	cursorIndex := startCursor
	indentStr := strings.Repeat(" ", indent)
	
	for _, child := range children {
		if child.IsGroup {
			contentBuilder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, child.Label))
			cursorIndex = d.renderGroupedChildren(contentBuilder, child.Children, cursorIndex, indent+2)
		} else {
			nodeTitle := child.Node.GetTitle()
			if len(nodeTitle) > d.width-indent-2 && d.width > indent+5 {
				nodeTitle = nodeTitle[:d.width-indent-5] + "..."
			}
			if cursorIndex == d.cursor {
				contentBuilder.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("%s> %s", indentStr, nodeTitle)))
				contentBuilder.WriteString("\n")
			} else {
				contentBuilder.WriteString(fmt.Sprintf("%s  %s\n", indentStr, nodeTitle))
			}
			cursorIndex++
		}
	}
	
	return cursorIndex
}

func (d *NodeDetail) renderChildNode(contentBuilder *strings.Builder, childNode models.Node, cursorIndex int) {
	nodeTitle := childNode.GetTitle()
	if len(nodeTitle) > d.width-4 && d.width > 7 {
		nodeTitle = nodeTitle[:d.width-7] + "..."
	}
	if cursorIndex == d.cursor {
		contentBuilder.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("  > %s", nodeTitle)))
		contentBuilder.WriteString("\n")
	} else {
		contentBuilder.WriteString(fmt.Sprintf("    %s\n", nodeTitle))
	}
}
