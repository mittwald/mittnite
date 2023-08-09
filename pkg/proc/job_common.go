package proc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

const (
	// longest duration between two restarts
	maxBackOff = 300 * time.Second
)

func (job *CommonJob) Init() {
	job.restart = false
	job.stop = false

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

func (job *CommonJob) Run(ctx context.Context, _ chan<- error) error {
	if job.stop {
		return nil
	}

	l := log.WithField("job.name", job.Config.Name)

	attempts := 0
	maxAttempts := job.Config.MaxAttempts

	backOff := 1 * time.Second

	if maxAttempts == 0 {
		maxAttempts = 3
	}

	if maxAttempts < 0 {
		maxAttempts = -1
	}

	p := make(chan *os.Process)
	defer close(p)

	go func() {
		for range p {
			job.phase.Set(JobPhaseReasonStarted)
		}
	}()

	for { // restart failed jobs as long mittnite is running
		if job.stop {
			return nil
		}

		job.ctx, job.interrupt = context.WithCancel(context.Background())
		err := job.startOnce(ctx, p)
		switch err {
		case nil:
			if job.Config.OneTime {
				l.Info("one-time job has ended successfully")
				job.phase.Set(JobPhaseReasonCompleted)
				return nil
			}
			l.Warn("job exited without errors")
		case ProcessWillBeRestartedError:
			l.Info("restart process")
			continue
		case ProcessWillBeStoppedError:
			l.Info("stop process")
			job.phase.Set(JobPhaseReasonStopped)
			return nil
		}

		attempts++
		if maxAttempts == -1 || attempts < maxAttempts {
			currBackOff := backOff
			backOff = calculateNextBackOff(currBackOff, maxBackOff)

			job.phase.Set(JobPhaseReasonCrashLooping)
			l.
				WithField("job.maxAttempts", maxAttempts).
				WithField("job.usedAttempts", attempts).
				WithField("job.nextRestartIn", currBackOff.String()).
				Info("remaining attempts")

			job.crashLoopSleep(currBackOff)
			continue
		}

		job.phase.Set(JobPhaseReasonFailed)

		if job.Config.CanFail {
			l.WithField("job.maxAttempts", maxAttempts).Warn("reached max retries")
			return nil
		}

		return fmt.Errorf("reached max retries for job %s; last error: %w", job.Config.Name, err)
	}
}

func (job *CommonJob) Watch() {
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

		if !signal {
			continue
		}

		l := log.WithField("job.name", job.Config.Name)
		if watch.PreCommand != nil {
			if err := job.executeWatchCommand(watch.PreCommand); err != nil {
				l.WithError(err).Warn("failed to execute pre watch command")
			}
		}

		if watch.Restart {
			job.MarkForRestart()
		}
		job.Signal(syscall.Signal(watch.Signal))

		if watch.PostCommand != nil {
			if err := job.executeWatchCommand(watch.PostCommand); err != nil {
				l.WithError(err).Warn("failed to execute post watch command")
			}
		}
	}
}

func (job *CommonJob) IsRunning() bool {
	if job.cmd == nil {
		return false
	}
	if job.cmd.Process == nil {
		return false
	}
	if job.cmd.Process.Pid > 0 {
		return syscall.Kill(job.cmd.Process.Pid, syscall.Signal(0)) == nil
	}
	return true
}

func (job *CommonJob) Restart() {
	job.restart = true
	job.SignalAll(syscall.SIGTERM)
	job.interrupt()
}

func (job *CommonJob) Stop() {
	job.stop = true
	job.SignalAll(syscall.SIGTERM)
	job.interrupt()
}

func (job *CommonJob) Status() *CommonJobStatus {
	running := job.IsRunning()
	var pid int
	if running {
		pid = job.cmd.Process.Pid
	}
	return &CommonJobStatus{
		Pid:     pid,
		Running: job.IsRunning(),
		Phase:   job.phase,
		Config:  job.Config,
	}
}

func (job *CommonJob) executeWatchCommand(watchCmd *config.WatchCommand) error {
	if len(watchCmd.Command) == 0 {
		return errors.New("command is missing")
	}
	cmd := exec.Command(watchCmd.Command, watchCmd.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if watchCmd.Env != nil {
		cmd.Env = append(cmd.Env, watchCmd.Env...)
	}

	log.WithField("job.name", job.Config.Name).
		Info("executing watch command")
	return cmd.Run()
}

func (job *CommonJob) crashLoopSleep(duration time.Duration) {
	timeout := make(chan bool)

	go func() {
		defer close(timeout)
		<-time.After(duration)
		timeout <- true
	}()

	for {
		select {
		case <-timeout:
			return
		case <-job.ctx.Done():
			return
		}
	}
}
