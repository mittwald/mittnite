// +build integration

package probe

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	smtpHost = flag.String("smtp.host", svcHost("127.0.0.1", "smtp"), "SMTP integration server host")
	smtpPort = flag.Uint("smtp.port", svcPort(12525, 1025), "SMTP integration server port")
)

func TestSmtpProbeExecOk(t *testing.T) {
	subject := newSmtpIntegrationSubject()
	err := subject.Exec()

	assert.NoError(t, err, "Exec")
}

func newSmtpIntegrationSubject() *smtpProbe {
	addr := fmt.Sprintf("%s:%d", *smtpHost, *smtpPort)

	return &smtpProbe{
		addr: addr,
	}
}
