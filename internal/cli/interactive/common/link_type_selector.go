package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LinkType represents the type of link that can be selected
type LinkType string

const (
	GitCommitLink LinkType = "git_commit"
	SpecLink      LinkType = "spec_link"
)

// LinkOption represents a link option that can be selected
type LinkOption struct {
	Type        LinkType
	Label       string
	Description string
}

// FilterValue returns a value for filtering - implements list.Item
func (o LinkOption) FilterValue() string {
	return o.Label
}

// Predefined link options - these are the core link types
var (
	GitCommitOption = LinkOption{
		Type:        GitCommitLink,
		Label:       "Git Commit",
		Description: "Link to a git commit",
	}

	SpecOption = LinkOption{
		Type:        SpecLink,
		Label:       "Specification",
		Description: "Link to another specification",
	}
)

// LinkOptionSelectedMsg is sent when a link option is selected
type LinkOptionSelectedMsg struct {
	Option LinkOption
}

// linkDelegate handles rendering of link option items in the list
type linkDelegate struct {
	isInFocus bool // Whether the list is currently in focus
}

func (d linkDelegate) Height() int                             { return 1 }
func (d linkDelegate) Spacing() int                            { return 0 }
func (d linkDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d linkDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	option, ok := listItem.(LinkOption)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, option.Label)

	maxWidth := m.Width() - 2 // account for padding
	if len(str) > maxWidth && maxWidth > 3 {
		str = str[:maxWidth-3] + "..."
	} else if len(str) > maxWidth && maxWidth > 0 {
		str = str[:maxWidth]
	}

	if index == m.Index() && d.isInFocus {
		// Show selector and highlight when this item is selected and list is in focus
		fmt.Fprint(w, HighlightStyle().Render("> "+str))
	} else {
		// No selector when list is not in focus or item is not selected
		fmt.Fprint(w, defaultStyle.Render("  "+str))
	}
}

// LinkTypeSelector is a reusable component for selecting link options
type LinkTypeSelector struct {
	list     list.Model
	title    string
	width    int
	height   int
	delegate linkDelegate
}

// NewLinkTypeSelector creates a new link selector component
func NewLinkTypeSelector(title string) LinkTypeSelector {
	delegate := linkDelegate{
		isInFocus: true, // Start in focus by default
	}

	// Hardcoded options - always the same
	options := []LinkOption{GitCommitOption, SpecOption}

	// Convert options to list items
	items := make([]list.Item, len(options))
	for i, option := range options {
		items[i] = option
	}

	l := list.New(items, delegate, 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return LinkTypeSelector{
		list:     l,
		title:    title,
		delegate: delegate,
	}
}

// SetFocus sets whether the link selector list is currently in focus
func (s *LinkTypeSelector) SetFocus(inFocus bool) {
	s.delegate.isInFocus = inFocus
	s.list.SetDelegate(s.delegate)
}

// IsFocused returns whether the link selector list is currently in focus
func (s *LinkTypeSelector) IsFocused() bool {
	return s.delegate.isInFocus
}

// SetSize sets the dimensions of the link selector
func (s *LinkTypeSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width, height)
}

// GetSelectedOption returns the currently selected option, if any
func (s *LinkTypeSelector) GetSelectedOption() *LinkOption {
	if item := s.list.SelectedItem(); item != nil {
		if option, ok := item.(LinkOption); ok {
			return &option
		}
	}
	return nil
}

// ResetCursor resets the list cursor to the first item
func (s *LinkTypeSelector) ResetCursor() {
	s.list.Select(0)
}

// Update handles tea messages and updates the component
func (s *LinkTypeSelector) Update(msg tea.Msg) (*LinkTypeSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if selectedOption := s.GetSelectedOption(); selectedOption != nil {
				return s, func() tea.Msg {
					return LinkOptionSelectedMsg{Option: *selectedOption}
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

// View renders the link selector
func (s *LinkTypeSelector) View() string {
	return s.list.View()
}
