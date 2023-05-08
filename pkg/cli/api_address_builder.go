package cli

import (
	"context"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
)

func (api *APIClient) buildHTTPClientAndURL() (*http.Client, *url.URL, error) {
	u, err := url.Parse(api.apiAddress)
	if err != nil {
		return nil, nil, err
	}
	if u.Scheme != "unix" {
		return &http.Client{}, u, nil
	}

	socketPath := u.Path
	u.Scheme = "http"
	u.Host = "unix"
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}, u, nil
}

func (api *APIClient) buildWebsocketURL() (*websocket.Dialer, *url.URL, error) {
	u, err := url.Parse(api.apiAddress)
	if err != nil {
		return nil, nil, err
	}
	if u.Scheme != "unix" {
		u.Scheme = "ws"
		return websocket.DefaultDialer, u, nil
	}
	socketPath := u.Path

	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}

	u.Scheme = "ws"
	u.Host = "unix"
	return dialer, u, nil
}
