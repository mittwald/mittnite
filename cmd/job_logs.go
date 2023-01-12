package cmd

import (
	"github.com/mittwald/mittnite/pkg/cli"
	log "github.com/sirupsen/logrus"
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
	Args:       cobra.ExactArgs(1),
	ArgAliases: []string{"job"},
	Short:      "Get logs from job",
	Long:       "This command can be used to get the logs of a managed job.",
	Run: func(cmd *cobra.Command, args []string) {
		job := args[0]
		apiClient := cli.NewApiClient(apiAddress)

		if tailLen < -1 {
			tailLen = -1
		}
		resp := apiClient.JobLogs(job, follow, tailLen)
		if err := resp.Print(); err != nil {
			log.Errorf("failed to print output: %s", err.Error())
		}
	},
}
