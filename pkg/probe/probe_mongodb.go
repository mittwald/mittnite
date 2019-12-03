package probe

import (
	"context"
	"fmt"
	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
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

func NewMongoDBProbe(cfg *config.MongoDB) *mongoDBProbe {
	cfg.User = helper.ResolveEnv(cfg.User)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Database = helper.ResolveEnv(cfg.Database)
	cfg.Port = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Port), "27017")

	connCfg := mongoDBProbe{
		user:     cfg.User,
		password: cfg.Password,
		hostname: cfg.Hostname,
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
	if err != nil {
		return err
	}
	defer func(){_ = client.Disconnect(ctx)}()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"kind": "probe", "name": "mongodb", "status": "alive", "host": fmt.Sprintf("%s:%s", m.hostname, m.port)}).Debug()

	return nil
}
