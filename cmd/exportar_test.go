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

	"orquesta/db"
)

func TestExportarEstadoUsaSoloAPIParaVotos(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/agentes", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiAgentesResponse{
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true},
			},
		})
	})
	mux.HandleFunc("/api/propuestas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPropuestasResponse{
			Propuestas: []*db.Propuesta{
				{
					ID:           1,
					Codigo:       "OP-200",
					Titulo:       "Exportar por API",
					Tipo:         "implementacion",
					Estado:       db.PropuestaAbierta,
					PropuestoPor: "Codex1",
					CreatedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				},
			},
		})
	})
	mux.HandleFunc("/api/propuestas/OP-200", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiPropuestaDetalleResponse{
			Propuesta: &db.Propuesta{
				ID:           1,
				Codigo:       "OP-200",
				Titulo:       "Exportar por API",
				Tipo:         "implementacion",
				Estado:       db.PropuestaAbierta,
				PropuestoPor: "Codex1",
				CreatedAt:    time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC),
				Votos: []*db.Voto{
					{Agente: "Codex2", Posicion: db.VotoAcuerdo, Comentario: "ok"},
				},
			},
			ConteoVotos: apiConteoVotosResponse{Acuerdo: 1},
		})
	})
	mux.HandleFunc("/api/tareas", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiTareasResponse{
			Tareas: []*db.Tarea{
				{ID: 7, Titulo: "Tarea API", Estado: db.EstadoEnProgreso, Prioridad: db.PrioridadAlta, Modulo: "core"},
			},
		})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	out := capturarStdout(t, func() {
		if err := exportarEstadoCmd.RunE(exportarEstadoCmd, nil); err != nil {
			t.Fatalf("exportar estado via API: %v", err)
		}
	})

	for _, token := range []string{"# Orquesta", "OP-200", "Codex2", "Tarea API"} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida exportar estado sin %q:\n%s", token, out)
		}
	}
}
