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

// NodeDetail encapsulates all state and logic for a spec detail
// (separated from the viewport logic)
type NodeDetail struct {
	node                models.Node
	links               []*models.SpecCommitLink
	implementationNodes []models.Node
	regularChildNodes   []models.Node
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
	// Check if this is a project node
	if d.node != nil && d.node.GetType() == "project" {
		// Separate implementation nodes from other children
		implementations := make([]models.Node, 0)
		others := make([]models.Node, 0)

		for _, child := range childNodes {
			if child.GetType() == "implementation" {
				implementations = append(implementations, child)
			} else {
				others = append(others, child)
			}
		}

		d.implementationNodes = implementations
		d.regularChildNodes = others
	} else {
		// For non-project nodes, all children are "regular" children
		d.implementationNodes = make([]models.Node, 0)
		d.regularChildNodes = childNodes
	}
}

func (d *NodeDetail) GetSelectedChild() models.Node {
	totalChildren := len(d.implementationNodes) + len(d.regularChildNodes)
	if d.cursor >= 0 && d.cursor < totalChildren {
		if d.cursor < len(d.implementationNodes) {
			return d.implementationNodes[d.cursor]
		} else {
			otherIndex := d.cursor - len(d.implementationNodes)
			return d.regularChildNodes[otherIndex]
		}
	}
	return nil
}

func (d *NodeDetail) SelectNextChild() {
	totalChildren := len(d.implementationNodes) + len(d.regularChildNodes)
	if totalChildren == 0 {
		d.cursor = -1
		return
	}
	d.cursor++
	if d.cursor >= totalChildren {
		d.cursor = totalChildren - 1
	}
}

func (d *NodeDetail) SelectPrevChild() {
	totalChildren := len(d.implementationNodes) + len(d.regularChildNodes)
	if totalChildren == 0 {
		d.cursor = -1
		return
	}
	d.cursor--
	if d.cursor < 0 {
		d.cursor = 0
	}
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

func (d *NodeDetail) renderChildNode(contentBuilder *strings.Builder, childNode models.Node, cursorIndex int) {
	nodeTitle := childNode.GetTitle()
	if len(nodeTitle) > d.width-2 && d.width > 5 {
		nodeTitle = nodeTitle[:d.width-5] + "..."
	}
	if cursorIndex == d.cursor {
		contentBuilder.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("> %s", nodeTitle)))
		contentBuilder.WriteString("\n")
	} else {
		fmt.Fprintf(contentBuilder, "  %s\n", nodeTitle)
	}
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
	if len(d.regularChildNodes) == 0 {
		contentBuilder.WriteString("  -\n")
	} else {
		for i, childNode := range d.regularChildNodes {
			cursorIndex := len(d.implementationNodes) + i
			d.renderChildNode(&contentBuilder, childNode, cursorIndex)
		}
	}

	// Use lipgloss to constrain the entire output to the component width
	style := lipgloss.NewStyle().Width(d.width)
	return style.Render(contentBuilder.String())
}
