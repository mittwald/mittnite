package proc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (job *BootJob) Run(ctx context.Context) error {
	l := log.WithField("job.name", job.Config.Name)

	job.cmd = exec.Command(job.Config.Command, job.Config.Args...)
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

	errChan := make(chan error, 1)
	go func() {
		errChan <- job.cmd.Wait()
	}()

	select {
	// job errChan or failed
	case err := <-errChan:
		if err != nil {
			l.WithError(err).Error("job exited with error")
		} else {
			l.Info("boot job completed")
		}

		if err != nil {
			if job.Config.CanFail {
				l.WithError(err).Warn("job failed, but is allowed to fail")
				return nil
			}

			l.WithError(err).Error("boot job failed")
			return errors.Wrapf(err, "error while exec'ing boot job '%s'", job.Config.Name)
		}
		close(errChan)

	case <-ctx.Done():
		// ctx canceled, try to terminate job
		_ = job.cmd.Process.Signal(syscall.SIGTERM)

		select {
		case <-time.After(time.Second * ShutdownWaitingTimeSeconds):
			// process seems to hang, kill process
			_ = job.cmd.Process.Kill()
			l.WithField("job.name", job.Config.Name).Warn("forcefully killed job")
			return nil

		case err := <-errChan:
			// all good
			close(errChan)
			return err
		}
	}

	return nil
}
