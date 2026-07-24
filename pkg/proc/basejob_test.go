package proc

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/stretchr/testify/require"
)

func startTestJob(t *testing.T) (*baseJob, chan error) {
	t.Helper()

	job := &baseJob{}
	err := job.init(&config.BaseJobConfig{
		Name:    "test-job",
		Command: "sleep",
		Args:    []string{"30"},
	})
	require.NoError(t, err)

	// receiving from the process channel synchronizes with startOnce having
	// set job.cmd, so the job can safely be signaled afterwards
	p := make(chan *os.Process, 1)
	errChan := make(chan error, 1)
	go func() {
		errChan <- job.startOnce(context.Background(), p)
	}()

	select {
	case proc := <-p:
		require.NotNil(t, proc, "process should be running")
	case <-time.After(5 * time.Second):
		t.Fatal("process did not start")
	}

	return job, errChan
}

func TestStartOnceReportsExpectedStopAsIntentional(t *testing.T) {
	job, errChan := startTestJob(t)

	job.stopExpected.Store(true)
	job.Signal(syscall.SIGTERM)

	select {
	case err := <-errChan:
		require.ErrorIs(t, err, ProcessStoppedIntentionallyError)
	case <-time.After(5 * time.Second):
		t.Fatal("startOnce did not return")
	}
}

func TestStartOnceReportsUnexpectedSignalAsError(t *testing.T) {
	job, errChan := startTestJob(t)

	job.Signal(syscall.SIGTERM)

	select {
	case err := <-errChan:
		require.Error(t, err)
		require.NotErrorIs(t, err, ProcessStoppedIntentionallyError)
		require.EqualError(t, err, "signal: terminated")
	case <-time.After(5 * time.Second):
		t.Fatal("startOnce did not return")
	}
}
