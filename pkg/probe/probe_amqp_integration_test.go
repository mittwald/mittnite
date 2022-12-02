// +build integration

package probe

import (
	"flag"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	amqpHost     = flag.String("amqp.host", svcHost("127.0.0.1", "amqp"), "AMQp integration server host")
	amqpPort     = flag.Uint("amqp.port", svcPort(15672, 5672), "AMQp integration server port")
	amqpVhost    = flag.String("amqp.vhost", "", "AMQp virtual host")
	amqpUsername = flag.String("amqp.username", "", "AMQp integration username")
	amqpPassword = flag.String("amqp.password", "", "AMQp integration password")
)

func TestAmqpProbeExecOk(t *testing.T) {
	subject := newAmqpIntegrationSubject()
	err := subject.Exec()

	assert.NoError(t, err, "Exec")
}

func newAmqpIntegrationSubject() *amqpProbe {
	port := strconv.FormatUint(uint64(*amqpPort), 10)

	return &amqpProbe{
		user:        *amqpUsername,
		password:    *amqpPassword,
		hostname:    *amqpHost,
		virtualHost: *amqpVhost,
		port:        port,
	}
}
