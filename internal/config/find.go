package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findInPath(configDir string) ([]string, error) {
	var matches []string

	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), "hcl") {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return matches, err
	}

	if len(matches) == 0 {
		return matches, fmt.Errorf("could not find any configuration files in %s", configDir)
	}

	return matches, nil
}
