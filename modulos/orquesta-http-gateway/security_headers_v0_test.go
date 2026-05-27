package orquestahttpgateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAppGatewayMuxV0AplicaHeadersSeguridadHTMLYJSONV0(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		handlers   RouteHandlersV0
		wantPolicy string
		wantCSP    string
	}{
		{
			name:       "html",
			path:       RouteNuevaAppV0,
			handlers:   RouteHandlersV0{NuevaApp: markerHandler("html")},
			wantPolicy: controlPlaneHTMLHeaderPolicyV0,
			wantCSP:    "frame-ancestors 'none'",
		},
		{
			name:       "json",
			path:       RouteAppSpecV0,
			handlers:   RouteHandlersV0{AppSpec: markerHandler("json")},
			wantPolicy: controlPlaneJSONHeaderPolicyV0,
			wantCSP:    "default-src 'none'",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)

			NewAppGatewayMuxV0(tc.handlers).ServeHTTP(rec, req)

			assertControlPlaneHeaderV0(t, rec.Header(), tc.wantPolicy, tc.wantCSP)
		})
	}
}

func TestControlPlaneHTTPHeadersV0NoCopianDatosDeRequestV0(t *testing.T) {
	handler := NewControlPlaneHTTPHeadersV0(markerHandler("ok"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v0/apps/spec?token=secret", nil)
	req.Header.Set("Cookie", "session=secret")
	req.Header.Set("Authorization", "Bearer secret")

	handler.ServeHTTP(rec, req)

	for name, values := range rec.Header() {
		joined := strings.Join(values, " ")
		if strings.Contains(joined, "secret") || strings.Contains(joined, "token=") {
			t.Fatalf("header %s filtra request: %q", name, joined)
		}
	}
}

func TestControlPlaneHTTPHeaderPolicyForPathV0(t *testing.T) {
	if got := ControlPlaneHTTPHeaderPolicyForPathV0("/mcp"); got != controlPlaneMCPHeaderPolicyV0 {
		t.Fatalf("mcp policy=%q", got)
	}
	if got := ControlPlaneHTTPHeaderPolicyForPathV0("/api/v0/runs/control"); got != controlPlaneJSONHeaderPolicyV0 {
		t.Fatalf("api policy=%q", got)
	}
	if got := ControlPlaneHTTPHeaderPolicyForPathV0("/ops"); got != controlPlaneHTMLHeaderPolicyV0 {
		t.Fatalf("html policy=%q", got)
	}
}

func assertControlPlaneHeaderV0(t *testing.T, header http.Header, wantPolicy string, wantCSP string) {
	t.Helper()
	if got := header.Get("X-Orquesta-Control-Plane-Header-Policy"); got != wantPolicy {
		t.Fatalf("policy=%q want %q headers=%v", got, wantPolicy, header)
	}
	if got := header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff=%q headers=%v", got, header)
	}
	if got := header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache=%q headers=%v", got, header)
	}
	if got := header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("referrer=%q headers=%v", got, header)
	}
	if got := header.Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("frame=%q headers=%v", got, header)
	}
	if !strings.Contains(header.Get("Content-Security-Policy"), wantCSP) {
		t.Fatalf("csp=%q want fragment %q", header.Get("Content-Security-Policy"), wantCSP)
	}
}
