package common

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SlugEditorCompleteMsg is sent when slug editing is complete
type SlugEditorCompleteMsg struct {
	SpecID string
	Slug   string
}

// SlugEditorCancelMsg is sent when user cancels slug editing
type SlugEditorCancelMsg struct{}

// baseSlugEditor contains the core slug editor logic without overlay management
type baseSlugEditor struct {
	specID    string
	slugInput textinput.Model
	width     int
	height    int
}

// newBaseSlugEditor creates a new base slug editor component
func newBaseSlugEditor(specID, initialSlug string) baseSlugEditor {
	slugInput := textinput.New()
	slugInput.Placeholder = "enter-slug-here"
	slugInput.Focus()
	slugInput.SetValue(initialSlug)
	slugInput.CharLimit = 100

	return baseSlugEditor{
		specID:    specID,
		slugInput: slugInput,
	}
}

// Init initializes the base slug editor
func (s baseSlugEditor) Init() tea.Cmd {
	return nil
}

// SetSize sets the dimensions of the base slug editor
func (s *baseSlugEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.slugInput.Width = width - 10 // Leave some margin
}

// Update handles tea messages for the base slug editor
func (s baseSlugEditor) Update(msg tea.Msg) (baseSlugEditor, tea.Cmd) {
	switch msg := msg.(type) {
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
		}
	}

	var cmd tea.Cmd
	s.slugInput, cmd = s.slugInput.Update(msg)
	return s, cmd
}

// View renders the base slug editor
func (s *baseSlugEditor) View() string {
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(2, 4).
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("7"))

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	title := titleStyle.Render("Edit Slug Before Organizing")

	instructionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	instructions := instructionStyle.Render("Press Enter to confirm, Esc to cancel")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		"Slug:",
		s.slugInput.View(),
		"",
		instructions,
	)

	return dialogStyle.Render(content)
}

// SlugEditor is a slug editor component
type SlugEditor struct {
	inner baseSlugEditor
}

// NewSlugEditor creates a new slug editor component
func NewSlugEditor(specID, initialSlug string) *SlugEditor {
	inner := newBaseSlugEditor(specID, initialSlug)

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
