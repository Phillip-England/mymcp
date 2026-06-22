package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSandboxAllowsRelativePaths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(path, []byte("sandboxed"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/tool/read", strings.NewReader("notes.txt"))
	request.Header.Set(sandboxHeader, root)
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "sandboxed") {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}
}

func TestSandboxRejectsPathsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		method  string
		target  string
		body    string
		request string
	}{
		{name: "absolute read", method: http.MethodGet, request: "/tool/read", body: outside},
		{name: "relative traversal", method: http.MethodGet, request: "/tool/read", body: filepath.Join("..", filepath.Base(filepath.Dir(outside)), filepath.Base(outside))},
		{name: "ls", method: http.MethodGet, request: "/tool/ls", body: filepath.Dir(outside)},
		{name: "tree", method: http.MethodGet, request: "/tool/tree", body: filepath.Dir(outside)},
		{name: "write", method: http.MethodPost, request: "/tool/write", target: outside, body: "changed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestPath := tt.request
			if tt.target != "" {
				query := url.Values{"path": {tt.target}, "mode": {writeModeOverwrite}}
				requestPath += "?" + query.Encode()
			}
			request := httptest.NewRequest(tt.method, requestPath, strings.NewReader(tt.body))
			request.Header.Set(sandboxHeader, root)
			response := httptest.NewRecorder()
			newRouter().ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; body = %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestSandboxRejectsSymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	outsideRoot := t.TempDir()
	outsideFile := filepath.Join(outsideRoot, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("unchanged"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outsideRoot, link); err != nil {
		t.Fatal(err)
	}

	readRequest := httptest.NewRequest(http.MethodGet, "/tool/read", strings.NewReader(filepath.Join(link, "outside.txt")))
	readRequest.Header.Set(sandboxHeader, root)
	readResponse := httptest.NewRecorder()
	newRouter().ServeHTTP(readResponse, readRequest)
	if readResponse.Code != http.StatusForbidden {
		t.Fatalf("read status = %d, want 403; body = %q", readResponse.Code, readResponse.Body.String())
	}

	query := url.Values{"path": {filepath.Join(link, "new.txt")}, "mode": {writeModeOverwrite}}
	writeRequest := httptest.NewRequest(http.MethodPost, "/tool/write?"+query.Encode(), strings.NewReader("escape"))
	writeRequest.Header.Set(sandboxHeader, root)
	writeResponse := httptest.NewRecorder()
	newRouter().ServeHTTP(writeResponse, writeRequest)
	if writeResponse.Code != http.StatusForbidden {
		t.Fatalf("write status = %d, want 403; body = %q", writeResponse.Code, writeResponse.Body.String())
	}
	if _, err := os.Stat(filepath.Join(outsideRoot, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("write escaped sandbox; stat error = %v", err)
	}
}

func TestFilesystemToolsRequireSandboxHeader(t *testing.T) {
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/tool/read", body: os.TempDir()},
		{method: http.MethodGet, path: "/tool/ls", body: os.TempDir()},
		{method: http.MethodGet, path: "/tool/tree", body: os.TempDir()},
		{method: http.MethodPost, path: "/tool/write?path=file.txt&mode=overwrite", body: "contents"},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		response := httptest.NewRecorder()
		newRouter().ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Errorf("%s %s status = %d, want 403", tt.method, tt.path, response.Code)
		}
	}
}
