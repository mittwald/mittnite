package cmd

import (
	"github.com/spf13/cobra"
)

var (
	apiAddress string
)

func init() {
	ctlCommand.PersistentFlags().StringVarP(&apiAddress, "api-address", "", defaultApiAddress, "write mittnites process id to this file")
}

var ctlCommand = &cobra.Command{
	Use:   "mittnitectl",
	Short: "controll mittnite (with --api) from cli",
	Long:  "This command can be used to control mittnite by command line.",
}
