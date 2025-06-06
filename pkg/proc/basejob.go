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
	errFunc(job.signalAll(sig))
}

func (job *baseJob) signalAll(sig syscall.Signal) error {
	if job.cmd.Process.Pid <= 0 { // Ensure PID is positive before negating
		return syscall.Errno(syscall.ESRCH) // No such process or invalid PID
	}
	return syscall.Kill(-job.cmd.Process.Pid, sig)
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
		job.signal(sig),
	)
}

func (job *baseJob) signal(sig os.Signal) error {
	process, err := os.FindProcess(job.cmd.Process.Pid)
	if err != nil {
		return err
	}
	return process.Signal(sig)
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

func (job *baseJob) StreamStdOut(ctx context.Context, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	if len(job.Config.Stdout) == 0 {
		return
	}
	job.readStdFile(ctx, &job.stdOutWg, job.Config.Stdout, outChan, errChan, follow, tailLen)
}

func (job *baseJob) StreamStdErr(ctx context.Context, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	if len(job.Config.Stderr) == 0 {
		return
	}
	job.readStdFile(ctx, &job.stdErrWg, job.Config.Stderr, outChan, errChan, follow, tailLen)
}

func (job *baseJob) StreamStdOutAndStdErr(ctx context.Context, outChan chan []byte, stdOutErrChan, stdErrErrChan chan error, follow bool, tailLen int) {
	job.StreamStdOut(ctx, outChan, stdOutErrChan, follow, tailLen)
	if job.Config.Stdout != job.Config.Stderr {
		job.StreamStdErr(ctx, outChan, stdErrErrChan, follow, tailLen)
	}
}

func (job *baseJob) startOnce(ctx context.Context, process chan<- *os.Process) error {
	l := log.WithField("job.name", job.Config.Name)
	defer job.closeStdFiles()

	if err := job.CreateAndOpenStdFile(job.Config); err != nil {
		return err
	}

	cmd := exec.Command(job.Config.Command, job.Config.Args...)
	cmd.Env = os.Environ()
	cmd.Dir = job.Config.WorkingDirectory

	// pipe command's stdout and stderr through timestamp function if timestamps are enabled
	// otherwise just redirect stdout and err to job.stdout and job.stderr
	if job.Config.EnableTimestamps {
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe for process: %s", err.Error())
		}
		defer stdoutPipe.Close()

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe for process: %s", err.Error())
		}
		defer stderrPipe.Close()

		go job.logWithTimestamp(stdoutPipe, job.stdout)
		go job.logWithTimestamp(stderrPipe, job.stderr)
	} else {
		cmd.Stdout = job.stdout
		cmd.Stderr = job.stderr
	}

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

	// Only set job.Cmd if cmd.Start() was successful
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
		if job.cmd != nil && job.cmd.Process != nil { // Check if process actually started
			if termErr := job.signalAll(syscall.SIGTERM); termErr != nil {
				if e, ok := termErr.(syscall.Errno); ok && e == syscall.ESRCH {
					// ESRCH (Error No Such Process) is fine, means process group already gone
				} else {
					l.WithError(termErr).Error("failed to send SIGTERM to job's process group")
				}
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
		if job.cmd != nil && job.cmd.Process != nil { // Check if process actually started
			// ctx canceled, try to terminate job
			_ = job.signalAll(syscall.SIGTERM)
			l.WithField("job.name", job.Config.Name).Info("sent SIGTERM to job's process group on ctx.Done")

			select {
			case <-time.After(time.Second * ShutdownWaitingTimeSeconds):
				// process seems to hang, kill process
				_ = job.signalAll(syscall.SIGKILL) // Use func for consistency
				l.WithField("job.name", job.Config.Name).Error("forcefully killed job")
				return nil

			case err := <-errChan:
				// all good
				return err
			}
		}
		// If job.Cmd or job.Cmd.Process is nil, it means the process didn't start or already cleaned up.
		l.WithField("job.name", job.Config.Name).Info("context done, but process was not running or already cleaned up.")
		return ctx.Err() // Return context error
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

func (job *baseJob) logWithTimestamp(r io.Reader, w io.Writer) {
	l := log.WithField("job.name", job.Config.Name)

	var layout string

	// has custom timestamp layout?
	if job.Config.CustomTimestampFormat != "" {
		layout = job.Config.CustomTimestampFormat
		l.Infof("using custom timestamp layout '%s'", layout)
	} else {
		existingLayout, exists := TimeLayouts[job.Config.TimestampFormat]
		if !exists {
			layout = time.RFC3339
			l.Warningf("unknown timestamp layout '%s', defaulting to RFC3339", job.Config.TimestampFormat)
		} else {
			layout = existingLayout
			l.Infof("logging with timestamp layout '%s'", job.Config.TimestampFormat)
		}
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		timestamp := time.Now().Format(layout)
		line := fmt.Sprintf("[%s] %s\n", timestamp, scanner.Text())
		if _, err := w.Write([]byte(line)); err != nil {
			l.Errorf("error writing log for process: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		l.Errorf("error reading from process: %v\n", err)
	}
}

func (job *baseJob) readStdFile(ctx context.Context, wg *sync.WaitGroup, filePath string, outChan chan []byte, errChan chan error, follow bool, tailLen int) {
	stdFile, err := os.OpenFile(filePath, os.O_RDONLY, 0o666)
	if err != nil {
		errChan <- err
		return
	}

	defer stdFile.Close()

	seekTail(ctx, wg, tailLen, stdFile, outChan)

	read := func() {
		scanner := bufio.NewScanner(stdFile)

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
	for {
		select {
		default:
			read()
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
