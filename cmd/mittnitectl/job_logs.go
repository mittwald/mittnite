package main

import (
	"fmt"
	"github.com/mittwald/mittnite/pkg/cli"
	"github.com/spf13/cobra"
)

var (
	follow  bool
	tailLen int
)

func init() {
	jobLogsCommand.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "output appended data as the file grows")
	jobLogsCommand.PersistentFlags().IntVarP(&tailLen, "tail", "", -1, "output last n lines")
	jobCommand.AddCommand(jobLogsCommand)
}

var jobLogsCommand = &cobra.Command{
	Use:        "logs <job>",
	Args:       cobra.MaximumNArgs(1),
	ArgAliases: []string{"job"},
	Short:      "Get logs from job",
	Long:       "This command can be used to get the logs of a managed job.",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiClient := cli.NewApiClient(apiAddress)

		job, err := determineJobName(args, apiClient)
		if err != nil {
			return err
		}

		if tailLen < -1 {
			tailLen = -1
		}
		resp := apiClient.JobLogs(job, follow, tailLen)
		if err := resp.Print(); err != nil {
			return fmt.Errorf("failed to print output: %w", err)
		}

		return nil
	},
}
