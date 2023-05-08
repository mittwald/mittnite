package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/tidwall/pretty"
	"io"
	"net/http"
	"strings"
)

var _ APIResponse = &TypedAPIResponse[struct{}]{}

type TypedAPIResponse[TBody any] struct {
	StatusCode  int   `json:"statusCode"`
	Body        TBody `json:"body"`
	Error       error `json:"error"`
	contentType string
}

func NewTypedAPIResponse[TBody any](body TBody) func(resp *http.Response, err error) *TypedAPIResponse[TBody] {
	return func(resp *http.Response, err error) *TypedAPIResponse[TBody] {
		apiRes := TypedAPIResponse[TBody]{
			Error: err,
		}
		if resp == nil {
			return &apiRes
		}

		apiRes.StatusCode = resp.StatusCode
		apiRes.contentType = resp.Header.Get("Content-Type")

		out, err := io.ReadAll(resp.Body)
		if err != nil {
			apiRes.Error = fmt.Errorf("failed to parse body: %s", err.Error())
			return &apiRes
		}

		switch strings.Split(resp.Header.Get("Content-Type"), ";")[0] {
		case "application/json":
			if err := json.Unmarshal(out, &body); err != nil {
				apiRes.Error = errors.Wrapf(err, "failed to parse body as JSON")
				return &apiRes
			}
		case "text/plain":
			apiRes.Error = fmt.Errorf(strings.TrimSpace(string(out)))
			return &apiRes
		default:
			apiRes.Error = fmt.Errorf("unknown content type %s", strings.Split(resp.Header.Get("Content-Type"), ";")[0])
			return &apiRes
		}

		apiRes.Body = body

		return &apiRes
	}
}

func (resp *TypedAPIResponse[TBody]) Err() error {
	return resp.Error
}

func (resp *TypedAPIResponse[TBody]) Print() error {
	var out string
	if resp.Error != nil {
		fmt.Println(resp.Error.Error())
		return nil
	}

	jsonBody, err := json.Marshal(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal body as JSON")
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, jsonBody, "", "    "); err != nil {
		return err
	}
	out = string(pretty.Color(buf.Bytes(), nil))

	fmt.Println(out)
	return nil
}
