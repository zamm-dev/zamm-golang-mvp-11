package nodes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/common"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
)

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Select       key.Binding
	Create       key.Binding
	Edit         key.Binding
	OpenMarkdown key.Binding
	Delete       key.Binding
	Link         key.Binding
	Remove       key.Binding
	Move         key.Binding
	Organize     key.Binding
	Help         key.Binding
	Back         key.Binding
	Quit         key.Binding
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
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	OpenMarkdown: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "edit in VSCode"),
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
	Move: key.NewBinding(
		key.WithKeys("m", "M"),
		key.WithHelp("m", "move"),
	),
	Organize: key.NewBinding(
		key.WithKeys("o", "O"),
		key.WithHelp("o", "organize"),
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
		{k.Create, k.Edit, k.OpenMarkdown, k.Delete},
		{k.Link, k.Remove, k.Move},
		{k.Organize, k.Help, k.Quit},
	}
}

// NodeExplorer represents the main two-pane spec exploration interface
type NodeExplorer struct {
	leftPane  NodeDetailView
	rightPane NodeDetailView

	currentSpec models.Node
	activeSpec  models.Node

	linkService LinkService

	width  int
	height int

	keys     keyMap
	help     help.Model
	showHelp bool
}

func NewSpecExplorer(linkService LinkService, specService services.SpecService) NodeExplorer {
	if linkService == nil {
		panic("linkService cannot be nil in NewSpecExplorer")
	}

	explorer := NodeExplorer{
		leftPane:    NewNodeDetailView(linkService, specService),
		rightPane:   NewNodeDetailView(linkService, specService),
		linkService: linkService,
		keys:        keys,
		help:        help.New(),
		showHelp:    false,
	}

	rootSpec, err := linkService.GetRootNode()
	if err != nil {
		panic(fmt.Sprintf("failed to get root node in NewSpecExplorer: %v", err))
	}
	if rootSpec == nil {
		panic("root node is nil in NewSpecExplorer - this should never happen")
	}

	explorer.currentSpec = rootSpec
	explorer.activeSpec = rootSpec
	explorer.setCurrentNode(rootSpec)

	return explorer
}

func (e *NodeExplorer) SetSize(width, height int) {
	e.width = width
	e.height = height
	paneWidth := e.paneWidth()
	e.leftPane.SetSize(paneWidth, height)
	e.rightPane.SetSize(paneWidth, height)
	e.help.Width = paneWidth
}

func (e *NodeExplorer) paneWidth() int {
	return (e.width - 1) / 2
}

func (e *NodeExplorer) Refresh() tea.Cmd {
	return e.setCurrentNode(e.currentSpec)
}

func (e *NodeExplorer) Update(msg tea.Msg) (NodeExplorer, tea.Cmd) {
	// Assert that specs are never nil
	if e.currentSpec == nil {
		panic("currentSpec is nil in SpecExplorer.Update")
	}
	if e.activeSpec == nil {
		panic("activeSpec is nil in SpecExplorer.Update")
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, e.keys.Help):
			e.showHelp = !e.showHelp
			return *e, nil
		case key.Matches(msg, e.keys.Up) || key.Matches(msg, e.keys.Down):
			// Handle child selection in left pane only
			if key.Matches(msg, e.keys.Up) {
				e.leftPane.SelectPrevChild()
			} else {
				e.leftPane.SelectNextChild()
			}
			// Update active spec based on selection
			selectedChild := e.leftPane.GetSelectedChild()
			if selectedChild != nil {
				e.activeSpec = selectedChild
			} else {
				e.activeSpec = e.currentSpec
			}
			// Update right pane with new active spec details (don't affect left pane cursor)
			e.updateRightPaneOnly()
			return *e, nil
		case key.Matches(msg, e.keys.Select):
			// Navigate to the active spec if it's different from current
			if e.activeSpec.ID() != e.currentSpec.ID() {
				return *e, e.navigateToChildren(e.activeSpec)
			}
			return *e, nil
		case key.Matches(msg, e.keys.Create):
			return *e, func() tea.Msg { return CreateNewSpecMsg{ParentSpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Edit):
			return *e, func() tea.Msg { return EditSpecMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.OpenMarkdown):
			return *e, func() tea.Msg { return OpenMarkdownMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Delete):
			return *e, func() tea.Msg { return DeleteSpecMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Link):
			return *e, func() tea.Msg { return LinkCommitSpecMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Remove):
			return *e, func() tea.Msg { return RemoveLinkSpecMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Move):
			return *e, func() tea.Msg { return MoveSpecMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Organize):
			// Check if node has a slug, if not, go to slug editing screen first
			if e.activeSpec.GetSlug() == nil {
				// Generate auto-slug from title for editing
				autoSlug := e.generateAutoSlug(e.activeSpec.Title())
				return *e, func() tea.Msg {
					return EditSlugMsg{SpecID: e.activeSpec.ID(), OriginalTitle: e.activeSpec.Title(), InitialSlug: autoSlug}
				}
			}
			return *e, func() tea.Msg { return OrganizeSpecMsg{SpecID: e.activeSpec.ID()} }
		case key.Matches(msg, e.keys.Back):
			// If a child is selected, clear selection
			if e.leftPane.GetSelectedChild() != nil {
				e.leftPane.ResetCursor()
				e.activeSpec = e.currentSpec
				// Update right pane with current spec details (don't affect left pane cursor)
				e.updateRightPaneOnly()
				return *e, nil
			}
			// Otherwise navigate back to parent
			return *e, e.navigateBack()
		case key.Matches(msg, e.keys.Quit):
			return *e, func() tea.Msg { return ExitMsg{} }
		}
	}
	var leftCmd, rightCmd tea.Cmd
	leftModel, leftCmd := e.leftPane.Update(msg)
	rightModel, rightCmd := e.rightPane.Update(msg)
	e.leftPane = *leftModel.(*NodeDetailView)
	e.rightPane = *rightModel.(*NodeDetailView)
	if leftCmd != nil && rightCmd != nil {
		return *e, tea.Batch(leftCmd, rightCmd)
	} else if leftCmd != nil {
		return *e, leftCmd
	} else if rightCmd != nil {
		return *e, rightCmd
	}
	return *e, nil
}

