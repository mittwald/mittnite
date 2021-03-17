package helper

import (
	"net/url"
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

func SetDefaultStringIfEmpty(current, fallback, key, probeType string) string {
	if len(current) == 0 {
		log.WithFields(log.Fields{"kind": "cfg", "name": probeType, "key": key, "default": fallback}).Info("no input for probe specified, assuming default")
		return fallback
	}
	return current
}

func AddValueToURLValuesIfNotEmpty(key, value string, q *url.Values) {
	if len(value) > 0 {
		q.Add(key, value)
	}
}
