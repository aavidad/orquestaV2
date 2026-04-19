package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestAPIHandlerRuntimeProcessMailboxReturns503WhenBatchHangs(t *testing.T) {
	prevTimeout := apiRuntimeProcessMailboxTimeout
	prevExecutor := apiRuntimeProcessMailboxExecutor
	t.Cleanup(func() {
		apiRuntimeProcessMailboxTimeout = prevTimeout
		apiRuntimeProcessMailboxExecutor = prevExecutor
	})

	apiRuntimeProcessMailboxTimeout = 20 * time.Millisecond
	apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
		time.Sleep(200 * time.Millisecond)
		return 0, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/runtime/process-mailbox", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	apiHandlerRuntimeProcessMailbox(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "runtime mailbox temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}
