package proc

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
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
