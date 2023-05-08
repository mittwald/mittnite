package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/mittwald/mittnite/pkg/proc"
)

func jobStatusLine(job string, status proc.CommonJobStatus) string {
	if status.Running {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			styleRunning.Render("▶︎"), " ",
			styleHighlight.Render(job), " (",
			styleRunning.Render("running"), "; reason=",
			styleHighlight.Render(string(status.Phase.Reason)), "; pid=",
			styleHighlight.Render(fmt.Sprintf("%d", status.Pid)), ")",
		)
	} else if status.Phase.Reason == proc.JobPhaseReasonStopped {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			styleStopped.Render("◼︎"), " ",
			styleHighlight.Render(job), " (",
			styleStopped.Render("stopped"), ")",
		)
	} else {
		return lipgloss.JoinHorizontal(lipgloss.Left,
			styleFailed.Render("◼︎"), " ",
			styleHighlight.Render(job), " (",
			styleFailed.Render("not running"), "; reason=",
			styleHighlight.Render(string(status.Phase.Reason)), ")",
		)
	}
}
