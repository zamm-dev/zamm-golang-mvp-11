package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	interactive "github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/common"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/cli/interactive/speclistview"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/models"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
)

// MenuState represents the current state of the interactive menu
type MenuState int

const (
	SpecListView MenuState = iota
	NodeTypeSelection
	LinkSelection
	SpecEditor
	ImplementationForm
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
	specListView   speclistview.SpecExplorer

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

	// Node type selection
	nodeTypeSelector *common.NodeTypeSelector
	pendingNodeType  common.NodeType

	// Implementation form data
	implForm       common.ImplementationForm
	implRepoURL    *string
	implBranch     *string
	implFolderPath *string

	// Debug logging
	debugWriter io.Writer
}

type linkItem struct {
	ID        string
	CommitID  string
	RepoPath  string
	LinkLabel string
}

// Custom messages
type nodesLoadedMsg struct {
	nodes []interactive.Spec
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
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Interactive mode for managing specs and links",
		Long:  "Start an interactive session to manage specifications and links using arrow keys for navigation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get the debug flag value from this command's local flags
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return fmt.Errorf("failed to get debug flag: %w", err)
			}
			return a.runInteractiveMode(debug)
		},
	}

	// Add debug flag specific to this command
	cmd.Flags().Bool("debug", false, "Enable debug logging for bubbletea messages")

	return cmd
}

// NewModel creates a new Model with the given debug writer
func NewModel(app *App, debugWriter io.Writer) *Model {
	ti := textinput.New()
	ti.Focus()

	combinedSvc := &combinedService{
		linkService: app.linkService,
		specService: app.specService,
	}

	return &Model{
		app:            app,
		state:          SpecListView,
		textInput:      ti,
		specListView:   speclistview.NewSpecExplorer(combinedSvc),
		linkEditor:     common.NewLinkEditor(common.LinkEditorConfig{Title: "", DefaultRepo: app.config.Git.DefaultRepo, CurrentSpecID: "", CurrentSpecTitle: "", IsUnlinkMode: false, IsMoveMode: false}, app.linkService, app.specService),
		terminalWidth:  80, // Default terminal width
		terminalHeight: 24, // Default terminal height
		debugWriter:    debugWriter,
	}
}

// runInteractiveMode starts the interactive mode with TUI
func (a *App) runInteractiveMode(debug bool) error {
	// Perform complete initialization
	if err := a.InitializeZamm(); err != nil {
		return fmt.Errorf("failed to initialize zamm: %w", err)
	}

	var debugWriter io.Writer
	var debugFile *os.File
	if debug {
		var err error
		debugFile, err = createDebugLogFile()
		if err != nil {
			return fmt.Errorf("failed to create debug log file: %w", err)
		}
		debugWriter = debugFile
	}

	model := NewModel(a, debugWriter)

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	// Ensure proper cleanup of debug file on program exit
	if debugFile != nil {
		if closeErr := debugFile.Close(); closeErr != nil {
			// Log to stderr but don't override the main error
			fmt.Fprintf(os.Stderr, "Warning: failed to close debug log file: %v\n", closeErr)
		}
	}

	return err
}

