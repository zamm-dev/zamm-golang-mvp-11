package common

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	overlay "github.com/rmhubbert/bubbletea-overlay"
)

// SpecEditorMode represents the current editing mode
type SpecEditorMode int

const (
	EditingTitle SpecEditorMode = iota
	EditingContent
)

// SpecEditorConfig configures the behavior of the spec editor
type SpecEditorConfig struct {
	Title          string // Title shown to user (e.g., "Create New Specification" or "Edit Specification")
	InitialTitle   string // Initial title value (empty for new spec)
	InitialContent string // Initial content value (empty for new spec)
}

// SpecEditorCompleteMsg is sent when editing is complete
type SpecEditorCompleteMsg struct {
	Title   string
	Content string
}

// SpecEditorCancelMsg is sent when user cancels editing
type SpecEditorCancelMsg struct{}

// baseSpecEditor contains the core editor logic without overlay management
type baseSpecEditor struct {
	config          SpecEditorConfig
	mode            SpecEditorMode
	titleInput      textinput.Model
	contentTextarea textarea.Model
	width           int
	height          int

	// Change tracking
	initialTitle   string
	initialContent string
	hasChanges     bool
}

// ConfirmationDialog is a simple confirmation dialog component
type ConfirmationDialog struct {
	message string
}

// NewConfirmationDialog creates a new confirmation dialog
func NewConfirmationDialog(message string) *ConfirmationDialog {
	return &ConfirmationDialog{
		message: message,
	}
}

// Init initializes the confirmation dialog
func (c *ConfirmationDialog) Init() tea.Cmd {
	return nil
}

// Update handles tea messages for the confirmation dialog
func (c *ConfirmationDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			return c, func() tea.Msg {
				return SpecEditorCancelMsg{}
			}
		case "n", "N", "esc":
			// Return to editing mode immediately
			return c, func() tea.Msg {
				return ConfirmationDismissMsg{}
			}
		}
	}
	return c, nil
}

// View renders the confirmation dialog
func (c *ConfirmationDialog) View() string {
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2).
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("7"))

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("1"))
	title := titleStyle.Render("⚠️  Unsaved Changes")

	content := c.message + "\n\nPress 'y' to exit without saving, 'n' to continue editing"

	layout := lipgloss.JoinVertical(lipgloss.Left, title, content)
	return dialogStyle.Render(layout)
}

// ConfirmationDismissMsg is sent when user dismisses the confirmation dialog
type ConfirmationDismissMsg struct{}

// newBaseSpecEditor creates a new base spec editor component
func newBaseSpecEditor(config SpecEditorConfig) baseSpecEditor {
	titleInput := textinput.New()
	titleInput.Placeholder = "Enter specification title"
	titleInput.Focus()
	titleInput.SetValue(config.InitialTitle)

	contentTextarea := textarea.New()
	contentTextarea.Placeholder = "Enter specification content..."
	if config.InitialContent != "" {
		contentTextarea.SetValue(config.InitialContent)
	}
	// Ensure textarea is properly initialized
	contentTextarea.Focus()
	contentTextarea.Blur()

	return baseSpecEditor{
		config:          config,
		mode:            EditingTitle, // Start with title focused
		titleInput:      titleInput,
		contentTextarea: contentTextarea,
		initialTitle:    config.InitialTitle,
		initialContent:  config.InitialContent,
		hasChanges:      false,
	}
}

// Init initializes the base spec editor
func (s baseSpecEditor) Init() tea.Cmd {
	return nil
}

// SetSize sets the dimensions of the base spec editor
func (s *baseSpecEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.titleInput.Width = width

	// Store dimensions but don't set textarea size yet to avoid panic
	// We'll set it in the first Update call when it's safe
}

// checkForChanges checks if there are unsaved changes
func (s *baseSpecEditor) checkForChanges() bool {
	currentTitle := strings.TrimSpace(s.titleInput.Value())
	currentContent := strings.TrimSpace(s.contentTextarea.Value())

	return currentTitle != s.initialTitle || currentContent != s.initialContent
}

// Update handles tea messages and updates the base component
func (s baseSpecEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
		return s, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return s, func() tea.Msg {
				return SpecEditorCancelMsg{}
			}
		case tea.KeyTab:
			if s.mode == EditingTitle {
				s.mode = EditingContent
				s.titleInput.Blur()
				s.contentTextarea.Focus()
			} else {
				s.mode = EditingTitle
				s.contentTextarea.Blur()
				s.titleInput.Focus()
			}
			return s, nil
		case tea.KeyShiftTab:
			if s.mode == EditingContent {
				s.mode = EditingTitle
				s.contentTextarea.Blur()
				s.titleInput.Focus()
			} else {
				s.mode = EditingContent
				s.titleInput.Blur()
				s.contentTextarea.Focus()
			}
			return s, nil
		case tea.KeyCtrlS:
			title := strings.TrimSpace(s.titleInput.Value())
			content := strings.TrimSpace(s.contentTextarea.Value())

			if title == "" {
				return s, nil // Don't save with empty title
			}

			return s, func() tea.Msg {
				return SpecEditorCompleteMsg{
					Title:   title,
					Content: content,
				}
			}
		}
	}

	// Update the appropriate input field
	var cmd tea.Cmd
	if s.mode == EditingTitle {
		s.titleInput, cmd = s.titleInput.Update(msg)
	} else if s.mode == EditingContent {
		s.contentTextarea, cmd = s.contentTextarea.Update(msg)
	}

	return s, cmd
}

