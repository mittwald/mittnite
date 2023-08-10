package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl"
	log "github.com/sirupsen/logrus"
)

func (ignitionConfig *Ignition) GenerateFromConfigDir(configDir string) error {
	log.Infof("looking for configuration files in %s", configDir)

	configDir = strings.TrimRight(configDir, "/")

	matches, err := findInPath(configDir)
	if err != nil {
		return err
	}

	for _, m := range matches {
		log.Infof("found config file: %s", m)

		contents, err := os.ReadFile(m)
		if err != nil {
			return err
		}

		err = hcl.Unmarshal(contents, ignitionConfig)
		if err != nil {
			err = fmt.Errorf("could not parse configuration file %s: %s", m, err.Error())
			return err
		}
	}

	for _, job := range ignitionConfig.Jobs {
		if job.MaxAttempts_ != nil {
			log.Warnf("field max_attempts in job %s is deprecated in favor of maxAttempts", job.Name)
			job.MaxAttempts = job.MaxAttempts_
		}
	}

	return nil
}
