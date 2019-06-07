package probe

import (
	"context"
	"fmt"
	"github.com/mittwald/mittnite/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/url"
	"time"
)

type mongoDBProbe struct {
	user     string
	password string
	hostname string
	database string
}

func NewMongoDBProbe(cfg *config.MongoDBConfig) *mongoDBProbe {
	cfg.User = resolveEnv(cfg.User)
	cfg.Password = resolveEnv(cfg.Password)
	cfg.Host = resolveEnv(cfg.Host)
	cfg.Database = resolveEnv(cfg.Database)

	connCfg := mongoDBProbe{
		user:     cfg.User,
		password: cfg.Password,
		hostname: cfg.Host,
		database: cfg.Database,
	}

	return &connCfg
}

func (m *mongoDBProbe) Exec() error {
	u := url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%d", m.hostname, 27017),
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

	log.Println("mongodb is alive")

	return nil
}
