package common

import (
	"github.com/charmbracelet/lipgloss"
)

var defaultStyle = lipgloss.NewStyle()

func HighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
}

// ActiveNodeStyle returns the style for active nodes (bold blue)
func ActiveNodeStyle() lipgloss.Style {
	return HighlightStyle()
}
