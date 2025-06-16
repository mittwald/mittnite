package cmd

import (
	"net"
	"net/http"
	"net/http/pprof"

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
				mux := http.NewServeMux()
				mux.HandleFunc("/debug/pprof/", pprof.Index)
				mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
				mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
				mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
				mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

				listener, err := net.Listen("tcp", ":0")
				if err != nil {
					log.Errorf("pprof server failed to listen: %v", err)
					return
				}
				log.Infof("Starting pprof server on http://localhost%s/debug/pprof/", listener.Addr().String())
				err = http.Serve(listener, mux)
				if err != nil {
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
