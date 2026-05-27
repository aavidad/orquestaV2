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

func TestSecurityModeProgramacionDesactivaRailsDeDetalleAunqueSePidieranOn(t *testing.T) {
	t.Setenv(SecurityModeEnvV0, SecurityModeProgrammingV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")

	if DetailProhibitedRailsEnabledV0() {
		t.Fatal("modo programacion debe quitar rails de detalle")
	}
	if TextContainsOperationalSensitiveDetailV0("access_token=abc") {
		t.Fatal("modo programacion no debe bloquear por detalle sensible")
	}
}

func TestSecurityModeProduccionPermiteRailsDeDetalleOptIn(t *testing.T) {
	t.Setenv(SecurityModeEnvV0, SecurityModeProductionV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")

	if !DetailProhibitedRailsEnabledV0() {
		t.Fatal("produccion con rail on debe activar detalle")
	}
	if !TextContainsOperationalSensitiveDetailV0("access_token=abc") {
		t.Fatal("produccion con rail on debe detectar detalle sensible")
	}
}
