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
	SpecSelection
	SpecLinkTypeSelection
	// Unlink modes
	UnlinkTypeSelection
	GitCommitLinkSelection
	SpecLinkSelection
)

// LinkEditorConfig configures the behavior of the link editor
type LinkEditorConfig struct {
	Title             string // Title shown to user
	DefaultRepo       string // Default repository path for git commits
	SelectedSpecID    string // ID of the spec being linked
	SelectedSpecTitle string // Title of the spec being linked
	IsUnlinkMode      bool   // Whether this is for unlinking (true) or linking (false)
}

// LinkEditorCompleteMsg is sent when link operation is complete
type LinkEditorCompleteMsg struct {
	Operation string // "create" or "remove"
	LinkType  string // "git_commit" or "spec"
	// For git commit links
	CommitHash  string
	CommitID    string
	RepoPath    string
	GitLinkType string
	// For spec links
	TargetSpecID    string
	TargetSpecTitle string
	SpecLinkType    string
}

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
	selectedLinkType    LinkType
	selectedChildSpecID string
	inputLinkLabel      string

	// Services
	linkService services.LinkService
	specService services.SpecService

	// Available specs for selection
	availableSpecs []interactive.Spec

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

// loadAvailableSpecs loads all specs except the current one for selection
func (l *LinkEditor) loadAvailableSpecs() tea.Cmd {
	return func() tea.Msg {
		specs, err := l.specService.ListSpecs()
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error loading specs: %v", err)}
		}

		// Filter out the current spec
		filteredSpecs := make([]interactive.Spec, 0, len(specs))
		for _, spec := range specs {
			if spec.ID != l.config.SelectedSpecID {
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
		linkedSpecs, err := l.specService.GetChildren(l.config.SelectedSpecID)
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

		l.availableSpecs = specs
		return nil
	}
}

// loadGitCommitLinks loads git commit links for the spec
func (l *LinkEditor) loadGitCommitLinks() tea.Cmd {
	return func() tea.Msg {
		links, err := l.linkService.GetCommitsForSpec(l.config.SelectedSpecID)
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

		l.gitCommitLinks = linkItems
		return nil
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
		case SpecSelection:
			return l.updateSpecSelection(msg)
		case SpecLinkTypeSelection:
			return l.updateLinkTypeInput(msg)
		case UnlinkTypeSelection:
			selector, cmd := l.linkSelector.Update(msg)
			l.linkSelector = *selector
			if cmd != nil {
				return l, cmd
			}
			return l, nil
		case GitCommitLinkSelection:
			return l.updateGitCommitLinkSelection(msg)
		case SpecLinkSelection:
			return l.updateSpecLinkSelection(msg)
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
		// Set the loaded specs and update the selector
		l.availableSpecs = msg.Specs
		l.specSelector.SetSpecs(l.availableSpecs)
		return l, nil
	}

	switch l.mode {
	case SpecSelection:
		selector, cmd := l.specSelector.Update(msg)
		l.specSelector = *selector
		return l, cmd
	}

	return l, nil
}

// updateSpecSelection handles updates for spec selection
func (l LinkEditor) updateSpecSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape key to go back
	if msg.String() == "esc" {
		l.mode = LinkTypeSelection
		return l, nil
	}

	selector, cmd := l.specSelector.Update(msg)
	l.specSelector = *selector
	return l, cmd
}

// updateLinkTypeInput handles updates for link type input
func (l LinkEditor) updateLinkTypeInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		l.mode = SpecSelection
		l.resetInputs()
		return l, nil
	case tea.KeyEnter:
		label := strings.TrimSpace(l.textInput.Value())
		if label == "" {
			label = "child"
		}
		return l, l.createSpecLink(label)
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

// updateSpecLinkSelection handles updates for spec link selection
func (l LinkEditor) updateSpecLinkSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape key to go back
	if msg.String() == "esc" {
		l.mode = UnlinkTypeSelection
		return l, nil
	}

	selector, cmd := l.specSelector.Update(msg)
	l.specSelector = *selector
	return l, cmd
}

