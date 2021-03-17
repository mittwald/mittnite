package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

type templateData struct {
	Env    map[string]string
	Params map[string]interface{}
}

func RenderFiles(configs []config.File) error {
	log.Info("generating configuration files")

	for i := range configs {
		err := renderFile(&configs[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func renderFile(cfg *config.File) error {
	if cfg.Template == "" {
		return fmt.Errorf("configuration file %s has no specified source", cfg.Target)
	}

	if cfg.Overwrite != nil && !*cfg.Overwrite {
		if _, err := os.Stat(cfg.Target); err == nil {
			log.Infof("skipping already existing configuration file %s", cfg.Target)
			return nil
		}
	}

	log.Infof("creating configuration file %s from template %s", cfg.Target, cfg.Template)

	tplContents, err := os.ReadFile(cfg.Template)
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

	defer func() { _ = out.Close() }()

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
