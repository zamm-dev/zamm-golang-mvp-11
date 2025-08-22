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
	node          models.Node
	links         []*models.SpecCommitLink
	childGrouping models.ChildGroup
	cursor        int
	table         table.Model
	width         int
	height        int
	linkService   LinkService
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

	links, err := d.linkService.GetCommitsForSpec(node.GetID())
	if err != nil {
		d.links = nil
	} else {
		d.links = links
	}

	// Retrieve child nodes for this node
	childNodes, err := d.linkService.GetChildNodes(node.GetID())
	if err != nil {
		childNodes = []models.Node{}
	}

	d.updateChildGroups(childNodes)

	d.updateCommitsTable()
	d.cursor = -1
}

func isImplementationNode(node models.Node) bool {
	return node.GetType() == "implementation"
}

func (d *NodeDetail) updateChildGroups(childNodes []models.Node) {
	d.childGrouping = d.node.GetChildGrouping()
	d.childGrouping.AppendUnmatched(childNodes)
	d.childGrouping.UngroupedLabel = "Children"

	if d.node.GetType() == "project" {
		d.childGrouping.Regroup("Implementations", isImplementationNode)
	}
}

func (d *NodeDetail) GetSelectedChild() models.Node {
	return d.childGrouping.NodeAt(d.cursor)
}

func (d *NodeDetail) SelectNextChild() {
	if d.cursor < d.childGrouping.Size()-1 {
		d.cursor++
	}
}

func (d *NodeDetail) SelectPrevChild() {
	if d.cursor > 0 {
		d.cursor--
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
		contentBuilder.WriteString("[No linked commits found]")
	} else {
		contentBuilder.WriteString(d.table.View())
	}

	contentBuilder.WriteString("\n\n")

	// Display regular children section
	if d.childGrouping.IsEmpty() {
		contentBuilder.WriteString("[No children]")
	} else {
		renderer := &cliChildrenRenderer{
			sb:     &contentBuilder,
			width:  d.width,
			cursor: d.cursor,
		}
		d.childGrouping.Render(renderer)
	}

	// Use lipgloss to constrain the entire output to the component width
	style := lipgloss.NewStyle().Width(d.width)
	return style.Render(contentBuilder.String())
}

type cliChildrenRenderer struct {
	sb     *strings.Builder
	width  int
	index  int
	cursor int
}

func (r *cliChildrenRenderer) RenderGroupStart(nestingLevel int, label string) {
	fmt.Fprintf(r.sb, "%*s%s:\n", nestingLevel*2, "", label)
}

func (r *cliChildrenRenderer) RenderGroupEnd(nestingLevel int) {
	fmt.Fprintf(r.sb, "\n")
}

func (r *cliChildrenRenderer) RenderNode(nestingLevel int, node models.Node) {
	nodeTitle := node.GetTitle()
	indentStr := strings.Repeat(" ", nestingLevel*2)
	// -2 for prepended `> `, -1 for ellipsis, -1 for buffer
	maxTitleWidth := r.width - len(indentStr) - 4
	if len(nodeTitle) > maxTitleWidth && maxTitleWidth > 0 {
		nodeTitle = nodeTitle[:maxTitleWidth] + "…"
	}
	if r.index == r.cursor {
		// newline must come after formatting, or else the next line will somehow be off by the length
		// of the entire string
		r.sb.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("%s> %s", indentStr, nodeTitle)))
		r.sb.WriteString("\n")
	} else {
		fmt.Fprintf(r.sb, "%s  %s\n", indentStr, nodeTitle)
	}
	r.index++
}
