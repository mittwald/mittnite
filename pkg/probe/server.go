package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	cfg        *config.Ignition
	probes     map[string]Probe
	waitProbes map[string]Probe
}

func (h *Handler) Wait(interrupt chan os.Signal) error {
	log.Info("waiting for probe readiness")

	timer := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-timer.C:
			ready := true

			for i := range h.waitProbes {
				err := h.waitProbes[i].Exec()
				var pathErr *os.PathError
				if errors.As(err, &pathErr) {
					log.WithFields(log.Fields{"kind": "probe", "name": i, "err": err}).Fatal("path does not exist")
					return nil
				}
				if err != nil {
					log.WithFields(log.Fields{"kind": "probe", "name": i, "err": err}).Warn("not ready yet")
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

func (h *Handler) HandleStatus(res http.ResponseWriter, req *http.Request) {
	response := StatusResponse{
		Probes: make(map[string]*ProbeResult),
	}

	results := make(chan *ProbeResult, len(h.probes))
	timeout := time.NewTimer(1 * time.Second)

	for i := range h.probes {
		response.Probes[i] = &ProbeResult{i, false, "timed out"}

		go func(p Probe, name string) {
			err := p.Exec()

			if err != nil {
				results <- &ProbeResult{Name: name, OK: false, Message: err.Error()}
			} else {
				results <- &ProbeResult{Name: name, OK: true, Message: ""}
			}
		}(h.probes[i], i)
	}

	success := true

	for i := 0; i < len(h.probes); i++ {
		select {
		case result := <-results:
			response.Probes[result.Name] = result
			success = success && result.OK
		case <-timeout.C:
			success = false
			log.WithFields(log.Fields{"kind": "probe"}).Error("timed out")
			break
		}
	}

	res.Header().Set("Content-Type", "application/json")

	if !success {
		res.WriteHeader(503)
	}

	_ = json.NewEncoder(res).Encode(&response)
}

func NewProbeHandler(cfg *config.Ignition) (*Handler, error) {
	probes, err := buildProbesFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	waitProbes := filterWaitProbes(cfg, probes)

	handler := &Handler{cfg, probes, waitProbes}
	return handler, nil
}

func RunProbeServer(ph *Handler, signals chan os.Signal, probePort int) error {
	m := mux.NewRouter()
	m.Path("/status").HandlerFunc(ph.HandleStatus)

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", probePort),
		Handler: m,
	}

	go func() {
		for s := range signals {
			if s == syscall.SIGINT || s == syscall.SIGTERM {
				log.WithField("receivedSignal", s.String()).Error("shutting down monitoring server")
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

func filterWaitProbes(cfg *config.Ignition, probes map[string]Probe) map[string]Probe {
	result := make(map[string]Probe)
	for i := range cfg.Probes {
		if cfg.Probes[i].Wait {
			result[cfg.Probes[i].Name] = probes[cfg.Probes[i].Name]
		}
	}
	return result
}

func buildProbesFromConfig(cfg *config.Ignition) (map[string]Probe, error) {
	var errs []error

	result := make(map[string]Probe)
	for i := range cfg.Probes {
		p, err := newProbe(cfg.Probes[i])
		if err != nil {
			errs = append(errs, err)
		} else if p != nil {
			result[cfg.Probes[i].Name] = p
		}
	}

	var err error
	if len(errs) != 0 {
		err = fmt.Errorf("%+v", errs)
	}

	return result, err
}

func newProbe(p config.Probe) (Probe, error) {
	if p.Filesystem != "" {
		return &filesystemProbe{p.Filesystem}, nil
	} else if p.MySQL != nil {
		return NewMySQLProbe(p.MySQL), nil
	} else if p.Redis != nil {
		return NewRedisProbe(p.Redis), nil
	} else if p.MongoDB != nil {
		return NewMongoDBProbe(p.MongoDB)
	} else if p.Amqp != nil {
		return NewAmqpProbe(p.Amqp), nil
	} else if p.HTTP != nil {
		return NewHttpProbe(p.HTTP)
	} else if p.SMTP != nil {
		return NewSmtpProbe(p.SMTP), nil
	}

	return nil, nil
}
