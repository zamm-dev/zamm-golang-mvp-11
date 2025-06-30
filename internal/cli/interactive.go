package cli

import (
	"fmt"
	"path/filepath"
	"strings"

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
	CreateSpecTitle
	CreateSpecContent
	EditSpecTitle
	EditSpecContent
	LinkSpecCommit
	LinkSpecRepo
	LinkSpecType
	ConfirmDelete
	// New states for link type selection and hierarchical specs
	LinkTypeSelection
	UnlinkTypeSelection
	LinkSpecToSpecSelection
	LinkSpecToSpecType
	UnlinkSpecFromSpecSelection
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

	// Hierarchical spec links
	selectedChildSpecID string
	inputLinkType       string

	// Spec selector components
	specSelector common.SpecSelector
}

type linkItem struct {
	ID        string
	CommitID  string
	RepoPath  string
	LinkType  string
	CreatedAt string
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
	// Check for and run migrations if needed
	fmt.Println("Checking for database migrations...")
	if err := a.storage.RunMigrationsIfNeeded(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
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
		specSelector:   common.NewSpecSelector(common.DefaultSpecSelectorConfig()),
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
		m.specSelector.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if m.showMessage {
			if msg.String() == "enter" || msg.String() == " " || msg.String() == "esc" {
				m.showMessage = false
				m.message = ""
				m.state = SpecListView
				m.cursor = 0
				return m, m.loadSpecsCmd()
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
		case LinkTypeSelection:
			return m.updateLinkTypeSelection(msg)
		case UnlinkTypeSelection:
			return m.updateUnlinkTypeSelection(msg)
		case LinkSpecToSpecSelection:
			// Use spec selector for spec selection
			var cmd tea.Cmd
			selector, selectorCmd := m.specSelector.Update(msg)
			m.specSelector = *selector
			cmd = selectorCmd

			// Handle escape key to go back
			if msg.String() == "esc" {
				m.state = LinkTypeSelection
				m.cursor = 0
				return m, nil
			}

			return m, cmd
		case UnlinkSpecFromSpecSelection:
			// Use spec selector for spec unlink selection
			var cmd tea.Cmd
			selector, selectorCmd := m.specSelector.Update(msg)
			m.specSelector = *selector
			cmd = selectorCmd

			// Handle escape key to go back
			if msg.String() == "esc" {
				m.state = UnlinkTypeSelection
				m.cursor = 0
				return m, nil
			}

			return m, cmd
		case CreateSpecTitle, CreateSpecContent,
			EditSpecTitle, EditSpecContent,
			LinkSpecCommit, LinkSpecRepo, LinkSpecType,
			LinkSpecToSpecType:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)

			switch m.state {
			case CreateSpecTitle, CreateSpecContent:
				return m.updateCreateSpec(msg)
			case EditSpecTitle, EditSpecContent:
				return m.updateEditSpec(msg)
			case LinkSpecCommit, LinkSpecRepo, LinkSpecType:
				return m.updateLinkSpec(msg)
			case LinkSpecToSpecType:
				return m.updateLinkSpecToSpecType(msg)
			}
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
		m.specListView.SetSpecs(m.specs)
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
			m.state = CreateSpecTitle
			m.promptText = "Enter title:"
			m.textInput.Focus()
			return m, nil
		}

	case speclistview.EditSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.editingSpecID = msg.SpecID
			// Find current title for pre-filling
			for _, spec := range m.specs {
				if spec.ID == msg.SpecID {
					m.inputTitle = spec.Title
					m.textInput.SetValue(spec.Title)
					break
				}
			}
			m.state = EditSpecTitle
			m.promptText = "Enter new title (or press Enter to keep current):"
			m.textInput.Focus()
			return m, nil
		}

	case speclistview.LinkCommitSpecMsg:
		if m.state == SpecListView {
			m.resetInputs()
			m.selectedSpecID = msg.SpecID
			m.state = LinkTypeSelection
			m.cursor = 0
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
			m.state = UnlinkTypeSelection
			m.cursor = 0
			return m, nil
		}

	case common.SpecSelectedMsg:
		if m.state == LinkSpecToSpecSelection {
			// Handle spec selection for linking
			if msg.Spec.ID != "" && msg.Spec.ID != m.selectedSpecID {
				m.resetInputs()
				m.selectedChildSpecID = msg.Spec.ID
				m.state = LinkSpecToSpecType
				m.promptText = "Enter link type (or press Enter for 'child'):"
				m.textInput.Focus()
				return m, nil
			}
		} else if m.state == UnlinkSpecFromSpecSelection {
			// Handle spec selection for unlinking
			// Just unlink between the current spec and the selected spec
			selectedSpecID := msg.Spec.ID
			return m, m.unlinkSpecsCmd(m.selectedSpecID, selectedSpecID)
		}

	case speclistview.ExitMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
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
		return m, m.loadSpecsCmd()
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

// updateCreateSpec handles updates for creating new specifications
func (m *Model) updateCreateSpec(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = SpecListView
		m.cursor = 0
		m.resetInputs()
		return m, m.loadSpecsCmd()
	case tea.KeyEnter:
		if m.state == CreateSpecTitle {
			if strings.TrimSpace(m.textInput.Value()) == "" {
				return m, nil // Don't proceed with empty title
			}
			m.inputTitle = strings.TrimSpace(m.textInput.Value())
			m.textInput.Reset()
			m.state = CreateSpecContent
			m.promptText = "Enter content (press Ctrl+S to finish):"
			return m, nil
		} else if m.state == CreateSpecContent {
			// Add line to content
			m.contentLines = append(m.contentLines, m.textInput.Value())
			m.textInput.Reset()
			return m, nil
		}
	case tea.KeyCtrlS:
		if m.state == CreateSpecContent {
			content := strings.Join(m.contentLines, "\n")
			return m, m.createSpecCmd(m.inputTitle, content)
		}
	}
	return m, nil
}

