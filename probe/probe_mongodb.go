package probe

import (
	"context"
	"fmt"
	"github.com/mittwald/mittnite/config"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"net/url"
	"time"
)

type mongoDBProbe struct {
	user     string
	password string
	hostname string
	database string
	port     string
}

func NewMongoDBProbe(cfg *config.MongoDBConfig) *mongoDBProbe {
	cfg.User = resolveEnv(cfg.User)
	cfg.Password = resolveEnv(cfg.Password)
	cfg.URL = resolveEnv(cfg.URL)
	cfg.Database = resolveEnv(cfg.Database)

	connCfg := mongoDBProbe{
		user:     cfg.User,
		password: cfg.Password,
		hostname: cfg.URL,
		database: cfg.Database,
		port:     cfg.Port,
	}

	return &connCfg
}

func (m *mongoDBProbe) Exec() error {
	u := url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%s", m.hostname, m.port),
		Path:   m.database,
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(u.String()))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	defer client.Disconnect(ctx)
	if err != nil {
		return err
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	log.Info("mongodb is alive")

	return nil
}
