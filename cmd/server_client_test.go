/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCommandSupportsServerMode(t *testing.T) {
	casos := []struct {
		nombre string
		args   []string
		want   bool
	}{
		{nombre: "proyecto listar", args: []string{"proyecto", "listar"}, want: true},
		{nombre: "proyecto ver", args: []string{"proyecto", "ver", "orquestador"}, want: true},
		{nombre: "proyecto descubrir local", args: []string{"proyecto", "descubrir", "."}, want: false},
		{nombre: "conector listar", args: []string{"conector", "listar"}, want: true},
		{nombre: "conector ver", args: []string{"conector", "ver", "codex-cli"}, want: true},
		{nombre: "conector registrar", args: []string{"conector", "registrar"}, want: true},
		{nombre: "config ver", args: []string{"config", "ver"}, want: true},
		{nombre: "config set", args: []string{"config", "set", "clave", "valor"}, want: true},
		{nombre: "asignacion listar", args: []string{"asignacion", "listar"}, want: true},
		{nombre: "asignacion activar", args: []string{"asignacion", "activar", "Codex1", "orquestador"}, want: true},
		{nombre: "lock listar", args: []string{"lock", "listar"}, want: true},
		{nombre: "lock tomar", args: []string{"lock", "tomar", "Codex1", "file", "x"}, want: true},
		{nombre: "lock renovar", args: []string{"lock", "renovar", "1", "Codex1", "lease"}, want: true},
		{nombre: "lock liberar", args: []string{"lock", "liberar", "1", "Codex1", "lease"}, want: true},
		{nombre: "worktree listar", args: []string{"worktree", "listar"}, want: true},
		{nombre: "worktree crear", args: []string{"worktree", "crear", "Codex1", "orquestador"}, want: true},
		{nombre: "worktree cerrar", args: []string{"worktree", "cerrar", "1"}, want: true},
		{nombre: "runtime listar", args: []string{"runtime", "listar"}, want: true},
		{nombre: "runtime ver", args: []string{"runtime", "ver", "12"}, want: true},
		{nombre: "sesion inicio", args: []string{"sesion", "inicio", "Codex1"}, want: true},
		{nombre: "sesion historial", args: []string{"sesion", "historial"}, want: true},
		{nombre: "sesion ver", args: []string{"sesion", "ver", "12"}, want: true},
		{nombre: "sesion nuevo-codex", args: []string{"sesion", "nuevo-codex"}, want: true},
		{nombre: "exportar diagnostico", args: []string{"exportar", "diagnostico"}, want: true},
		{nombre: "runtime no soportado", args: []string{"runtime", "foo"}, want: false},
		{nombre: "serve", args: []string{"serve"}, want: false},
	}

	for _, tc := range casos {
		tc := tc
		t.Run(tc.nombre, func(t *testing.T) {
			if got := commandSupportsServerMode(tc.args); got != tc.want {
				t.Fatalf("commandSupportsServerMode(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestShouldPreferServerForCurrentCommandUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/runtimes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"runtimes": []any{}})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	restaurarURL := cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)
	defer restaurarURL()

	restaurarForceLocal := cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")
	defer restaurarForceLocal()

	restaurarArgs := cambiarArgs(t, []string{"orquesta", "runtime", "listar"})
	defer restaurarArgs()

	if !shouldPreferServerForCurrentCommand() {
		t.Fatalf("se esperaba preferencia por el servidor local")
	}
}

func newTestHTTPServerOrSkip(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(strings.ToLower(msg), "operation not permitted") {
				t.Skipf("sandbox sin permisos para listeners locales: %s", msg)
			}
			panic(r)
		}
	}()
	return httptest.NewServer(handler)
}

func cambiarEnv(t *testing.T, key, value string) func() {
	t.Helper()
	anterior, ok := os.LookupEnv(key)
	if value == "" {
		_ = os.Unsetenv(key)
	} else {
		if err := os.Setenv(key, value); err != nil {
			t.Fatalf("setenv %s: %v", key, err)
		}
	}
	return func() {
		t.Helper()
		if ok {
			_ = os.Setenv(key, anterior)
			return
		}
		_ = os.Unsetenv(key)
	}
}

func cambiarArgs(t *testing.T, args []string) func() {
	t.Helper()
	prev := os.Args
	os.Args = append([]string(nil), args...)
	return func() {
		t.Helper()
		os.Args = prev
	}
}
