package proc

import (
	"bufio"
	"container/list"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	_ Job = &CommonJob{}

	ProcessWillBeRestartedError = errors.New("process will be restarted")
	ProcessWillBeStoppedError   = errors.New("process will be stopped")
)

func (job *baseJob) SignalAll(sig syscall.Signal) {
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

	log.WithField("job.name", job.Config.Name).Infof("sending signal %d to process group", sig)
	errFunc(syscall.Kill(-job.cmd.Process.Pid, sig))
}

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

func (job *baseJob) Reset() {
	job.phase = JobPhase{}
}

func (job *baseJob) MarkForRestart() {
	job.restart = true
}

func (job *baseJob) IsControllable() bool {
	return job.Config.Controllable
}

func (job *baseJob) GetPhase() *JobPhase {
	return &job.phase
}

func (job *baseJob) GetName() string {
	return job.Config.Name
}

func (job *baseJob) StreamStdOut(ctx context.Context, wg *sync.WaitGroup, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	if len(job.Config.Stdout) == 0 {
		return
	}
	job.streamLogFile(ctx, wg, job.Config.Stdout, outChan, errChan, follow, tailLen)
}

func (job *baseJob) StreamStdErr(ctx context.Context, wg *sync.WaitGroup, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	if len(job.Config.Stderr) == 0 {
		return
	}
	job.streamLogFile(ctx, wg, job.Config.Stderr, outChan, errChan, follow, tailLen)
}

func (job *baseJob) StreamStdOutAndStdErr(ctx context.Context, wg *sync.WaitGroup, outChan chan []byte, stdOutErrChan, stdErrErrChan chan error, follow bool, tailLen int) {
	job.StreamStdOut(ctx, wg, outChan, stdOutErrChan, follow, tailLen)
	if job.Config.Stdout != job.Config.Stderr {
		job.StreamStdErr(ctx, wg, outChan, stdErrErrChan, follow, tailLen)
	}
}

func (job *baseJob) readLog(ctx context.Context, wg *sync.WaitGroup, outChan chan []byte, errChan chan error, fileHandle *os.File, follow bool, tailLen int) {
	scanner := bufio.NewScanner(fileHandle)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case outChan <- scanner.Bytes():
		default:
			if follow {
				continue
			}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case <-ctx.Done():
			return
		case errChan <- err:
		default:
			return
		}
	}
}

func (job *baseJob) streamLogFile(ctx context.Context, wg *sync.WaitGroup, filePath string, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	stdFile, err := os.OpenFile(filePath, os.O_RDONLY, 0o666)
	if err != nil {
		errChan <- err
		return
	}

	defer stdFile.Close()

	seekTail(ctx, wg, tailLen, stdFile, outChan)
	wg.Wait()

	for {
		select {
		default:
			job.readLog(ctx, wg, outChan, errChan, stdFile, follow, tailLen)
			if !follow {
				errChan <- io.EOF
				return
			}

			continue
		case <-ctx.Done():
			return
		}
	}
}

func (job *baseJob) startOnce(ctx context.Context, process chan<- *os.Process) error {
	l := log.WithField("job.name", job.Config.Name)
	defer job.closeStdFiles()

	cmd := exec.Command(job.Config.Command, job.Config.Args...)
	cmd.Env = os.Environ()
	cmd.Dir = job.Config.WorkingDirectory
	cmd.Stdout = job.stdout
	cmd.Stderr = job.stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if job.Config.Env != nil {
		cmd.Env = append(cmd.Env, job.Config.Env...)
	}

	l.Info("starting job")

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start job %s: %s", job.Config.Name, err.Error())
	}

	// Only set job.cmd if cmd.Start() was successful
	job.cmd = cmd

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
		if err := syscall.Kill(-job.cmd.Process.Pid, syscall.SIGTERM); err != nil {
			if e, ok := err.(syscall.Errno); ok && e == 3 {
				// this is fine; error 3 means that the process group does not exist anymore
			} else {
				l.WithError(err).Error("failed to send SIGTERM to job's process group")
			}
		}

		if job.restart {
			l.Info("job stopped for restart")
			job.restart = false
			return ProcessWillBeRestartedError
		}

		if job.stop {
			l.Info("job stopped")
			return ProcessWillBeStoppedError
		}

		if err != nil {
			l.WithError(err).Error("job exited with error")
		}
		return err
	case <-ctx.Done():
		// ctx canceled, try to terminate job
		_ = syscall.Kill(-job.cmd.Process.Pid, syscall.SIGTERM)
		l.WithField("job.name", job.Config.Name).Info("sent SIGTERM to job's process group")

		select {
		case <-time.After(time.Second * ShutdownWaitingTimeSeconds):
			// process seems to hang, kill process
			_ = syscall.Kill(-job.cmd.Process.Pid, syscall.SIGKILL)
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
	hasStderr := len(job.Config.Stderr) > 0 && job.Config.Stderr != job.Config.Stdout
	if hasStdout {
		job.stdout.Close()
	}

	if hasStderr {
		job.stderr.Close()
	}
}

func (job *baseJob) readStdFile(ctx context.Context, filePath string, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	stdFile, err := os.OpenFile(filePath, os.O_RDONLY, 0o666)
	if err != nil {
		errChan <- err
		return
	}
	defer stdFile.Close()
	wg := sync.WaitGroup{}
	seekTail(ctx, &wg, tailLen, stdFile, outChan)

	read := func() {
		scanner := bufio.NewScanner(stdFile)
		for scanner.Scan() {
			line := scanner.Bytes()
			select {
			case outChan <- line:
			default:
				log.WithField("job.name", job.Config.Name).
					Warnf("failed to read logs for job %s: log streaming channel is closed.", job.Config.Name)
				return
			}
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
			if !follow {
				errChan <- io.EOF
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func seekTail(ctx context.Context, wg *sync.WaitGroup, lines int, stdFile *os.File, outChan chan []byte) {
	wg.Add(1)
	defer wg.Done()

	if lines < 0 {
		return
	}

	if lines == 0 {
		_, _ = stdFile.Seek(0, io.SeekEnd)
		return
	}

	scanner := bufio.NewScanner(stdFile)
	tailBuffer := list.New()
	for scanner.Scan() {
		line := scanner.Bytes()
		if tailBuffer.Len() >= lines {
			tailBuffer.Remove(tailBuffer.Front())
		}
		tailBuffer.PushBack(line)
	}
	for tailBuffer.Len() > 0 {
		select {
		case <-ctx.Done():
			return
		default:
			item := tailBuffer.Front()
			line, ok := item.Value.([]byte)
			if ok {
				outChan <- line
			}
			tailBuffer.Remove(item)
		}
	}
}

func prepareStdFile(filePath string) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(filePath), 0o755); err != nil {
		return nil, err
	}
	return os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0o666)
}
