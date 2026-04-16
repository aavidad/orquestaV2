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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/rpclocal"
)

func TestCommandSupportsServerMode(t *testing.T) {
	casos := []struct {
		nombre string
		args   []string
		want   bool
	}{
		{nombre: "proyecto listar", args: []string{"proyecto", "listar"}, want: true},
		{nombre: "proyecto ver", args: []string{"proyecto", "ver", "orquestador"}, want: true},
		{nombre: "proyecto descubrir", args: []string{"proyecto", "descubrir", "."}, want: true},
		{nombre: "proyecto fusionar", args: []string{"proyecto", "fusionar", "orquesta", "orquestador"}, want: true},
		{nombre: "proyecto microciclo", args: []string{"proyecto", "microciclo", "orquestador"}, want: true},
		{nombre: "proyecto microciclo limpio", args: []string{"proyecto", "microciclo", "orquestador", "--limpiar-pruebas"}, want: true},
		{nombre: "pool listar", args: []string{"pool", "listar"}, want: true},
		{nombre: "pool ver", args: []string{"pool", "ver", "codex"}, want: true},
		{nombre: "politica modelo listar", args: []string{"politica-modelo", "listar"}, want: true},
		{nombre: "modelo resolver", args: []string{"modelo", "resolver", "--perfil", "programador"}, want: true},
		{nombre: "modelo pipeline local", args: []string{"modelo", "pipeline-local", "--proyecto", "orquestador"}, want: true},
		{nombre: "modelo pipeline paso", args: []string{"modelo", "pipeline-paso", "--proyecto", "orquestador"}, want: true},
		{nombre: "modelo pipeline despachar", args: []string{"modelo", "pipeline-despachar", "--proyecto", "orquestador"}, want: true},
		{nombre: "modelo runtime activos", args: []string{"modelo", "runtime-activos"}, want: true},
		{nombre: "modelo runtime descargar", args: []string{"modelo", "runtime-descargar"}, want: true},
		{nombre: "progreso ver", args: []string{"progreso", "ver", "orquestador"}, want: true},
		{nombre: "progreso fase listar", args: []string{"progreso", "fase", "listar", "orquestador"}, want: true},
		{nombre: "progreso fase registrar", args: []string{"progreso", "fase", "registrar", "orquestador"}, want: true},
		{nombre: "progreso fase actualizar", args: []string{"progreso", "fase", "actualizar", "12"}, want: true},
		{nombre: "progreso tarea registrar", args: []string{"progreso", "tarea", "registrar", "12"}, want: true},
		{nombre: "reglas listar", args: []string{"reglas", "listar"}, want: true},
		{nombre: "reglas crear", args: []string{"reglas", "crear"}, want: true},
		{nombre: "skills listar", args: []string{"skills", "listar"}, want: true},
		{nombre: "skills crear", args: []string{"skills", "crear"}, want: true},
		{nombre: "skills detectar carencia", args: []string{"skills", "detectar-carencia", "--rol", "programador"}, want: true},
		{nombre: "skills importar", args: []string{"skills", "importar", "--rol", "programador", "--repo", "openai/skills", "--skill", "openai-docs"}, want: true},
		{nombre: "skills remotas", args: []string{"skills", "remotas", "--q", "openai"}, want: true},
		{nombre: "skills borrar", args: []string{"skills", "borrar", "12"}, want: true},
		{nombre: "workflows listar", args: []string{"workflows", "listar"}, want: true},
		{nombre: "workflows crear", args: []string{"workflows", "crear"}, want: true},
		{nombre: "permisos fijar", args: []string{"permisos", "fijar"}, want: true},
		{nombre: "conector listar", args: []string{"conector", "listar"}, want: true},
		{nombre: "conector ver", args: []string{"conector", "ver", "codex-cli"}, want: true},
		{nombre: "conector registrar", args: []string{"conector", "registrar"}, want: true},
		{nombre: "config ver", args: []string{"config", "ver"}, want: true},
		{nombre: "config set", args: []string{"config", "set", "clave", "valor"}, want: true},
		{nombre: "config agente nuevo", args: []string{"config", "agente-nuevo", "Codex9", "programador"}, want: true},
		{nombre: "config agente retirar", args: []string{"config", "agente-retirar", "Codex9"}, want: true},
		{nombre: "config agente rehabilitar", args: []string{"config", "agente-rehabilitar", "Codex9"}, want: true},
		{nombre: "asignacion listar", args: []string{"asignacion", "listar"}, want: true},
		{nombre: "asignacion activar", args: []string{"asignacion", "activar", "Codex1", "orquestador"}, want: true},
		{nombre: "lock listar", args: []string{"lock", "listar"}, want: true},
		{nombre: "lock tomar", args: []string{"lock", "tomar", "Codex1", "file", "x"}, want: true},
		{nombre: "lock renovar", args: []string{"lock", "renovar", "1", "Codex1", "lease"}, want: true},
		{nombre: "lock liberar", args: []string{"lock", "liberar", "1", "Codex1", "lease"}, want: true},
		{nombre: "worktree listar", args: []string{"worktree", "listar"}, want: true},
		{nombre: "worktree resolver", args: []string{"worktree", "resolver", "Codex1", "orquestador"}, want: true},
		{nombre: "worktree crear", args: []string{"worktree", "crear", "Codex1", "orquestador"}, want: true},
		{nombre: "worktree cerrar", args: []string{"worktree", "cerrar", "1"}, want: true},
		{nombre: "runtime listar", args: []string{"runtime", "listar"}, want: true},
		{nombre: "runtime ver", args: []string{"runtime", "ver", "12"}, want: true},
		{nombre: "runtime handles", args: []string{"runtime", "handles"}, want: true},
		{nombre: "runtime despertar", args: []string{"runtime", "despertar"}, want: true},
		{nombre: "runtime procesar mailbox", args: []string{"runtime", "procesar-mailbox"}, want: true},
		{nombre: "runtime procesar autonomia", args: []string{"runtime", "procesar-autonomia"}, want: true},
		{nombre: "runtime procesar reanimaciones", args: []string{"runtime", "procesar-reanimaciones"}, want: true},
		{nombre: "runtime limpiar pruebas", args: []string{"runtime", "limpiar-pruebas", "--agente", "Codex1"}, want: true},
		{nombre: "runtime traza", args: []string{"runtime", "traza", "--agente", "Codex1"}, want: true},
		{nombre: "runtime transcript", args: []string{"runtime", "transcript", "--agente", "Codex1"}, want: true},
		{nombre: "runtime ordenes", args: []string{"runtime", "ordenes"}, want: true},
		{nombre: "runtime orden nueva", args: []string{"runtime", "orden-nueva", "Codex1", "checkpoint"}, want: true},
		{nombre: "runtime orden cancelar", args: []string{"runtime", "orden-cancelar", "15"}, want: true},
		{nombre: "runtime nudge", args: []string{"runtime", "nudge", "Codex1", "retoma", "el", "bloqueo"}, want: true},
		{nombre: "runtime discordia", args: []string{"runtime", "discordia", "alberto", "Codex2", "hay", "desacuerdo"}, want: true},
		{nombre: "runtime checkpoints", args: []string{"runtime", "checkpoints", "--agente", "Codex1"}, want: true},
		{nombre: "runtime checkpoint nuevo", args: []string{"runtime", "checkpoint-nuevo", "Codex1"}, want: true},
		{nombre: "runtime mailbox", args: []string{"runtime", "mailbox", "--to", "Codex1"}, want: true},
		{nombre: "runtime mailbox limpiar", args: []string{"runtime", "mailbox-limpiar", "--to", "Codex1"}, want: true},
		{nombre: "runtime mailbox enviar", args: []string{"runtime", "mailbox-enviar", "Codex1", "Codex2", "handoff"}, want: true},
		{nombre: "runtime mailbox entregar", args: []string{"runtime", "mailbox-entregar", "12"}, want: true},
		{nombre: "runtime mailbox consumir", args: []string{"runtime", "mailbox-consumir", "12"}, want: true},
		{nombre: "sesion inicio", args: []string{"sesion", "inicio", "Codex1"}, want: true},
		{nombre: "sesion historial", args: []string{"sesion", "historial"}, want: true},
		{nombre: "sesion ver", args: []string{"sesion", "ver", "12"}, want: true},
		{nombre: "sesion nuevo-codex", args: []string{"sesion", "nuevo-codex"}, want: true},
		{nombre: "sesion presupuesto ver", args: []string{"sesion", "presupuesto", "ver", "--agente", "Codex1"}, want: true},
		{nombre: "sesion presupuesto registrar", args: []string{"sesion", "presupuesto", "registrar", "--agente", "Codex1"}, want: true},
		{nombre: "memoria listar", args: []string{"memoria", "listar", "--proyecto", "orquestador"}, want: true},
		{nombre: "memoria ver", args: []string{"memoria", "ver", "Core_API", "--proyecto", "orquestador"}, want: true},
		{nombre: "memoria guardar", args: []string{"memoria", "guardar", "Core_API", "api", "--valor", "{}"}, want: true},
		{nombre: "agente control arrancar", args: []string{"agente", "control", "arrancar", "Codex1", "--proyecto", "orquestador"}, want: true},
		{nombre: "agente cuentas", args: []string{"agente", "cuentas", "--activos"}, want: true},
		{nombre: "agente presupuesto", args: []string{"agente", "presupuesto", "--activos"}, want: true},
		{nombre: "agente ranking cuentas", args: []string{"agente", "ranking-cuentas", "--activos"}, want: true},
		{nombre: "agente observar cuenta", args: []string{"agente", "observar-cuenta", "Codex7", "--email", "berserk@avidad.com"}, want: true},
		{nombre: "agente investigar", args: []string{"agente", "investigar", "refactor"}, want: true},
		{nombre: "agente overview", args: []string{"agente", "overview", "Codex1"}, want: true},
		{nombre: "agente reanimaciones", args: []string{"agente", "reanimaciones"}, want: true},
		{nombre: "agente retirar", args: []string{"agente", "retirar", "Codex1"}, want: true},
		{nombre: "agente fusionar", args: []string{"agente", "fusionar", "codex1", "Codex1"}, want: true},
		{nombre: "agente lanzar plan", args: []string{"agente", "lanzar-plan", "scripts/agentes.orquestador.plan"}, want: true},
		{nombre: "importar historial", args: []string{"importar", "historial"}, want: true},
		{nombre: "exportar diagnostico", args: []string{"exportar", "diagnostico"}, want: true},
		{nombre: "logs", args: []string{"logs"}, want: true},
		{nombre: "auditoria listar", args: []string{"auditoria", "listar"}, want: true},
		{nombre: "respaldo bd", args: []string{"respaldo", "bd"}, want: true},
		{nombre: "tarea cancelar", args: []string{"tarea", "cancelar", "12", "Codex1"}, want: true},
		{nombre: "tarea notas", args: []string{"tarea", "notas", "12"}, want: true},
		{nombre: "propuesta actualizar", args: []string{"propuesta", "actualizar", "OP-116", "--titulo", "Nuevo"}, want: true},
		{nombre: "runtime purgar-handles", args: []string{"runtime", "purgar-handles", "--agente", "Codex1"}, want: true},
		{nombre: "runtime purgar-ordenes", args: []string{"runtime", "purgar-ordenes", "--agente", "Codex1"}, want: true},
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

func TestShouldPreferAPIClientUsaHTTPServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "metodo no soportado", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "orquesta"})
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

	if !shouldPreferAPIClient([]string{"runtime", "listar"}) {
		t.Fatalf("se esperaba preferencia por el cliente HTTP API")
	}
	if shouldPreferAPIClient([]string{"server", "status"}) {
		t.Fatalf("no deberia preferir API para comandos de server")
	}
}

