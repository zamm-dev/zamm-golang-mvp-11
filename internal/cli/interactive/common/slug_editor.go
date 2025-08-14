package common

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SlugEditorCompleteMsg struct {
	Slug string
}

type SlugEditorCancelMsg struct{}

type SlugEditor struct {
	title     string
	nodeTitle string
	input     textinput.Model
	width     int
	height    int
}

func NewSlugEditor(title, nodeTitle, initialSlug string) SlugEditor {
	input := textinput.New()
	input.Placeholder = "Enter slug for organization"
	input.Focus()
	input.SetValue(initialSlug)
	input.Width = 50

	return SlugEditor{
		title:     title,
		nodeTitle: nodeTitle,
		input:     input,
	}
}

func (s SlugEditor) Init() tea.Cmd {
	return textinput.Blink
}

// SetSize sets the dimensions of the slug editor
func (s *SlugEditor) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.input.Width = width - 4
}

// Update handles tea messages and updates the component
func (s SlugEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
		return s, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return s, func() tea.Msg {
				return SlugEditorCancelMsg{}
			}
		case tea.KeyEnter:
			slug := strings.TrimSpace(s.input.Value())
			if slug == "" {
				return s, nil
			}
			return s, func() tea.Msg {
				return SlugEditorCompleteMsg{
					Slug: slug,
				}
			}
		}
	}

	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s SlugEditor) View() string {
	var sb strings.Builder

	sb.WriteString(s.title + "\n")
	sb.WriteString(strings.Repeat("=", len(s.title)) + "\n\n")

	sb.WriteString("Node: " + s.nodeTitle + "\n\n")

	sb.WriteString("Edit the slug that will be used for organizing this node:\n\n")

	sb.WriteString(s.input.View() + "\n\n")

	sb.WriteString("Press Enter to organize with this slug, Esc to cancel")

	content := sb.String()
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(lipgloss.Color("6")).
		Padding(1, 2).
		Width(s.width - 4).
		Align(lipgloss.Center)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, style.Render(content))
}

func sanitizeSlug(title string) string {
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "untitled"
	}
	return slug
}
