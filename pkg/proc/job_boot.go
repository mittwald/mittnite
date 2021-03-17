package proc

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (job *BootJob) Run(ctx context.Context) error {
	l := log.WithField("job.name", job.Config.Name)
	err := job.startOnce(ctx, nil)
	if job.Config.CanFail {
		l.WithError(err).Warn("job failed, but is allowed to fail")
		return nil
	}
	return err
}
