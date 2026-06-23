package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRouteReadsSingleFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("first line\nsecond line"), 0o644); err != nil {
		t.Fatal(err)
	}

	response := performReadRequest(t, path, false)

	want := "===== FILE: " + path + " =====\n" +
		"first line\nsecond line\n" +
		"===== END FILE: " + path + " =====\n\n"
	if response.Code != http.StatusOK || response.Body.String() != want {
		t.Fatalf("status = %d, body = %q; want status 200, body %q", response.Code, response.Body.String(), want)
	}
}

func TestReadRouteReadsDirectChildrenByDefault(t *testing.T) {
	root := createReadFixture(t)

	response := performReadRequest(t, root, false)
	body := response.Body.String()

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", response.Code, body)
	}
	if !strings.Contains(body, filepath.Join(root, "first.txt")) || !strings.Contains(body, "first contents") {
		t.Fatalf("body does not contain direct child: %q", body)
	}
	if strings.Contains(body, "nested.txt") || strings.Contains(body, "nested contents") {
		t.Fatalf("body contains nested child without recursive flag: %q", body)
	}
}

func TestReadRouteReadsChildrenRecursively(t *testing.T) {
	root := createReadFixture(t)

	response := performReadRequest(t, root, true)
	body := response.Body.String()

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %q", response.Code, body)
	}
	for _, want := range []string{
		filepath.Join(root, "first.txt"),
		"first contents",
		filepath.Join(root, "subdirectory", "nested.txt"),
		"nested contents",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body does not contain %q: %q", want, body)
		}
	}
}

func TestReadRouteRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		query      string
		wantStatus int
	}{
		{name: "empty path", wantStatus: http.StatusBadRequest},
		{name: "missing path", path: filepath.Join(t.TempDir(), "missing"), wantStatus: http.StatusNotFound},
		{name: "invalid recursive flag", path: t.TempDir(), query: "?recursive=sometimes", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/tool/read"+tt.query, strings.NewReader(tt.path))
			response := httptest.NewRecorder()
			newRouter().ServeHTTP(response, request)
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func createReadFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	nested := filepath.Join(root, "subdirectory")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(root, "first.txt"):    "first contents\n",
		filepath.Join(nested, "nested.txt"): "nested contents\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func performReadRequest(t *testing.T, path string, recursive bool) *httptest.ResponseRecorder {
	t.Helper()
	requestPath := "/tool/read"
	if recursive {
		requestPath += "?recursive=true"
	}
	request := httptest.NewRequest(http.MethodGet, requestPath, strings.NewReader(path))
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)
	return response
}
