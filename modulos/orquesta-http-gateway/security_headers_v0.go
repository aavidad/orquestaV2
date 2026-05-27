package orquestahttpgateway

import (
	"net/http"
	"strings"
)

const (
	controlPlaneHTMLHeaderPolicyV0 = "control-plane-html-v0"
	controlPlaneJSONHeaderPolicyV0 = "control-plane-json-v0"
	controlPlaneMCPHeaderPolicyV0  = "control-plane-mcp-v0"

	controlPlaneHTMLCSPV0 = "default-src 'self'; base-uri 'none'; form-action 'self'; " +
		"frame-ancestors 'none'; object-src 'none'; img-src 'self' data:; " +
		"script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'"
	controlPlaneJSONCSPV0 = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'"
)

func NewControlPlaneHTTPHeadersV0(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ApplyControlPlaneHTTPHeadersV0(w.Header(), ControlPlaneHTTPHeaderPolicyForPathV0(r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func ApplyControlPlaneHTTPHeadersV0(header http.Header, policyRef string) {
	if header == nil {
		return
	}
	policyRef = strings.TrimSpace(policyRef)
	if policyRef == "" {
		policyRef = controlPlaneJSONHeaderPolicyV0
	}
	if policyRef == controlPlaneHTMLHeaderPolicyV0 {
		header.Set("Content-Security-Policy", controlPlaneHTMLCSPV0)
	} else {
		header.Set("Content-Security-Policy", controlPlaneJSONCSPV0)
	}
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Cache-Control", "no-store")
	header.Set("Pragma", "no-cache")
	header.Set("X-Orquesta-Control-Plane-Header-Policy", policyRef)
}

func ControlPlaneHTTPHeaderPolicyForPathV0(path string) string {
	path = strings.TrimSpace(path)
	if path == "/mcp" {
		return controlPlaneMCPHeaderPolicyV0
	}
	if strings.HasPrefix(path, "/api/") {
		return controlPlaneJSONHeaderPolicyV0
	}
	return controlPlaneHTMLHeaderPolicyV0
}
