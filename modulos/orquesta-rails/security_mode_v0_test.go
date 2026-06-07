package orquestarails

import "testing"

func TestSecurityModeV0NormalizaProgramacionYProduccion(t *testing.T) {
	for _, value := range []string{"programming", "programacion", "dev", "open"} {
		if got := NormalizeSecurityModeV0(value); got != SecurityModeProgrammingV0 {
			t.Fatalf("mode %q=%q want programming", value, got)
		}
	}
	if got := NormalizeSecurityModeV0("production_high"); got != SecurityModeProductionHighV0 {
		t.Fatalf("mode high=%q", got)
	}
	if got := NormalizeSecurityModeV0(""); got != SecurityModeProductionV0 {
		t.Fatalf("mode default=%q", got)
	}
}

func TestRailsModeV0OfflinePorDefecto(t *testing.T) {
	for _, value := range []string{"", "off", "offline", "disabled", "audit", "enforced", "on", "strict"} {
		if got := NormalizeRailsModeV0(value); got != RailsModeOfflineV0 {
			t.Fatalf("rails mode %q=%q want offline", value, got)
		}
	}
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	if RailsAuditV0() || RailsEnforcedV0() {
		t.Fatal("rails no deben reactivarse por env")
	}
}

func TestSecurityModeProgramacionDesactivaRailsDeDetalleAunqueSePidieranOn(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(SecurityModeEnvV0, SecurityModeProgrammingV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")

	if DetailProhibitedRailsEnabledV0() {
		t.Fatal("modo programacion debe quitar rails de detalle")
	}
	if !TextContainsOperationalSensitiveDetailV0("access_token=abc") {
		t.Fatal("secreto efectivo debe detectarse aunque modo programacion quite rails blandos")
	}
}

func TestSecurityModeProduccionNoPermiteReactivarRailsDeDetalle(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(SecurityModeEnvV0, SecurityModeProductionV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")

	if DetailProhibitedRailsEnabledV0() {
		t.Fatal("produccion no debe activar detalle aunque rail este on")
	}
	if !TextContainsOperationalSensitiveDetailV0("access_token=abc") {
		t.Fatal("secreto efectivo debe detectarse aunque rails esten quitados")
	}
}

func TestSecurityModeProduccionNoReactivaDetalleSinRailsEnforced(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeOfflineV0)
	t.Setenv(SecurityModeEnvV0, SecurityModeProductionV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")

	if DetailProhibitedRailsEnabledV0() {
		t.Fatal("produccion no debe activar detalle con modo rails offline")
	}
	if !TextContainsOperationalSensitiveDetailV0("access_token=abc") {
		t.Fatal("secreto efectivo debe detectarse aunque modo rails offline")
	}
}
