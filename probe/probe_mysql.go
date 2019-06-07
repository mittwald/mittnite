package probe

import (
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"github.com/mittwald/mittnite/config"
	"log"
)

type mySQLProbe struct {
	dsn string
}

func NewMySQLProbe(cfg *config.MySQLConfig) *mySQLProbe {
	cfg.User = resolveEnv(cfg.User)
	cfg.Database = resolveEnv(cfg.Database)
	cfg.Password = resolveEnv(cfg.Password)
	cfg.Host = resolveEnv(cfg.Host)

	connCfg := mysql.Config{
		User:   cfg.User,
		Passwd: cfg.Password,
		Net:    "tcp",
		Addr:   cfg.Host + ":3306",
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

	log.Println("connected")

	defer db.Close()
	r, err := db.Query("SELECT 1")
	if err != nil {
		return err
	}

	log.Println("selected successfully")

	r.Close()

	return nil
}
