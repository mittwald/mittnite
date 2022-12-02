// +build integration

package probe

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	httpHost = flag.String("http.host", svcHost("127.0.0.1", "http"), "HTTP integration server host")
	httpPort = flag.Uint("http.port", svcPort(18080, 80), "HTTP integration server port")
)

func TestHttpProbeExecOk(t *testing.T) {
	subject := newHttpIntegrationSubject("/anything")
	err := subject.Exec()

	assert.NoError(t, err, "Exec")
}

func TestHttpProbeExecErrorStatusCode(t *testing.T) {
	subject := newHttpIntegrationSubject("/status/503")
	err := subject.Exec()

	assert.ErrorContains(t, err, "returned status code", "Exec")
}

func newHttpIntegrationSubject(path string) *httpGetProbe {
	host := fmt.Sprintf("%s:%d", *httpHost, *httpPort)

	return &httpGetProbe{
		scheme:  "http",
		host:    host,
		path:    path,
		timeout: "5s",
	}
}
