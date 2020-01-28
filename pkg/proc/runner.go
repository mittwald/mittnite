package proc

import (
	"fmt"
	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func RunServices(cfg *config.Ignition, signals chan os.Signal) error {
	errs := make(chan error)

	signalsOut := make([]chan os.Signal, len(cfg.Jobs))
	for i := range cfg.Jobs {
		signalsOut[i] = make(chan os.Signal)
	}

	stop := false

	go func() {
		for sig := range signals {
			log.Infof("jobrunner: received signal %s", sig.String())
			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				log.Info("stopping service runner")
				stop = true
			}

			for i := range cfg.Jobs {
				signalsOut[i] <- sig
			}
		}
	}()

	wg := sync.WaitGroup{}

	for i, job := range cfg.Jobs {
		var cmd *exec.Cmd

		go func(job *config.JobConfig, signals <-chan os.Signal) {
			for sig := range signals {
				if cmd != nil && cmd.Process != nil {
					log.Infof("passing signal %s to job %s", sig.String(), job.Name)
					_ = cmd.Process.Signal(sig)
				}
			}
		}(&job, signalsOut[i])

		for _, w := range job.Watches {
			go func(jobCfg config.JobConfig, w *config.Watch) {
				var mtime time.Time
				stat, err := os.Stat(w.Filename)

				if err == nil {
					log.Infof("file %s's last modification was %s", w.Filename, stat.ModTime().String())
					mtime = stat.ModTime()
				}

				timer := time.NewTicker(15 * time.Second)

				for range timer.C {
					stat, err = os.Stat(w.Filename)
					if err == nil && mtime != stat.ModTime() && cmd != nil && cmd.Process != nil {
						log.Infof("file %s changed, signalling process %s", w.Filename, jobCfg.Name)
						_ = cmd.Process.Signal(syscall.Signal(w.Signal))
						mtime = stat.ModTime()
					}
				}
			}(job, &w)
		}

		wg.Add(1)
		go func(job config.JobConfig, errs chan<- error) {
			defer wg.Done()

			maxAttempts := job.MaxAttempts
			failedAttempts := 0

			if maxAttempts == 0 {
				maxAttempts = 3
			}

			for !stop {
				log.Infof("starting job %s", job.Name)

				cmd = exec.Command(job.Command, job.Args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				_ = cmd.Start()

				err := cmd.Wait()
				if err != nil {
					if job.CanFail {
						log.Warnf("Failable job %s exited with error: %s", job.Name, err)
					} else {
						log.Errorf("job %s exited with error: %s", job.Name, err)
					}
					failedAttempts++

					if failedAttempts >= maxAttempts {
						if job.CanFail {
							log.Warnf("reached max retries for job %s", job.Name)
							stop = true
						} else {
							log.Errorf("reached max retries for job %s", job.Name)
							errs <- fmt.Errorf("reached max retries for job %s", job.Name)
							break
						}
					}
				}
			}

			log.Infof("ending job %s", job.Name)

		}(cfg.Jobs[i], errs)
	}

	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDone)
	}()

	// wait for them all to finish, or one to fail
	select {
	case <-allDone:
	case err := <-errs:
		log.Error("job return error, shutting down other services")
		signals <- syscall.SIGINT // notify other (already running) jobs to shut down
		return err
	}

	return nil
}
