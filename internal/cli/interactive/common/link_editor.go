package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/services"
)

// LinkEditorMode represents the current state of the link editor
type LinkEditorMode int

const (
	LinkTypeSelection LinkEditorMode = iota
	LinkGitCommitForm
	ChildSpecSelection
	ParentSpecSelection
	ChildSpecLinkTypeSelection
	ParentSpecLinkTypeSelection
	// Unlink modes
	UnlinkTypeSelection
	GitCommitLinkSelection
	ChildSpecLinkSelection
	ParentSpecLinkSelection
)

// LinkEditorConfig configures the behavior of the link editor
type LinkEditorConfig struct {
	Title            string // Title shown to user
	DefaultRepo      string // Default repository path for git commits
	CurrentSpecID    string // ID of the spec being linked
	CurrentSpecTitle string // Title of the spec being linked
	IsUnlinkMode     bool   // Whether this is for unlinking (true) or linking (false)
}

// LinkEditorCompleteMsg is sent when link operation is complete
type LinkEditorCompleteMsg struct{}

// LinkEditorCancelMsg is sent when user cancels the link editor
type LinkEditorCancelMsg struct{}

// LinkEditorErrorMsg is sent when an error occurs
type LinkEditorErrorMsg struct {
	Error string
}

// SpecsLoadedMsg is sent when specs are loaded asynchronously
// Used to trigger a re-render after async loadAvailableSpecs
type SpecsLoadedMsg struct {
	Specs []interactive.Spec
}

// GitCommitLinksLoadedMsg is sent when git commit links are loaded asynchronously
type GitCommitLinksLoadedMsg struct {
	Links []linkItem
}

// LinkEditor is a component that manages the entire link creation flow
type LinkEditor struct {
	config        LinkEditorConfig
	mode          LinkEditorMode
	linkSelector  LinkTypeSelector
	gitCommitForm GitCommitForm
	specSelector  SpecSelector
	textInput     textinput.Model
	promptText    string

	// State tracking
	selectedLinkType  LinkType
	selectedSpecID    string
	selectedSpecTitle string
	inputLinkLabel    string

	// Services
	linkService services.LinkService
	specService services.SpecService

	// For unlink operations
	gitCommitLinks []linkItem
	cursor         int

	// Screen dimensions
	width  int
	height int
}

type linkItem struct {
	CommitID  string
	RepoPath  string
	LinkLabel string
}

// NewLinkEditor creates a new link editor component
func NewLinkEditor(config LinkEditorConfig, linkService services.LinkService, specService services.SpecService) LinkEditor {
	// Initialize link type selector with appropriate title
	var title string
	if config.IsUnlinkMode {
		title = "Select link type to remove:"
	} else {
		title = "Select link type to add:"
	}
	linkSelector := NewLinkTypeSelector(title)

	// Initialize spec selector
	specSelector := NewSpecSelector(SpecSelectorConfig{
		Title: "🔗 Choose spec to link to",
	})

	// Initialize text input for link type
	textInput := textinput.New()
	textInput.Placeholder = "Enter link type (or press Enter for 'child')"

	// Always initialize git commit form (will be configured when needed)
	gitCommitFormConfig := GitCommitFormConfig{
		InitialCommit:   "",
		InitialRepo:     config.DefaultRepo,
		InitialLinkType: "implements",
	}
	gitCommitForm := NewGitCommitForm(gitCommitFormConfig)

	// Set initial mode based on config
	initialMode := LinkTypeSelection
	if config.IsUnlinkMode {
		initialMode = UnlinkTypeSelection
	}

	return LinkEditor{
		config:        config,
		mode:          initialMode,
		linkSelector:  linkSelector,
		gitCommitForm: gitCommitForm,
		specSelector:  specSelector,
		textInput:     textInput,
		linkService:   linkService,
		specService:   specService,
	}
}

// SetSize sets the dimensions of the link editor
func (l *LinkEditor) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.linkSelector.SetSize(width, height-3)
	// somehow this needs to be 1 less than the usual height
	l.specSelector.SetSize(width, height-4)
	l.gitCommitForm.SetSize(width, height-3)
}

