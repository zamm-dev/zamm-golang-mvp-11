package common

import (
	"github.com/charmbracelet/lipgloss"
)

func HighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
}
