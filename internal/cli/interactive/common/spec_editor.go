package common

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

// SpecEditor is a reusable component for creating/editing specifications
type SpecEditor struct {
	config          SpecEditorConfig
	mode            SpecEditorMode
	titleInput      textinput.Model
	contentTextarea textarea.Model
	width           int
	height          int
	initialized     bool // Track if textarea has been properly sized
}

// NewSpecEditor creates a new spec editor component
func NewSpecEditor(config SpecEditorConfig) SpecEditor {
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

	return SpecEditor{
		config:          config,
		mode:            EditingTitle, // Start with title focused
		titleInput:      titleInput,
		contentTextarea: contentTextarea,
	}
}

// SetSize sets the dimensions of the spec editor
func (s *SpecEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.titleInput.Width = width

	// Store dimensions but don't set textarea size yet to avoid panic
	// We'll set it in the first Update call when it's safe
}

// Update handles tea messages and updates the component
func (s *SpecEditor) Update(msg tea.Msg) (*SpecEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
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
	} else {
		s.contentTextarea, cmd = s.contentTextarea.Update(msg)
	}

	return s, cmd
}

// View renders the spec editor
func (s *SpecEditor) View() string {
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
