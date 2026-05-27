package orquestaweb

import "testing"

func TestWebRESTEndpointURLV0PreservaBasePathYEndpointRelativo(t *testing.T) {
	got := webRESTEndpointURLV0("http://orquesta.internal/control/", "/api/v0/apps/spec", "")
	if got != "http://orquesta.internal/control/api/v0/apps/spec" {
		t.Fatalf("url = %q", got)
	}
}

func TestWebRESTEndpointURLV0RechazaDestinoAmbiguo(t *testing.T) {
	for _, tc := range []struct {
		name     string
		baseURL  string
		endpoint string
	}{
		{name: "base_userinfo", baseURL: "https://u:p@example.test", endpoint: "/api/v0/apps/spec"},
		{name: "base_query", baseURL: "https://example.test?token=x", endpoint: "/api/v0/apps/spec"},
		{name: "endpoint_absoluto", baseURL: "https://example.test", endpoint: "https://evil.test/api"},
		{name: "endpoint_query", baseURL: "https://example.test", endpoint: "/api/v0/apps/spec?x=1"},
		{name: "endpoint_parent", baseURL: "https://example.test", endpoint: "/api/../secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := webRESTEndpointURLV0(tc.baseURL, tc.endpoint, ""); got != webRESTURLPolicyInvalidV0 {
				t.Fatalf("url valida inesperada: %q", got)
			}
		})
	}
}
