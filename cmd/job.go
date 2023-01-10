package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	ctlCommand.AddCommand(jobCommand)
}

var jobCommand = &cobra.Command{
	Use:   "job",
	Short: "Control a job via command line",
	Long:  "This command can be used to control a managed job.",
}
