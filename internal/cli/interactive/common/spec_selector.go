package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spec represents a specification node
type Spec struct {
	ID      string
	Title   string
	Content string
	Type    string
}

// FilterValue implements list.Item interface for bubbles list component
func (s Spec) FilterValue() string {
	return s.Title
}

// SpecSelectedMsg is sent when a spec is selected
type SpecSelectedMsg struct {
	Spec Spec
}

// SpecSelectorConfig configures the behavior of the spec selector
type SpecSelectorConfig struct {
	Title string
}

// specDelegate handles rendering of spec items in the list
type specDelegate struct {
	isInFocus bool // Whether the list is currently in focus
}

func (d specDelegate) Height() int                             { return 1 }
func (d specDelegate) Spacing() int                            { return 0 }
func (d specDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d specDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	spec, ok := listItem.(Spec)
	if !ok {
		return
	}

	str := spec.Title
	maxWidth := m.Width() - 2 // account for padding
	if len(str) > maxWidth && maxWidth > 3 {
		str = str[:maxWidth-3] + "..."
	} else if len(str) > maxWidth && maxWidth > 0 {
		str = str[:maxWidth]
	}

	if index == m.Index() && d.isInFocus {
		// Show selector and highlight when this item is selected and list is in focus
		_, _ = fmt.Fprint(w, HighlightStyle().Render("> "+str))
	} else {
		// No selector when list is not in focus or item is not selected
		_, _ = fmt.Fprint(w, defaultStyle.Render("  "+str))
	}
}

// SpecSelector is a reusable component for selecting specifications
type SpecSelector struct {
	list     list.Model
	config   SpecSelectorConfig
	width    int
	height   int
	delegate specDelegate
}

// NewSpecSelector creates a new spec selector component
func NewSpecSelector(config SpecSelectorConfig) SpecSelector {
	delegate := specDelegate{isInFocus: true} // Start in focus by default
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = config.Title
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return SpecSelector{
		list:     l,
		config:   config,
		delegate: delegate,
	}
}

// SetFocus sets whether the spec selector list is currently in focus
func (s *SpecSelector) SetFocus(inFocus bool) {
	s.delegate.isInFocus = inFocus
	s.list.SetDelegate(s.delegate)
}

// IsFocused returns whether the spec selector list is currently in focus
func (s *SpecSelector) IsFocused() bool {
	return s.delegate.isInFocus
}

// SetSize sets the dimensions of the spec selector
func (s *SpecSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width, height)
}

// SetSpecs sets the available specifications
func (s *SpecSelector) SetSpecs(specs []Spec) {
	// Convert to list items
	items := make([]list.Item, len(specs))
	for i, spec := range specs {
		items[i] = spec
	}
	s.list.SetItems(items)
}

// GetSelectedSpec returns the currently selected spec, if any
func (s *SpecSelector) GetSelectedSpec() *Spec {
	if item := s.list.SelectedItem(); item != nil {
		if spec, ok := item.(Spec); ok {
			return &spec
		}
	}
	return nil
}

// ResetCursor resets the list cursor to the first item
func (s *SpecSelector) ResetCursor() {
	s.list.Select(0)
}

// Update handles tea messages and updates the component
func (s *SpecSelector) Update(msg tea.Msg) (*SpecSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if s.list.SettingFilter() { // let list handle filter input
			break
		}

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
