package proc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	ProcessWillBeRestartedError = errors.New("process will be restarted")
	ProcessWillBeStoppedError   = errors.New("process will be stopped")
)

func (job *baseJob) Signal(sig os.Signal) {
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

	log.WithField("job.name", job.Config.Name).Infof("sending signal %d to process", sig)
	errFunc(
		job.cmd.Process.Signal(sig),
	)
}

func (job *baseJob) MarkForRestart() {
	job.restart = true
}

func (job *baseJob) IsControllable() bool {
	return job.Config.Controllable
}

func (job *baseJob) GetName() string {
	return job.Config.Name
}

func (job *baseJob) StreamStdOut(ctx context.Context, outChan chan []byte, errChan chan error) {
	if len(job.Config.Stdout) == 0 {
		return
	}
	job.readStdFile(ctx, job.Config.Stdout, outChan, errChan)
}

func (job *baseJob) StreamStdErr(ctx context.Context, outChan chan []byte, errChan chan error) {
	if len(job.Config.Stderr) == 0 {
		return
	}
	job.readStdFile(ctx, job.Config.Stderr, outChan, errChan)
}

func (job *baseJob) StreamStdOutAndStdErr(ctx context.Context, outChan chan []byte, errChan chan error) {
	job.StreamStdOut(ctx, outChan, errChan)
	if job.Config.Stdout != job.Config.Stderr {
		job.StreamStdErr(ctx, outChan, errChan)
	}
}

func (job *baseJob) startOnce(ctx context.Context, process chan<- *os.Process) error {
	l := log.WithField("job.name", job.Config.Name)
	defer job.closeStdFiles()

	job.cmd = exec.Command(job.Config.Command, job.Config.Args...)
	job.cmd.Env = os.Environ()
	job.cmd.Dir = job.Config.WorkingDirectory
	job.cmd.Stdout = job.stdout
	job.cmd.Stderr = job.stderr

	if job.Config.Env != nil {
		job.cmd.Env = append(job.cmd.Env, job.Config.Env...)
	}

	l.Info("starting job")

	err := job.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start job %s: %s", job.Config.Name, err.Error())
	}

	if process != nil {
		process <- job.cmd.Process
	}

	errChan := make(chan error, 1)
	defer close(errChan)

	go func() {
		errChan <- job.cmd.Wait()
	}()

	select {
	// job errChan or failed
	case err := <-errChan:
		if job.restart {
			job.restart = false
			return ProcessWillBeRestartedError
		}

		if job.stop {
			return ProcessWillBeStoppedError
		}

		if err != nil {
			l.WithError(err).Error("job exited with error")
		}
		return err
	case <-ctx.Done():
		// ctx canceled, try to terminate job
		_ = job.cmd.Process.Signal(syscall.SIGTERM)
		l.WithField("job.name", job.Config.Name).Info("sent SIGTERM to job")

		select {
		case <-time.After(time.Second * ShutdownWaitingTimeSeconds):
			// process seems to hang, kill process
			_ = job.cmd.Process.Kill()
			l.WithField("job.name", job.Config.Name).Error("forcefully killed job")
			return nil

		case err := <-errChan:
			// all good
			return err
		}
	}
}

func (job *baseJob) closeStdFiles() {
	hasStdout := len(job.Config.Stdout) > 0
	hasStderr := len(job.Config.Stderr) > 0
	if !hasStdout && !hasStderr {
		return
	}

	if err := job.stdout.Close(); err != nil {
		log.WithField("job.name", job.Config.Name).
			WithField("job.stdout", job.Config.Stdout).
			Warn("failed to close stdout file: %s", err.Error)
	}

	if hasStderr && job.Config.Stderr != job.Config.Stdout {
		if err := job.stderr.Close(); err != nil {
			log.WithField("job.name", job.Config.Name).
				WithField("job.stdout", job.Config.Stdout).
				Warn("failed to close stderr file: %s", err.Error)
		}
	}
}

func (job *baseJob) readStdFile(ctx context.Context, filePath string, outChan chan []byte, errChan chan error) {
	stdFile, err := os.OpenFile(filePath, os.O_RDONLY, 0o666)
	if err != nil {
		errChan <- err
		return
	}
	defer stdFile.Close()

	read := func() {
		scanner := bufio.NewScanner(stdFile)
		for scanner.Scan() {
			outChan <- scanner.Bytes()
		}
		if err := scanner.Err(); err != nil {
			errChan <- err
			return
		}
	}

	for {
		select {
		default:
			read()
		case <-ctx.Done():
			return
		}
	}
}

func prepareStdFile(filePath string) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(filePath), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0o666)
}
