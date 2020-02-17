package proc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	attempts := 0
	maxAttempts := job.Config.MaxAttempts

	if maxAttempts == 0 {
		maxAttempts = 3
	}

	for { // restart failed jobs as long mittnite is running

		log.Infof("starting job %s", job.Config.Name)
		job.cmd = exec.CommandContext(ctx, job.Config.Command, job.Config.Args...)
		job.cmd.Stdout = os.Stdout
		job.cmd.Stderr = os.Stderr

		err := job.cmd.Start()
		if err != nil {
			return fmt.Errorf("failed to start job %s: %s", job.Config.Name, err.Error())
		}

		err = job.cmd.Wait()
		if err != nil {
			log.Errorf("job %s exited with error: %s", job.Config.Name, err)
		} else {
			if job.Config.OneTime {
				log.Infof("one-time job %s has ended successfully", job.Config.Name)
				return nil
			}
			log.Warnf("job %s exited without errors", job.Config.Name)
		}

		if ctx.Err() != nil { // execution cancelled
			return nil
		}

		attempts++
		if attempts < maxAttempts {
			log.Infof("job %s has %d attempts remaining", job.Config.Name, maxAttempts-attempts)
			continue
		}

		if job.Config.CanFail {
			log.Warnf("")
			return nil
		}

		return fmt.Errorf("reached max retries for job %s", job.Config.Name)
	}
}

func (job *Job) Signal(sig os.Signal) {
	fmt.Println("JOB SIGNAL")
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
