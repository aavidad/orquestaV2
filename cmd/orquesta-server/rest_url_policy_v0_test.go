package main

import "testing"

func TestCommandRESTEndpointURLV0PreservaBasePath(t *testing.T) {
	got, err := commandRESTEndpointURLV0("http://127.0.0.1:8787/base/", "/api/v0/external-work/run")
	if err != nil {
		t.Fatalf("commandRESTEndpointURLV0: %v", err)
	}
	if got != "http://127.0.0.1:8787/base/api/v0/external-work/run" {
		t.Fatalf("url = %q", got)
	}
}

func TestCommandRESTEndpointURLV0RechazaDestinoAmbiguo(t *testing.T) {
	for _, tc := range []struct {
		name     string
		baseURL  string
		endpoint string
	}{
		{name: "base_userinfo", baseURL: "http://u:p@127.0.0.1:8787", endpoint: "/api/v0/server/status"},
		{name: "base_query", baseURL: "http://127.0.0.1:8787?token=x", endpoint: "/api/v0/server/status"},
		{name: "endpoint_absolute", baseURL: "http://127.0.0.1:8787", endpoint: "http://evil.test/status"},
		{name: "endpoint_parent", baseURL: "http://127.0.0.1:8787", endpoint: "/api/../status"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := commandRESTEndpointURLV0(tc.baseURL, tc.endpoint); err == nil {
				t.Fatalf("url valida inesperada: %q", got)
			}
		})
	}
}
