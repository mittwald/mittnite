package cmd

import (
	"fmt"
	"github.com/hashicorp/hcl"
	"github.com/mittwald/mittnite/internal/types"
	"github.com/mittwald/mittnite/pkg/files"
	"github.com/mittwald/mittnite/pkg/probe"
	"github.com/mittwald/mittnite/pkg/proc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

var (
	Version string
	Commit  string
	BuiltAt string
)

func init() {
	rootCmd.AddCommand(up)
}

var up = &cobra.Command{
	Use: "up",
	Run: func(cmd *cobra.Command, args []string) {

		log.Infof("mittnite process manager, version %s (commit %s), built at %s", Version, Commit, BuiltAt)
		log.Infof("looking for configuration files in %s", configDir)

		configDir = strings.TrimRight(configDir, "/")

		var matches []string

		err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Fatal(err)
			}
			if strings.HasSuffix(info.Name(), "hcl") {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}

		if len(matches) == 0 {
			log.Fatalf("could not find any configuration files in %s", configDir)
		}

		ignitionConfig := types.IgnitionConfig{}

		for _, m := range matches {
			log.Infof("found config file: %s", m)

			contents, err := ioutil.ReadFile(m)
			if err != nil {
				panic(err)
			}

			err = hcl.Unmarshal(contents, &ignitionConfig)
			if err != nil {
				err = fmt.Errorf("could not parse configuration file %s: %s", m, err.Error())
				panic(err)
			}
		}

		err = files.RenderConfigurationFiles(ignitionConfig.Files)
		if err != nil {
			panic(err)
		}

		signals := make(chan os.Signal)
		signal.Notify(signals)

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

		probeHandler, _ := probe.NewProbeHandler(&ignitionConfig)

		go func() {
			err := probe.RunProbeServer(probeHandler, probeSignals)
			if err != nil {
				log.Fatalf("probe server stopped with error: %s", err)
				panic(err)
			} else {
				log.Info("probe server stopped without error")
			}
		}()

		err = probeHandler.Wait(readinessSignals)
		if err != nil {
			panic(err)
		}

		err = proc.RunServices(&ignitionConfig, procSignals)
		if err != nil {
			log.Fatalf("service runner stopped with error: %s", err)
		} else {
			log.Print("service runner stopped without error")
		}
	},
}
