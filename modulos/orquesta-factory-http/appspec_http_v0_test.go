package orquestafactoryhttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestAppSpecHTTPV0PostValidoDevuelveEnvelopeCanonico(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(fixedHTTPClockV0())
	body := mustJSONV0(t, validMinimalRequestV0())
	req := httptest.NewRequest(http.MethodPost, AppSpecHTTPPathV0, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Correlation-ID", "corr-fty-007")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Correlation-ID"); got != "corr-fty-007" {
		t.Fatalf("correlation=%q", got)
	}

	var envelope AppSpecHTTPResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.AppSpec.SchemaVersion != orquestafactory.AppSpecSchemaV0 {
		t.Fatalf("app_spec schema=%q", envelope.AppSpec.SchemaVersion)
	}
	if envelope.AppSpec.CreatedAt != "2026-05-04T14:00:00Z" {
		t.Fatalf("created_at=%q", envelope.AppSpec.CreatedAt)
	}
	if envelope.Backlog.SchemaVersion != orquestafactory.BacklogInicialPropuestoSchemaV0 {
		t.Fatalf("backlog schema=%q", envelope.Backlog.SchemaVersion)
	}
	if envelope.Backlog.SpecID != envelope.AppSpec.SpecID {
		t.Fatalf("backlog spec_id=%q, app_spec spec_id=%q", envelope.Backlog.SpecID, envelope.AppSpec.SpecID)
	}
	if envelope.AppSpec.Data.PersistenceRequired || len(envelope.AppSpec.Connectors.Required) != 0 {
		t.Fatalf("request minima should not require DB/runtime connectors: data=%+v connectors=%+v", envelope.AppSpec.Data, envelope.AppSpec.Connectors)
	}
}

func TestAppSpecHTTPV0UsaRequestIDComoCorrelationIDSiFaltaHeader(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(fixedHTTPClockV0())
	body := mustJSONV0(t, validMinimalRequestV0())
	req := httptest.NewRequest(http.MethodPost, AppSpecHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Correlation-ID"); got != "req-fty-003-minima" {
		t.Fatalf("correlation=%q", got)
	}
}

func TestAppSpecHTTPV0JSONDesconocidoDevuelve400ErroresPublicos(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(fixedHTTPClockV0())
	body := []byte(`{"schema_version":"app_spec_request.v0","request_id":"req-fty-007-invalid","source":"orquesta-web","locale":"es-ES","nombre":"Panel","objetivo":"Gestionar reservas.","tipo_app":"web","desconocido":true}`)
	req := httptest.NewRequest(http.MethodPost, AppSpecHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHTTPErrorV0(t, rec, http.StatusBadRequest, orquestafactory.ErrAppSpecInvalida)
	if strings.Contains(rec.Body.String(), string(body)) {
		t.Fatalf("error response should not echo private body: %s", rec.Body.String())
	}
}

func TestAppSpecHTTPV0RequestInvalidaDevuelve400ErroresPublicos(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(fixedHTTPClockV0())
	reqBody := validMinimalRequestV0()
	reqBody.Nombre = ""
	body := mustJSONV0(t, reqBody)
	req := httptest.NewRequest(http.MethodPost, AppSpecHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHTTPErrorV0(t, rec, http.StatusBadRequest, orquestafactory.ErrAppSpecInvalida)
	if got := rec.Header().Get("X-Correlation-ID"); got != reqBody.RequestID {
		t.Fatalf("correlation=%q", got)
	}
}

func TestAppSpecHTTPV0MetodoNoPermitido(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(fixedHTTPClockV0())
	req := httptest.NewRequest(http.MethodGet, AppSpecHTTPPathV0, nil)
	req.Header.Set("X-Correlation-ID", "corr-method")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHTTPErrorV0(t, rec, http.StatusMethodNotAllowed, orquestafactory.ErrAppSpecInvalida)
	if got := rec.Header().Get("Allow"); got != http.MethodPost {
		t.Fatalf("allow=%q", got)
	}
	if got := rec.Header().Get("X-Correlation-ID"); got != "corr-method" {
		t.Fatalf("correlation=%q", got)
	}
}

func TestAppSpecHTTPV0BacklogIncluyeFasesMicrotareasYSchema(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(fixedHTTPClockV0())
	body := mustJSONV0(t, validMinimalRequestV0())
	req := httptest.NewRequest(http.MethodPost, AppSpecHTTPPathV0, bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var envelope AppSpecHTTPResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Backlog.Fases) == 0 {
		t.Fatalf("expected fases: %+v", envelope.Backlog)
	}
	if len(envelope.Backlog.Microtareas) == 0 {
		t.Fatalf("expected microtareas: %+v", envelope.Backlog)
	}
	for _, task := range envelope.Backlog.Microtareas {
		assertCompleteMicrotaskV0(t, task)
	}
}

func fixedHTTPClockV0() AppSpecHTTPClockV0 {
	return func() time.Time {
		return time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
	}
}

func validMinimalRequestV0() orquestafactory.AppSpecRequestV0 {
	return orquestafactory.AppSpecRequestV0{
		SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
		RequestID:     "req-fty-003-minima",
		Source:        "orquesta-web",
		Locale:        "es-ES",
		Nombre:        "Panel de reservas",
		Objetivo:      "Gestionar solicitudes de reserva.",
		TipoApp:       "web",
	}
}

func mustJSONV0(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func assertHTTPErrorV0(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status=%d, want %d, body=%s", rec.Code, status, rec.Body.String())
	}
	var envelope AppSpecHTTPErrorResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if len(envelope.Errores) == 0 {
		t.Fatalf("expected errores: %s", rec.Body.String())
	}
	if !hasIssueCodeV0(envelope.Errores, code) {
		t.Fatalf("missing error code %q in %+v", code, envelope.Errores)
	}
}

func assertCompleteMicrotaskV0(t *testing.T, task orquestafactory.MicrotareaPropuestaV0) {
	t.Helper()
	required := map[string]string{
		"fase":            task.Fase,
		"modulo_sugerido": task.ModuloSugerido,
		"objetivo":        task.Objetivo,
		"contrato":        task.Contrato,
		"validacion":      task.Validacion,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("task %s missing %s: %+v", task.ID, field, task)
		}
	}
	if task.WriteSetPrevisto == nil || len(task.WriteSetPrevisto) == 0 {
		t.Fatalf("task %s missing write-set: %+v", task.ID, task)
	}
	if task.Bloqueos == nil {
		t.Fatalf("task %s should serialize bloqueos as array: %+v", task.ID, task)
	}
}

func hasIssueCodeV0(issues []orquestafactory.ValidationIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
