package cli

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	interactive "github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/common"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive/speclistview"
	"github.com/yourorg/zamm-mvp/internal/models"
	"github.com/yourorg/zamm-mvp/internal/services"
)

// MenuState represents the current state of the interactive menu
type MenuState int

const (
	SpecListView MenuState = iota
	LinkSelection
	SpecEditor
	ConfirmDelete
	// New states for link editing components
	LinkEditor
	UnlinkEditor
)

// Model represents the state of our TUI application
type Model struct {
	app            *App
	state          MenuState
	cursor         int
	specs          []interactive.Spec
	links          []linkItem
	selectedSpecID string
	message        string
	showMessage    bool
	specListView   speclistview.Model

	terminalWidth  int
	terminalHeight int

	// Input fields for forms
	inputTitle   string
	inputContent string
	inputCommit  string
	inputRepo    string
	inputType    string
	textInput    textinput.Model
	promptText   string

	// State tracking
	editingSpecID string
	contentLines  []string
	confirmAction string
	parentSpecID  string // ID of parent spec when creating new spec

	// Link editing components
	linkEditor common.LinkEditor

	// Spec selector components
	specEditor common.SpecEditor
}

type linkItem struct {
	ID        string
	CommitID  string
	RepoPath  string
	LinkLabel string
}

// Custom messages
type specsLoadedMsg struct {
	specs []interactive.Spec
	err   error
}

type linksLoadedMsg struct {
	links []linkItem
	err   error
}

type operationCompleteMsg struct {
	message string
}

// createInteractiveCommand creates the interactive mode command
func (a *App) createInteractiveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Interactive mode for managing specs and links",
		Long:  "Start an interactive session to manage specifications and links using arrow keys for navigation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runInteractiveMode()
		},
	}
}

