package proc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

func NewRunner(ctx context.Context, api *Api, keepRunning bool, ignitionConfig *config.Ignition) *Runner {
	return &Runner{
		IgnitionConfig: ignitionConfig,
		ctx:            ctx,
		jobs:           []Job{},
		bootJobs:       make([]*BootJob, 0, len(ignitionConfig.BootJobs)),
		api:            api,
		keepRunning:    keepRunning,
	}
}

func (r *Runner) StartAPI() error {
	return r.startAPIV1()
}

func waitGroupToChannel(wg *sync.WaitGroup) <-chan struct{} {
	d := make(chan struct{})
	go func() {
		wg.Wait()
		close(d)
	}()

	return d
}

func (r *Runner) Boot() error {
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
		}(job, r.ctx)
	}

	select {
	case <-waitGroupToChannel(&wg):
		return nil

	case <-r.ctx.Done():
		log.Warn("context cancelled")
		return r.ctx.Err()

	case err := <-bootErrs:
		log.Error("boot job error occurred: ", err)
		return err
	}
}

func (r *Runner) Run() error {
	r.errChan = make(chan error)
	r.waitGroup = &sync.WaitGroup{}
	if r.keepRunning {
		r.waitGroup.Add(1)
		defer r.waitGroup.Done()
	}
	ticker := time.NewTicker(5 * time.Second)

	r.exec()

	wgChan := waitGroupToChannel(r.waitGroup)
	for {
		select {
		case <-r.ctx.Done():
			log.Warn("context cancelled")
			return r.ctx.Err()

		// wait for them all to finish, or one to fail
		case <-wgChan:
			return nil

		// watch files
		case <-ticker.C:
			for _, job := range r.jobs {
				job.Watch()
				if r.keepRunning {
					commonJob, ok := job.(*CommonJob)
					if ok && !commonJob.IsRunning() {
						r.removeJob(job)
						r.addAndStartJob(job)
					}
				}
			}

		// handle errors
		case err := <-r.errChan:
			log.Error(err)
			return err
		}
	}
}

func (r *Runner) Init() error {
	for j := range r.IgnitionConfig.Jobs {
		var job Job
		var err error

		// init non-lazy jobs
		if r.IgnitionConfig.Jobs[j].Laziness == nil {
			job, err = NewCommonJob(&r.IgnitionConfig.Jobs[j])
		} else {
			job, err = NewLazyJob(&r.IgnitionConfig.Jobs[j])
		}
		if err != nil {
			return fmt.Errorf("error initializing job %s: %w", r.IgnitionConfig.Jobs[j].Name, err)
		}
		r.addJobIfNotExists(job)
	}

	return nil
}

func (r *Runner) exec() {
	for i := range r.jobs {
		r.startJob(r.jobs[i])
	}
}

func (r *Runner) jobExistsAndIsControllable(job *CommonJob) bool {
	return job != nil && job.IsControllable()
}

func (r *Runner) addAndStartJob(job Job) {
	r.addJobIfNotExists(job)
	r.startJob(job)
}

func (r *Runner) addJobIfNotExists(job Job) {
	for _, j := range r.jobs {
		if j.GetName() == job.GetName() {
			return
		}
	}
	r.jobs = append(r.jobs, job)
}

func (r *Runner) startJob(job Job) {
	job.Init()

	r.waitGroup.Add(1)
	go func() {
		defer func() {
			r.waitGroup.Done()
		}()

		if err := job.Run(r.ctx, r.errChan); err != nil {
			r.errChan <- err
		}
	}()
}

func (r *Runner) removeJob(job Job) {
	for i, j := range r.jobs {
		if j.GetName() == job.GetName() {
			r.jobs[i] = r.jobs[len(r.jobs)-1]
			r.jobs[len(r.jobs)-1] = nil
			r.jobs = r.jobs[:len(r.jobs)-1]
			return
		}
	}
}

func (r *Runner) findCommonJobByName(name string) *CommonJob {
	for i, job := range r.jobs {
		if job.GetName() == name {
			commonJob, ok := r.jobs[i].(*CommonJob)
			if !ok {
				return nil
			}
			return commonJob
		}
	}
	return nil
}

func (r *Runner) findCommonIgnitionJobByName(name string) (*CommonJob, error) {
	for i, ignJob := range r.IgnitionConfig.Jobs {
		if ignJob.Name == name && ignJob.Laziness == nil {
			return NewCommonJob(&r.IgnitionConfig.Jobs[i])
		}
	}
	return nil, fmt.Errorf("can't find ignition config for job %q", name)
}
