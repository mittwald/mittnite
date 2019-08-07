package probe

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/mittwald/mittnite/config"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"syscall"
	"time"
)

type ProbeHandler struct {
	cfg *config.IgnitionConfig

	probes     map[string]Probe
	waitProbes map[string]Probe
}

func (s *ProbeHandler) Wait(interrupt chan os.Signal) error {
	log.Info("waiting for probe readiness")

	timer := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-timer.C:
			ready := true

			for i := range s.waitProbes {
				err := s.waitProbes[i].Exec()
				if err != nil {
					log.Warn("probe %s is not yet ready: %s", i, err)
					ready = false
				}
			}

			if ready {
				return nil
			}
		case s := <-interrupt:
			if s == syscall.SIGTERM || s == syscall.SIGINT {
				return errors.New("readiness interrupted")
			}
		}
	}
}

func (s *ProbeHandler) HandleStatus(res http.ResponseWriter, req *http.Request) {
	response := StatusResponse{
		Probes: make(map[string]*ProbeResult),
	}

	results := make(chan *ProbeResult, len(s.probes))
	timeout := time.NewTimer(1 * time.Second)

	for i := range s.probes {
		response.Probes[i] = &ProbeResult{i, false, "timed out"}

		go func(p Probe, name string) {
			err := p.Exec()

			if err != nil {
				results <- &ProbeResult{Name: name, OK: false, Message: err.Error()}
			} else {
				results <- &ProbeResult{Name: name, OK: true, Message: ""}
			}
		}(s.probes[i], i)
	}

	success := true

	for i := 0; i < len(s.probes); i++ {
		select {
		case result := <-results:
			response.Probes[result.Name] = result
			success = success && result.OK
		case <-timeout.C:
			success = false
			log.Error("probe timed out")
			break
		}
	}

	res.Header().Set("Content-Type", "application/json")

	if !success {
		res.WriteHeader(503)
	}

	_ = json.NewEncoder(res).Encode(&response)
}

func NewProbeHandler(cfg *config.IgnitionConfig) (*ProbeHandler, error) {
	probes := buildProbesFromConfig(cfg)
	waitProbes := filterWaitProbes(cfg, probes)

	handler := &ProbeHandler{cfg, probes, waitProbes}
	return handler, nil
}

func RunProbeServer(ph *ProbeHandler, signals chan os.Signal) error {
	m := mux.NewRouter()
	m.Path("/status").HandlerFunc(ph.HandleStatus)

	server := http.Server{
		Addr:    ":9102",
		Handler: m,
	}

	go func() {
		for s := range signals {
			if s == syscall.SIGINT || s == syscall.SIGTERM {
				log.Printf("shutting down monitoring server after receiving %s", s.String())
				_ = server.Shutdown(context.Background())
			}
		}
	}()

	err := server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func filterWaitProbes(cfg *config.IgnitionConfig, probes map[string]Probe) map[string]Probe {
	result := make(map[string]Probe)
	for i := range cfg.Probes {
		if cfg.Probes[i].Wait {
			result[cfg.Probes[i].Name] = probes[cfg.Probes[i].Name]
		}
	}
	return result
}

func buildProbesFromConfig(cfg *config.IgnitionConfig) map[string]Probe {
	result := make(map[string]Probe)
	for i := range cfg.Probes {
		if cfg.Probes[i].Filesystem != "" {
			result[cfg.Probes[i].Name] = &filesystemProbe{cfg.Probes[i].Filesystem}
		} else if cfg.Probes[i].MySQL != nil {
			result[cfg.Probes[i].Name] = NewMySQLProbe(cfg.Probes[i].MySQL)
		} else if cfg.Probes[i].Redis != nil {
			result[cfg.Probes[i].Name] = NewRedisProbe(cfg.Probes[i].Redis)
		} else if cfg.Probes[i].MongoDB != nil {
			result[cfg.Probes[i].Name] = NewMongoDBProbe(cfg.Probes[i].MongoDB)
		} else if cfg.Probes[i].Amqp != nil {
			result[cfg.Probes[i].Name] = NewAmqpProbe(cfg.Probes[i].Amqp)
		} else if cfg.Probes[i].HTTP != nil {
			result[cfg.Probes[i].Name] = NewHttpProbe(cfg.Probes[i].HTTP)
		}
	}
	return result
}
