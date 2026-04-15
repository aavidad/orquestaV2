package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIRuntimeProcessDegradadosWaitPasaAFondoSiExcedeTimeout(t *testing.T) {
	prevBatch := runtimeProcessDegradadosBatchDetailed
	prevTimeout := runtimeProcessDegradadosWaitTimeout
	done := make(chan struct{})
	runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
		time.Sleep(20 * time.Millisecond)
		close(done)
		return runtimeProcessDegradadosSummary{Count: 3}, nil
	}
	runtimeProcessDegradadosWaitTimeout = 2 * time.Millisecond
	t.Cleanup(func() {
		runtimeProcessDegradadosBatchDetailed = prevBatch
		runtimeProcessDegradadosWaitTimeout = prevTimeout
		runtimeProcessDegradadosEnCurso.Store(false)
	})

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
	if !payload.OK || payload.Count != 0 || !payload.Accepted || !payload.Running {
		t.Fatalf("payload inesperado tras timeout: %+v", payload)
	}

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("el batch diferido no terminó")
	}

	deadline := time.Now().Add(200 * time.Millisecond)
	for runtimeProcessDegradadosEnCurso.Load() && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if runtimeProcessDegradadosEnCurso.Load() {
		t.Fatal("la marca en curso deberia limpiarse tras completar el batch diferido")
	}
}

func TestAPIRuntimeProcessDegradadosWaitRespetaBatchEnCurso(t *testing.T) {
	runtimeProcessDegradadosEnCurso.Store(true)
	t.Cleanup(func() { runtimeProcessDegradadosEnCurso.Store(false) })

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
	if !payload.OK || payload.Count != 0 || payload.Accepted || !payload.Running {
		t.Fatalf("payload inesperado con batch ya en curso: %+v", payload)
	}
}
