package probe

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/mittwald/mittnite/internal/types"
	log "github.com/sirupsen/logrus"
)

type mySQLProbe struct {
	dsn string
}

func NewMySQLProbe(cfg *types.MySQLConfig) *mySQLProbe {
	cfg.User = resolveEnv(cfg.User)
	cfg.Database = resolveEnv(cfg.Database)
	cfg.Password = resolveEnv(cfg.Password)
	cfg.URL = resolveEnv(cfg.URL)
	cfg.Port = resolveEnv(cfg.Port)

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
