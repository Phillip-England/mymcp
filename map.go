package main

import (
	"fmt"
	"strings"

	"github.com/phillip-england/mymcp/internal/protocol"
)

const guideIntroduction = `mymcp server endpoint map

Send exactly one JSON object per request. The accepted fields are:
  "method"   required string; the HTTP method shown below
  "path"     required string; a server-relative path with optional query
  "body"     optional string; the exact plain-text HTTP request body
  "headers"  optional object containing string-to-string HTTP headers

Unknown fields, absolute URLs, unknown endpoints, incorrect methods, and
unsupported query parameters are rejected before an HTTP request is sent.

Filesystem paths are used directly. Relative paths start at the server process's
working directory. The server has no application-level filesystem sandbox.

Control endpoints are terminal. After a successful /tool/error or /tool/success
response, stop the agentic loop and surface the response body to the user.
`

func modelGuide() string {
	var output strings.Builder
	output.WriteString(guideIntroduction)

	category := ""
	for _, endpoint := range protocol.Endpoints() {
		if endpoint.Category != category {
			category = endpoint.Category
			fmt.Fprintf(&output, "\n=== %s ===\n", strings.ToUpper(category))
		}
		fmt.Fprintf(
			&output,
			"\n%s %s\nPurpose: %s\nRequest: %s\nValid request JSON:\n%s\nResponse: %s\n",
			endpoint.Method,
			endpoint.Path,
			endpoint.Purpose,
			endpoint.Request,
			endpoint.Example,
			endpoint.Response,
		)
	}
	return output.String()
}
