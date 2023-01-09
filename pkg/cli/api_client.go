package cli

import (
	"context"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
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
		return api.JobLogs(job, true)
	default:
		return &CommonApiResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "",
			Error:      fmt.Errorf("unknown action %s", action),
		}
	}
}

func (api *ApiClient) JobStart(job string) ApiResponse {
	client, url, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	url.Path = fmt.Sprintf("/v1/job/%s/start", job)
	return NewApiResponse(client.Post(url.String(), "application/json", nil))
}

func (api *ApiClient) JobRestart(job string) ApiResponse {
	client, url, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}
	url.Path = fmt.Sprintf("/v1/job/%s/restart", job)
	return NewApiResponse(client.Post(url.String(), "application/json", nil))
}

func (api *ApiClient) JobStop(job string) ApiResponse {
	client, url, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	url.Path = fmt.Sprintf("/v1/job/%s/stop", job)
	return NewApiResponse(client.Post(url.String(), "application/json", nil))
}

func (api *ApiClient) JobStatus(job string) ApiResponse {
	client, url, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	url.Path = fmt.Sprintf("/v1/job/%s/status", job)
	return NewApiResponse(client.Get(url.String()))
}

func (api *ApiClient) JobList() ApiResponse {
	client, url, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	url.Path = "/v1/jobs"
	return NewApiResponse(client.Get(url.String()))
}

func (api *ApiClient) JobLogs(job string, follow bool) ApiResponse {
	dialer, url, err := api.buildWebsocketAddress()
	if err != nil {
		return &CommonApiResponse{Error: err}
	}

	url.Path = fmt.Sprintf("/v1/job/%s/logs", job)
	if follow {
		url.RawQuery = "follow=true"
	}

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
	return NewStreamingApiResponse(url, dialer, handler)
}
