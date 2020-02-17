package proc

import (
	"context"
	"os/exec"
	"time"

	"github.com/mittwald/mittnite/internal/config"
)

type Runner struct {
	IgnitionConfig *config.Ignition
	jobs           []*Job
	ctx            context.Context
}

type Job struct {
	Config        *config.JobConfig
	watchingFiles map[string]time.Time
	cmd           *exec.Cmd
}
