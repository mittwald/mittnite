package main

import (
	"flag"
	"fmt"
	"github.com/hashicorp/hcl"
	"github.com/mittwald/mittnite/config"
	"github.com/mittwald/mittnite/files"
	"github.com/mittwald/mittnite/probe"
	"github.com/mittwald/mittnite/proc"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
)

type InitFlags struct {
	ConfigDir string
}

var (
	Version string
	Commit  string
	BuiltAt string
)

func main() {
	initFlags := InitFlags{}

	flag.StringVar(&initFlags.ConfigDir, "config-dir", "/etc/mittnite.d", "Directory from which to read configuration files")
	flag.Parse()

	log.Printf("mittnite process manager, version %s (commit %s), built at %s", Version, Commit, BuiltAt)
	log.Printf("looking for configuration files in %s", initFlags.ConfigDir)

	initFlags.ConfigDir = strings.TrimRight(initFlags.ConfigDir, "/")

	matches, err := filepath.Glob(initFlags.ConfigDir + "/*.hcl")
	if err != nil {
		panic(err)
	}

	if len(matches) == 0 {
		log.Fatalf("could not find any configuration files in %s", initFlags.ConfigDir)
	}

	ignitionConfig := config.IgnitionConfig{}

	for _, m := range matches {
		log.Printf("found config file: %s", m)

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
			log.Printf("received event %s", s.String())

			readinessSignals <- s
			probeSignals <- s
			procSignals <- s
		}
	}()

	probeHandler, _ := probe.NewProbeHandler(&ignitionConfig)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := probe.RunProbeServer(probeHandler, probeSignals)
		log.Printf("probe server stopped: %s", err)
	}()

	go func() {
		defer wg.Done()
		err := probeHandler.Wait(readinessSignals)
		if err != nil {
			panic(err)
		}

		err = proc.RunServices(&ignitionConfig, procSignals)
		log.Printf("service runner stopped: %s", err)
	}()

	wg.Wait()
}
