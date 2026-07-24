package proc

import (
	"bufio"
	"bytes"
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

	ProcessWillBeRestartedError      = errors.New("process will be restarted")
	ProcessWillBeStoppedError        = errors.New("process will be stopped")
	ProcessStoppedIntentionallyError = errors.New("process stopped intentionally")
)

func (job *baseJob) SignalAll(sig syscall.Signal) {
	l := log.WithField("job.name", job.Config.Name).WithField("signal", sig)

	if job.cmd == nil || job.cmd.Process == nil {
		l.Warn("failed to send signal to process group: job is not running")
		return
	}

	l.Info("sending signal to process group")
	if err := syscall.Kill(-job.cmd.Process.Pid, sig); err != nil {
		l.WithError(err).Warn("failed to send signal to process group")
	}
}

func (job *baseJob) Signal(sig os.Signal) {
	l := log.WithField("job.name", job.Config.Name).WithField("signal", sig)

	if job.cmd == nil || job.cmd.Process == nil {
		l.Warn("failed to send signal to process: job is not running")
		return
	}

	l.Info("sending signal to process")
	if err := job.cmd.Process.Signal(sig); err != nil {
		l.WithError(err).Warn("failed to send signal to process")
	}
}

func (job *baseJob) Reset() {
	job.phaseMu.Lock()
	defer job.phaseMu.Unlock()
	job.phase = JobPhase{}
}

func (job *baseJob) MarkForRestart() {
	job.restart.Store(true)
}

func (job *baseJob) IsControllable() bool {
	return job.Config.Controllable
}

// GetPhase returns a snapshot of the job's current phase.
func (job *baseJob) GetPhase() JobPhase {
	job.phaseMu.Lock()
	defer job.phaseMu.Unlock()
	return job.phase
}

func (job *baseJob) SetPhase(reason JobPhaseReason) {
	job.phaseMu.Lock()
	defer job.phaseMu.Unlock()
	if job.phase.Reason == reason {
		return
	}
	job.phase = JobPhase{Reason: reason, LastChange: time.Now()}
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
	job.stopExpected.Store(false)
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

		layout := job.resolveTimestampLayout(l)
		go job.logWithTimestamp(stdoutPipe, job.stdout, layout)
		go job.logWithTimestamp(stderrPipe, job.stderr, layout)
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

	// Only set job.cmd if cmd.Start() was successful
	job.cmd = cmd
	registerManagedPid(cmd.Process.Pid, job.Config.Name)

	if process != nil {
		process <- job.cmd.Process
	}

	// buffered so the wait goroutine can always deliver its result and exit,
	// even when startOnce returns through the force-kill path without reading
	errChan := make(chan error, 1)

	go func() {
		err := job.cmd.Wait()
		unregisterManagedPid(cmd.Process.Pid)
		errChan <- err
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

		if job.restart.Load() {
			l.Info("job stopped for restart")
			job.restart.Store(false)
			return ProcessWillBeRestartedError
		}

		if job.stop.Load() {
			l.Info("job stopped")
			return ProcessWillBeStoppedError
		}

		if err != nil {
			if job.stopExpected.Load() && terminatedBySignal(err, syscall.SIGTERM) {
				job.stopExpected.Store(false)
				l.Info("process stopped as requested")
				return ProcessStoppedIntentionallyError
			}
			l.WithError(err).Error("job exited with error")
		}
		return err
	case <-ctx.Done():
		// ctx canceled, try to terminate job
		_ = syscall.Kill(-job.cmd.Process.Pid, syscall.SIGTERM)
		l.Info("sent SIGTERM to job's process group")

		select {
		case <-time.After(time.Second * ShutdownWaitingTimeSeconds):
			// process seems to hang, kill process
			_ = syscall.Kill(-job.cmd.Process.Pid, syscall.SIGKILL)
			l.Error("forcefully killed job")
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

// resolveTimestampLayout determines the timestamp layout for the job's log
// output and logs the choice once per process start. timestamp.layout always
// carries the effective Go time layout; timestamp.format the configured key.
func (job *baseJob) resolveTimestampLayout(l *log.Entry) string {
	if job.Config.CustomTimestampFormat != "" {
		l.WithField("timestamp.layout", job.Config.CustomTimestampFormat).
			Info("logging with custom timestamp layout")
		return job.Config.CustomTimestampFormat
	}

	layout, exists := TimeLayouts[job.Config.TimestampFormat]
	if !exists {
		l.WithField("timestamp.format", job.Config.TimestampFormat).
			WithField("timestamp.layout", time.RFC3339).
			Warn("unknown timestamp format, defaulting to RFC3339")
		return time.RFC3339
	}

	l.WithField("timestamp.format", job.Config.TimestampFormat).
		WithField("timestamp.layout", layout).
		Info("logging with timestamp layout")
	return layout
}

func (job *baseJob) logWithTimestamp(r io.Reader, w io.Writer, layout string) {
	l := log.WithField("job.name", job.Config.Name)

	scanner := bufio.NewScanner(r)
	prefix := []byte{'['}
	suffix := []byte{']', ' '}
	newline := []byte{'\n'}

	var timeBuffer []byte
	var lineBuffer bytes.Buffer

	for scanner.Scan() {
		// Reuse time buffer, completly avoiding allocations
		timeBuffer = timeBuffer[:0]
		timeBuffer = time.Now().AppendFormat(timeBuffer, layout)

		// Reset line buffer
		lineBuffer.Reset()
		lineBuffer.Write(prefix)
		lineBuffer.Write(timeBuffer)
		lineBuffer.Write(suffix)
		lineBuffer.Write(scanner.Bytes())
		lineBuffer.Write(newline)

		if _, err := w.Write(lineBuffer.Bytes()); err != nil {
			l.WithError(err).Error("error writing log line for process")
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		l.WithError(err).Error("error reading from process")
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
