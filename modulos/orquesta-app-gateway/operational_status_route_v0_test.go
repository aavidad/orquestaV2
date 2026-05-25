package orquestaappgateway

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestOperationalStatusAPIRouteV0(t *testing.T) {
	source := &recordingOperationalStatusSourceV0{
		diagnostic: appGatewayOperationalDiagnosticV0("corr-ref-app-gateway-operational-status"),
	}
	handler := NewHTTPHandlerV0(ConfigV0{
		Timeout:           time.Second,
		OperationalStatus: source,
	})
	body := bytes.NewBufferString(`{
		"schema_version":"operational_status_query.v0",
		"request_id":"request-ref-app-gateway-operational-status",
		"correlation_id":"corr-ref-app-gateway-operational-status",
		"consumer":{"module":"orquesta-web","channel":"web"},
		"locale":"es",
		"scope":"sistema",
		"include_sections":["estado","salud"],
		"limit":5
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/operational-status/query", body)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if source.query.Consumer.Module != "orquesta-web" ||
		!strings.Contains(rec.Body.String(), "diagnostico_compacto.v0") {
		t.Fatalf("query=%+v body=%s", source.query, rec.Body.String())
	}
}

type recordingOperationalStatusSourceV0 struct {
	query      orquestaobservability.OperationalStatusQueryV0
	diagnostic orquestaobservability.DiagnosticoCompactoV0
}

func (source *recordingOperationalStatusSourceV0) QueryOperationalStatusV0(
	query orquestaobservability.OperationalStatusQueryV0,
) (orquestaobservability.DiagnosticoCompactoV0, error) {
	source.query = query
	return source.diagnostic, nil
}

func appGatewayOperationalDiagnosticV0(correlationID string) orquestaobservability.DiagnosticoCompactoV0 {
	return orquestaobservability.DiagnosticoCompactoV0{
		SchemaVersion: "diagnostico_compacto.v0",
		DiagnosticID:  "diagnostic-ref-app-gateway-operational-status",
		GeneratedAt:   "2026-05-25T10:00:00Z",
		CorrelationID: correlationID,
		Scope:         "sistema",
		ProjectionRef: "projection-ref-app-gateway-operational-status",
		Freshness: orquestaobservability.DiagnosticoFreshnessV0{
			WatermarkRef: "watermark-ref-app-gateway-operational-status",
		},
		Estado:  "ok",
		Privacy: orquestaobservability.DiagnosticoPrivacyV0{},
	}
}
