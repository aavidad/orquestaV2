package orquestafactory

import "testing"

func TestValidateAppSpecRequestV0AllowsOperationalCapabilityConnectorNames(t *testing.T) {
	req := validMinimalRequestV0()
	req.Integraciones = []ConnectorRequestV0{
		{Nombre: "database audit", Proposito: "Auditar cambios de datos", Requerido: true},
		{Nombre: "runtime metrics", Proposito: "Observar salud operativa"},
		{Nombre: "cola de tareas", Proposito: "Coordinar trabajo asincrono"},
		{Nombre: "deploy notifications", Proposito: "Avisar cambios publicados"},
	}

	if issues := ValidateAppSpecRequestV0(req); len(issues) != 0 {
		t.Fatalf("operational capability names must be accepted: %+v", issues)
	}
}

func TestValidateAppSpecRequestV0RejectsProviderBackendSelection(t *testing.T) {
	cases := []struct {
		name      string
		connector ConnectorRequestV0
	}{
		{
			name: "backend concrete",
			connector: ConnectorRequestV0{
				Nombre:        "storage",
				Proposito:     "Usar backend postgres para datos",
				Restricciones: []string{"driver postgres"},
				Requerido:     true,
			},
		},
		{
			name: "provider as connector name",
			connector: ConnectorRequestV0{
				Nombre:    "postgres database",
				Proposito: "Guardar datos de negocio",
				Requerido: true,
			},
		},
		{
			name: "cloud sdk",
			connector: ConnectorRequestV0{
				Nombre:        "asset integration",
				Proposito:     "Subir ficheros con SDK cloud",
				Restricciones: []string{"provider gestionado"},
				Requerido:     true,
			},
		},
	}

	for _, tc := range cases {
		req := validMinimalRequestV0()
		req.Integraciones = []ConnectorRequestV0{tc.connector}

		issues := ValidateAppSpecRequestV0(req)

		if !hasIssueCodeV0(issues, ErrConectorRequeridoNoDisponible) ||
			!hasIssueFieldV0(issues, "integraciones.0.nombre") {
			t.Fatalf("%s: expected provider backend issue, got %+v", tc.name, issues)
		}
	}
}

func TestValidateAppSpecRequestV0RejectsConnectorDSNOrCredential(t *testing.T) {
	req := validMinimalRequestV0()
	req.Integraciones = []ConnectorRequestV0{{
		Nombre:        "analytics",
		Proposito:     "Enviar metricas",
		Restricciones: []string{"dsn=postgres://user:pass@example.invalid/db"},
		Requerido:     true,
	}}

	issues := ValidateAppSpecRequestV0(req)

	if !hasIssueCodeV0(issues, ErrConectorRequeridoNoDisponible) {
		t.Fatalf("expected credential/DSN issue, got %+v", issues)
	}
}
