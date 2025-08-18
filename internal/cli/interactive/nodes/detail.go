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

type ChildGroup struct {
	Label    string
	Children []models.Node
	Groups   []ChildGroup
}

// NodeDetail encapsulates all state and logic for a spec detail
// (separated from the viewport logic)
type NodeDetail struct {
	node                models.Node
	links               []*models.SpecCommitLink
	implementationNodes []models.Node
	childGroups         []ChildGroup
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
	d.childGroups = d.organizeChildrenByGroups(childNodes)

	if d.node != nil && d.node.GetType() == "project" {
		implementations := make([]models.Node, 0)
		
		d.childGroups = d.extractImplementations(d.childGroups, &implementations)
		
		d.implementationNodes = implementations
	} else {
		d.implementationNodes = make([]models.Node, 0)
	}
}

func (d *NodeDetail) organizeChildrenByGroups(allChildren []models.Node) []ChildGroup {
	if d.node == nil || d.node.GetChildGroups() == nil {
		return []ChildGroup{{Children: allChildren}}
	}

	childMap := make(map[string]models.Node)
	for _, child := range allChildren {
		childMap[child.GetID()] = child
	}

	groups := []ChildGroup{}
	usedChildIDs := make(map[string]bool)

	for label, value := range *d.node.GetChildGroups() {
		group := d.processTreeNode(label, value, childMap, usedChildIDs)
		if len(group.Children) > 0 || len(group.Groups) > 0 {
			groups = append(groups, group)
		}
	}

	ungrouped := []models.Node{}
	for _, child := range allChildren {
		if !usedChildIDs[child.GetID()] {
			ungrouped = append(ungrouped, child)
		}
	}
	
	if len(ungrouped) > 0 {
		groups = append(groups, ChildGroup{Children: ungrouped})
	}

	return groups
}

func (d *NodeDetail) processTreeNode(label string, value interface{}, childMap map[string]models.Node, usedChildIDs map[string]bool) ChildGroup {
	group := ChildGroup{Label: label}

	switch v := value.(type) {
	case string:
		if child, exists := childMap[v]; exists {
			usedChildIDs[v] = true
			group.Children = append(group.Children, child)
		}
	case map[string]interface{}:
		for subLabel, subValue := range v {
			subGroup := d.processTreeNode(subLabel, subValue, childMap, usedChildIDs)
			if len(subGroup.Children) > 0 || len(subGroup.Groups) > 0 {
				group.Groups = append(group.Groups, subGroup)
			}
		}
	case []interface{}:
		for _, item := range v {
			switch itemValue := item.(type) {
			case string:
				if child, exists := childMap[itemValue]; exists {
					usedChildIDs[itemValue] = true
					group.Children = append(group.Children, child)
				}
			case map[string]interface{}:
				for subLabel, subValue := range itemValue {
					subGroup := d.processTreeNode(subLabel, subValue, childMap, usedChildIDs)
					if len(subGroup.Children) > 0 || len(subGroup.Groups) > 0 {
						group.Groups = append(group.Groups, subGroup)
					}
				}
			}
		}
	}

	return group
}

func (d *NodeDetail) extractImplementations(groups []ChildGroup, implementations *[]models.Node) []ChildGroup {
	result := []ChildGroup{}
	
	for _, group := range groups {
		newGroup := ChildGroup{Label: group.Label}
		
		for _, child := range group.Children {
			if child.GetType() == "implementation" {
				*implementations = append(*implementations, child)
			} else {
				newGroup.Children = append(newGroup.Children, child)
			}
		}
		
		newGroup.Groups = d.extractImplementations(group.Groups, implementations)
		
		if len(newGroup.Children) > 0 || len(newGroup.Groups) > 0 {
			result = append(result, newGroup)
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
	result = append(result, d.flattenChildGroups(d.childGroups)...)
	
	return result
}

func (d *NodeDetail) flattenChildGroups(groups []ChildGroup) []models.Node {
	result := make([]models.Node, 0)
	for _, group := range groups {
		result = append(result, group.Children...)
		result = append(result, d.flattenChildGroups(group.Groups)...)
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
				d.renderChildNode(&contentBuilder, childNode, i, 1)
			}
		}
		contentBuilder.WriteString("\n")
	}

	// Display regular children section
	contentBuilder.WriteString("Child Nodes:\n")
	
	cursorIndex := len(d.implementationNodes)
	
	if len(d.childGroups) == 0 {
		contentBuilder.WriteString("  -\n")
	} else {
		d.renderChildGroups(&contentBuilder, d.childGroups, cursorIndex, 1)
	}

	// Use lipgloss to constrain the entire output to the component width
	style := lipgloss.NewStyle().Width(d.width)
	return style.Render(contentBuilder.String())
}

func (d *NodeDetail) renderChildGroups(contentBuilder *strings.Builder, groups []ChildGroup, startCursor int, indent int) int {
	cursorIndex := startCursor
	indentStr := strings.Repeat(" ", indent)
	
	for _, group := range groups {
		if group.Label != "" {
			contentBuilder.WriteString(fmt.Sprintf("%s%s:\n", indentStr, group.Label))
		}
		
		childIndent := indent + 1
		if group.Label != "" {
			childIndent = indent + 2
		}
		for _, child := range group.Children {
			cursorIndex = d.renderChildNode(contentBuilder, child, cursorIndex, childIndent)
		}
		
		nestedIndent := indent + 1
		if group.Label != "" {
			nestedIndent = indent + 2
		}
		cursorIndex = d.renderChildGroups(contentBuilder, group.Groups, cursorIndex, nestedIndent)
	}
	
	return cursorIndex
}

func (d *NodeDetail) renderChildNode(contentBuilder *strings.Builder, childNode models.Node, cursorIndex int, indent int) int {
	indentStr := strings.Repeat(" ", indent*2)
	nodeTitle := childNode.GetTitle()
	if len(nodeTitle) > d.width-len(indentStr)-2 && d.width > len(indentStr)+5 {
		nodeTitle = nodeTitle[:d.width-len(indentStr)-5] + "..."
	}
	if cursorIndex == d.cursor {
		contentBuilder.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("%s> %s", indentStr, nodeTitle)))
		contentBuilder.WriteString("\n")
	} else {
		contentBuilder.WriteString(fmt.Sprintf("%s  %s\n", indentStr, nodeTitle))
	}
	return cursorIndex + 1
}
