package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIRuntimeProcessAutonomiaWaitEjecutaBatchEnPrimerPlano(t *testing.T) {
	prev := runtimeProcessAutonomiaBatch
	runtimeProcessAutonomiaBatch = func() (int, error) { return 3, nil }
	t.Cleanup(func() { runtimeProcessAutonomiaBatch = prev })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-autonomia", strings.NewReader(`{"wait":true}`))
	req.Header.Set("Content-Type", "application/json")

	apiHandlerRuntimeProcessAutonomia(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload apiRuntimeProcessAutonomiaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !payload.OK || payload.Count != 3 {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.Accepted || payload.Running {
		t.Fatalf("no deberia marcar accepted/running en modo wait: %+v", payload)
	}
}
