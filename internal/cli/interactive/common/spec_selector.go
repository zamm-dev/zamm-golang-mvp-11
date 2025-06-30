package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
)

// SpecSelectedMsg is sent when a spec is selected
type SpecSelectedMsg struct {
	Spec interactive.Spec
}

// SpecSelectorConfig configures the behavior of the spec selector
type SpecSelectorConfig struct {
	Title string
}

// DefaultSpecSelectorConfig returns sensible default configuration
func DefaultSpecSelectorConfig() SpecSelectorConfig {
	return SpecSelectorConfig{
		Title: "Select Specification",
	}
}

// specDelegate handles rendering of spec items in the list
type specDelegate struct{}

var specStyle = lipgloss.NewStyle()

func (d specDelegate) Height() int                             { return 1 }
func (d specDelegate) Spacing() int                            { return 0 }
func (d specDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d specDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	spec, ok := listItem.(interactive.Spec)
	if !ok {
		return
	}

	str := spec.Title
	maxWidth := m.Width() - 2 // account for padding
	if len(str) > maxWidth {
		str = str[:maxWidth-3] + "..."
	}

	fn := specStyle.Render
	if index == m.Index() {
		fmt.Fprint(w, specStyle.Foreground(lipgloss.Color("2")).Render("> "+str))
	} else {
		fmt.Fprint(w, fn("  "+str))
	}
}

// SpecSelector is a reusable component for selecting specifications
type SpecSelector struct {
	list   list.Model
	config SpecSelectorConfig
	width  int
	height int
}

// NewSpecSelector creates a new spec selector component
func NewSpecSelector(config SpecSelectorConfig) SpecSelector {
	l := list.New([]list.Item{}, specDelegate{}, 0, 0)
	l.Title = config.Title
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return SpecSelector{
		list:   l,
		config: config,
	}
}

// SetSize sets the dimensions of the spec selector
func (s *SpecSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width, height)
}

// SetSpecs sets the available specifications
func (s *SpecSelector) SetSpecs(specs []interactive.Spec) {
	// Convert to list items
	items := make([]list.Item, len(specs))
	for i, spec := range specs {
		items[i] = spec
	}
	s.list.SetItems(items)
}

// GetSelectedSpec returns the currently selected spec, if any
func (s *SpecSelector) GetSelectedSpec() *interactive.Spec {
	if item := s.list.SelectedItem(); item != nil {
		if spec, ok := item.(interactive.Spec); ok {
			return &spec
		}
	}
	return nil
}

// Update handles tea messages and updates the component
func (s *SpecSelector) Update(msg tea.Msg) (*SpecSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if selectedSpec := s.GetSelectedSpec(); selectedSpec != nil {
				return s, func() tea.Msg {
					return SpecSelectedMsg{Spec: *selectedSpec}
				}
			}
			return s, nil
		}
	}

	// Update the list component
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

// View renders the spec selector
func (s *SpecSelector) View() string {
	if len(s.list.Items()) == 0 {
		return fmt.Sprintf("%s\n\nNo specifications available.\nPress Esc to go back.", s.config.Title)
	}

	return s.list.View()
}
