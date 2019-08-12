package main

import (
	"fmt"
	"github.com/hashicorp/hcl"
	"github.com/mittwald/mittnite/cmd"
	"github.com/mittwald/mittnite/config"
	"github.com/mittwald/mittnite/files"
	"github.com/mittwald/mittnite/probe"
	"github.com/mittwald/mittnite/proc"
	log "github.com/sirupsen/logrus"
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

func Init() cmd.InitFlags {
	flags := cmd.InitFlags{ConfigDir: "/etc/mittnite.d"}
	Formatter := new(log.TextFormatter)
	Formatter.TimestampFormat = "02-01-2006 15:04:05"
	Formatter.FullTimestamp = true
	log.SetFormatter(Formatter)

	cmd.AddCommands(&flags)

	return flags

}

func main() {

	initFlags := Init()

	log.Infof("mittnite process manager, version %s (commit %s), built at %s", Version, Commit, BuiltAt)
	log.Infof("looking for configuration files in %s", initFlags.ConfigDir)

	initFlags.ConfigDir = strings.TrimRight(initFlags.ConfigDir, "/")

	var matches []string

	err := filepath.Walk(initFlags.ConfigDir, func(path string, info os.FileInfo, err error) error {
		spl := strings.Split(info.Name(), ".")
		if spl[len(spl)-1] == "hcl" {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	if len(matches) == 0 {
		log.Fatalf("could not find any configuration files in %s", initFlags.ConfigDir)
	}

	ignitionConfig := config.IgnitionConfig{}

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

	log.Println(ignitionConfig)

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
			log.Infof("probe server stopped with error: %s", err)
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
		log.Printf("service runner stopped with error: %s", err)
		panic(err)
	} else {
		log.Print("service runner stopped without error")
	}
}
