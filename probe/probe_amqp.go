package probe

import (
	"fmt"
	"github.com/mittwald/mittnite/config"
	"github.com/streadway/amqp"
	"log"
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
}

func NewAmqpProbe(cfg *config.AmqpConfig) *amqpProbe {
	cfg.User = resolveEnv(cfg.User)
	cfg.Password = resolveEnv(cfg.Password)
	cfg.Hostname = resolveEnv(cfg.Hostname)
	cfg.VirtualHost = resolveEnv(cfg.VirtualHost)
	if cfg.VirtualHost == "" {
		cfg.VirtualHost = defaultVirtualHost
	}

	connCfg := amqpProbe{
		user:        cfg.User,
		password:    cfg.Password,
		hostname:    cfg.Hostname,
		virtualHost: cfg.VirtualHost,
	}

	return &connCfg
}

func (a *amqpProbe) Exec() error {
	u := url.URL{
		Scheme: "amqp",
		Host:   fmt.Sprintf("%s:%d", a.hostname, 5672),
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

	log.Println("amqp is alive")

	return nil
}
