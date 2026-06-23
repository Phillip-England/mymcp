package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/phillip-england/mymcp/internal/protocol"
)

func TestTerminalRoutes(t *testing.T) {
	tests := []struct {
		path    string
		outcome string
		message string
	}{
		{path: "/tool/error", outcome: "error", message: "cannot access the required service"},
		{path: "/tool/success", outcome: "success", message: "goal complete"},
	}

	for _, tt := range tests {
		t.Run(tt.outcome, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.message))
			response := httptest.NewRecorder()
			newRouter().ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body = %q", response.Code, response.Body.String())
			}
			if got := response.Header().Get(protocol.TerminalHeader); got != tt.outcome {
				t.Errorf("terminal header = %q, want %q", got, tt.outcome)
			}
			if response.Body.String() != tt.message {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.message)
			}
		})
	}
}

func TestTerminalRoutesRequireMessage(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/tool/error", strings.NewReader(" \n\t"))
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if got := response.Header().Get(protocol.TerminalHeader); got != "" {
		t.Errorf("invalid request has terminal header %q", got)
	}
}

func TestTerminalRoutesRejectOversizedMessage(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/tool/success",
		strings.NewReader(strings.Repeat("x", maxTerminalMessageSize+1)),
	)
	response := httptest.NewRecorder()
	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}
