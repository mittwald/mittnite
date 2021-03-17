package probe

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
	log "github.com/sirupsen/logrus"
)

type redisProbe struct {
	addr     string
	password string
}

func NewRedisProbe(cfg *config.Redis) *redisProbe {
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Port = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Port), "6379", "port", "redis")

	return &redisProbe{
		addr:     fmt.Sprintf("%s:%s", cfg.Hostname, cfg.Port),
		password: cfg.Password,
	}
}

func (r *redisProbe) Exec() error {
	client := redis.NewClient(&redis.Options{
		Addr:     r.addr,
		Password: r.password,
	})
	defer client.Close()

	_, err := client.Ping().Result()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"kind": "probe", "name": "redis", "status": "alive", "host": r.addr}).Debug()

	return nil
}
