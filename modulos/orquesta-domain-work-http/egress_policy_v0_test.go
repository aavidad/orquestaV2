package orquestadomainworkhttp_test

import (
	"testing"

	orquestadomainworkhttp "orquesta/modulos/orquesta-domain-work-http"
)

func TestClientV0AplicaPoliticaEgressYClasificaDestino(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		policy   orquestadomainworkhttp.EgressPolicyV0
		wantCode string
		wantHost string
		wantClas string
	}{
		{
			name:     "rechaza credenciales",
			baseURL:  "https://user:secret@example.test",
			policy:   orquestadomainworkhttp.AllowlistEgressPolicyV0("example.test"),
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPBaseURLInvalidV0,
		},
		{
			name:     "bloquea metadata en smoke local",
			baseURL:  "http://169.254.169.254",
			policy:   orquestadomainworkhttp.SmokeLocalEgressPolicyV0(),
			wantCode: orquestadomainworkhttp.ErrDomainWorkHTTPEgressDestinationDeniedV0,
		},
		{
			name:     "permite allowlist externa",
			baseURL:  "https://api.example.test:8443/base",
			policy:   orquestadomainworkhttp.AllowlistEgressPolicyV0("api.example.test:8443"),
			wantHost: "api.example.test:8443",
			wantClas: orquestadomainworkhttp.DestinationClassExternalUnknownV0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
				BaseURL:      tt.baseURL,
				EgressPolicy: tt.policy,
			})
			if tt.wantCode != "" {
				requireErrorCodeV0(t, err, tt.wantCode)
				return
			}
			if err != nil {
				t.Fatalf("NewClientV0: %v", err)
			}
			destination := client.DestinationV0()
			if destination.Host != tt.wantHost || destination.Classification != tt.wantClas {
				t.Fatalf("destination=%+v", destination)
			}
		})
	}
}

func TestClientV0RechazaPathsQueTransportanDestinoOQuery(t *testing.T) {
	tests := []struct {
		name       string
		createPath string
		submitPath string
		wantCode   string
	}{
		{
			name:       "create query",
			createPath: "/jobs?token=secret",
			wantCode:   orquestadomainworkhttp.ErrDomainWorkHTTPCreatePathInvalidV0,
		},
		{
			name:       "create host override",
			createPath: "//evil.example/jobs",
			wantCode:   orquestadomainworkhttp.ErrDomainWorkHTTPCreatePathInvalidV0,
		},
		{
			name:       "submit url absoluta",
			submitPath: "https://evil.example/artifacts",
			wantCode:   orquestadomainworkhttp.ErrDomainWorkHTTPSubmitPathInvalidV0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := orquestadomainworkhttp.NewClientV0(orquestadomainworkhttp.ConfigV0{
				BaseURL:            "http://127.0.0.1:1",
				CreateJobPath:      tt.createPath,
				SubmitArtifactPath: tt.submitPath,
			})
			requireErrorCodeV0(t, err, tt.wantCode)
		})
	}
}
