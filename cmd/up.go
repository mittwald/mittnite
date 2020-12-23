package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/pkg/files"
	"github.com/mittwald/mittnite/pkg/probe"
	"github.com/mittwald/mittnite/pkg/proc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(up)
}

var up = &cobra.Command{
	Use:   "up",
	Short: "Render config files, start probes and processes",
	Long:  "This sub-command renders the configuration files, starts the probes and launches all configured processes",
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

		signals := make(chan os.Signal)
		signal.Notify(signals,
			syscall.SIGTERM,
			syscall.SIGINT,
		)

		readinessSignals := make(chan os.Signal, 1)
		probeSignals := make(chan os.Signal, 1)
		procSignals := make(chan os.Signal, 1)

		go func() {
			for s := range signals {
				log.Infof("received event %s", s.String())
				readinessSignals <- s
				probeSignals <- s
				procSignals <- s
			}
		}()

		probeHandler, _ := probe.NewProbeHandler(ignitionConfig)

		go func() {
			err := probe.RunProbeServer(probeHandler, probeSignals)
			if err != nil {
				log.Fatalf("probe server stopped with error: %s", err)
			} else {
				log.Info("probe server stopped without error")
			}
		}()

		err = probeHandler.Wait(readinessSignals)
		if err != nil {
			log.Fatalf("probe handler failed while waiting for readiness signals: '%+v'", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		runner := proc.NewRunner(ignitionConfig)
		go func() {
			<-procSignals
			cancel()
		}()

		if err := runner.Boot(ctx); err != nil {
			log.WithError(err).Fatal("runner error'ed during initialization")
		} else {
			log.Info("initialization complete")
		}

		if err := runner.Run(ctx); err != nil {
			log.WithError(err).Fatal("service runner stopped with error")
		} else {
			log.Print("service runner stopped without error")
		}
	},
}
