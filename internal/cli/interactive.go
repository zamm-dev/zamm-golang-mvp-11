package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	interactive "github.com/yourorg/zamm-mvp/internal/cli/interactive"
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
	specLinks           []specLinkItem
	selectedChildSpecID string
	inputLinkType       string
}

type linkItem struct {
	ID        string
	CommitID  string
	RepoPath  string
	LinkType  string
	CreatedAt string
}

type specLinkItem struct {
	ID           string
	ParentSpecID string
	ChildSpecID  string
	ParentTitle  string
	ChildTitle   string
	LinkType     string
	CreatedAt    string
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

type specLinksLoadedMsg struct {
	links []specLinkItem
	err   error
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
			return m.updateLinkSpecToSpecSelection(msg)
		case UnlinkSpecFromSpecSelection:
			return m.updateUnlinkSpecFromSpecSelection(msg)
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

	case specLinksLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading spec links: %v", msg.err)
			m.showMessage = true
			return m, nil
		}
		m.specLinks = msg.links

		if len(m.specLinks) == 0 {
			m.message = "No specification links found."
			m.showMessage = true
			return m, nil
		}

		m.state = UnlinkSpecFromSpecSelection
		m.cursor = 0
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

// loadSpecLinksForSpecCmd returns a command to load spec-to-spec links for the selected spec
func (m *Model) loadSpecLinksForSpecCmd() tea.Cmd {
	return func() tea.Msg {
		parents, err := m.app.specService.GetParentSpecs(m.selectedSpecID)
		if err != nil {
			return specLinksLoadedMsg{err: err}
		}

		children, err := m.app.specService.GetChildSpecs(m.selectedSpecID)
		if err != nil {
			return specLinksLoadedMsg{err: err}
		}

		// Combine parents and children into a single list for display
		var linkItems []specLinkItem

		// Add parents (where selectedSpec is the child)
		for _, link := range parents {
			var parentTitle string
			for _, spec := range m.specs {
				if spec.ID == link.FromSpecID {
					parentTitle = spec.Title
					break
				}
			}
			linkItems = append(linkItems, specLinkItem{
				ID:           link.ID,
				ParentSpecID: link.FromSpecID,
				ChildSpecID:  link.ToSpecID,
				ParentTitle:  parentTitle,
				ChildTitle:   "", // Current spec
				LinkType:     link.LinkType,
				CreatedAt:    link.CreatedAt.Format("2006-01-02 15:04"),
			})
		}

		// Add children (where selectedSpec is the parent)
		for _, link := range children {
			var childTitle string
			for _, spec := range m.specs {
				if spec.ID == link.ToSpecID {
					childTitle = spec.Title
					break
				}
			}
			linkItems = append(linkItems, specLinkItem{
				ID:           link.ID,
				ParentSpecID: link.FromSpecID,
				ChildSpecID:  link.ToSpecID,
				ParentTitle:  "", // Current spec
				ChildTitle:   childTitle,
				LinkType:     link.LinkType,
				CreatedAt:    link.CreatedAt.Format("2006-01-02 15:04"),
			})
		}

		return specLinksLoadedMsg{links: linkItems}
	}
}

// linkSpecsCmd returns a command to create a hierarchical link between specs
func (m *Model) linkSpecsCmd(parentSpecID, childSpecID, linkType string) tea.Cmd {
	return func() tea.Msg {
		link, err := m.app.specService.LinkSpecs(parentSpecID, childSpecID, linkType)
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
			linkType, parentTitle, childTitle, link.ID)}
	}
}

