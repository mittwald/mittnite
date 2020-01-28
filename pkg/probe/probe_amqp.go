package probe

import (
	"fmt"
	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"net/url"
)

const (
	defaultVirtualHost = "/"
)

type amqpProbe struct {
	user        string
	password    string
	hostname    string
	virtualHost string
	port        string
}

func NewAmqpProbe(cfg *config.Amqp) *amqpProbe {
	cfg.User = helper.ResolveEnv(cfg.User)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Port = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Port), "5672", "port", "amqp")
	cfg.VirtualHost = helper.ResolveEnv(cfg.VirtualHost)
	if cfg.VirtualHost == "" {
		cfg.VirtualHost = defaultVirtualHost
	}

	connCfg := amqpProbe{
		user:        cfg.User,
		password:    cfg.Password,
		hostname:    cfg.Hostname,
		virtualHost: cfg.VirtualHost,
		port:        cfg.Port,
	}

	return &connCfg
}

func (a *amqpProbe) Exec() error {
	u := url.URL{
		Scheme: "amqp",
		Host:   fmt.Sprintf("%s:%s", a.hostname, a.port),
		Path:   a.virtualHost,
	}

	if a.user != "" && a.password != "" {
		u.User = url.UserPassword(a.user, a.password)
	}

	conn, err := amqp.Dial(u.String())
	if err != nil {
		return fmt.Errorf("failed to dial amqp with url '%s': %s", u.String(), err.Error())
	}
	defer conn.Close()

	log.WithFields(log.Fields{"kind": "probe", "name": "amqp", "status": "alive", "host": fmt.Sprintf("%s:%s", a.hostname, a.port)}).Debug()

	return nil
}
