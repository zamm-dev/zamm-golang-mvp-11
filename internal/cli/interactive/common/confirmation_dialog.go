package common

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type ConfirmationDialogConfig struct {
	Title         string
	Message       string
	ConfirmAction string      // "delete_spec", "delete_link", etc.
	TargetID      string      // ID of the target being confirmed
	TargetTitle   string      // Title/description of the target
	ExtraData     interface{} // Additional data needed for the action
}

type ConfirmationAcceptedMsg struct {
	Action    string
	TargetID  string
	ExtraData interface{}
}

type ConfirmationCancelledMsg struct{}

type DeleteConfirmationDialog struct {
	config ConfirmationDialogConfig
	width  int
	height int
}

// NewDeleteConfirmationDialog creates a new confirmation dialog component
func NewDeleteConfirmationDialog(config ConfirmationDialogConfig) DeleteConfirmationDialog {
	return DeleteConfirmationDialog{
		config: config,
	}
}

// Init initializes the confirmation dialog
func (d DeleteConfirmationDialog) Init() tea.Cmd {
	return nil
}

// SetSize sets the dimensions of the confirmation dialog
func (d *DeleteConfirmationDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// Update handles tea messages and updates the component
func (d DeleteConfirmationDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.SetSize(msg.Width, msg.Height)
		return d, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return d, tea.Quit
		case "esc", "n":
			return d, func() tea.Msg { return ConfirmationCancelledMsg{} }
		case "y":
			return d, func() tea.Msg {
				return ConfirmationAcceptedMsg{
					Action:    d.config.ConfirmAction,
					TargetID:  d.config.TargetID,
					ExtraData: d.config.ExtraData,
				}
			}
		}
	}
	return d, nil
}

// View renders the confirmation dialog
func (d DeleteConfirmationDialog) View() string {
	var sb strings.Builder

	sb.WriteString("⚠️  Confirm Deletion\n")
	sb.WriteString("===================\n\n")

	switch d.config.ConfirmAction {
	case "delete_spec":
		sb.WriteString(fmt.Sprintf("Are you sure you want to delete the specification '%s'?\n\n", d.config.TargetTitle))
	case "delete_link":
		if linkData, ok := d.config.ExtraData.(LinkItem); ok {
			sb.WriteString(fmt.Sprintf("Are you sure you want to delete the link to commit %s?\n\n", linkData.CommitID[:12]+"..."))
		} else {
			sb.WriteString("Are you sure you want to delete this link?\n\n")
		}
	default:
		sb.WriteString(d.config.Message + "\n\n")
	}

	sb.WriteString("Press 'y' to confirm, 'n' or Esc to cancel")
	return sb.String()
}
