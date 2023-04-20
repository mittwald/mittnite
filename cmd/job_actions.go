package cmd

import (
	"fmt"
	"github.com/mittwald/mittnite/pkg/cli"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	jobCommand.AddCommand(buildJobActionCommand("start", "Start a job", "This command can be used to start a managed job."))
	jobCommand.AddCommand(buildJobActionCommand("restart", "Restart a job", "This command can be used to restart a managed job."))
	jobCommand.AddCommand(buildJobActionCommand("stop", "Stop a job", "This command can be used to stop a managed job."))
	jobCommand.AddCommand(buildJobActionCommand("status", "Show job status", "This command can be used to show the status of a managed job."))
}

func buildJobActionCommand(action string, shortDesc, longDesc string) *cobra.Command {
	return &cobra.Command{
		Use:        fmt.Sprintf("%s <job>", action),
		Args:       cobra.MaximumNArgs(1),
		ArgAliases: []string{"job"},
		Short:      shortDesc,
		Long:       longDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := cli.NewApiClient(apiAddress)
			job := ""

			if len(args) == 0 {
				jobs := apiClient.JobList()
				if jobs.Err() != nil {
					return errors.Wrapf(jobs.Err(), "failed to list jobs")
				}

				if len(jobs.Body) == 0 {
					return errors.New("no jobs found")
				}

				if len(jobs.Body) > 1 {
					return errors.New("more than one job found; please provide a job name as argument")
				}

				job = jobs.Body[0]
			} else {
				job = args[0]
			}

			resp := apiClient.CallAction(job, action)
			if err := resp.Print(); err != nil {
				log.Errorf("failed to print output: %s", err.Error())
			}

			return nil
		},
	}
}
