package orquestaopesconnector

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestRESTClientV0RedirectRevalidaDestinoYPath(t *testing.T) {
	tests := []struct {
		name      string
		location  string
		wantError string
	}{
		{name: "mismo origen con query controlada", location: "/api/jobs?status=pending"},
		{name: "otro origen", location: "https://external.example.test/api/jobs", wantError: ErrOPESHTTPRedirectDeniedV0},
		{name: "fragmento", location: "/api/jobs#secret", wantError: ErrOPESHTTPRedirectDeniedV0},
		{name: "path no controlado", location: "//external.example.test/api/jobs", wantError: ErrOPESHTTPRedirectDeniedV0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					w.Header().Set("Location", tt.location)
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				_ = json.NewEncoder(w).Encode([]ExternalJobV0{{
					ID:            "job-ref-redirect-001",
					Type:          "plan_tema",
					Status:        "pending",
					ExecutionMode: "external",
				}})
			}))
			defer server.Close()
			client := NewRESTClientV0(RESTClientConfigV0{BaseURL: server.URL})

			_, err := client.ListExternalJobsV0(context.Background(), ExternalJobQueryV0{Status: "pending"})
			if tt.wantError != "" {
				requireConnectorErrorV0(t, err, tt.wantError)
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

func requireConnectorErrorV0(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || err.Error() != code {
		t.Fatalf("err=%v want=%s", err, code)
	}
}