// unlinkSpecsCmd returns a command to remove a hierarchical link between specs
func (m *Model) unlinkSpecsCmd(parentSpecID, childSpecID string) tea.Cmd {
	return func() tea.Msg {
		err := m.app.specService.UnlinkSpecs(parentSpecID, childSpecID)
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
		return m.renderLinkSpecToSpecSelection()
	case UnlinkSpecFromSpecSelection:
		return m.renderUnlinkSpecFromSpecSelection()
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
func (m *Model) renderLinkSpecToSpecSelection() string {
	s := "🔗 Link to Specification\n"
	s += "========================\n\n"

	// Find selected spec title
	var currentSpecTitle string
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			currentSpecTitle = spec.Title
			break
		}
	}

	s += fmt.Sprintf("Link '%s' to which specification?\n\n", currentSpecTitle)

	if len(m.specs) == 0 {
		s += "No specifications available.\n\n"
		s += "Press Esc to go back"
		return s
	}

	for i, spec := range m.specs {
		// Skip the current spec
		if spec.ID == m.selectedSpecID {
			continue
		}

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, spec.Title)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to select, Esc to go back"
	return s
}

// renderUnlinkSpecFromSpecSelection renders the spec unlink selection menu
func (m *Model) renderUnlinkSpecFromSpecSelection() string {
	s := "🗑️  Remove Specification Link\n"
	s += "=============================\n\n"

	if len(m.specLinks) == 0 {
		s += "No specification links found.\n\n"
		s += "Press Esc to go back"
		return s
	}

	// Find selected spec title
	var currentSpecTitle string
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			currentSpecTitle = spec.Title
			break
		}
	}

	s += fmt.Sprintf("Specification links for '%s':\n\n", currentSpecTitle)

	for i, link := range m.specLinks {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		if link.ParentSpecID == m.selectedSpecID {
			// Current spec is the parent
			s += fmt.Sprintf("%s → %s (%s)\n", cursor, link.ChildTitle, link.LinkType)
		} else {
			// Current spec is the child
			s += fmt.Sprintf("%s ← %s (%s)\n", cursor, link.ParentTitle, link.LinkType)
		}
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to remove, Esc to go back"
	return s
}

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
		} else if m.cursor == 1 {
			// Unlink from Parent Spec
			m.resetInputs()
			m.state = UnlinkSpecFromSpecSelection
			m.cursor = 0
			return m, m.loadSpecLinksForSpecCmd()
		}
	}
	return m, nil
}

// updateLinkSpecToSpecSelection handles spec-to-spec linking selection
func (m *Model) updateLinkSpecToSpecSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = LinkTypeSelection
		m.cursor = 0
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.specs)-1 {
			m.cursor++
		}
	case "enter", " ":
		if m.cursor < len(m.specs) {
			// Skip if trying to link to itself
			selectedSpec := m.specs[m.cursor]
			if selectedSpec.ID == m.selectedSpecID {
				return m, nil
			}
			m.resetInputs()
			m.selectedChildSpecID = selectedSpec.ID
			m.state = LinkSpecToSpecType
			m.promptText = "Enter link type (or press Enter for 'child'):"
			m.textInput.Focus()
			return m, nil
		}
	}
	return m, nil
}

// updateUnlinkSpecFromSpecSelection handles spec-to-spec unlinking selection
func (m *Model) updateUnlinkSpecFromSpecSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = UnlinkTypeSelection
		m.cursor = 0
		return m, nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.specLinks)-1 {
			m.cursor++
		}
	case "enter", " ":
		if m.cursor < len(m.specLinks) {
			selectedLink := m.specLinks[m.cursor]
			return m, m.unlinkSpecsCmd(selectedLink.ParentSpecID, selectedLink.ChildSpecID)
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
		return m, nil
	case tea.KeyEnter:
		linkType := strings.TrimSpace(m.textInput.Value())
		if linkType == "" {
			linkType = "child"
		}
		return m, m.linkSpecsCmd(m.selectedSpecID, m.selectedChildSpecID, linkType)
	}
	return m, nil
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

func (cs *combinedService) GetChildSpecs(specID string) ([]*models.SpecSpecLink, error) {
	return cs.specService.GetChildSpecs(specID)
}
