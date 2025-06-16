package cmd

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var configDir string
var enableProfile bool

func init() {
	rootCmd.PersistentFlags().StringVarP(&configDir, "config-dir", "c", "/etc/mittnite.d", "set directory to where your .hcl-configs are located")
	rootCmd.PersistentFlags().BoolVar(&enableProfile, "profile", false, "enable pprof http server")
}

var rootCmd = &cobra.Command{
	Use:     "mittnite",
	Short:   "Mittnite - Smart init system for containers",
	Long:    "Mittnite is a small, but smart init system designed for usage as `ENTRYPOINT` in container images",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if enableProfile {
			go func() {
				// pprof handlers are auto-registered on the default ServeMux when imported.
				listener, err := net.Listen("tcp", "127.0.0.1:")
				if err != nil {
					log.Errorf("pprof server failed to listen: %v", err)
					return
				}
				log.Infof("Starting pprof server on http://%s/debug/pprof/", listener.Addr().String())
				if err = http.Serve(listener, nil); err != nil {
					log.Errorf("pprof server error: %v", err)
				}
			}()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Warn("Running 'mittnite' without any arguments - defaulting to 'up'. This behaviour may change in future releases!")
		up.Run(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
