package cmd

import (
	"fmt"
	"github.com/mittwald/mittnite/pkg/cli"
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
		Args:       cobra.ExactArgs(1),
		ArgAliases: []string{"job"},
		Short:      shortDesc,
		Long:       longDesc,
		Run: func(cmd *cobra.Command, args []string) {
			job := args[0]
			apiClient := cli.NewApiClient(apiAddress)

			resp := apiClient.CallAction(job, action)
			if err := resp.Print(); err != nil {
				log.Errorf("failed to print output: %s", err.Error())
			}
		},
	}
}