// updateEditSpec handles updates for editing specifications
func (m *Model) updateEditSpec(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = SpecListView
		m.cursor = 0
		m.resetInputs()
		return m, m.loadSpecsCmd()
	case tea.KeyEnter:
		if m.state == EditSpecTitle {
			if strings.TrimSpace(m.textInput.Value()) != "" {
				m.inputTitle = strings.TrimSpace(m.textInput.Value())
			}
			m.textInput.Reset()
			m.state = EditSpecContent
			m.promptText = "Enter new content (press Ctrl+S to finish, or Ctrl+K to keep existing):"
			return m, nil
		} else if m.state == EditSpecContent {
			// Add line to content
			m.contentLines = append(m.contentLines, m.textInput.Value())
			m.textInput.Reset()
			return m, nil
		}
	case tea.KeyCtrlS:
		if m.state == EditSpecContent {
			content := strings.Join(m.contentLines, "\n")
			if strings.TrimSpace(content) == "" {
				// Keep existing content
				for _, spec := range m.specs {
					if spec.ID == m.editingSpecID {
						content = spec.Content
						break
					}
				}
			}
			return m, m.updateSpecCmd(m.editingSpecID, m.inputTitle, content)
		}
	case tea.KeyCtrlK:
		if m.state == EditSpecContent {
			// Keep existing content
			var existingContent string
			for _, spec := range m.specs {
				if spec.ID == m.editingSpecID {
					existingContent = spec.Content
					break
				}
			}
			return m, m.updateSpecCmd(m.editingSpecID, m.inputTitle, existingContent)
		}
	}
	return m, nil
}

