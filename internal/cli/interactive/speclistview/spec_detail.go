package speclistview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/common"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// SpecDetail encapsulates all state and logic for a spec detail
// (separated from the viewport logic)
type SpecDetail struct {
	node       models.Node
	links      []*models.SpecCommitLink
	childNodes []models.Node
	cursor     int
	table      table.Model
	width      int
	height     int
}

func NewSpecDetail() *SpecDetail {
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
	return &SpecDetail{
		table:  commitsTable,
		cursor: -1,
	}
}

func (d *SpecDetail) SetSize(width, height int) {
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

func (d *SpecDetail) SetSpec(node models.Node, links []*models.SpecCommitLink, childNodes []models.Node) {
	d.node = node
	d.links = links
	d.childNodes = childNodes
	d.updateCommitsTable()
	d.cursor = -1
}

func (d *SpecDetail) GetSelectedChild() models.Node {
	if d.cursor >= 0 && d.cursor < len(d.childNodes) {
		return d.childNodes[d.cursor]
	}
	return nil
}

func (d *SpecDetail) SelectNextChild() {
	if len(d.childNodes) == 0 {
		d.cursor = -1
		return
	}
	d.cursor++
	if d.cursor >= len(d.childNodes) {
		d.cursor = len(d.childNodes) - 1
	}
}

func (d *SpecDetail) SelectPrevChild() {
	if len(d.childNodes) == 0 {
		d.cursor = -1
		return
	}
	d.cursor--
	if d.cursor < 0 {
		d.cursor = 0
	}
}

func (d *SpecDetail) ResetCursor() {
	d.cursor = -1
}

func (d *SpecDetail) updateCommitsTable() {
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

func (d *SpecDetail) View() string {
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
	contentBuilder.WriteString("\n\nChild Nodes:\n")
	if len(d.childNodes) == 0 {
		contentBuilder.WriteString("  -\n")
	} else {
		for i, childNode := range d.childNodes {
			nodeTitle := childNode.GetTitle()
			if len(nodeTitle) > d.width-2 && d.width > 5 {
				nodeTitle = nodeTitle[:d.width-5] + "..."
			}
			if i == d.cursor {
				contentBuilder.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("> %s", nodeTitle)))
				contentBuilder.WriteString("\n")
			} else {
				contentBuilder.WriteString(fmt.Sprintf("  %s\n", nodeTitle))
			}
		}
	}

	// Use lipgloss to constrain the entire output to the component width
	style := lipgloss.NewStyle().Width(d.width)
	return style.Render(contentBuilder.String())
}
