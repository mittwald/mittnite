package proc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func (job *Job) Init() {
	for w := range job.Config.Watches {
		watch := &job.Config.Watches[w]
		job.watchingFiles = make(map[string]time.Time)

		paths, err := filepath.Glob(watch.Filename)
		if err != nil {
			continue
		}

		for _, p := range paths {
			stat, err := os.Stat(p)
			if err != nil {
				continue
			}

			job.watchingFiles[p] = stat.ModTime()
		}
	}
}

func (job *Job) Run(ctx context.Context, errors chan<- error) error {
	listerWaitGroup := sync.WaitGroup{}
	defer listerWaitGroup.Wait()

	for i := range job.Config.Listeners {
		listener, err := NewListener(job, &job.Config.Listeners[i])
		if err != nil {
			return err
		}
		job.listeners = append(job.listeners, listener)

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

	if job.CanStartLazily() {
		job.startProcessReaper(ctx)

		log.Infof("holding off starting job %s until first request", job.Config.Name)
		return nil
	}

	p := make(chan *os.Process)
	go func() {
		job.process = <-p
	}()

	return job.start(ctx, p)
}

func (job *Job) startProcessReaper(ctx context.Context) {
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

func (job *Job) Stop() {
	job.cancel = true

	// First closing all listeners to prevent new job starts
	for _, listener := range job.listeners {
		listener.Shutdown()
	}

	if !job.running {
		return
	}

	// send SIGTERM to the process for a graceful shutdown
	job.Signal(syscall.SIGTERM)
	timer := time.NewTicker(1 * time.Second)
	attempts := 0

	// wait for the process to stop
	for {
		select {
		case <-timer.C:
			if !job.running {
				return
			}

			// kill the process after the max wait time
			if attempts >= SchutdownWaitingTimeSeconds {
				if job.kill != nil {
					job.kill()
				}
				return
			}
			attempts++
		}
	}
}

func (job *Job) Signal(sig os.Signal) {
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

func (job *Job) Watch() {
	for w := range job.Config.Watches {
		watch := &job.Config.Watches[w]
		signal := false
		paths, err := filepath.Glob(watch.Filename)
		if err != nil {
			log.Warnf("failed to watch %s: %s", watch.Filename, err.Error())
			continue
		}

		// check existing files
		for _, p := range paths {
			stat, err := os.Stat(p)
			if err != nil {
				continue
			}

			mtime := stat.ModTime()
			if mtime.Equal(job.watchingFiles[p]) {
				continue
			}

			log.Infof("file %s changed, signalling process %s", p, job.Config.Name)
			job.watchingFiles[p] = mtime
			signal = true
		}

		// check deleted files
		for p := range job.watchingFiles {
			_, err := os.Stat(p)
			if os.IsNotExist(err) {
				log.Infof("file %s changed, signalling process %s", p, job.Config.Name)
				delete(job.watchingFiles, p)
				signal = true
			}
		}

		// send signal
		if signal {
			job.Signal(syscall.Signal(watch.Signal))
		}
	}
}
