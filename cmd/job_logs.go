package cmd

import (
	"github.com/mittwald/mittnite/pkg/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var follow bool

func init() {
	jobLogsCommand.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "output appended data as the file grows")
	jobCommand.AddCommand(jobLogsCommand)
}

var jobLogsCommand = &cobra.Command{
	Use:   "logs <job>",
	Short: "Get logs from job",
	Long:  "This command can be used to get the logs of a managed job.",
	Run: func(cmd *cobra.Command, args []string) {
		job := args[0]
		apiClient := cli.NewApiClient(apiAddress)

		resp := apiClient.JobLogs(job, follow)
		if err := resp.Print(); err != nil {
			log.Errorf("failed to print output: %s", err.Error())
		}
	},
}
