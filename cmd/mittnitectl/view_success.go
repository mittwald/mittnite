package main

import "github.com/charmbracelet/lipgloss"

var styleSuccessBox = lipgloss.NewStyle().
	Padding(0, 1).
	Margin(1, 0).
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(colorSuccess).
	Width(80)
