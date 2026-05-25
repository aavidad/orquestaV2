package orquestamcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestMCPOperationalStatusHTTPHandlerV0UsaSourceCompartido(t *testing.T) {
	source := &fakeOperationalStatusSourceV0{
		diagnostic: validMCPOperationalDiagnosticV0("corr-ref-operational-status-http"),
	}
	body := bytes.NewBufferString(`{
		"schema_version":"operational_status_query.v0",
		"request_id":"request-ref-operational-status-http",
		"correlation_id":"corr-ref-operational-status-http",
		"consumer":{"module":"orquesta-mcp","channel":"mcp"},
		"locale":"es",
		"scope":"sistema",
		"include_sections":["estado","salud"],
		"limit":5
	}`)
	req := httptest.NewRequest(http.MethodPost, MCPOperationalStatusHTTPPathV0, body)
	rec := httptest.NewRecorder()

	NewMCPOperationalStatusHTTPHandlerV0(source).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if source.query.Consumer.Module != "orquesta-mcp" ||
		rec.Header().Get("X-Correlation-ID") != "corr-ref-operational-status-http" {
		t.Fatalf("query=%+v headers=%v", source.query, rec.Header())
	}
}

func TestMCPOperationalStatusHTTPHandlerV0SourceNoDisponible(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, MCPOperationalStatusHTTPPathV0, bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()

	NewMCPOperationalStatusHTTPHandlerV0(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeOperationalStatusSourceV0 struct {
	query      orquestaobservability.OperationalStatusQueryV0
	diagnostic orquestaobservability.DiagnosticoCompactoV0
	err        error
}

func (source *fakeOperationalStatusSourceV0) QueryOperationalStatusV0(
	query orquestaobservability.OperationalStatusQueryV0,
) (orquestaobservability.DiagnosticoCompactoV0, error) {
	source.query = query
	if source.err != nil {
		return orquestaobservability.DiagnosticoCompactoV0{}, source.err
	}
	return source.diagnostic, nil
}

func validMCPOperationalDiagnosticV0(correlationID string) orquestaobservability.DiagnosticoCompactoV0 {
	percent := 100.0
	diagnostic := orquestaobservability.DiagnosticoCompactoV0{
		SchemaVersion: "diagnostico_compacto.v0",
		DiagnosticID:  "diagnostic-ref-mcp-operational-status",
		GeneratedAt:   "2026-05-25T10:00:00Z",
		CorrelationID: correlationID,
		Scope:         "sistema",
		ProjectionRef: "projection-ref-mcp-operational-status",
		Freshness: orquestaobservability.DiagnosticoFreshnessV0{
			WatermarkRef: "watermark-ref-mcp-operational-status",
		},
		Estado: "ok",
		Progreso: orquestaobservability.DiagnosticoProgresoV0{
			Completed: 1,
			Total:     1,
			Percent:   &percent,
			Phase:     "running",
			Summary:   "diagnostico compacto",
		},
		Privacy: orquestaobservability.DiagnosticoPrivacyV0{},
	}
	data, _ := json.Marshal(diagnostic)
	var clone orquestaobservability.DiagnosticoCompactoV0
	_ = json.Unmarshal(data, &clone)
	return clone
}
