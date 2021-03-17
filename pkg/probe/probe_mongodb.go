package probe

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type mongoDBProbe struct {
	url *url.URL
}

func NewMongoDBProbe(cfg *config.MongoDB) (*mongoDBProbe, error) {
	u := &url.URL{}
	var err error

	cfg.URL = helper.ResolveEnv(cfg.URL)
	if cfg.URL != "" {
		u, err = url.Parse(cfg.URL)
	} else {
		log.WithFields(log.Fields{"kind": "probe", "name": "mongodb"}).Warn("probe is now configured by 'url', this configuration will explode in a future release")

		cfg.User = helper.ResolveEnv(cfg.User)
		cfg.Password = helper.ResolveEnv(cfg.Password)
		cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
		cfg.Database = helper.ResolveEnv(cfg.Database)
		cfg.Port = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Port), "27017", "port", "mongodb")
		cfg.ReplicaSetName = helper.ResolveEnv(cfg.ReplicaSetName)
		cfg.AuthenticationDatabase = helper.ResolveEnv(cfg.AuthenticationDatabase)
		cfg.AuthenticationMechanism = helper.ResolveEnv(cfg.AuthenticationMechanism)
		cfg.GssapiServiceName = helper.ResolveEnv(cfg.GssapiServiceName)

		q := url.Values{}

		helper.AddValueToURLValuesIfNotEmpty("replicaSet", cfg.ReplicaSetName, &q)
		helper.AddValueToURLValuesIfNotEmpty("gssapiServiceName", cfg.GssapiServiceName, &q)
		helper.AddValueToURLValuesIfNotEmpty("authMechanism", cfg.AuthenticationMechanism, &q)

		u.Scheme = "mongodb"
		u.Host = fmt.Sprintf("%s:%s", cfg.Hostname, cfg.Port)
		u.Path = cfg.Database
		u.RawQuery = q.Encode()

		if cfg.User != "" && cfg.Password != "" {
			helper.AddValueToURLValuesIfNotEmpty("authSource", cfg.AuthenticationDatabase, &q)
			u.User = url.UserPassword(cfg.User, cfg.Password)
		}
	}

	connCfg := mongoDBProbe{
		url: u,
	}

	return &connCfg, errors.WithStack(err)
}

func (m *mongoDBProbe) Exec() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(m.url.String()))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = client.Disconnect(ctx) }()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"kind": "probe", "name": "mongodb", "status": "alive", "host": m.url.Host}).Debug()

	return nil
}
