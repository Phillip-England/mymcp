package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouteJSONSendsMappedRequestAndInjectsSandbox(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RequestURI() != "/tool/read?recursive=true" {
			t.Errorf("request = %s %s", r.Method, r.URL.RequestURI())
		}
		if got := r.Header.Get(sandboxHeader); got != "/trusted/workspace" {
			t.Errorf("sandbox = %q, want harness sandbox", got)
		}
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		if string(body) != "docs" {
			t.Errorf("body = %q, want docs", body)
		}
		w.Header().Set("X-Test", "response")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("routed"))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "/trusted/workspace")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{
        "method":"GET",
        "path":"/tool/read?recursive=true",
        "headers":{"X-Mymcp-Sandbox":"/model/chosen","X-Trace":"abc"},
        "body":"docs"
    }`)
	response, err := client.RouteJSON(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusAccepted || response.Text() != "routed" || response.Header.Get("X-Test") != "response" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRouteJSONReturnsServerErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "/workspace")
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.RouteJSON(context.Background(), []byte(`{"method":"GET","path":"/tool/ls","body":"missing"}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound || response.Text() != "not found\n" {
		t.Fatalf("response = %#v", response)
	}
}

func TestRouteJSONRejectsInvalidRequestsBeforeSending(t *testing.T) {
	requests := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "malformed JSON", payload: `{`, want: "decode request JSON"},
		{name: "missing path", payload: `{"method":"GET"}`, want: "requires method and path"},
		{name: "unknown field", payload: `{"method":"GET","path":"/","tool":"read"}`, want: "unknown field"},
		{name: "unknown route", payload: `{"method":"GET","path":"/tool/delete"}`, want: "known endpoint"},
		{name: "wrong method", payload: `{"method":"POST","path":"/tool/ls"}`, want: "requires method GET"},
		{name: "absolute URL", payload: `{"method":"GET","path":"https://example.com/"}`, want: "relative"},
		{name: "protocol-relative path", payload: `{"method":"GET","path":"//example.com/"}`, want: "known endpoint"},
		{name: "unsupported query", payload: `{"method":"GET","path":"/tool/ls?recursive=true"}`, want: "unsupported query"},
	}

	client, err := NewClient("http://127.0.0.1:1", "/workspace")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range requests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.RouteJSON(context.Background(), []byte(tt.payload))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestRouteJSONRequiresHarnessSandboxForTools(t *testing.T) {
	client, err := NewClient("http://example.test", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.RouteJSON(context.Background(), []byte(`{"method":"GET","path":"/tool/tree","body":"."}`))
	if err == nil || !strings.Contains(err.Error(), "SDK sandbox") {
		t.Fatalf("error = %v, want SDK sandbox error", err)
	}
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	for _, baseURL := range []string{"localhost:8765", "ftp://example.com", "https://example.com/base", "https://example.com?x=1"} {
		if _, err := NewClient(baseURL, "/workspace"); err == nil {
			t.Errorf("NewClient(%q) succeeded, want error", baseURL)
		}
	}
}
