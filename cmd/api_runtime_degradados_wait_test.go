package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIRuntimeProcessDegradadosWaitEjecutaBatchEnPrimerPlano(t *testing.T) {
	prev := runtimeProcessDegradadosBatch
	runtimeProcessDegradadosBatch = func() (int, error) { return 2, nil }
	t.Cleanup(func() { runtimeProcessDegradadosBatch = prev })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-degradados", strings.NewReader(`{"wait":true}`))
	req.Header.Set("Content-Type", "application/json")

	apiHandlerRuntimeProcessDegradados(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiRuntimeProcessAutonomiaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !payload.OK || payload.Count != 2 {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.Accepted || payload.Running {
		t.Fatalf("no deberia marcar accepted/running en modo wait: %+v", payload)
	}
}
