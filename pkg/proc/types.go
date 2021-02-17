package proc

import (
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

	jobs     []*Job
	lazyJobs []*LazyJob
	bootJobs []*BootJob
}

type BaseJob struct {
	Config *config.BaseJobConfig

	cmd     *exec.Cmd
}

type Job struct {
	BaseJob
	Config *config.JobConfig

	watchingFiles map[string]time.Time
}

type BootJob struct {
	BaseJob
	Config *config.BootJobConfig

	timeout time.Duration
}

type LazyJob struct {
	BaseJob
	Config *config.JobConfig

	process *os.Process

	lazyStartLock     sync.Mutex
	activeConnections uint32

	spinUpTimeout        time.Duration
	coolDownTimeout      time.Duration
	lastConnectionClosed time.Time
}

func NewJob(c *config.JobConfig) *Job {
	j := Job{
		BaseJob: BaseJob{
			Config: &c.BaseJobConfig,
		},
		Config: c,
	}

	return &j
}

func NewLazyJob(c *config.JobConfig) (*LazyJob, error) {
	j := LazyJob{
		BaseJob: BaseJob{
			Config: &c.BaseJobConfig,
		},
		Config: c,
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
		BaseJob: BaseJob{
			Config: &c.BaseJobConfig,
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
