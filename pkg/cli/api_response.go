package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/pretty"
	"io"
	"net/http"
)

type ApiResponse interface {
	Print() error
}

var _ ApiResponse = &CommonApiResponse{}

type CommonApiResponse struct {
	StatusCode  int    `json:"statusCode"`
	Body        string `json:"body"`
	Error       error  `json:"error"`
	contentType string
}

func NewApiResponse(resp *http.Response, err error) ApiResponse {
	apiRes := &CommonApiResponse{
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
	return apiRes
}

func (resp *CommonApiResponse) Print() error {
	var out string
	if resp.Error != nil {
		fmt.Println(resp.Error.Error())
		return nil
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