// runInteractiveMode starts the interactive mode with TUI
func (a *App) runInteractiveMode() error {
	// Perform complete initialization
	if err := a.InitializeZamm(); err != nil {
		return fmt.Errorf("failed to initialize zamm: %w", err)
	}

	ti := textinput.New()
	ti.Focus()

	combinedSvc := &combinedService{
		linkService: a.linkService,
		specService: a.specService,
	}

	model := Model{
		app:            a,
		state:          SpecListView,
		textInput:      ti,
		specListView:   speclistview.New(combinedSvc),
		linkEditor:     common.NewLinkEditor(common.LinkEditorConfig{Title: "", DefaultRepo: a.config.Git.DefaultRepo, SelectedSpecID: "", SelectedSpecTitle: "", IsUnlinkMode: false}, a.linkService, a.specService),
		terminalWidth:  80, // Default terminal width
		terminalHeight: 24, // Default terminal height
	}

	p := tea.NewProgram(&model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init is the first function that will be called
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadSpecsCmd())
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.specListView.SetSize(msg.Width, msg.Height)
		m.specEditor.SetSize(msg.Width, msg.Height)
		m.linkEditor.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if m.showMessage {
			if msg.String() == "enter" || msg.String() == " " || msg.String() == "esc" {
				m.showMessage = false
				m.message = ""
				m.state = SpecListView
				m.cursor = 0
				return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
			}
			return m, nil
		}

		switch m.state {
		case SpecListView:
			// Delegate to the spec list view
			var cmd tea.Cmd
			m.specListView, cmd = m.specListView.Update(msg)
			return m, cmd
		case LinkSelection:
			return m.updateLinkSelection(msg)
		case SpecEditor:
			var cmd tea.Cmd
			editor, editorCmd := m.specEditor.Update(msg)
			if specEditor, ok := editor.(*common.SpecEditor); ok {
				m.specEditor = *specEditor
			}
			cmd = editorCmd
			return m, cmd

		case ConfirmDelete:
			return m.updateConfirmDelete(msg)
		}

	case specsLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading specs: %v", msg.err)
			m.showMessage = true
			return m, nil
		}
		m.specs = msg.specs
		return m, nil

	case linksLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading links: %v", msg.err)
			m.showMessage = true
			return m, nil
		}
		m.links = msg.links

		if len(m.links) == 0 {
			m.message = "No links found for this specification."
			m.showMessage = true
			return m, nil
		}

		m.state = LinkSelection
		m.cursor = 0
		return m, nil

	case operationCompleteMsg:
		m.message = msg.message
		m.showMessage = true
		return m, nil

	case speclistview.CreateNewSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.parentSpecID = msg.ParentSpecID // Store parent spec ID for later use

			config := common.SpecEditorConfig{
				Title:          "📝 Create New Specification",
				InitialTitle:   "",
				InitialContent: "",
			}
			m.specEditor = common.NewSpecEditor(config)
			m.specEditor.SetSize(m.terminalWidth, m.terminalHeight)
			m.state = SpecEditor
			return m, nil
		}

	case speclistview.EditSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.editingSpecID = msg.SpecID

			// Find current title and content for pre-filling
			var currentTitle, currentContent string
			for _, spec := range m.specs {
				if spec.ID == msg.SpecID {
					currentTitle = spec.Title
					currentContent = spec.Content
					break
				}
			}

			config := common.SpecEditorConfig{
				Title:          "✏️  Edit Specification",
				InitialTitle:   currentTitle,
				InitialContent: currentContent,
			}
			m.specEditor = common.NewSpecEditor(config)
			m.specEditor.SetSize(m.terminalWidth, m.terminalHeight)
			m.state = SpecEditor
			return m, nil
		}

	case speclistview.LinkCommitSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.selectedSpecID = msg.SpecID

			// Find selected spec title for the title
			var specTitle string
			for _, spec := range m.specs {
				if spec.ID == msg.SpecID {
					specTitle = spec.Title
					break
				}
			}

			// Create link editor for linking mode
			config := common.LinkEditorConfig{
				Title:             "Link Specification",
				DefaultRepo:       m.app.config.Git.DefaultRepo,
				SelectedSpecID:    msg.SpecID,
				SelectedSpecTitle: specTitle,
				IsUnlinkMode:      false,
			}
			m.linkEditor = common.NewLinkEditor(config, m.app.linkService, m.app.specService)
			m.linkEditor.SetSize(m.terminalWidth, m.terminalHeight)

			m.state = LinkEditor
			return m, nil
		}

	case speclistview.DeleteSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.selectedSpecID = msg.SpecID
			m.state = ConfirmDelete
			m.confirmAction = "delete_spec"
			return m, nil
		}

	case speclistview.RemoveLinkSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.selectedSpecID = msg.SpecID

			// Find selected spec title for the title
			var specTitle string
			for _, spec := range m.specs {
				if spec.ID == msg.SpecID {
					specTitle = spec.Title
					break
				}
			}

			// Create link editor for unlinking mode
			config := common.LinkEditorConfig{
				Title:             "Remove Links",
				DefaultRepo:       m.app.config.Git.DefaultRepo,
				SelectedSpecID:    msg.SpecID,
				SelectedSpecTitle: specTitle,
				IsUnlinkMode:      true,
			}
			m.linkEditor = common.NewLinkEditor(config, m.app.linkService, m.app.specService)
			m.linkEditor.SetSize(m.terminalWidth, m.terminalHeight)

			m.state = LinkEditor
			return m, nil
		}

	case common.SpecEditorCompleteMsg:
		if m.state == SpecEditor {
			// Determine if this is create or edit based on whether we have an editingSpecID
			if m.editingSpecID != "" {
				// Edit existing spec
				return m, m.updateSpecCmd(m.editingSpecID, msg.Title, msg.Content)
			} else {
				// Create new spec
				return m, m.createSpecCmd(msg.Title, msg.Content)
			}
		}

	case common.SpecEditorCancelMsg:
		if m.state == SpecEditor {
			m.state = SpecListView
			m.resetInputs()
			return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
		}

	case common.LinkEditorCompleteMsg:
		if m.state == LinkEditor {
			m.state = SpecListView
			m.resetInputs()
			return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
		}

	case common.LinkEditorCancelMsg:
		if m.state == LinkEditor {
			m.state = SpecListView
			m.resetInputs()
			return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
		}

	case common.LinkEditorErrorMsg:
		if m.state == LinkEditor {
			return m, func() tea.Msg {
				return operationCompleteMsg{message: fmt.Sprintf("Error: %s. Press Enter to continue...", msg.Error)}
			}
		}

	case speclistview.ExitMsg:
		return m, tea.Quit
	}

	// Handle unhandled messages based on state
	switch m.state {
	case LinkEditor:
		var cmd tea.Cmd
		editor, cmd := m.linkEditor.Update(msg)
		if linkEditor, ok := editor.(common.LinkEditor); ok {
			m.linkEditor = linkEditor
		}
		return m, cmd
	default:
		return m, nil
	}
}

