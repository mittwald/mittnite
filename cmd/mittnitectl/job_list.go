package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/mittwald/mittnite/pkg/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	jobListCommand.Flags().BoolP("json", "j", false, "output as JSON")

	jobCommand.AddCommand(jobListCommand)
}

var jobListCommand = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	Long:  "This command can be used to list all managed jobs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := cli.NewApiClient(apiAddress)

		resp := apiClient.JobList()
		if resp.Err() != nil {
			return resp.Err()
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			if err := resp.Print(); err != nil {
				log.Errorf("failed to print output: %s", err.Error())
			}
			return nil
		}

		fmt.Print("The following processes are managed, and controllable:\n\n")

		for _, job := range resp.Body {
			status := apiClient.JobStatus(job)
			if status.Err() != nil {
				return fmt.Errorf("failed to get status of job %s: %w", job, status.Err())
			}

			fmt.Println(styleListItem.Render(jobStatusLine(job, status.Body)))
		}

		fmt.Println(styleInfoBox.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				"To change the status of one of these processes, you can use the following commands:",
				styleCommandBlock.Render(lipgloss.JoinVertical(lipgloss.Left,
					styleCommand.Render(cmd.Parent().CommandPath()+" start")+styleParam.Render(" <job>"),
					styleCommand.Render(cmd.Parent().CommandPath()+" stop")+styleParam.Render(" <job>"),
					styleCommand.Render(cmd.Parent().CommandPath()+" restart")+styleParam.Render(" <job>"),
				)),
				"Visit https://github.com/mittwald/mittnite to learn more about using the mittnite init system.",
			),
		))

		return nil
	},
}
