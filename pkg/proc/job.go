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

func (job *Job) Run(ctx context.Context) error {
	ctx, job.cancelAll = context.WithCancel(ctx)

	listerWaitGroup := sync.WaitGroup{}
	defer listerWaitGroup.Wait()

	for i := range job.Config.Listeners {
		listener, err := NewListener(ctx, job, &job.Config.Listeners[i])
		if err != nil {
			return err
		}

		listerWaitGroup.Add(1)

		go func() {
			if err := listener.Run(); err != nil {
				log.WithError(err).Error("listener stopped with error")
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

				diff := time.Now().Sub(job.lastConnectionClosed)
				if diff < job.coolDownTimeout {
					continue
				}

				job.lazyStartLock.Lock()

				if job.cancelProcess != nil {
					job.cancelProcess()
				}

				job.lazyStartLock.Unlock()
			}
		}
	}()
}

func (job *Job) Signal(sig os.Signal) {
	errFunc := func(err error) {
		if err != nil {
			log.Warnf("failed to send signal %d to job %s: %s", sig, job.Config.Name, err.Error())
		}
	}

	if sig == syscall.SIGTERM && job.cancelAll != nil {
		job.cancelAll()
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