// updateMainMenu handles updates for the main menu
// updateLinkSelection handles updates for link selection
func (m *Model) updateLinkSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = SpecListView
		m.cursor = 0
		return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.links)-1 {
			m.cursor++
		}
	case "enter", " ":
		if len(m.links) > 0 {
			return m.executeLinkAction()
		}
	}
	return m, nil
}

// updateConfirmDelete handles updates for delete confirmation
func (m *Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "n":
		m.state = SpecListView
		m.cursor = 0
		m.resetInputs()
		return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
	case "y":
		if m.confirmAction == "delete_spec" {
			return m, m.deleteSpecCmd(m.selectedSpecID)
		} else if m.confirmAction == "delete_link" {
			if m.cursor < len(m.links) {
				selectedLink := m.links[m.cursor]
				return m, m.deleteLinkCmd(m.selectedSpecID, selectedLink.CommitID, selectedLink.RepoPath)
			}
		}
		if m.confirmAction == "delete_link" {
			m.state = SpecListView
		} else {
			m.state = SpecListView
		}
		m.cursor = 0
		m.resetInputs()
		return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
	}
	return m, nil
}

// executeAction executes the selected main menu action
// executeLinkAction executes the action on the selected link
func (m *Model) executeLinkAction() (tea.Model, tea.Cmd) {
	m.resetInputs()
	m.state = ConfirmDelete
	m.confirmAction = "delete_link"
	return m, nil
}

// loadSpecsCmd returns a command to load specs
func (m *Model) loadSpecsCmd() tea.Cmd {
	return func() tea.Msg {
		specs, err := m.app.specService.ListSpecs()
		if err != nil {
			return specsLoadedMsg{err: err}
		}

		specItems := make([]interactive.Spec, len(specs))
		for i, spec := range specs {
			specItems[i] = interactive.Spec{
				ID:      spec.ID,
				Title:   spec.Title,
				Content: spec.Content,
			}
		}

		return specsLoadedMsg{specs: specItems}
	}
}

// loadLinksForSpecCmd returns a command to load links for the selected spec
func (m *Model) loadLinksForSpecCmd() tea.Cmd {
	return func() tea.Msg {
		links, err := m.app.linkService.GetCommitsForSpec(m.selectedSpecID)
		if err != nil {
			return linksLoadedMsg{err: err}
		}

		linkItems := make([]linkItem, len(links))
		for i, link := range links {
			linkItems[i] = linkItem{
				CommitID:  link.CommitID,
				RepoPath:  link.RepoPath,
				LinkLabel: link.LinkLabel,
			}
		}

		return linksLoadedMsg{links: linkItems}
	}
}

// resetInputs clears all input fields
func (m *Model) resetInputs() {
	m.inputTitle = ""
	m.inputContent = ""
	m.inputCommit = ""
	m.inputRepo = ""
	m.inputType = ""
	m.promptText = ""
	m.editingSpecID = ""
	m.contentLines = []string{}
	m.confirmAction = ""
	m.parentSpecID = ""
	m.textInput.Reset()
	m.textInput.Blur()
}

