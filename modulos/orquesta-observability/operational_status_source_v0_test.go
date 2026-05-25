package orquestaobservability

import "testing"

func TestFilterDiagnosticoCompactoForQueryV0AplicaContratoCompacto(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()
	query := validOperationalStatusQueryV0()
	query.IncludeSections = []string{
		OperationalStatusSectionEstadoV0,
		OperationalStatusSectionSaludV0,
	}
	query.Limit = 1

	filtered, err := FilterDiagnosticoCompactoForQueryV0(diagnostic, query)
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(filtered.Salud) != 1 {
		t.Fatalf("salud=%d", len(filtered.Salud))
	}
	if filtered.Progreso.Completed != 0 || len(filtered.Bloqueos) != 0 ||
		len(filtered.ActividadReciente) != 0 || len(filtered.Referencias) != 0 {
		t.Fatalf("filtered=%+v", filtered)
	}
}
