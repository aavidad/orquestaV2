package main

import (
	"net/http"
	"net/url"
	"testing"
)

func TestCommandHTTPRedirectPolicyV0SoloPermiteMismoOrigen(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		target  string
		want    bool
	}{
		{name: "mismo origen", baseURL: "http://127.0.0.1:8787", target: "http://127.0.0.1:8787/api/v0/runs/control", want: true},
		{name: "otro origen", baseURL: "http://127.0.0.1:8787", target: "https://external.example.test/api/v0/runs/control"},
		{name: "fragmento", baseURL: "http://127.0.0.1:8787", target: "http://127.0.0.1:8787/api#secret"},
		{name: "credenciales", baseURL: "http://127.0.0.1:8787", target: "http://user:pass@127.0.0.1:8787/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := url.Parse(tt.target)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := commandHTTPRedirectAllowedV0(&http.Request{URL: target}, tt.baseURL)
			if got != tt.want {
				t.Fatalf("allowed=%v want=%v", got, tt.want)
			}
		})
	}
}
