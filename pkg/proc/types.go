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
	jobs           []*Job
	bootJobs       []*BootJob
}

type BootJob struct {
	Config  *config.BootJobConfig
	cmd     *exec.Cmd
	process *os.Process
	timeout time.Duration
}

type Job struct {
	Config        *config.JobConfig
	watchingFiles map[string]time.Time
	cmd           *exec.Cmd
	process       *os.Process

	lazyStartLock sync.Mutex

	spinUpTimeout        time.Duration
	coolDownTimeout      time.Duration
	lastConnectionClosed time.Time
	activeConnections    uint32
}

func NewJob(c *config.JobConfig) (*Job, error) {
	j := Job{
		Config: c,
	}

	if c.Laziness != nil {
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
	}

	return &j, nil
}

func NewBootJob(c *config.BootJobConfig) (*BootJob, error) {
	bj := BootJob{
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
