# mymcp

`mymcp` is a small HTTP tool server and Go SDK for LLM harnesses. It exposes a
deliberately narrow set of loop-control and filesystem operations. Responses are
plain text so models can consume them directly and humans can inspect them while
debugging.

The `/map` endpoint is the machine-oriented tool manual. It contains precise
request shapes, validated examples, rejection rules, and response meanings for
an LLM to digest. This README is the human-facing guide for operating and
extending it.

## Design

The project keeps three views of the protocol aligned:

1. `internal/protocol` is the single endpoint catalog used for SDK validation.
2. `server.go` attaches one HTTP handler to every catalog entry and refuses to
   start when an entry is missing or unrecognized.
3. `/map` generates model instructions from the same catalog, so route examples
   and endpoint descriptions cannot silently drift from SDK routing.

The tool surface is intentionally small. Loop control needs `success` and
`error`; filesystem work needs focused listing, recursive inspection, reading,
and writing. Directory mutation and deletion are omitted because they are not
required for the core workflow and are especially risky without a sandbox.

## Filesystem access

Filesystem endpoints accept paths directly. Relative paths resolve from the
server process's working directory, and absolute paths are used as supplied.
There is no application-level sandbox, so run the server with the operating-system
permissions and isolation appropriate for its users.

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

Tools are grouped into control and filesystem categories. Paths remain directly
under `/tool` because six routes do not justify deeper URL nesting.

### Finish the agentic loop

`POST /tool/success` and `POST /tool/error` take a required final message as
plain text. Both return `200 OK` with that message and an `X-Mymcp-Terminal`
header set to `success` or `error`. A harness must stop the loop when that header
is present and surface the message to the user.

```bash
curl -X POST --data-binary 'The goal is complete.' \
  http://localhost:8765/tool/success

curl -X POST --data-binary 'Cannot complete the goal: credentials are missing.' \
  http://localhost:8765/tool/error
```

Messages must contain non-whitespace text and may be at most 65536 bytes.

## Filesystem Tools

### List one directory

`GET /tool/ls` takes a directory path as its plain-text body and lists only its
direct children. The tab-separated result includes each child's type, exact byte
size, and name. Directory sizes are the recursive sum of regular-file content.

```bash
curl -X GET \
  --data '/workspace/my-project/docs' \
  http://localhost:8765/tool/ls
```

### Inspect a directory tree

`GET /tool/tree` takes a directory path as its plain-text body and lists that
directory and every descendant path. Each row includes the entry type, exact byte
size, and full path. Directory sizes are recursive regular-file-content totals.

```bash
curl -X GET \
  --data '/workspace/my-project/docs' \
  http://localhost:8765/tool/tree
```

### Read files

`GET /tool/read` takes a file or directory path as its plain-text body. A file is
returned directly. A directory returns its direct child files by default; add
`recursive=true` to include files in all descendant directories. Each file is
surrounded by clear `FILE` and `END FILE` markers.

```bash
curl -X GET \
  --data '/workspace/my-project/README.md' \
  http://localhost:8765/tool/read

curl -X GET \
  --data '/workspace/my-project/docs' \
  'http://localhost:8765/tool/read?recursive=true'
```

### Write files

`POST /tool/write` takes a URL-encoded `path`, an explicit `mode`, and the exact
file content in the request body. `overwrite` replaces the full file; `append`
adds content to its end. Missing files are created with `0644` permissions, but
their parent directory must already exist.

```bash
curl -X POST \
  --data-binary 'replacement content' \
  'http://localhost:8765/tool/write?path=%2Fworkspace%2Fmy-project%2Fnotes.txt&mode=overwrite'

curl -X POST \
  --data-binary 'additional content' \
  'http://localhost:8765/tool/write?path=%2Fworkspace%2Fmy-project%2Fnotes.txt&mode=append'
```

To edit or delete part of a file, read it first, modify the returned content in
memory, and send the complete revised content back with `mode=overwrite`. An empty
overwrite body empties the file.

## Go SDK and LLM harnesses

An LLM does not call this server directly. It emits a JSON object describing the
HTTP request it wants to make, and a host application's harness passes that JSON
to the SDK. The SDK performs three jobs:

1. Decode the model's JSON using the request shape documented by `/map`.
2. Confirm the method, path, and query map to a known `mymcp` endpoint.
3. Make the HTTP request and return its status, headers, body, and terminal outcome.

Install the SDK from this module:

```bash
go get github.com/phillip-england/mymcp/sdk
```

Create one client for a server, then pass the model's JSON string to `Route`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/phillip-england/mymcp/sdk"
)

func main() {
	client, err := sdk.NewClient("http://localhost:8765")
	if err != nil {
		log.Fatal(err)
	}

	modelJSON := `{
		"method": "GET",
		"path": "/tool/ls",
		"body": "/workspace/my-project/docs"
	}`
	response, err := client.Route(context.Background(), modelJSON)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("status: %d\n%s", response.StatusCode, response.Text())
	if response.Terminal {
		fmt.Printf("loop finished with %s\n", response.Outcome)
	}
}
```

`RouteJSON` remains available when the harness already has a byte slice. The
accepted JSON fields are `method`, `path`, optional string-to-string
`headers`, and optional string `body`. Unknown fields, malformed JSON, absolute
URLs, unknown endpoints, incorrect methods, and unsupported query parameters are
rejected without making a network request. Server responses, including `4xx` and
`5xx` responses, are returned normally so the harness can give the model the
server's actionable error text. Transport and JSON-routing failures are returned
as Go errors.

For `/tool/success` and `/tool/error`, `Response.Terminal` is true and
`Response.Outcome` is `sdk.OutcomeSuccess` or `sdk.OutcomeError`. Stop the
agentic loop and use `Response.Text()` as its final user-facing message.

## Request logging

The server logs each request in a compact, human-readable format:

```text
[METHOD][PATH][STATUS][DURATION]
```

Query parameters and file contents are not logged.

## Development

Format and verify the project with:

```bash
make fmt
make check
```

The tests execute every model-facing catalog example through the SDK validator.
This catches stale documentation as part of the normal test suite.

## Adding A Tool

Keep additions narrow and follow the existing dependency direction:

1. Add one endpoint entry in `internal/protocol/endpoints.go`, including its
   category, request contract, valid JSON example, and response meaning.
2. Implement a small HTTP handler in a tool-specific file. Move reusable domain
   logic into a separate helper rather than growing the handler.
3. Attach the handler by endpoint name in `server.go`.
4. Add focused handler tests. `TestCatalogExamplesAreRoutable` and
   `TestMapDocumentsEveryEndpointWithJSONExamples` cover the shared wiring.

Do not add a tool when an existing endpoint can express the operation clearly.
Do not put tool-specific routing rules in the SDK; the shared catalog owns them.

## Project layout

- `server.go` connects shared endpoint definitions to HTTP handlers.
- `internal/protocol` is the shared server/SDK endpoint contract.
- `tool_http.go` contains request parsing and response helpers shared by tools.
- `filesystem.go` contains reusable filesystem inspection operations.
- `terminal.go` implements terminal success and error control tools.
- `ls.go`, `tree.go`, `read.go`, and `write.go` are thin filesystem tool handlers.
- `map.go` contains the model-facing endpoint guide returned by `/map`.
- `middleware.go` contains request logging.
- `sdk/client.go` handles HTTP transport, while `sdk/request.go` validates and
  routes model-generated JSON.
