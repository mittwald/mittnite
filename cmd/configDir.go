package cmd

import (
	"github.com/spf13/cobra"
)

// configDirCmd represents the configDir command
var configDirCmd = &cobra.Command{
	Use:   "configDir",
	Short: "Change default config dir for .hcl files",
	Run: func(cmd *cobra.Command, args []string) {
		Init.ConfigDir = args[0]
	},
}
