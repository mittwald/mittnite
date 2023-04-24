package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"
	"io"
	"net/http"
	"strings"
)

type APIResponse interface {
	Print() error
	Err() error
}

var _ APIResponse = &CommonAPIResponse{}

type CommonAPIResponse struct {
	StatusCode  int    `json:"statusCode"`
	Body        string `json:"body"`
	Error       error  `json:"error"`
	contentType string
}

func NewAPIResponse(resp *http.Response, err error) APIResponse {
	apiRes := &CommonAPIResponse{
		Error: err,
	}
	if resp == nil {
		return apiRes
	}

	apiRes.StatusCode = resp.StatusCode
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to parse body: %s", err.Error())
		return apiRes
	}
	apiRes.Body = string(out)
	apiRes.contentType = resp.Header.Get("Content-Type")

	if err == nil && apiRes.StatusCode >= 400 {
		apiRes.Error = fmt.Errorf("unexpected status code %d: %s", apiRes.StatusCode, strings.TrimSpace(apiRes.Body))
	}

	return apiRes
}

func (resp *CommonAPIResponse) Err() error {
	return resp.Error
}

func (resp *CommonAPIResponse) Print() error {
	var out string
	if resp.Error != nil {
		return resp.Error
	}

	if len(resp.Body) == 0 {
		return nil
	}

	switch resp.contentType {
	default:
		out = resp.Body

	case "application/json":
		var buf bytes.Buffer
		if err := json.Indent(&buf, []byte(resp.Body), "", "    "); err != nil {
			return err
		}
		out = string(pretty.Color(buf.Bytes(), nil))
	}

	fmt.Println(out)
	return nil
}