// loadSpecsExceptCurrent loads all specs except the current one for selection
func (l *LinkEditor) loadSpecsExceptCurrent() tea.Cmd {
	return func() tea.Msg {
		specs, err := l.specService.ListSpecs()
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error loading specs: %v", err)}
		}

		// Filter out the current spec
		filteredSpecs := make([]interactive.Spec, 0, len(specs))
		for _, spec := range specs {
			if spec.ID != l.config.CurrentSpecID {
				filteredSpecs = append(filteredSpecs, interactive.Spec{
					ID:      spec.ID,
					Title:   spec.Title,
					Content: spec.Content,
				})
			}
		}

		return SpecsLoadedMsg{Specs: filteredSpecs}
	}
}

// loadChildSpecs loads child specs that can be unlinked
func (l *LinkEditor) loadChildSpecs() tea.Cmd {
	return func() tea.Msg {
		linkedSpecs, err := l.specService.GetChildren(l.config.CurrentSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error loading linked specs: %v", err)}
		}

		specs := make([]interactive.Spec, 0, len(linkedSpecs))
		for _, spec := range linkedSpecs {
			specs = append(specs, interactive.Spec{
				ID:      spec.ID,
				Title:   spec.Title,
				Content: spec.Content,
			})
		}

		return SpecsLoadedMsg{Specs: specs}
	}
}

// loadParentSpecs loads parent specs that can be unlinked
func (l *LinkEditor) loadParentSpecs() tea.Cmd {
	return func() tea.Msg {
		linkedSpecs, err := l.specService.GetParents(l.config.CurrentSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error loading linked parent specs: %v", err)}
		}

		specs := make([]interactive.Spec, 0, len(linkedSpecs))
		for _, spec := range linkedSpecs {
			specs = append(specs, interactive.Spec{
				ID:      spec.ID,
				Title:   spec.Title,
				Content: spec.Content,
			})
		}

		return SpecsLoadedMsg{Specs: specs}
	}
}

// loadGitCommitLinks loads git commit links for the spec
func (l *LinkEditor) loadGitCommitLinks() tea.Cmd {
	return func() tea.Msg {
		links, err := l.linkService.GetCommitsForSpec(l.config.CurrentSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error loading git commit links: %v", err)}
		}

		linkItems := make([]linkItem, len(links))
		for i, link := range links {
			linkItems[i] = linkItem{
				CommitID:  link.CommitID,
				RepoPath:  link.RepoPath,
				LinkLabel: link.LinkLabel,
			}
		}

		return GitCommitLinksLoadedMsg{Links: linkItems}
	}
}

// Init initializes the link editor
func (l LinkEditor) Init() tea.Cmd {
	return nil
}

// Update handles tea messages and updates the component
func (l LinkEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.SetSize(msg.Width, msg.Height)
		return l, nil
	case tea.KeyMsg:
		switch l.mode {
		case LinkTypeSelection:
			selector, cmd := l.linkSelector.Update(msg)
			l.linkSelector = *selector
			return l, cmd
		case LinkGitCommitForm:
			form, cmd := l.gitCommitForm.Update(msg)
			l.gitCommitForm = *form
			return l, cmd
		case ChildSpecSelection:
			return l.updateSpecSelection(msg)
		case ParentSpecSelection:
			return l.updateSpecSelection(msg)
		case ChildSpecLinkTypeSelection:
			return l.updateChildSpecLinkTypeInput(msg)
		case ParentSpecLinkTypeSelection:
			return l.updateParentSpecLinkTypeInput(msg)
		case UnlinkTypeSelection:
			selector, cmd := l.linkSelector.Update(msg)
			l.linkSelector = *selector
			return l, cmd
		case GitCommitLinkSelection:
			return l.updateGitCommitLinkSelection(msg)
		case ChildSpecLinkSelection:
			return l.updateSpecSelection(msg)
		case ParentSpecLinkSelection:
			return l.updateSpecSelection(msg)
		}

	case LinkOptionSelectedMsg:
		return l.handleLinkOptionSelected(msg)
	case LinkTypeCancelledMsg:
		return l, func() tea.Msg {
			return LinkEditorCancelMsg{}
		}
	case GitCommitFormCompleteMsg:
		return l.handleGitCommitFormComplete(msg)
	case GitCommitFormCancelMsg:
		return l, func() tea.Msg {
			return LinkEditorCancelMsg{}
		}
	case SpecSelectedMsg:
		return l.handleSpecSelected(msg)
	case LinkEditorErrorMsg:
		// Return error message to parent
		return l, func() tea.Msg {
			return LinkEditorErrorMsg{Error: msg.Error}
		}
	case SpecsLoadedMsg:
		// Set the loaded specs directly to the selector
		l.specSelector.SetSpecs(msg.Specs)
		return l, nil
	case GitCommitLinksLoadedMsg:
		// Set the loaded git commit links
		l.gitCommitLinks = msg.Links
		return l, nil
	}

	switch l.mode {
	case ChildSpecSelection, ParentSpecSelection, ChildSpecLinkSelection, ParentSpecLinkSelection:
		selector, cmd := l.specSelector.Update(msg)
		l.specSelector = *selector
		return l, cmd
	}

	return l, nil
}

