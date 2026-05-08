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
	runner.jobs[0].GetPhase().Set(JobPhaseReasonStopped)

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

	runner.jobs[0].GetPhase().Set(JobPhaseReasonCompleted)

	runner.tick()

	assert.True(t, runner.jobs[0].GetPhase().Is(JobPhaseReasonCompleted), "completed one-time job must stay completed")
}
