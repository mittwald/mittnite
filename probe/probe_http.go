package probe

import (
	"errors"
	"fmt"
	"github.com/mittwald/mittnite/config"
	"log"
	"net/http"
	"time"
)

type httpGetProbe struct {
	url     string
	timeout string
}

func NewHttpProbe(cfg *config.HttpGetConfig) *httpGetProbe {
	cfg.URL = resolveEnv(cfg.URL)
	cfg.Timeout = resolveEnv(cfg.Timeout)

	connCfg := httpGetProbe{
		url:     cfg.URL,
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
			return errors.New(fmt.Sprintf("invalid timeout duration: %s", err))
		}
	}

	var client = &http.Client{
		Timeout: timeout,
	}
	res, err := client.Get(h.url)
	if err != nil {
		return err
	}

	if res.StatusCode >= 200 && res.StatusCode < 400 {
		log.Printf("http service '%s' is alive", h.url)
		return nil
	}

	return errors.New(fmt.Sprintf("http service returned status code %d", res.StatusCode))
}
