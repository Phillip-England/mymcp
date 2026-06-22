# mymcp

`mymcp` is a small HTTP server and Go SDK that give an LLM harness a controlled
set of filesystem tools. It can list directories, inspect a directory tree, read
files, and create or update files. The server returns plain text so tool output
stays easy for a model to parse and for a human to inspect while debugging.

The `/map` endpoint is the machine-oriented tool manual. It contains precise
request shapes, valid and invalid examples, and response meanings for an LLM to
digest. This README explains the project for people operating or extending it.

## Filesystem sandbox

Every request to `/tool/ls`, `/tool/read`, `/tool/tree`, or `/tool/write` must include an
`X-Mymcp-Sandbox` header. Its value is the directory that request may access:

```text
X-Mymcp-Sandbox: /workspace/my-project
```

The restriction is request-scoped, so a harness can assign a different workspace
to each model, session, or request. Relative target paths are resolved from the
sandbox directory. Absolute target paths work only when they resolve to the
sandbox directory or one of its descendants.

The server canonicalizes the sandbox and target paths before access. It rejects
`..` traversal, absolute paths outside the sandbox, and symlinks that resolve
outside it with `403 Forbidden`. The header itself must name an existing
directory. `/` and `/map` do not access files, so they do not require the header;
a harness may still attach it consistently to every request.

This check confines normal tool requests, but it is not a replacement for an OS
security boundary. A hostile local process could race path validation by changing
symlinks, and a compromised server process is not constrained by an HTTP header.
Run the server in a container, VM, or restricted operating-system account when
the threat model includes untrusted local processes or arbitrary code execution.

## Running the server

The project requires Go 1.26 or a compatible newer release.

```bash
go run .
```

The server uses its dedicated port `8765` by default. Set `PORT` to override it:

```bash
PORT=9000 go run .
```

Check that it is running and retrieve the model-oriented endpoint map:

```bash
curl http://localhost:8765/
curl http://localhost:8765/map
```

## Tools

### List one directory

`GET /tool/ls` takes a directory path as its plain-text body and lists only its
direct children. The tab-separated result includes each child's type, exact byte
size, and name. Directory sizes are the recursive sum of regular-file content.

```bash
curl -X GET \
  -H 'X-Mymcp-Sandbox: /workspace/my-project' \
  --data 'docs' \
  http://localhost:8765/tool/ls
```

### Inspect a directory tree

`GET /tool/tree` takes a directory path as its plain-text body and lists that
directory and every descendant path. Each row includes the entry type, exact byte
size, and full path. Directory sizes are recursive regular-file-content totals.

```bash
curl -X GET \
  -H 'X-Mymcp-Sandbox: /workspace/my-project' \
  --data 'docs' \
  http://localhost:8765/tool/tree
```

### Read files

`GET /tool/read` takes a file or directory path as its plain-text body. A file is
returned directly. A directory returns its direct child files by default; add
`recursive=true` to include files in all descendant directories. Each file is
surrounded by clear `FILE` and `END FILE` markers.

```bash
curl -X GET \
  -H 'X-Mymcp-Sandbox: /workspace/my-project' \
  --data 'README.md' \
  http://localhost:8765/tool/read

curl -X GET \
  -H 'X-Mymcp-Sandbox: /workspace/my-project' \
  --data 'docs' \
  'http://localhost:8765/tool/read?recursive=true'
```

### Write files

`POST /tool/write` takes a URL-encoded `path`, an explicit `mode`, and the exact
file content in the request body. `overwrite` replaces the full file; `append`
adds content to its end. Missing files are created with `0644` permissions, but
their parent directory must already exist.

```bash
curl -X POST \
  -H 'X-Mymcp-Sandbox: /workspace/my-project' \
  --data-binary 'replacement content' \
  'http://localhost:8765/tool/write?path=notes.txt&mode=overwrite'

curl -X POST \
  -H 'X-Mymcp-Sandbox: /workspace/my-project' \
  --data-binary 'additional content' \
  'http://localhost:8765/tool/write?path=notes.txt&mode=append'
```

To edit or delete part of a file, read it first, modify the returned content in
memory, and send the complete revised content back with `mode=overwrite`. An empty
overwrite body empties the file.

## Go SDK and LLM harnesses

An LLM does not call this server directly. It emits a JSON object describing the
HTTP request it wants to make, and a host application's harness passes that JSON
to the SDK. The SDK performs four jobs:

1. Decode the model's JSON using the request shape documented by `/map`.
2. Confirm the method, path, and query map to a known `mymcp` endpoint.
3. Inject the harness-configured sandbox, overriding any sandbox chosen by the model.
4. Make the HTTP request and return its status, headers, and body.

Install the SDK from this module:

```bash
go get github.com/phillip-england/mymcp/sdk
```

Create one client for a server and trusted workspace, then pass model output to
`RouteJSON`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/phillip-england/mymcp/sdk"
)

func main() {
	client, err := sdk.NewClient("http://localhost:8765", "/workspace/my-project")
	if err != nil {
		log.Fatal(err)
	}

	modelJSON := []byte(`{
        "method": "GET",
        "path": "/tool/ls",
        "body": "docs"
    }`)
	response, err := client.RouteJSON(context.Background(), modelJSON)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("status: %d\n%s", response.StatusCode, response.Text())
}
```

The accepted JSON fields are `method`, `path`, optional string-to-string
`headers`, and optional string `body`. Unknown fields, malformed JSON, absolute
URLs, unknown endpoints, incorrect methods, and unsupported query parameters are
rejected without making a network request. Server responses, including `4xx` and
`5xx` responses, are returned normally so the harness can give the model the
server's actionable error text. Transport and JSON-routing failures are returned
as Go errors.

The SDK sandbox is intentionally configured outside model JSON. This prevents a
model from expanding its own filesystem access by generating a different
`X-Mymcp-Sandbox` header.

## Request logging

The server logs each request's method, URL, response status, response size,
duration, and remote address. File contents and the sandbox header are not logged.

## Development

Format and verify the project with:

```bash
gofmt -w *.go
go test ./...
go vet ./...
```

## Project layout

- `server.go` connects shared endpoint definitions to HTTP handlers.
- `internal/protocol` is the shared server/SDK endpoint contract.
- `tool_http.go` contains request parsing and response helpers shared by tools.
- `sandbox.go` validates and confines filesystem paths.
- `filesystem.go` contains reusable filesystem inspection operations.
- `ls.go`, `tree.go`, `read.go`, and `write.go` are thin tool handlers.
- `map.go` contains the model-facing endpoint guide returned by `/map`.
- `middleware.go` contains request logging.
- `sdk/client.go` handles HTTP transport, while `sdk/request.go` validates and
  routes model-generated JSON.
