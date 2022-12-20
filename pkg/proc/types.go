package proc

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"
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
	jobs        []Job
	bootJobs    []*BootJob
	api         *Api
	waitGroup   *sync.WaitGroup
	ctx         context.Context
	errChan     chan error
	keepRunning bool

	IgnitionConfig *config.Ignition
}

type Api struct {
	listenAddr string
	srv        *http.Server
	router     *mux.Router
}

func NewApi(listenAddress string) *Api {
	return &Api{
		router:     mux.NewRouter(),
		listenAddr: listenAddress,
	}
}

type baseJob struct {
	Config *config.BaseJobConfig

	cmd     *exec.Cmd
	restart bool
	stop    bool
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

type CommonJobStatus struct {
	Pid     int               `json:"pid,omitempty"`
	Running bool              `json:"running"`
	Config  *config.JobConfig `json:"config"`
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

	GetName() string
}

func NewCommonJob(c *config.JobConfig) *CommonJob {
	j := CommonJob{
		baseJob: baseJob{
			Config: &c.BaseJobConfig,
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
