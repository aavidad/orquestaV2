package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIRuntimeProcessDegradadosWaitDevuelveResumenDetallado(t *testing.T) {
	prev := runtimeProcessDegradadosBatchDetailed
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		return runtimeProcessDegradadosSummary{
			Count:                     4,
			GhostAssignmentsCompacted: 1,
			ReactivatedWithoutRuntime: 2,
			IdleAutoassigned:          1,
		}, nil
	}
	t.Cleanup(func() { runtimeProcessDegradadosBatchDetailed = prev })

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
	if !payload.OK || payload.Count != 4 {
		t.Fatalf("payload inesperado: %+v", payload)
	}
	if payload.GhostAssignmentsCompacted != 1 || payload.ReactivatedWithoutRuntime != 2 || payload.IdleAutoassigned != 1 {
		t.Fatalf("resumen inesperado: %+v", payload)
	}
	if payload.Accepted || payload.Running {
		t.Fatalf("no deberia marcar accepted/running en modo wait: %+v", payload)
	}
}
