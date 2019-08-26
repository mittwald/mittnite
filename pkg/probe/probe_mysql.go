package probe

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/mittwald/mittnite/internal/config"
	"github.com/mittwald/mittnite/internal/helper"
	log "github.com/sirupsen/logrus"
)

type mySQLProbe struct {
	dsn string
}

func NewMySQLProbe(cfg *config.MySQL) *mySQLProbe {
	cfg.User = helper.ResolveEnv(cfg.User)
	cfg.Database = helper.ResolveEnv(cfg.Database)
	cfg.Password = helper.ResolveEnv(cfg.Password)
	cfg.Hostname = helper.ResolveEnv(cfg.Hostname)
	cfg.Port = helper.ResolveEnv(cfg.Port)

	connCfg := mysql.Config{
		User:   cfg.User,
		Passwd: cfg.Password,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		DBName: cfg.Database,
	}

	return &mySQLProbe{
		dsn: connCfg.FormatDSN(),
	}
}

func (m *mySQLProbe) Exec() error {
	db, err := sql.Open("mysql", m.dsn)
	if err != nil {
		return err
	}

	log.Info("connected")

	defer db.Close()
	r, err := db.Query("SELECT 1")
	if err != nil {
		return err
	}

	log.Info("selected successfully")

	r.Close()

	return nil
}
