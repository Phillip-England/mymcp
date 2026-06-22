package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
		"GET /\n",
		"GET /map\n",
		"GET /tool/ls\n",
		"GET /tool/read\n",
		"GET /tool/tree\n",
		"POST /tool/write\n",
		"Valid request:",
		"Invalid request",
		"read-modify-overwrite",
		"FILESYSTEM SANDBOX RULES:",
		`"headers": {"X-Mymcp-Sandbox": "/workspace"}`,
		`"method": "GET"`,
		`"body": "/workspace/README.md"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("map body does not contain %q", want)
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
	request.Header.Set(sandboxHeader, root)
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %q", response.Code, http.StatusOK, response.Body.String())
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"TYPE\tSIZE_BYTES\tPATH",
		"directory\t11\t" + resolvedRoot,
		"file\t6\t" + filepath.Join(resolvedRoot, "README.md"),
		"directory\t5\t" + filepath.Join(resolvedRoot, "docs"),
		"file\t5\t" + filepath.Join(resolvedRoot, "docs", "guide.txt"),
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
	request.Header.Set(sandboxHeader, root)
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
	request.Header.Set(sandboxHeader, root)
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
			request.Header.Set(sandboxHeader, existingTestDirectory(tt.body))
			response := httptest.NewRecorder()
			newRouter().ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}
