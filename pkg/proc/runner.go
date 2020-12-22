package proc

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

func NewRunner(ignitionConfig *config.Ignition) *Runner {
	return &Runner{
		IgnitionConfig: ignitionConfig,
		jobs:           make([]*Job, 0, len(ignitionConfig.Jobs)),
		bootJobs:       make([]*BootJob, 0, len(ignitionConfig.BootJobs)),
		shutdownChan:   make(chan error, 1),
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
	errs := make(chan error)

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
				errs <- err
			}
		}(job, ctx)
	}

	select {
	case <-waitGroupToChannel(&wg):
		return nil

	case <-ctx.Done():
		log.Warn("context cancelled")
		return ctx.Err()

	case err, ok := <-errs:
		if ok && err != nil {
			log.Error(err)
			r.Shutdown(errors.New(RunnerShuwtdownCause))
			return err
		}
	}

	return nil
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
			r.Shutdown(errors.New(RunnerShuwtdownCause))

		case <-r.shutdownChan:
			wg := sync.WaitGroup{}
			for i := range r.jobs {
				wg.Add(1)
				go func(job *Job) {
					job.Stop()
					wg.Done()
				}(r.jobs[i])
			}
			wg.Wait()
			return nil
		}
	}
}

func (r *Runner) Shutdown(err error) {
	r.shutdownChan <- err
}
