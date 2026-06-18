package orquestarails

import (
	"testing"
)

// Estos tests blindan una regresion historica: en su dia los rails de "detalle
// prohibido" (lista OperationalDetailMarkersV0 con palabras sueltas como
// `token`, `secret`, `password`, `nombre`...) vetaban entregas legitimas porque
// la palabra suelta hacia match en texto normal. La correccion fue dejar
// RailsEnforcedV0()=false duro y exigir env var explicita para el rail de
// detalle. Si alguien reactiva eso por defecto, estos tests deben fallar.

func TestRailsEnforcedV0OffPorDefecto(t *testing.T) {
	if RailsEnforcedV0() {
		t.Fatalf("RailsEnforcedV0 debe estar OFF: reactivarlo revive la regresion de palabras sueltas")
	}
}

func TestDetailProhibitedRailsV0OffPorDefectoSinEnv(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "")
	t.Setenv(SecurityModeEnvV0, "")
	if DetailProhibitedRailsEnabledV0() {
		t.Fatalf("el rail de detalle prohibido no debe activarse sin opt-in explicito")
	}
}

func TestDetailProhibitedRailsV0OffEnModoProgramacionAunConEnv(t *testing.T) {
	t.Setenv(DetailProhibitedRailsEnvV0, "strict")
	t.Setenv(SecurityModeEnvV0, "programming")
	if DetailProhibitedRailsEnabledV0() {
		t.Fatalf("en modo programacion el rail de detalle no debe vetar aunque la env este activa")
	}
}

func TestOperationalDetailMarkerV0NoVetaPalabrasSueltasPorDefecto(t *testing.T) {
	// Con rails off (defecto), una palabra suelta como `nombre`, `token` o
	// `password` en texto pedagogico NO debe marcar detalle operativo.
	for _, value := range []string{
		"el alumno debe escribir su nombre y su contrasena",
		"ejemplo: token de sesion en el examen",
		"campo password del formulario de la oposicion",
	} {
		if TextContainsOperationalDetailMarkerV0(value) {
			t.Fatalf("texto pedagogico no debe disparar rail de detalle: %q", value)
		}
	}
}
