package probe

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
	log "github.com/sirupsen/logrus"
)

type httpProbe struct {
	method  string
	scheme  string
	host    string
	path    string
	payload string
	headers map[string]string
	timeout time.Duration
	status  *regexp.Regexp
}

func NewHttpProbe(cfg *config.HTTP) (*httpProbe, error) {
	cfg.Method = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Method), "GET", "method", "http")
	cfg.Scheme = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Scheme), "http", "scheme", "http")
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Port = helper.ResolveEnv(cfg.Port)
	cfg.Path = helper.ResolveEnv(cfg.Path)
	cfg.Timeout = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Timeout), "5s", "timeout", "http")
	cfg.ExpectStatus = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.ExpectStatus), `(1|2|3)\d\d\s`, "expectStatus", "http")

	method := strings.ToUpper(cfg.Method)
	host := cfg.Hostname
	if cfg.Port != "" {
		host = net.JoinHostPort(cfg.Hostname, cfg.Port)
	}

	status, err := regexp.Compile(cfg.ExpectStatus)
	if err != nil {
		return nil, fmt.Errorf("invalid HTTP status line regexp: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout duration: %w", err)
	}

	connCfg := &httpProbe{
		method:  method,
		scheme:  cfg.Scheme,
		host:    host,
		path:    cfg.Path,
		status:  status,
		timeout: timeout,
		payload: cfg.Payload,
		headers: cfg.Headers,
	}

	return connCfg, nil
}

func (h *httpProbe) Exec() error {
	u := url.URL{
		Scheme: h.scheme,
		Host:   h.host,
		Path:   h.path,
	}
	urlStr := u.String()
	client := &http.Client{
		Timeout: h.timeout,
	}

	data := strings.NewReader(h.payload)
	req, err := http.NewRequest(h.method, u.String(), data)
	if err != nil {
		return err
	}

	for k, v := range h.headers {
		req.Header.Set(k, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if !h.status.MatchString(res.Status) {
		return fmt.Errorf("http service %q returned status %q", urlStr, res.Status)
	}

	log.WithFields(log.Fields{"kind": "probe", "name": "http", "status": "alive", "host": urlStr}).Debug()
	return nil
}
