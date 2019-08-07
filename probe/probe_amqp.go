package probe

import (
	"fmt"
	"github.com/mittwald/mittnite/config"
	"github.com/streadway/amqp"
	"log"
	"net/url"
	"strconv"
)

const (
	defaultVirtualHost = "/"
)

type amqpProbe struct {
	user        string
	password    string
	hostname    string
	virtualHost string
	port        int
}

func NewAmqpProbe(cfg *config.AmqpConfig) *amqpProbe {
	cfg.Credentials.User = resolveEnv(cfg.Credentials.User)
	cfg.Credentials.Password = resolveEnv(cfg.Credentials.Password)
	cfg.Host.Url = resolveEnv(cfg.Host.Url)
	cfg.VirtualHost = resolveEnv(cfg.VirtualHost)
	if cfg.VirtualHost == "" {
		cfg.VirtualHost = defaultVirtualHost
	}

	connCfg := amqpProbe{
		user:        cfg.Credentials.User,
		password:    cfg.Credentials.Password,
		hostname:    cfg.Host.Url,
		virtualHost: cfg.VirtualHost,
		port:        cfg.Host.Port,
	}

	return &connCfg
}

func (a *amqpProbe) Exec() error {
	u := url.URL{
		Scheme: "amqp",
		Host:   fmt.Sprintf("%s:%d", a.hostname, strconv.Itoa(a.port)),
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
