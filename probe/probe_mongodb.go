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
	port     int
}

func NewMongoDBProbe(cfg *config.MongoDBConfig) *mongoDBProbe {
	cfg.Credentials.User = resolveEnv(cfg.Credentials.User)
	cfg.Credentials.Password = resolveEnv(cfg.Credentials.Password)
	cfg.Host.Url = resolveEnv(cfg.Host.Url)
	cfg.Database = resolveEnv(cfg.Database)

	connCfg := mongoDBProbe{
		user:     cfg.Credentials.User,
		password: cfg.Credentials.Password,
		hostname: cfg.Host.Url,
		database: cfg.Database,
		port:     cfg.Host.Port,
	}

	return &connCfg
}

func (m *mongoDBProbe) Exec() error {
	u := url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%d", m.hostname, m.port),
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
