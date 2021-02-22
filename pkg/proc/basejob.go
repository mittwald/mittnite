package proc

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func (job *baseJob) Signal(sig os.Signal) {
	errFunc := func(err error) {
		if err != nil {
			log.Warnf("failed to send signal %d to job %s: %s", sig, job.Config.Name, err.Error())
		}
	}

	if job.cmd == nil || job.cmd.Process == nil {
		errFunc(
			fmt.Errorf("job is not running"),
		)
		return
	}

	errFunc(
		job.cmd.Process.Signal(sig),
	)
}

func (job *baseJob) startOnce(ctx context.Context, process chan<- *os.Process) error {
	l := log.WithField("job.name", job.Config.Name)

	job.cmd = exec.Command(job.Config.Command, job.Config.Args...)
	job.cmd.Stdout = os.Stdout
	job.cmd.Stderr = os.Stderr
	if job.Config.Env != nil {
		job.cmd.Env = append(os.Environ(), job.Config.Env...)
	}

	l.Info("starting job")

	err := job.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start job %s: %s", job.Config.Name, err.Error())
	}

	if process != nil {
		process <- job.cmd.Process
	}

	errChan := make(chan error, 1)
	defer func() {
		close(errChan)
	}()
	go func() {
		errChan <- job.cmd.Wait()
	}()

	select {
	// job errChan or failed
	case err := <-errChan:
		if err != nil {
			l.WithError(err).Error("job exited with error")
		}
		return err
	case <-ctx.Done():
		// ctx canceled, try to terminate job
		_ = job.cmd.Process.Signal(syscall.SIGTERM)
		l.WithField("job.name", job.Config.Name).Info("sent SIGTERM to job")

		select {
		case <-time.After(time.Second * ShutdownWaitingTimeSeconds):
			// process seems to hang, kill process
			_ = job.cmd.Process.Kill()
			l.WithField("job.name", job.Config.Name).Error("forcefully killed job")
			return nil

		case err := <-errChan:
			// all good
			return err
		}
	}
}
