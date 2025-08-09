package common

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// NodeType represents the type of node to create
type NodeType int

const (
	NodeTypeSpecification NodeType = iota
	NodeTypeImplementation
)

// NodeTypeOption is a selectable option in the list
type NodeTypeOption struct {
	Type  NodeType
	Label string
}

func (o NodeTypeOption) FilterValue() string { return o.Label }

var (
	SpecificationOption  = NodeTypeOption{Type: NodeTypeSpecification, Label: "[S]pecification"}
	ImplementationOption = NodeTypeOption{Type: NodeTypeImplementation, Label: "[I]mplementation"}
)

// Messages
type NodeTypeSelectedMsg struct{ Type NodeType }
type NodeTypeCancelledMsg struct{}

// delegate for list rendering
type nodeTypeDelegate struct{}

func (d nodeTypeDelegate) Height() int                             { return 1 }
func (d nodeTypeDelegate) Spacing() int                            { return 0 }
func (d nodeTypeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d nodeTypeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	option, ok := listItem.(NodeTypeOption)
	if !ok {
		return
	}
	if index == m.Index() {
		_, _ = fmt.Fprint(w, HighlightStyle().Render("> "+option.Label))
	} else {
		_, _ = fmt.Fprint(w, defaultStyle.Render("  "+option.Label))
	}
}

// NodeTypeSelector component
type NodeTypeSelector struct {
	list     list.Model
	delegate nodeTypeDelegate
	title    string
}

func NewNodeTypeSelector(title string) NodeTypeSelector {
	delegate := nodeTypeDelegate{}
	options := []list.Item{SpecificationOption, ImplementationOption}
	l := list.New(options, delegate, 0, 0)
	l.Title = title
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.Styles.Title = lipgloss.NewStyle().Bold(true)

	return NodeTypeSelector{list: l, delegate: delegate, title: title}
}

func (s *NodeTypeSelector) SetSize(width, height int) { s.list.SetSize(width, height) }

func (s *NodeTypeSelector) GetSelectedOption() *NodeTypeOption {
	if item := s.list.SelectedItem(); item != nil {
		if opt, ok := item.(NodeTypeOption); ok {
			return &opt
		}
	}
	return nil
}

func (s *NodeTypeSelector) Init() tea.Cmd { return nil }

func (s *NodeTypeSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			return s, func() tea.Msg { return NodeTypeSelectedMsg{Type: NodeTypeSpecification} }
		case "i":
			return s, func() tea.Msg { return NodeTypeSelectedMsg{Type: NodeTypeImplementation} }
		case "enter":
			if selected := s.GetSelectedOption(); selected != nil {
				return s, func() tea.Msg { return NodeTypeSelectedMsg{Type: selected.Type} }
			}
		case "esc":
			return s, func() tea.Msg { return NodeTypeCancelledMsg{} }
		}
	}
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return s, cmd
}

func (s *NodeTypeSelector) View() string { return s.list.View() }
