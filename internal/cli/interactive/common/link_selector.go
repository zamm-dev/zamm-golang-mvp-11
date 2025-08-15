package common

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type LinkSelectorConfig struct {
	Title            string
	CurrentSpecID    string
	CurrentSpecTitle string
	Links            []LinkItem
}

type LinkItem struct {
	ID        string
	CommitID  string
	RepoPath  string
	LinkLabel string
}

type LinkSelectorCompleteMsg struct {
	Action       string // "delete_link"
	SelectedLink LinkItem
}

type LinkSelectorCancelMsg struct{}

type LinkSelectedMsg struct {
	Link LinkItem
}

type LinkSelector struct {
	config LinkSelectorConfig
	cursor int
	width  int
	height int
}

func NewLinkSelector(config LinkSelectorConfig) LinkSelector {
	return LinkSelector{
		config: config,
		cursor: 0,
	}
}

func (s LinkSelector) Init() tea.Cmd {
	return nil
}

// SetSize sets the dimensions of the link selector
func (s *LinkSelector) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Update handles tea messages and updates the component
func (s LinkSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return s, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return s, tea.Quit
		case "esc":
			return s, func() tea.Msg { return LinkSelectorCancelMsg{} }
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.config.Links)-1 {
				s.cursor++
			}
		case "enter", " ":
			if len(s.config.Links) > 0 {
				selectedLink := s.config.Links[s.cursor]
				return s, func() tea.Msg {
					return LinkSelectorCompleteMsg{
						Action:       "delete_link",
						SelectedLink: selectedLink,
					}
				}
			}
		}
	}
	return s, nil
}

func (s LinkSelector) View() string {
	var sb strings.Builder

	sb.WriteString("🗑️  Delete Specification Link\n")
	sb.WriteString("=============================\n\n")

	if len(s.config.Links) == 0 {
		sb.WriteString("No links found for this specification.\n\n")
		sb.WriteString("Press Esc to return to main menu")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("Links for '%s':\n\n", s.config.CurrentSpecTitle))

	for i, link := range s.config.Links {
		cursor := " "
		if s.cursor == i {
			cursor = ">"
		}
		repoName := filepath.Base(link.RepoPath)
		sb.WriteString(fmt.Sprintf("%s %s (%s, %s)\n", cursor, link.CommitID[:12]+"...", repoName, link.LinkLabel))
	}

	sb.WriteString("\nUse ↑/↓ arrows to navigate, Enter to delete, Esc to go back")
	return sb.String()
}
