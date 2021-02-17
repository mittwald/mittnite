package proc

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func (job *LazyJob) AssertStarted(ctx context.Context) error {
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
		if err := job.startOnce(ctx, p); err != nil {
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

func (job *LazyJob) Run(ctx context.Context, errors chan<- error) error {
	listerWaitGroup := sync.WaitGroup{}
	defer listerWaitGroup.Wait()

	for i := range job.Config.Listeners {
		listener, err := NewListener(job, &job.Config.Listeners[i])
		if err != nil {
			return err
		}

		listerWaitGroup.Add(1)

		go func() {
			listerWaitGroup.Wait()
		}()

		go func() {
			if err := listener.Run(ctx); err != nil {
				log.WithError(err).Error("listener stopped with error")
				errors <- err
			}

			listerWaitGroup.Done()
		}()
	}

	job.startProcessReaper(ctx)

	log.Infof("holding off starting job %s until first request", job.Config.Name)
	return nil
}

func (job *LazyJob) startProcessReaper(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if job.activeConnections > 0 {
					continue
				}

				diff := time.Since(job.lastConnectionClosed)
				if diff < job.coolDownTimeout {
					continue
				}

				job.lazyStartLock.Lock()

				job.Signal(syscall.SIGTERM)

				job.lazyStartLock.Unlock()
			}
		}
	}()
}
