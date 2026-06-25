package orquestaserver

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const controlPlaneDeniedCodeV0 = "control_plane_authorization_required"

const controlPlaneReadOnlyAppIntakeGuidedTurnPathV0 = "/api/v0/apps/intake/guided-turn"

type controlPlaneDecisionV0 struct {
	Mutable       bool
	Allowed       bool
	Principal     string
	PermissionRef string
	Reason        string
}

func (runtime *RuntimeV0) controlPlaneGuardHTTPHandlerV0(next http.Handler) http.Handler {
	if next == nil {
		return next
	}
	config := NormalizeConfigV0(runtime.config)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decision := controlPlaneAuthorizeRequestV0(config, r)
		if decision.Mutable {
			runtime.auditControlPlaneDecisionV0(r, decision)
		}
		if !decision.Allowed {
			writeControlPlaneDeniedV0(w, decision.Reason)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func controlPlaneAuthorizeRequestV0(config ConfigV0, r *http.Request) controlPlaneDecisionV0 {
	config = NormalizeConfigV0(config)
	decision := controlPlaneDecisionV0{
		Mutable:       isMutableControlPlaneRequestV0(r),
		Allowed:       true,
		Principal:     config.ControlPlane.Principal,
		PermissionRef: config.ControlPlane.PermissionRef,
		Reason:        config.ControlPlane.PublicReason,
	}
	if !decision.Mutable || controlPlaneAddrIsLoopbackV0(config.Addr) {
		return decision
	}
	decision.Allowed = false
	decision.Reason = controlPlaneDeniedCodeV0
	if config.ControlPlane.Token == "" {
		return decision
	}
	if controlPlaneTokenMatchesV0(config.ControlPlane.Token, controlPlaneRequestTokenV0(r)) {
		decision.Allowed = true
		decision.Reason = firstNonEmptyControlPlaneV0(config.ControlPlane.PublicReason, "remote_control_plane_opt_in")
		if principal := compactControlPlaneIDV0(r.Header.Get("X-Orquesta-Principal")); principal != "" {
			decision.Principal = principal
		}
	}
	return decision
}

func isMutableControlPlaneRequestV0(r *http.Request) bool {
	if r == nil || r.URL == nil || methodIsReadOnlyControlPlaneV0(r.Method) {
		return false
	}
	path := r.URL.Path
	if path == "/healthz" || path == ServerStatusLegacyEndpointV0 || path == ServerStatusEndpointV0 {
		return false
	}
	if path == controlPlaneReadOnlyAppIntakeGuidedTurnPathV0 {
		return false
	}
	if path == "/mcp" || strings.HasPrefix(path, "/api/v0/") {
		return true
	}
	switch path {
	case "/nueva-app", "/app-change", "/run-control", "/run-queue", "/ops":
		return true
	default:
		return false
	}
}

func methodIsReadOnlyControlPlaneV0(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "", http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func controlPlaneRequestTokenV0(r *http.Request) string {
	if r == nil {
		return ""
	}
	if token := strings.TrimSpace(r.Header.Get("X-Orquesta-Control-Token")); token != "" {
		return token
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(auth) >= len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		return strings.TrimSpace(auth[len(prefix):])
	}
	return ""
}

func controlPlaneTokenMatchesV0(want string, got string) bool {
	want = strings.TrimSpace(want)
	got = strings.TrimSpace(got)
	if want == "" || got == "" || len(want) != len(got) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func compactControlPlaneIDV0(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 96 {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == ':' || r == '.' {
			continue
		}
		return ""
	}
	return value
}

func (runtime *RuntimeV0) auditControlPlaneDecisionV0(r *http.Request, decision controlPlaneDecisionV0) {
	if runtime == nil || r == nil {
		return
	}
	status := "denied"
	if decision.Allowed {
		status = "allowed"
	}
	runtime.auditEventV0(r.Context(), "control_plane_authorization", status, "", map[string]interface{}{
		"method":         r.Method,
		"path":           r.URL.Path,
		"principal":      decision.Principal,
		"permission_ref": decision.PermissionRef,
		"bind":           NormalizeConfigV0(runtime.config).Addr,
		"decision":       status,
		"reason":         decision.Reason,
	})
}

func writeControlPlaneDeniedV0(w http.ResponseWriter, reason string) {
	_ = writeServerJSONResponseV0(w, http.StatusUnauthorized, map[string]interface{}{
		"status": "error",
		"errores_publicos": []map[string]string{{
			"code":    controlPlaneDeniedCodeV0,
			"message": firstNonEmptyControlPlaneV0(reason, controlPlaneDeniedCodeV0),
		}},
	})
}

func firstNonEmptyControlPlaneV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
