package config

import (
	"fmt"
	"github.com/hashicorp/hcl"
	"github.com/mittwald/mittnite/pkg/files"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"strings"
)

func (ignitionConfig *Ignition) GenerateFromConfigDir(configDir string) error {
	configDir = strings.TrimRight(configDir, "/")

	matches, err := findFilesInPath(configDir)
	if err != nil {
		return err
	}

	for _, m := range matches {
		log.Infof("found config file: %s", m)

		contents, err := ioutil.ReadFile(m)
		if err != nil {
			return err
		}

		err = hcl.Unmarshal(contents, ignitionConfig)
		if err != nil {
			err = fmt.Errorf("could not parse configuration file %s: %s", m, err.Error())
			return err
		}
	}

	err = files.RenderConfigurationFiles(ignitionConfig.Files)
	if err != nil {
		return err
	}

	return nil
}
