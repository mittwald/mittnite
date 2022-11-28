package probe

import (
	"net"
	"net/smtp"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
	log "github.com/sirupsen/logrus"
)

type smtpProbe struct {
	addr string
}

func NewSmtpProbe(cfg *config.SMTP) *smtpProbe {
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Port = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Port), "25", "port", "smtp")

	return &smtpProbe{
		addr: net.JoinHostPort(cfg.Hostname, cfg.Port),
	}
}

func (s *smtpProbe) Exec() error {
	client, err := smtp.Dial(s.addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Noop(); err != nil {
		return err
	}

	if err := client.Quit(); err != nil {
		return err
	}

	log.WithFields(log.Fields{"kind": "probe", "name": "smtp", "status": "alive", "host": s.addr}).Debug()

	return nil
}
