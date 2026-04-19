package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIHandlerRuntimeProcessOrdersReturns503WhenBatchHangs(t *testing.T) {
	prevTimeout := apiRuntimeProcessOrdersTimeout
	prevExecutor := apiRuntimeProcessOrdersExecutor
	t.Cleanup(func() {
		apiRuntimeProcessOrdersTimeout = prevTimeout
		apiRuntimeProcessOrdersExecutor = prevExecutor
	})

	apiRuntimeProcessOrdersTimeout = 20 * time.Millisecond
	apiRuntimeProcessOrdersExecutor = func() (int, error) {
		time.Sleep(200 * time.Millisecond)
		return 0, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-orders", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessOrders(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "runtime orders temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}
