package nodes

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/common"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
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
	specService   services.SpecService
}

func NewNodeDetail(linkService LinkService, specService services.SpecService) *NodeDetail {
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
		specService: specService,
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

	links, err := d.linkService.GetCommitsForSpec(node.ID())
	if err != nil {
		d.links = nil
	} else {
		d.links = links
	}

	d.childGrouping, err = GetOrganizedChildren(d.specService, node)
	if err != nil {
		d.childGrouping = models.ChildGroup{}
	}

	d.updateCommitsTable()
	d.cursor = -1
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
	contentBuilder.WriteString(fmt.Sprintf("%s\n%s\n\n%s\n\n", d.node.Title(), strings.Repeat("=", d.width), d.node.Content()))
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
	nodeTitle := node.Title()
	// account for prepended `> ` taking up the first level of indentation
	indentStr := strings.Repeat(" ", (nestingLevel-1)*2)
	// -1 for ellipsis, -1 for buffer
	maxTitleWidth := r.width - len(indentStr) - 2
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
