package speclistview

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourorg/zamm-mvp/internal/cli/interactive"
	"github.com/yourorg/zamm-mvp/internal/models"
)

// CreateNewSpecMsg signals that the user wants to create a new specification
type CreateNewSpecMsg struct{}
type LinkCommitSpecMsg struct {
	SpecID string
}
type EditSpecMsg struct {
	SpecID string
}
type DeleteSpecMsg struct {
	SpecID string
}
type RemoveLinkSpecMsg struct {
	SpecID string
}

// Model represents the state of the spec list view screen
type LinkService interface {
	GetCommitsForSpec(specID string) ([]*models.SpecCommitLink, error)
}

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Create key.Binding
	Edit   key.Binding
	Delete key.Binding
	Link   key.Binding
	Remove key.Binding
	Help   key.Binding
	Return key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "next"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "prev"),
	),
	Create: key.NewBinding(
		key.WithKeys("c", "C"),
		key.WithHelp("c", "create"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e", "E"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", "D"),
		key.WithHelp("d", "delete"),
	),
	Link: key.NewBinding(
		key.WithKeys("l", "L"),
		key.WithHelp("l", "link commit"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r", "R"),
		key.WithHelp("r", "remove commit"),
	),
	Return: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("h", "?"),
		key.WithHelp("h", "help"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Create, k.Edit, k.Delete, k.Link, k.Remove, k.Help, k.Return}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Create, k.Edit, k.Delete},
		{k.Link, k.Remove},
		{k.Help, k.Return},
	}
}

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

type Model struct {
	list        list.Model
	keys        keyMap
	help        help.Model
	specs       []interactive.Spec
	links       []*models.SpecCommitLink
	linkService LinkService

	width  int
	height int
}

// New creates a new model for the spec list view screen
func New(linkService LinkService) Model {
	list := list.New([]list.Item{}, specDelegate{}, 0, 0)
	list.Title = "Specifications"
	list.SetShowHelp(false)
	list.Styles.Title = lipgloss.NewStyle().Bold(true)
	return Model{
		list:        list,
		keys:        keys,
		help:        help.New(),
		linkService: linkService,
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.help.Width = width
	m.setListSize()
}

func (m *Model) paneWidth() int {
	return (m.width - 1) / 2 // width of each half pane, minus 1 for padding
}

func (m *Model) setListSize() {
	if m.help.ShowAll {
		m.list.SetSize(m.paneWidth(), m.height-4)
	} else {
		m.list.SetSize(m.paneWidth(), m.height-3)
	}
}

// SetSpecs sets the specifications to be displayed
func (m *Model) SetSpecs(specs []interactive.Spec) {
	m.specs = specs

	items := make([]list.Item, len(specs))
	for i, s := range specs {
		items[i] = s
	}
	m.list.SetItems(items)

	// Fetch links for the first spec if available
	if m.linkService != nil && len(specs) > 0 {
		links, err := m.linkService.GetCommitsForSpec(specs[0].ID)
		if err == nil {
			m.links = links
		} else {
			m.links = nil
		}
	}
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up) || key.Matches(msg, m.keys.Down):
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			spec, ok := m.list.SelectedItem().(interactive.Spec)
			if ok {
				links, err := m.linkService.GetCommitsForSpec(spec.ID)
				if err == nil {
					m.links = links
				} else {
					m.links = nil
				}
			}
			return *m, cmd
		case key.Matches(msg, m.keys.Create):
			return *m, func() tea.Msg { return CreateNewSpecMsg{} }
		case key.Matches(msg, m.keys.Edit):
			spec, ok := m.list.SelectedItem().(interactive.Spec)
			if !ok {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return EditSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Delete):
			spec, ok := m.list.SelectedItem().(interactive.Spec)
			if !ok {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return DeleteSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Link):
			spec, ok := m.list.SelectedItem().(interactive.Spec)
			if !ok {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return LinkCommitSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Remove):
			spec, ok := m.list.SelectedItem().(interactive.Spec)
			if !ok {
				return *m, nil // No valid spec selected
			}
			return *m, func() tea.Msg { return RemoveLinkSpecMsg{SpecID: spec.ID} }
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.setListSize() // Adjust list size based on help visibility
			return *m, nil
		}
	}
	return *m, nil
}

// View renders the spec list view screen
func (m *Model) View() string {
	if len(m.specs) == 0 {
		return "No specifications found.\n\nPress Esc to return to main menu"
	}

	paneWidth := m.paneWidth()

	// Layout: left (list), right (details)
	var left strings.Builder
	left.WriteString(m.list.View())
	left.WriteString(m.help.View(m.keys))
	finalLeft := lipgloss.JoinVertical(lipgloss.Top, m.list.View(), m.help.View(m.keys))

	// Right: details for selected spec
	var right strings.Builder
	spec, ok := m.list.SelectedItem().(interactive.Spec)
	if ok {
		right.WriteString(fmt.Sprintf("%s\n%s\n\n", spec.Title, strings.Repeat("=", paneWidth)))
		right.WriteString(spec.Content)
		right.WriteString("\n\nLinked Commits:\n")
		if len(m.links) == 0 {
			right.WriteString("  (none)\n")
		} else {
			right.WriteString("  COMMIT           REPO             TYPE         CREATED\n")
			right.WriteString("  ──────           ────             ────         ───────\n")
			for _, l := range m.links {
				commitID := l.CommitID
				if len(commitID) > 12 {
					commitID = commitID[:12] + "..."
				}
				repo := l.RepoPath
				linkType := l.LinkType
				created := l.CreatedAt.Format("2006-01-02 15:04")
				right.WriteString(fmt.Sprintf("  %-16s %-16s %-12s %s\n", commitID, repo, linkType, created))
			}
		}
	} else {
		right.WriteString("No specification selected. Create or select one to view details.\n")
	}
	finalRight := lipgloss.NewStyle().Width(paneWidth + 1).PaddingLeft(1).Render(right.String())

	return lipgloss.JoinHorizontal(lipgloss.Left, finalLeft, finalRight)
}
