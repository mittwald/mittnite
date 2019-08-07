package probe

import (
	"github.com/go-redis/redis"
	"github.com/mittwald/mittnite/config"
	"strconv"
)

type redisProbe struct {
	addr     string
	password string
}

func NewRedisProbe(cfg *config.RedisConfig) *redisProbe {
	cfg.Host.Url = resolveEnv(cfg.Host.Url)
	cfg.Password = resolveEnv(cfg.Password)
	// cfg.Port = resolveEnv(cfg.Port)

	return &redisProbe{
		addr:     cfg.Host.Url + ":" + strconv.Itoa(cfg.Host.Port),
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
