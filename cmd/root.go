package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var configDir string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configDir, "config-dir", "c", "/etc/mittnite.d", "set directory to where your .hcl-configs are located")
}

var rootCmd = &cobra.Command{
	Use:     "mittnite",
	Short:   "Mittnite - Smart init system for containers",
	Long:    "Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn("Running 'mittnite' without any arguments - defaulting to 'up'. This behaviour may change in future releases!")
		up.Run(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
