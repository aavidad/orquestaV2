package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newLocalHTTPServerForTestV0(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener := newLocalTCPListenerForTestV0(t)
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

func requireLocalTCPForTestV0(t *testing.T) {
	t.Helper()
	listener := newLocalTCPListenerForTestV0(t)
	if err := listener.Close(); err != nil {
		t.Fatalf("cerrar probe TCP: %v", err)
	}
}

func newLocalTCPListenerForTestV0(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local TCP no disponible para httptest: %v", err)
	}
	return listener
}
