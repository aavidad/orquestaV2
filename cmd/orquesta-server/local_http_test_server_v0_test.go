package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newLocalHTTPServerForTestV0(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	requireLocalTCPForTestV0(t)
	return httptest.NewServer(handler)
}

func requireLocalTCPForTestV0(t *testing.T) {
	t.Helper()
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP no disponible para httptest: %v", err)
	}
	if err := probe.Close(); err != nil {
		t.Fatalf("cerrar probe TCP: %v", err)
	}
}
