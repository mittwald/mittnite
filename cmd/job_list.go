package cmd

import (
	"github.com/mittwald/mittnite/pkg/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	jobCommand.AddCommand(jobListCommand)
}

var jobListCommand = &cobra.Command{
	Use:   "list",
	Short: "List jobs",
	Long:  "This command can be used to list all managed jobs.",
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := cli.NewApiClient(apiAddress)

		resp := apiClient.JobList()
		if err := resp.Print(); err != nil {
			log.Errorf("failed to print output: %s", err.Error())
		}
	},
}
