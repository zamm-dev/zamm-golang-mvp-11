package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// MenuState represents the current state of the interactive menu
type MenuState int

const (
	MainMenu MenuState = iota
	SpecSelection
	LinkSelection
)

// MenuAction represents an action that can be performed
type MenuAction int

const (
	ActionListSpecs MenuAction = iota
	ActionCreateSpec
	ActionEditSpec
	ActionDeleteSpec
	ActionLinkSpec
	ActionViewLinks
	ActionDeleteLink
	ActionExit
)

// Model represents the state of our TUI application
type Model struct {
	app            *App
	state          MenuState
	cursor         int
	specs          []specItem
	links          []linkItem
	choices        []string
	action         MenuAction
	selectedSpecID string
	message        string
	showMessage    bool
}

type specItem struct {
	ID        string
	Title     string
	Content   string
	CreatedAt string
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
	specs []specItem
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
	model := Model{
		app:   a,
		state: MainMenu,
		choices: []string{
			"📋 List specifications",
			"📝 Create new specification",
			"✏️  Edit specification",
			"🗑️  Delete specification",
			"🔗 Link specification to commit",
			"👀 View spec-commit links",
			"🗑️  Delete spec-commit link",
			"🚪 Exit",
		},
	}

	p := tea.NewProgram(&model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Init is the first function that will be called
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showMessage {
			if msg.String() == "enter" || msg.String() == " " || msg.String() == "esc" {
				m.showMessage = false
				m.message = ""
				m.state = MainMenu
				m.cursor = 0
				return m, nil
			}
			return m, nil
		}

		switch m.state {
		case MainMenu:
			return m.updateMainMenu(msg)
		case SpecSelection:
			return m.updateSpecSelection(msg)
		case LinkSelection:
			return m.updateLinkSelection(msg)
		}

	case specsLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Error loading specs: %v", msg.err)
			m.showMessage = true
			return m, nil
		}
		m.specs = msg.specs

		if len(m.specs) == 0 {
			m.message = "No specifications found."
			m.showMessage = true
			return m, nil
		}

		if m.action == ActionListSpecs {
			m.message = m.formatSpecsList()
			m.showMessage = true
			return m, nil
		}

		m.state = SpecSelection
		m.cursor = 0
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

		if m.action == ActionViewLinks {
			m.message = m.formatLinksList()
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
	}

	return m, nil
}

// updateMainMenu handles updates for the main menu
func (m *Model) updateMainMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case "enter", " ":
		return m.executeAction()
	}
	return m, nil
}

// updateSpecSelection handles updates for spec selection
func (m *Model) updateSpecSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = MainMenu
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
		if len(m.specs) > 0 {
			m.selectedSpecID = m.specs[m.cursor].ID
			return m.executeSpecAction()
		}
	}
	return m, nil
}

// updateLinkSelection handles updates for link selection
func (m *Model) updateLinkSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.state = MainMenu
		m.cursor = 0
		return m, nil
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

// executeAction executes the selected main menu action
func (m *Model) executeAction() (tea.Model, tea.Cmd) {
	m.action = MenuAction(m.cursor)

	switch m.action {
	case ActionListSpecs:
		return m, m.loadSpecsCmd()
	case ActionCreateSpec:
		return m, tea.Sequence(tea.ExitAltScreen, func() tea.Msg {
			m.interactiveCreateSpec()
			return operationCompleteMsg{message: "Spec creation completed. Press Enter to continue..."}
		}, tea.EnterAltScreen)
	case ActionEditSpec, ActionDeleteSpec, ActionLinkSpec, ActionViewLinks, ActionDeleteLink:
		return m, m.loadSpecsCmd()
	case ActionExit:
		return m, tea.Quit
	}

	return m, nil
}

// executeSpecAction executes the action on the selected spec
func (m *Model) executeSpecAction() (tea.Model, tea.Cmd) {
	switch m.action {
	case ActionEditSpec:
		return m, tea.Sequence(tea.ExitAltScreen, func() tea.Msg {
			msg := m.editSelectedSpec()
			return operationCompleteMsg{message: msg}
		}, tea.EnterAltScreen)
	case ActionDeleteSpec:
		return m, tea.Sequence(tea.ExitAltScreen, func() tea.Msg {
			msg := m.deleteSelectedSpec()
			return operationCompleteMsg{message: msg}
		}, tea.EnterAltScreen)
	case ActionLinkSpec:
		return m, tea.Sequence(tea.ExitAltScreen, func() tea.Msg {
			msg := m.linkSelectedSpec()
			return operationCompleteMsg{message: msg}
		}, tea.EnterAltScreen)
	case ActionViewLinks:
		return m, m.loadLinksForSpecCmd()
	case ActionDeleteLink:
		return m, m.loadLinksForSpecCmd()
	}
	return m, nil
}

