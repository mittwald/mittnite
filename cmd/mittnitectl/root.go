package main

import (
	"fmt"
	"github.com/mittwald/mittnite/cmd"
	"github.com/spf13/cobra"
	"os"
)

var (
	apiAddress string
)

func init() {
	ctlCommand.PersistentFlags().StringVarP(&apiAddress, "api-address", "", cmd.DefaultAPIAddress, "write mittnites process id to this file")
	ctlCommand.AddCommand(cmd.VersionCmd)
}

var ctlCommand = &cobra.Command{
	Use:           "mittnitectl",
	Short:         "control mittnite (with --api) from cli",
	Long:          "This command can be used to control mittnite by command line.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := ctlCommand.Execute(); err != nil {
		fmt.Println(renderError(err))
		os.Exit(1)
	}
}
