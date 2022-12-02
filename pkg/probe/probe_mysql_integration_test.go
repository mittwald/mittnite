// +build integration

package probe

import (
	"flag"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"

	"github.com/stretchr/testify/assert"
)

var (
	mysqlHost     = flag.String("mysql.host", svcHost("127.0.0.1", "mysql"), "MySQL integration server host")
	mysqlPort     = flag.Uint("mysql.port", svcPort(13306, 3306), "MySQL integration server port")
	mysqlUsername = flag.String("mysql.username", "tester", "MySQL integration username")
	mysqlPassword = flag.String("mysql.password", "integration_test", "MySQL integration password")
	mysqlDatabase = flag.String("mysql.database", "integration", "MySQL integration database")
)

func TestMysqlProbeExecOk(t *testing.T) {
	subject := newMysqlIntegrationSubject()
	err := subject.Exec()

	assert.NoError(t, err, "Exec")
}

func newMysqlIntegrationSubject() *mySQLProbe {
	addr := fmt.Sprintf("%s:%d", *mysqlHost, *mysqlPort)
	conf := mysql.Config{
		User:                 *mysqlUsername,
		Passwd:               *mysqlPassword,
		Net:                  "tcp",
		Addr:                 addr,
		DBName:               *mysqlDatabase,
		AllowNativePasswords: true,
	}

	return &mySQLProbe{
		dsn: conf.FormatDSN(),
	}
}
