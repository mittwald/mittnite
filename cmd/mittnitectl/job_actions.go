package main

import (
	"errors"
	"fmt"
	"github.com/mittwald/mittnite/pkg/cli"
	"github.com/spf13/cobra"
	"time"
)

func init() {
	jobCommand.AddCommand(buildJobActionCommand(
		"start",
		"Start a job",
		"This command can be used to start a managed job.",
		"▶️  starting job %s",
		"🕑 waiting for job %s to start",
		"🚀 job %s started",
		testRunning,
	))
	jobCommand.AddCommand(buildJobActionCommand(
		"restart",
		"Restart a job",
		"This command can be used to restart a managed job.",
		"🔄  restarting job %s",
		"🕑 waiting for job %s to restart",
		"🥳 job %s restarted",
		testRunning,
	))
	jobCommand.AddCommand(buildJobActionCommand(
		"stop",
		"Stop a job",
		"This command can be used to stop a managed job.",
		"⏸️  stopping job %s",
		"🕑 waiting for job %s to stop",
		"😵 job %s stopped",
		testStopped,
	))
}

func testRunning(job string, client *cli.APIClient) (bool, error) {
	status := client.JobStatus(job)
	if err := status.Err(); err != nil {
		return false, fmt.Errorf("failed to get status of job %s: %w", job, err)
	}

	return status.Body.Running, nil
}

func testStopped(job string, client *cli.APIClient) (bool, error) {
	status := client.JobStatus(job)
	if err := status.Err(); err != nil {
		return false, fmt.Errorf("failed to get status of job %s: %w", job, err)
	}

	return !status.Body.Running, nil
}

func waitForCondition(job string, client *cli.APIClient, waitTimeout time.Duration, waitFunc func(string, *cli.APIClient) (bool, error)) error {
	waitStart := time.Now()

	for {
		time.Sleep(100 * time.Millisecond)

		ok, err := waitFunc(job, client)
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		if time.Since(waitStart) > waitTimeout {
			return fmt.Errorf("timeout waiting for job %s to reach desired state", job)
		}
	}
}

func buildJobActionCommand(
	action string,
	shortDesc,
	longDesc,
	startMsg,
	waitMsg,
	doneMsg string,
	waitFunc func(string, *cli.APIClient) (bool, error),
) *cobra.Command {
	cmd := cobra.Command{
		Use:        fmt.Sprintf("%s [--wait] <job>", action),
		Args:       cobra.MaximumNArgs(1),
		ArgAliases: []string{"job"},
		Short:      shortDesc,
		Long:       longDesc + "\n\nWhen only one job is managed, the job name can be omitted.",

		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := cli.NewApiClient(apiAddress)

			job, err := determineJobName(args, apiClient)
			if err != nil {
				return err
			}

			fmt.Printf(startMsg+"\n", styleHighlight.Render(job))

			resp := apiClient.CallAction(job, action)
			if err := resp.Print(); err != nil {
				return err
			}

			wait, waitErr := cmd.Flags().GetBool("wait")
			duration, durationErr := cmd.Flags().GetDuration("wait-for")

			if err := errors.Join(waitErr, durationErr); err != nil {
				return fmt.Errorf("failed to get wait flags: %w", err)
			}

			if wait {
				fmt.Printf(waitMsg+"\n", styleHighlight.Render(job))

				if err := waitForCondition(job, apiClient, duration, waitFunc); err != nil {
					return err
				}

				fmt.Println(styleSuccessBox.Render(
					fmt.Sprintf(doneMsg, styleHighlight.Render(job)),
				))
			}

			return nil
		},
	}

	cmd.Flags().BoolP("wait", "w", true, "wait for the job to have reached the desired state before completing.")
	cmd.Flags().Duration("wait-for", 5*time.Second, "maximum time to wait for the job to have reached the desired state before completing.")

	return &cmd
}
