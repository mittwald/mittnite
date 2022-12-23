package proc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
	"time"
)

func (r *Runner) startApiV1() error {
	if r.api == nil {
		return nil
	}

	r.api.RegisterMiddlewareFuncs(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			jobName, ok := mux.Vars(req)["job"]
			if !ok {
				http.Error(w, "job parameter is missing", http.StatusBadRequest)
				return
			}

			job := r.findCommonJobByName(jobName)
			if job == nil {
				var err error
				job, err = r.findCommonIgnitionJobByName(jobName)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			if r.jobExistsAndIsControllable(job) {
				r.addJobIfNotExists(job)
				next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), contextKeyJob, job)))
				return
			}

			http.Error(w, fmt.Sprintf("job %q not found or is not controllable", jobName), http.StatusNotFound)
		})
	})

	r.api.RegisterHandler("/v1/job/{job}/start", []string{http.MethodPost}, r.apiV1StartJob)
	r.api.RegisterHandler("/v1/job/{job}/restart", []string{http.MethodPost}, r.apiV1RestartJob)
	r.api.RegisterHandler("/v1/job/{job}/stop", []string{http.MethodPost}, r.apiV1StopJob)
	r.api.RegisterHandler("/v1/job/{job}/status", []string{http.MethodGet}, r.apiV1JobStatus)
	r.api.RegisterHandler("/v1/job/{job}/logs", []string{http.MethodGet}, r.apiV1JobLogs)
	return r.api.Start()
}

func (r *Runner) apiV1StartJob(writer http.ResponseWriter, req *http.Request) {
	job := req.Context().Value(contextKeyJob).(*CommonJob)
	if !job.IsRunning() {
		r.startJob(job)
	}
	writer.WriteHeader(http.StatusOK)
}

func (r *Runner) apiV1RestartJob(writer http.ResponseWriter, req *http.Request) {
	job := req.Context().Value(contextKeyJob).(*CommonJob)
	if !job.IsRunning() {
		r.startJob(job)
	} else {
		job.Restart()
	}
	writer.WriteHeader(http.StatusOK)
}

func (r *Runner) apiV1StopJob(writer http.ResponseWriter, req *http.Request) {
	job := req.Context().Value(contextKeyJob).(*CommonJob)
	job.Stop()
	r.removeJob(job)
	writer.WriteHeader(http.StatusOK)
}

func (r *Runner) apiV1JobStatus(writer http.ResponseWriter, req *http.Request) {
	job := req.Context().Value(contextKeyJob).(*CommonJob)
	out, err := json.Marshal(job.Status())
	if err != nil {
		http.Error(writer, "failed to get job status", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(out)
	writer.WriteHeader(http.StatusOK)
}

func (r *Runner) apiV1JobLogs(writer http.ResponseWriter, req *http.Request) {
	conn, err := r.api.upgrader.Upgrade(writer, req, nil)
	if err != nil {
		http.Error(writer, "failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	job := req.Context().Value(contextKeyJob).(*CommonJob)
	if len(job.Config.Stdout) == 0 && len(job.Config.Stderr) == 0 {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("neither stdout, nor stderr is defined for this job"))
		return
	}

	streamCtx, cancel := context.WithCancel(context.Background())
	outChan := make(chan []byte)
	errChan := make(chan error)
	defer func() {
		cancel()
		close(outChan)
		close(errChan)
	}()

	// handle client disconnects
	go func() {
		_, _, err := conn.ReadMessage()
		if err != nil {
			cancel()
		}
	}()

	follow := strings.ToLower(req.FormValue("follow")) == "true"
	go job.StreamStdOutAndStdErr(streamCtx, outChan, errChan, follow)

	for {
		select {
		case logLine := <-outChan:
			if err := conn.WriteMessage(websocket.TextMessage, logLine); err != nil {
				break
			}

		case err = <-errChan:
			if errors.Is(err, io.EOF) {
				err = conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "EOF"),
					time.Now().Add(time.Second),
				)
				if err == nil {
					return
				}
			}
			log.WithField("job.name", job.Config.Name).
				Error(fmt.Sprintf("error during logs streaming: %s", err.Error()))
			break

		case <-streamCtx.Done():
			return
		}
	}
}
