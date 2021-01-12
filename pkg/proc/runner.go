package proc

import (
	"context"
	"sync"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

func NewRunner(ignitionConfig *config.Ignition) *Runner {
	return &Runner{
		IgnitionConfig: ignitionConfig,
		jobs:           make([]*Job, 0, len(ignitionConfig.Jobs)),
		bootJobs:       make([]*BootJob, 0, len(ignitionConfig.BootJobs)),
	}
}

func waitGroupToChannel(wg *sync.WaitGroup) <-chan struct{} {
	d := make(chan struct{})
	go func() {
		wg.Wait()
		close(d)
	}()

	return d
}

func (r *Runner) Boot(ctx context.Context) error {
	wg := sync.WaitGroup{}
	bootErrs := make(chan error)

	for j := range r.IgnitionConfig.BootJobs {
		job, err := NewBootJob(&r.IgnitionConfig.BootJobs[j])
		if err != nil {
			return err
		}

		r.bootJobs = append(r.bootJobs, job)
	}

	for _, job := range r.bootJobs {
		wg.Add(1)
		go func(job *BootJob, ctx context.Context) {
			defer wg.Done()

			if job.timeout != 0 {
				toCtx, cancel := context.WithTimeout(ctx, job.timeout)
				defer cancel()

				ctx = toCtx
			}

			if err := job.Run(ctx); err != nil {
				bootErrs <- err
			}
		}(job, ctx)
	}

	select {
	case <-waitGroupToChannel(&wg):
		return nil

	case <-ctx.Done():
		log.Warn("context cancelled")
		return ctx.Err()

	case err := <-bootErrs:
		log.Error("boot job error occurred: ", err)
		return err
	}
}

func (r *Runner) exec(ctx context.Context, wg *sync.WaitGroup, errChan chan<- error) {
	for j := range r.IgnitionConfig.Jobs {
		job, err := NewJob(&r.IgnitionConfig.Jobs[j])
		if err != nil {
			errChan <- err
			return
		}

		job.Init()

		r.jobs = append(r.jobs, job)

		// execute job command
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := job.Run(ctx, errChan)
			if err != nil {
				errChan <- err
			}
		}()
	}
}

func (r *Runner) Run(ctx context.Context) error {
	errChan := make(chan error)
	ticker := time.NewTicker(5 * time.Second)

	wg := sync.WaitGroup{}

	r.exec(ctx, &wg, errChan)

	for {
		select {
		// wait for them all to finish, or one to fail
		case <-waitGroupToChannel(&wg):
			return nil

		// watch files
		case <-ticker.C:
			for _, job := range r.jobs {
				job.Watch()
			}

		// handle errors
		case err := <-errChan:
			log.Error(err)

		}
	}
}