// updateLinkSpec handles updates for linking specifications
func (m *Model) updateLinkSpec(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = SpecListView
		m.cursor = 0
		m.resetInputs()
		return m, m.loadSpecsCmd()
	case tea.KeyEnter:
		switch m.state {
		case LinkSpecCommit:
			if strings.TrimSpace(m.textInput.Value()) == "" {
				return m, nil
			}
			m.inputCommit = strings.TrimSpace(m.textInput.Value())
			m.textInput.Reset()
			m.state = LinkSpecRepo
			m.promptText = "Enter repository path (or press Enter for default):"
			return m, nil
		case LinkSpecRepo:
			m.inputRepo = strings.TrimSpace(m.textInput.Value())
			if m.inputRepo == "" {
				m.inputRepo = m.app.config.Git.DefaultRepo
			}
			m.textInput.Reset()
			m.state = LinkSpecType
			m.promptText = "Enter link type (or press Enter for 'implements'):"
			return m, nil
		case LinkSpecType:
			m.inputType = strings.TrimSpace(m.textInput.Value())
			if m.inputType == "" {
				m.inputType = "implements"
			}
			return m, m.createLinkCmd(m.selectedSpecID, m.inputCommit, m.inputRepo, m.inputType)
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
		return m, m.loadSpecsCmd()
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
		return m, m.loadSpecsCmd()
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
				ID:        spec.ID,
				Title:     spec.Title,
				Content:   spec.Content,
				CreatedAt: spec.CreatedAt.Format("2006-01-02 15:04"),
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
				ID:        link.ID,
				CommitID:  link.CommitID,
				RepoPath:  link.RepoPath,
				LinkType:  link.LinkType,
				CreatedAt: link.CreatedAt.Format("2006-01-02 15:04"),
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
	m.selectedChildSpecID = ""
	m.inputLinkType = ""
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
func (m *Model) createLinkCmd(specID, commitID, repoPath, linkType string) tea.Cmd {
	return func() tea.Msg {
		link, err := m.app.linkService.LinkSpecToCommit(specID, commitID, repoPath, linkType)
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

		return operationCompleteMsg{message: fmt.Sprintf("✅ Created link between '%s' and commit %s (ID: %s). Press Enter to continue...",
			specTitle, commitID[:12]+"...", link.ID)}
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

// linkSpecsCmd returns a command to create a hierarchical link between specs
func (m *Model) linkSpecsCmd(parentSpecID, childSpecID string) tea.Cmd {
	return func() tea.Msg {
		link, err := m.app.specService.AddChildToParent(childSpecID, parentSpecID)
		if err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		// Find spec titles for display
		var parentTitle, childTitle string
		for _, spec := range m.specs {
			if spec.ID == parentSpecID {
				parentTitle = spec.Title
			}
			if spec.ID == childSpecID {
				childTitle = spec.Title
			}
		}

		return operationCompleteMsg{message: fmt.Sprintf("✅ Created %s link from '%s' to '%s' (ID: %s). Press Enter to continue...",
			"parent-child", parentTitle, childTitle, link.ID)}
	}
}

// unlinkSpecsCmd returns a command to remove a hierarchical link between specs
func (m *Model) unlinkSpecsCmd(parentSpecID, childSpecID string) tea.Cmd {
	return func() tea.Msg {
		err := m.app.specService.RemoveChildFromParent(childSpecID, parentSpecID)
		if err != nil {
			return operationCompleteMsg{message: fmt.Sprintf("Error: %v. Press Enter to continue...", err)}
		}

		// Find spec titles for display
		var parentTitle, childTitle string
		for _, spec := range m.specs {
			if spec.ID == parentSpecID {
				parentTitle = spec.Title
			}
			if spec.ID == childSpecID {
				childTitle = spec.Title
			}
		}

		return operationCompleteMsg{message: fmt.Sprintf("✅ Removed link from '%s' to '%s'. Press Enter to continue...",
			parentTitle, childTitle)}
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
	case LinkTypeSelection:
		return m.renderLinkTypeSelection()
	case UnlinkTypeSelection:
		return m.renderUnlinkTypeSelection()
	case LinkSpecToSpecSelection:
		return m.specSelector.View()
	case UnlinkSpecFromSpecSelection:
		return m.specSelector.View()
	case CreateSpecTitle, CreateSpecContent:
		return m.renderCreateSpec()
	case EditSpecTitle, EditSpecContent:
		return m.renderEditSpec()
	case LinkSpecCommit, LinkSpecRepo, LinkSpecType:
		return m.renderLinkSpec()
	case LinkSpecToSpecType:
		return m.renderLinkSpecToSpecType()
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
		s += fmt.Sprintf("%s %s (%s, %s)\n", cursor, link.CommitID[:12]+"...", repoName, link.LinkType)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to delete, Esc to go back"
	return s
}

// renderCreateSpec renders the create specification form
func (m *Model) renderCreateSpec() string {
	s := "📝 Create New Specification\n"
	s += "===========================\n\n"

	if m.state == CreateSpecTitle {
		s += m.promptText + "\n"
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to continue, Esc to cancel"
	} else if m.state == CreateSpecContent {
		s += fmt.Sprintf("Title: %s\n\n", m.inputTitle)
		s += m.promptText + "\n\n"

		// Show entered content lines
		for _, line := range m.contentLines {
			s += "  " + line + "\n"
		}
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to add line, Ctrl+S to finish, Esc to cancel"
	}

	return s
}

// renderEditSpec renders the edit specification form
func (m *Model) renderEditSpec() string {
	s := "✏️  Edit Specification\n"
	s += "======================\n\n"

	if m.state == EditSpecTitle {
		// Show current title
		var currentTitle string
		for _, spec := range m.specs {
			if spec.ID == m.editingSpecID {
				currentTitle = spec.Title
				break
			}
		}
		s += fmt.Sprintf("Current title: %s\n\n", currentTitle)
		s += m.promptText + "\n"
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to continue, Esc to cancel"
	} else if m.state == EditSpecContent {
		s += fmt.Sprintf("Title: %s\n\n", m.inputTitle)
		s += m.promptText + "\n\n"

		// Show entered content lines
		for _, line := range m.contentLines {
			s += "  " + line + "\n"
		}
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to add line, Ctrl+S to finish, Ctrl+K to keep existing, Esc to cancel"
	}

	return s
}

// renderLinkSpec renders the link specification form
func (m *Model) renderLinkSpec() string {
	s := "🔗 Link Specification to Commit\n"
	s += "===============================\n\n"

	// Show selected spec
	var specTitle string
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			specTitle = spec.Title
			break
		}
	}
	s += fmt.Sprintf("Linking: %s\n\n", specTitle)

	switch m.state {
	case LinkSpecCommit:
		s += m.promptText + "\n"
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to continue, Esc to cancel"
	case LinkSpecRepo:
		s += fmt.Sprintf("Commit: %s\n\n", m.inputCommit)
		s += m.promptText + "\n"
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to continue, Esc to cancel"
	case LinkSpecType:
		s += fmt.Sprintf("Commit: %s\n", m.inputCommit)
		s += fmt.Sprintf("Repository: %s\n\n", m.inputRepo)
		s += m.promptText + "\n"
		s += m.textInput.View() + "\n\n"
		s += "Press Enter to finish, Esc to cancel"
	}

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

// renderLinkTypeSelection renders the link type selection menu
func (m *Model) renderLinkTypeSelection() string {
	s := "🔗 Link Type Selection\n"
	s += "=====================\n\n"

	// Find selected spec title
	var specTitle string
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			specTitle = spec.Title
			break
		}
	}

	s += fmt.Sprintf("Select link type for '%s':\n\n", specTitle)

	options := []string{
		"Git Commit",
		"Parent Specification",
	}

	for i, option := range options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %d. %s\n", cursor, i+1, option)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to select, Esc to go back"
	return s
}

// renderUnlinkTypeSelection renders the unlink type selection menu
func (m *Model) renderUnlinkTypeSelection() string {
	s := "🗑️  Unlink Type Selection\n"
	s += "=========================\n\n"

	// Find selected spec title
	var specTitle string
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			specTitle = spec.Title
			break
		}
	}

	s += fmt.Sprintf("Select link type to remove from '%s':\n\n", specTitle)

	options := []string{
		"Git Commit Links",
		"Specification Links",
	}

	for i, option := range options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %d. %s\n", cursor, i+1, option)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to select, Esc to go back"
	return s
}

