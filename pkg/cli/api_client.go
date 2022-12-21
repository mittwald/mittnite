package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

const (
	ApiActionJobStart   = "start"
	ApiActionJobRestart = "restart"
	ApiActionJobStop    = "stop"
	ApiActionJobStatus  = "status"
)

type ApiClient struct {
	apiAddress string
}

func NewApiClient(apiAddress string) *ApiClient {
	return &ApiClient{
		apiAddress: apiAddress,
	}
}

func (api *ApiClient) CallAction(job, action string) *ApiResponse {
	switch action {
	case ApiActionJobStart:
		return api.JobStart(job)
	case ApiActionJobRestart:
		return api.JobRestart(job)
	case ApiActionJobStop:
		return api.JobStop(job)
	case ApiActionJobStatus:
		return api.JobStatus(job)
	default:
		return &ApiResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "",
			Error:      fmt.Errorf("unknown action %s", action),
		}
	}
}

func (api *ApiClient) JobStart(job string) *ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &ApiResponse{Error: err}
	}
	return NewApiResponse(client.Post(fmt.Sprintf("%s/v1/job/%s/start", addr, job), "application/json", nil))
}

func (api *ApiClient) JobRestart(job string) *ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &ApiResponse{Error: err}
	}
	return NewApiResponse(client.Post(fmt.Sprintf("%s/v1/job/%s/restart", addr, job), "application/json", nil))
}

func (api *ApiClient) JobStop(job string) *ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &ApiResponse{Error: err}
	}
	return NewApiResponse(client.Post(fmt.Sprintf("%s/v1/job/%s/stop", addr, job), "application/json", nil))
}

func (api *ApiClient) JobStatus(job string) *ApiResponse {
	client, addr, err := api.buildHttpClientAndAddress()
	if err != nil {
		return &ApiResponse{Error: err}
	}

	return NewApiResponse(client.Get(fmt.Sprintf("%s/v1/job/%s/status", addr, job)))
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
