package proc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/mittwald/mittnite/internal/config"
	log "github.com/sirupsen/logrus"
)

type Listener struct {
	config        *config.Listener
	job           *LazyJob
	socket        net.Listener
	spinUpTimeout time.Duration
}

type acceptResult struct {
	conn net.Conn
	err  error
}

func NewListener(j *LazyJob, c *config.Listener) (*Listener, error) {
	log.WithField("address", c.Address).Info("starting TCP listener")

	// deprecation check
	if c.Protocol != "" {
		if c.ForwardProtocol == "" {
			log.Warnf("field protocol in job %s is deprecated in favor of forwardProtocol", j.Config.Name)
			c.ForwardProtocol = c.Protocol
		} else {
			log.Warnf("field protocol in job %s is ignored because it is deprecated and forwardProtocol is already set", j.Config.Name)
		}
	}

	listener, err := net.Listen(getProto(c.ListenProtocol), c.Address)
	if err != nil {
		return nil, err
	}

	return &Listener{
		config:        c,
		job:           j,
		socket:        listener,
		spinUpTimeout: j.spinUpTimeout,
	}, nil
}

func (l *Listener) Run(ctx context.Context) error {
	runErrors := l.run(ctx)

	select {
	case err := <-runErrors:
		return err
	case <-ctx.Done():
		return errors.New("context closed")
	}
}

func (l *Listener) provideUpstreamConnection() (net.Conn, error) {
	timeout := time.NewTimer(l.spinUpTimeout)
	ticker := time.NewTicker(20 * time.Millisecond)

	defer ticker.Stop()
	defer timeout.Stop()

	for {
		select {
		case <-ticker.C:
			conn, err := net.Dial(getProto(l.config.ForwardProtocol), l.config.Forward)
			if err == nil {
				return conn, nil
			}
		case <-timeout.C:
			return nil, fmt.Errorf("job %s did not start after %s", l.job.Config.Name, l.spinUpTimeout)
		}
	}
}

func (l *Listener) run(ctx context.Context) <-chan error {
	errChan := make(chan error)

	go func() {
		for {
			var conn net.Conn
			connChan := make(chan acceptResult, 1)
			go func() {
				conn, err := l.socket.Accept()
				connChan <- acceptResult{
					conn: conn,
					err:  err,
				}
			}()

			select {
			case <-ctx.Done():
				// received sigterm before new connection could have been established,
				// we are about to shut down, close socket and listener and return
				if err := l.socket.Close(); err != nil {
					log.WithField("reason", err.Error()).Warn("cannot reliably close socket")
				}
				return
			case ar := <-connChan:
				if ar.err != nil {
					errChan <- ar.err
					return
				}
				conn = ar.conn
			}

			log.WithField("client.addr", conn.RemoteAddr()).Info("accepted connection")

			if err := l.job.AssertStarted(ctx); err != nil {
				errChan <- err
				return
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
				defer upstream.Close()

				toUpstreamErrors := make(chan error)
				fromUpstreamErrors := make(chan error)

				go func() {
					defer func() {
						if tcpUpstream, ok := upstream.(*net.TCPConn); ok {
							tcpUpstream.CloseWrite()
						}
						close(toUpstreamErrors)
					}()

					if _, err := io.Copy(upstream, conn); err != nil && !errors.Is(err, io.EOF) {
						toUpstreamErrors <- err
					}
				}()

				go func() {
					defer func() {
						if tcpConn, ok := conn.(*net.TCPConn); ok {
							tcpConn.CloseWrite()
						}
						close(fromUpstreamErrors)
					}()

					if _, err := io.Copy(conn, upstream); err != nil && !errors.Is(err, io.EOF) {
						fromUpstreamErrors <- err
					}
				}()

				// wait for both channels (do not use `select-case`)
				<-toUpstreamErrors
				<-fromUpstreamErrors
			}()
		}
	}()

	return errChan
}

func getProto(proto string) string {
	if proto == "" {
		proto = "tcp"
	}
	return proto
}
