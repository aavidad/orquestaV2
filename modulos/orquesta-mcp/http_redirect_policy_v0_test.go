package orquestamcp

import (
	"net/http"
	"net/url"
	"testing"
)

func TestMCPHTTPRedirectPolicyV0SoloPermiteMismoOrigen(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		target  string
		want    bool
	}{
		{name: "mismo origen", baseURL: "http://orquesta.internal", target: "http://orquesta.internal/api/v0/apps/spec", want: true},
		{name: "otro origen", baseURL: "http://orquesta.internal", target: "https://external.example.test/api/v0/apps/spec"},
		{name: "fragmento", baseURL: "http://orquesta.internal", target: "http://orquesta.internal/api#secret"},
		{name: "credenciales", baseURL: "http://orquesta.internal", target: "http://user:pass@orquesta.internal/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := url.Parse(tt.target)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := mcpHTTPRedirectAllowedV0(&http.Request{URL: target}, tt.baseURL)
			if got != tt.want {
				t.Fatalf("allowed=%v want=%v", got, tt.want)
			}
		})
	}
}
