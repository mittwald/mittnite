package main

import (
	"github.com/charmbracelet/lipgloss"
)

var colorSuccess = lipgloss.Color("#00B785")

var styleRunning = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
var styleStopped = lipgloss.NewStyle().Foreground(lipgloss.Color("#e08dff")).Bold(true)
var styleFailed = lipgloss.NewStyle().Foreground(lipgloss.Color("#e1244c")).Bold(true)
var styleHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color("#407FF8")).Bold(true)
var styleNotSet = lipgloss.NewStyle().Foreground(lipgloss.Color("#5D689C"))

var styleCommand = lipgloss.NewStyle().Foreground(lipgloss.Color("#407FF8")).Bold(true)
var styleCommandBlock = lipgloss.NewStyle().Margin(1, 0).PaddingLeft(2)
var styleParam = lipgloss.NewStyle().Foreground(lipgloss.Color("#00B785"))

var styleListItem = lipgloss.NewStyle().Padding(0, 2)
var styleInfoBox = lipgloss.NewStyle().
	Padding(0, 1).
	Margin(1, 0).
	BorderStyle(lipgloss.RoundedBorder()).
	Width(80)
