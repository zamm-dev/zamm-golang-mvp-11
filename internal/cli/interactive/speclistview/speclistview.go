package speclistview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/common"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// CreateNewSpecMsg signals that the user wants to create a new specification
type CreateNewSpecMsg struct {
	ParentSpecID string // ID of parent spec
}
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
	GetChildSpecs(specID *string) ([]*models.SpecNode, error) // nil specID returns top-level specs
	GetSpecByID(specID string) (*interactive.Spec, error)
	GetParentSpec(specID string) (*interactive.Spec, error)
	GetRootSpec() (*interactive.Spec, error)
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
	viewport     viewport.Model

	// Navigation state
	currentSpec interactive.Spec // always defined - root node by default
	activeSpec  interactive.Spec // the currently active (highlighted) spec

	width  int
	height int
}

// New creates a new model for the spec list view screen
func New(linkService LinkService) Model {
	config := common.SpecSelectorConfig{
		Title: "Specifications",
	}
	specSelector := common.NewSpecSelector(config)

	model := Model{
		specSelector: specSelector,
		keys:         keys,
		help:         help.New(),
		linkService:  linkService,
		viewport:     viewport.New(0, 0),
	}

	// Initialize with root spec as current node
	if linkService != nil {
		rootSpec, err := linkService.GetRootSpec()
		if err == nil && rootSpec != nil {
			// Set both currentSpec and activeSpec to the root spec
			model.currentSpec = *rootSpec
			model.activeSpec = *rootSpec
			model.setCurrentNode(&model.currentSpec)
		}
	}

	return model
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width
	m.specSelector.SetSize(m.paneWidth(), m.height-3)

	// Set viewport size for the right pane
	m.viewport.Width = m.paneWidth()
	m.viewport.Height = m.height
}

func (m *Model) paneWidth() int {
	return (m.width - 1) / 2 // width of each half pane, minus 1 for padding
}

// Refresh refreshes the current view data by reloading specs for the current node
func (m *Model) Refresh() tea.Cmd {
	return m.setCurrentNode(&m.currentSpec)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down):
			// Handle navigation between child specs
			if len(m.specs) > 0 {
				// Check if list is already in focus
				if !m.specSelector.IsFocused() {
					// First time navigating - reset cursor to 0 and set focus
					m.specSelector.ResetCursor()
					m.specSelector.SetFocus(true)

					// Update active spec to the first child spec (at position 0)
					selectedSpec := m.specSelector.GetSelectedSpec()
					if selectedSpec != nil {
						m.activeSpec = *selectedSpec
						m.updateDetailsForSpec(*selectedSpec)
					}
					return *m, nil
				} else {
					// List is already in focus - handle normal navigation
					var cmd tea.Cmd
					selector, selectorCmd := m.specSelector.Update(msg)
					m.specSelector = *selector
					cmd = selectorCmd

					// Update active spec to the selected child spec
					selectedSpec := m.specSelector.GetSelectedSpec()
					if selectedSpec != nil {
						m.activeSpec = *selectedSpec
						m.updateDetailsForSpec(*selectedSpec)
					}
					return *m, cmd
				}
			}
		case key.Matches(msg, m.keys.Select):
			// Navigate to children of the active spec (if any)
			if m.activeSpec.ID != m.currentSpec.ID {
				// Active spec is a child - navigate to it
				return *m, m.navigateToChildren(&m.activeSpec)
			}
			return *m, nil
		case key.Matches(msg, m.keys.Create):
			// Use the active spec ID as parent
			return *m, func() tea.Msg { return CreateNewSpecMsg{ParentSpecID: m.activeSpec.ID} }
		case key.Matches(msg, m.keys.Edit):
			// Edit the active spec
			return *m, func() tea.Msg { return EditSpecMsg{SpecID: m.activeSpec.ID} }
		case key.Matches(msg, m.keys.Delete):
			// Delete the active spec
			return *m, func() tea.Msg { return DeleteSpecMsg{SpecID: m.activeSpec.ID} }
		case key.Matches(msg, m.keys.Link):
			// Link commit to the active spec
			return *m, func() tea.Msg { return LinkCommitSpecMsg{SpecID: m.activeSpec.ID} }
		case key.Matches(msg, m.keys.Remove):
			// Remove link from the active spec
			return *m, func() tea.Msg { return RemoveLinkSpecMsg{SpecID: m.activeSpec.ID} }
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return *m, nil
		case key.Matches(msg, m.keys.Back):
			// If active spec is a child, set active spec back to current node
			if m.activeSpec.ID != m.currentSpec.ID {
				m.activeSpec = m.currentSpec
				m.updateDetailsForSpec(m.activeSpec)

				// Update focus state - list is no longer in focus since we're back to current node
				m.specSelector.SetFocus(false)
				return *m, nil
			}
			// If active spec is current node, navigate back to parent
			return *m, m.navigateBack()
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

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	return *m, cmd
}

