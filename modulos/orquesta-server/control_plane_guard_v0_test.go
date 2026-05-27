package orquestaserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateConfigV0RechazaBindRemotoSinOptInV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		Addr:     "0.0.0.0:8787",
		StateDir: t.TempDir(),
	})

	if err := ValidateConfigV0(config); err == nil {
		t.Fatalf("ValidateConfigV0 remoto sin opt-in debe fallar")
	}
}

func TestValidateConfigV0AceptaBindRemotoSoloConTokenV0(t *testing.T) {
	config := NormalizeConfigV0(ConfigV0{
		Addr:     "0.0.0.0:8787",
		StateDir: t.TempDir(),
		ControlPlane: ControlPlaneConfigV0{
			RemoteAccessOptIn: true,
			Token:             "token-ref-control-plane-test",
			Principal:         "principal-ref-operator",
			PermissionRef:     "permission-ref-control-plane",
			PublicReason:      "remote_control_plane_opt_in",
		},
	})

	if err := ValidateConfigV0(config); err != nil {
		t.Fatalf("ValidateConfigV0 remoto con token: %v", err)
	}
}

func TestControlPlaneGuardV0BloqueaMutacionRemotaSinTokenV0(t *testing.T) {
	runtime := newControlPlaneRuntimeForTestV0(t, ConfigV0{
		Addr:     "0.0.0.0:8787",
		StateDir: t.TempDir(),
		ControlPlane: ControlPlaneConfigV0{
			RemoteAccessOptIn: true,
			Token:             "secret-control-plane-token",
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", nil)
	runtime.HandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() == "" || rec.Body.String() == "secret-control-plane-token" {
		t.Fatalf("respuesta no debe filtrar token: %q", rec.Body.String())
	}
}

func TestControlPlaneGuardV0PermiteMutacionRemotaConTokenYPrincipalV0(t *testing.T) {
	runtime := newControlPlaneRuntimeForTestV0(t, ConfigV0{
		Addr:     "0.0.0.0:8787",
		StateDir: t.TempDir(),
		ControlPlane: ControlPlaneConfigV0{
			RemoteAccessOptIn: true,
			Token:             "secret-control-plane-token",
			Principal:         "principal-ref-default",
			PermissionRef:     "permission-ref-control-plane",
			PublicReason:      "remote_control_plane_opt_in",
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", nil)
	req.Header.Set("Authorization", "Bearer secret-control-plane-token")
	req.Header.Set("X-Orquesta-Principal", "principal-ref-operator")
	runtime.HandlerV0().ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestControlPlaneGuardV0MantieneStatusLecturaSinTokenV0(t *testing.T) {
	runtime := newControlPlaneRuntimeForTestV0(t, ConfigV0{
		Addr:     "0.0.0.0:8787",
		StateDir: t.TempDir(),
		ControlPlane: ControlPlaneConfigV0{
			RemoteAccessOptIn: true,
			Token:             "secret-control-plane-token",
		},
	})

	rec := httptest.NewRecorder()
	runtime.HandlerV0().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuditQueryKeysV0NoGuardaValoresDeQueryV0(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control?token=secret&run_ref=run-ref-001", nil)
	keys := auditQueryKeysV0(req)
	if len(keys) != 2 || keys[0] != "run_ref" || keys[1] != "sensitive_query_key_redacted" {
		t.Fatalf("query keys=%v", keys)
	}
}

func newControlPlaneRuntimeForTestV0(t *testing.T, config ConfigV0) *RuntimeV0 {
	t.Helper()
	config.AuditDisabled = true
	runtime, err := NewRuntimeV0(config, RuntimeDepsV0{
		AppHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		}),
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	return runtime
}
