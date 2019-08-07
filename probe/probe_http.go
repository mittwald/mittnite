package probe

import (
	"fmt"
	"github.com/mittwald/mittnite/config"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strconv"
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
	cfg.Host.Url = resolveEnv(cfg.Host.Url)
	// cfg.Host.Port = resolveEnv(cfg.Host.Port)
	cfg.Path = resolveEnv(cfg.Path)
	cfg.Timeout = resolveEnv(cfg.Timeout)

	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}

	host := cfg.Host.Url
	if cfg.Host.Port != 0 {
		host += ":" + strconv.Itoa(cfg.Host.Port)
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

	u := url.URL{
		Scheme: h.scheme,
		Host:   h.host,
		Path:   h.path,
	}
	urlStr := u.String()

	var client = &http.Client{
		Timeout: timeout,
	}
	res, err := client.Get(urlStr)
	if err != nil {
		return err
	}

	if res.StatusCode >= 200 && res.StatusCode < 400 {
		log.Infof("http service '%s' is alive", urlStr)
		return nil
	}

	return fmt.Errorf("http service '%s' returned status code %d", urlStr, res.StatusCode)
}
