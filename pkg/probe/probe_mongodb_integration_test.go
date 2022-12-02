// +build integration

package probe

import (
	"flag"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	mongodbHost     = flag.String("mongodb.host", svcHost("127.0.0.1", "mongo"), "MongoDB integration server host")
	mongodbPort     = flag.Uint("mongodb.port", svcPort(17017, 27017), "MongoDB integration server port")
	mongodbDatabase = flag.String("mongodb.database", "integration", "MongoDB integration database")
)

func TestMongoDBProbeExecOk(t *testing.T) {
	subject := newMongoDBIntegrationSubject()
	err := subject.Exec()

	assert.NoError(t, err, "Exec")
}

func newMongoDBIntegrationSubject() *mongoDBProbe {
	addr := fmt.Sprintf("%s:%d", *mongodbHost, *mongodbPort)
	location := &url.URL{
		Scheme: "mongodb",
		Host:   addr,
		Path:   *mongodbDatabase,
	}

	return &mongoDBProbe{
		url: location,
	}
}