// executeLinkAction executes the action on the selected link
func (m *Model) executeLinkAction() (tea.Model, tea.Cmd) {
	return m, tea.Sequence(tea.ExitAltScreen, func() tea.Msg {
		msg := m.deleteSelectedLink()
		return operationCompleteMsg{message: msg}
	}, tea.EnterAltScreen)
}

// loadSpecsCmd returns a command to load specs
func (m *Model) loadSpecsCmd() tea.Cmd {
	return func() tea.Msg {
		specs, err := m.app.specService.ListSpecs()
		if err != nil {
			return specsLoadedMsg{err: err}
		}

		specItems := make([]specItem, len(specs))
		for i, spec := range specs {
			specItems[i] = specItem{
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

// formatSpecsList formats the specs list for display
func (m *Model) formatSpecsList() string {
	if len(m.specs) == 0 {
		return "No specifications found. Press Enter to continue..."
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("Found %d specifications:\n\n", len(m.specs)))

	// Simple text formatting instead of tabwriter for message display
	s.WriteString("TITLE                                              CREATED          ID\n")
	s.WriteString("─────                                              ───────          ──\n")

	for _, spec := range m.specs {
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		s.WriteString(fmt.Sprintf("%-50s %-16s %s\n",
			title,
			spec.CreatedAt,
			spec.ID[:8]+"...",
		))
	}

	s.WriteString("\nPress Enter to continue...")
	return s.String()
}

// formatLinksList formats the links list for display
func (m *Model) formatLinksList() string {
	// Find selected spec title
	selectedSpecTitle := ""
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			selectedSpecTitle = spec.Title
			break
		}
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("Links for '%s':\n\n", selectedSpecTitle))

	if len(m.links) == 0 {
		s.WriteString("No links found.")
	} else {
		s.WriteString("COMMIT           REPO             TYPE         CREATED\n")
		s.WriteString("──────           ────             ────         ───────\n")

		for _, link := range m.links {
			repoName := filepath.Base(link.RepoPath)
			s.WriteString(fmt.Sprintf("%-16s %-16s %-12s %s\n",
				link.CommitID[:12]+"...",
				repoName,
				link.LinkType,
				link.CreatedAt,
			))
		}
	}

	s.WriteString("\nPress Enter to continue...")
	return s.String()
}

// View renders the UI
func (m *Model) View() string {
	if m.showMessage {
		return m.message
	}

	switch m.state {
	case MainMenu:
		return m.renderMainMenu()
	case SpecSelection:
		return m.renderSpecSelection()
	case LinkSelection:
		return m.renderLinkSelection()
	default:
		return "Loading..."
	}
}

// renderMainMenu renders the main menu
func (m *Model) renderMainMenu() string {
	s := "🚀 ZAMM Interactive Mode\n"
	s += "========================\n\n"
	s += "What would you like to do?\n\n"

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to select, q to quit"
	return s
}

// renderSpecSelection renders the spec selection screen
func (m *Model) renderSpecSelection() string {
	actionTitle := map[MenuAction]string{
		ActionEditSpec:   "📝 Edit Specification",
		ActionDeleteSpec: "🗑️  Delete Specification",
		ActionLinkSpec:   "🔗 Link Specification to Commit",
		ActionViewLinks:  "👀 View Specification Links",
		ActionDeleteLink: "🗑️  Delete Specification Link",
	}

	s := actionTitle[m.action] + "\n"
	s += strings.Repeat("=", len(actionTitle[m.action])-3) + "\n\n" // -3 for emoji

	if len(m.specs) == 0 {
		s += "No specifications found.\n\n"
		s += "Press Esc to return to main menu"
		return s
	}

	s += "Select a specification:\n\n"

	for i, spec := range m.specs {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		s += fmt.Sprintf("%s %s (%s)\n", cursor, title, spec.CreatedAt)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to select, Esc to go back"
	return s
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

// Terminal interaction functions (executed outside of TUI)

func (m *Model) interactiveCreateSpec() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("📝 Create New Specification")
	fmt.Println("===========================")

	fmt.Print("Enter title: ")
	title, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	title = strings.TrimSpace(title)

	fmt.Print("Enter content (end with empty line): ")
	var contentLines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			break
		}
		contentLines = append(contentLines, line)
	}
	content := strings.Join(contentLines, "\n")

	spec, err := m.app.specService.CreateSpec(title, content)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("✅ Created specification: %s\n", spec.Title)
		fmt.Printf("   ID: %s\n", spec.ID)
	}

	fmt.Println("\nPress Enter to continue...")
	reader.ReadString('\n')
}