// handleLinkOptionSelected handles when a link option is selected
func (l LinkEditor) handleLinkOptionSelected(msg LinkOptionSelectedMsg) (tea.Model, tea.Cmd) {
	l.selectedLinkType = msg.LinkType

	if l.config.IsUnlinkMode {
		// Handle unlink mode
		if msg.LinkType == GitCommitLink {
			// Load git commit links and show selection
			l.mode = GitCommitLinkSelection
			return l, l.loadGitCommitLinks()
		} else if msg.LinkType == SpecLink {
			// Show spec selector for unlinking
			l.specSelector.SetSpecs(l.availableSpecs)
			l.mode = SpecLinkSelection
			return l, l.loadChildSpecs()
		}
	} else {
		// Handle link mode
		if msg.LinkType == GitCommitLink {
			// Reset git commit form with fresh values
			config := GitCommitFormConfig{
				InitialCommit:   "",
				InitialRepo:     l.config.DefaultRepo,
				InitialLinkType: "implements",
			}
			l.gitCommitForm = NewGitCommitForm(config)
			l.mode = LinkGitCommitForm
			return l, nil
		} else if msg.LinkType == SpecLink {
			// Show spec selector
			l.specSelector.SetSpecs(l.availableSpecs)
			l.mode = SpecSelection
			return l, l.loadAvailableSpecs()
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
	if msg.Spec.ID != "" && msg.Spec.ID != l.config.SelectedSpecID {
		if l.config.IsUnlinkMode {
			// For unlink mode, directly remove the link
			return l, l.removeSpecLink(msg.Spec.ID)
		} else {
			// For link mode, show link type input
			l.selectedChildSpecID = msg.Spec.ID
			l.mode = SpecLinkTypeSelection
			l.promptText = "Enter link type (or press Enter for 'child'):"
			l.textInput.Focus()
			return l, nil
		}
	}
	return l, nil
}

// createGitCommitLink creates a git commit link
func (l LinkEditor) createGitCommitLink(commitHash, repoPath, linkType string) tea.Cmd {
	return func() tea.Msg {
		_, err := l.linkService.LinkSpecToCommit(l.config.SelectedSpecID, commitHash, repoPath, linkType)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error creating git commit link: %v", err)}
		}

		return LinkEditorCompleteMsg{
			Operation:   "create",
			LinkType:    "git_commit",
			CommitHash:  commitHash,
			RepoPath:    repoPath,
			GitLinkType: linkType,
		}
	}
}

// removeGitCommitLink removes a git commit link
func (l LinkEditor) removeGitCommitLink(commitID, repoPath string) tea.Cmd {
	return func() tea.Msg {
		err := l.linkService.UnlinkSpecFromCommit(l.config.SelectedSpecID, commitID, repoPath)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error removing git commit link: %v", err)}
		}

		return LinkEditorCompleteMsg{
			Operation: "remove",
			LinkType:  "git_commit",
			CommitID:  commitID,
			RepoPath:  repoPath,
		}
	}
}

// createSpecLink creates a spec-to-spec link
func (l LinkEditor) createSpecLink(linkType string) tea.Cmd {
	return func() tea.Msg {
		_, err := l.specService.AddChildToParent(l.selectedChildSpecID, l.config.SelectedSpecID, linkType)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error creating spec link: %v", err)}
		}

		// Find target spec title for display
		var targetSpecTitle string
		for _, spec := range l.availableSpecs {
			if spec.ID == l.selectedChildSpecID {
				targetSpecTitle = spec.Title
				break
			}
		}

		return LinkEditorCompleteMsg{
			Operation:       "create",
			LinkType:        "spec",
			TargetSpecID:    l.selectedChildSpecID,
			TargetSpecTitle: targetSpecTitle,
			SpecLinkType:    linkType,
		}
	}
}

// removeSpecLink removes a spec-to-spec link
func (l LinkEditor) removeSpecLink(targetSpecID string) tea.Cmd {
	return func() tea.Msg {
		err := l.specService.RemoveChildFromParent(targetSpecID, l.config.SelectedSpecID)
		if err != nil {
			return LinkEditorErrorMsg{Error: fmt.Sprintf("Error removing spec link: %v", err)}
		}

		// Find target spec title for display
		var targetSpecTitle string
		for _, spec := range l.availableSpecs {
			if spec.ID == targetSpecID {
				targetSpecTitle = spec.Title
				break
			}
		}

		return LinkEditorCompleteMsg{
			Operation:       "remove",
			LinkType:        "spec",
			TargetSpecID:    targetSpecID,
			TargetSpecTitle: targetSpecTitle,
		}
	}
}

// resetInputs clears all input fields
func (l LinkEditor) resetInputs() {
	l.selectedChildSpecID = ""
	l.inputLinkLabel = ""
	l.textInput.Reset()
	l.textInput.Blur()
}

// renderHeader renders the consistent spec title header
func (l LinkEditor) renderHeader() string {
	header := l.config.SelectedSpecTitle
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
	case SpecSelection, SpecLinkSelection:
		childContent = l.specSelector.View()
	case SpecLinkTypeSelection:
		childContent = l.renderSpecLinkTypeSelection()
	case GitCommitLinkSelection:
		childContent = l.renderGitCommitLinkSelection()
	default:
		childContent = "Loading..."
	}

	return lipgloss.JoinVertical(lipgloss.Top, header, childContent)
}

// renderSpecLinkTypeSelection renders the link type input form
func (l LinkEditor) renderSpecLinkTypeSelection() string {
	// Find spec titles
	var targetSpecTitle string
	for _, spec := range l.availableSpecs {
		if spec.ID == l.selectedChildSpecID {
			targetSpecTitle = spec.Title
			break
		}
	}

	s := fmt.Sprintf("Linking '%s' to '%s'\n\n", l.config.SelectedSpecTitle, targetSpecTitle)
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

	s := fmt.Sprintf("Git commit links for '%s':\n\n", l.config.SelectedSpecTitle)

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
