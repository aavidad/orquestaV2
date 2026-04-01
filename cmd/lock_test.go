/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"orquesta/coordinacion"
)

func TestLockUsaAPI(t *testing.T) {
	lock := &coordinacion.Lock{
		ID:         52,
		Agent:      "Codex1",
		ScopeType:  "worktree",
		ScopeKey:   "orq-codex1",
		State:      coordinacion.LockActive,
		LeaseToken: "lease-123",
		ExpiresAt:  time.Date(2026, 3, 24, 20, 0, 0, 0, time.UTC),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/locks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"locks": []*coordinacion.Lock{lock}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"lock": lock})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/locks/52/renovar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		renovado := *lock
		renovado.ExpiresAt = time.Date(2026, 3, 24, 21, 0, 0, 0, time.UTC)
		_ = json.NewEncoder(w).Encode(map[string]any{"lock": &renovado})
	})
	mux.HandleFunc("/api/locks/52/liberar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		liberado := *lock
		liberado.State = coordinacion.LockReleased
		_ = json.NewEncoder(w).Encode(map[string]any{"lock": &liberado})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(lockListarCmd)
	_ = lockListarCmd.Flags().Set("agente", "Codex1")
	_ = lockListarCmd.Flags().Set("proyecto", "orquestador")
	_ = lockListarCmd.Flags().Set("estado", "activa")
	outListar := capturarStdout(t, func() {
		if err := lockListarCmd.RunE(lockListarCmd, nil); err != nil {
			t.Fatalf("lock listar via api: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "orq-codex1"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida listar sin %q:\n%s", token, outListar)
		}
	}

	resetCommandFlags(lockTomarCmd)
	_ = lockTomarCmd.Flags().Set("proyecto", "orquestador")
	_ = lockTomarCmd.Flags().Set("motivo", "prueba")
	outTomar := capturarStdout(t, func() {
		if err := lockTomarCmd.RunE(lockTomarCmd, []string{"Codex1", "worktree", "orq-codex1"}); err != nil {
			t.Fatalf("lock tomar via api: %v", err)
		}
	})
	if !strings.Contains(outTomar, "Lock 52 tomado") {
		t.Fatalf("salida tomar inesperada:\n%s", outTomar)
	}

	resetCommandFlags(lockRenovarCmd)
	outRenovar := capturarStdout(t, func() {
		if err := lockRenovarCmd.RunE(lockRenovarCmd, []string{"52", "Codex1", "lease-123"}); err != nil {
			t.Fatalf("lock renovar via api: %v", err)
		}
	})
	if !strings.Contains(outRenovar, "Lock 52 renovado") {
		t.Fatalf("salida renovar inesperada:\n%s", outRenovar)
	}

	resetCommandFlags(lockLiberarCmd)
	outLiberar := capturarStdout(t, func() {
		if err := lockLiberarCmd.RunE(lockLiberarCmd, []string{"52", "Codex1", "lease-123"}); err != nil {
			t.Fatalf("lock liberar via api: %v", err)
		}
	})
	if !strings.Contains(outLiberar, "Lock 52 liberado") {
		t.Fatalf("salida liberar inesperada:\n%s", outLiberar)
	}
}

func TestLockListarFiltraActivosPorDefecto(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/locks", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("estado"); got != string(coordinacion.LockActive) {
			t.Fatalf("estado por defecto inesperado: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"locks": []*coordinacion.Lock{}})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(lockListarCmd)
	if err := lockListarCmd.RunE(lockListarCmd, nil); err != nil {
		t.Fatalf("lock listar via api: %v", err)
	}
}

func TestLockRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(lockListarCmd)
	err := lockListarCmd.RunE(lockListarCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}
