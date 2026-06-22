package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus string
	}{
		{name: "success", path: "/?source=test", wantStatus: "status=200"},
		{name: "not found", path: "/missing", wantStatus: "status=404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := log.New(&output, "", 0)
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()

			newHandler(logger).ServeHTTP(response, request)

			entry := output.String()
			for _, want := range []string{
				"request method=GET",
				"path=\"" + tt.path + "\"",
				tt.wantStatus,
				"bytes=",
				"duration=",
				"remote=",
			} {
				if !strings.Contains(entry, want) {
					t.Errorf("log entry %q does not contain %q", entry, want)
				}
			}
		})
	}
}