// Init is the first function that will be called
func (m *Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.loadSpecsCmd())
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Debug logging: dump all messages when debug writer is available
	if m.debugWriter != nil {
		spew.Fdump(m.debugWriter, msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height
		m.specListView.SetSize(msg.Width, msg.Height)
		m.specEditor.SetSize(msg.Width, msg.Height)
		m.linkEditor.SetSize(msg.Width, msg.Height)
		if m.nodeTypeSelector != nil {
			m.nodeTypeSelector.SetSize(msg.Width, msg.Height)
		}
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

	case nodesLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading specs: %v", msg.err)
			m.showMessage = true
			return m, nil
		}
		m.specs = msg.nodes
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

			// First show node type selector
			nts := common.NewNodeTypeSelector("Choose node type to create:")
			m.nodeTypeSelector = &nts
			m.nodeTypeSelector.SetSize(m.terminalWidth, m.terminalHeight)
			m.state = NodeTypeSelection
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
				Title:            "Link Specification",
				DefaultRepo:      m.app.config.Git.DefaultRepo,
				CurrentSpecID:    msg.SpecID,
				CurrentSpecTitle: specTitle,
				IsUnlinkMode:     false,
				IsMoveMode:       false,
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
				Title:            "Remove Links",
				DefaultRepo:      m.app.config.Git.DefaultRepo,
				CurrentSpecID:    msg.SpecID,
				CurrentSpecTitle: specTitle,
				IsUnlinkMode:     true,
				IsMoveMode:       false,
			}
			m.linkEditor = common.NewLinkEditor(config, m.app.linkService, m.app.specService)
			m.linkEditor.SetSize(m.terminalWidth, m.terminalHeight)

			m.state = LinkEditor
			return m, nil
		}

	case speclistview.MoveSpecMsg:
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

			// Create link editor for move mode
			config := common.LinkEditorConfig{
				Title:            "Move Spec",
				DefaultRepo:      m.app.config.Git.DefaultRepo,
				CurrentSpecID:    msg.SpecID,
				CurrentSpecTitle: specTitle,
				IsUnlinkMode:     false,
				IsMoveMode:       true,
			}
			m.linkEditor = common.NewLinkEditor(config, m.app.linkService, m.app.specService)
			m.linkEditor.SetSize(m.terminalWidth, m.terminalHeight)

			m.state = LinkEditor
			return m, m.linkEditor.Init()
		}

	case common.SpecEditorCompleteMsg:
		if m.state == SpecEditor {
			// Determine if this is create or edit based on whether we have an editingSpecID
			if m.editingSpecID != "" {
				// Edit existing spec
				return m, m.updateSpecCmd(m.editingSpecID, msg.Title, msg.Content)
			} else {
				// If creating an implementation, collect extra fields next
				if m.pendingNodeType == common.NodeTypeImplementation {
					m.inputTitle = msg.Title
					m.inputContent = msg.Content
					m.implForm = common.NewImplementationForm("🔧 Implementation Details")
					m.implForm.SetSize(m.terminalWidth, m.terminalHeight)
					m.state = ImplementationForm
					return m, nil
				}
				// Otherwise create a spec immediately
				return m, m.createSpecCmd(msg.Title, msg.Content)
			}
		}

	case common.ImplementationFormSubmitMsg:
		if m.state == ImplementationForm {
			m.implRepoURL = msg.RepoURL
			m.implBranch = msg.Branch
			m.implFolderPath = msg.FolderPath
			// Use stashed title/content
			return m, m.createImplementationCmd(m.inputTitle, m.inputContent)
		}

	case common.ImplementationFormCancelMsg:
		if m.state == ImplementationForm {
			m.state = SpecListView
			m.resetInputs()
			return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
		}

	case common.NodeTypeSelectedMsg:
		if m.state == NodeTypeSelection {
			m.pendingNodeType = msg.Type
			// Open the editor with an appropriate title
			var title string
			if msg.Type == common.NodeTypeImplementation {
				title = "🧩 Create New Implementation"
			} else {
				title = "📝 Create New Specification"
			}
			config := common.SpecEditorConfig{Title: title, InitialTitle: "", InitialContent: ""}
			m.specEditor = common.NewSpecEditor(config)
			m.specEditor.SetSize(m.terminalWidth, m.terminalHeight)
			m.state = SpecEditor
			return m, nil
		}

	case common.NodeTypeCancelledMsg:
		if m.state == NodeTypeSelection {
			m.state = SpecListView
			m.resetInputs()
			return m, tea.Batch(m.loadSpecsCmd(), m.specListView.Refresh())
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
	case NodeTypeSelection:
		var cmd tea.Cmd
		selector, cmd := m.nodeTypeSelector.Update(msg)
		if nts, ok := selector.(*common.NodeTypeSelector); ok {
			m.nodeTypeSelector = nts
		}
		return m, cmd
	case ImplementationForm:
		var cmd tea.Cmd
		formModel, cmd := m.implForm.Update(msg)
		if f, ok := formModel.(common.ImplementationForm); ok {
			m.implForm = f
		}
		return m, cmd
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
		switch m.confirmAction {
		case "delete_spec":
			return m, m.deleteSpecCmd(m.selectedSpecID)
		case "delete_link":
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
		nodes, err := m.app.specService.ListNodes()
		if err != nil {
			return nodesLoadedMsg{err: err}
		}

		// Include all node types (specs, projects, implementations)
		var nodeItems []interactive.Spec
		for _, node := range nodes {
			nodeItems = append(nodeItems, interactive.Spec{
				ID:      node.GetID(),
				Title:   node.GetTitle(),
				Content: node.GetContent(),
			})
		}

		return nodesLoadedMsg{nodes: nodeItems}
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
	m.implRepoURL = nil
	m.implBranch = nil
	m.implFolderPath = nil
	m.textInput.Reset()
	m.textInput.Blur()
	m.nodeTypeSelector = nil
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

// createImplementationCmd returns a command to create a new implementation node
func (m *Model) createImplementationCmd(title, content string) tea.Cmd {
	return func() tea.Msg {
		impl, err := m.app.specService.CreateImplementation(title, content, m.implRepoURL, m.implBranch, m.implFolderPath)
		if err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		// If there's a parent spec ID, create the parent-child relationship
		if m.parentSpecID != "" {
			_, err := m.app.specService.AddChildToParent(impl.ID, m.parentSpecID, "child")
			if err != nil {
				return operationCompleteMsg{message: fmt.Sprintf("Error creating parent-child relationship: %v. Press Enter to continue...", err)}
			}
		}

		return operationCompleteMsg{message: fmt.Sprintf("✅ Created implementation: %s. Press Enter to continue...", impl.Title)}
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
	case ImplementationForm:
		return m.implForm.View()
	case NodeTypeSelection:
		if m.nodeTypeSelector != nil {
			return m.nodeTypeSelector.View()
		}
		return "Choose node type..."
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

// combinedService adapts both LinkService and SpecService to provide
// the interface needed by speclistview
type combinedService struct {
	linkService services.LinkService
	specService services.SpecService
}

func (cs *combinedService) GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error) {
	return cs.linkService.GetCommitsForSpec(specID)
}

func (cs *combinedService) GetChildNodes(specID string) ([]models.Node, error) {
	nodes, err := cs.specService.GetChildren(specID)
	if err != nil {
		return nil, err
	}

	// Return all nodes, not just specs
	return nodes, nil
}

func (cs *combinedService) GetNodeByID(specID string) (models.Node, error) {
	node, err := cs.specService.GetNode(specID)
	if err != nil {
		return nil, err
	}

	return node, nil
}

func (cs *combinedService) GetParentNode(specID string) (models.Node, error) {
	parents, err := cs.specService.GetParents(specID)
	if err != nil {
		return nil, err
	}

	if len(parents) == 0 {
		return nil, nil // No parent
	}

	// For simplicity, return the first parent if multiple exist
	return parents[0], nil
}

func (cs *combinedService) GetRootNode() (models.Node, error) {
	rootNode, err := cs.specService.GetRootSpec()
	if err != nil {
		return nil, err
	}
	if rootNode == nil {
		return nil, nil
	}

	return rootNode, nil
}
