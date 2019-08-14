package probe

import (
	"fmt"
	"github.com/spf13/viper"
)

func resolveEnv(in string) string {
	// if strings.HasPrefix(in, "ENV:") {
	// 	return os.Getenv(in[4:])
	// }
	// return in
	env := viper.Get(in)

	if env == nil {
		return in
	}
	return fmt.Sprintf("%s", env)
}
