package probe

import (
	"fmt"
	"github.com/mittwald/mittnite/internal/helper"
	"github.com/mittwald/mittnite/internal/types"
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

func NewAmqpProbe(cfg *types.AmqpConfig) *amqpProbe {
	cfg.User = helper.ResolveEnv(cfg.User)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Port = helper.ResolveEnv(cfg.Port)
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

	log.Info("amqp is alive")

	return nil
}