// getEscapeMode returns the mode to transition to when escape is pressed
func (l LinkEditor) getEscapeMode() LinkEditorMode {
	switch l.mode {
	case ChildSpecSelection, ParentSpecSelection:
		return LinkTypeSelection
	case ChildSpecLinkSelection, ParentSpecLinkSelection:
		return UnlinkTypeSelection
	default:
		return LinkTypeSelection // fallback
	}
}

// updateSpecSelection handles updates for all spec selection modes
func (l LinkEditor) updateSpecSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape key to go back
	if msg.String() == "esc" {
		l.mode = l.getEscapeMode()
		return l, nil
	}

	selector, cmd := l.specSelector.Update(msg)
	l.specSelector = *selector
	return l, cmd
}

// updateChildSpecLinkTypeInput handles updates for child spec link type input
func (l LinkEditor) updateChildSpecLinkTypeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		l.mode = ChildSpecSelection
		l.resetInputs()
		return l, nil
	case tea.KeyEnter:
		label := strings.TrimSpace(l.textInput.Value())
		if label == "" {
			label = "child"
		}
		return l, l.createChildSpecLink(label)
	}

	l.textInput, _ = l.textInput.Update(msg)
	return l, nil
}

// updateParentSpecLinkTypeInput handles updates for parent spec link type input
func (l LinkEditor) updateParentSpecLinkTypeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		l.mode = ParentSpecSelection
		l.resetInputs()
		return l, nil
	case tea.KeyEnter:
		label := strings.TrimSpace(l.textInput.Value())
		if label == "" {
			label = "child"
		}
		return l, l.createParentSpecLink(label)
	}

	l.textInput, _ = l.textInput.Update(msg)
	return l, nil
}

// updateGitCommitLinkSelection handles updates for git commit link selection
func (l LinkEditor) updateGitCommitLinkSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape key to go back
	if msg.String() == "esc" {
		l.mode = UnlinkTypeSelection
		return l, nil
	}

	// Handle navigation
	switch msg.String() {
	case "up", "k":
		if l.cursor > 0 {
			l.cursor--
		}
	case "down", "j":
		if l.cursor < len(l.gitCommitLinks)-1 {
			l.cursor++
		}
	case "enter", " ":
		if len(l.gitCommitLinks) > 0 && l.cursor < len(l.gitCommitLinks) {
			selectedLink := l.gitCommitLinks[l.cursor]
			return l, l.removeGitCommitLink(selectedLink.CommitID, selectedLink.RepoPath)
		}
	}

	return l, nil
}

// handleLinkOptionSelected handles when a link option is selected
func (l LinkEditor) handleLinkOptionSelected(msg LinkOptionSelectedMsg) (tea.Model, tea.Cmd) {
	l.selectedLinkType = msg.LinkType

	if l.config.IsUnlinkMode {
		// Handle unlink mode
		switch msg.LinkType {
		case GitCommitLink:
			// Load git commit links and show selection
			l.mode = GitCommitLinkSelection
			return l, l.loadGitCommitLinks()
		case ChildSpecLink:
			// Show spec selector for unlinking child specs
			l.mode = ChildSpecLinkSelection
			return l, l.loadChildSpecs()
		case ParentSpecLink:
			// Show spec selector for unlinking parent specs
			l.mode = ParentSpecLinkSelection
			return l, l.loadParentSpecs()
		}
	} else {
		// Handle link mode
		switch msg.LinkType {
		case GitCommitLink:
			// Reset git commit form with fresh values
			config := GitCommitFormConfig{
				InitialCommit:   "",
				InitialRepo:     l.config.DefaultRepo,
				InitialLinkType: "implements",
			}
			l.gitCommitForm = NewGitCommitForm(config)
			l.mode = LinkGitCommitForm
			return l, nil
		case ChildSpecLink:
			// Show spec selector for adding child specs
			l.mode = ChildSpecSelection
			return l, l.loadSpecsExceptCurrent()
		case ParentSpecLink:
			// Show spec selector for adding parent specs
			l.mode = ParentSpecSelection
			return l, l.loadSpecsExceptCurrent()
		}
	}

	return l, nil
}

