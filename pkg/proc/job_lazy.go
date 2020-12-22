package proc

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func (job *Job) CanStartLazily() bool {
	if len(job.Config.Listeners) == 0 {
		return false
	}

	return job.Config.Laziness != nil
}

func (job *Job) AssertStarted(ctx context.Context) error {
	if job.process != nil {
		return nil
	}

	job.lazyStartLock.Lock()
	defer job.lazyStartLock.Unlock()

	// Yes, this is tested twice. I know.
	// https://en.wikipedia.org/wiki/Double-checked_locking
	if job.process != nil {
		return nil
	}

	p := make(chan *os.Process)
	e := make(chan error)

	go func() {
		if err := job.start(ctx, p); err != nil {
			e <- err
		}

		job.process = nil
	}()

	select {
	case err := <-e:
		return err
	case job.process = <-p:
		return nil
	}
}

func (job *Job) start(ctx context.Context, process chan<- *os.Process) error {
	l := log.WithField("job.name", job.Config.Name)

	attempts := 0
	maxAttempts := job.Config.MaxAttempts

	if maxAttempts == 0 {
		maxAttempts = 3
	}

	ctx, job.kill = context.WithCancel(ctx)

	job.cmd = exec.CommandContext(ctx, job.Config.Command, job.Config.Args...)
	job.cmd.Stdout = os.Stdout
	job.cmd.Stderr = os.Stderr
	if job.Config.Env != nil {
		job.cmd.Env = append(os.Environ(), job.Config.Env...)
	}

	for { // restart failed jobs as long mittnite is running
		if job.cancel {
			return nil
		}

		l.Info("starting job")

		err := job.cmd.Start()
		if err != nil {
			return fmt.Errorf("failed to start job %s: %s", job.Config.Name, err.Error())
		}

		job.running = true
		if process != nil {
			process <- job.cmd.Process
		}

		err = job.cmd.Wait()
		job.running = false
		if err != nil {
			l.WithError(err).Error("job exited with error")
		} else {
			if job.Config.OneTime {
				l.Info("one-time job has ended successfully")
				return nil
			}
			l.Warn("job exited without errors")
		}

		if ctx.Err() != nil { // execution cancelled
			return nil
		}

		attempts++
		if attempts < maxAttempts {
			l.WithField("job.maxAttempts", maxAttempts).WithField("job.usedAttempts", attempts).Info("remaining attempts")
			continue
		}

		if job.Config.CanFail {
			l.WithField("job.maxAttempts", maxAttempts).Warn("reached max retries")
			return nil
		}

		return fmt.Errorf("reached max retries for job %s", job.Config.Name)
	}
}
