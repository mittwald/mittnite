package cmd

import (
	"github.com/spf13/cobra"
)

var (
	apiAddress string
)

func init() {
	ctlCommand.PersistentFlags().StringVarP(&apiAddress, "api-address", "", defaultAPIAddress, "write mittnites process id to this file")
}

var ctlCommand = &cobra.Command{
	Use:   "mittnitectl",
	Short: "control mittnite (with --api) from cli",
	Long:  "This command can be used to control mittnite by command line.",
}
