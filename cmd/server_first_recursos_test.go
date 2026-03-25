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

func TestConfigUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if clave := r.URL.Query().Get("clave"); clave != "" {
				_ = json.NewEncoder(w).Encode(apiConfigResponse{Clave: clave, Valor: "/tmp/workspace"})
				return
			}
			_ = json.NewEncoder(w).Encode(apiConfigResponse{Config: map[string]string{"workspace_root": "/tmp/workspace", "version": "1.0.0"}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/agentes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/agentes/Codex9/retirar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/agentes/Codex9/rehabilitar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	outVerClave := capturarStdout(t, func() {
		if err := configVerCmd.RunE(configVerCmd, []string{"workspace_root"}); err != nil {
			t.Fatalf("config ver clave via api: %v", err)
		}
	})
	if !strings.Contains(outVerClave, "workspace_root = /tmp/workspace") {
		t.Fatalf("salida config ver clave inesperada:\n%s", outVerClave)
	}

	outVerTodo := capturarStdout(t, func() {
		if err := configVerCmd.RunE(configVerCmd, nil); err != nil {
			t.Fatalf("config ver via api: %v", err)
		}
	})
	for _, token := range []string{"workspace_root", "version"} {
		if !strings.Contains(outVerTodo, token) {
			t.Fatalf("salida config ver sin %q:\n%s", token, outVerTodo)
		}
	}

	outSet := capturarStdout(t, func() {
		if err := configSetCmd.RunE(configSetCmd, []string{"workspace_root", "/srv/workspace"}); err != nil {
			t.Fatalf("config set via api: %v", err)
		}
	})
	if !strings.Contains(outSet, "workspace_root = /srv/workspace") {
		t.Fatalf("salida config set inesperada:\n%s", outSet)
	}

	outNuevo := capturarStdout(t, func() {
		if err := configAgenteNuevoCmd.RunE(configAgenteNuevoCmd, []string{"Codex9", "programador"}); err != nil {
			t.Fatalf("config agente-nuevo via api: %v", err)
		}
	})
	if !strings.Contains(outNuevo, "Codex9") {
		t.Fatalf("salida config agente-nuevo inesperada:\n%s", outNuevo)
	}

	outRetirar := capturarStdout(t, func() {
		if err := configAgenteRetirarCmd.RunE(configAgenteRetirarCmd, []string{"Codex9"}); err != nil {
			t.Fatalf("config agente-retirar via api: %v", err)
		}
	})
	if !strings.Contains(outRetirar, "retirado") {
		t.Fatalf("salida config agente-retirar inesperada:\n%s", outRetirar)
	}

	outRehabilitar := capturarStdout(t, func() {
		if err := configAgenteRehabilitarCmd.RunE(configAgenteRehabilitarCmd, []string{"Codex9"}); err != nil {
			t.Fatalf("config agente-rehabilitar via api: %v", err)
		}
	})
	if !strings.Contains(outRehabilitar, "rehabilitado") {
		t.Fatalf("salida config agente-rehabilitar inesperada:\n%s", outRehabilitar)
	}
}

func TestConfigExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	if err := configVerCmd.RunE(configVerCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("config ver deberia exigir servidor, err=%v", err)
	}

	if err := configSetCmd.RunE(configSetCmd, []string{"workspace_root", "/srv/workspace"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("config set deberia exigir servidor, err=%v", err)
	}

	if err := configAgenteNuevoCmd.RunE(configAgenteNuevoCmd, []string{"Codex9", "programador"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("config agente-nuevo deberia exigir servidor, err=%v", err)
	}

	if err := configAgenteRetirarCmd.RunE(configAgenteRetirarCmd, []string{"Codex9"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("config agente-retirar deberia exigir servidor, err=%v", err)
	}

	if err := configAgenteRehabilitarCmd.RunE(configAgenteRehabilitarCmd, []string{"Codex9"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("config agente-rehabilitar deberia exigir servidor, err=%v", err)
	}
}

func TestConectorUsaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	conector := &db.Conector{
		ID:           12,
		Slug:         "codex",
		Nombre:       "Codex CLI",
		Transporte:   "cli",
		Comando:      "codex",
		ArgsJSON:     "[]",
		EnvJSON:      "{}",
		MetadataJSON: "{}",
		Activo:       true,
	}
	mux.HandleFunc("/api/conectores", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(apiConectoresResponse{Conectores: []*db.Conector{conector}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(apiConectorResponse{Conector: conector})
		default:
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/conectores/codex", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiConectorResponse{Conector: conector})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	outListar := capturarStdout(t, func() {
		if err := conectorListarCmd.RunE(conectorListarCmd, nil); err != nil {
			t.Fatalf("conector listar via api: %v", err)
		}
	})
	for _, token := range []string{"SLUG", "codex", "Codex CLI"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida conector listar sin %q:\n%s", token, outListar)
		}
	}

	outVer := capturarStdout(t, func() {
		if err := conectorVerCmd.RunE(conectorVerCmd, []string{"codex"}); err != nil {
			t.Fatalf("conector ver via api: %v", err)
		}
	})
	for _, token := range []string{"Conector #12", "codex", "Codex CLI"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida conector ver sin %q:\n%s", token, outVer)
		}
	}

	resetCommandFlags(conectorRegistrarCmd)
	_ = conectorRegistrarCmd.Flags().Set("slug", "codex")
	_ = conectorRegistrarCmd.Flags().Set("nombre", "Codex CLI")
	_ = conectorRegistrarCmd.Flags().Set("comando", "codex")
	outRegistrar := capturarStdout(t, func() {
		if err := conectorRegistrarCmd.RunE(conectorRegistrarCmd, nil); err != nil {
			t.Fatalf("conector registrar via api: %v", err)
		}
	})
	if !strings.Contains(outRegistrar, "id: 12") {
		t.Fatalf("salida conector registrar inesperada:\n%s", outRegistrar)
	}
}

func TestConectorExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	if err := conectorListarCmd.RunE(conectorListarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("conector listar deberia exigir servidor, err=%v", err)
	}

	if err := conectorVerCmd.RunE(conectorVerCmd, []string{"codex"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("conector ver deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(conectorRegistrarCmd)
	_ = conectorRegistrarCmd.Flags().Set("slug", "codex")
	_ = conectorRegistrarCmd.Flags().Set("nombre", "Codex CLI")
	_ = conectorRegistrarCmd.Flags().Set("comando", "codex")
	if err := conectorRegistrarCmd.RunE(conectorRegistrarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("conector registrar deberia exigir servidor, err=%v", err)
	}
}

func TestProyectoUsaAPI(t *testing.T) {
	padreID := int64(2)
	proyecto := &db.Proyecto{
		ID:        7,
		Slug:      "orquestador",
		Nombre:    "Orquestador",
		RutaAbs:   "/tmp/orquestador",
		Tipo:      db.ProyectoRepo,
		ParentID:  &padreID,
		Activo:    true,
		CreatedAt: time.Date(2026, 3, 24, 18, 0, 0, 0, time.UTC),
	}
	padre := &db.Proyecto{
		ID:      2,
		Slug:    "grupo-pm",
		Nombre:  "PlataformaMunicipal",
		RutaAbs: "/tmp/PlataformaMunicipal",
		Tipo:    db.ProyectoGrupo,
		Activo:  true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/proyectos", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiProyectosResponse{Proyectos: []*db.Proyecto{proyecto}})
	})
	mux.HandleFunc("/api/proyectos/descubrir", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiProyectosResponse{Proyectos: []*db.Proyecto{proyecto}})
	})
	mux.HandleFunc("/api/proyectos/orquestador", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiProyectoResponse{Proyecto: proyecto})
	})
	mux.HandleFunc("/api/proyectos/2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiProyectoResponse{Proyecto: padre})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	outListar := capturarStdout(t, func() {
		if err := proyectoListarCmd.RunE(proyectoListarCmd, nil); err != nil {
			t.Fatalf("proyecto listar via api: %v", err)
		}
	})
	for _, token := range []string{"SLUG", "orquestador", "/tmp/orquestador"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida proyecto listar sin %q:\n%s", token, outListar)
		}
	}

	outVer := capturarStdout(t, func() {
		if err := proyectoVerCmd.RunE(proyectoVerCmd, []string{"orquestador"}); err != nil {
			t.Fatalf("proyecto ver via api: %v", err)
		}
	})
	for _, token := range []string{"Proyecto #7", "orquestador", "2 (grupo-pm)"} {
		if !strings.Contains(outVer, token) {
			t.Fatalf("salida proyecto ver sin %q:\n%s", token, outVer)
		}
	}

	outDescubrir := capturarStdout(t, func() {
		if err := proyectoDescubrirCmd.RunE(proyectoDescubrirCmd, []string{"/tmp"}); err != nil {
			t.Fatalf("proyecto descubrir via api: %v", err)
		}
	})
	if !strings.Contains(outDescubrir, "1 proyectos") {
		t.Fatalf("salida proyecto descubrir inesperada:\n%s", outDescubrir)
	}
}

func TestProyectoExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	if err := proyectoListarCmd.RunE(proyectoListarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("proyecto listar deberia exigir servidor, err=%v", err)
	}

	if err := proyectoVerCmd.RunE(proyectoVerCmd, []string{"orquestador"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("proyecto ver deberia exigir servidor, err=%v", err)
	}

	if err := proyectoDescubrirCmd.RunE(proyectoDescubrirCmd, []string{"/tmp"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("proyecto descubrir deberia exigir servidor, err=%v", err)
	}
}

func TestAsignacionUsaAPI(t *testing.T) {
	asignacion := &db.Asignacion{
		ID:           14,
		Agente:       "Codex1",
		ProyectoID:   7,
		ProyectoSlug: "orquestador",
		Estado:       db.AsignacionActiva,
		Nota:         "principal",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/asignaciones", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiAsignacionesResponse{Asignaciones: []*db.Asignacion{asignacion}})
	})
	mux.HandleFunc("/api/asignaciones/activar", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", srv.URL)()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	resetCommandFlags(asignacionListarCmd)
	_ = asignacionListarCmd.Flags().Set("agente", "Codex1")
	_ = asignacionListarCmd.Flags().Set("proyecto", "orquestador")
	_ = asignacionListarCmd.Flags().Set("estado", "activa")
	outListar := capturarStdout(t, func() {
		if err := asignacionListarCmd.RunE(asignacionListarCmd, nil); err != nil {
			t.Fatalf("asignacion listar via api: %v", err)
		}
	})
	for _, token := range []string{"Codex1", "orquestador", "principal"} {
		if !strings.Contains(outListar, token) {
			t.Fatalf("salida asignacion listar sin %q:\n%s", token, outListar)
		}
	}

	resetCommandFlags(asignacionActivarCmd)
	_ = asignacionActivarCmd.Flags().Set("nota", "principal")
	outActivar := capturarStdout(t, func() {
		if err := asignacionActivarCmd.RunE(asignacionActivarCmd, []string{"Codex1", "orquestador"}); err != nil {
			t.Fatalf("asignacion activar via api: %v", err)
		}
	})
	if !strings.Contains(outActivar, "Codex1 asignado a orquestador") {
		t.Fatalf("salida asignacion activar inesperada:\n%s", outActivar)
	}
}

func TestAsignacionExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	resetCommandFlags(asignacionListarCmd)
	if err := asignacionListarCmd.RunE(asignacionListarCmd, nil); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("asignacion listar deberia exigir servidor, err=%v", err)
	}

	resetCommandFlags(asignacionActivarCmd)
	_ = asignacionActivarCmd.Flags().Set("nota", "principal")
	if err := asignacionActivarCmd.RunE(asignacionActivarCmd, []string{"Codex1", "orquestador"}); err == nil || !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("asignacion activar deberia exigir servidor, err=%v", err)
	}
}
