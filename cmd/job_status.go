package cmd

import (
	"errors"
	"fmt"
	"github.com/fatih/color"
	"github.com/mittwald/mittnite/pkg/cli"
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
			c := color.New(color.FgHiBlue, color.Bold)
			b := color.New(color.FgBlue).SprintFunc()

			if resp.Body.Running {
				c = color.New(color.FgHiGreen, color.Bold)
				c.Printf("▶︎ RUNNING: %s (PID: %d)\n", job, resp.Body.Pid)
			} else {
				c = color.New(color.FgHiRed, color.Bold)
				c.Printf("◼︎ NOT RUNNING: %s\n", job)
			}

			fmt.Printf("  Command:           %s   Arguments: %s\n", b(resp.Body.Config.Command), b(resp.Body.Config.Args))
			fmt.Printf("  Working directory: %s\n", b(resp.Body.Config.WorkingDirectory))
			fmt.Printf("  Can Fail:          %s (Max restart attempts: %s)\n", b(resp.Body.Config.CanFail), b(resp.Body.Config.MaxAttempts))
		}

		if !resp.Body.Running && exitWithStatus {
			os.Exit(1)
		}

		return nil
	},
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