func (m *Model) editSelectedSpec() string {
	reader := bufio.NewReader(os.Stdin)

	// Find the selected spec
	var selectedSpec *specItem
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			selectedSpec = &spec
			break
		}
	}

	if selectedSpec == nil {
		fmt.Println("Error: Selected specification not found")
		fmt.Println("Press Enter to continue...")
		reader.ReadString('\n')
		return "Error: Selected specification not found. Press Enter to continue..."
	}

	fmt.Printf("Editing: %s\n", selectedSpec.Title)
	fmt.Printf("Current content:\n%s\n\n", selectedSpec.Content)

	fmt.Printf("Enter new title (or press Enter to keep '%s'): ", selectedSpec.Title)
	titleInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	newTitle := strings.TrimSpace(titleInput)
	if newTitle == "" {
		newTitle = selectedSpec.Title
	}

	fmt.Print("Enter new content (end with empty line, or press Enter twice to keep existing): ")
	var contentLines []string
	emptyLineCount := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			emptyLineCount++
			if emptyLineCount >= 2 {
				break
			}
		} else {
			emptyLineCount = 0
		}
		contentLines = append(contentLines, line)
	}

	newContent := strings.Join(contentLines, "\n")
	if strings.TrimSpace(newContent) == "" {
		newContent = selectedSpec.Content
	}

	updatedSpec, err := m.app.specService.UpdateSpec(selectedSpec.ID, newTitle, newContent)
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	return fmt.Sprintf("✅ Updated specification: %s. Press Enter to continue...", updatedSpec.Title)
}

func (m *Model) deleteSelectedSpec() string {
	reader := bufio.NewReader(os.Stdin)

	// Find the selected spec
	var selectedSpec *specItem
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			selectedSpec = &spec
			break
		}
	}

	if selectedSpec == nil {
		fmt.Println("Error: Selected specification not found")
		fmt.Println("Press Enter to continue...")
		reader.ReadString('\n')
		return "Error: Selected specification not found. Press Enter to continue..."
	}

	fmt.Printf("⚠️  Are you sure you want to delete '%s'? (y/N): ", selectedSpec.Title)
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		return "Deletion cancelled. Press Enter to continue..."
	}

	if err := m.app.specService.DeleteSpec(selectedSpec.ID); err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	return fmt.Sprintf("✅ Deleted specification: %s. Press Enter to continue...", selectedSpec.Title)
}

func (m *Model) linkSelectedSpec() string {
	reader := bufio.NewReader(os.Stdin)

	// Find the selected spec
	var selectedSpec *specItem
	for _, spec := range m.specs {
		if spec.ID == m.selectedSpecID {
			selectedSpec = &spec
			break
		}
	}

	if selectedSpec == nil {
		fmt.Println("Error: Selected specification not found")
		fmt.Println("Press Enter to continue...")
		reader.ReadString('\n')
		return "Error: Selected specification not found. Press Enter to continue..."
	}

	fmt.Printf("Linking specification: %s\n\n", selectedSpec.Title)

	fmt.Print("Enter commit hash: ")
	commitInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}
	commitID := strings.TrimSpace(commitInput)

	fmt.Print("Enter repository path (or press Enter for current directory): ")
	repoInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}
	repoPath := strings.TrimSpace(repoInput)
	if repoPath == "" {
		repoPath = m.app.config.Git.DefaultRepo
	}

	fmt.Print("Enter link type (implements/references, default: implements): ")
	typeInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}
	linkType := strings.TrimSpace(typeInput)
	if linkType == "" {
		linkType = "implements"
	}

	link, err := m.app.linkService.LinkSpecToCommit(selectedSpec.ID, commitID, repoPath, linkType)
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	return fmt.Sprintf("✅ Created link between '%s' and commit %s (ID: %s). Press Enter to continue...",
		selectedSpec.Title, commitID[:12]+"...", link.ID)
}

func (m *Model) deleteSelectedLink() string {
	reader := bufio.NewReader(os.Stdin)

	if m.cursor >= len(m.links) {
		fmt.Println("Error: Invalid link selection")
		fmt.Println("Press Enter to continue...")
		reader.ReadString('\n')
		return "Error: Invalid link selection. Press Enter to continue..."
	}

	selectedLink := m.links[m.cursor]

	fmt.Printf("⚠️  Are you sure you want to delete the link to commit %s? (y/N): ", selectedLink.CommitID[:12]+"...")
	confirm, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		return "Deletion cancelled. Press Enter to continue..."
	}

	if err := m.app.linkService.UnlinkSpecFromCommit(m.selectedSpecID, selectedLink.CommitID, selectedLink.RepoPath); err != nil {
		return fmt.Sprintf("Error: %v. Press Enter to continue...", err)
	}

	return fmt.Sprintf("✅ Deleted link to commit %s. Press Enter to continue...", selectedLink.CommitID[:12]+"...")
}
