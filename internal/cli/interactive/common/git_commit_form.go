package common

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const gitHashLabel = "Commit Hash: "
const repoPathLabel = "Repository: "

// GitCommitFormMode represents the current editing mode
type GitCommitFormMode int

const (
	EditingCommitHash GitCommitFormMode = iota
	EditingRepoPath
	SelectingLinkType
)

// GitCommitFormConfig configures the behavior of the git commit form
type GitCommitFormConfig struct {
	InitialCommit   string // Initial commit hash value
	InitialRepo     string // Initial repository path value (defaults to ".")
	InitialLinkType string // Initial link type value (defaults to "implements")
}

// GitCommitFormCompleteMsg is sent when form is complete
type GitCommitFormCompleteMsg struct {
	CommitHash string
	RepoPath   string
	LinkType   string
}

// GitCommitFormCancelMsg is sent when user cancels the form
type GitCommitFormCancelMsg struct{}

// LinkTypeOption represents a link type option
type LinkTypeOption struct {
	Value string
	Label string
}

// FilterValue returns a value for filtering - implements list.Item
func (o LinkTypeOption) FilterValue() string {
	return o.Label
}

// gitCommitDelegate handles rendering of link type items in the list
type gitCommitDelegate struct {
	isInFocus bool // Whether the list is currently in focus
}

func (d gitCommitDelegate) Height() int                             { return 1 }
func (d gitCommitDelegate) Spacing() int                            { return 0 }
func (d gitCommitDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d gitCommitDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	option, ok := listItem.(LinkTypeOption)
	if !ok {
		return
	}

	if index == m.Index() {
		// Show selector and highlight when this item is selected and list is in focus
		var style lipgloss.Style
		if d.isInFocus {
			style = HighlightStyle()
		} else {
			style = defaultStyle
		}
		fmt.Fprint(w, style.Render("> "+option.Label))
	} else {
		// No selector when item is not selected
		fmt.Fprint(w, defaultStyle.Render("  "+option.Label))
	}
}

// Predefined link type options
var defaultLinkTypeOptions = []list.Item{
	LinkTypeOption{Value: "implements", Label: "Implementation"},
	LinkTypeOption{Value: "fixes", Label: "Fix"},
	LinkTypeOption{Value: "refactors", Label: "Refactor"},
	LinkTypeOption{Value: "documents", Label: "Documentation"},
	LinkTypeOption{Value: "tests", Label: "Test"},
}

// GitCommitForm is a reusable component for collecting git commit information
type GitCommitForm struct {
	config       GitCommitFormConfig
	mode         GitCommitFormMode
	commitInput  textinput.Model
	repoInput    textinput.Model
	linkTypeList list.Model
}

// NewGitCommitForm creates a new git commit form component
func NewGitCommitForm(config GitCommitFormConfig) GitCommitForm {
	// Create commit hash input
	commitInput := textinput.New()
	commitInput.Placeholder = "Enter git commit hash"
	commitInput.SetValue(config.InitialCommit)

	// Create repository path input
	repoInput := textinput.New()
	repoInput.Placeholder = "Enter repository path (default: .)"
	if config.InitialRepo != "" {
		repoInput.SetValue(config.InitialRepo)
	} else {
		repoInput.SetValue(".")
	}

	// Create link type list
	delegate := gitCommitDelegate{}
	// Use reasonable default dimensions instead of (0, 0) to ensure proper initial rendering
	// The actual size will be set later via SetSize(), but this prevents rendering issues
	defaultWidth := 40
	defaultHeight := 2 + len(defaultLinkTypeOptions) // 2 lines for title and spacing
	linkTypeList := list.New(defaultLinkTypeOptions, delegate, defaultWidth, defaultHeight)
	linkTypeList.Title = "Link Type"
	linkTypeList.SetShowHelp(false)
	linkTypeList.SetShowPagination(false)
	linkTypeList.SetShowStatusBar(false)
	linkTypeList.Styles.Title = lipgloss.NewStyle().Bold(true)

	// Set initial selection based on config
	if config.InitialLinkType != "" {
		for i, item := range defaultLinkTypeOptions {
			if option, ok := item.(LinkTypeOption); ok && option.Value == config.InitialLinkType {
				linkTypeList.Select(i)
				break
			}
		}
	}

	form := GitCommitForm{
		config:       config,
		mode:         EditingCommitHash, // Start with commit hash focused
		commitInput:  commitInput,
		repoInput:    repoInput,
		linkTypeList: linkTypeList,
	}

	// Set initial focus state
	form.updateFocus()

	return form
}

