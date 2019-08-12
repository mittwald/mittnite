package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var configDir string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configDir, "config-dir", "c", "/etc/mittnite.d", "")
}

var rootCmd = &cobra.Command{
	Use:   "up",
	Short: "Mittnite - Smart init system for containers",
	Long:  "Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images.",
	Run: func(cmd *cobra.Command, args []string) {
		up.Run(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
