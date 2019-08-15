package files

import (
	"fmt"
	"github.com/mittwald/mittnite/internal/types"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type templateData struct {
	Env    map[string]string
	Params map[string]interface{}
}

func RenderConfigurationFiles(configs []types.FileConfig) error {
	log.Info("generating configuration files")

	for i := range configs {
		err := RenderConfigurationFile(&configs[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func RenderConfigurationFile(cfg *types.FileConfig) error {
	if cfg.Template != "" {
		log.Infof("creating configuration file %s from template %s", cfg.Target, cfg.Template)

		tplContents, err := ioutil.ReadFile(cfg.Template)
		if err != nil {
			return err
		}

		tpl, err := template.New(cfg.Target).Parse(string(tplContents))
		if err != nil {
			return err
		}

		folderPath, err := filepath.Abs(filepath.Dir(cfg.Target))
		if err != nil {
			return err
		}
		err = os.MkdirAll(folderPath, os.ModePerm)
		if err != nil {
			return err
		}

		out, err := os.Create(cfg.Target)
		if err != nil {
			return err
		}

		defer out.Close()

		data := templateData{
			Env:    make(map[string]string),
			Params: cfg.Parameters,
		}

		for _, e := range os.Environ() {
			e := strings.SplitN(e, "=", 2)
			if len(e) > 1 {
				data.Env[e[0]] = e[1]
			}
		}

		err = tpl.Execute(out, &data)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("configuration file %s has no specified source", cfg.Target)
}
