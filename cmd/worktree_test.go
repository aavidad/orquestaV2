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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func resetCommandFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		_ = cmd.Flags().Set(f.Name, f.DefValue)
		f.Changed = false
	})
}

func TestWorktreeUsaAPI(t *testing.T) {
	worktree := &coordinacion.Worktree{
		ID:        41,
		ProjectID: 7,
		Agent:     "Codex1",
		Branch:    "orq-codex1",
		Path:      "/tmp/orquestador/.orquesta-worktrees/orq-codex1",
		State:     coordinacion.WorktreeActive,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/worktrees", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"worktrees": []*coordinacion.Worktree{worktree}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"worktree": worktree})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/worktrees/41/cerrar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		cerrado := *worktree
		cerrado.State = coordinacion.WorktreeClosed
		cerrado.ClosedAt = ptrTime(time.Date(2026, 3, 24, 19, 0, 0, 0, time.UTC))
		_ = json.NewEncoder(w).Encode(map[string]any{"worktree": &cerrado})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(worktreeListarCmd)
	_ = worktreeListarCmd.Flags().Set("agente", "Codex1")
	_ = worktreeListarCmd.Flags().Set("proyecto", "orquestador")
	_ = worktreeListarCmd.Flags().Set("estado", "activa")
	outListar := capturarStdout(t, func() {
		if err := worktreeListarCmd.RunE(worktreeListarCmd, nil); err != nil {
			t.Fatalf("worktree listar via api: %v", err)
		}
	})
	for _, token := range []string{"ID", "Codex1", "orq-codex1"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida listar sin %q:\n%s", token, outListar)
		}
	}

	resetCommandFlags(worktreeResolverCmd)
	outResolver := capturarStdout(t, func() {
		if err := worktreeResolverCmd.RunE(worktreeResolverCmd, []string{"Codex1", "orquestador"}); err != nil {
			t.Fatalf("worktree resolver via api: %v", err)
		}
	})
	if !strings.Contains(outResolver, worktree.Path) {
		t.Fatalf("salida resolver inesperada:\n%s", outResolver)
	}

	resetCommandFlags(worktreeCrearCmd)
	_ = worktreeCrearCmd.Flags().Set("branch", "orq-codex1")
	outCrear := capturarStdout(t, func() {
		if err := worktreeCrearCmd.RunE(worktreeCrearCmd, []string{"Codex1", "orquestador"}); err != nil {
			t.Fatalf("worktree crear via api: %v", err)
		}
	})
	if !strings.Contains(outCrear, "Worktree 41 creado") {
		t.Fatalf("salida crear inesperada:\n%s", outCrear)
	}

	resetCommandFlags(worktreeCerrarCmd)
	outCerrar := capturarStdout(t, func() {
		if err := worktreeCerrarCmd.RunE(worktreeCerrarCmd, []string{"41"}); err != nil {
			t.Fatalf("worktree cerrar via api: %v", err)
		}
	})
	if !strings.Contains(outCerrar, "Worktree 41 cerrado") {
		t.Fatalf("salida cerrar inesperada:\n%s", outCerrar)
	}
}

func TestWorktreeRequiereServidor(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	resetCommandFlags(worktreeListarCmd)
	err := worktreeListarCmd.RunE(worktreeListarCmd, nil)
	if err == nil {
		t.Fatalf("se esperaba error sin servidor")
	}
	if !strings.Contains(err.Error(), "requiere el servidor de Orquesta activo") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