func TestRuntimeMutacionesCriticasSoportanServerMode(t *testing.T) {
	cases := [][]string{
		{"runtime", "orden-nueva", "Codex1", "checkpoint"},
		{"runtime", "nudge", "Codex1"},
		{"runtime", "discordia", "Codex1", "bloqueo"},
		{"runtime", "checkpoint-nuevo", "Codex1"},
		{"runtime", "mailbox-enviar", "Codex1", "Codex2", "handoff"},
		{"runtime", "mailbox-entregar", "42"},
		{"runtime", "mailbox-consumir", "42"},
		{"runtime", "mailbox-limpiar", "--to", "Codex1"},
		{"runtime", "purgar-handles", "--agente", "Codex6"},
		{"runtime", "purgar-ordenes", "--agente", "Codex6"},
		{"runtime", "despertar", "--mailbox"},
		{"runtime", "procesar-reanimaciones"},
		{"runtime", "limpiar-pruebas", "--agente", "Codex1"},
	}
	for _, args := range cases {
		if !commandSupportsServerMode(args) {
			t.Fatalf("commandSupportsServerMode(%v)=false, deberia ser true", args)
		}
	}
}

func TestShouldPreferAPIClientConServidorExplicitoAunqueNoResponda(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "http://127.0.0.1:1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	if !shouldPreferAPIClient([]string{"runtime", "listar"}) {
		t.Fatalf("con ORQUESTA_SERVER_URL explicito deberia preferir API para evitar fallback local")
	}
}

