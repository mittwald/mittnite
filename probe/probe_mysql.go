package probe

import (
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"github.com/mittwald/mittnite/config"
	log "github.com/sirupsen/logrus"
	"strconv"
)

type mySQLProbe struct {
	dsn string
}

func NewMySQLProbe(cfg *config.MySQLConfig) *mySQLProbe {
	cfg.Credentials.User = resolveEnv(cfg.Credentials.User)
	cfg.Database = resolveEnv(cfg.Database)
	cfg.Credentials.Password = resolveEnv(cfg.Credentials.Password)
	cfg.Host.Url = resolveEnv(cfg.Host.Url)

	connCfg := mysql.Config{
		User:   cfg.Credentials.User,
		Passwd: cfg.Credentials.Password,
		Net:    "tcp",
		Addr:   cfg.Host.Url + ":" + strconv.Itoa(cfg.Host.Port),
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
