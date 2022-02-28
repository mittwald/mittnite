package proc

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/mittwald/mittnite/internal/config"
)

const (
	ShutdownWaitingTimeSeconds = 10
)

type Runner struct {
	IgnitionConfig *config.Ignition

	jobs     []Job
	bootJobs []*BootJob
}

type baseJob struct {
	Config      *config.BaseJobConfig
	restartChan chan interface{}

	cmd *exec.Cmd
}

type BootJob struct {
	baseJob
	Config *config.BootJobConfig

	timeout time.Duration
}

type CommonJob struct {
	baseJob
	Config *config.JobConfig

	watchingFiles map[string]time.Time
}

type LazyJob struct {
	CommonJob

	process *os.Process

	lazyStartLock     sync.Mutex
	activeConnections uint32

	spinUpTimeout        time.Duration
	coolDownTimeout      time.Duration
	lastConnectionClosed time.Time
}

type Job interface {
	Init()
	Run(context.Context, chan<- error) error
	Watch()
	TearDown()
}

func NewCommonJob(c *config.JobConfig) *CommonJob {
	j := CommonJob{
		baseJob: baseJob{
			Config:      &c.BaseJobConfig,
			restartChan: make(chan interface{}),
		},
		Config: c,
	}

	return &j
}

func NewLazyJob(c *config.JobConfig) (*LazyJob, error) {
	j := LazyJob{
		CommonJob: *NewCommonJob(c),
	}

	if c.Laziness.SpinUpTimeout != "" {
		t, err := time.ParseDuration(c.Laziness.SpinUpTimeout)
		if err != nil {
			return nil, err
		}

		j.spinUpTimeout = t
	} else {
		j.spinUpTimeout = 5 * time.Second
	}

	if c.Laziness.CoolDownTimeout != "" {
		t, err := time.ParseDuration(c.Laziness.CoolDownTimeout)
		if err != nil {
			return nil, err
		}

		j.coolDownTimeout = t
	} else {
		j.coolDownTimeout = 15 * time.Minute
	}

	return &j, nil
}

func NewBootJob(c *config.BootJobConfig) (*BootJob, error) {
	bj := BootJob{
		baseJob: baseJob{
			Config:      &c.BaseJobConfig,
			restartChan: make(chan interface{}),
		},
		Config: c,
	}

	if ts := c.Timeout; ts != "" {
		t, err := time.ParseDuration(ts)
		if err != nil {
			return nil, err
		}

		bj.timeout = t
	} else {
		bj.timeout = 30 * time.Second
	}

	return &bj, nil
}
