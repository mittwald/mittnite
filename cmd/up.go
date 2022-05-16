package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/pkg/files"
	"github.com/mittwald/mittnite/pkg/probe"
	"github.com/mittwald/mittnite/pkg/proc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	probeListenPort int
	pidFile         string
)

func init() {
	log.StandardLogger().ExitFunc = func(i int) {
		defer func() {
			_ = recover() // prevent from printing trace
		}()
		panic(fmt.Sprintf("exit %d", i))
	}
	rootCmd.AddCommand(up)
	up.PersistentFlags().IntVarP(&probeListenPort, "probe-listen-port", "p", 9102, "set the port to listen for probe requests")
	up.PersistentFlags().StringVarP(&pidFile, "pidfile", "", "", "write mittnites process id to this file")
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

		if err := writePidFile(); err != nil {
			log.Fatalf("failed to write pid file to %q: %s", pidFile, err)
		}
		defer func() {
			if err := deletePidFile(); err != nil {
				log.Errorf("error while cleaning up the pid file: %s", err)
			}
		}()

		if err := ignitionConfig.GenerateFromConfigDir(configDir); err != nil {
			log.Fatalf("failed while trying to generate ignition config from dir '%+v', err: '%+v'", configDir, err)
		}

		if err := files.RenderFiles(ignitionConfig.Files); err != nil {
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
			log.Infof("probeServer listens on port %d", probeListenPort)
			err := probe.RunProbeServer(probeHandler, probeSignals, probeListenPort)
			if err != nil {
				log.Fatalf("probe server stopped with error: %s", err)
			} else {
				log.Info("probe server stopped without error")
			}
		}()

		if err := probeHandler.Wait(readinessSignals); err != nil {
			log.Fatalf("probe handler failed while waiting for readiness signals: '%+v'", err)
		}

		go proc.ReapChildren()

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

func writePidFile() error {
	if pidFile == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(pidFile), 0o755); err != nil {
		return err
	}
	if stats, err := os.Stat(pidFile); err == nil {
		if stats.Size() > 0 {
			return fmt.Errorf("pidFile %q already exists", pidFile)
		}
	}

	return os.WriteFile(pidFile, pidToByteString(), 0644)

}

func deletePidFile() error {
	if pidFile == "" {
		return nil
	}

	pid := pidToByteString()
	content, err := os.ReadFile(pidFile)
	if err != nil {
		return err
	}

	if bytes.Compare(pid, content) != 0 {
		return fmt.Errorf("won't delete pid file %q because it does not contain the expected content", pidFile)
	}

	return os.Remove(pidFile)
}

func pidToByteString() []byte {
	return []byte(fmt.Sprintf("%d", os.Getpid()))
}
