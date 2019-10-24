package helper

import (
	"os"
	"strings"
	log "github.com/sirupsen/logrus"
)

func ResolveEnv(in string) string {
	if strings.HasPrefix(in, "ENV:") {
		return os.Getenv(in[4:])
	}
	return in
}

func SetDefaultPort(port string, defaultPort string) string {
	if len(port) == 0 {
		log.Infof("No port specified or env variable not found, assuming default port %s", defaultPort)
		return defaultPort
	}
	return port
}