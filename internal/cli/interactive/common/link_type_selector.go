package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LinkType represents the type of link that can be selected
type LinkType int

const (
	GitCommitLink LinkType = iota
	SpecLink
)

// LinkOption represents a link option that can be selected
type LinkOption struct {
	Type  LinkType
	Label string
}

// FilterValue returns a value for filtering - implements list.Item
func (o LinkOption) FilterValue() string {
	return o.Label
}

// Predefined link options - these are the core link types
var (
	GitCommitOption = LinkOption{
		Type:  GitCommitLink,
		Label: "[G]it Commit",
	}

	SpecOption = LinkOption{
		Type:  SpecLink,
		Label: "[S]pecification",
	}
)

// LinkOptionSelectedMsg is sent when a link option is selected
type LinkOptionSelectedMsg struct {
	LinkType LinkType
}

// LinkTypeCancelledMsg is sent when the user cancels link type selection
type LinkTypeCancelledMsg struct{}

// linkDelegate handles rendering of link option items in the list
type linkDelegate struct{}

func (d linkDelegate) Height() int                             { return 1 }
func (d linkDelegate) Spacing() int                            { return 0 }
func (d linkDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d linkDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	option, ok := listItem.(LinkOption)
	if !ok {
		return
	}

	if index == m.Index() {
		// Show selector and highlight when this item is selected and list is in focus
		fmt.Fprint(w, HighlightStyle().Render("> "+option.Label))
	} else {
		// No selector when list is not in focus or item is not selected
		fmt.Fprint(w, defaultStyle.Render("  "+option.Label))
	}
}

// LinkTypeSelector is a reusable component for selecting link options
type LinkTypeSelector struct {
	list     list.Model
	title    string
	delegate linkDelegate
}

// NewLinkTypeSelector creates a new link selector component
func NewLinkTypeSelector(title string) LinkTypeSelector {
	delegate := linkDelegate{}

	// Hardcoded options - always the same
	options := []list.Item{GitCommitOption, SpecOption}
	l := list.New(options, delegate, 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return LinkTypeSelector{
		list:     l,
		title:    title,
		delegate: delegate,
	}
}

// SetSize sets the dimensions of the link selector
func (s *LinkTypeSelector) SetSize(width, height int) {
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

// Update handles tea messages and updates the component
func (s *LinkTypeSelector) Update(msg tea.Msg) (*LinkTypeSelector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "g":
			return s, func() tea.Msg {
				return LinkOptionSelectedMsg{LinkType: GitCommitLink}
			}
		case "s":
			return s, func() tea.Msg {
				return LinkOptionSelectedMsg{LinkType: SpecLink}
			}
		case "enter":
			if selectedOption := s.GetSelectedOption(); selectedOption != nil {
				return s, func() tea.Msg {
					return LinkOptionSelectedMsg{LinkType: selectedOption.Type}
				}
			}
			return s, nil
		case "esc":
			return s, func() tea.Msg {
				return LinkTypeCancelledMsg{}
			}
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
