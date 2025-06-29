package speclistview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
)

// Model represents the state of the spec list view screen
type Model struct {
	specs  []interactive.Spec
	cursor int
}

// New creates a new model for the spec list view screen
func New() Model {
	return Model{}
}

// SetSpecs sets the specifications to be displayed
func (m *Model) SetSpecs(specs []interactive.Spec) {
	m.specs = specs
	m.cursor = 0
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.specs)-1 {
				m.cursor++
			}
		}
	}
	return *m, nil
}

// View renders the spec list view screen
func (m *Model) View() string {
	if len(m.specs) == 0 {
		return "No specifications found.\n\nPress Esc to return to main menu"
	}

	var s strings.Builder
	s.WriteString("📋 Specifications List\n")
	s.WriteString("=====================\n\n")

	for i, spec := range m.specs {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		title := spec.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		s.WriteString(fmt.Sprintf("%s %s (%s)\n", cursor, title, spec.CreatedAt))
	}

	s.WriteString("\nUse ↑/↓ arrows to navigate, Esc to go back")
	return s.String()
}
