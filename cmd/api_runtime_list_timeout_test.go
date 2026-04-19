package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestAPIHandlerRuntimeOrdersReturns503WhenListHangs(t *testing.T) {
	prevFn := apiListRuntimeOrdersFn
	prevTimeout := apiListRuntimeOrdersTimeout
	t.Cleanup(func() {
		apiListRuntimeOrdersFn = prevFn
		apiListRuntimeOrdersTimeout = prevTimeout
	})

	apiListRuntimeOrdersTimeout = 20 * time.Millisecond
	apiListRuntimeOrdersFn = func(filter db.FiltroRuntimeOrders) ([]*db.RuntimeOrder, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runtime-orders", nil)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeOrders(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "runtime orders temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestAPIHandlerRuntimeMailboxReturns503WhenListHangs(t *testing.T) {
	prevFn := apiListRuntimeMailboxFn
	prevTimeout := apiListRuntimeMailboxTimeout
	t.Cleanup(func() {
		apiListRuntimeMailboxFn = prevFn
		apiListRuntimeMailboxTimeout = prevTimeout
	})

	apiListRuntimeMailboxTimeout = 20 * time.Millisecond
	apiListRuntimeMailboxFn = func(filter db.FiltroRuntimeMailbox) ([]*db.RuntimeMailboxMessage, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runtime-mailbox", nil)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeMailbox(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "runtime mailbox temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestAPIHandlerRuntimeOrdersReturns503WhenProjectLookupHangs(t *testing.T) {
	prevFn := apiGetRuntimeProjectFn
	prevTimeout := apiRuntimeProjectLookupTimeout
	t.Cleanup(func() {
		apiGetRuntimeProjectFn = prevFn
		apiRuntimeProjectLookupTimeout = prevTimeout
	})

	apiRuntimeProjectLookupTimeout = 20 * time.Millisecond
	apiGetRuntimeProjectFn = func(ref string) (*db.Proyecto, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runtime-orders?proyecto=orquestador", nil)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeOrders(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "runtime orders temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}

func TestAPIHandlerRuntimeMailboxReturns503WhenProjectLookupHangs(t *testing.T) {
	prevFn := apiGetRuntimeProjectFn
	prevTimeout := apiRuntimeProjectLookupTimeout
	t.Cleanup(func() {
		apiGetRuntimeProjectFn = prevFn
		apiRuntimeProjectLookupTimeout = prevTimeout
	})

	apiRuntimeProjectLookupTimeout = 20 * time.Millisecond
	apiGetRuntimeProjectFn = func(ref string) (*db.Proyecto, error) {
		time.Sleep(200 * time.Millisecond)
		return nil, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/runtime-mailbox?proyecto=orquestador", nil)
	rec := httptest.NewRecorder()
	apiHandlerRuntimeMailbox(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status inesperado=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "runtime mailbox temporalmente degradado") {
		t.Fatalf("body inesperado: %s", rec.Body.String())
	}
}
