package proc

import (
	"context"
	"errors"
	"fmt"
	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type Listener struct {
	ctx               context.Context
	config            *config.Listener
	job               *Job
	socket            net.Listener
	spinUpTimeout     time.Duration
}

func NewListener(ctx context.Context, j *Job, c *config.Listener) (*Listener, error) {
	log.WithField("address", c.Address).Info("starting TCP listener")

	listener, err := net.Listen("tcp", c.Address)
	if err != nil {
		return nil, err
	}

	return &Listener{
		ctx:             ctx,
		config:          c,
		job:             j,
		socket:          listener,
		spinUpTimeout:   j.spinUpTimeout,
	}, nil
}

func (l *Listener) Run() error {
	runErrors := l.run()

	select {
	case err := <-runErrors:
		return err
	case <-l.ctx.Done():
		return errors.New("context closed")
	}
}

func (l *Listener) provideUpstreamConnection() (net.Conn, error) {
	prot := l.config.Protocol
	if prot == "" {
		prot = "tcp"
	}

	timeout := time.NewTimer(l.spinUpTimeout)
	ticker := time.NewTicker(20 * time.Millisecond)

	defer ticker.Stop()
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			conn, err := net.Dial(prot, l.config.Forward)
			if err == nil {
				return conn, nil
			}
		case <-timeout.C:
			return nil, fmt.Errorf("job %s did not start after %s", l.job.Config.Name, l.spinUpTimeout)
		}
	}
}

func (l *Listener) run() <-chan error {
	errChan := make(chan error)

	go func() {
		for {
			var err error

			conn, err := l.socket.Accept()
			if err != nil {
				errChan <- err
				return
			}

			log.WithField("client.addr", conn.RemoteAddr()).Info("accepted connection")

			if l.job.CanStartLazily() {
				if err := l.job.AssertStarted(l.ctx); err != nil {
					errChan <- err
					return
				}
			}

			go func() {
				defer conn.Close()

				atomic.AddUint32(&l.job.activeConnections, 1)
				defer func() {
					// this might be a tiny bit racy, which is fine in this case.
					l.job.lastConnectionClosed = time.Now()
					atomic.AddUint32(&l.job.activeConnections, ^uint32(0))
				}()

				upstream, err := l.provideUpstreamConnection()
				if err != nil {
					log.WithError(err).Error("error while dialling upstream")
					return
				}

				toUpstreamErrors := make(chan error)
				fromUpstreamErrors := make(chan error)

				go func() {
					if _, err := io.Copy(upstream, conn); err != nil && err != io.EOF {
						toUpstreamErrors <- err
					}
					close(toUpstreamErrors)
				}()

				go func() {
					if _, err := io.Copy(conn, upstream); err != nil && err != io.EOF {
						fromUpstreamErrors <- err
					}
					close(fromUpstreamErrors)
				}()

				select {
				case <-toUpstreamErrors:
				case <-fromUpstreamErrors:
					return
				}

				return
			}()
		}
	}()

	return errChan
}
