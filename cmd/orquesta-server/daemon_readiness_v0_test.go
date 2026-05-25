package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestServerReadinessOKV0UsaReadinessNoHealthzV0(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case orquestaserver.ServerReadinessEndpointV0:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	if serverReadinessOKV0(strings.TrimPrefix(server.URL, "http://")) {
		t.Fatalf("readiness false no debe pasar aunque healthz este vivo")
	}
}