// SetSize sets the dimensions of the git commit form
func (g *GitCommitForm) SetSize(width, height int) {
	g.commitInput.Width = width - len(gitHashLabel)
	g.repoInput.Width = width - len(repoPathLabel)

	g.linkTypeList.SetSize(width, 2+len(defaultLinkTypeOptions)) // 2 lines for title and spacing
}

// updateFocus updates the focus and blur states based on the current mode
func (g *GitCommitForm) updateFocus() {
	// First blur everything
	g.commitInput.Blur()
	g.repoInput.Blur()
	g.linkTypeList.SetDelegate(gitCommitDelegate{isInFocus: false})

	// Then focus the appropriate component based on mode
	switch g.mode {
	case EditingCommitHash:
		g.commitInput.Focus()
	case EditingRepoPath:
		g.repoInput.Focus()
	case SelectingLinkType:
		g.linkTypeList.SetDelegate(gitCommitDelegate{isInFocus: true})
	}
}

// Update handles tea messages and updates the component
func (g *GitCommitForm) Update(msg tea.Msg) (*GitCommitForm, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return g, func() tea.Msg {
				return GitCommitFormCancelMsg{}
			}
		case tea.KeyTab:
			// Move to next field
			switch g.mode {
			case EditingCommitHash:
				g.mode = EditingRepoPath
			case EditingRepoPath:
				g.mode = SelectingLinkType
			case SelectingLinkType:
				g.mode = EditingCommitHash
			}
			g.updateFocus()
			return g, nil
		case tea.KeyShiftTab:
			// Move to previous field
			switch g.mode {
			case EditingCommitHash:
				g.mode = SelectingLinkType
			case EditingRepoPath:
				g.mode = EditingCommitHash
			case SelectingLinkType:
				g.mode = EditingRepoPath
			}
			g.updateFocus()
			return g, nil
		case tea.KeyEnter:
			// Submit form when Enter is pressed from any field
			commitHash := strings.TrimSpace(g.commitInput.Value())
			if commitHash == "" {
				return g, nil // Don't submit with empty commit hash
			}

			repoPath := strings.TrimSpace(g.repoInput.Value())
			if repoPath == "" {
				repoPath = "."
			}

			// Get selected link type
			linkType := "implements" // default
			if selectedItem := g.linkTypeList.SelectedItem(); selectedItem != nil {
				if option, ok := selectedItem.(LinkTypeOption); ok {
					linkType = option.Value
				}
			}

			return g, func() tea.Msg {
				return GitCommitFormCompleteMsg{
					CommitHash: commitHash,
					RepoPath:   repoPath,
					LinkType:   linkType,
				}
			}
		}
	}

	// Update the appropriate input field or list
	var cmd tea.Cmd
	switch g.mode {
	case EditingCommitHash:
		g.commitInput, cmd = g.commitInput.Update(msg)
	case EditingRepoPath:
		g.repoInput, cmd = g.repoInput.Update(msg)
	case SelectingLinkType:
		g.linkTypeList, cmd = g.linkTypeList.Update(msg)
	}

	return g, cmd
}

// View renders the git commit form
func (g *GitCommitForm) View() string {
	var sb strings.Builder

	// Commit hash input
	commitStyle := defaultStyle
	if g.mode == EditingCommitHash {
		commitStyle = HighlightStyle()
	}
	sb.WriteString(commitStyle.Render(gitHashLabel) + g.commitInput.View() + "\n")

	// Repository path input
	repoStyle := defaultStyle
	if g.mode == EditingRepoPath {
		repoStyle = HighlightStyle()
	}
	sb.WriteString(repoStyle.Render(repoPathLabel) + g.repoInput.View() + "\n\n")

	// Link type list
	sb.WriteString(g.linkTypeList.View() + "\n\n")

	// Instructions
	sb.WriteString("Press Tab/Shift+Tab to switch fields, Enter to submit, Esc to cancel")

	return sb.String()
}