// createSpecCmd returns a command to create a new spec
func (m *Model) createSpecCmd(title, content string) tea.Cmd {
	return func() tea.Msg {
		spec, err := m.app.specService.CreateSpec(title, content)
		if err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		// If there's a parent spec ID, create the parent-child relationship
		if m.parentSpecID != "" {
			_, err := m.app.specService.AddChildToParent(spec.ID, m.parentSpecID, "child")
			if err != nil {
				return operationCompleteMsg{message: fmt.Sprintf("Error creating parent-child relationship: %v. Press Enter to continue...", err)}
			}
		}

		return operationCompleteMsg{message: fmt.Sprintf("✅ Created specification: %s. Press Enter to continue...", spec.Title)}
	}
}

// updateSpecCmd returns a command to update an existing spec
func (m *Model) updateSpecCmd(specID, title, content string) tea.Cmd {
	return func() tea.Msg {
		spec, err := m.app.specService.UpdateSpec(specID, title, content)
		if err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return operationCompleteMsg{message: fmt.Sprintf("✅ Updated specification: %s. Press Enter to continue...", spec.Title)}
	}
}

// createLinkCmd returns a command to create a new link
func (m *Model) createLinkCmd(specID, commitID, repoPath, label string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.app.linkService.LinkSpecToCommit(specID, commitID, repoPath, label)
		if err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		// Find spec title for display
		var specTitle string
		for _, spec := range m.specs {
			if spec.ID == specID {
				specTitle = spec.Title
				break
			}
		}

		return operationCompleteMsg{message: fmt.Sprintf("✅ Created link between '%s' and commit %s. Press Enter to continue...",
			specTitle, commitID[:12]+"...")}
	}
}

// deleteSpecCmd returns a command to delete a spec
func (m *Model) deleteSpecCmd(specID string) tea.Cmd {
	return func() tea.Msg {
		// Find spec title for display
		var specTitle string
		for _, spec := range m.specs {
			if spec.ID == specID {
				specTitle = spec.Title
				break
			}
		}

		if err := m.app.specService.DeleteSpec(specID); err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return operationCompleteMsg{message: fmt.Sprintf("✅ Deleted specification: %s. Press Enter to continue...", specTitle)}
	}
}

// deleteLinkCmd returns a command to delete a link
func (m *Model) deleteLinkCmd(specID, commitID, repoPath string) tea.Cmd {
	return func() tea.Msg {
		if err := m.app.linkService.UnlinkSpecFromCommit(specID, commitID, repoPath); err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}
		return operationCompleteMsg{message: fmt.Sprintf("✅ Deleted link to commit %s. Press Enter to continue...", commitID[:12]+"...")}
	}
}

// View renders the UI
func (m *Model) View() string {
	if m.showMessage {
		return m.message
	}

	switch m.state {
	case SpecListView:
		return m.specListView.View()
	case LinkSelection:
		return m.renderLinkSelection()
	case SpecEditor:
		return m.specEditor.View()
	case LinkEditor:
		return m.linkEditor.View()
	case ConfirmDelete:
		return m.renderConfirmDelete()
	default:
		return "Loading..."
	}
}

// renderLinkSelection renders the link selection screen
func (m *Model) renderLinkSelection() string {
	s := "🗑️  Delete Specification Link\n"
	s += "=============================\n\n"

	if len(m.links) == 0 {
		s += "No links found for this specification.\n\n"
		s += "Press Esc to return to main menu"
		return s
	}

	// Find selected spec title
	selectedSpecTitle := ""
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			selectedSpecTitle = spec.Title
			break
		}
	}

	s += fmt.Sprintf("Links for '%s':\n\n", selectedSpecTitle)

	for i, link := range m.links {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		repoName := filepath.Base(link.RepoPath)
		s += fmt.Sprintf("%s %s (%s, %s)\n", cursor, link.CommitID[:12]+"...", repoName, link.LinkLabel)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to delete, Esc to go back"
	return s
}

