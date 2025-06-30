package speclistview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/common"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// CreateNewSpecMsg signals that the user wants to create a new specification
type CreateNewSpecMsg struct{}
type LinkCommitSpecMsg struct {
	SpecID string
}
type EditSpecMsg struct {
	SpecID string
}
type DeleteSpecMsg struct {
	SpecID string
}
type RemoveLinkSpecMsg struct {
	SpecID string
}
type ExitMsg struct{}

// Model represents the state of the spec list view screen
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
	GetChildSpecs(specID string) ([]*models.SpecNode, error)
	GetSpecByID(specID string) (*interactive.Spec, error)
	GetTopLevelSpecs() ([]interactive.Spec, error)
	GetParentSpec(specID string) (*interactive.Spec, error)
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Create key.Binding
	Edit   key.Binding
	Delete key.Binding
	Link   key.Binding
	Remove key.Binding
	Help   key.Binding
	Back   key.Binding
	Quit   key.Binding
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
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "back"),
	),
	Create: key.NewBinding(
		key.WithKeys("c", "C"),
		key.WithHelp("c", "create"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e", "E"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", "D"),
		key.WithHelp("d", "delete"),
	),
	Link: key.NewBinding(
		key.WithKeys("l", "L"),
		key.WithHelp("l", "link commit"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r", "R"),
		key.WithHelp("r", "remove link"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "Q"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("h", "?"),
		key.WithHelp("h", "help"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Select, k.Back},
		{k.Create, k.Edit, k.Delete},
		{k.Link, k.Remove},
		{k.Help, k.Quit},
	}
}

type Model struct {
	specSelector common.SpecSelector
	keys         keyMap
	help         help.Model
	specs        []interactive.Spec
	links        []*models.SpecCommitLink
	childSpecs   []*models.SpecNode
	linkService  LinkService

	// Navigation state
	currentSpec *interactive.Spec // nil for top level

	width  int
	height int
}

// New creates a new model for the spec list view screen
func New(linkService LinkService) Model {
	config := common.SpecSelectorConfig{
		Title: "Specifications",
	}
	specSelector := common.NewSpecSelector(config)

	return Model{
		specSelector: specSelector,
		keys:         keys,
		help:         help.New(),
		linkService:  linkService,
		currentSpec:  nil, // Start at top level
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width
	m.specSelector.SetSize(m.paneWidth(), m.height-3)
}

func (m *Model) paneWidth() int {
	return (m.width - 1) / 2 // width of each half pane, minus 1 for padding
}

// SetSpecs sets the specifications to be displayed
func (m *Model) SetSpecs(specs []interactive.Spec) {
	m.specs = specs
	m.specSelector.SetSpecs(specs)

	// Fetch links and child specs for the first spec if available
	if m.linkService != nil && len(specs) > 0 {
		links, err := m.linkService.GetCommitsForSpec(specs[0].ID)
		if err == nil {
			m.links = links
		} else {
			m.links = nil
		}

		childSpecs, err := m.linkService.GetChildSpecs(specs[0].ID)
		if err == nil {
			m.childSpecs = childSpecs
		} else {
			m.childSpecs = nil
		}
	}
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down):
			var cmd tea.Cmd
			selector, selectorCmd := m.specSelector.Update(msg)
			m.specSelector = *selector
			cmd = selectorCmd

			spec := m.specSelector.GetSelectedSpec()
			if spec != nil {
				links, err := m.linkService.GetCommitsForSpec(spec.ID)
				if err == nil {
					m.links = links
				} else {
					m.links = nil
				}

				childSpecs, err := m.linkService.GetChildSpecs(spec.ID)
				if err == nil {
					m.childSpecs = childSpecs
				} else {
					m.childSpecs = nil
				}
			}
			return *m, cmd
		case key.Matches(msg, m.keys.Select):
			m.navigateToChildren()
			return *m, nil
		case key.Matches(msg, m.keys.Create):
			return *m, func() tea.Msg { return CreateNewSpecMsg{} }
		case key.Matches(msg, m.keys.Edit):
			spec := m.specSelector.GetSelectedSpec()
			if spec == nil {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return EditSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Delete):
			spec := m.specSelector.GetSelectedSpec()
			if spec == nil {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return DeleteSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Link):
			spec := m.specSelector.GetSelectedSpec()
			if spec == nil {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return LinkCommitSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Remove):
			spec := m.specSelector.GetSelectedSpec()
			if spec == nil {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return RemoveLinkSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			// Adjust spec selector size based on help visibility
			if m.help.ShowAll {
				m.specSelector.SetSize(m.paneWidth(), m.height-4)
			} else {
				m.specSelector.SetSize(m.paneWidth(), m.height-3)
			}
			return *m, nil
		case key.Matches(msg, m.keys.Back):
			m.navigateBack()
			return *m, nil
		case key.Matches(msg, m.keys.Quit):
			return *m, func() tea.Msg { return ExitMsg{} }
		}
	case common.SpecSelectedMsg:
		// Handle spec selection from the selector component
		// Navigate to the selected spec's children (make it the current node)
		if msg.Spec.ID != "" {
			return *m, m.setCurrentNode(&msg.Spec)
		}
		return *m, nil
	}
	return *m, nil
}

