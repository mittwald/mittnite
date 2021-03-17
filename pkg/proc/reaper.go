package proc

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func ReapChildren() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGCHLD)

	for {
		sig := <-signals
		log.WithField("signal", sig).Info("handling signal")
		for {
			var s syscall.WaitStatus

			pid, err := syscall.Wait4(-1, &s, 0, nil)
			for syscall.EINTR == err {
				pid, err = syscall.Wait4(pid, &s, 0, nil)
			}

			if syscall.ECHILD == err {
				break
			}

			log.WithField("pid", pid).WithField("status", s).
				Info("reaped child")
		}
	}
}
