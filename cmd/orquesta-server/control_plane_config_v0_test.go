package main

import "testing"

func TestServerConfigFromEnvV0RechazaBindRemotoSinOptInV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_ADDR", "0.0.0.0:8787")
	t.Setenv("ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM", "")
	t.Setenv("ORQUESTA_SERVER_CONTROL_TOKEN", "")

	if _, err := serverConfigFromEnvV0(); err == nil {
		t.Fatalf("serverConfigFromEnvV0 debe rechazar bind remoto sin opt-in")
	}
}

func TestServerConfigFromEnvV0AceptaBindRemotoConTokenOptInV0(t *testing.T) {
	t.Setenv("ORQUESTA_CODEX_PROJECT_WORKDIR", t.TempDir())
	t.Setenv("ORQUESTA_SERVER_ADDR", "0.0.0.0:8787")
	t.Setenv("ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM", "1")
	t.Setenv("ORQUESTA_SERVER_CONTROL_TOKEN", "secret-control-plane-token-0123456789")
	t.Setenv("ORQUESTA_SERVER_CONTROL_PRINCIPAL", "principal-ref-operator")
	t.Setenv("ORQUESTA_SERVER_CONTROL_PERMISSION_REF", "permission-ref-control-plane")
	t.Setenv("ORQUESTA_SERVER_CONTROL_PUBLIC_REASON", "remote_control_plane_opt_in")

	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatalf("serverConfigFromEnvV0: %v", err)
	}
	if !config.ControlPlane.RemoteAccessOptIn ||
		config.ControlPlane.Principal != "principal-ref-operator" ||
		config.ControlPlane.PermissionRef != "permission-ref-control-plane" {
		t.Fatalf("control plane config=%+v", config.ControlPlane)
	}
}
