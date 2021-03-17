package cmd

import (
	"os"
	"os/exec"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/pkg/files"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(renderFiles)
}

// TODO(@hermsi1337): WTH do we even need this for!?
var renderFiles = &cobra.Command{
	Use:   "renderfiles",
	Short: "Renders configuration files",
	Long:  "This command renders the configured configuration files, before (optionally) starting another process",
	Run: func(cmd *cobra.Command, args []string) {
		ignitionConfig := &config.Ignition{
			Probes: nil,
			Files:  nil,
			Jobs:   nil,
		}

		err := ignitionConfig.GenerateFromConfigDir(configDir)
		if err != nil {
			log.Fatalf("failed while trying to generate ignition config from dir '%+v', err: '%+v'", configDir, err)
		}

		err = files.RenderFiles(ignitionConfig.Files)
		if err != nil {
			log.Fatalf("failed while rendering files from ignition config, err: '%+v'", err)
		}

		if len(args) > 0 {
			log.Infof("additional command/args provided - executing: '%+v'", args)
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				log.Fatalf("failed to execute additional args '%+v', err: '%+v'", args, err)
			}
		}
	},
}
