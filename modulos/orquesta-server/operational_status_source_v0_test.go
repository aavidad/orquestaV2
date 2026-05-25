package orquestaserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestResidentOperationalStatusSourceV0DerivaDiagnosticoCompactoRedactado(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{
		Addr:           "127.0.0.1:8787",
		ProjectWorkDir: "/tmp/proyecto-secreto",
		RuntimeWorkDir: "/tmp/runtime-secreto",
	}, now)
	tracker.MarkServingV0("127.0.0.1:8787", now)
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status:       StartupCheckStatusReadyV0,
		Ready:        true,
		Message:      "ready with /tmp/runtime-secreto",
		EvidenceRefs: []string{"evidence-ref-startup-ready"},
	}, now)
	query := residentOperationalStatusQueryV0("corr-ref-resident-operational-status")

	diagnostic, err := (ResidentOperationalStatusSourceV0{
		Tracker: tracker,
		Clock:   fixedServerClockV0{now: now},
	}).QueryOperationalStatusV0(query)

	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic invalid: %v", err)
	}
	body, _ := json.Marshal(diagnostic)
	for _, forbidden := range []string{"proyecto-secreto", "runtime-secreto", "project_work_dir", "runtime_work_dir"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("diagnostic leaks %q: %s", forbidden, string(body))
		}
	}
	if diagnostic.Estado != orquestaobservability.DiagnosticoEstadoOKV0 || len(diagnostic.Warnings) == 0 {
		t.Fatalf("diagnostic=%+v", diagnostic)
	}
}

func TestHandlerV0OperationalStatusQueryV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:8787"}, now)
	tracker.MarkServingV0("127.0.0.1:8787", now)
	handler := NewHandlerV0(HandlerConfigV0{
		Tracker: tracker,
		OperationalStatusSource: ResidentOperationalStatusSourceV0{
			Tracker: tracker,
			Clock:   fixedServerClockV0{now: now},
		},
	})
	payload, _ := json.Marshal(residentOperationalStatusQueryV0("corr-ref-handler-operational-status"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/operational-status/query", bytes.NewReader(payload))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "diagnostico_compacto.v0") ||
		!strings.Contains(rec.Body.String(), "runtime_not_available") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func residentOperationalStatusQueryV0(correlationID string) orquestaobservability.OperationalStatusQueryV0 {
	return orquestaobservability.OperationalStatusQueryV0{
		SchemaVersion: orquestaobservability.OperationalStatusQuerySchemaVersionV0,
		RequestID:     "request-ref-resident-operational-status",
		CorrelationID: correlationID,
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:          "es",
		Scope:           orquestaobservability.OperationalStatusScopeSistemaV0,
		IncludeSections: []string{"estado", "progreso", "salud", "bloqueos", "actividad_reciente"},
		Limit:           10,
	}
}

type fixedServerClockV0 struct {
	now time.Time
}

func (clock fixedServerClockV0) Now() time.Time {
	return clock.now
}
