package cmd

import (
	"github.com/mittwald/mittnite/pkg/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	ctlCommand.AddCommand(jobCommand)
	jobCommand.SetHelpTemplate(`{{.Long}}
	
Usage:
  {{.UseLine}}

Arguments:
    name:   the name of the job
    action: possible values are "start", "restart", "stop", "status" and "logs"

Flags:
  {{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Global Flags:
  {{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
`)
}

var jobCommand = &cobra.Command{
	Use:        "job <name> <action>",
	Args:       cobra.ExactArgs(2),
	ArgAliases: []string{"name", "action"},
	Short:      "Control a job via command line",
	Long:       "This command can be used to control a job managed by mittnite.",
	Run: func(cmd *cobra.Command, args []string) {
		job := args[0]
		action := args[1]
		apiClient := cli.NewApiClient(apiAddress)

		resp := apiClient.CallAction(job, action)
		if err := resp.Print(); err != nil {
			log.Errorf("failed to print output: %s", err.Error())
		}
	},
}
