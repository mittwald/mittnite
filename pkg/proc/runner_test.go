package proc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeepRunningDoesNotDuplicateCrashLoopingJob(t *testing.T) {
	maxAttempts := -1 // unlimited retries
	jobConfig := config.JobConfig{
		BaseJobConfig: config.BaseJobConfig{
			Name:    "test-crashing-job",
			Command: "false", // exits immediately with error
		},
		MaxAttempts: &maxAttempts,
	}

	ignitionConfig := &config.Ignition{
		Jobs: []config.JobConfig{jobConfig},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(ctx, nil, true, ignitionConfig)
	require.NoError(t, runner.Init())

	runner.errChan = make(chan error, 16)
	runner.waitGroup = &sync.WaitGroup{}
	runner.waitGroup.Add(1) // keepRunning hold
	runner.exec()

	// Drain errors so goroutines don't block
	go func() {
		for range runner.errChan {
		}
	}()

	// Wait for the job to crash and enter CrashLooping phase
	require.Eventually(t, func() bool {
		return runner.jobs[0].GetPhase().Is(JobPhaseReasonCrashLooping)
	}, 10*time.Second, 50*time.Millisecond, "job should enter CrashLooping phase")

	jobCountBefore := len(runner.jobs)

	// Simulate the ticker firing — this must not spawn a duplicate goroutine
	runner.tick()

	assert.Equal(t, jobCountBefore, len(runner.jobs), "job count should not change for a CrashLooping job")
	assert.True(t, runner.jobs[0].GetPhase().Is(JobPhaseReasonCrashLooping), "phase should still be CrashLooping")
}

func TestTickDoesNotRestartStoppedJob(t *testing.T) {
	jobConfig := config.JobConfig{
		BaseJobConfig: config.BaseJobConfig{
			Name:    "test-stopped-job",
			Command: "true",
		},
	}

	ignitionConfig := &config.Ignition{
		Jobs: []config.JobConfig{jobConfig},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(ctx, nil, true, ignitionConfig)
	require.NoError(t, runner.Init())

	runner.errChan = make(chan error, 16)
	runner.waitGroup = &sync.WaitGroup{}

	// Simulate a job that was explicitly stopped via `mittnitectl job stop`.
	runner.jobs[0].SetPhase(JobPhaseReasonStopped)

	runner.tick()

	assert.True(t, runner.jobs[0].GetPhase().Is(JobPhaseReasonStopped), "explicitly stopped job must remain stopped")
}

func TestTickDoesNotRestartCompletedOneTimeJob(t *testing.T) {
	jobConfig := config.JobConfig{
		BaseJobConfig: config.BaseJobConfig{
			Name:    "test-onetime-job",
			Command: "true",
		},
		OneTime: true,
	}

	ignitionConfig := &config.Ignition{
		Jobs: []config.JobConfig{jobConfig},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(ctx, nil, true, ignitionConfig)
	require.NoError(t, runner.Init())

	runner.errChan = make(chan error, 16)
	runner.waitGroup = &sync.WaitGroup{}

	runner.jobs[0].SetPhase(JobPhaseReasonCompleted)

	runner.tick()

	assert.True(t, runner.jobs[0].GetPhase().Is(JobPhaseReasonCompleted), "completed one-time job must stay completed")
}

func TestRunWaitsForJobShutdownOnCancel(t *testing.T) {
	jobConfig := config.JobConfig{
		BaseJobConfig: config.BaseJobConfig{
			Name:    "test-shutdown-job",
			Command: "sleep",
			Args:    []string{"30"},
		},
	}

	ignitionConfig := &config.Ignition{
		Jobs: []config.JobConfig{jobConfig},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(ctx, nil, false, ignitionConfig)
	require.NoError(t, runner.Init())

	runResult := make(chan error, 1)
	go func() { runResult <- runner.Run() }()

	require.Eventually(t, func() bool {
		return runner.jobs[0].GetPhase().Is(JobPhaseReasonStarted)
	}, 5*time.Second, 20*time.Millisecond, "job should start")

	cancel()

	select {
	case err := <-runResult:
		assert.ErrorIs(t, err, context.Canceled, "cancellation should surface as context.Canceled, not a job error")
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not return after context cancellation")
	}

	assert.True(t, runner.jobs[0].GetPhase().Is(JobPhaseReasonStopped),
		"job goroutine should have terminated the process before the runner returned")
}

func TestRunKeepRunningOutlivesFinishedJobs(t *testing.T) {
	jobConfig := config.JobConfig{
		BaseJobConfig: config.BaseJobConfig{
			Name:    "test-onetime-job",
			Command: "true",
		},
		OneTime: true,
	}

	ignitionConfig := &config.Ignition{
		Jobs: []config.JobConfig{jobConfig},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(ctx, nil, true, ignitionConfig)
	require.NoError(t, runner.Init())

	runResult := make(chan error, 1)
	go func() { runResult <- runner.Run() }()

	require.Eventually(t, func() bool {
		return runner.jobs[0].GetPhase().Is(JobPhaseReasonCompleted)
	}, 5*time.Second, 20*time.Millisecond, "one-time job should complete")

	select {
	case err := <-runResult:
		t.Fatalf("runner returned despite keepRunning: %v", err)
	case <-time.After(300 * time.Millisecond):
		// still running, as it should be
	}

	cancel()

	select {
	case err := <-runResult:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not return after context cancellation")
	}
}