// setCurrentNode sets the current spec and updates the display with its children
func (m *Model) setCurrentNode(currentSpec *interactive.Spec) tea.Cmd {
	var specs []interactive.Spec
	var title string

	if currentSpec == nil {
		// At top level
		topLevelSpecs, err := m.linkService.GetTopLevelSpecs()
		if err != nil {
			return nil
		}
		specs = topLevelSpecs
		title = "Specifications"
	} else {
		// Get children of the current spec
		childSpecNodes, err := m.linkService.GetChildSpecs(currentSpec.ID)
		if err != nil {
			return nil
		}

		// Convert child spec nodes to interactive.Spec objects
		childSpecs := make([]interactive.Spec, 0, len(childSpecNodes))
		for _, node := range childSpecNodes {
			childSpecs = append(childSpecs, interactive.Spec{
				ID:        node.ID,
				Title:     node.Title,
				Content:   node.Content,
				CreatedAt: node.CreatedAt.Format("2006-01-02 15:04"),
			})
		}
		specs = childSpecs
		title = currentSpec.Title
	}

	// Update model state
	m.currentSpec = currentSpec
	m.specs = specs

	// Update spec selector with new specs and title
	config := common.SpecSelectorConfig{
		Title: title,
	}
	m.specSelector = common.NewSpecSelector(config)
	m.specSelector.SetSpecs(specs)
	m.specSelector.SetSize(m.paneWidth(), m.height-3)

	// Update details for first spec if available
	if len(specs) > 0 {
		m.updateDetailsForSpec(specs[0])
	}

	return nil
}

// navigateToChildren navigates to the children of the selected spec
func (m *Model) navigateToChildren() tea.Cmd {
	spec := m.specSelector.GetSelectedSpec()
	if spec == nil {
		return nil
	}

	// Check if this spec has children before navigating
	childSpecLinks, err := m.linkService.GetChildSpecs(spec.ID)
	if err != nil || len(childSpecLinks) == 0 {
		return nil // No children or error
	}

	return m.setCurrentNode(spec)
}

// navigateBack navigates back to the parent level
func (m *Model) navigateBack() tea.Cmd {
	if m.currentSpec == nil {
		return nil // Already at top level
	}

	// Get parent spec
	parentSpec, err := m.linkService.GetParentSpec(m.currentSpec.ID)
	if err != nil || parentSpec == nil {
		// No parent found, go back to top level
		return m.setCurrentNode(nil)
	}

	return m.setCurrentNode(parentSpec)
}

// updateDetailsForSpec updates the links and child specs for the given spec
func (m *Model) updateDetailsForSpec(spec interactive.Spec) {
	if m.linkService == nil {
		return
	}

	links, err := m.linkService.GetCommitsForSpec(spec.ID)
	if err == nil {
		m.links = links
	} else {
		m.links = nil
	}

	childSpecs, err := m.linkService.GetChildSpecs(spec.ID)
	if err == nil {
		m.childSpecs = childSpecs
	} else {
		m.childSpecs = nil
	}
}

// View renders the spec list view screen
func (m *Model) View() string {
	if len(m.specs) == 0 {
		return "No specifications found.\n\nPress Esc to return to main menu"
	}

	paneWidth := m.paneWidth()

	// Layout: left (list), right (details)
	finalLeft := lipgloss.JoinVertical(lipgloss.Top, m.specSelector.View(), m.help.View(m.keys))

	// Right: details for selected spec
	var right strings.Builder
	spec := m.specSelector.GetSelectedSpec()
	if spec != nil {
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

		right.WriteString("\n\nChild Specifications:\n")
		if len(m.childSpecs) == 0 {
			right.WriteString("  -\n")
		} else {
			for _, cs := range m.childSpecs {
				// cs is now directly a SpecNode
				specTitle := cs.Title

				if len(specTitle) > paneWidth-2 {
					specTitle = specTitle[:paneWidth-5] + "..."
				}
				right.WriteString(fmt.Sprintf("  %s\n", specTitle))
			}
		}
	} else {
		right.WriteString("No specification selected. Create or select one to view details.\n")
	}
	finalRight := lipgloss.NewStyle().Width(paneWidth + 1).PaddingLeft(1).Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Left, finalLeft, finalRight)
}
