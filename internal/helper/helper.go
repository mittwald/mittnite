package helper

import (
	"os"
	"strings"
)

func ResolveEnv(in string) string {
	if strings.HasPrefix(in, "ENV:") {
		return os.Getenv(in[4:])
	}
	return in
}
