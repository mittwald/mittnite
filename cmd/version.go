package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	Version string
	Commit  string
	BuiltAt string
)

func init() {
	rootCmd.AddCommand(VersionCmd)
}

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show extended information about the current version of mittnite",
	Long:  "All software has versions. This is mittnite's",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Mittnite process manager, version %s (commit %s), built at %s", Version, Commit, BuiltAt)
	},
}
