package speclistview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// CreateNewSpecMsg signals that the user wants to create a new specification
type CreateNewSpecMsg struct{}

// Model represents the state of the spec list view screen
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
}

type Model struct {
	specs       []interactive.Spec
	cursor      int
	links       []*models.SpecCommitLink
	linkService LinkService
}

// New creates a new model for the spec list view screen
func New() Model {
	return Model{}
}

// SetSpecs sets the specifications to be displayed
func (m *Model) SetSpecs(specs []interactive.Spec) {
	m.specs = specs
	m.cursor = 0
	// Fetch links for the first spec if available
	if m.linkService != nil && len(specs) > 0 {
		links, err := m.linkService.GetCommitsForSpec(specs[0].ID)
		if err == nil {
			m.links = links
		} else {
			m.links = nil
		}
	}
}

// SetLinkService injects the link service for DB access
func (m *Model) SetLinkService(svc LinkService) {
	m.linkService = svc
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	oldCursor := m.cursor
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.specs)-1 {
				m.cursor++
			}
		case "c":
			return *m, func() tea.Msg { return CreateNewSpecMsg{} }
		}
	}
	// If cursor changed, fetch links for the new selected spec
	if m.linkService != nil && m.cursor != oldCursor && m.cursor >= 0 && m.cursor < len(m.specs) {
		specID := m.specs[m.cursor].ID
		links, err := m.linkService.GetCommitsForSpec(specID)
		if err == nil {
			m.links = links
		} else {
			m.links = nil
		}
	}
	return *m, nil
}

// View renders the spec list view screen
func (m *Model) View() string {
	if len(m.specs) == 0 {
		return "No specifications found.\n\nPress Esc to return to main menu"
	}

	// Layout: left (list), right (details)
	const leftWidth = 40
	var left strings.Builder
	left.WriteString("📋 Specifications List\n")
	left.WriteString("=====================\n\n")

	for i, spec := range m.specs {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		title := spec.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}
		left.WriteString(fmt.Sprintf("%s %s\n", cursor, title))
	}

	left.WriteString("\nUse ↑/↓ arrows to navigate, 'c' to create new specification, Esc to go back")

	// Right: details for selected spec
	var right strings.Builder
	if m.cursor >= 0 && m.cursor < len(m.specs) {
		spec := m.specs[m.cursor]
		right.WriteString("📝 Spec Details\n")
		right.WriteString("=====================\n\n")
		right.WriteString(fmt.Sprintf("ID: %s\n", spec.ID))
		right.WriteString(fmt.Sprintf("Title: %s\n", spec.Title))
		right.WriteString(fmt.Sprintf("Created: %s\n", spec.CreatedAt))
		right.WriteString("\nContent:\n")
		right.WriteString(spec.Content)
		right.WriteString("\n\nLinked Commits:\n")
		if len(m.links) == 0 {
			right.WriteString("  (none)\n")
		} else {
			right.WriteString("  COMMIT           REPO             TYPE         CREATED\n")
			right.WriteString("  ──────           ────             ────         ───────\n")
			for _, l := range m.links {
				commitID := l.CommitID
				if len(commitID) > 12 {
					commitID = commitID[:12] + "..."
				}
				repo := l.RepoPath
				linkType := l.LinkType
				created := l.CreatedAt.Format("2006-01-02 15:04")
				right.WriteString(fmt.Sprintf("  %-16s %-16s %-12s %s\n", commitID, repo, linkType, created))
			}
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, left.String(), right.String())
}