// renderLinkSpecToSpecSelection renders the spec-to-spec selection menu
// renderLinkSpecToSpecType renders the link type input form
func (m *Model) renderLinkSpecToSpecType() string {
	s := "🔗 Specification Link Type\n"
	s += "==========================\n\n"

	// Find spec titles
	var parentTitle, childTitle string
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			parentTitle = spec.Title
		}
		if spec.ID == m.selectedChildSpecID {
			childTitle = spec.Title
		}
	}

	s += fmt.Sprintf("Linking '%s' to '%s'\n\n", parentTitle, childTitle)
	s += m.promptText + "\n"
	s += m.textInput.View() + "\n\n"
	s += "Press Enter to finish, Esc to cancel"

	return s
}

// updateLinkTypeSelection handles the link type selection menu
func (m *Model) updateLinkTypeSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = SpecListView
		m.cursor = 0
		return m, m.loadSpecsCmd()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 { // 0: Git Commit, 1: Parent Spec
			m.cursor++
		}
	case "enter", " ":
		if m.cursor == 0 {
			// Link to Git Commit
			m.resetInputs()
			m.state = LinkSpecCommit
			m.promptText = "Enter commit hash:"
			m.textInput.Focus()
			return m, nil
		} else if m.cursor == 1 {
			// Link to Parent Spec
			m.resetInputs()
			m.state = LinkSpecToSpecSelection
			m.cursor = 0

			// Configure spec selector for linking mode
			config := common.SpecSelectorConfig{
				Title: "🔗 Link to Specification",
			}
			m.specSelector = common.NewSpecSelector(config)

			// Filter out the current spec
			filteredSpecs := make([]interactive.Spec, 0, len(m.specs))
			for _, spec := range m.specs {
				if spec.ID != m.selectedSpecID {
					filteredSpecs = append(filteredSpecs, spec)
				}
			}
			m.specSelector.SetSpecs(filteredSpecs)
			m.specSelector.SetSize(m.terminalWidth, m.terminalHeight)

			return m, nil
		}
	}
	return m, nil
}

