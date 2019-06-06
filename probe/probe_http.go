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
	url string
}

func NewHttpProbe(cfg *config.HttpGetConfig) *httpGetProbe {
	cfg.Url = resolveEnv(cfg.Url)

	connCfg := httpGetProbe{
		url: cfg.Url,
	}

	return &connCfg
}

func (h *httpGetProbe) Exec() error {
	var client = &http.Client{
		Timeout: time.Second * 5,
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
