package probe

import (
	"fmt"
	"github.com/mittwald/mittnite/config"
	"log"
	"net/http"
	"time"
)

type httpGetProbe struct {
	scheme  string
	host    string
	path    string
	timeout string
}

func NewHttpProbe(cfg *config.HttpGetConfig) *httpGetProbe {
	cfg.Scheme = resolveEnv(cfg.Scheme)
	cfg.Host = resolveEnv(cfg.Host)
	cfg.Port = resolveEnv(cfg.Port)
	cfg.Path = resolveEnv(cfg.Path)
	cfg.Timeout = resolveEnv(cfg.Timeout)

	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}

	host := cfg.Host
	if cfg.Port != "" {
		host += ":" + cfg.Port
	}

	connCfg := httpGetProbe{
		scheme:  cfg.Scheme,
		host:    host,
		path:    cfg.Path,
		timeout: cfg.Timeout,
	}

	return &connCfg
}

func (h *httpGetProbe) Exec() error {
	var timeout = time.Second * 5
	if h.timeout != "" {
		duration, err := time.ParseDuration(h.timeout)
		if err == nil {
			timeout = duration
		} else {
			return fmt.Errorf("invalid timeout duration: %s", err)
		}
	}

	url := fmt.Sprintf("%s://%s%s", h.scheme, h.host, h.path)

	var client = &http.Client{
		Timeout: timeout,
	}
	res, err := client.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode >= 200 && res.StatusCode < 400 {
		log.Printf("http service '%s' is alive", url)
		return nil
	}

	return fmt.Errorf("http service '%s' returned status code %d", url, res.StatusCode)
}
