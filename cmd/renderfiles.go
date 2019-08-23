package cmd

import (
	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(renderFiles)
}

var renderFiles = &cobra.Command{
	Use: "renderfiles",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("looking for configuration files in %s", configDir)

		ignitionConfig := &config.Ignition{
			Probes: nil,
			Files:  nil,
			Jobs:   nil,
		}

		err := ignitionConfig.GenerateFromConfigDir(configDir)
		if err != nil {
			panic(err)
		}

		err = config.RenderConfigurationFiles(ignitionConfig.Files)
		if err != nil {
			panic(err)
		}
	},
}
