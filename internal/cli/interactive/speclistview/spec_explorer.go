package speclistview

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/common"
)

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
	Create key.Binding
	Edit   key.Binding
	Delete key.Binding
	Link   key.Binding
	Remove key.Binding
	Move   key.Binding
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
	Move: key.NewBinding(
		key.WithKeys("m", "M"),
		key.WithHelp("m", "move"),
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
		{k.Link, k.Remove, k.Move},
		{k.Help, k.Quit},
	}
}

// SpecExplorer represents the main two-pane spec exploration interface
type SpecExplorer struct {
	leftPane  SpecDetailView
	rightPane SpecDetailView

	currentSpec interactive.Spec
	activeSpec  interactive.Spec

	linkService LinkService

	width  int
	height int

	keys     keyMap
	help     help.Model
	showHelp bool
}

func NewSpecExplorer(linkService LinkService) SpecExplorer {
	explorer := SpecExplorer{
		leftPane:    NewSpecDetailView(),
		rightPane:   NewSpecDetailView(),
		linkService: linkService,
		keys:        keys,
		help:        help.New(),
		showHelp:    false,
	}
	if linkService != nil {
		rootSpec, err := linkService.GetRootSpec()
		if err == nil && rootSpec != nil {
			explorer.currentSpec = *rootSpec
			explorer.activeSpec = *rootSpec
			explorer.setCurrentNode(&explorer.currentSpec)
		}
	}
	return explorer
}

func (e *SpecExplorer) SetSize(width, height int) {
	e.width = width
	e.height = height
	paneWidth := e.paneWidth()
	e.leftPane.SetSize(paneWidth, height)
	e.rightPane.SetSize(paneWidth, height)
	e.help.Width = paneWidth
}

func (e *SpecExplorer) paneWidth() int {
	return (e.width - 1) / 2
}

func (e *SpecExplorer) Refresh() tea.Cmd {
	return e.setCurrentNode(&e.currentSpec)
}

func (e *SpecExplorer) Update(msg tea.Msg) (SpecExplorer, tea.Cmd) {
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
				e.activeSpec = *selectedChild
			} else {
				e.activeSpec = e.currentSpec
			}
			// Update right pane with new active spec details (don't affect left pane cursor)
			e.updateRightPaneOnly()
			return *e, nil
		case key.Matches(msg, e.keys.Select):
			// Navigate to the active spec if it's different from current
			if e.activeSpec.ID != e.currentSpec.ID {
				return *e, e.navigateToChildren(&e.activeSpec)
			}
			return *e, nil
		case key.Matches(msg, e.keys.Create):
			return *e, func() tea.Msg { return CreateNewSpecMsg{ParentSpecID: e.activeSpec.ID} }
		case key.Matches(msg, e.keys.Edit):
			return *e, func() tea.Msg { return EditSpecMsg{SpecID: e.activeSpec.ID} }
		case key.Matches(msg, e.keys.Delete):
			return *e, func() tea.Msg { return DeleteSpecMsg{SpecID: e.activeSpec.ID} }
		case key.Matches(msg, e.keys.Link):
			return *e, func() tea.Msg { return LinkCommitSpecMsg{SpecID: e.activeSpec.ID} }
		case key.Matches(msg, e.keys.Remove):
			return *e, func() tea.Msg { return RemoveLinkSpecMsg{SpecID: e.activeSpec.ID} }
		case key.Matches(msg, e.keys.Move):
			return *e, func() tea.Msg { return MoveSpecMsg{SpecID: e.activeSpec.ID} }
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
	e.leftPane, leftCmd = e.leftPane.Update(msg)
	e.rightPane, rightCmd = e.rightPane.Update(msg)
	if leftCmd != nil && rightCmd != nil {
		return *e, tea.Batch(leftCmd, rightCmd)
	} else if leftCmd != nil {
		return *e, leftCmd
	} else if rightCmd != nil {
		return *e, rightCmd
	}
	return *e, nil
}

func (e *SpecExplorer) setCurrentNode(currentSpec *interactive.Spec) tea.Cmd {
	if currentSpec == nil {
		// This should only happen during initialization error - try to get root spec
		rootSpec, err := e.linkService.GetRootSpec()
		if err != nil {
			return nil
		}
		currentSpec = rootSpec
	}

	// Update model state
	e.currentSpec = *currentSpec
	e.activeSpec = *currentSpec // Set active spec to current node by default

	// Update details for both panes
	e.updateDetailsForSpec(e.activeSpec)

	return nil
}

func (e *SpecExplorer) navigateToChildren(spec *interactive.Spec) tea.Cmd {
	return e.setCurrentNode(spec)
}

func (e *SpecExplorer) navigateBack() tea.Cmd {
	// Get parent spec
	parentSpec, err := e.linkService.GetParentSpec(e.currentSpec.ID)
	if err != nil || parentSpec == nil {
		// No parent found - check if we're already at root
		rootSpec, err := e.linkService.GetRootSpec()
		if err != nil || rootSpec == nil || rootSpec.ID == e.currentSpec.ID {
			// Already at root or can't get root, stay where we are
			return nil
		}
		// Go to root spec
		return e.setCurrentNode(rootSpec)
	}

	return e.setCurrentNode(parentSpec)
}

func (e *SpecExplorer) updateDetailsForSpec(spec interactive.Spec) {
	if e.linkService == nil {
		return
	}

	// Left pane always shows current node details
	currentLinks, err := e.linkService.GetCommitsForSpec(e.currentSpec.ID)
	if err != nil {
		currentLinks = nil
	}

	currentChildSpecs, err := e.linkService.GetChildSpecs(e.currentSpec.ID)
	if err != nil {
		currentChildSpecs = nil
	}

	// Right pane shows active spec details (selected child or current node)
	activeLinks, err := e.linkService.GetCommitsForSpec(e.activeSpec.ID)
	if err != nil {
		activeLinks = nil
	}

	activeChildSpecs, err := e.linkService.GetChildSpecs(e.activeSpec.ID)
	if err != nil {
		activeChildSpecs = nil
	}

	// Update left pane with current node data (preserve cursor)
	e.leftPane.SetSpec(e.currentSpec, currentLinks, currentChildSpecs)

	// Update right pane with active spec data
	e.rightPane.SetSpec(e.activeSpec, activeLinks, activeChildSpecs)
}

// updateRightPaneOnly updates only the right pane without affecting left pane cursor
func (e *SpecExplorer) updateRightPaneOnly() {
	if e.linkService == nil {
		return
	}

	// Right pane shows active spec details (selected child or current node)
	activeLinks, err := e.linkService.GetCommitsForSpec(e.activeSpec.ID)
	if err != nil {
		activeLinks = nil
	}

	activeChildSpecs, err := e.linkService.GetChildSpecs(e.activeSpec.ID)
	if err != nil {
		activeChildSpecs = nil
	}

	// Update right pane with active spec data
	e.rightPane.SetSpec(e.activeSpec, activeLinks, activeChildSpecs)
}

func (e *SpecExplorer) View() string {
	left := e.leftPane.View()
	if e.showHelp {
		left = lipgloss.JoinVertical(lipgloss.Top, left, e.help.View(e.keys))
	}

	// Determine right pane content based on whether a child is selected
	var right string
	if e.activeSpec.ID == e.currentSpec.ID {
		// No child selected - show instruction message
		right = "Select a child specification to view its details"
	} else {
		// Child selected - show child details
		right = e.rightPane.View()
	}

	// Apply styling based on which spec is active
	var leftStyle, rightStyle lipgloss.Style
	if e.activeSpec.ID == e.currentSpec.ID {
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
