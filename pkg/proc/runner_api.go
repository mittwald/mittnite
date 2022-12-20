package proc

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
)

func (r *Runner) startApi() error {
	return r.startApiV1()
}

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
				job = r.findCommonIgnitionJobByName(jobName)
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

func (r *Runner) apiMiddleWare(req *http.Request) int {
	jobName, ok := mux.Vars(req)["job"]
	if !ok {
		return http.StatusBadRequest
	}

	existingJob := r.findCommonJobByName(jobName)
	newJob := r.findCommonIgnitionJobByName(jobName)

	if !r.jobExistsAndIsControllable(existingJob) && !r.jobExistsAndIsControllable(newJob) {
		return http.StatusNotFound
	}

	return http.StatusOK
}
