// Package protocol defines the HTTP contract shared by the server and SDK.
package protocol

import (
	"fmt"
	"net/http"
	"net/url"
)

const (
	TerminalHeader  = "X-Mymcp-Terminal"
	TerminalError   = "error"
	TerminalSuccess = "success"
)

const (
	HealthName  = "health"
	MapName     = "map"
	ErrorName   = "error"
	SuccessName = "success"
	LSName      = "ls"
	ReadName    = "read"
	TreeName    = "tree"
	WriteName   = "write"
)

// Endpoint is one entry in the shared server, SDK, and model-facing catalog.
type Endpoint struct {
	Name       string
	Category   string
	Method     string
	Path       string
	Purpose    string
	Request    string
	Example    string
	Response   string
	queryNames map[string]struct{}
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
	_, ok := e.queryNames[name]
	return ok
}

// ValidateQuery rejects unknown or repeated query parameters.
func (e Endpoint) ValidateQuery(query url.Values) error {
	for name, values := range query {
		if !e.AllowsQuery(name) {
			return fmt.Errorf("unsupported query parameter %q for %s", name, e.Path)
		}
		if len(values) != 1 {
			return fmt.Errorf("query parameter %q must appear once for %s", name, e.Path)
		}
	}
	return nil
}

var endpoints = []Endpoint{
	{
		Name:     HealthName,
		Category: "Server",
		Method:   http.MethodGet,
		Path:     "/",
		Purpose:  "Check whether the server is running.",
		Request:  "No request body or query parameters.",
		Example:  `{"method":"GET","path":"/"}`,
		Response: "200 with a plain-text health message.",
	},
	{
		Name:     MapName,
		Category: "Server",
		Method:   http.MethodGet,
		Path:     "/map",
		Purpose:  "Return the complete model-facing server and tool guide.",
		Request:  "No request body or query parameters.",
		Example:  `{"method":"GET","path":"/map"}`,
		Response: "200 with this guide as plain text.",
	},
	{
		Name:     ErrorName,
		Category: "Control",
		Method:   http.MethodPost,
		Path:     "/tool/error",
		Purpose:  "Stop the agentic loop because the user's goal cannot be completed.",
		Request:  "A specific, user-facing terminal reason in the body; 1 to 65536 bytes.",
		Example:  `{"method":"POST","path":"/tool/error","body":"Cannot complete the goal because credentials are unavailable."}`,
		Response: "200 with the message and X-Mymcp-Terminal: error; 400 for an empty or oversized message.",
	},
	{
		Name:     SuccessName,
		Category: "Control",
		Method:   http.MethodPost,
		Path:     "/tool/success",
		Purpose:  "Stop the agentic loop because the user's goal is complete.",
		Request:  "The final user-facing success message in the body; 1 to 65536 bytes.",
		Example:  `{"method":"POST","path":"/tool/success","body":"The goal is complete."}`,
		Response: "200 with the message and X-Mymcp-Terminal: success; 400 for an empty or oversized message.",
	},
	{
		Name:       LSName,
		Category:   "Filesystem",
		Method:     http.MethodGet,
		Path:       "/tool/ls",
		Purpose:    "List the direct children of one directory with type and byte size.",
		Request:    "A file-system directory path in the body; at most 4096 bytes.",
		Example:    `{"method":"GET","path":"/tool/ls","body":"/workspace/project"}`,
		Response:   "200 with tab-separated entries; 400 for invalid input; 404 when missing; 500 when inaccessible.",
		queryNames: set(),
	},
	{
		Name:       ReadName,
		Category:   "Filesystem",
		Method:     http.MethodGet,
		Path:       "/tool/read",
		Purpose:    "Read a file or the regular files in a directory.",
		Request:    "A file or directory path in the body; optional recursive=true|false query; at most 4096 bytes.",
		Example:    `{"method":"GET","path":"/tool/read?recursive=true","body":"/workspace/docs"}`,
		Response:   "200 with delimited file contents; 400 for invalid input; 404 when missing; 500 when inaccessible.",
		queryNames: set("recursive"),
	},
	{
		Name:       TreeName,
		Category:   "Filesystem",
		Method:     http.MethodGet,
		Path:       "/tool/tree",
		Purpose:    "List a directory and all descendants with type and byte size.",
		Request:    "A file-system directory path in the body; at most 4096 bytes.",
		Example:    `{"method":"GET","path":"/tool/tree","body":"/workspace/project"}`,
		Response:   "200 with a tab-separated tree; 400 for invalid input; 404 when missing; 500 when inaccessible.",
		queryNames: set(),
	},
	{
		Name:       WriteName,
		Category:   "Filesystem",
		Method:     http.MethodPost,
		Path:       "/tool/write",
		Purpose:    "Create, overwrite, or append to one file.",
		Request:    "Required URL-encoded path and mode=overwrite|append query parameters; exact file content in the body.",
		Example:    `{"method":"POST","path":"/tool/write?path=notes.txt&mode=overwrite","body":"replacement content\n"}`,
		Response:   "200 with mode, byte count, and path; 400 for invalid input; 404 when the parent is missing; 500 on write failure.",
		queryNames: set("path", "mode"),
	},
}

// Endpoints returns the complete catalog in display order.
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
