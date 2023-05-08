package main

import (
	"github.com/charmbracelet/lipgloss"
)

var styleErrorWrapper = lipgloss.NewStyle().Padding(0, 0).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#E1244C"))
var styleErrorHeadingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E1244C")).Bold(true)
var styleErrorBodyStyle = lipgloss.NewStyle().PaddingLeft(3).Foreground(lipgloss.Color("#E1244C")).Width(80).MaxWidth(80)

func renderError(err error) string {
	return styleErrorWrapper.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			styleErrorHeadingStyle.Render("💥 AN ERROR OCCURRED WHILE HANDLING YOUR COMMAND"),
			styleErrorBodyStyle.Render(err.Error()),
			styleErrorBodyStyle.MarginTop(1).Render("If you think this is a bug, please feel free to open an issue at https://github.com/mittwald/mittnite"),
		),
	)
}