func (e *NodeExplorer) setCurrentNode(currentNode models.Node) tea.Cmd {
	if currentNode == nil {
		// This should only happen during initialization error - try to get root spec
		rootSpec, err := e.linkService.GetRootNode()
		if err != nil {
			return nil
		}
		currentNode = rootSpec
	}

	// Update model state
	e.currentSpec = currentNode
	e.activeSpec = currentNode // Set active spec to current node by default

	// Update details for both panes
	e.updateDetailsForSpec()

	return nil
}

func (e *NodeExplorer) navigateToChildren(node models.Node) tea.Cmd {
	return e.setCurrentNode(node)
}

func (e *NodeExplorer) navigateBack() tea.Cmd {
	// Get parent spec
	parentSpec, err := e.linkService.GetParentNode(e.currentSpec.ID())
	if err != nil || parentSpec == nil {
		// No parent found - check if we're already at root
		rootSpec, err := e.linkService.GetRootNode()
		if err != nil || rootSpec == nil || rootSpec.ID() == e.currentSpec.ID() {
			// Already at root or can't get root, stay where we are
			return nil
		}
		// Go to root spec
		return e.setCurrentNode(rootSpec)
	}

	return e.setCurrentNode(parentSpec)
}

func (e *NodeExplorer) updateDetailsForSpec() {
	if e.linkService == nil {
		return
	}

	// Left pane always shows current node details
	e.leftPane.SetSpec(e.currentSpec)

	// Right pane shows active spec details (selected child or current node)
	e.rightPane.SetSpec(e.activeSpec)
}

// updateRightPaneOnly updates only the right pane without affecting left pane cursor
func (e *NodeExplorer) updateRightPaneOnly() {
	if e.linkService == nil {
		return
	}

	// Right pane shows active spec details (selected child or current node)
	e.rightPane.SetSpec(e.activeSpec)
}

func (e *NodeExplorer) View() string {
	// Assert that specs are never nil
	if e.currentSpec == nil {
		panic("currentSpec is nil in SpecExplorer.View")
	}
	if e.activeSpec == nil {
		panic("activeSpec is nil in SpecExplorer.View")
	}

	left := e.leftPane.View()
	if e.showHelp {
		left = lipgloss.JoinVertical(lipgloss.Top, left, e.help.View(e.keys))
	}

	// Determine right pane content based on whether a child is selected
	var right string
	if e.activeSpec.ID() == e.currentSpec.ID() {
		// No child selected - show instruction message
		right = "Select a child specification to view its details"
	} else {
		// Child selected - show child details
		right = e.rightPane.View()
	}

	// Apply styling based on which spec is active
	var leftStyle, rightStyle lipgloss.Style
	if e.activeSpec.ID() == e.currentSpec.ID() {
		// No child selected - left pane is active
		leftStyle = common.ActiveNodeStyle()
		rightStyle = lipgloss.NewStyle()
	} else {
		// Child selected - right pane is active
		leftStyle = lipgloss.NewStyle()
		rightStyle = common.ActiveNodeStyle()
	}

	left = leftStyle.Width(e.paneWidth()).Render(left)
	right = rightStyle.Width(e.paneWidth()).Render(right)

	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(lipgloss.Color("240")).
		Height(e.height).
		Render("")
	return lipgloss.JoinHorizontal(lipgloss.Left, left, border, right)
}

// generateAutoSlug creates a slug from the given title using the same logic as the spec service
func (e *NodeExplorer) generateAutoSlug(title string) string {
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "untitled"
	}
	return slug
}
