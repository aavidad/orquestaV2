package orquestaweb

import (
	"encoding/json"
	"strings"
	"testing"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestWebOperationalStatusQueryV0UsaContratoPublicoReadOnly(t *testing.T) {
	query := NewWebOperationalStatusQueryV0(WebOperationalStatusQueryInputV0{
		RequestID:     "req_20260504_000901",
		CorrelationID: "corr_20260504_000901",
		Locale:        "es-ES",
		SubjectRef:    "flow_20260504_000901",
	})

	if err := orquestaobservability.ValidateOperationalStatusQueryV0(query); err != nil {
		t.Fatalf("query publica invalida: %v", err)
	}
	if query.SchemaVersion != orquestaobservability.OperationalStatusQuerySchemaVersionV0 ||
		query.Consumer.Module != "orquesta-web" ||
		query.Consumer.Channel != orquestaobservability.OperationalStatusConsumerWebChannelV0 ||
		query.Scope != orquestaobservability.OperationalStatusScopeFlujoV0 ||
		query.Limit != 10 {
		t.Fatalf("query no conserva frontera web read-only: %+v", query)
	}
	wantSections := []string{"estado", "progreso", "bloqueos", "salud", "actividad_reciente"}
	if strings.Join(query.IncludeSections, ",") != strings.Join(wantSections, ",") {
		t.Fatalf("secciones=%+v", query.IncludeSections)
	}
}

func TestWebOperationalStatusPanelV0CompactaDiagnosticoSinDumpInterno(t *testing.T) {
	diagnostic := validOperationalDiagnosticForWebV0()

	panel := NewWebOperationalStatusPanelV0("es-ES", diagnostic)

	if panel.SchemaVersion != WebOperationalStatusPanelSchemaV0 ||
		panel.Estado != WebOperationalStatusEstadoBlockedV0 ||
		panel.EstadoVisual != "blocked" ||
		panel.FaseActual != "orquestacion" ||
		!panel.TieneBloqueos ||
		!panel.PrivacyOK {
		t.Fatalf("panel inesperado: %+v", panel)
	}
	if len(panel.Fases) != 1 || !panel.Fases[0].Actual || panel.Fases[0].Progress.Completed != 2 || panel.Fases[0].Progress.Total != 5 {
		t.Fatalf("fases/progreso incompletos: %+v", panel.Fases)
	}
	if len(panel.Bloqueos) != 1 || panel.Bloqueos[0].Ref != "blocker_20260504_000901" || panel.Bloqueos[0].EvidenceRefs[0] != "event_20260504_000901" {
		t.Fatalf("bloqueos=%+v", panel.Bloqueos)
	}
	if len(panel.Salud) != 1 || panel.Salud[0].I18nKey != "operational.health.runtime.blocked" {
		t.Fatalf("salud=%+v", panel.Salud)
	}
	if len(panel.ActividadReciente) != 1 || panel.ActividadReciente[0].EventRef != "event_20260504_000902" {
		t.Fatalf("actividad=%+v", panel.ActividadReciente)
	}
	if panel.Frescura.WatermarkRef != "watermark_20260504_000901" || !panel.Frescura.Partial {
		t.Fatalf("frescura=%+v", panel.Frescura)
	}

	raw, err := json.MarshalIndent(panel, "", "  ")
	if err != nil {
		t.Fatalf("marshal panel: %v", err)
	}
	got := string(raw)
	forbidden := []string{
		"DiagnosticoCompactoV0",
		"contains_secret",
		"contains_transcript",
		"contains_prompt",
		"contains_completion",
		"contains_connection_detail",
		"sql",
		"dsn",
		"provider",
		"runtime_provider",
		"transcript",
	}
	for _, value := range forbidden {
		if strings.Contains(strings.ToLower(got), value) {
			t.Fatalf("panel expone detalle prohibido %q: %s", value, got)
		}
	}
}

func TestWebOperationalStatusPanelV0EstadoDesconocidoNoInventaFases(t *testing.T) {
	diagnostic := validOperationalDiagnosticForWebV0()
	diagnostic.Estado = "estado_futuro"
	diagnostic.Progreso = orquestaobservability.DiagnosticoProgresoV0{}
	diagnostic.Bloqueos = nil

	panel := NewWebOperationalStatusPanelV0("en-US", diagnostic)

	if panel.Estado != WebOperationalStatusEstadoUnknownV0 || panel.EstadoVisual != "unknown" {
		t.Fatalf("estado futuro debe degradar a unknown: %+v", panel)
	}
	if len(panel.Fases) != 0 || panel.TieneBloqueos {
		t.Fatalf("no debe inventar fases ni bloqueos: fases=%+v bloqueos=%+v", panel.Fases, panel.Bloqueos)
	}
}

func validOperationalDiagnosticForWebV0() orquestaobservability.DiagnosticoCompactoV0 {
	percent := 40.0
	return orquestaobservability.DiagnosticoCompactoV0{
		SchemaVersion: orquestaobservability.DiagnosticoCompactoSchemaVersionV0,
		DiagnosticID:  "diag_20260504_000901",
		GeneratedAt:   "2026-05-04T10:00:00Z",
		CorrelationID: "corr_20260504_000901",
		Scope:         orquestaobservability.OperationalStatusScopeFlujoV0,
		SubjectRef:    "flow_20260504_000901",
		ProjectionRef: "projection_20260504_000901",
		Freshness: orquestaobservability.DiagnosticoFreshnessV0{
			WatermarkRef:  "watermark_20260504_000901",
			MaxAgeSeconds: 120,
			Partial:       true,
		},
		Estado: orquestaobservability.DiagnosticoEstadoBlockedV0,
		Progreso: orquestaobservability.DiagnosticoProgresoV0{
			Completed: 2,
			Total:     5,
			Percent:   &percent,
			Phase:     "orquestacion",
			Summary:   "Esperando validacion publica",
		},
		Salud: []orquestaobservability.DiagnosticoSaludCheckV0{{
			Area:         "runtime",
			Severity:     "warning",
			Estado:       orquestaobservability.DiagnosticoEstadoBlockedV0,
			I18nKey:      "operational.health.runtime.blocked",
			EvidenceRefs: []string{"event_20260504_000901"},
		}},
		Bloqueos: []orquestaobservability.DiagnosticoBloqueoV0{{
			BlockerRef:   "blocker_20260504_000901",
			Severity:     "warning",
			OwnerArea:    "core",
			Summary:      "Falta aprobacion de alcance",
			EvidenceRefs: []string{"event_20260504_000901", "event_20260504_000901"},
		}},
		ActividadReciente: []orquestaobservability.DiagnosticoActividadV0{{
			ActivityRef: "activity_20260504_000901",
			OccurredAt:  "2026-05-04T09:59:00Z",
			Area:        "core",
			Summary:     "Fase de orquestacion actualizada",
			EventRef:    "event_20260504_000902",
		}},
		Warnings: []orquestaobservability.DiagnosticoWarningV0{{
			Code:    "proyeccion_parcial",
			Section: "progreso",
			Summary: "Vista parcial",
		}},
		Privacy: orquestaobservability.DiagnosticoPrivacyV0{},
	}
}