func TestShouldPreferAPIClientParaServerFirstSinDescubrimientoPrevio(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	if !shouldPreferAPIClient([]string{"status"}) {
		t.Fatalf("status deberia mantenerse en server-first aunque no haya descubrimiento previo")
	}
	if shouldPreferAPIClient([]string{"persistencia", "info"}) {
		t.Fatalf("persistencia info no deberia preferir API")
	}
}

func TestShouldPreferAPIClientUsaRPCLocalAunqueAPIStatusSeaLenta(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{OK: true})
	})
	mux.HandleFunc("/api/server", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{"name": "orquesta"})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()

	statePath := filepath.Join(t.TempDir(), "orquesta-localrpc.json")
	if err := rpclocal.SaveState(statePath, &rpclocal.State{
		Addr:    strings.TrimPrefix(srv.URL, "http://"),
		ScopeID: rpclocal.CurrentScopeID(),
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", statePath)()
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	if !shouldPreferAPIClient([]string{"runtime", "listar"}) {
		t.Fatalf("se esperaba preferencia por API apoyándose en localrpc aunque /api/status sea lento")
	}
}

func TestServerBaseURLUsaHealthzSiFaltaStatefile(t *testing.T) {
	var advertisedAddr string
	mux := http.NewServeMux()
	mux.HandleFunc(rpclocal.HealthPath, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(rpclocal.HealthResponse{
			OK:      true,
			Addr:    advertisedAddr,
			PID:     4242,
			ScopeID: rpclocal.CurrentScopeID(),
		})
	})
	srv := newTestHTTPServerOrSkip(t, mux)
	defer srv.Close()
	advertisedAddr = strings.TrimPrefix(srv.URL, "http://")

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_INFO", t.TempDir()+"/missing-state.json")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_STATE", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_ADDR", strings.TrimPrefix(srv.URL, "http://"))()

	if got := serverBaseURL(); got != srv.URL {
		t.Fatalf("serverBaseURL=%q, want %q", got, srv.URL)
	}
}

