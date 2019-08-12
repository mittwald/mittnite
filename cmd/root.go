package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

type InitFlags struct {
	ConfigDir string
}

var RootCmd = &cobra.Command{
	Use:   "mittnite",
	Short: "mittnite",
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("starting with default config dir '/etc/mittnite.d'")
	},
}

var Init *InitFlags

func AddCommands(flags *InitFlags) {

	Init = flags
	RootCmd.AddCommand(configDirCmd)

	if err := RootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
