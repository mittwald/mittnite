package proc

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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
	upgrader   websocket.Upgrader
}

func NewApi(listenAddress string) *Api {
	return &Api{
		router:     mux.NewRouter(),
		listenAddr: listenAddress,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

type baseJob struct {
	Config *config.BaseJobConfig

	cmd     *exec.Cmd
	restart bool
	stop    bool
	stdout  *os.File
	stderr  *os.File
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

func newBaseJob(c *config.BaseJobConfig) (*baseJob, error) {
	job := &baseJob{
		Config:  c,
		cmd:     nil,
		restart: false,
		stop:    false,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}
	if len(c.Stdout) == 0 {
		return job, nil
	}

	stdout, err := prepareStdFile(c.Stdout)
	if err != nil {
		return nil, err
	}
	job.stdout = stdout

	if len(c.Stderr) == 0 {
		return job, nil
	}

	if c.Stderr == c.Stdout {
		job.stderr = job.stdout
		return job, nil
	}

	stderr, err := prepareStdFile(c.Stderr)
	if err != nil {
		return nil, err
	}
	job.stderr = stderr

	return job, nil
}

func NewCommonJob(c *config.JobConfig) (*CommonJob, error) {
	job, err := newBaseJob(&c.BaseJobConfig)
	if err != nil {
		return nil, err
	}
	j := CommonJob{
		baseJob: *job,
		Config:  c,
	}

	return &j, nil
}

func NewLazyJob(c *config.JobConfig) (*LazyJob, error) {
	commonJob, err := NewCommonJob(c)
	if err != nil {
		return nil, err
	}
	j := LazyJob{
		CommonJob: *commonJob,
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
