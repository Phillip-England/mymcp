package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantLog string
	}{
		{name: "success", path: "/", wantLog: `^\[GET\]\[/\]\[200\]\[[^]]+\]\n$`},
		{name: "invalid query", path: "/?source=test", wantLog: `^\[GET\]\[/\]\[400\]\[[^]]+\]\n$`},
		{name: "not found", path: "/missing", wantLog: `^\[GET\]\[/missing\]\[404\]\[[^]]+\]\n$`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := log.New(&output, "", 0)
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			response := httptest.NewRecorder()

			newHandler(logger).ServeHTTP(response, request)

			if !regexp.MustCompile(tt.wantLog).MatchString(output.String()) {
				t.Errorf("log entry = %q, want match for %q", output.String(), tt.wantLog)
			}
		})
	}
}
