package proc

import (
	"context"
	"sync"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

func NewRunner(ctx context.Context, ignitionConfig *config.Ignition) *Runner {
	return &Runner{
		IgnitionConfig: ignitionConfig,
		ctx:            ctx,
	}
}

func (r *Runner) exec(ctx context.Context, wg *sync.WaitGroup, errChan chan error) {
	for j := range r.IgnitionConfig.Jobs {
		job := &r.IgnitionConfig.Jobs[j]

		job.Init()

		// execute job command
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := job.Run(ctx)
			if err != nil {
				errChan <- err
			}
		}()
	}
}

func (r *Runner) Run() error {
	errChan := make(chan error)
	ticker := time.NewTicker(5 * time.Second)

	wg := sync.WaitGroup{}

	r.exec(r.ctx, &wg, errChan)

	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDone)
	}()

	for {
		select {
		// wait for them all to finish, or one to fail
		case <-allDone:
			return nil

		// watch files
		case <-ticker.C:
			for _, job := range r.IgnitionConfig.Jobs {
				job.Watch()
			}

		// handle errors
		case err := <-errChan:
			log.Error("job return error, shutting down other services")
			return err
		}
	}
}
