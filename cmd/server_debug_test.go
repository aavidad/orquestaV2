package cmd

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveServerDebugOptionsUsaFlagBase(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("debug", false, "")
	cmd.Flags().Bool("debug-http", false, "")
	cmd.Flags().Bool("debug-control-plane", false, "")
	if err := cmd.Flags().Set("debug", "true"); err != nil {
		t.Fatalf("set debug: %v", err)
	}

	opts := resolveServerDebugOptions(cmd)
	if !opts.Enabled || !opts.HTTP || !opts.ControlPlane {
		t.Fatalf("opts debug inesperadas: %+v", opts)
	}
}

func TestWrapServeMuxWithDebugRegistraRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	handler := wrapServeMuxWithDebug(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}), serverDebugOptions{Enabled: true, HTTP: true}, logger)

	req := httptest.NewRequest(http.MethodGet, "/debug?x=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status inesperado: %d", rec.Code)
	}
	logLine := buf.String()
	for _, token := range []string{"method=GET", "path=/debug", `query="x=1"`, "status=201"} {
		if !strings.Contains(logLine, token) {
			t.Fatalf("log sin %q: %s", token, logLine)
		}
	}
}
