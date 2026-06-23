package protocol

import (
	"net/url"
	"strings"
	"testing"
)

func TestEndpointsAreUniqueAndDiscoverable(t *testing.T) {
	seenNames := make(map[string]bool)
	seenPaths := make(map[string]bool)
	for _, endpoint := range Endpoints() {
		if seenNames[endpoint.Name] {
			t.Errorf("duplicate endpoint name %q", endpoint.Name)
		}
		if seenPaths[endpoint.Path] {
			t.Errorf("duplicate endpoint path %q", endpoint.Path)
		}
		seenNames[endpoint.Name] = true
		seenPaths[endpoint.Path] = true
		if endpoint.Name == "" || endpoint.Category == "" || endpoint.Purpose == "" || endpoint.Request == "" || endpoint.Example == "" || endpoint.Response == "" {
			t.Errorf("endpoint %q has incomplete catalog metadata: %#v", endpoint.Path, endpoint)
		}
		found, ok := Lookup(endpoint.Path)
		if !ok || found.Name != endpoint.Name || found.Method != endpoint.Method {
			t.Errorf("Lookup(%q) = %#v, %v; want method %s", endpoint.Path, found, ok, endpoint.Method)
		}
	}
}

func TestValidateQuery(t *testing.T) {
	endpoint, ok := Lookup("/tool/read")
	if !ok {
		t.Fatal("read endpoint is missing")
	}
	for _, tt := range []struct {
		name  string
		query url.Values
		want  string
	}{
		{name: "valid", query: url.Values{"recursive": {"true"}}},
		{name: "unknown", query: url.Values{"depth": {"2"}}, want: "unsupported"},
		{name: "repeated", query: url.Values{"recursive": {"true", "false"}}, want: "must appear once"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := endpoint.ValidateQuery(tt.query)
			if tt.want == "" && err != nil {
				t.Fatal(err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestEndpointPatterns(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/", want: "GET /{$}"},
		{path: "/tool/read", want: "GET /tool/read"},
		{path: "/tool/error", want: "POST /tool/error"},
		{path: "/tool/success", want: "POST /tool/success"},
		{path: "/tool/write", want: "POST /tool/write"},
	}
	for _, tt := range tests {
		endpoint, ok := Lookup(tt.path)
		if !ok {
			t.Fatalf("Lookup(%q) did not find endpoint", tt.path)
		}
		if got := endpoint.Pattern(); got != tt.want {
			t.Errorf("Pattern() = %q, want %q", got, tt.want)
		}
	}
}
