package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIHandlerConfigReturns503WhenListHangs(t *testing.T) {
	prevFn := apiConfigListFn
	prevTimeout := apiConfigTimeout
	t.Cleanup(func() {
		apiConfigListFn = prevFn
		apiConfigTimeout = prevTimeout
	})

	apiConfigTimeout = 20 * time.Millisecond
	apiConfigListFn = func() (map[string]string, error) {
		time.Sleep(200 * time.Millisecond)
		return map[string]string{"workspace_root": "/tmp/workspace"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	apiHandlerConfig(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "config temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestAPIHandlerConfigReturns503WhenSetHangs(t *testing.T) {
	prevFn := apiConfigSetFn
	prevTimeout := apiConfigTimeout
	t.Cleanup(func() {
		apiConfigSetFn = prevFn
		apiConfigTimeout = prevTimeout
	})

	apiConfigTimeout = 20 * time.Millisecond
	apiConfigSetFn = func(clave, valor string) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader([]byte(`{"clave":"workspace_root","valor":"/srv/workspace"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	apiHandlerConfig(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "config temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}
