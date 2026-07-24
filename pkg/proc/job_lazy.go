package proc

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

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
	// buffered so a startOnce failure is never dropped, even when it happens
	// before the select below starts receiving
	e := make(chan error, 1)

	go func() {
		err := job.startOnce(ctx, p)
		switch {
		case err == nil:
			l.Info("process terminated")
		case errors.Is(err, ProcessStoppedIntentionallyError):
			l.Info("process stopped after cool-down")
		default:
			l.WithError(err).Error("process terminated with error")
			e <- err
		}

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

	job.startCoolDownWatcher(ctx)

	log.WithField("job.name", job.Config.Name).Info("holding off starting job until first request")
	return nil
}

// startCoolDownWatcher stops the job's process once it has been idle for
// longer than the configured cool-down timeout; the next connection spins it
// up again. Not to be confused with the zombie reaper in reaper_linux.go.
func (job *LazyJob) startCoolDownWatcher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if atomic.LoadUint32(&job.activeConnections) > 0 {
					continue
				}

				idle := time.Since(job.lastConnectionClosed)
				if idle < job.coolDownTimeout {
					continue
				}

				if job.process == nil {
					continue
				}

				job.lazyStartLock.Lock()

				log.WithField("job.name", job.Config.Name).
					WithField("idle.duration", idle.Round(time.Second).String()).
					Info("stopping idle lazy job after cool-down timeout")

				job.stopExpected.Store(true)
				job.Signal(syscall.SIGTERM)

				job.lazyStartLock.Unlock()
			}
		}
	}()
}
