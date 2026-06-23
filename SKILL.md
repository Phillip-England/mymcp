---
name: use-mymcp
description: Interact with mymcp control and filesystem tools through an LLM harness. Use when an agent must complete or stop a loop, inspect files, or write and edit files through mymcp request JSON.
---

# Use mymcp

Route tool work through the harness-provided mymcp bridge. Produce strict
HTTP-request JSON for the bridge; do not assume the model can make HTTP requests
itself. The harness owns the server address.

## Start with the map

Request `/map` when endpoint behavior is unclear or may have changed:

```json
{"method":"GET","path":"/map"}
```

Treat that response as the authoritative protocol reference. The server uses
`http://localhost:8765` by default, but the harness may configure another address.

## Build requests

Send exactly one JSON object with these fields:

- `method`: HTTP method.
- `path`: Server-relative endpoint and optional query string. Never use an absolute URL.
- `body`: Optional plain-text request body encoded as a JSON string.
- `headers`: Optional string-to-string headers. Usually omit this field.

Do not add unknown fields. The harness-configured SDK validates the request before
sending it. Filesystem paths are interpreted directly by the server: relative
paths start from its working directory, and absolute paths are used as supplied.

## Choose an operation

### Finish successfully

When the user's goal is complete, call `/tool/success` exactly once with the
final user-facing message. This terminates the agentic loop:

```json
{"method":"POST","path":"/tool/success","body":"The goal is complete."}
```

### Stop with an error

When the goal cannot be completed, call `/tool/error` exactly once with a
specific, actionable reason. Use this only for a terminal blocker, not for a
recoverable tool failure:

```json
{"method":"POST","path":"/tool/error","body":"Cannot complete the goal because the required credentials are unavailable."}
```

Do not issue another tool request after either terminal tool succeeds.

### List direct children

Use `/tool/ls` to inspect only one directory level:

```json
{"method":"GET","path":"/tool/ls","body":"src"}
```

The response contains `TYPE`, `SIZE_BYTES`, and `NAME` columns.

### Inspect a full tree

Use `/tool/tree` for a directory and all descendants:

```json
{"method":"GET","path":"/tool/tree","body":"."}
```

The response contains `TYPE`, `SIZE_BYTES`, and `PATH` columns. Prefer `/tool/ls`
when recursive output is unnecessary.

### Read files

Read one file:

```json
{"method":"GET","path":"/tool/read","body":"README.md"}
```

Read direct file children of a directory with the same endpoint. Add recursion
only when all descendant files are needed:

```json
{"method":"GET","path":"/tool/read?recursive=true","body":"docs"}
```

Each returned text block is delimited with `FILE` and `END FILE` markers. Keep
content associated with the path shown in those markers.

### Write files

URL-encode the destination in the `path` query. Use `overwrite` to replace the
entire file or `append` to add exact content at its end:

```json
{"method":"POST","path":"/tool/write?path=notes.txt&mode=overwrite","body":"complete replacement content\n"}
```

```json
{"method":"POST","path":"/tool/write?path=logs%2Fagent.txt&mode=append","body":"new line\n"}
```

Parent directories must already exist. An empty overwrite body empties the file.

## Edit safely

To change or delete part of a file:

1. Read the current file with `/tool/read`.
2. Modify the complete content in memory.
3. Send the complete revised content with `mode=overwrite`.
4. Read the file again when verification is important.

Do not use append for general editing. Preserve unrelated content, formatting,
and final-newline behavior unless the task requires changing them.

## Handle responses

Check the returned HTTP status before trusting the body:

- `2xx`: Use the response as the operation result.
- `400`: Correct malformed input, query parameters, or target type.
- `404`: Recheck the path or required parent directory.
- `500`: Report the server-side access failure with its response text.

The control tools return `X-Mymcp-Terminal: success` or
`X-Mymcp-Terminal: error`. The harness must stop the loop and surface the body
when this header is present.

Use clear paths and avoid touching files unrelated to the task. If the harness
exposes no mymcp bridge, state that the integration is unavailable rather than
claiming an operation ran.