// renderConfirmDelete renders the delete confirmation dialog
func (m *Model) renderConfirmDelete() string {
	s := "⚠️  Confirm Deletion\n"
	s += "===================\n\n"

	if m.confirmAction == "delete_spec" {
		var specTitle string
		for _, spec := range m.specs {
			if spec.ID == m.selectedSpecID {
				specTitle = spec.Title
				break
			}
		}
		s += fmt.Sprintf("Are you sure you want to delete the specification '%s'?\n\n", specTitle)
	} else if m.confirmAction == "delete_link" {
		if m.cursor < len(m.links) {
			selectedLink := m.links[m.cursor]
			s += fmt.Sprintf("Are you sure you want to delete the link to commit %s?\n\n", selectedLink.CommitID[:12]+"...")
		}
	}

	s += "Press 'y' to confirm, 'n' or Esc to cancel"
	return s
}

// getChildSpecs retrieves child specifications for the given spec
func (m *Model) getChildSpecs(specID string) ([]interactive.Spec, error) {
	linkedSpecs, err := m.app.specService.GetChildren(specID)
	if err != nil {
		return nil, err
	}

	specs := make([]interactive.Spec, 0, len(linkedSpecs))
	for _, spec := range linkedSpecs {
		specs = append(specs, interactive.Spec{
			ID:      spec.ID,
			Title:   spec.Title,
			Content: spec.Content,
		})
	}
	return specs, nil
}

// combinedService adapts both LinkService and SpecService to provide
// the interface needed by speclistview
type combinedService struct {
	linkService services.LinkService
	specService services.SpecService
}

func (cs *combinedService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return cs.linkService.GetCommitsForSpec(specID)
}

func (cs *combinedService) GetChildSpecs(specID *string) ([]*models.SpecNode, error) {
	if specID == nil {
		// Get all specs and filter for top-level ones (those without parents)
		allSpecs, err := cs.specService.ListSpecs()
		if err != nil {
			return nil, err
		}

		var topLevelSpecs []*models.SpecNode
		for _, spec := range allSpecs {
			// Check if this spec has any parents
			parents, err := cs.specService.GetParents(spec.ID)
			if err != nil {
				continue // Skip specs we can't check
			}

			// If no parents, it's a top-level spec
			if len(parents) == 0 {
				topLevelSpecs = append(topLevelSpecs, spec)
			}
		}

		return topLevelSpecs, nil
	}

	return cs.specService.GetChildren(*specID)
}

func (cs *combinedService) GetSpecByID(specID string) (*interactive.Spec, error) {
	specNode, err := cs.specService.GetSpec(specID)
	if err != nil {
		return nil, err
	}
	if specNode == nil {
		return nil, nil
	}

	// Convert models.SpecNode to interactive.Spec
	return &interactive.Spec{
		ID:      specNode.ID,
		Title:   specNode.Title,
		Content: specNode.Content,
	}, nil
}

func (cs *combinedService) GetParentSpec(specID string) (*interactive.Spec, error) {
	parents, err := cs.specService.GetParents(specID)
	if err != nil {
		return nil, err
	}

	if len(parents) == 0 {
		return nil, nil // No parent
	}

	// For simplicity, return the first parent if multiple exist
	parent := parents[0]

	return &interactive.Spec{
		ID:      parent.ID,
		Title:   parent.Title,
		Content: parent.Content,
	}, nil
}

func (cs *combinedService) GetRootSpec() (*interactive.Spec, error) {
	rootNode, err := cs.specService.GetRootSpec()
	if err != nil {
		return nil, err
	}
	if rootNode == nil {
		return nil, nil
	}

	// Convert models.SpecNode to interactive.Spec
	return &interactive.Spec{
		ID:      rootNode.ID,
		Title:   rootNode.Title,
		Content: rootNode.Content,
	}, nil
}
