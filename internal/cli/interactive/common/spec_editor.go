package common

import (
	"fmt"
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
	ShowExisting   bool   // Whether to show existing values for edit mode
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
	// Force initialization by calling Focus then Blur
	contentTextarea.Focus()
	contentTextarea.Blur()

	return SpecEditor{
		config:          config,
		mode:            EditingTitle,
		titleInput:      titleInput,
		contentTextarea: contentTextarea,
	}
}

// SetSize sets the dimensions of the spec editor
func (s *SpecEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.titleInput.Width = width - 4 // Account for padding

	// Don't set textarea dimensions immediately - wait until we're in content editing mode
	// This prevents panics during early initialization
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
		case tea.KeyEnter:
			if s.mode == EditingTitle {
				if strings.TrimSpace(s.titleInput.Value()) == "" {
					return s, nil // Don't proceed with empty title
				}
				s.mode = EditingContent
				s.titleInput.Blur()

				// Set textarea dimensions now that we're switching to content mode
				if s.width > 10 && s.height > 15 {
					s.contentTextarea.SetWidth(s.width - 4)
					s.contentTextarea.SetHeight(s.height - 10)
				}

				s.contentTextarea.Focus()
				return s, nil
			}
		case tea.KeyCtrlS:
			if s.mode == EditingContent {
				title := strings.TrimSpace(s.titleInput.Value())
				content := strings.TrimSpace(s.contentTextarea.Value())

				// For edit mode, if content is empty, keep existing content
				if content == "" && s.config.InitialContent != "" {
					content = s.config.InitialContent
				}

				return s, func() tea.Msg {
					return SpecEditorCompleteMsg{
						Title:   title,
						Content: content,
					}
				}
			}
		case tea.KeyCtrlK:
			if s.mode == EditingContent && s.config.ShowExisting {
				// Keep existing content (for edit mode)
				title := strings.TrimSpace(s.titleInput.Value())
				return s, func() tea.Msg {
					return SpecEditorCompleteMsg{
						Title:   title,
						Content: s.config.InitialContent,
					}
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
	var sb strings.Builder

	// Header
	sb.WriteString(s.config.Title + "\n")
	sb.WriteString(strings.Repeat("=", len(s.config.Title)) + "\n\n")

	if s.mode == EditingTitle {
		// Show existing title if in edit mode
		if s.config.ShowExisting && s.config.InitialTitle != "" {
			sb.WriteString(fmt.Sprintf("Current title: %s\n\n", s.config.InitialTitle))
		}

		sb.WriteString("Enter title:\n")
		sb.WriteString(s.titleInput.View() + "\n\n")
		sb.WriteString("Press Enter to continue, Esc to cancel")
	} else if s.mode == EditingContent {
		sb.WriteString(fmt.Sprintf("Title: %s\n\n", s.titleInput.Value()))

		if s.config.ShowExisting && strings.TrimSpace(s.contentTextarea.Value()) == "" && s.config.InitialContent != "" {
			sb.WriteString("Current content:\n")
			for _, line := range strings.Split(s.config.InitialContent, "\n") {
				sb.WriteString("  " + line + "\n")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("Enter content (press Ctrl+S to finish")
		if s.config.ShowExisting {
			sb.WriteString(", Ctrl+K to keep existing")
		}
		sb.WriteString("):\n\n")

		sb.WriteString(s.contentTextarea.View() + "\n\n")
		sb.WriteString("Press Ctrl+S to finish")
		if s.config.ShowExisting {
			sb.WriteString(", Ctrl+K to keep existing")
		}
		sb.WriteString(", Esc to cancel")
	}

	return sb.String()
}
