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

func TestWriteRouteOverwritesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("remove this text"), 0o600); err != nil {
		t.Fatal(err)
	}

	response := performWriteRequest(t, path, writeModeOverwrite, "replacement text")

	assertWrittenFile(t, response, path, "replacement text")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %o, want existing permissions 600", info.Mode().Perm())
	}
}

func TestWriteRouteAppendsToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(path, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	response := performWriteRequest(t, path, writeModeAppend, "second\n")

	assertWrittenFile(t, response, path, "first\nsecond\n")
}

func TestWriteRouteCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.txt")

	response := performWriteRequest(t, path, writeModeOverwrite, "new contents")

	assertWrittenFile(t, response, path, "new contents")
}

func TestWriteRouteCanEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(path, []byte("delete everything"), 0o644); err != nil {
		t.Fatal(err)
	}

	response := performWriteRequest(t, path, writeModeOverwrite, "")

	assertWrittenFile(t, response, path, "")
}

func TestWriteRouteRejectsInvalidInput(t *testing.T) {
	directory := t.TempDir()
	tests := []struct {
		name       string
		path       string
		mode       string
		wantStatus int
	}{
		{name: "missing path", mode: writeModeOverwrite, wantStatus: http.StatusBadRequest},
		{name: "missing mode", path: filepath.Join(directory, "file.txt"), wantStatus: http.StatusBadRequest},
		{name: "invalid mode", path: filepath.Join(directory, "file.txt"), mode: "replace", wantStatus: http.StatusBadRequest},
		{name: "directory path", path: directory, mode: writeModeOverwrite, wantStatus: http.StatusBadRequest},
		{name: "missing parent", path: filepath.Join(directory, "missing", "file.txt"), mode: writeModeOverwrite, wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performWriteRequest(t, tt.path, tt.mode, "contents")
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func assertWrittenFile(t *testing.T, response *httptest.ResponseRecorder, path, want string) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", response.Code, response.Body.String())
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != want {
		t.Fatalf("contents = %q, want %q", contents, want)
	}
}

func performWriteRequest(t *testing.T, path, mode, contents string) *httptest.ResponseRecorder {
	t.Helper()
	query := make(url.Values)
	if path != "" {
		query.Set("path", path)
	}
	if mode != "" {
		query.Set("mode", mode)
	}
	request := httptest.NewRequest(http.MethodPost, "/tool/write?"+query.Encode(), strings.NewReader(contents))
	request.Header.Set(sandboxHeader, existingTestDirectory(path))
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)
	return response
}