// View renders the base spec editor
func (s baseSpecEditor) View() string {
	s.contentTextarea.SetWidth(s.width)
	s.contentTextarea.SetHeight(s.height - 8)

	var sb strings.Builder

	// Header
	sb.WriteString(s.config.Title + "\n")
	sb.WriteString(strings.Repeat("=", len(s.config.Title)) + "\n\n")

	// Title input
	sb.WriteString(s.titleInput.View() + "\n\n")
	sb.WriteString(s.contentTextarea.View() + "\n\n")

	// Instructions
	sb.WriteString("Press Tab/Shift+Tab to switch fields, Ctrl+S to save, Esc to cancel")

	return sb.String()
}

// SpecEditor manages the overlay state and wraps baseSpecEditor
type SpecEditor struct {
	state              SpecEditorState
	baseEditor         baseSpecEditor
	confirmationDialog *ConfirmationDialog
	overlay            tea.Model
	width              int
	height             int
}

// SpecEditorState represents the current state
type SpecEditorState int

const (
	Editing SpecEditorState = iota
	ShowingConfirmation
)

// NewSpecEditor creates a new spec editor component
func NewSpecEditor(config SpecEditorConfig) SpecEditor {
	baseEditor := newBaseSpecEditor(config)
	confirmationDialog := NewConfirmationDialog("You have unsaved changes. Are you sure you want to exit?")

	overlay := overlay.New(
		confirmationDialog,
		baseEditor,
		overlay.Center,
		overlay.Center,
		0,
		0,
	)

	return SpecEditor{
		state:              Editing,
		baseEditor:         baseEditor,
		confirmationDialog: confirmationDialog,
		overlay:            overlay,
	}
}

// Init initializes the spec editor
func (s *SpecEditor) Init() tea.Cmd {
	return nil
}

// SetSize sets the dimensions of the spec editor
func (s *SpecEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.baseEditor.SetSize(width, height)
}

// Update handles tea messages and updates the component
func (s *SpecEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
		if s.overlay != nil {
			if setter, ok := s.overlay.(interface{ SetSize(int, int) }); ok {
				setter.SetSize(msg.Width, msg.Height)
			}
		}
		return s, nil
	case tea.KeyMsg:
		// If showing confirmation overlay, handle y/n/esc directly
		if s.state == ShowingConfirmation {
			switch msg.String() {
			case "y", "Y":
				return s, func() tea.Msg { return SpecEditorCancelMsg{} }
			case "n", "N", "esc":
				s.state = Editing
				return s, nil
			}
		}

		// Handle Esc key for showing confirmation
		if msg.Type == tea.KeyEsc && s.state == Editing {
			if s.baseEditor.checkForChanges() {
				s.state = ShowingConfirmation
				return s, nil
			} else {
				// No changes, exit immediately
				return s, func() tea.Msg {
					return SpecEditorCancelMsg{}
				}
			}
		}

		// Update the base editor
		baseEditor, cmd := s.baseEditor.Update(msg)
		if updatedBase, ok := baseEditor.(baseSpecEditor); ok {
			s.baseEditor = updatedBase
		}
		return s, cmd
	}

	// Handle confirmation dialog messages
	switch msg.(type) {
	case SpecEditorCancelMsg:
		// User confirmed exit
		return s, func() tea.Msg {
			return SpecEditorCancelMsg{}
		}
	case ConfirmationDismissMsg:
		// User dismissed confirmation, return to editing
		s.state = Editing
		return s, nil
	}

	// Update the base editor for other messages
	baseEditor, cmd := s.baseEditor.Update(msg)
	if updatedBase, ok := baseEditor.(baseSpecEditor); ok {
		s.baseEditor = updatedBase
	}
	return s, cmd
}

// View renders the spec editor
func (s *SpecEditor) View() string {
	if s.state == ShowingConfirmation {
		// Recreate the overlay with the current base editor state
		overlay := overlay.New(
			s.confirmationDialog,
			s.baseEditor,
			overlay.Center,
			overlay.Center,
			0,
			0,
		)
		return overlay.View()
	}
	return s.baseEditor.View()
}
