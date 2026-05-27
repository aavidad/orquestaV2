package orquestadomainworkhttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
)

func TestClientV0RedirectRevalidaDestinoYPath(t *testing.T) {
	tests := []struct {
		name      string
		location  string
		wantError string
	}{
		{name: "mismo origen y path controlado", location: "/jobs"},
		{name: "otro origen", location: "https://external.example.test/jobs", wantError: orquestadomainworkhttp.ErrDomainWorkHTTPRedirectDeniedV0},
		{name: "query no controlada", location: "/jobs?token=secret", wantError: orquestadomainworkhttp.ErrDomainWorkHTTPRedirectDeniedV0},
		{name: "path no configurado", location: "/login", wantError: orquestadomainworkhttp.ErrDomainWorkHTTPRedirectDeniedV0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					w.Header().Set("Location", tt.location)
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				_ = json.NewEncoder(w).Encode(orquestadomainwork.DomainWorkJobV0{
					SchemaVersion: orquestadomainwork.DomainWorkJobSchemaV0,
					Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
					JobRef:        "job-ref-redirect-001",
					DomainRef:     "domain-ref-non-opes-001",
					WorkKind:      "compose_external_summary",
				})
			}))
			defer server.Close()

			client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
				BaseURL:       server.URL,
				CreateJobPath: "/jobs",
			})
			if err != nil {
				t.Fatalf("NewClientV0: %v", err)
			}
			_, err = client.CreateDomainWorkJobV0(context.Background(), validJobRequestV0())
			if tt.wantError != "" {
				requireErrorCodeV0(t, err, tt.wantError)
				if calls != 1 {
					t.Fatalf("calls=%d", calls)
				}
				return
			}
			if err != nil || calls != 2 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}
