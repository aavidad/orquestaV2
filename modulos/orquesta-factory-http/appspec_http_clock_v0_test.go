package orquestafactoryhttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
)

func TestAppSpecHTTPV0RejectsZeroInjectedClock(t *testing.T) {
	handler := NewAppSpecHTTPHandlerV0(func() time.Time { return time.Time{} })
	req := httptest.NewRequest(http.MethodPost, AppSpecHTTPPathV0, bytes.NewReader(mustJSONV0(t, validMinimalRequestV0())))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertHTTPErrorV0(t, rec, http.StatusBadRequest, orquestafactory.ErrAppSpecInvalida)
	if !hasIssueFieldV0(mustHTTPErrorEnvelopeV0(t, rec).Errores, "received_at") {
		t.Fatalf("expected received_at issue: %s", rec.Body.String())
	}
}

func mustHTTPErrorEnvelopeV0(t *testing.T, rec *httptest.ResponseRecorder) AppSpecHTTPErrorResponseV0 {
	t.Helper()
	var envelope AppSpecHTTPErrorResponseV0
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	return envelope
}

func hasIssueFieldV0(issues []orquestafactory.ValidationIssue, field string) bool {
	for _, issue := range issues {
		if issue.Field == field {
			return true
		}
	}
	return false
}