// updateUnlinkTypeSelection handles the unlink type selection menu
func (m *Model) updateUnlinkTypeSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = SpecListView
		m.cursor = 0
		return m, m.loadSpecsCmd()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 { // 0: Git Commit, 1: Parent Spec
			m.cursor++
		}
	case "enter", " ":
		if m.cursor == 0 {
			// Unlink from Git Commit
			return m, m.loadLinksForSpecCmd()
		} else if m.cursor == 1 { // Unlink from Parent Spec
			m.resetInputs()

			// Get child specs that can be unlinked directly
			unlinkSpecs, err := m.getLinkedSpecs(m.selectedSpecID, models.Incoming)
			if err != nil {
				m.message = fmt.Sprintf("Error loading linked specs: %v", err)
				m.showMessage = true
				return m, nil
			}

			if len(unlinkSpecs) == 0 {
				m.message = "No child specifications found to unlink."
				m.showMessage = true
				return m, nil
			}

			// Configure the spec selector for unlink mode
			config := common.SpecSelectorConfig{
				Title: "🗑️ Remove Specification Link",
			}
			m.specSelector = common.NewSpecSelector(config)
			m.specSelector.SetSpecs(unlinkSpecs)
			m.specSelector.SetSize(m.terminalWidth, m.terminalHeight)

			m.state = UnlinkSpecFromSpecSelection
			m.cursor = 0
			return m, nil
		}
	}
	return m, nil
}

// updateLinkSpecToSpecType handles the link type input for spec-to-spec linking
func (m *Model) updateLinkSpecToSpecType(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.state = LinkSpecToSpecSelection
		m.cursor = 0
		m.resetInputs()

		// Reconfigure spec selector and filter out current spec
		config := common.SpecSelectorConfig{
			Title: "🔗 Link to Specification",
		}
		m.specSelector = common.NewSpecSelector(config)

		// Filter out the current spec
		filteredSpecs := make([]interactive.Spec, 0, len(m.specs))
		for _, spec := range m.specs {
			if spec.ID != m.selectedSpecID {
				filteredSpecs = append(filteredSpecs, spec)
			}
		}
		m.specSelector.SetSpecs(filteredSpecs)
		m.specSelector.SetSize(m.terminalWidth, m.terminalHeight)

		return m, nil
	case tea.KeyEnter:
		linkType := strings.TrimSpace(m.textInput.Value())
		if linkType == "" {
			linkType = "child"
		}
		return m, m.linkSpecsCmd(m.selectedSpecID, m.selectedChildSpecID)
	}
	return m, nil
}

// getLinkedSpecs retrieves linked specifications based on direction
func (m *Model) getLinkedSpecs(specID string, direction models.Direction) ([]interactive.Spec, error) {
	var linkedSpecs []*models.SpecNode
	var err error

	if direction == models.Incoming {
		linkedSpecs, err = m.app.specService.GetParents(specID)
	} else {
		linkedSpecs, err = m.app.specService.GetChildren(specID)
	}

	if err != nil {
		return nil, err
	}

	specs := make([]interactive.Spec, 0, len(linkedSpecs))
	for _, spec := range linkedSpecs {
		specs = append(specs, interactive.Spec{
			ID:        spec.ID,
			Title:     spec.Title,
			Content:   spec.Content,
			CreatedAt: spec.CreatedAt.Format("2006-01-02 15:04"),
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

func (cs *combinedService) GetChildSpecs(specID string) ([]*models.SpecNode, error) {
	return cs.specService.GetChildren(specID)
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

func (cs *combinedService) GetTopLevelSpecs() ([]interactive.Spec, error) {
	// Get all specs and filter for top-level ones (those without parents)
	allSpecs, err := cs.specService.ListSpecs()
	if err != nil {
		return nil, err
	}

	var topLevelSpecs []interactive.Spec
	for _, spec := range allSpecs {
		// Check if this spec has any parents
		parents, err := cs.specService.GetParents(spec.ID)
		if err != nil {
			continue // Skip specs we can't check
		}

		// If no parents, it's a top-level spec
		if len(parents) == 0 {
			topLevelSpecs = append(topLevelSpecs, interactive.Spec{
				ID:      spec.ID,
				Title:   spec.Title,
				Content: spec.Content,
			})
		}
	}

	return topLevelSpecs, nil
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
		ID:        parent.ID,
		Title:     parent.Title,
		Content:   parent.Content,
		CreatedAt: parent.CreatedAt.Format("2006-01-02 15:04"),
	}, nil
}
