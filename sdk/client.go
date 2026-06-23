// Package sdk routes model-produced JSON requests to a mymcp server.
package sdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/phillip-england/mymcp/internal/protocol"
)

// Outcome describes why an agentic loop terminated.
type Outcome string

const (
	OutcomeNone    Outcome = ""
	OutcomeError   Outcome = protocol.TerminalError
	OutcomeSuccess Outcome = protocol.TerminalSuccess
)

// Response contains the complete response returned by the mymcp server.
// Non-2xx HTTP responses are returned here rather than converted into SDK errors.
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	Terminal   bool
	Outcome    Outcome
}

// Text returns the response body as a string.
func (r *Response) Text() string {
	return string(r.Body)
}

// OK reports whether the server returned a successful HTTP status.
func (r *Response) OK() bool {
	return r.StatusCode >= http.StatusOK && r.StatusCode < http.StatusMultipleChoices
}

// Client sends validated JSON requests to one mymcp server.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewClient constructs an SDK client.
func NewClient(baseURL string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("base URL must be an absolute HTTP or HTTPS URL")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("base URL must not contain a path, query, or fragment")
	}
	parsed.Path = ""
	return &Client{baseURL: parsed, httpClient: http.DefaultClient}, nil
}

// WithHTTPClient replaces the HTTP transport used by the client. A nil value
// restores http.DefaultClient.
func (c *Client) WithHTTPClient(client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	c.httpClient = client
	return c
}

// Route validates a model-produced JSON string, maps it to a known mymcp
// endpoint, sends it, and returns the complete server response.
func (c *Client) Route(ctx context.Context, payload string) (*Response, error) {
	return c.routeJSON(ctx, []byte(payload))
}

// RouteJSON is the byte-oriented form of Route.
func (c *Client) RouteJSON(ctx context.Context, payload []byte) (*Response, error) {
	return c.routeJSON(ctx, payload)
}

func (c *Client) routeJSON(ctx context.Context, payload []byte) (*Response, error) {
	requestSpec, target, err := decodeAndRoute(payload)
	if err != nil {
		return nil, err
	}
	destination := c.baseURL.ResolveReference(target)
	request, err := http.NewRequestWithContext(ctx, requestSpec.Method, destination.String(), strings.NewReader(requestSpec.Body))
	if err != nil {
		return nil, fmt.Errorf("build HTTP request: %w", err)
	}
	for name, value := range requestSpec.Headers {
		request.Header.Set(name, value)
	}

	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpResponse, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send mymcp request: %w", err)
	}
	defer httpResponse.Body.Close()
	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("read mymcp response: %w", err)
	}
	outcome := OutcomeNone
	switch httpResponse.Header.Get(protocol.TerminalHeader) {
	case protocol.TerminalError:
		outcome = OutcomeError
	case protocol.TerminalSuccess:
		outcome = OutcomeSuccess
	}
	return &Response{
		StatusCode: httpResponse.StatusCode,
		Header:     httpResponse.Header.Clone(),
		Body:       body,
		Terminal:   outcome != OutcomeNone,
		Outcome:    outcome,
	}, nil
}
