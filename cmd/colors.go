package cmd

import "github.com/fatih/color"

var colorHighlight = color.New(color.FgHiBlue).SprintFunc()
var colorRunning = color.New(color.FgHiGreen, color.Bold).SprintFunc()
var colorStopped = color.New(color.FgHiYellow, color.Bold).SprintFunc()
var colorFailed = color.New(color.FgHiRed, color.Bold).SprintFunc()
var colorCmd = color.New(color.Bold).SprintFunc()
