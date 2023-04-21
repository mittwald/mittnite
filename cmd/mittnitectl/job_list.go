package main

import (
	"fmt"
	"github.com/mittwald/mittnite/pkg/cli"
	"github.com/mittwald/mittnite/pkg/proc"
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

			if status.Body.Running {
				fmt.Printf("%s %s (%s; reason: %s; pid: %d)\n", colorRunning("▶︎"), colorHighlight(job), colorRunning("running"), colorHighlight(status.Body.Phase.Reason), status.Body.Pid)
			} else if status.Body.Phase.Reason == proc.JobPhaseReasonStopped {
				fmt.Printf("%s %s (%s)\n", colorStopped("◼︎"), colorHighlight(job), colorStopped("stopped"))
			} else {
				fmt.Printf("%s %s (%s; reason: %s)\n", colorFailed("◼︎"), colorHighlight(job), colorFailed("not running"), colorHighlight(status.Body.Phase.Reason))
			}
		}

		fmt.Print("\nTo change the status of one of these processes, you can use the following commands:\n\n")
		fmt.Printf("  %s %s\n", colorCmd(cmd.Parent().CommandPath()+" start"), colorHighlight("<job>"))
		fmt.Printf("  %s %s\n", colorCmd(cmd.Parent().CommandPath()+" stop"), colorHighlight("<job>"))
		fmt.Printf("  %s %s\n", colorCmd(cmd.Parent().CommandPath()+" restart"), colorHighlight("<job>"))

		return nil
	},
}