// handleGitCommitFormComplete handles when git commit form is completed
func (l LinkEditor) handleGitCommitFormComplete(msg GitCommitFormCompleteMsg) (tea.Model, tea.Cmd) {
	return l, l.createGitCommitLink(msg.CommitHash, msg.RepoPath, msg.LinkType)
}

// handleSpecSelected handles when a spec is selected
func (l LinkEditor) handleSpecSelected(msg SpecSelectedMsg) (tea.Model, tea.Cmd) {
	if msg.Spec.ID != "" && msg.Spec.ID != l.config.CurrentSpecID {
		if l.config.IsUnlinkMode {
			// For unlink mode, directly remove the link based on current mode
			switch l.mode {
			case ChildSpecLinkSelection:
				return l, l.removeChildSpecLink(msg.Spec.ID)
			case ParentSpecLinkSelection:
				return l, l.removeParentSpecLink(msg.Spec.ID)
			default:
				// Fallback to old behavior for compatibility
				return l, l.removeSpecLink(msg.Spec.ID)
			}
		} else {
			// For link mode, show link type input based on current mode
			l.selectedSpecID = msg.Spec.ID
			l.selectedSpecTitle = msg.Spec.Title
			switch l.mode {
			case ChildSpecSelection:
				l.mode = ChildSpecLinkTypeSelection
				l.promptText = "Enter link type (or press Enter for 'child'):"
			case ParentSpecSelection:
				l.mode = ParentSpecLinkTypeSelection
				l.promptText = "Enter link type (or press Enter for 'child'):"
			default:
				// Fallback to old behavior for compatibility
				l.mode = ChildSpecLinkTypeSelection
				l.promptText = "Enter link type (or press Enter for 'child'):"
			}
			l.textInput.Focus()
			return l, nil
		}
	}
	return l, nil
}

