package speclistview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// CreateNewSpecMsg signals that the user wants to create a new specification
type CreateNewSpecMsg struct{}
type LinkCommitSpecMsg struct {
	SpecID string
}

// Model represents the state of the spec list view screen
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Create key.Binding
	Link   key.Binding
	Help   key.Binding
	Return key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "next"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "prev"),
	),
	Create: key.NewBinding(
		key.WithKeys("c", "C"),
		key.WithHelp("c", "create"),
	),
	Link: key.NewBinding(
		key.WithKeys("l", "L"),
		key.WithHelp("l", "link commit"),
	),
	Return: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("h", "?"),
		key.WithHelp("h", "help"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Create, k.Link, k.Help, k.Return}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Create, k.Link},
		{k.Help, k.Return},
	}
}

type Model struct {
	keys        keyMap
	help        help.Model
	specs       []interactive.Spec
	cursor      int
	links       []*models.SpecCommitLink
	linkService LinkService

	width  int
	height int
}

// New creates a new model for the spec list view screen
func New(linkService LinkService) Model {
	return Model{
		keys:        keys,
		help:        help.New(),
		linkService: linkService,
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width
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

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	oldCursor := m.cursor
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.cursor < len(m.specs)-1 {
				m.cursor++
			}
		case key.Matches(msg, m.keys.Create):
			return *m, func() tea.Msg { return CreateNewSpecMsg{} }
		case key.Matches(msg, m.keys.Link):
			if m.cursor >= 0 && m.cursor < len(m.specs) {
				specID := m.specs[m.cursor].ID
				return *m, func() tea.Msg { return LinkCommitSpecMsg{SpecID: specID} }
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return *m, nil
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

	paneWidth := (m.width - 1) / 2 // width of each half pane, minus 1 for padding

	// Layout: left (list), right (details)
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

	left.WriteString(m.help.View(m.keys))
	finalLeft := lipgloss.NewStyle().Width(paneWidth).Render(left.String())

	// Right: details for selected spec
	var right strings.Builder
	if m.cursor >= 0 && m.cursor < len(m.specs) {
		spec := m.specs[m.cursor]
		right.WriteString(fmt.Sprintf("%s\n%s\n\n", spec.Title, strings.Repeat("=", paneWidth)))
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
	finalRight := lipgloss.NewStyle().Width(paneWidth + 1).PaddingLeft(1).Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Left, finalLeft, finalRight)
}