// setCurrentNode sets the current spec and updates the display with its children
func (m *Model) setCurrentNode(currentSpec *interactive.Spec) tea.Cmd {
	var specs []interactive.Spec
	var title string

	if currentSpec == nil {
		// This should only happen during initialization error - try to get root spec
		rootSpec, err := m.linkService.GetRootSpec()
		if err != nil {
			return nil
		}
		currentSpec = rootSpec
	}

	// Get children of the current spec
	childSpecNodes, err := m.linkService.GetChildSpecs(&currentSpec.ID)
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

	// Update model state
	m.currentSpec = *currentSpec
	m.activeSpec = *currentSpec // Set active spec to current node by default
	m.specs = specs

	// Update spec selector with new specs and title
	config := common.SpecSelectorConfig{
		Title: title,
	}
	m.specSelector = common.NewSpecSelector(config)
	m.specSelector.SetSpecs(specs)
	m.specSelector.SetSize(m.paneWidth(), m.height-3)

	// Initially, list is not in focus - focus is set when user starts navigating
	m.specSelector.SetFocus(false)

	// Update details for the current spec (active spec)
	m.updateDetailsForSpec(m.activeSpec)

	return nil
}

// navigateToChildren navigates to the children of the given spec
func (m *Model) navigateToChildren(spec *interactive.Spec) tea.Cmd {
	return m.setCurrentNode(spec)
}

// navigateBack navigates back to the parent level
func (m *Model) navigateBack() tea.Cmd {
	// Get parent spec
	parentSpec, err := m.linkService.GetParentSpec(m.currentSpec.ID)
	if err != nil || parentSpec == nil {
		// No parent found - check if we're already at root
		rootSpec, err := m.linkService.GetRootSpec()
		if err != nil || rootSpec == nil || rootSpec.ID == m.currentSpec.ID {
			// Already at root or can't get root, stay where we are
			return nil
		}
		// Go to root spec
		return m.setCurrentNode(rootSpec)
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

	childSpecs, err := m.linkService.GetChildSpecs(&spec.ID)
	if err == nil {
		m.childSpecs = childSpecs
	} else {
		m.childSpecs = nil
	}
}

// generateRightPaneContent generates the content for the right pane viewport
func (m *Model) generateRightPaneContent() string {
	paneWidth := m.paneWidth()

	// Determine if current node is active (no child is selected)
	isCurrentNodeActive := m.activeSpec.ID == m.currentSpec.ID

	var contentBuilder strings.Builder
	if isCurrentNodeActive {
		contentBuilder.WriteString("Select a child specification to view its details\n\n")
	} else {
		contentBuilder.WriteString(fmt.Sprintf("%s\n%s\n\n", m.activeSpec.Title, strings.Repeat("=", paneWidth)))
		contentBuilder.WriteString(m.activeSpec.Content)
		contentBuilder.WriteString("\n\nLinked Commits:\n")
		if len(m.links) == 0 {
			contentBuilder.WriteString("  (none)\n")
		} else {
			contentBuilder.WriteString("  COMMIT           REPO             TYPE         CREATED\n")
			contentBuilder.WriteString("  ──────           ────             ────         ───────\n")
			for _, l := range m.links {
				commitID := l.CommitID
				if len(commitID) > 12 {
					commitID = commitID[:12] + "..."
				}
				repo := l.RepoPath
				linkType := l.LinkType
				created := l.CreatedAt.Format("2006-01-02 15:04")
				contentBuilder.WriteString(fmt.Sprintf("  %-16s %-16s %-12s %s\n", commitID, repo, linkType, created))
			}
		}

		contentBuilder.WriteString("\n\nChild Specifications:\n")
		if len(m.childSpecs) == 0 {
			contentBuilder.WriteString("  -\n")
		} else {
			for _, cs := range m.childSpecs {
				// cs is now directly a SpecNode
				specTitle := cs.Title

				if len(specTitle) > paneWidth-2 && paneWidth > 5 {
					specTitle = specTitle[:paneWidth-5] + "..."
				}
				contentBuilder.WriteString(fmt.Sprintf("  %s\n", specTitle))
			}
		}
	}

	var rightStyle lipgloss.Style
	if isCurrentNodeActive {
		rightStyle = lipgloss.NewStyle()
	} else {
		rightStyle = common.ActiveNodeStyle()
	}
	return rightStyle.Width(paneWidth).MarginLeft(1).Render(contentBuilder.String())
}

// View renders the spec list view screen
func (m *Model) View() string {
	paneWidth := m.paneWidth()

	// Determine if current node is active (no child is selected)
	isCurrentNodeActive := m.activeSpec.ID == m.currentSpec.ID

	// Layout: left (list), right (details)
	left := lipgloss.JoinVertical(lipgloss.Top, m.specSelector.View(), m.help.View(m.keys))

	var leftStyle lipgloss.Style
	if isCurrentNodeActive {
		leftStyle = common.ActiveNodeStyle()
	} else {
		leftStyle = lipgloss.NewStyle()
	}
	left = leftStyle.Width(paneWidth).Render(left)

	// Right: viewport with details for active spec
	m.viewport.SetContent(m.generateRightPaneContent())
	right := m.viewport.View()

	return lipgloss.JoinHorizontal(lipgloss.Left, left, right)
}
