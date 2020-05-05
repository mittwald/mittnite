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
	hostname                string
	port                    string
	user                    string
	password                string
	database                string
	replicaSetName          string
	authenticationDatabase  string
	authenticationMechanism string
	gssapiServiceName       string
}

func NewMongoDBProbe(cfg *config.MongoDB) *mongoDBProbe {
	cfg.User = helper.ResolveEnv(cfg.User)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Database = helper.ResolveEnv(cfg.Database)
	cfg.Port = helper.SetDefaultStringIfEmpty(helper.ResolveEnv(cfg.Port), "27017", "port", "mongodb")
	cfg.ReplicaSetName = helper.ResolveEnv(cfg.ReplicaSetName)
	cfg.AuthenticationDatabase = helper.ResolveEnv(cfg.AuthenticationDatabase)
	cfg.AuthenticationMechanism = helper.ResolveEnv(cfg.AuthenticationMechanism)
	cfg.GssapiServiceName = helper.ResolveEnv(cfg.GssapiServiceName)

	connCfg := mongoDBProbe{
		hostname:                cfg.Hostname,
		port:                    cfg.Port,
		user:                    cfg.User,
		password:                cfg.Password,
		database:                cfg.Database,
		replicaSetName:          cfg.ReplicaSetName,
		authenticationDatabase:  cfg.AuthenticationDatabase,
		authenticationMechanism: cfg.AuthenticationMechanism,
		gssapiServiceName:       cfg.GssapiServiceName,
	}

	return &connCfg
}

func (m *mongoDBProbe) Exec() error {
	q := url.Values{}

	helper.AddValueToURLValuesIfNotEmpty("replicaSet", m.replicaSetName, &q)
	helper.AddValueToURLValuesIfNotEmpty("gssapiServiceName", m.gssapiServiceName, &q)
	helper.AddValueToURLValuesIfNotEmpty("authMechanism", m.authenticationMechanism, &q)
	if m.user != "" && m.password != "" {
		helper.AddValueToURLValuesIfNotEmpty("authSource", m.authenticationDatabase, &q)
	}

	u := url.URL{
		Scheme:   "mongodb",
		Host:     fmt.Sprintf("%s:%s", m.hostname, m.port),
		Path:     m.database,
		RawQuery: q.Encode(),
	}

	if m.user != "" && m.password != "" {
		u.User = url.UserPassword(m.user, m.password)
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
	defer func() { _ = client.Disconnect(ctx) }()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{"kind": "probe", "name": "mongodb", "status": "alive", "host": fmt.Sprintf("%s:%s", m.hostname, m.port)}).Debug()

	return nil
}
