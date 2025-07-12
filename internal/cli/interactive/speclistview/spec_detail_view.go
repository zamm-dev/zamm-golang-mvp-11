package speclistview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/common"
	"github.com/yourorg/zamm-mvp/internal/models"
)

const typeWidth = 6
const commitWidth = 8
const repoWidth = 16

// SpecDetailView represents a view for displaying a single spec's details
type SpecDetailView struct {
	links      []*models.SpecCommitLink
	childSpecs []*models.Spec
	viewport   viewport.Model
	table      table.Model

	// The spec being displayed
	spec interactive.Spec

	// Child selection state
	cursor int // Index of selected child (-1 if no children or not focused)

	width  int
	height int
}

// NewSpecDetailView creates a new spec detail view
func NewSpecDetailView() SpecDetailView {
	columns := []table.Column{
		{Title: "TYPE", Width: typeWidth},
		{Title: "COMMIT", Width: commitWidth},
		{Title: "REPO", Width: repoWidth},
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
	return SpecDetailView{
		viewport: viewport.New(0, 0),
		table:    commitsTable,
		cursor:   -1, // No child selected initially
	}
}

// SetSize sets the size of the detail view
func (v *SpecDetailView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.viewport.Width = width
	v.viewport.Height = height
	extraPadding := 7
	columns := []table.Column{
		{Title: "TYPE", Width: typeWidth},
		{Title: "COMMIT", Width: commitWidth},
		{Title: "REPO", Width: width - typeWidth - commitWidth - extraPadding},
	}
	v.table.SetColumns(columns)
}

// SetSpec updates the spec being displayed and refreshes the view
func (v *SpecDetailView) SetSpec(spec interactive.Spec, links []*models.SpecCommitLink, childSpecs []*models.Spec) {
	v.spec = spec
	v.links = links
	v.childSpecs = childSpecs
	v.updateCommitsTable()
	v.viewport.SetYOffset(0)

	// Reset cursor when spec changes
	v.cursor = -1
}

// GetSelectedChild returns the currently selected child spec, or nil if none selected
func (v *SpecDetailView) GetSelectedChild() *interactive.Spec {
	if v.cursor >= 0 && v.cursor < len(v.childSpecs) {
		childSpec := v.childSpecs[v.cursor]
		return &interactive.Spec{
			ID:      childSpec.ID,
			Title:   childSpec.Title,
			Content: childSpec.Content,
		}
	}
	return nil
}

// SelectNextChild moves cursor to next child
func (v *SpecDetailView) SelectNextChild() {
	if len(v.childSpecs) == 0 {
		v.cursor = -1
		return
	}
	v.cursor++
	if v.cursor >= len(v.childSpecs) {
		v.cursor = len(v.childSpecs) - 1 // Stop at the end
	}
}

// SelectPrevChild moves cursor to previous child
func (v *SpecDetailView) SelectPrevChild() {
	if len(v.childSpecs) == 0 {
		v.cursor = -1
		return
	}
	v.cursor--
	if v.cursor < 0 {
		v.cursor = 0 // Stop at the beginning
	}
}

// ResetCursor resets the cursor to no selection
func (v *SpecDetailView) ResetCursor() {
	v.cursor = -1
}

// Update handles only viewport scrolling
func (v *SpecDetailView) Update(msg tea.Msg) (SpecDetailView, tea.Cmd) {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return *v, cmd
}

// updateCommitsTable updates the table with current commit links
func (v *SpecDetailView) updateCommitsTable() {
	if v.links == nil {
		v.table.SetRows([]table.Row{})
		return
	}

	rows := make([]table.Row, len(v.links))
	for i, link := range v.links {
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

	v.table.SetRows(rows)
	v.table.SetHeight(len(rows) + 2) // +2 for header and separator
}

// generateContent generates the content for the viewport
func (v *SpecDetailView) generateContent() string {
	var contentBuilder strings.Builder

	// Show spec title and content
	contentBuilder.WriteString(fmt.Sprintf("%s\n%s\n\n%s\n\n", v.spec.Title, strings.Repeat("=", v.width), v.spec.Content))

	if len(v.links) == 0 {
		contentBuilder.WriteString("[No linked commits found]\n")
	} else {
		// Use the table component for displaying commits
		contentBuilder.WriteString(v.table.View())
	}

	contentBuilder.WriteString("\n\nChild Specifications:\n")
	if len(v.childSpecs) == 0 {
		contentBuilder.WriteString("  -\n")
	} else {
		for i, cs := range v.childSpecs {
			specTitle := cs.Title

			if len(specTitle) > v.width-2 && v.width > 5 {
				specTitle = specTitle[:v.width-5] + "..."
			}

			// Highlight selected child with ActiveNodeStyle
			if i == v.cursor {
				contentBuilder.WriteString(common.ActiveNodeStyle().Render(fmt.Sprintf("> %s", specTitle)))
				contentBuilder.WriteString("\n")
			} else {
				contentBuilder.WriteString(fmt.Sprintf("  %s\n", specTitle))
			}
		}
	}

	return contentBuilder.String()
}

// View renders the spec detail view
func (v *SpecDetailView) View() string {
	// Set viewport content
	v.viewport.SetContent(v.generateContent())

	// The help model is removed, so this will always return the viewport view
	return v.viewport.View()
}
