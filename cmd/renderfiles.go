package cmd

import (
	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/pkg/files"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(renderFiles)
}

var renderFiles = &cobra.Command{
	Use: "renderfiles",
	Run: func(cmd *cobra.Command, args []string) {
		ignitionConfig := &config.Ignition{
			Probes: nil,
			Files:  nil,
			Jobs:   nil,
		}

		err := ignitionConfig.GenerateFromConfigDir(configDir)
		if err != nil {
			panic(err)
		}

		err = files.RenderFiles(ignitionConfig.Files)
		if err != nil {
			panic(err)
		}
	},
}
