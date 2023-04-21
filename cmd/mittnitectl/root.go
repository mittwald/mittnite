package main

import (
	"github.com/mittwald/mittnite/cmd"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	apiAddress string
)

func init() {
	ctlCommand.PersistentFlags().StringVarP(&apiAddress, "api-address", "", cmd.DefaultAPIAddress, "write mittnites process id to this file")
	ctlCommand.AddCommand(cmd.VersionCmd)
}

var ctlCommand = &cobra.Command{
	Use:   "mittnitectl",
	Short: "control mittnite (with --api) from cli",
	Long:  "This command can be used to control mittnite by command line.",
}

func Execute() {
	if err := ctlCommand.Execute(); err != nil {
		log.Fatal(err)
	}
}
