package proc

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

func (job *BootJob) Run(ctx context.Context) error {
	l := log.WithField("job.name", job.Config.Name)

	ctx, job.cancelProcess = context.WithCancel(ctx)

	job.cmd = exec.CommandContext(ctx, job.Config.Command, job.Config.Args...)
	job.cmd.Stdout = os.Stdout
	job.cmd.Stderr = os.Stderr
	if job.Config.Env != nil {
		job.cmd.Env = append(os.Environ(), job.Config.Env...)
	}

	l.Info("starting boot job")

	err := job.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start job %s: %s", job.Config.Name, err.Error())
	}

	job.process = job.cmd.Process

	err = job.cmd.Wait()
	if err != nil {
		l.WithError(err).Error("job exited with error")
	} else {
		l.Info("boot job completed")
	}

	if ctx.Err() != nil { // execution cancelled
		return ctx.Err()
	}

	if err != nil {
		if job.Config.CanFail {
			l.WithError(err).Warn("job failed, but is allowed to fail")
			return nil
		}

		l.WithError(err).Error("boot job failed")
		return errors.Wrapf(err, "error while exec'ing boot job '%s'", job.Config.Name)
	}

	return nil
}
