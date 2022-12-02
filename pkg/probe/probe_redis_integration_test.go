// +build integration

package probe

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	redisHost = flag.String("redis.host", svcHost("127.0.0.1", "redis"), "Redis integration server host")
	redisPort = flag.Uint("redis.port", svcPort(16379, 6379), "Redis integration server port")
)

func TestRedisProbeExecOk(t *testing.T) {
	subject := newRedisIntegrationSubject()
	err := subject.Exec()

	assert.NoError(t, err, "Exec")
}

func newRedisIntegrationSubject() *redisProbe {
	addr := fmt.Sprintf("%s:%d", *redisHost, *redisPort)

	return &redisProbe{
		addr: addr,
	}
}
