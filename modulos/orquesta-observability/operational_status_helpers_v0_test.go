package orquestaobservability

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func validOperationalStatusQueryV0() OperationalStatusQueryV0 {
	return OperationalStatusQueryV0{
		SchemaVersion: OperationalStatusQuerySchemaVersionV0,
		RequestID:     "req_20260504_000001",
		CorrelationID: "corr_20260504_000001",
		Consumer: OperationalStatusConsumerV0{
			Module:  "orquesta-cli",
			Channel: "cli",
		},
		Locale:     "es-ES",
		Scope:      OperationalStatusScopeProyectoV0,
		SubjectRef: "project_20260504_000001",
		TraceRef:   "trace_20260504_000001",
		TimeWindow: &OperationalStatusTimeWindowV0{
			From: "2026-05-04T08:00:00Z",
			To:   "2026-05-04T10:00:00Z",
		},
		IncludeSections: []string{
			OperationalStatusSectionEstadoV0,
			OperationalStatusSectionProgresoV0,
			OperationalStatusSectionBloqueosV0,
		},
		Limit: 10,
		Freshness: &OperationalStatusFreshnessRequestV0{
			MaxAgeSeconds: 120,
			WatermarkRef:  "watermark_20260504_000001",
		},
	}
}

func validDiagnosticoCompactoV0() DiagnosticoCompactoV0 {
	percent := 50.0
	return DiagnosticoCompactoV0{
		SchemaVersion: DiagnosticoCompactoSchemaVersionV0,
		DiagnosticID:  "diagnostic_20260504_000001",
		GeneratedAt:   "2026-05-04T10:00:00Z",
		CorrelationID: "corr_20260504_000001",
		Scope:         OperationalStatusScopeProyectoV0,
		SubjectRef:    "project_20260504_000001",
		ProjectionRef: "projection_20260504_000001",
		Freshness: DiagnosticoFreshnessV0{
			WatermarkRef:  "watermark_20260504_000001",
			MaxAgeSeconds: 60,
			Partial:       false,
			Stale:         false,
		},
		Estado: DiagnosticoEstadoDegradedV0,
		Progreso: DiagnosticoProgresoV0{
			Completed: 2,
			Total:     4,
			Percent:   &percent,
			Phase:     "validacion",
			Summary:   "Progreso agregado compacto",
		},
		Salud: []DiagnosticoSaludCheckV0{
			{
				Area:         "runtime",
				Severity:     OrquestaEventSeverityWarningV0,
				Estado:       DiagnosticoEstadoDegradedV0,
				I18nKey:      "observability.health.runtime_degraded",
				EvidenceRefs: []string{"event_20260504_000010"},
			},
		},
		Bloqueos: []DiagnosticoBloqueoV0{
			{
				BlockerRef:   "blocker_20260504_000001",
				Severity:     OrquestaEventSeverityWarningV0,
				OwnerArea:    "runtime",
				Summary:      "Espera evidencia agregada",
				EvidenceRefs: []string{"artifact_20260504_000001"},
			},
		},
		ActividadReciente: []DiagnosticoActividadV0{
			{
				ActivityRef: "activity_20260504_000001",
				OccurredAt:  "2026-05-04T09:59:00Z",
				Area:        "core",
				Summary:     "Transicion agregada observada",
				EventRef:    "event_20260504_000011",
			},
		},
		Contadores: map[string]float64{
			"tasks_completed": 2,
			"tasks_total":     4,
		},
		Referencias: []DiagnosticoReferenciaV0{
			{
				Rel:        "event",
				TargetType: "event",
				TargetRef:  "event_20260504_000011",
			},
		},
		Warnings: []DiagnosticoWarningV0{
			{
				Code:    "frescura_parcial",
				Section: "progreso",
				Summary: "Seccion calculada con proyeccion parcial",
			},
		},
		Privacy: DiagnosticoPrivacyV0{},
	}
}

func cloneOperationalStatusQueryV0(t *testing.T, query OperationalStatusQueryV0) OperationalStatusQueryV0 {
	t.Helper()
	data, err := json.Marshal(query)
	if err != nil {
		t.Fatalf("marshal query: %v", err)
	}
	var clone OperationalStatusQueryV0
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("unmarshal query: %v", err)
	}
	return clone
}

func assertOperationalStatusIssueV0(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", want)
	}
	var validationErr OperationalStatusValidationErrorV0
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type=%T, want OperationalStatusValidationErrorV0", err)
	}
	for _, issue := range validationErr.Issues {
		if issue.Code == want {
			return
		}
	}
	t.Fatalf("missing issue %q in %+v", want, validationErr.Issues)
}

func leftPadOperationalStatusTestV0(index int) string {
	return fmt.Sprintf("%06d", index)
}