func TestRequireServerForCurrentCommand(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarArgs(t, []string{"orquesta", "runtime", "listar"})()

	if !requireServerForCurrentCommand() {
		t.Fatalf("se esperaba exigir servidor para runtime listar")
	}
}

func TestAPIGetNoHaceFallbackConServidorExplicitoCaido(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "http://127.0.0.1:1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "")()

	var resp apiRuntimesResponse
	ok, err := apiGet("/api/runtimes", &resp)
	if !ok {
		t.Fatalf("se esperaba bloqueo explicito sin fallback local")
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "no se pudo contactar") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestAPIGetNoHaceFallbackCuandoServidorEsObligatorio(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_REQUIRE_SERVER", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarArgs(t, []string{"orquesta", "runtime", "listar"})()

	var resp apiRuntimesResponse
	ok, err := apiGet("/api/runtimes", &resp)
	if !ok {
		t.Fatalf("se esperaba bloqueo explicito sin fallback local")
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "requiere el servidor") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestAPIGetUsaAPILocalEnRecuperacionLocalDB(t *testing.T) {
	prepararDBTemporalCmd(t)

	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()

	var resp apiStatusResponse
	ok, err := apiGet("/api/status", &resp)
	if !ok {
		t.Fatalf("se esperaba uso de API local en recuperacion")
	}
	if err != nil {
		t.Fatalf("apiGet local: %v", err)
	}
	if resp.ConteoTareas == nil {
		t.Fatalf("respuesta status local vacia: %+v", resp)
	}
}

func TestServerBaseURLUsaPuertoPorDefectoDeServe(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "")()

	if got := serverBaseURL(); got != defaultServerURL {
		t.Fatalf("serverBaseURL() = %q, want %q", got, defaultServerURL)
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
