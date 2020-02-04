package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

func (job *Job) ID() string {
	if job.id == "" {
		job.id = uuid.New().String()
	}

	return job.id
}

func (job *Job) Init() {
	for w := range job.Watches {
		watch := &job.Watches[w]
		watch.matchingFiles = make(map[string]time.Time)

		paths, err := filepath.Glob(watch.Filename)
		if err != nil {
			continue
		}

		for _, p := range paths {
			stat, err := os.Stat(p)
			if err != nil {
				continue
			}

			watch.matchingFiles[p] = stat.ModTime()
		}
	}
}

func (job *Job) Run(ctx context.Context) error {
	attempts := 0
	maxAttempts := job.MaxAttempts

	if maxAttempts == 0 {
		maxAttempts = 3
	}

	for { // restart failed jobs as long mittnite is running

		log.Infof("starting job %s", job.Name)
		job.cmd = exec.CommandContext(ctx, job.Command, job.Args...)
		job.cmd.Stdout = os.Stdout
		job.cmd.Stderr = os.Stderr

		err := job.cmd.Start()
		if err != nil {
			return fmt.Errorf("failed to start job %s: %s", job.Name, err.Error())
		}

		err = job.cmd.Wait()
		if err != nil {
			log.Errorf("job %s exited with error: %s", job.Name, err)
		} else {
			log.Warnf("job %s exited without errors", job.Name)
		}

		if ctx.Err() != nil { // execution cancelled
			return nil
		}

		attempts++
		if attempts < maxAttempts {
			log.Infof("job %s has %d attempts remaining", job.Name, maxAttempts-attempts)
			continue
		}

		if job.CanFail {
			log.Warnf("")
			return nil
		}

		return fmt.Errorf("reached max retries for job %s", job.Name)
	}
}

func (job *Job) Signal(sig os.Signal) {
	errFunc := func(err error) {
		if err != nil {
			log.Warnf("failed to send signal %d to job %s: %s", sig, job.Name, err.Error())
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
	for w := range job.Watches {
		watch := &job.Watches[w]
		signal := false
		paths, err := filepath.Glob(watch.Filename)
		if err != nil {
			log.Warnf("failed to watch %s: %s", watch, err)
			continue
		}

		// check existing files
		for _, p := range paths {
			stat, err := os.Stat(p)
			if err != nil {
				continue
			}

			mtime := stat.ModTime()
			if mtime.Equal(watch.matchingFiles[p]) {
				continue
			}

			log.Infof("file %s changed, signalling process %s", p, job.Name)
			watch.matchingFiles[p] = mtime
			signal = true
		}

		// check deleted files
		for p := range watch.matchingFiles {
			_, err := os.Stat(p)
			if os.IsNotExist(err) {
				log.Infof("file %s changed, signalling process %s", p, job.Name)
				delete(watch.matchingFiles, p)
				signal = true
			}
		}

		// send signal
		if signal {
			job.Signal(syscall.Signal(watch.Signal))
		}
	}
}
