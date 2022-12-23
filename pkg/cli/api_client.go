package cli

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
)

const (
	ApiActionJobStart   = "start"
	ApiActionJobRestart = "restart"
	ApiActionJobStop    = "stop"
	ApiActionJobStatus  = "status"
	ApiActionJobLogs    = "logs"
)

type ApiClient struct {
	apiAddress string
}

func NewApiClient(apiAddress string) *ApiClient {
	return &ApiClient{
		apiAddress: apiAddress,
	}
}

func (api *ApiClient) CallAction(job, action string) ApiResponse {
	switch action {
	case ApiActionJobStart:
		return api.JobStart(job)
	case ApiActionJobRestart:
		return api.JobRestart(job)
	case ApiActionJobStop:
		return api.JobStop(job)
	case ApiActionJobStatus:
		return api.JobStatus(job)
	case ApiActionJobLogs:
		return api.JobLogs(job)
	default:
		return &CommonApiResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "",
			Error:      fmt.Errorf("unknown action %s", action),
		}
	}
}

func (api *ApiClient) JobStart(job string) ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}
	return NewApiResponse(client.Post(fmt.Sprintf("%s/v1/job/%s/start", addr, job), "application/json", nil))
}

func (api *ApiClient) JobRestart(job string) ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}
	return NewApiResponse(client.Post(fmt.Sprintf("%s/v1/job/%s/restart", addr, job), "application/json", nil))
}

func (api *ApiClient) JobStop(job string) ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}
	return NewApiResponse(client.Post(fmt.Sprintf("%s/v1/job/%s/stop", addr, job), "application/json", nil))
}

func (api *ApiClient) JobStatus(job string) ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	return NewApiResponse(client.Get(fmt.Sprintf("%s/v1/job/%s/status", addr, job)))
}

func (api *ApiClient) JobLogs(job string) ApiResponse {
	dialer, addr, err := api.buildWebsocketAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	urlStr := fmt.Sprintf("%s/v1/job/%s/logs", addr, job)
	handler := func(ctx context.Context, conn *websocket.Conn, msgChan chan []byte, errChan chan error) {
		for {
			select {
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					errChan <- err
					return
				}
				msgChan <- msg
			case <-ctx.Done():
				return
			}
		}
	}
	return NewStreamingApiResponse(urlStr, dialer, handler)
}

func (api *ApiClient) buildHttpClientAndAddress() (*http.Client, string, error) {
	u, err := url.Parse(api.apiAddress)
	if err != nil {
		return nil, "", err
	}
	if u.Scheme != "unix" {
		return &http.Client{}, api.apiAddress, nil
	}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", u.Path)
			},
		},
	}, "http://unix", nil
}

func (api *ApiClient) buildWebsocketAddress() (*websocket.Dialer, string, error) {
	u, err := url.Parse(api.apiAddress)
	if err != nil {
		return nil, "", err
	}
	if u.Scheme != "unix" {
		u.Scheme = "ws"
		return websocket.DefaultDialer, u.String(), nil
	}

	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial("unix", u.Path)
		},
	}

	return dialer, "ws://unix", nil
}
