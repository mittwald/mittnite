package main

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/mittwald/mittnite/pkg/cli"
	"github.com/mittwald/mittnite/pkg/proc"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	jobStatusCmd.Flags().BoolP("json", "j", false, "Print job status as JSON")
	jobStatusCmd.Flags().Bool("exit-with-status", false, "Exit with status code 0 if job is running, 1 if not running")

	jobCommand.AddCommand(&jobStatusCmd)
}

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
			b := color.New(color.FgHiBlue).SprintFunc()

			if resp.Body.Running {
				fmt.Printf("%s %s (%s; pid: %d)\n\n", colorRunning("▶︎"), b(job), colorRunning("running"), resp.Body.Pid)
			} else if resp.Body.Phase.Reason == proc.JobPhaseReasonStopped {
				fmt.Printf("%s %s (%s)\n", colorStopped("◼︎"), colorHighlight(job), colorStopped("stopped"))
			} else {
				fmt.Printf("%s %s (%s)\n\n", colorFailed("◼︎"), b(job), colorFailed("not running"))
			}

			fmt.Printf("  Command:           %s   Arguments: %s\n", b(resp.Body.Config.Command), b(resp.Body.Config.Args))
			fmt.Printf("  Working directory: %s\n", wrapNotSet(resp.Body.Config.WorkingDirectory))
			fmt.Printf("  Can Fail:          %s (Max restart attempts: %s)\n", b(resp.Body.Config.CanFail), b(resp.Body.Config.MaxAttempts))
		}

		fmt.Print("\nTo change the status of this process, you can use the following commands:\n\n")
		fmt.Printf("  %s %s\n", colorCmd(cmd.Parent().CommandPath()+" start"), colorHighlight(job))
		fmt.Printf("  %s %s\n", colorCmd(cmd.Parent().CommandPath()+" stop"), colorHighlight(job))
		fmt.Printf("  %s %s\n", colorCmd(cmd.Parent().CommandPath()+" restart"), colorHighlight(job))

		if !resp.Body.Running && exitWithStatus {
			os.Exit(1)
		}

		return nil
	},
}

func wrapNotSet(s string) string {
	if s == "" {
		return color.New(color.FgHiYellow).Sprint("<not set>")
	}

	return color.New(color.FgHiBlue).Sprintf(s)
}

func determineJobName(args []string, apiClient *cli.ApiClient) (string, error) {
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
