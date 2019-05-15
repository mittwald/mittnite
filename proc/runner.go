package proc

import (
	"github.com/mittwald/mittnite/config"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

func RunServices(cfg *config.IgnitionConfig, signals chan os.Signal) error {
	signalsOut := make([]chan os.Signal, len(cfg.Jobs))
	for i := range cfg.Jobs {
		signalsOut[i] = make(chan os.Signal)
	}

	stop := false

	go func() {
		for sig := range signals {
			log.Printf("jobrunner: received signal %s", sig.String())
			if sig == syscall.SIGINT || sig == syscall.SIGTERM {
				log.Printf("stopping service runner")
				stop = true
			}

			for i := range cfg.Jobs {
				signalsOut[i] <- sig
			}
		}
	}()

	wg := sync.WaitGroup{}

	for i := range cfg.Jobs {
		wg.Add(1)

		var cmd *exec.Cmd

		go func(job *config.JobConfig, signals <-chan os.Signal) {
			for sig := range signals {
				if cmd != nil && cmd.Process != nil {
					log.Printf("passing signal %s to job %s", sig.String(), job.Name)
					cmd.Process.Signal(sig)
				}
			}
		}(&cfg.Jobs[i], signalsOut[i])

		for i := range cfg.Jobs[i].Watches {
			go func(j int, w *config.WatchConfig) {
				var mtime time.Time
				stat, err := os.Stat(w.Filename)

				if err == nil {
					log.Printf("file %s's last modification was %s", w.Filename, stat.ModTime().String())
					mtime = stat.ModTime()
				}

				timer := time.NewTicker(15 * time.Second)

				for range timer.C {
					stat, err = os.Stat(w.Filename)
					if err == nil && mtime != stat.ModTime() && cmd != nil && cmd.Process != nil {
						log.Printf("file %s changed, signalling process %s", w.Filename, cfg.Jobs[j].Name)
						cmd.Process.Signal(syscall.Signal(w.Signal))
						mtime = stat.ModTime()
					}
				}
			}(i, &cfg.Jobs[i].Watches[i])
		}

		go func(job config.JobConfig, s <-chan os.Signal) {
			maxAttempts := job.MaxAttempts
			failedAttempts := 0

			if maxAttempts == 0 {
				maxAttempts = 3
			}

			for !stop {
				log.Printf("starting job %s", job.Name)

				cmd = exec.Command(job.Command, job.Args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				err := cmd.Start()

				err = cmd.Wait()
				if err != nil {
					log.Printf("job %s exited with error: %s", job.Name, err)
					failedAttempts++

					if failedAttempts >= maxAttempts {
						log.Printf("reached max retries for job %s", job.Name)
						break
					}
				}
			}

			log.Printf("ending job %s", job.Name)

			wg.Done()
		}(cfg.Jobs[i], signalsOut[i])
	}

	wg.Wait()
	return nil
}