// createGitCommitLink creates a git commit link
func (l LinkEditor) createGitCommitLink(commitHash, repoPath, linkType string) tea.Cmd {
	return func() tea.Msg {
		_, err := l.linkService.LinkSpecToCommit(l.config.CurrentSpecID, commitHash, repoPath, linkType)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error creating git commit link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// removeGitCommitLink removes a git commit link
func (l LinkEditor) removeGitCommitLink(commitID, repoPath string) tea.Cmd {
	return func() tea.Msg {
		err := l.linkService.UnlinkSpecFromCommit(l.config.CurrentSpecID, commitID, repoPath)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error removing git commit link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// createChildSpecLink creates a child spec link
func (l LinkEditor) createChildSpecLink(linkType string) tea.Cmd {
	return func() tea.Msg {
		_, err := l.specService.AddChildToParent(l.selectedSpecID, l.config.CurrentSpecID, linkType)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error creating child spec link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// createParentSpecLink creates a parent spec link
func (l LinkEditor) createParentSpecLink(linkType string) tea.Cmd {
	return func() tea.Msg {
		// For parent links, the selected spec is the child, and we're adding a parent
		_, err := l.specService.AddChildToParent(l.config.CurrentSpecID, l.selectedSpecID, linkType)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error creating parent spec link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// removeSpecLink removes a spec-to-spec link
func (l LinkEditor) removeSpecLink(targetSpecID string) tea.Cmd {
	return func() tea.Msg {
		err := l.specService.RemoveChildFromParent(targetSpecID, l.config.CurrentSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error removing spec link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// removeChildSpecLink removes a child spec link
func (l LinkEditor) removeChildSpecLink(targetSpecID string) tea.Cmd {
	return func() tea.Msg {
		err := l.specService.RemoveChildFromParent(targetSpecID, l.config.CurrentSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error removing child spec link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// removeParentSpecLink removes a parent spec link
func (l LinkEditor) removeParentSpecLink(targetSpecID string) tea.Cmd {
	return func() tea.Msg {
		// For parent links, the current spec is the child, and we're removing a parent
		err := l.specService.RemoveChildFromParent(l.config.CurrentSpecID, targetSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error removing parent spec link: %v", err)}
		}

		return LinkEditorCompleteMsg{}
	}
}

// resetInputs clears all input fields
func (l LinkEditor) resetInputs() {
	l.selectedSpecID = ""
	l.selectedSpecTitle = ""
	l.inputLinkLabel = ""
	l.textInput.Reset()
	l.textInput.Blur()
}

// renderHeader renders the consistent spec title header
func (l LinkEditor) renderHeader() string {
	header := l.config.CurrentSpecTitle
	// Use full width for underline, defaulting to header length if width not set
	underlineWidth := l.width
	if underlineWidth == 0 {
		underlineWidth = len(header)
	}
	underline := strings.Repeat("=", underlineWidth)
	return header + "\n" + underline + "\n"
}

// View renders the link editor
func (l LinkEditor) View() string {
	// Always show the spec title header first
	header := l.renderHeader()

	// Then render the appropriate child component
	var childContent string
	switch l.mode {
	case LinkTypeSelection, UnlinkTypeSelection:
		childContent = l.linkSelector.View()
	case LinkGitCommitForm:
		childContent = l.gitCommitForm.View()
	case ChildSpecSelection, ParentSpecSelection, ChildSpecLinkSelection, ParentSpecLinkSelection:
		childContent = l.specSelector.View()
	case ChildSpecLinkTypeSelection:
		childContent = l.renderChildSpecLinkTypeSelection()
	case ParentSpecLinkTypeSelection:
		childContent = l.renderParentSpecLinkTypeSelection()
	case GitCommitLinkSelection:
		childContent = l.renderGitCommitLinkSelection()
	default:
		childContent = "Loading..."
	}

	return lipgloss.JoinVertical(lipgloss.Top, header, childContent)
}

// renderChildSpecLinkTypeSelection renders the child spec link type input form
func (l LinkEditor) renderChildSpecLinkTypeSelection() string {
	targetSpecTitle := l.selectedSpecTitle

	s := fmt.Sprintf("Adding '%s' as child of '%s'\n\n", targetSpecTitle, l.config.CurrentSpecTitle)
	s += l.promptText + "\n"
	s += l.textInput.View() + "\n\n"
	s += "Press Enter to finish, Esc to cancel"

	s = lipgloss.NewStyle().Width(l.width).Render(s)

	return s
}

// renderParentSpecLinkTypeSelection renders the parent spec link type input form
func (l LinkEditor) renderParentSpecLinkTypeSelection() string {
	targetSpecTitle := l.selectedSpecTitle

	s := fmt.Sprintf("Adding '%s' as parent of '%s'\n\n", targetSpecTitle, l.config.CurrentSpecTitle)
	s += l.promptText + "\n"
	s += l.textInput.View() + "\n\n"
	s += "Press Enter to finish, Esc to cancel"

	s = lipgloss.NewStyle().Width(l.width).Render(s)

	return s
}

// renderGitCommitLinkSelection renders the git commit link selection screen
func (l LinkEditor) renderGitCommitLinkSelection() string {
	if len(l.gitCommitLinks) == 0 {
		s := "No git commit links found for this specification.\n\n"
		s += "Press Esc to return to main menu"
		return s
	}

	s := fmt.Sprintf("Git commit links for '%s':\n\n", l.config.CurrentSpecTitle)

	for i, link := range l.gitCommitLinks {
		cursor := " "
		if l.cursor == i {
			cursor = ">"
		}
		repoName := link.RepoPath
		if len(repoName) > 20 {
			repoName = repoName[:17] + "..."
		}
		s += fmt.Sprintf("%s %s (%s, %s)\n", cursor, link.CommitID[:12]+"...", repoName, link.LinkLabel)
	}

	s += "\nUse ↑/↓ arrows to navigate, Enter to delete, Esc to go back"
	return s
}
