package proc

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// LazyJobReapGracePeriod defines the time to wait after SIGTERM before sending SIGKILL.
// It's a variable to allow modification for testing.
var LazyJobReapGracePeriod = 10 * time.Second

// GetLazyJobReapGracePeriodForTest returns the current grace period. Used by tests.
func GetLazyJobReapGracePeriodForTest() time.Duration {
	return LazyJobReapGracePeriod
}

// SetCoolDownTimeout allows tests to set a custom coolDownTimeout.
func (job *LazyJob) SetCoolDownTimeout(d time.Duration) {
	job.coolDownTimeout = d
}

// SetActiveConnections allows tests to set activeConnections.
func (job *LazyJob) SetActiveConnections(n uint32) {
	job.activeConnections = n
}

// SetLastConnectionClosed allows tests to set lastConnectionClosed.
func (job *LazyJob) SetLastConnectionClosed(t time.Time) {
	job.lastConnectionClosed = t
}

// SetProcess allows tests to set the job's process.
// This is primarily for simulating a running process.
// It sets both job.process (used by LazyJob specific logic)
// and job.Cmd.Process (used by BaseJob signal methods via job.Cmd).
func (job *LazyJob) SetProcess(p *os.Process) {
	job.process = p
	if job.Cmd != nil { // Ensure Cmd is initialized
		job.Cmd.Process = p
	} else if p != nil { // If Cmd is nil, but we have a process, create a dummy Cmd
		// This case is tricky; ideally Cmd should be fully set up by a Start-like method.
		// For testing reaper logic which relies on job.Cmd.Process.Pid, we need it.
		// This might need refinement depending on how tests set up job.Cmd.
		// log.Warn("SetProcess called on LazyJob with nil Cmd; creating a minimal Cmd. May need attention.")
		// job.Cmd = &exec.Cmd{} // This alone is not enough, Path and SysProcAttr are needed by reaper's callers
	}
}

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

	job.StartProcessReaper(ctx) // Renamed to be exported

	log.Infof("holding off starting job %s until first request", job.Config.Name)
	return nil
}

// StartProcessReaper starts the goroutine that monitors the lazy job's process
// and terminates it if it's idle for too long.
func (job *LazyJob) StartProcessReaper(ctx context.Context) {
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
				if job.activeConnections > 0 {
					continue
				}

				diff := time.Since(job.lastConnectionClosed)
				if diff < job.coolDownTimeout {
					continue
				}

				if job.process == nil {
					continue
				}

				// All conditions met to reap the process
				job.lazyStartLock.Lock()

				// Verify all conditions again inside the lock
				if job.process == nil || job.activeConnections != 0 || job.lastConnectionClosed.IsZero() || time.Since(job.lastConnectionClosed) < job.coolDownTimeout {
					job.lazyStartLock.Unlock()
					continue
				}

				// Ensure Cmd and Cmd.Process are not nil before accessing PID
				if job.Cmd == nil || job.Cmd.Process == nil {
					l.Warn("job.process is not nil, but job.Cmd or job.Cmd.Process is nil; skipping reap cycle")
					job.lazyStartLock.Unlock()
					continue
				}
				pidToReap := job.Cmd.Process.Pid
				l.Infof("sending SIGTERM to idle process PID %d", pidToReap)
				if err := job.Signal(syscall.SIGTERM); err != nil {
					l.WithError(err).Warnf("failed to send SIGTERM to PID %d", pidToReap)
					job.lazyStartLock.Unlock()
					continue
				}

				job.lazyStartLock.Unlock()

				graceTimer := time.NewTimer(LazyJobReapGracePeriod) // Use the variable
				defer graceTimer.Stop()

				select {
				case <-graceTimer.C:
					job.lazyStartLock.Lock()
					// Check if the process we sent SIGTERM to is still running
					if job.process != nil && job.Cmd != nil && job.Cmd.Process != nil && job.Cmd.Process.Pid == pidToReap {
						l.Warnf("process PID %d did not exit after SIGTERM and grace period; sending SIGKILL", pidToReap)
						if err := job.SignalAll(syscall.SIGKILL); err != nil {
							l.WithError(err).Errorf("failed to send SIGKILL to PID %d", pidToReap)
						}
					} else if job.process != nil {
						// Process is not nil, but it's not the one we targeted.
						// This could happen if the job was quickly restarted.
						currentPid := -1
						if job.Cmd != nil && job.Cmd.Process != nil {
							currentPid = job.Cmd.Process.Pid
						}
						l.Warnf("original process PID %d seems to have exited or changed; current PID is %d. Skipping SIGKILL.", pidToReap, currentPid)
					} else {
						// job.process is nil, so it was cleaned up.
						l.Infof("process PID %d exited gracefully after SIGTERM", pidToReap)
					}
					job.lazyStartLock.Unlock()
				case <-ctx.Done():
					l.Info("context done during SIGTERM grace period for PID %d", pidToReap)
					return // Exit the reaper goroutine
				}
			}
		}
	}()
}
