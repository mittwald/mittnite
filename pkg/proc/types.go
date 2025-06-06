package proc

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/mittwald/mittnite/internal/config"
)

const (
	ShutdownWaitingTimeSeconds = 10
)

var TimeLayouts = map[string]string{
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
	"DateTime":    time.DateTime,
	"DateOnly":    time.DateOnly,
	"TimeOnly":    time.TimeOnly,
}

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

	ctx       context.Context
	interrupt context.CancelFunc
	stdErrWg  sync.WaitGroup
	stdOutWg  sync.WaitGroup

	cmd       *exec.Cmd
	restart   bool
	stop      bool
	stdout    *os.File
	stderr    *os.File
	lastError error
	phase     JobPhase
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
	Phase   JobPhase          `json:"phase"`
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
	Reset()

	GetPhase() *JobPhase
	GetName() string
}

func newBaseJob(jobConfig *config.BaseJobConfig) (*baseJob, error) {
	job := &baseJob{
		Config:  jobConfig,
		cmd:     nil,
		restart: false,
		stop:    false,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
	}
	job.phase.Set(JobPhaseReasonAwaitingReadiness)
	if len(jobConfig.Stdout) == 0 {
		return job, nil
	}

	return job, job.CreateAndOpenStdFile(jobConfig)
}

func (job *baseJob) CreateAndOpenStdFile(jobConfig *config.BaseJobConfig) error {
	if jobConfig.Stdout != "" {
		stdout, err := prepareStdFile(jobConfig.Stdout)
		if err != nil {
			return err
		}
		job.stdout = stdout
	}

	if jobConfig.Stderr != "" {
		if jobConfig.Stderr == jobConfig.Stdout {
			job.stderr = job.stdout
			return nil
		}

		stderr, err := prepareStdFile(jobConfig.Stderr)
		if err != nil {
			return err
		}
		job.stderr = stderr
	}

	return nil
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

	commonJob.phase.Set(JobPhaseReasonAwaitingReadiness)

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
