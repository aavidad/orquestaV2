package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestGetStatusBodyV0UsaRutaVersionada(t *testing.T) {
	var gotPath string
	server := newLocalHTTPServerForTestV0(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			SchemaVersion: orquestaserver.StateSchemaVersionV0,
			Status:        "running",
			Addr:          "127.0.0.1:8787",
		}))
	}))
	defer server.Close()

	body, err := getStatusBodyV0(strings.TrimPrefix(server.URL, "http://"))

	if err != nil {
		t.Fatalf("getStatusBodyV0: %v", err)
	}
	if gotPath != orquestaserver.ServerStatusEndpointV0 {
		t.Fatalf("path=%q", gotPath)
	}
	if !strings.Contains(string(body), `"status":"running"`) {
		t.Fatalf("body=%s", string(body))
	}
}
