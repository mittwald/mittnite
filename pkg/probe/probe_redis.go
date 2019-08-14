package probe

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/mittwald/mittnite/internal/helper"
	"github.com/mittwald/mittnite/internal/types"
	log "github.com/sirupsen/logrus"
)

type redisProbe struct {
	addr     string
	password string
}

func NewRedisProbe(cfg *types.RedisConfig) *redisProbe {
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Port = helper.ResolveEnv(cfg.Port)

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

	_, err := client.Ping().Result()
	if err != nil {
		return err
	}
	log.Info("redis is alive")
	return nil
}
