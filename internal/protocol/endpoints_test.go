package protocol

import "testing"

func TestEndpointsAreUniqueAndDiscoverable(t *testing.T) {
	seen := make(map[string]bool)
	for _, endpoint := range Endpoints() {
		if seen[endpoint.Path] {
			t.Errorf("duplicate endpoint path %q", endpoint.Path)
		}
		seen[endpoint.Path] = true
		found, ok := Lookup(endpoint.Path)
		if !ok || found.Method != endpoint.Method {
			t.Errorf("Lookup(%q) = %#v, %v; want method %s", endpoint.Path, found, ok, endpoint.Method)
		}
	}
}

func TestEndpointPatterns(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/", want: "GET /{$}"},
		{path: "/tool/read", want: "GET /tool/read"},
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
