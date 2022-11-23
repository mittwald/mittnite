package pidfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type PIDFile struct {
	path string
	fd   int
}

func New(path string) PIDFile {
	return PIDFile{
		path: path,
	}
}

func (f PIDFile) Acquire() error {
	if f.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return errors.Wrapf(err, "failed to create pid file directory %q", filepath.Dir(f.path))
	}

	fd, err := syscall.Open(f.path, syscall.O_CREAT|syscall.O_EXCL|syscall.O_WRONLY, 0o644)
	switch err {
	case syscall.EEXIST:
		if err := f.removePIDFileIfOutdated(); err != nil {
			return err
		}

		return f.Acquire()
	case nil:
		if _, err := syscall.Write(fd, pidToByteString()); err != nil {
			return errors.Wrapf(err, "failed to write pid to pid file %q", f.path)
		}

		log.Info("acquired pid file ", f.path)
	default:
		return errors.Wrapf(err, "failed to open pid file %q", f.path)
	}

	f.fd = fd
	return nil
}

func (f PIDFile) removePIDFileIfOutdated() error {
	pidStr, err := os.ReadFile(f.path)
	if err != nil {
		return errors.Wrapf(err, "failed to read pid file '%s'", f.path)
	}

	pid, err := strconv.Atoi(string(pidStr))
	if err != nil {
		return errors.Wrapf(err, "failed to parse pid file '%s'", f.path)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return errors.Wrapf(err, "failed to find process with pid %d", pid)
	}

	if err := process.Signal(syscall.Signal(0)); err == nil {
		return fmt.Errorf("pid file %q already exists and contains the PID of a running process", f.path)
	}

	log.Info("existing pid file contains the PID of a non-running process; removing it")

	if err := os.Remove(f.path); err != nil {
		return errors.Wrapf(err, "failed to remove pid file %q", f.path)
	}

	return nil
}

func (f PIDFile) Release() error {
	if f.path == "" {
		return nil
	}

	if err := syscall.Close(f.fd); err != nil {
		return errors.Wrapf(err, "failed to close pid file %q", f.path)
	}

	if err := os.Remove(f.path); err != nil {
		return errors.Wrapf(err, "failed to remove pid file %q", f.path)
	}

	log.Info("released pid file ", f.path)
	return nil
}

func pidToByteString() []byte {
	return []byte(fmt.Sprintf("%d", os.Getpid()))
}
