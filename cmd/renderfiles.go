package cmd

import (
	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/pkg/files"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
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

		if len(args) > 0 {
			log.Infof("additional command/args provided - executing: '%+v'", args)
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				panic(err)
			}
		}
	},
}
