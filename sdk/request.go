package sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/phillip-england/mymcp/internal/protocol"
)

// JSONRequest is the request shape described by the server's /map endpoint.
type JSONRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

func decodeAndRoute(payload []byte) (JSONRequest, *url.URL, error) {
	request, err := decodeJSONRequest(payload)
	if err != nil {
		return request, nil, err
	}
	target, err := parseRequestTarget(request.Path)
	if err != nil {
		return request, nil, err
	}

	endpoint, ok := protocol.Lookup(target.Path)
	if !ok {
		return request, nil, fmt.Errorf("request does not map to a known endpoint: %s", target.Path)
	}
	request.Method = strings.ToUpper(request.Method)
	if request.Method != endpoint.Method {
		return request, nil, fmt.Errorf("%s requires method %s", target.Path, endpoint.Method)
	}
	query, err := url.ParseQuery(target.RawQuery)
	if err != nil {
		return request, nil, fmt.Errorf("invalid query string: %w", err)
	}
	if err := endpoint.ValidateQuery(query); err != nil {
		return request, nil, err
	}
	return request, target, nil
}

func decodeJSONRequest(payload []byte) (JSONRequest, error) {
	var request JSONRequest
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("decode request JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return request, err
	}
	if request.Method == "" || request.Path == "" {
		return request, errors.New("request JSON requires method and path")
	}
	return request, nil
}

func parseRequestTarget(path string) (*url.URL, error) {
	target, err := url.ParseRequestURI(path)
	if err != nil {
		return nil, fmt.Errorf("invalid request path: %w", err)
	}
	if target.IsAbs() || target.Host != "" || !strings.HasPrefix(target.Path, "/") {
		return nil, errors.New("request path must be relative to the configured mymcp server")
	}
	if target.Fragment != "" {
		return nil, errors.New("request path must not contain a fragment")
	}
	return target, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request JSON must contain exactly one object")
		}
		return fmt.Errorf("decode request JSON: %w", err)
	}
	return nil
}
