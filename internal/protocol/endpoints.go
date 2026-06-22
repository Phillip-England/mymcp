// Package protocol defines the HTTP contract shared by the server and SDK.
package protocol

import "net/http"

const SandboxHeader = "X-Mymcp-Sandbox"

// Endpoint describes one route and the query parameters accepted by the SDK.
type Endpoint struct {
	Method      string
	Path        string
	queryParams map[string]struct{}
}

// Pattern returns the Go 1.22+ ServeMux registration pattern for the endpoint.
func (e Endpoint) Pattern() string {
	if e.Path == "/" {
		return e.Method + " /{$}"
	}
	return e.Method + " " + e.Path
}

// AllowsQuery reports whether the endpoint accepts a query parameter.
func (e Endpoint) AllowsQuery(name string) bool {
	_, ok := e.queryParams[name]
	return ok
}

var endpoints = []Endpoint{
	{Method: http.MethodGet, Path: "/"},
	{Method: http.MethodGet, Path: "/map"},
	{Method: http.MethodGet, Path: "/tool/ls"},
	{Method: http.MethodGet, Path: "/tool/read", queryParams: set("recursive")},
	{Method: http.MethodGet, Path: "/tool/tree"},
	{Method: http.MethodPost, Path: "/tool/write", queryParams: set("path", "mode")},
}

// Endpoints returns a copy of the registered endpoint list.
func Endpoints() []Endpoint {
	return append([]Endpoint(nil), endpoints...)
}

// Lookup finds endpoint metadata by URL path.
func Lookup(path string) (Endpoint, bool) {
	for _, endpoint := range endpoints {
		if endpoint.Path == path {
			return endpoint, true
		}
	}
	return Endpoint{}, false
}

func set(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
