package proc

import (
	"encoding/json"
	"net/http"
)

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
