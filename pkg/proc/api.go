package proc

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	contextKeyJob = "job"
)

func (api *Api) RegisterHandler(router *mux.Router, path string, methods []string, handler func(http.ResponseWriter, *http.Request)) {
	router.
		Path(path).
		HandlerFunc(handler).
		Methods(methods...)
}

func (api *Api) RegisterMiddlewareFuncs(middlewareFunc ...mux.MiddlewareFunc) {
	api.router.Use(middlewareFunc...)
}

func (api *Api) Start() error {
	api.srv = &http.Server{
		Addr:    api.listenAddr,
		Handler: api.router,
	}

	log.Infof("remote api listens on %s", api.srv.Addr)
	if err := api.listen(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (api *Api) Shutdown() error {
	if api.srv == nil {
		return nil
	}

	log.Info("shutting down remote api")
	return api.srv.Shutdown(context.Background())
}

func (api *Api) listen() error {
	socketParts := strings.Split(api.srv.Addr, "unix://")
	if len(socketParts) <= 1 {
		return api.listenOnPort()
	}

	return api.listenOnUnixSocket(socketParts[1])
}

func (api *Api) listenOnUnixSocket(socketFile string) error {
	socketDir := path.Dir(socketFile)
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return errors.Wrap(err, "failed to prepare folder for socket-file")
	}
	conn, err := net.Listen("unix", socketFile)
	if err != nil {
		return err
	}
	return api.srv.Serve(conn)
}

func (api *Api) listenOnPort() error {
	return api.srv.ListenAndServe()
}
