package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillip-england/mymcp/internal/protocol"
)

func TestSkillCommandWritesEmbeddedSkillFile(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "SKILL.md")
	var stdout bytes.Buffer

	if err := run([]string{"skill", destination}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("run skill command: %v", err)
	}

	written, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read written skill file: %v", err)
	}
	want, err := os.ReadFile("SKILL.md")
	if err != nil {
		t.Fatalf("read source skill file: %v", err)
	}
	if string(written) != string(want) {
		t.Fatalf("written skill file does not match embedded SKILL.md")
	}
	if got, want := stdout.String(), "wrote "+destination+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestSkillCommandRequiresPath(t *testing.T) {
	err := run([]string{"skill"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run skill command succeeded without path")
	}
	if !strings.Contains(err.Error(), "usage: mymcp skill <path>") {
		t.Fatalf("error = %q, want usage", err.Error())
	}
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "health", path: "/", wantStatus: http.StatusOK, wantBody: "mymcp is running\n"},
		{name: "map", path: "/map", wantStatus: http.StatusOK, wantBody: "mymcp server endpoint map"},
		{name: "unknown", path: "/unknown", wantStatus: http.StatusNotFound, wantBody: "404 page not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()

			newRouter().ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if !strings.Contains(response.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want it to contain %q", response.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestMapDocumentsEveryEndpointWithJSONExamples(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/map", nil)
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/plain; charset=utf-8", contentType)
	}

	body := response.Body.String()
	for _, want := range []string{
		"Send exactly one JSON object per request.",
		"=== SERVER ===",
		"=== CONTROL ===",
		"=== FILESYSTEM ===",
		"Valid request JSON:",
		"X-Mymcp-Terminal: error",
		"X-Mymcp-Terminal: success",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("map body does not contain %q", want)
		}
	}
	for _, endpoint := range protocol.Endpoints() {
		for _, want := range []string{
			endpoint.Method + " " + endpoint.Path + "\n",
			endpoint.Purpose,
			endpoint.Example,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("map does not document %s: missing %q", endpoint.Name, want)
			}
		}
	}
}

func TestRoutesRejectOtherMethods(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestRoutesRejectUnsupportedAndRepeatedQueryParameters(t *testing.T) {
	for _, path := range []string{
		"/tool/ls?recursive=true",
		"/tool/read?recursive=true&recursive=false",
		"/tool/read?recursive=true;depth=2",
	} {
		request := httptest.NewRequest(http.MethodGet, path, strings.NewReader("."))
		response := httptest.NewRecorder()
		newRouter().ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("%s returned status %d, want 400", path, response.Code)
		}
	}
}

func TestTreeRoute(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "docs")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "guide.txt"), []byte("guide"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/tool/tree", strings.NewReader(root))
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", response.Code, http.StatusOK, response.Body.String())
	}
	want := strings.Join([]string{
		"TYPE\tSIZE_BYTES\tPATH",
		"directory\t11\t" + root,
		"file\t6\t" + filepath.Join(root, "README.md"),
		"directory\t5\t" + filepath.Join(root, "docs"),
		"file\t5\t" + filepath.Join(root, "docs", "guide.txt"),
		"",
	}, "\n")
	if response.Body.String() != want {
		t.Fatalf("body = %q, want %q", response.Body.String(), want)
	}
}

func TestLSRouteListsDirectChildrenWithSizes(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "docs")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("readme"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "guide.txt"), []byte("guide"), 0o644); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/tool/ls", strings.NewReader(root))
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)

	want := "TYPE\tSIZE_BYTES\tNAME\nfile\t6\tREADME.md\ndirectory\t5\tdocs\n"
	if response.Code != http.StatusOK || response.Body.String() != want {
		t.Fatalf("status = %d, body = %q; want status 200, body %q", response.Code, response.Body.String(), want)
	}
}

func TestLSRouteRejectsFilePath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("contents"), 0o644); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/tool/ls", strings.NewReader(path))
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %q", response.Code, response.Body.String())
	}
}

func TestTreeRouteRejectsInvalidPaths(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "empty", body: "", wantStatus: http.StatusBadRequest},
		{name: "missing", body: filepath.Join(t.TempDir(), "missing"), wantStatus: http.StatusNotFound},
		{name: "file", body: file.Name(), wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/tool/tree", strings.NewReader(tt.body))
			response := httptest.NewRecorder()
			newRouter().ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}
