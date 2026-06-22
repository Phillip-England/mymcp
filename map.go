package main

const endpointGuide = `mymcp server endpoint map

This document describes every endpoint exposed by the server. Request examples are
JSON representations of HTTP requests. The "body" value is the exact plain-text
request body to send; the tool endpoints do not accept a JSON-encoded body.

HARNESS AND SDK:
  An LLM produces one JSON request object matching the examples in this map. A
  harness may pass those bytes to the Go SDK Client.RouteJSON method. The SDK
  validates the method, path, and query against known endpoints, makes the HTTP
  request, and returns the response. The harness configures the SDK sandbox; the
  SDK overrides any model-provided X-Mymcp-Sandbox value with that trusted value.

FILESYSTEM SANDBOX RULES:
  Every /tool/* request MUST include this header:
    X-Mymcp-Sandbox: <existing directory path>

  The header limits that request to the named directory and its descendants.
  Relative target paths resolve from the sandbox directory. Absolute target paths
  are accepted only when they resolve inside the sandbox. Path traversal and
  symlinks that escape the sandbox return 403 Forbidden. GET / and GET /map do not
  access the filesystem and do not require the header.

===============================================================================
GET /
===============================================================================

Purpose:
  Check whether the server is running.

Request:
  Method: GET
  Path:   /
  Body:   none

Valid request:
{
  "method": "GET",
  "path": "/"
}

Invalid request (POST is not allowed):
{
  "method": "POST",
  "path": "/"
}

Success response:
  Status: 200 OK
  Body:   mymcp is running

===============================================================================
GET /map
===============================================================================

Purpose:
  Return this complete server and endpoint guide as plain text.

Request:
  Method: GET
  Path:   /map
  Body:   none

Valid request:
{
  "method": "GET",
  "path": "/map"
}

Invalid request (the endpoint only supports GET):
{
  "method": "PUT",
  "path": "/map"
}

Success response:
  Status:       200 OK
  Content-Type: text/plain; charset=utf-8

===============================================================================
GET /tool/ls
===============================================================================

Purpose:
  List only the direct children of one directory. Results are sorted by name and
  contain entry type, size in bytes, and name. A directory's size is the recursive
  sum of regular-file content beneath it. This endpoint does not list descendants.

Request:
  Method: GET
  Path:   /tool/ls
  Header: X-Mymcp-Sandbox: <existing directory path> (required)
  Body:   a plain-text directory path, at most 4096 bytes

Valid request:
{
  "method": "GET",
  "path": "/tool/ls",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/project"
}

Invalid request (a file path is not a directory):
{
  "method": "GET",
  "path": "/tool/ls",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/project/README.md"
}

Success response format:
  TYPE<TAB>SIZE_BYTES<TAB>NAME
  file<TAB>120<TAB>README.md
  directory<TAB>4096<TAB>docs

Responses:
  200 OK                  Direct children as tab-separated plain text.
  400 Bad Request         Invalid sandbox, empty/oversized body, or a path that is
                          not a directory.
  403 Forbidden           Missing sandbox header or target outside the sandbox.
  404 Not Found           The supplied directory does not exist.
  500 Internal Server Error
                          The directory or its children could not be accessed.

===============================================================================
GET /tool/read
===============================================================================

Purpose:
  Read one file, the regular files directly inside a directory, or every regular
  file below a directory. Each returned file is wrapped in FILE and END FILE
  markers. Directory results are ordered by filesystem path.

Request:
  Method: GET
  Path:   /tool/read
  Header: X-Mymcp-Sandbox: <existing directory path> (required)
  Query:  recursive=true or recursive=false (optional; defaults to false)
  Body:   a plain-text file or directory path, at most 4096 bytes

Valid request (read one file):
{
  "method": "GET",
  "path": "/tool/read",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/README.md"
}

Valid request (read all regular files below a directory):
{
  "method": "GET",
  "path": "/tool/read?recursive=true",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/docs"
}

Invalid request (the required path body is empty):
{
  "method": "GET",
  "path": "/tool/read",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": ""
}

Invalid request (recursive must be true or false):
{
  "method": "GET",
  "path": "/tool/read?recursive=sometimes",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/docs"
}

Invalid request (the body is a JSON object instead of a plain-text path):
{
  "method": "GET",
  "path": "/tool/read",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "{\"path\":\"/workspace/README.md\"}"
}

Responses:
  200 OK                  File contents as plain text.
  400 Bad Request         Invalid sandbox, empty/oversized body, invalid recursive
                          value, or an unsupported file type.
  403 Forbidden           Missing sandbox header or target outside the sandbox.
  404 Not Found           The supplied path does not exist.
  500 Internal Server Error
                          The path or its contents could not be accessed.

===============================================================================
GET /tool/tree
===============================================================================

Purpose:
  List the supplied directory and every descendant path. Each row contains entry
  type, size in bytes, and full path. The root directory is the first row. A
  directory's size is the recursive sum of regular-file content beneath it.

Request:
  Method: GET
  Path:   /tool/tree
  Header: X-Mymcp-Sandbox: <existing directory path> (required)
  Body:   a plain-text directory path, at most 4096 bytes

Valid request:
{
  "method": "GET",
  "path": "/tool/tree",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/project"
}

Invalid request (a file path is not a directory):
{
  "method": "GET",
  "path": "/tool/tree",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/project/README.md"
}

Invalid request (the required directory path body is empty):
{
  "method": "GET",
  "path": "/tool/tree",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": ""
}

Invalid request (POST is not allowed):
{
  "method": "POST",
  "path": "/tool/tree",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "/workspace/project"
}

Responses:
  200 OK                  Recursive entries as tab-separated plain text with the
                          header TYPE, SIZE_BYTES, PATH.
  400 Bad Request         Invalid sandbox, empty/oversized body, or a path that is
                          not a directory.
  403 Forbidden           Missing sandbox header or target outside the sandbox.
  404 Not Found           The supplied directory does not exist.
  500 Internal Server Error
                          The directory or its descendants could not be accessed.

===============================================================================
POST /tool/write
===============================================================================

Purpose:
  Create or write one file. Overwrite mode replaces the entire file. Append mode
  adds content at the end of the file. Existing file permissions are preserved;
  newly created files use 0644 permissions. Parent directories must already exist.

  To delete or edit only part of a file, first use GET /tool/read to read its
  current contents. Modify that text in memory, removing or changing the desired
  portions, then send the complete revised contents here with mode=overwrite.
  This read-modify-overwrite workflow provides full-file editing control.

Request:
  Method: POST
  Path:   /tool/write
  Header: X-Mymcp-Sandbox: <existing directory path> (required)
  Query:  path=<file path> (required, URL-encoded)
          mode=overwrite or mode=append (required)
  Body:   the exact content to write; an empty body is valid

Valid request (replace the entire file):
{
  "method": "POST",
  "path": "/tool/write?path=%2Fworkspace%2Fnotes.txt&mode=overwrite",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "complete replacement contents\n"
}

Valid request (append to a file):
{
  "method": "POST",
  "path": "/tool/write?path=%2Fworkspace%2Fnotes.txt&mode=append",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "additional contents\n"
}

Invalid request (mode is required to avoid accidental truncation):
{
  "method": "POST",
  "path": "/tool/write?path=%2Fworkspace%2Fnotes.txt",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "contents"
}

Invalid request (the destination must be a file path):
{
  "method": "POST",
  "path": "/tool/write?path=%2Fworkspace&mode=overwrite",
  "headers": {"X-Mymcp-Sandbox": "/workspace"},
  "body": "contents"
}

Responses:
  200 OK                  The mode, written byte count, and destination path.
  400 Bad Request         Invalid sandbox, missing/invalid query values, or a
                          directory destination.
  403 Forbidden           Missing sandbox header or target outside the sandbox.
  404 Not Found           The destination's parent directory does not exist.
  500 Internal Server Error
                          The destination could not be accessed or written.
`
