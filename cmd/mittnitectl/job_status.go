package main

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/mittwald/mittnite/pkg/cli"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	jobStatusCmd.Flags().BoolP("json", "j", false, "Print job status as JSON")
	jobStatusCmd.Flags().Bool("exit-with-status", false, "Exit with status code 0 if job is running, 1 if not running")

	jobCommand.AddCommand(&jobStatusCmd)
}

var styleStatusMainLine = lipgloss.NewStyle().Margin(1, 0)
var styleStatusDetails = lipgloss.NewStyle().PaddingLeft(2)
var styleStatusLeftColumn = lipgloss.NewStyle().Width(20)
var styleStatusAddendum = lipgloss.NewStyle().PaddingLeft(3)

var jobStatusCmd = cobra.Command{
	Use:        "status <job>",
	Args:       cobra.MaximumNArgs(1),
	ArgAliases: []string{"job"},
	Short:      "Show job status",
	Long:       "This command can be used to show the status of a managed job.\n\nWhen only one job is managed, the job name can be omitted.",

	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := cli.NewApiClient(apiAddress)

		job, err := determineJobName(args, apiClient)
		if err != nil {
			return err
		}

		resp := apiClient.JobStatus(job)
		if resp.Err() != nil {
			return fmt.Errorf("failed to get status of job %s: %w", job, resp.Err())
		}

		exitWithStatus, _ := cmd.Flags().GetBool("exit-with-status")

		if printJson, _ := cmd.Flags().GetBool("json"); printJson {
			if err := resp.Print(); err != nil {
				return fmt.Errorf("failed to print output: %w", err)
			}
		} else {
			fmt.Println(styleStatusMainLine.Render(jobStatusLine(job, resp.Body)))
			fmt.Println(styleStatusDetails.Render(lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.JoinHorizontal(
					lipgloss.Left,
					styleStatusLeftColumn.Render("command:"),
					styleHighlight.Render(resp.Body.Config.Command),
					styleStatusAddendum.Render("(args: "), styleHighlight.Render(fmt.Sprintf("%v", resp.Body.Config.Args)), ")",
				),
				lipgloss.JoinHorizontal(lipgloss.Left, styleStatusLeftColumn.Render("working directory:"), wrapNotSet(resp.Body.Config.WorkingDirectory)),
				lipgloss.JoinHorizontal(
					lipgloss.Left,
					styleStatusLeftColumn.Render("allowed to fail:"),
					styleHighlight.Render(fmt.Sprintf("%t", resp.Body.Config.CanFail)),
					styleStatusAddendum.Render("(max restart attempts: "), styleHighlight.Render(fmt.Sprintf("%d", resp.Body.Config.MaxAttempts)), ")",
				),
				lipgloss.JoinHorizontal(
					lipgloss.Left,
					styleStatusLeftColumn.Render("stdout log file:"),
					wrapNotSet(resp.Body.Config.Stdout),
				),
				lipgloss.JoinHorizontal(
					lipgloss.Left,
					styleStatusLeftColumn.Render("stderr log file:"),
					wrapNotSet(resp.Body.Config.Stderr),
				),
			)))
		}

		fmt.Println(styleInfoBox.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				"To change the status of this processes, you can use the following commands:",
				styleCommandBlock.Render(lipgloss.JoinVertical(lipgloss.Left,
					styleCommand.Render(cmd.Parent().CommandPath()+" start")+styleParam.Render(" "+job),
					styleCommand.Render(cmd.Parent().CommandPath()+" stop")+styleParam.Render(" "+job),
					styleCommand.Render(cmd.Parent().CommandPath()+" restart")+styleParam.Render(" "+job),
				)),
				"To view the process output, you can use the following command:",
				styleCommandBlock.Render(lipgloss.JoinVertical(lipgloss.Left,
					styleCommand.Render(cmd.Parent().CommandPath()+" logs")+styleParam.Render(" "+job),
				)),
				"Visit https://github.com/mittwald/mittnite to learn more about using the mittnite init system.",
			),
		))

		if !resp.Body.Running && exitWithStatus {
			os.Exit(1)
		}

		return nil
	},
}

func wrapNotSet(s string) string {
	if s == "" {
		return styleNotSet.Render("<not set>")
	}

	return styleHighlight.Render(s)
}

func determineJobName(args []string, apiClient *cli.APIClient) (string, error) {
	if len(args) != 0 {
		return args[0], nil
	}

	jobs := apiClient.JobList()
	if jobs.Err() != nil {
		return "", fmt.Errorf("failed to list jobs: %w", jobs.Err())
	}

	if len(jobs.Body) == 0 {
		return "", errors.New("no jobs found")
	}

	if len(jobs.Body) > 1 {
		return "", errors.New("more than one job found; please provide a job name as argument")
	}

	return jobs.Body[0], nil
}
