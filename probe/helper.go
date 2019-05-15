package probe

import (
	"os"
	"strings"
)

func resolveEnv(in string) string {
	if strings.HasPrefix(in, "ENV:") {
		return os.Getenv(in[4:])
	}
	return in
}
