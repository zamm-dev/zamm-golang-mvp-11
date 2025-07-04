package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LinkOption represents a link option that can be selected
type LinkOption struct {
	ID          string
	Label       string
	Description string
}

// FilterValue returns a value for filtering - implements list.Item
func (o LinkOption) FilterValue() string {
	return o.Label
}

// LinkOptionSelectedMsg is sent when a link option is selected
type LinkOptionSelectedMsg struct {
	Option LinkOption
}

// LinkSelectorConfig configures the behavior of the link selector
type LinkSelectorConfig struct {
	Title       string
	Options     []LinkOption
	ShowNumbers bool // Whether to show numbers before options
}

// DefaultLinkSelectorConfig returns sensible default configuration
func DefaultLinkSelectorConfig() LinkSelectorConfig {
	return LinkSelectorConfig{
		Title:       "Select Option",
		Options:     []LinkOption{},
		ShowNumbers: true,
	}
}

// linkDelegate handles rendering of link option items in the list
type linkDelegate struct {
	isInFocus   bool // Whether the list is currently in focus
	showNumbers bool // Whether to show numbers before options
}

func (d linkDelegate) Height() int                             { return 1 }
func (d linkDelegate) Spacing() int                            { return 0 }
func (d linkDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d linkDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	option, ok := listItem.(LinkOption)
	if !ok {
		return
	}

	str := option.Label
	if d.showNumbers {
		str = fmt.Sprintf("%d. %s", index+1, str)
	}

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

// LinkSelector is a reusable component for selecting link options
type LinkSelector struct {
	list     list.Model
	config   LinkSelectorConfig
	width    int
	height   int
	delegate linkDelegate
}

// NewLinkSelector creates a new link selector component
func NewLinkSelector(config LinkSelectorConfig) LinkSelector {
	delegate := linkDelegate{
		isInFocus:   true, // Start in focus by default
		showNumbers: config.ShowNumbers,
	}

	// Convert options to list items
	items := make([]list.Item, len(config.Options))
	for i, option := range config.Options {
		items[i] = option
	}

	l := list.New(items, delegate, 0, 0)
	l.Title = config.Title
	l.SetShowHelp(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return LinkSelector{
		list:     l,
		config:   config,
		delegate: delegate,
	}
}

// SetFocus sets whether the link selector list is currently in focus
func (s *LinkSelector) SetFocus(inFocus bool) {
	s.delegate.isInFocus = inFocus
	s.list.SetDelegate(s.delegate)
}

// IsFocused returns whether the link selector list is currently in focus
func (s *LinkSelector) IsFocused() bool {
	return s.delegate.isInFocus
}

// SetSize sets the dimensions of the link selector
func (s *LinkSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width, height)
}

// SetOptions sets the available options
func (s *LinkSelector) SetOptions(options []LinkOption) {
	// Convert to list items
	items := make([]list.Item, len(options))
	for i, option := range options {
		items[i] = option
	}
	s.list.SetItems(items)
	s.config.Options = options
}

// GetSelectedOption returns the currently selected option, if any
func (s *LinkSelector) GetSelectedOption() *LinkOption {
	if item := s.list.SelectedItem(); item != nil {
		if option, ok := item.(LinkOption); ok {
			return &option
		}
	}
	return nil
}

// ResetCursor resets the list cursor to the first item
func (s *LinkSelector) ResetCursor() {
	s.list.Select(0)
}

// Update handles tea messages and updates the component
func (s *LinkSelector) Update(msg tea.Msg) (*LinkSelector, tea.Cmd) {
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
func (s *LinkSelector) View() string {
	if len(s.list.Items()) == 0 {
		return fmt.Sprintf("%s\n\nNo options available.\nPress Esc to go back.", s.config.Title)
	}

	return s.list.View()
}
