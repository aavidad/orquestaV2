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

	"orquesta/db"
)

func TestPropuestaActualizarCLIAnexaDescripcionYSoportaAliasAgente(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/propuestas/OP-920/accion", func(w http.ResponseWriter, r *http.Request) {
		var req apiPropuestaAccionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode accion propuesta: %v", err)
		}
		if req.Accion != "actualizar" || req.AnexarDesc == nil || *req.AnexarDesc != " + detalle operativo" || req.Agente != "Codex4" {
			t.Fatalf("payload inesperado: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := propuestaActualizarCmd.Flags().Set("agente", "Codex4"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	if err := propuestaActualizarCmd.Flags().Set("anexar-descripcion", " + detalle operativo"); err != nil {
		t.Fatalf("set anexar-descripcion: %v", err)
	}
	t.Cleanup(func() {
		_ = propuestaActualizarCmd.Flags().Set("agente", "")
		_ = propuestaActualizarCmd.Flags().Set("anexar-descripcion", "")
	})

	out := capturarStdout(t, func() {
		if err := propuestaActualizarCmd.RunE(propuestaActualizarCmd, []string{"OP-920"}); err != nil {
			t.Fatalf("propuesta actualizar: %v", err)
		}
	})
	if !strings.Contains(out, "Propuesta OP-920 actualizada") {
		t.Fatalf("salida inesperada:\n%s", out)
	}
}

func TestPropuestaVotosCLIFiltraPorAgenteSinImportarMayusculas(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/propuestas/OP-921", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPropuestaDetalleResponse{
			Propuesta: &db.Propuesta{
				Codigo: "OP-921",
				Titulo: "Filtro de votos",
				Estado: db.PropuestaAbierta,
				Votos: []*db.Voto{
					{Agente: "Codex2", Posicion: db.VotoAcuerdo, Comentario: "ok"},
					{Agente: "Claude1", Posicion: db.VotoDesacuerdo, Comentario: "no"},
				},
			},
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if err := propuestaVotosCmd.Flags().Set("agente", "codex2"); err != nil {
		t.Fatalf("set agente: %v", err)
	}
	t.Cleanup(func() {
		_ = propuestaVotosCmd.Flags().Set("agente", "")
	})

	out := capturarStdout(t, func() {
		if err := propuestaVotosCmd.RunE(propuestaVotosCmd, []string{"OP-921"}); err != nil {
			t.Fatalf("propuesta votos: %v", err)
		}
	})

	if !strings.Contains(out, "Codex2") {
		t.Fatalf("faltó el voto filtrado:\n%s", out)
	}
	if strings.Contains(out, "Claude1") {
		t.Fatalf("salieron votos de otros agentes:\n%s", out)
	}
}
