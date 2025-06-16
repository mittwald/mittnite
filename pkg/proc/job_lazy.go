package proc

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const lazyJobReapGracePeriod = 10 * time.Second

func (job *LazyJob) AssertStarted(ctx context.Context) error {
	l := log.WithField("job.name", job.Config.Name)

	if job.process != nil {
		l.Info("process already running")
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
		if err := job.startOnce(ctx, p); err != nil {
			l.WithError(err).Error("process terminated with error")

			select {
			case e <- err:
			default:
			}
		}

		l.Info("process terminated")

		job.lazyStartLock.Lock()
		defer job.lazyStartLock.Unlock()
		job.process = nil
	}()

	select {
	case err := <-e:
		return err
	case job.process = <-p:
		return nil
	}
}

func (job *LazyJob) Run(ctx context.Context, errors chan<- error) error {
	listenerWaitGroup := sync.WaitGroup{}
	defer listenerWaitGroup.Wait()

	for i := range job.Config.Listeners {
		listener, err := NewListener(job, &job.Config.Listeners[i])
		if err != nil {
			return err
		}

		listenerWaitGroup.Add(1)

		go func() {
			if err := listener.Run(ctx); err != nil {
				log.WithError(err).Error("listener stopped with error")
				errors <- err
			}

			listenerWaitGroup.Done()
		}()
	}

	job.startProcessReaper(ctx)

	log.Infof("holding off starting job %s until first request", job.Config.Name)
	return nil
}

func (job *LazyJob) startProcessGracePeriod(ctx context.Context, pid int) {
	l := log.WithField("job.name", job.Config.Name)
	graceTimer := time.NewTimer(lazyJobReapGracePeriod)
	defer graceTimer.Stop()

	select {
	case <-ctx.Done():
		l.Infof("context done during SIGTERM grace period for PID %d", pid)
		return
	case <-graceTimer.C:
		job.lazyStartLock.Lock()
		defer job.lazyStartLock.Unlock()

		// Check if the process we sent SIGTERM to is still running
		if job.process != nil && job.HasStarted() && job.cmd.Process.Pid == pid {
			l.Warnf("process PID %d did not exit after SIGTERM and grace period; sending SIGKILL", pid)
			if err := job.signalAll(syscall.SIGKILL); err != nil {
				l.WithError(err).Errorf("failed to send1 SIGKILL to PID %d", pid)
			}
		} else if job.process != nil {
			// Process is not nil, but it's not the one we targeted.
			// This could happen if the job was quickly restarted.
			currentPid := -1
			if job.HasStarted() {
				currentPid = job.cmd.Process.Pid
			}
			l.Warnf("original process PID %d seems to have exited or changed; current PID is %d. Skipping SIGKILL.", pid, currentPid)
		} else {
			// job.process is nil, so it was cleaned up.
			l.Infof("process PID %d exited gracefully after SIGTERM", pid)
		}
	}
}

func (job *LazyJob) reapProcess() int {
	l := log.WithField("job.name", job.Config.Name)
	if job.activeConnections > 0 {
		return 0
	}
	if diff := time.Since(job.lastConnectionClosed); diff < job.coolDownTimeout {
		return 0
	}
	if job.process == nil {
		return 0
	}

	job.lazyStartLock.Lock()
	defer job.lazyStartLock.Unlock()

	// Verify all conditions again inside the lock
	if job.process == nil || job.activeConnections != 0 || job.lastConnectionClosed.IsZero() || time.Since(job.lastConnectionClosed) < job.coolDownTimeout {
		return 0
	}
	if !job.HasStarted() {
		l.Warn("job.process is not nil, but job.cmd or job.cmd.Process is nil; skipping reap cycle")
		return 0
	}

	pidToReap := job.cmd.Process.Pid
	l.Infof("sending SIGTERM to idle process PID %d", pidToReap)
	if err := job.signal(syscall.SIGTERM); err != nil {
		l.WithError(err).Warnf("failed to send SIGTERM to PID %d", pidToReap)
		return 0
	}

	return pidToReap
}

func (job *LazyJob) startProcessReaper(ctx context.Context) {
	reaperInterval := job.coolDownTimeout / 2
	if reaperInterval < 1*time.Second {
		reaperInterval = 1 * time.Second
	}
	ticker := time.NewTicker(reaperInterval)
	go func() {
		l := log.WithField("job.name", job.Config.Name)
		l.Info("starting lazy job process reaper")
		defer ticker.Stop()
		defer l.Info("stopping lazy job process reaper")

		for {
			select {
			case <-ctx.Done():
				l.Info("context done, stopping lazy job process reaper")
				return
			case <-ticker.C:
				pidToReap := job.reapProcess()
				if pidToReap == 0 {
					continue
				}
				job.startProcessGracePeriod(ctx, pidToReap)
			}
		}
	}()
}
