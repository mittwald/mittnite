package probe

import (
	"github.com/go-redis/redis"
	"github.com/mittwald/mittnite/config"
)

type redisProbe struct {
	addr     string
	password string
}

func NewRedisProbe(cfg *config.RedisConfig) *redisProbe {
	cfg.Host = resolveEnv(cfg.Host)
	cfg.Password = resolveEnv(cfg.Password)

	return &redisProbe{
		addr:     cfg.Host + ":6379",
		password: cfg.Password,
	}
}

func (r *redisProbe) Exec() error {
	client := redis.NewClient(&redis.Options{
		Addr:     r.addr,
		Password: r.password,
	})

	_, err := client.Ping().Result()
	return err
}
