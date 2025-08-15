package common

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zamm-dev/zamm-golang-mvp-11/internal/services"
)

// SlugEditorCompleteMsg is sent when slug editing is complete
type SlugEditorCompleteMsg struct {
	SpecID string
	Slug   string
}

// SlugEditorCancelMsg is sent when user cancels slug editing
type SlugEditorCancelMsg struct{}

// LLMSlugSuggestionMsg contains the LLM-generated slug suggestion
type LLMSlugSuggestionMsg struct {
	Suggestion string
	Error      error
}

// baseSlugEditor contains the core slug editor logic without overlay management
type baseSlugEditor struct {
	specID        string
	originalTitle string
	slugInput     textinput.Model
	width         int
	height        int
	llmService    services.LLMService
	llmSuggestion string
	llmError      error
	userHasEdited bool
	llmRequested  bool // Track if LLM was requested
}

// newBaseSlugEditor creates a new base slug editor component
func newBaseSlugEditor(specID, originalTitle, initialSlug string, llmService services.LLMService) baseSlugEditor {
	slugInput := textinput.New()
	slugInput.Placeholder = "generating-suggestion..."
	slugInput.Focus()

	// Check if we need LLM assistance (more than 3 words)
	// Split on hyphens since the slug has already been processed
	slugParts := strings.Split(initialSlug, "-")
	needsLLM := len(slugParts) > 3

	var llmRequested bool
	if needsLLM {
		// Start with empty slug when we need LLM assistance
		slugInput.SetValue("")
		llmRequested = true
	} else {
		// Use initial slug for short titles
		slugInput.SetValue(initialSlug)
	}

	slugInput.CharLimit = 100

	return baseSlugEditor{
		specID:        specID,
		originalTitle: originalTitle,
		slugInput:     slugInput,
		llmService:    llmService,
		llmRequested:  llmRequested,
	}
}

// Init initializes the base slug editor
func (s baseSlugEditor) Init() tea.Cmd {
	// If we need LLM assistance, start the async request
	// Check if the current slug has more than 3 parts (split on hyphens)
	currentSlug := s.slugInput.Value()
	if currentSlug == "" {
		// If slug is empty, we probably need LLM assistance
		if s.llmService == nil {
			// LLM service not available - return error immediately
			return func() tea.Msg {
				return LLMSlugSuggestionMsg{
					Suggestion: "",
					Error:      fmt.Errorf("LLM service not configured - set ANTHROPIC_API_KEY environment variable"),
				}
			}
		}
		if s.originalTitle != "" {
			// Make async LLM call for slug generation
			return func() tea.Msg {
				suggestion, err := s.llmService.GenerateSlug(s.originalTitle)
				return LLMSlugSuggestionMsg{
					Suggestion: suggestion,
					Error:      err,
				}
			}
		} else {
			return func() tea.Msg {
				return LLMSlugSuggestionMsg{
					Suggestion: "",
					Error:      fmt.Errorf("no original title provided for LLM processing"),
				}
			}
		}
	}
	return nil
}

// SetSize sets the dimensions of the base slug editor
func (s *baseSlugEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.slugInput.Width = width - 4 // Leave some margin for full-screen layout
}

// Update handles tea messages for the base slug editor
func (s baseSlugEditor) Update(msg tea.Msg) (baseSlugEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case LLMSlugSuggestionMsg:
		s.llmSuggestion = msg.Suggestion
		s.llmError = msg.Error

		// If user hasn't edited the field, set the suggestion
		if !s.userHasEdited && msg.Error == nil {
			s.slugInput.SetValue(msg.Suggestion)
			s.slugInput.Placeholder = "enter-slug-here"
		} else if msg.Error != nil {
			// Make the error more visible
			s.slugInput.Placeholder = "error-occurred-type-manually"
		}
		return s, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return s, func() tea.Msg {
				return SlugEditorCompleteMsg{
					SpecID: s.specID,
					Slug:   s.slugInput.Value(),
				}
			}
		case "esc":
			return s, func() tea.Msg {
				return SlugEditorCancelMsg{}
			}
		default:
			// Track if user has edited the field
			if !s.userHasEdited && len(msg.String()) == 1 {
				s.userHasEdited = true
				s.slugInput.Placeholder = "enter-slug-here"
			}

			// Only pass the message to textinput for non-handled keys
			var cmd tea.Cmd
			s.slugInput, cmd = s.slugInput.Update(msg)
			return s, cmd
		}
	}

	return s, nil
}

// View renders the base slug editor
func (s *baseSlugEditor) View() string {
	// Use fallback width if not set yet
	displayWidth := s.width
	if displayWidth <= 0 {
		displayWidth = 80 // Default terminal width
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	title := titleStyle.Render("Edit Slug Before Organizing")

	headerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
	header := headerStyle.Width(displayWidth).Render(title)

	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	instructions := instructionStyle.Render("Press Enter to confirm, Esc to cancel")

	slugLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render("Slug:")

	var content []string
	content = append(content, header, "")

	// Show original title for context
	if s.originalTitle != "" {
		titleLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Render("Title:")
		titleText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Width(displayWidth - 4). // Leave some margin
			Render(s.originalTitle)
		content = append(content, titleLabel, titleText, "")
	}

	content = append(content, slugLabel, s.slugInput.View())

	// Show LLM suggestion if user has edited and we have a suggestion
	if s.userHasEdited && s.llmSuggestion != "" && s.llmError == nil {
		suggestionLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("LLM suggestion:")
		suggestionText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Width(displayWidth - 4). // Leave some margin
			Render(s.llmSuggestion)
		content = append(content, "", suggestionLabel, suggestionText)
	}

	// Show error if LLM failed
	if s.llmError != nil {
		errorLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("LLM Error:")
		errorText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Width(displayWidth - 4). // Leave some margin
			Render(s.llmError.Error())
		content = append(content, "", errorLabel, errorText)
	}

	content = append(content, "", instructions)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

// SlugEditor is a slug editor component
type SlugEditor struct {
	inner baseSlugEditor
}

// NewSlugEditor creates a new slug editor component
func NewSlugEditor(specID, originalTitle, initialSlug string, llmService services.LLMService) *SlugEditor {
	inner := newBaseSlugEditor(specID, originalTitle, initialSlug, llmService)

	return &SlugEditor{
		inner: inner,
	}
}

// Init initializes the slug editor
func (s *SlugEditor) Init() tea.Cmd {
	return s.inner.Init()
}

// SetSize sets the dimensions of the slug editor
func (s *SlugEditor) SetSize(width, height int) {
	s.inner.SetSize(width, height)
}

// Update handles tea messages for the slug editor
func (s *SlugEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	inner, cmd := s.inner.Update(msg)
	s.inner = inner
	return s, cmd
}

// View renders the slug editor
func (s *SlugEditor) View() string {
	return s.inner.View()
}
