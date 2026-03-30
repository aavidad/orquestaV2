/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"orquesta/db"
	"orquesta/reviewapp"
	"orquesta/supervisionapp"
)

var cmdTestDBMu sync.Mutex
var cmdTestBootstrapOnce sync.Once
var cmdTestBootstrapData []byte
var cmdTestBootstrapErr error

func cargarPlantillaDBCmdTest() ([]byte, error) {
	cmdTestBootstrapOnce.Do(func() {
		tmp, err := os.MkdirTemp("", "orquesta-cmd-db-template-*")
		if err != nil {
			cmdTestBootstrapErr = err
			return
		}
		defer os.RemoveAll(tmp)

		anteriorDB := os.Getenv("ORQUESTA_DB")
		anteriorDSN, teniaDSN := os.LookupEnv("ORQUESTA_DB_DSN")
		anteriorDriver, teniaDriver := os.LookupEnv("ORQUESTA_DB_DRIVER")
		anteriorBackend, teniaBackend := os.LookupEnv("ORQUESTA_DB_BACKEND")
		anteriorMaxOpenConns, teniaMaxOpenConns := os.LookupEnv("ORQUESTA_DB_MAX_OPEN_CONNS")
		anteriorBootstrap, teniaBootstrap := os.LookupEnv("ORQUESTA_DB_BOOTSTRAP")
		anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
		anteriorForceLocal, teniaForceLocal := os.LookupEnv("ORQUESTA_FORCE_LOCAL_DB")
		anteriorDisableServer, teniaDisableServer := os.LookupEnv("ORQUESTA_DISABLE_SERVER_CLIENT")
		defer func() {
			db.Close()
			db.DB = nil
			if anteriorDB == "" {
				_ = os.Unsetenv("ORQUESTA_DB")
			} else {
				_ = os.Setenv("ORQUESTA_DB", anteriorDB)
			}
			if teniaDSN {
				_ = os.Setenv("ORQUESTA_DB_DSN", anteriorDSN)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_DSN")
			}
			if teniaDriver {
				_ = os.Setenv("ORQUESTA_DB_DRIVER", anteriorDriver)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
			}
			if teniaBackend {
				_ = os.Setenv("ORQUESTA_DB_BACKEND", anteriorBackend)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
			}
			if teniaMaxOpenConns {
				_ = os.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", anteriorMaxOpenConns)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS")
			}
			if teniaBootstrap {
				_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", anteriorBootstrap)
			} else {
				_ = os.Unsetenv("ORQUESTA_DB_BOOTSTRAP")
			}
			if anteriorRoot == "" {
				_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
			} else {
				_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
			}
			if teniaForceLocal {
				_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", anteriorForceLocal)
			} else {
				_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
			}
			if teniaDisableServer {
				_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", anteriorDisableServer)
			} else {
				_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
			}
		}()

		dbPath := filepath.Join(tmp, "orquesta-cmd-template.db")
		_ = os.Setenv("ORQUESTA_DB", dbPath)
		_ = os.Unsetenv("ORQUESTA_DB_DSN")
		_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
		_ = os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS")
		_ = os.Unsetenv("ORQUESTA_DB_BOOTSTRAP")
		_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp)
		_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1")
		_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1")

		if err := db.Open(); err != nil {
			cmdTestBootstrapErr = err
			return
		}
		if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
			cmdTestBootstrapErr = err
			return
		}
		db.Close()
		cmdTestBootstrapData, cmdTestBootstrapErr = os.ReadFile(dbPath)
	})
	return cmdTestBootstrapData, cmdTestBootstrapErr
}

func prepararDBTemporalCmd(t *testing.T) string {
	t.Helper()
	cmdTestDBMu.Lock()

	anteriorDB := os.Getenv("ORQUESTA_DB")
	anteriorDSN, teniaDSN := os.LookupEnv("ORQUESTA_DB_DSN")
	anteriorDriver, teniaDriver := os.LookupEnv("ORQUESTA_DB_DRIVER")
	anteriorBackend, teniaBackend := os.LookupEnv("ORQUESTA_DB_BACKEND")
	anteriorMaxOpenConns, teniaMaxOpenConns := os.LookupEnv("ORQUESTA_DB_MAX_OPEN_CONNS")
	anteriorBootstrap, teniaBootstrap := os.LookupEnv("ORQUESTA_DB_BOOTSTRAP")
	anteriorRoot := os.Getenv("ORQUESTA_WORKSPACE_ROOT")
	anteriorForceLocal, teniaForceLocal := os.LookupEnv("ORQUESTA_FORCE_LOCAL_DB")
	anteriorDisableServer, teniaDisableServer := os.LookupEnv("ORQUESTA_DISABLE_SERVER_CLIENT")
	t.Cleanup(func() {
		db.Close()
		db.DB = nil
		if anteriorDB == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", anteriorDB)
		}
		if teniaDSN {
			_ = os.Setenv("ORQUESTA_DB_DSN", anteriorDSN)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_DSN")
		}
		if teniaDriver {
			_ = os.Setenv("ORQUESTA_DB_DRIVER", anteriorDriver)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_DRIVER")
		}
		if teniaBackend {
			_ = os.Setenv("ORQUESTA_DB_BACKEND", anteriorBackend)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_BACKEND")
		}
		if teniaMaxOpenConns {
			_ = os.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", anteriorMaxOpenConns)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS")
		}
		if teniaBootstrap {
			_ = os.Setenv("ORQUESTA_DB_BOOTSTRAP", anteriorBootstrap)
		} else {
			_ = os.Unsetenv("ORQUESTA_DB_BOOTSTRAP")
		}
		if anteriorRoot == "" {
			_ = os.Unsetenv("ORQUESTA_WORKSPACE_ROOT")
		} else {
			_ = os.Setenv("ORQUESTA_WORKSPACE_ROOT", anteriorRoot)
		}
		if teniaForceLocal {
			_ = os.Setenv("ORQUESTA_FORCE_LOCAL_DB", anteriorForceLocal)
		} else {
			_ = os.Unsetenv("ORQUESTA_FORCE_LOCAL_DB")
		}
		if teniaDisableServer {
			_ = os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", anteriorDisableServer)
		} else {
			_ = os.Unsetenv("ORQUESTA_DISABLE_SERVER_CLIENT")
		}
		cmdTestDBMu.Unlock()
	})

	db.Close()
	db.DB = nil

	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "orquesta-api-test.db")
	plantilla, err := cargarPlantillaDBCmdTest()
	if err != nil {
		t.Fatalf("cargar plantilla db cmd test: %v", err)
	}
	if err := os.WriteFile(dbPath, plantilla, 0o600); err != nil {
		t.Fatalf("write plantilla db cmd test: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DB", dbPath); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_DSN"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_DSN: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_DRIVER"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_DRIVER: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_BACKEND"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_BACKEND: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_MAX_OPEN_CONNS"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_MAX_OPEN_CONNS: %v", err)
	}
	if err := os.Unsetenv("ORQUESTA_DB_BOOTSTRAP"); err != nil {
		t.Fatalf("unsetenv ORQUESTA_DB_BOOTSTRAP: %v", err)
	}
	if err := os.Setenv("ORQUESTA_WORKSPACE_ROOT", tmp); err != nil {
		t.Fatalf("setenv ORQUESTA_WORKSPACE_ROOT: %v", err)
	}
	if err := os.Setenv("ORQUESTA_FORCE_LOCAL_DB", "1"); err != nil {
		t.Fatalf("setenv ORQUESTA_FORCE_LOCAL_DB: %v", err)
	}
	if err := os.Setenv("ORQUESTA_DISABLE_SERVER_CLIENT", "1"); err != nil {
		t.Fatalf("setenv ORQUESTA_DISABLE_SERVER_CLIENT: %v", err)
	}
	if err := db.Open(); err != nil {
		t.Fatalf("open db temporal: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base cmd: %v", err)
	}
	return tmp
}

func TestAPIAgentesListaJSON(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); len(got) < 16 || got[:16] != "application/json" {
		t.Fatalf("content-type inesperado: %s", got)
	}
}

func TestAPIServerExponeMetadatosDescubrimiento(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/server", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload serverInfo
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/server: %v", err)
	}
	if payload.Name != "orquesta" {
		t.Fatalf("name inesperado: %+v", payload)
	}
	if payload.StorageMode != "single-process" || payload.StorageDriver == "" || payload.SQLPlaceholder == "" {
		t.Fatalf("payload de descubrimiento incompleto: %+v", payload)
	}
	if len(payload.Capabilities) == 0 {
		t.Fatalf("capabilities vacias: %+v", payload)
	}
}

func TestAPIStatusExponeResumenOperativoCompat(t *testing.T) {
	prepararDBTemporalCmd(t)

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /api/status: %v", err)
	}
	for _, key := range []string{"agentes", "conteo_tareas", "generado", "tareasPorEstado", "agentesActivos", "propuestasAbiertas", "tareasActivas"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("/api/status sin clave %q: %+v", key, payload)
		}
	}
}

func TestAPIAgentesYStatusAlineanActivoConSesionReal(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"CodexVisible1", "CodexVisible2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexVisible1'`); err != nil {
		t.Fatalf("marcar estado visible1: %v", err)
	}
	if _, err := db.DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='disponible' WHERE nombre='CodexVisible2'`); err != nil {
		t.Fatalf("marcar activo visible2: %v", err)
	}
	if _, err := db.DB.Exec(`INSERT INTO sesiones (agente, activa, estado, herramienta, host) VALUES (?,?,?,?,?)`,
		"CodexVisible1", 1, "activa", "codex", "localhost",
	); err != nil {
		t.Fatalf("insert sesion visible1: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgentes := httptest.NewRecorder()
	reqAgentes := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(recAgentes, reqAgentes)
	if recAgentes.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", recAgentes.Code, recAgentes.Body.String())
	}
	var agentesResp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(recAgentes.Body.Bytes(), &agentesResp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	estadoAgentes := map[string]*db.Agente{}
	for _, agente := range agentesResp.Agentes {
		estadoAgentes[agente.Nombre] = agente
	}
	if !estadoAgentes["CodexVisible1"].Activo || estadoAgentes["CodexVisible1"].EstadoSesion != "pensando" {
		t.Fatalf("agente visible1 inesperado via /api/agentes: %+v", estadoAgentes["CodexVisible1"])
	}
	if estadoAgentes["CodexVisible2"].Activo || estadoAgentes["CodexVisible2"].EstadoSesion != "" {
		t.Fatalf("agente visible2 inesperado via /api/agentes: %+v", estadoAgentes["CodexVisible2"])
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	estadoStatus := map[string]*db.Agente{}
	for _, agente := range statusResp.Agentes {
		estadoStatus[agente.Nombre] = agente
	}
	if !estadoStatus["CodexVisible1"].Activo || estadoStatus["CodexVisible1"].EstadoSesion != "pensando" {
		t.Fatalf("agente visible1 inesperado via /api/status: %+v", estadoStatus["CodexVisible1"])
	}
	if estadoStatus["CodexVisible2"].Activo || estadoStatus["CodexVisible2"].EstadoSesion != "" {
		t.Fatalf("agente visible2 inesperado via /api/status: %+v", estadoStatus["CodexVisible2"])
	}
}

func TestAPIAgentesYStatusOcultanSesionZombi(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"CodexZombie", "CodexHandle"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", agente, err)
		}
	}
	old := "2026-03-22 16:39:43"
	if _, err := db.DB.Exec(`UPDATE agentes SET estado_sesion='pensando' WHERE nombre='CodexZombie'`); err != nil {
		t.Fatalf("estado zombie: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)`,
		"CodexZombie", 1, "activa", "codex", "localhost", old,
	); err != nil {
		t.Fatalf("insert sesion zombie: %v", err)
	}
	var handleSesionID int64
	if err := db.DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,?)
		RETURNING id`,
		"CodexHandle", 1, "activa", "codex", "localhost", old,
	).Scan(&handleSesionID); err != nil {
		t.Fatalf("insert sesion CodexHandle: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at
		) VALUES (?,?,?,?,?,'activo','{}','{}',CURRENT_TIMESTAMP)`,
		"CodexHandle", handleSesionID, "cli", "session", "sess-codex-handle",
	); err != nil {
		t.Fatalf("insert handle CodexHandle: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recAgentes := httptest.NewRecorder()
	reqAgentes := httptest.NewRequest(http.MethodGet, "/api/agentes", nil)
	mux.ServeHTTP(recAgentes, reqAgentes)
	if recAgentes.Code != http.StatusOK {
		t.Fatalf("status agentes inesperado: %d body=%s", recAgentes.Code, recAgentes.Body.String())
	}
	var agentesResp struct {
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := json.Unmarshal(recAgentes.Body.Bytes(), &agentesResp); err != nil {
		t.Fatalf("decode agentes: %v", err)
	}
	estadoAgentes := map[string]*db.Agente{}
	for _, agente := range agentesResp.Agentes {
		estadoAgentes[agente.Nombre] = agente
	}
	if estadoAgentes["CodexZombie"].Activo {
		t.Fatalf("CodexZombie no deberia salir activo via /api/agentes: %+v", estadoAgentes["CodexZombie"])
	}
	if !estadoAgentes["CodexHandle"].Activo {
		t.Fatalf("CodexHandle deberia seguir activo via /api/agentes: %+v", estadoAgentes["CodexHandle"])
	}

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	estadoStatus := map[string]*db.Agente{}
	for _, agente := range statusResp.Agentes {
		estadoStatus[agente.Nombre] = agente
	}
	if estadoStatus["CodexZombie"].Activo {
		t.Fatalf("CodexZombie no deberia salir activo via /api/status: %+v", estadoStatus["CodexZombie"])
	}
	if !estadoStatus["CodexHandle"].Activo {
		t.Fatalf("CodexHandle deberia seguir activo via /api/status: %+v", estadoStatus["CodexHandle"])
	}
}

func TestAPIStatusNoCuentaSesionConHandleFallidoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexFallo", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	var sesionID int64
	if err := db.DB.QueryRow(`
		INSERT INTO sesiones (agente, activa, estado, herramienta, host, heartbeat_at)
		VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
		RETURNING id`,
		"CodexFallo", 1, "activa", "codex-cli", "localhost",
	).Scan(&sesionID); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO runtime_handles (
			agente, sesion_id, transporte, handle_kind, handle_ref, estado,
			capabilities_json, metadata_json, last_seen_at, updated_at, created_at
		) VALUES (?,?,?,?,?,'fallido','{}','{}',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"CodexFallo", sesionID, "cli", "process", "9999",
	); err != nil {
		t.Fatalf("insert handle fallido: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recStatus := httptest.NewRecorder()
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("status global inesperado: %d body=%s", recStatus.Code, recStatus.Body.String())
	}
	var statusResp apiStatusResponse
	if err := json.Unmarshal(recStatus.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	for _, agente := range statusResp.Agentes {
		if agente != nil && agente.Nombre == "CodexFallo" && agente.Activo {
			t.Fatalf("CodexFallo no deberia salir activo via /api/status: %+v", agente)
		}
	}
}

func TestAPIAsignacionActivarYSesionInicio(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.GetProyecto("autofirmav2"); err != nil {
		t.Fatalf("get proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	asignacionBody, _ := json.Marshal(apiAsignacionActivarRequest{
		Agente:   "Codex1",
		Proyecto: "autofirmav2",
		Nota:     "slot 1",
	})
	recAsignacion := httptest.NewRecorder()
	reqAsignacion := httptest.NewRequest(http.MethodPost, "/api/asignaciones/activar", bytes.NewReader(asignacionBody))
	mux.ServeHTTP(recAsignacion, reqAsignacion)
	if recAsignacion.Code != http.StatusOK {
		t.Fatalf("status asignacion inesperado: %d body=%s", recAsignacion.Code, recAsignacion.Body.String())
	}

	sesionBody, _ := json.Marshal(apiSesionInicioRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-001",
		Resumen:           "continuar desde API",
	})
	recSesion := httptest.NewRecorder()
	reqSesion := httptest.NewRequest(http.MethodPost, "/api/sesiones/inicio", bytes.NewReader(sesionBody))
	mux.ServeHTTP(recSesion, reqSesion)
	if recSesion.Code != http.StatusCreated {
		t.Fatalf("status sesion inesperado: %d body=%s", recSesion.Code, recSesion.Body.String())
	}
	var resp apiSesionInicioResponse
	if err := json.Unmarshal(recSesion.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode sesion inicio: %v", err)
	}
	if resp.Sesion == nil || resp.Sesion.Agente != "Codex1" {
		t.Fatalf("respuesta de sesion inesperada: %+v", resp.Sesion)
	}
	if resp.Rol == "" {
		t.Fatalf("se esperaba rol en la respuesta de sesion inicio")
	}
	if len(resp.Reglas) == 0 {
		t.Fatalf("se esperaban reglas en la respuesta de sesion inicio")
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "sess-001" {
		t.Fatalf("external session id inesperado: %s", sesion.ExternalSessionID)
	}
}

func TestAPISesionInicioArranqueLimpioOmiteContinuidadPrevia(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    "codex",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		ConectorID:         &conectorID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID:  "sess-previa",
		ResumePayloadJSON:  `{"continuidad":true}`,
		ResumenContinuidad: "seguir desde antes",
	}); err != nil {
		t.Fatalf("iniciar sesion previa: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	sesionBody, _ := json.Marshal(apiSesionInicioRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-nueva",
		ResumePayload:     `{"nueva":true}`,
		Resumen:           "no deberia persistir",
		ArranqueLimpio:    true,
	})
	recSesion := httptest.NewRecorder()
	reqSesion := httptest.NewRequest(http.MethodPost, "/api/sesiones/inicio", bytes.NewReader(sesionBody))
	mux.ServeHTTP(recSesion, reqSesion)
	if recSesion.Code != http.StatusCreated {
		t.Fatalf("status sesion inesperado: %d body=%s", recSesion.Code, recSesion.Body.String())
	}
	var resp apiSesionInicioResponse
	if err := json.Unmarshal(recSesion.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode sesion inicio: %v", err)
	}
	if resp.SesionPrevia != nil {
		t.Fatalf("no deberia exponer sesion previa: %+v", resp.SesionPrevia)
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "" || sesion.ResumePayloadJSON != "" || sesion.ResumenContinuidad != "" {
		t.Fatalf("continuidad no limpiada en arranque limpio: %+v", sesion)
	}
}

func TestAPISesionGuardarPermiteLimpiarContinuidad(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID:  "sess-guardar",
		ResumePayloadJSON:  `{"continuidad":true}`,
		ResumenContinuidad: "seguir guardado",
		Branch:             "feature/x",
	}); err != nil {
		t.Fatalf("iniciar sesion activa: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiSesionGuardarRequest{
		Agente:             "Codex1",
		Proyecto:           "autofirmav2",
		LimpiarContinuidad: true,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/sesiones/guardar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status guardar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	sesion, err := db.GetSesionActiva("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get sesion activa: %v", err)
	}
	if sesion.ExternalSessionID != "" || sesion.ResumePayloadJSON != "" || sesion.ResumenContinuidad != "" {
		t.Fatalf("continuidad no limpiada: %+v", sesion)
	}
	if sesion.Branch != "feature/x" {
		t.Fatalf("branch no deberia cambiar: %+v", sesion)
	}
}

func TestAPIAgenteAdoptarContextoPersisteContinuidadYGobernanza(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-cli",
		Nombre:     "Codex CLI",
		Transporte: "cli",
		Comando:    "codex",
		Activo:     true,
	}); err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Completar proyecto",
		Descripcion: "Trabajo pendiente",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiAgenteAdoptarContextoRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-cli",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		Branch:            "feature/adopcion",
		ExternalSessionID: "sess-codex-actual",
		Resumen:           "seguir este mismo proyecto hasta terminarlo",
		Nota:              "adopcion de la sesion actual",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/adoptar-contexto", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status adoptar contexto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteAdoptarContextoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode adoptar contexto: %v", err)
	}
	if resp.Sesion == nil || resp.Sesion.Agente != "Codex1" {
		t.Fatalf("sesion adoptada inesperada: %+v", resp.Sesion)
	}
	if resp.Sesion.Activa || resp.Sesion.Estado != "pausada" {
		t.Fatalf("la sesion adoptada deberia quedar aparcada hasta el takeover: %+v", resp.Sesion)
	}
	if resp.Checkpoint == nil || resp.Checkpoint.CheckpointKind != "contexto_adoptado" {
		t.Fatalf("checkpoint adoptado inesperado: %+v", resp.Checkpoint)
	}
	if resp.RuntimeOrderID == nil || *resp.RuntimeOrderID <= 0 {
		t.Fatalf("se esperaba runtime_order start para la adopcion: %+v", resp)
	}
	if resp.Gobernanza == nil || strings.TrimSpace(resp.Gobernanza.ResolucionActual) == "" {
		t.Fatalf("catalogo de gobernanza ausente: %+v", resp.Gobernanza)
	}

	sesion, err := db.ObtenerUltimaSesion("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get ultima sesion: %v", err)
	}
	if !strings.Contains(sesion.ResumePayloadJSON, `"governance_catalog"`) || !strings.Contains(sesion.ResumePayloadJSON, `"project_context"`) {
		t.Fatalf("resume payload adoptado incompleto: %s", sesion.ResumePayloadJSON)
	}
	if !strings.Contains(sesion.ResumenContinuidad, "Contexto adoptado por Orquesta") {
		t.Fatalf("resumen continuidad no adoptado: %s", sesion.ResumenContinuidad)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" || !strings.Contains(orders[0].PayloadJSON, `"motivo":"adopt_context"`) {
		t.Fatalf("runtime orders inesperadas tras adopcion: %+v", orders)
	}
}

func TestAPIAgenteAdoptarContextoNoDuplicaRuntimeExternoActivo(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	conectorID, err := db.UpsertConector(&db.Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("upsert conector: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "autofirmav2",
		Nombre:  "AutofirmaV2",
		RutaAbs: filepath.Join(tmp, "AutofirmaV2"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.ActivarAsignacion("Codex1", proyectoID, "frente principal"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ConectorID:         &conectorID,
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "AutofirmaV2"),
		Herramienta:        "codex-remote",
		ExternalSessionID:  "sess-live",
		ResumenContinuidad: "sesion externa viva",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiAgenteAdoptarContextoRequest{
		Agente:            "Codex1",
		Proyecto:          "autofirmav2",
		Conector:          "codex-remote",
		CWD:               filepath.Join(tmp, "AutofirmaV2"),
		ExternalSessionID: "sess-live",
		Resumen:           "seguir este mismo proyecto",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/adoptar-contexto", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status adoptar contexto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteAdoptarContextoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode adoptar contexto: %v", err)
	}
	if resp.RuntimeOrderID != nil {
		t.Fatalf("no deberia encolar start sobre runtime externo vivo: %+v", resp)
	}
	agente := "Codex1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia haber runtime orders nuevas: %+v", orders)
	}
}

func TestAPIPropuestaReabrirYRepararVotos(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &db.Propuesta{
		Titulo:       "Propuesta API",
		Descripcion:  "Reabrir y reparar desde API",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if _, err := db.DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor2"); err != nil {
		t.Fatalf("delete voto faltante: %v", err)
	}
	if err := db.CerrarPropuesta(propuesta.Codigo, string(db.PropuestaBacklog), "alberto"); err != nil {
		t.Fatalf("cerrar propuesta: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	reabrirBody, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion: "reabrir",
		Agente: "alberto",
	})
	recReabrir := httptest.NewRecorder()
	reqReabrir := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(reabrirBody))
	mux.ServeHTTP(recReabrir, reqReabrir)
	if recReabrir.Code != http.StatusOK {
		t.Fatalf("status reabrir inesperado: %d body=%s", recReabrir.Code, recReabrir.Body.String())
	}

	reabierta, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta reabierta: %v", err)
	}
	if reabierta.Estado != db.PropuestaAbierta {
		t.Fatalf("estado inesperado tras reabrir: %s", reabierta.Estado)
	}

	if _, err := db.DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1"); err != nil {
		t.Fatalf("delete segundo voto faltante: %v", err)
	}

	repararBody, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion: "reparar_votos",
		Agente: "alberto",
	})
	recReparar := httptest.NewRecorder()
	reqReparar := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(repararBody))
	mux.ServeHTTP(recReparar, reqReparar)
	if recReparar.Code != http.StatusOK {
		t.Fatalf("status reparar inesperado: %d body=%s", recReparar.Code, recReparar.Body.String())
	}

	var total int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1").Scan(&total); err != nil {
		t.Fatalf("contar voto reparado: %v", err)
	}
	if total != 1 {
		t.Fatalf("conteo inesperado tras reparar via API: %d", total)
	}
}

func TestAPIGobernanzaCatalogoEOverrides(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "ctx-proyecto",
		Nombre:  "ctx-proyecto",
		RutaAbs: filepath.Join(t.TempDir(), "ctx-proyecto"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.UpsertRegla(&db.Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "server-first",
		Descripcion: "usar API",
		Activa:      true,
	}); err != nil {
		t.Fatalf("upsert regla: %v", err)
	}
	skillID, err := db.UpsertSkill(&db.Skill{
		TipoAgente:  "programador",
		Nombre:      "docker-build",
		Descripcion: "build",
		CuandoUsar:  "siempre",
		Prioridad:   10,
		Activa:      false,
	})
	if err != nil {
		t.Fatalf("upsert skill: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	overrideBody, _ := json.Marshal(apiGovernanceOverrideSaveRequest{
		Actor:      "alberto",
		TipoAgente: "programador",
		ScopeTipo:  db.GovernanceScopeProyecto,
		ScopeRef:   "ctx-proyecto",
		Entidad:    db.GovernanceEntitySkill,
		EntidadID:  skillID,
		Accion:     db.GovernanceActionEnable,
	})
	recOverride := httptest.NewRecorder()
	reqOverride := httptest.NewRequest(http.MethodPost, "/api/gobernanza/overrides", bytes.NewReader(overrideBody))
	mux.ServeHTTP(recOverride, reqOverride)
	if recOverride.Code != http.StatusCreated {
		t.Fatalf("status override inesperado: %d body=%s", recOverride.Code, recOverride.Body.String())
	}

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/gobernanza/overrides?scope_tipo=proyecto&scope_ref=ctx-proyecto&tipo_agente=programador", nil)
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("status listar overrides inesperado: %d body=%s", recList.Code, recList.Body.String())
	}
	var overridesResp apiGovernanceOverridesResponse
	if err := json.Unmarshal(recList.Body.Bytes(), &overridesResp); err != nil {
		t.Fatalf("decode overrides: %v", err)
	}
	if len(overridesResp.Overrides) != 1 || overridesResp.Overrides[0].EntidadID != skillID {
		t.Fatalf("overrides inesperados: %+v", overridesResp.Overrides)
	}

	recCatalogo := httptest.NewRecorder()
	reqCatalogo := httptest.NewRequest(http.MethodGet, "/api/gobernanza/catalogo?agente=Codex1&proyecto=ctx-proyecto", nil)
	mux.ServeHTTP(recCatalogo, reqCatalogo)
	if recCatalogo.Code != http.StatusOK {
		t.Fatalf("status catalogo inesperado: %d body=%s", recCatalogo.Code, recCatalogo.Body.String())
	}
	var catalogoResp apiGovernanceCatalogResponse
	if err := json.Unmarshal(recCatalogo.Body.Bytes(), &catalogoResp); err != nil {
		t.Fatalf("decode catalogo: %v", err)
	}
	if catalogoResp.Catalogo == nil {
		t.Fatalf("catalogo nil")
	}
	if catalogoResp.Catalogo.ProyectoID == nil || *catalogoResp.Catalogo.ProyectoID != proyectoID {
		t.Fatalf("proyecto_id inesperado en catalogo: %+v", catalogoResp.Catalogo)
	}
	if catalogoResp.Catalogo.ResolucionActual != "rol+proyecto" {
		t.Fatalf("resolucion inesperada: %q", catalogoResp.Catalogo.ResolucionActual)
	}
	foundSkill := false
	for _, skill := range catalogoResp.Catalogo.Skills {
		if skill != nil && skill.ID == skillID {
			foundSkill = true
			break
		}
	}
	if !foundSkill {
		t.Fatalf("el catalogo efectivo deberia incluir la skill habilitada por override: %+v", catalogoResp.Catalogo.Skills)
	}
}

func TestAPIStatusIncluyeVotosDePropuestasAbiertas(t *testing.T) {
	prepararDBTemporalCmd(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &db.Propuesta{
		Titulo:       "Status con votos",
		Descripcion:  "Verificar carga de votos en /api/status",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	p, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if _, err := db.Votar(p.ID, "revisor1", db.VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("votar revisor1: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if len(resp.PropuestasAbiertas) == 0 {
		t.Fatalf("se esperaban propuestas abiertas en status")
	}
	var encontrada *db.Propuesta
	for _, item := range resp.PropuestasAbiertas {
		if item != nil && item.Codigo == propuesta.Codigo {
			encontrada = item
			break
		}
	}
	if encontrada == nil {
		t.Fatalf("no se encontró la propuesta %s en status", propuesta.Codigo)
	}
	if len(encontrada.Votos) == 0 {
		t.Fatalf("status no incluyó votos para la propuesta abierta")
	}
}

func TestAPIPropuestaActualizarAnexaDescripcion(t *testing.T) {
	prepararDBTemporalCmd(t)

	propuesta := &db.Propuesta{
		Codigo:       "OP-910",
		Titulo:       "Propuesta API update",
		Descripcion:  "Base",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := db.CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	appendText := " + docs"
	body, _ := json.Marshal(apiPropuestaAccionRequest{
		Accion:     "actualizar",
		Agente:     "alberto",
		AnexarDesc: &appendText,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/propuestas/"+propuesta.Codigo+"/accion", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status actualizar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := db.GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta actualizada: %v", err)
	}
	if updated.Descripcion != "Base + docs" {
		t.Fatalf("descripcion inesperada: %q", updated.Descripcion)
	}
}

func TestAPILenguajePoliticaMatrizYResolver(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.SetLanguagePolicy(&db.LanguagePolicy{
		DefaultLanguage:          "es",
		DocumentationMultilang:   true,
		AppsMultilang:            true,
		DocumentationDefaultLang: "es",
		AppsDefaultLang:          "en",
		AllowedLanguages:         []string{"es", "en", "fr"},
		Notes:                    "politica de prueba",
	}, "Codex3"); err != nil {
		t.Fatalf("SetLanguagePolicy: %v", err)
	}
	if _, err := db.SetLanguageMatrixEntry("project", "orquestador", "apps", "fr", "demo", "Codex3"); err != nil {
		t.Fatalf("SetLanguageMatrixEntry: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recPolicy := httptest.NewRecorder()
	reqPolicy := httptest.NewRequest(http.MethodGet, "/api/lenguaje/politica", nil)
	mux.ServeHTTP(recPolicy, reqPolicy)
	if recPolicy.Code != http.StatusOK {
		t.Fatalf("status politica inesperado: %d body=%s", recPolicy.Code, recPolicy.Body.String())
	}
	var policyResp struct {
		Politica *db.LanguagePolicy `json:"politica"`
	}
	if err := json.Unmarshal(recPolicy.Body.Bytes(), &policyResp); err != nil {
		t.Fatalf("decode politica: %v", err)
	}
	if policyResp.Politica == nil || policyResp.Politica.AppsDefaultLang != "en" {
		t.Fatalf("politica inesperada: %+v", policyResp.Politica)
	}

	recMatrix := httptest.NewRecorder()
	reqMatrix := httptest.NewRequest(http.MethodGet, "/api/lenguaje/matriz", nil)
	mux.ServeHTTP(recMatrix, reqMatrix)
	if recMatrix.Code != http.StatusOK {
		t.Fatalf("status matriz inesperado: %d body=%s", recMatrix.Code, recMatrix.Body.String())
	}
	var matrixResp struct {
		Matriz []*db.LanguageMatrixEntry `json:"matriz"`
	}
	if err := json.Unmarshal(recMatrix.Body.Bytes(), &matrixResp); err != nil {
		t.Fatalf("decode matriz: %v", err)
	}
	if len(matrixResp.Matriz) != 1 || matrixResp.Matriz[0].Language != "fr" {
		t.Fatalf("matriz inesperada: %+v", matrixResp.Matriz)
	}

	recResolve := httptest.NewRecorder()
	reqResolve := httptest.NewRequest(http.MethodGet, "/api/lenguaje/resolver?proyecto=orquestador&contexto=apps", nil)
	mux.ServeHTTP(recResolve, reqResolve)
	if recResolve.Code != http.StatusOK {
		t.Fatalf("status resolver inesperado: %d body=%s", recResolve.Code, recResolve.Body.String())
	}
	var resolveResp struct {
		Resolucion *db.LanguageResolution `json:"resolucion"`
	}
	if err := json.Unmarshal(recResolve.Body.Bytes(), &resolveResp); err != nil {
		t.Fatalf("decode resolucion: %v", err)
	}
	if resolveResp.Resolucion == nil || resolveResp.Resolucion.Idioma != "fr" {
		t.Fatalf("resolucion inesperada: %+v", resolveResp.Resolucion)
	}
}

func TestAPIProyectoDescubrirYGestionAgentes(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoDir := filepath.Join(tmp, "demo")
	if err := os.MkdirAll(filepath.Join(proyectoDir, ".git"), 0o755); err != nil {
		t.Fatalf("crear proyecto demo: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	descubrirBody, _ := json.Marshal(apiProyectoDescubrirRequest{Ruta: tmp})
	recDescubrir := httptest.NewRecorder()
	reqDescubrir := httptest.NewRequest(http.MethodPost, "/api/proyectos/descubrir", bytes.NewReader(descubrirBody))
	mux.ServeHTTP(recDescubrir, reqDescubrir)
	if recDescubrir.Code != http.StatusCreated {
		t.Fatalf("status descubrir inesperado: %d body=%s", recDescubrir.Code, recDescubrir.Body.String())
	}

	recProyecto := httptest.NewRecorder()
	reqProyecto := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo", nil)
	mux.ServeHTTP(recProyecto, reqProyecto)
	if recProyecto.Code != http.StatusOK {
		t.Fatalf("status proyecto inesperado: %d body=%s", recProyecto.Code, recProyecto.Body.String())
	}

	actualizarBody, _ := json.Marshal(apiProyectoActualizarRequest{
		RutaAbs: filepath.Join(tmp, "demo-renombrado"),
	})
	recActualizar := httptest.NewRecorder()
	reqActualizar := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo", bytes.NewReader(actualizarBody))
	mux.ServeHTTP(recActualizar, reqActualizar)
	if recActualizar.Code != http.StatusOK {
		t.Fatalf("status actualizar proyecto inesperado: %d body=%s", recActualizar.Code, recActualizar.Body.String())
	}
	var actualizarResp map[string]*db.Proyecto
	if err := json.Unmarshal(recActualizar.Body.Bytes(), &actualizarResp); err != nil {
		t.Fatalf("decode actualizar proyecto: %v", err)
	}
	if actualizarResp["proyecto"] == nil || actualizarResp["proyecto"].RutaAbs != filepath.Join(tmp, "demo-renombrado") {
		t.Fatalf("proyecto actualizado inesperado: %+v", actualizarResp["proyecto"])
	}

	agenteBody, _ := json.Marshal(apiAgenteRequest{Nombre: "temporal", Rol: "programador"})
	recAlta := httptest.NewRecorder()
	reqAlta := httptest.NewRequest(http.MethodPost, "/api/agentes", bytes.NewReader(agenteBody))
	mux.ServeHTTP(recAlta, reqAlta)
	if recAlta.Code != http.StatusCreated {
		t.Fatalf("status alta agente inesperado: %d body=%s", recAlta.Code, recAlta.Body.String())
	}

	agenteAutoBody, _ := json.Marshal(apiAgenteRequest{Proveedor: "claude", Rol: "programador"})
	recAltaAuto := httptest.NewRecorder()
	reqAltaAuto := httptest.NewRequest(http.MethodPost, "/api/agentes", bytes.NewReader(agenteAutoBody))
	mux.ServeHTTP(recAltaAuto, reqAltaAuto)
	if recAltaAuto.Code != http.StatusCreated {
		t.Fatalf("status alta agente auto inesperado: %d body=%s", recAltaAuto.Code, recAltaAuto.Body.String())
	}
	var altaAutoResp map[string]any
	if err := json.Unmarshal(recAltaAuto.Body.Bytes(), &altaAutoResp); err != nil {
		t.Fatalf("decode alta auto: %v", err)
	}
	if altaAutoResp["nombre"] != "Claude1" {
		t.Fatalf("nombre auto inesperado: %+v", altaAutoResp)
	}

	recRetirar := httptest.NewRecorder()
	reqRetirar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/retirar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRetirar, reqRetirar)
	if recRetirar.Code != http.StatusOK {
		t.Fatalf("status retirar inesperado: %d body=%s", recRetirar.Code, recRetirar.Body.String())
	}

	recRehabilitar := httptest.NewRecorder()
	reqRehabilitar := httptest.NewRequest(http.MethodPost, "/api/agentes/temporal/rehabilitar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recRehabilitar, reqRehabilitar)
	if recRehabilitar.Code != http.StatusOK {
		t.Fatalf("status rehabilitar inesperado: %d body=%s", recRehabilitar.Code, recRehabilitar.Body.String())
	}
}

func TestAPIProyectoFusionar(t *testing.T) {
	prepararDBTemporalCmd(t)

	destinoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear destino: %v", err)
	}
	origenID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("crear origen: %v", err)
	}
	if _, err := db.CrearTarea(&db.Tarea{
		Titulo:     "Tarea origen",
		ProyectoID: &origenID,
		Modulo:     "orquestacion",
		Prioridad:  db.PrioridadAlta,
		CreadoPor:  "Codex1",
	}); err != nil {
		t.Fatalf("crear tarea origen: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoFusionRequest{Origen: "orquesta", ArchivarOrigen: true})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquestador/fusionar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status fusionar proyecto inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiProyectoFusionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode fusion proyecto: %v", err)
	}
	if resp.Resultado == nil || resp.Resultado.DestinoID != destinoID {
		t.Fatalf("resultado de fusion inesperado: %+v", resp.Resultado)
	}

	tarea, err := db.GetTarea(1)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.ProyectoID == nil || *tarea.ProyectoID != destinoID {
		t.Fatalf("tarea no movida al destino: %+v", tarea.ProyectoID)
	}
	origenArchivado, err := db.GetProyecto(resp.Resultado.SlugArchivado)
	if err != nil {
		t.Fatalf("get origen archivado: %v", err)
	}
	if origenArchivado == nil || origenArchivado.Activo {
		t.Fatalf("origen no archivado: %+v", origenArchivado)
	}
}

func TestAPIProyectoFusionarRechazaDesactivarArchivoOrigen(t *testing.T) {
	prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "orquestador",
		RutaAbs: "/tmp/orquestador",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear destino: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: "/tmp/orquesta",
		Tipo:    db.ProyectoRaiz,
		Activo:  true,
	}); err != nil {
		t.Fatalf("crear origen: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoFusionRequest{Origen: "orquesta", ArchivarOrigen: false})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/orquestador/fusionar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status inesperado al desactivar archivado: %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "todavía no está soportado") {
		t.Fatalf("error inesperado: %s", rec.Body.String())
	}
}

func TestAPIProyectoFabricarApp(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(apiProyectoFabricarAppRequest{
		Tipo:        "web_api",
		Nombre:      "Demo App",
		Descripcion: "Aplicacion demo",
		Frontend:    true,
		API:         true,
		Docker:      true,
		I18n:        true,
		Idiomas:     []string{"es", "en"},
		Por:         "Codex3",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/fabricar-app", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status fabricar-app inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiProyectoFabricarAppResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode fabricar-app: %v", err)
	}
	if !resp.OK || resp.Slug != "demo-app" || resp.Created < 4 {
		t.Fatalf("respuesta fabricar-app inesperada: %+v", resp)
	}

	tareas, err := db.ListarTareas(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar tareas: %v", err)
	}
	if len(tareas) != resp.Created {
		t.Fatalf("tareas creadas=%d respuesta=%d", len(tareas), resp.Created)
	}
}

func TestAPIProyectoOperacionGetYPost(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/operacion", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("status get operacion inesperado: %d body=%s", recGet.Code, recGet.Body.String())
	}
	var getResp apiProyectoOperacionResponse
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get operacion: %v", err)
	}
	if getResp.Operacion == nil || getResp.Operacion.EstadoOperativo != db.ProyectoOperativoActivo {
		t.Fatalf("operacion inicial inesperada: %+v", getResp.Operacion)
	}

	body, _ := json.Marshal(apiProyectoOperacionSetRequest{
		EstadoOperativo:  string(db.ProyectoOperativoEsperandoHumano),
		Motivo:           "esperando aprobacion",
		ObjetivoPct:      60,
		MinAgentes:       1,
		MaxAgentes:       3,
		Prioridad:        250,
		ResumeAutomatico: true,
	})
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/operacion", bytes.NewReader(body))
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("status post operacion inesperado: %d body=%s", recPost.Code, recPost.Body.String())
	}
	var postResp apiProyectoOperacionResponse
	if err := json.Unmarshal(recPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode post operacion: %v", err)
	}
	if postResp.Operacion == nil || postResp.Operacion.EstadoOperativo != db.ProyectoOperativoEsperandoHumano || postResp.Operacion.ObjetivoPct != 60 {
		t.Fatalf("operacion guardada inesperada: %+v", postResp.Operacion)
	}
}

func TestAPIProyectoAutonomiaGetPostYCiclos(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recGet := httptest.NewRecorder()
	reqGet := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/autonomia", nil)
	mux.ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("status get autonomia inesperado: %d body=%s", recGet.Code, recGet.Body.String())
	}
	var getResp apiProyectoAutonomiaResponse
	if err := json.Unmarshal(recGet.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get autonomia: %v", err)
	}
	if getResp.Policy == nil || getResp.Policy.EstadoAutonomia != db.AutonomiaProyectoActiva || getResp.Policy.Enabled {
		t.Fatalf("autonomia inicial inesperada: %+v", getResp.Policy)
	}

	body, _ := json.Marshal(apiProyectoAutonomiaSaveRequest{
		Enabled:              true,
		ObjetivoGeneral:      "terminar el proyecto sin intervención humana",
		DefinitionOfDoneJSON: `{"tests":"green"}`,
		MaxWorkers:           3,
		SupervisorAgente:     "CodexSupervisor",
		ReviewerAgente:       "CodexReview",
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      string(db.AutonomiaProyectoActiva),
	})
	recPost := httptest.NewRecorder()
	reqPost := httptest.NewRequest(http.MethodPost, "/api/proyectos/demo-app/autonomia", bytes.NewReader(body))
	mux.ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("status post autonomia inesperado: %d body=%s", recPost.Code, recPost.Body.String())
	}
	var postResp apiProyectoAutonomiaResponse
	if err := json.Unmarshal(recPost.Body.Bytes(), &postResp); err != nil {
		t.Fatalf("decode post autonomia: %v", err)
	}
	if postResp.Policy == nil || !postResp.Policy.Enabled || postResp.Policy.MaxWorkers != 3 {
		t.Fatalf("autonomia guardada inesperada: %+v", postResp.Policy)
	}
	if postResp.Policy.SupervisorAgente != "CodexSupervisor" || postResp.Policy.ReviewerAgente != "CodexReview" {
		t.Fatalf("agentes preferidos de autonomia inesperados: %+v", postResp.Policy)
	}

	if _, err := supervisionService.RegisterCycle("demo-app", supervisionapp.CycleInput{
		Kind:         "supervision",
		Agente:       "Codex3",
		InputJSON:    `{"reason":"periodic"}`,
		DecisionJSON: `{"decision":"seguir"}`,
		Resultado:    "ok",
	}); err != nil {
		t.Fatalf("RegisterCycle: %v", err)
	}

	recCycles := httptest.NewRecorder()
	reqCycles := httptest.NewRequest(http.MethodGet, "/api/proyectos/demo-app/autonomia/ciclos?kind=supervision&limit=5", nil)
	mux.ServeHTTP(recCycles, reqCycles)
	if recCycles.Code != http.StatusOK {
		t.Fatalf("status ciclos inesperado: %d body=%s", recCycles.Code, recCycles.Body.String())
	}
	var cyclesResp struct {
		Cycles []*db.AutonomiaCiclo `json:"cycles"`
	}
	if err := json.Unmarshal(recCycles.Body.Bytes(), &cyclesResp); err != nil {
		t.Fatalf("decode ciclos: %v", err)
	}
	if len(cyclesResp.Cycles) != 1 || cyclesResp.Cycles[0].Kind != "supervision" {
		t.Fatalf("ciclos inesperados: %+v", cyclesResp.Cycles)
	}
}

func TestAPIReviewGatesCrearListarYResolver(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "demo-app",
		Nombre:  "Demo App",
		RutaAbs: filepath.Join(tmp, "demo-app"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	createBody, _ := json.Marshal(apiReviewGateCreateRequest{
		Proyecto:       "demo-app",
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexReview",
		SeverityMax:    "medium",
		FindingsJSON:   `[{"kind":"coverage","detail":"falta regression"}]`,
	})
	recCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/review-gates", bytes.NewReader(createBody))
	mux.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("status create gate inesperado: %d body=%s", recCreate.Code, recCreate.Body.String())
	}
	var createResp struct {
		Gate *reviewapp.Gate `json:"gate"`
	}
	if err := json.Unmarshal(recCreate.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create gate: %v", err)
	}
	if createResp.Gate == nil || createResp.Gate.ID == 0 || createResp.Gate.Estado != reviewapp.GateStatePending {
		t.Fatalf("gate creada inesperada: %+v", createResp.Gate)
	}

	recList := httptest.NewRecorder()
	reqList := httptest.NewRequest(http.MethodGet, "/api/review-gates?proyecto=demo-app&estado=pendiente&limit=5", nil)
	mux.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("status list gates inesperado: %d body=%s", recList.Code, recList.Body.String())
	}
	var listResp struct {
		Gates []*reviewapp.Gate `json:"gates"`
	}
	if err := json.Unmarshal(recList.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list gates: %v", err)
	}
	if len(listResp.Gates) != 1 || listResp.Gates[0].ID != createResp.Gate.ID {
		t.Fatalf("listado de gates inesperado: %+v", listResp.Gates)
	}

	resolveBody, _ := json.Marshal(apiReviewGateResolveRequest{
		Estado:         reviewapp.GateStateApproved,
		ReviewerAgente: "CodexReview",
		FindingsJSON:   `[]`,
	})
	recResolve := httptest.NewRecorder()
	reqResolve := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/review-gates/%d/resolver", createResp.Gate.ID), bytes.NewReader(resolveBody))
	mux.ServeHTTP(recResolve, reqResolve)
	if recResolve.Code != http.StatusOK {
		t.Fatalf("status resolve gate inesperado: %d body=%s", recResolve.Code, recResolve.Body.String())
	}
	var resolveResp struct {
		Gate *reviewapp.Gate `json:"gate"`
	}
	if err := json.Unmarshal(recResolve.Body.Bytes(), &resolveResp); err != nil {
		t.Fatalf("decode resolve gate: %v", err)
	}
	if resolveResp.Gate == nil || resolveResp.Gate.Estado != reviewapp.GateStateApproved || resolveResp.Gate.ResolvedAt == nil {
		t.Fatalf("gate resuelta inesperada: %+v", resolveResp.Gate)
	}
}

func TestAPIAgenteControlPlaneAcciones(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := db.IniciarSesion("Codex1"); err != nil {
		t.Fatalf("iniciar sesion origen: %v", err)
	}
	if _, err := db.IniciarSesion("Codex2"); err != nil {
		t.Fatalf("iniciar sesion destino: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Handoff API",
		Modulo:    "orquestador",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := db.IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	pausaBody, _ := json.Marshal(apiAgentePausarRequest{Agente: "Codex1", Minutos: 15, Motivo: "rate limit"})
	recPausa := httptest.NewRecorder()
	reqPausa := httptest.NewRequest(http.MethodPost, "/api/agente/pausar", bytes.NewReader(pausaBody))
	mux.ServeHTTP(recPausa, reqPausa)
	if recPausa.Code != http.StatusOK {
		t.Fatalf("status pausar inesperado: %d body=%s", recPausa.Code, recPausa.Body.String())
	}

	recReset := httptest.NewRecorder()
	reqReset := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex1/reset-reanimacion", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("status reset inesperado: %d body=%s", recReset.Code, recReset.Body.String())
	}

	handoffBody, _ := json.Marshal(apiRuntimeHandoffRequest{
		AgenteOrigen:  "Codex1",
		AgenteDestino: "Codex2",
		TareaID:       tareaID,
		Motivo:        "cambio de turno",
		Resumen:       "seguir desde api",
	})
	recHandoff := httptest.NewRecorder()
	reqHandoff := httptest.NewRequest(http.MethodPost, "/api/agente/handoff", bytes.NewReader(handoffBody))
	mux.ServeHTTP(recHandoff, reqHandoff)
	if recHandoff.Code != http.StatusCreated {
		t.Fatalf("status handoff inesperado: %d body=%s", recHandoff.Code, recHandoff.Body.String())
	}

	var resp struct {
		ID      int64 `json:"id"`
		OrderID int64 `json:"order_id"`
	}
	if err := json.Unmarshal(recHandoff.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode handoff: %v", err)
	}
	if resp.OrderID == 0 && resp.ID == 0 {
		t.Fatalf("order id inesperado: %d", resp.OrderID)
	}

	recEliminar := httptest.NewRecorder()
	reqEliminar := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex3/eliminar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(recEliminar, reqEliminar)
	if recEliminar.Code != http.StatusOK {
		t.Fatalf("status eliminar inesperado: %d body=%s", recEliminar.Code, recEliminar.Body.String())
	}
}

func TestAPIAgenteFusionarCreaRespaldoYMueveReferencias(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("registrar codex1: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "Fusion API",
		Modulo:    "orquestador",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "codex1",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	backupDir := filepath.Join(tmp, "backups")
	body, _ := json.Marshal(apiAgenteFusionRequest{
		Destino:          "Codex1",
		DestinoRespaldo:  backupDir,
		EtiquetaRespaldo: "fusion-api",
		Retener:          2,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/codex1/fusionar", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status fusion inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp apiAgenteFusionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode fusion: %v", err)
	}
	if resp.RutaRespaldo == "" {
		t.Fatalf("ruta respaldo vacia")
	}
	if _, err := os.Stat(resp.RutaRespaldo); err != nil {
		t.Fatalf("respaldo no creado: %v", err)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" {
		t.Fatalf("agente tarea inesperado tras fusion: %+v", tarea.Agente)
	}
	var agentesOrigen int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM agentes WHERE nombre = ?`, "codex1").Scan(&agentesOrigen); err != nil {
		t.Fatalf("count agente origen: %v", err)
	}
	if agentesOrigen != 0 {
		t.Fatalf("el agente origen deberia haberse eliminado")
	}
}

func TestAPIAgenteEliminarBloqueaTareasActivas(t *testing.T) {
	prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar Codex5: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:    "No borrar agente con trabajo vivo",
		Modulo:    "controlplane",
		Prioridad: db.PrioridadAlta,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := db.TomarTarea(tareaID, "Codex5"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agentes/Codex5/eliminar", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status eliminar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var apiErr apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if !strings.Contains(strings.ToLower(apiErr.Error), "tarea") {
		t.Fatalf("mensaje inesperado: %s", apiErr.Error)
	}
	if _, err := db.GetAgente("Codex5"); err != nil {
		t.Fatalf("el agente no deberia haberse eliminado: %v", err)
	}
}

func TestAPIAgenteTickAutoPausaPorAgotamiento(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-tick-auto-pausa",
		ResumenContinuidad: "tick final",
		Branch:             "main",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"agente":     "Codex1",
		"proyecto":   "orquestador",
		"cuota_pct":  3,
		"finalizado": true,
		"motivo":     "rate limit de proveedor",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agente/tick", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status tick inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	agente, err := db.GetAgente("Codex1")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.EstadoCuota != "enfriamiento" {
		t.Fatalf("estado_cuota inesperado: %s", agente.EstadoCuota)
	}
	if agente.ReanimarAt == nil {
		t.Fatalf("reanimar_at no deberia ser nil")
	}
	if agente.MotivoPausa == "" || agente.MotivoPausa != "Auto-pausa por agotamiento: rate limit de proveedor" {
		t.Fatalf("motivo_pausa inesperado: %q", agente.MotivoPausa)
	}
}

func TestAPIAgentePrepararDevuelveBundle(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if _, err := db.UpsertEntidadMemoria(&db.EntidadMemoria{
		Nombre:        "Core_API",
		Tipo:          string(db.EntidadMemoriaAPI),
		ValorJSON:     `{"version":"v2"}`,
		MetadataJSON:  `{"fuente":"manual"}`,
		VerificadoPor: "Codex1",
		ProyectoID:    &proyecto.ID,
	}); err != nil {
		t.Fatalf("upsert entidad memoria: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agente/preparar?agente=Codex1&proyecto=orquestador", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status preparar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var out agentePrepararOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode preparar: %v", err)
	}
	if out.Agente != "Codex1" {
		t.Fatalf("agente inesperado: %s", out.Agente)
	}
	if out.Proyecto.Slug != "orquestador" {
		t.Fatalf("proyecto inesperado: %s", out.Proyecto.Slug)
	}
	if out.Conector.Slug != "codex-cli" {
		t.Fatalf("conector inesperado: %s", out.Conector.Slug)
	}
	if out.Plan == nil {
		t.Fatalf("plan no deberia ser nil")
	}
	if strings.TrimSpace(out.BootstrapPrompt) == "" || out.Plan.BootstrapPrompt != out.BootstrapPrompt {
		t.Fatalf("bootstrap prompt inesperado: top=%q plan=%q", out.BootstrapPrompt, out.Plan.BootstrapPrompt)
	}
	if out.EstadoCuota == "" {
		t.Fatalf("estado_cuota no deberia venir vacio")
	}
	if len(out.Memoria) != 1 || out.Memoria[0].Nombre != "Core_API" {
		t.Fatalf("memoria inesperada: %+v", out.Memoria)
	}
}

func TestAPIAgentePrepararIntegraBootstrapRuntime(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := db.EnsureCapacidadModeloBaseCodex(); err != nil {
		t.Fatalf("seed capacidad/modelo base: %v", err)
	}
	if _, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	}); err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	proyecto, err := db.GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyecto.ID,
		CWD:                filepath.Join(tmp, "orquestador", "sesion-previa"),
		Herramienta:        "codex-cli",
		ResumenContinuidad: "continuidad previa",
		ResumePayloadJSON:  `{"previo":true}`,
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	handoffPayload, err := json.Marshal(db.HandoffPayload{
		AgenteOrigen:       "Codex0",
		AgenteDestino:      "Codex1",
		Motivo:             "traspaso",
		ResumenContinuidad: "handoff listo",
		ExternalSessionID:  "sess-handoff",
	})
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	orderID, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyecto.ID,
		Tipo:        "handoff",
		PayloadJSON: string(handoffPayload),
	})
	if err != nil {
		t.Fatalf("encolar handoff: %v", err)
	}
	if _, err := db.EncolarRuntimeOrder(&db.RuntimeOrder{
		Agente:     "Codex1",
		ProyectoID: &proyecto.ID,
		Tipo:       "nudge",
		PayloadJSON: `{
			"to_agente":"Codex1",
			"from_agente":"Coordinador",
			"kind":"handoff_note",
			"texto":"revisa el checkpoint"
		}`,
	}); err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}
	if _, err := db.ProcesarRuntimeOrdersBasicasBatch(); err != nil {
		t.Fatalf("procesar runtime orders basicas: %v", err)
	}
	if _, err := db.CrearRuntimeCheckpoint(&db.RuntimeCheckpoint{
		Agente:         "Codex1",
		ProyectoID:     &proyecto.ID,
		CheckpointKind: "handoff_prepare",
		Resumen:        "checkpoint reciente",
		Branch:         "feature/bootstrap",
		CWD:            filepath.Join(tmp, "orquestador", "checkpoint"),
		PayloadJSON:    `{"archivos":["a.go","b.go"]}`,
		ResumeStrategy: "resumen_y_payload",
		Source:         "test:bootstrap",
	}); err != nil {
		t.Fatalf("crear checkpoint: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, base_ref, estado, motivo)
		VALUES (?,?,?,?,?,?,?,?)`,
		proyecto.ID,
		"Codex1",
		"wt-codex1",
		filepath.Join(tmp, "orquestador", ".orquesta-worktrees", "wt-codex1"),
		"feature/wt-codex1",
		"HEAD",
		"activa",
		"continuidad",
	); err != nil {
		t.Fatalf("crear worktree: %v", err)
	}
	if _, err := db.CrearPropuesta(&db.Propuesta{
		Codigo:       "OP-901",
		Titulo:       "Revisar autenticacion",
		Descripcion:  "Pendiente de votacion",
		ProyectoID:   &proyecto.ID,
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "orquesta",
	}); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	reglaID, err := db.UpsertRegla(&db.Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "regla-api-prepare-governance",
		Descripcion: "forzar reconcile de governance en prepare",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("upsert regla: %v", err)
	}
	if _, err := db.GuardarGovernanceOverride("tester", &db.GovernanceOverride{
		TipoAgente: "programador",
		ScopeTipo:  db.GovernanceScopeProyecto,
		ScopeRef:   "orquestador",
		Entidad:    db.GovernanceEntityRegla,
		EntidadID:  reglaID,
		Accion:     db.GovernanceActionDisable,
	}); err != nil {
		t.Fatalf("guardar override gobernanza: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/agente/preparar?agente=Codex1&proyecto=orquestador", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status preparar inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var out agentePrepararOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode preparar: %v", err)
	}
	if out.Bootstrap == nil {
		t.Fatalf("bootstrap no deberia ser nil")
	}
	if out.Bootstrap.Order == nil || out.Bootstrap.Order.Tipo != "handoff" {
		t.Fatalf("order bootstrap inesperada: %+v", out.Bootstrap.Order)
	}
	if len(out.Bootstrap.Mailbox) == 0 {
		t.Fatalf("mailbox bootstrap vacio: %+v", out.Bootstrap.Mailbox)
	}
	hasHandoffNote := false
	hasGovernanceRefresh := false
	for _, msg := range out.Bootstrap.Mailbox {
		if msg == nil {
			continue
		}
		switch msg.Kind {
		case "handoff_note":
			hasHandoffNote = true
		case db.MailboxKindGovernanceRefresh:
			hasGovernanceRefresh = true
		}
	}
	if !hasHandoffNote {
		t.Fatalf("mailbox bootstrap sin handoff_note: %+v", out.Bootstrap.Mailbox)
	}
	if !hasGovernanceRefresh {
		t.Fatalf("mailbox bootstrap sin governance_refresh: %+v", out.Bootstrap.Mailbox)
	}
	if out.Bootstrap.Checkpoint == nil || out.Bootstrap.Checkpoint.Branch != "feature/bootstrap" {
		t.Fatalf("checkpoint bootstrap inesperado: %+v", out.Bootstrap.Checkpoint)
	}
	if out.Plan == nil || out.Plan.Modo != "resume" {
		t.Fatalf("plan de arranque inesperado: %+v", out.Plan)
	}
	if strings.TrimSpace(out.BootstrapPrompt) == "" || out.Plan.BootstrapPrompt != out.BootstrapPrompt {
		t.Fatalf("bootstrap prompt inesperado: top=%q plan=%q", out.BootstrapPrompt, out.Plan.BootstrapPrompt)
	}
	if out.Plan.WorkingDir != filepath.Join(tmp, "orquestador", "sesion-previa") {
		t.Fatalf("working_dir inesperado: %s", out.Plan.WorkingDir)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "handoff listo") {
		t.Fatalf("continuity prompt sin handoff: %s", out.Plan.ContinuityPrompt)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "runtime_order=handoff") || !strings.Contains(out.Plan.ContinuityPrompt, "mailbox=2") || !strings.Contains(out.Plan.ContinuityPrompt, "checkpoint#") {
		t.Fatalf("continuity prompt sin resumen bootstrap: %s", out.Plan.ContinuityPrompt)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "project_context") || !strings.Contains(out.Plan.ContinuityPrompt, "Worktree activa en feature/wt-codex1") || !strings.Contains(out.Plan.ContinuityPrompt, "1 propuesta(s) abiertas") {
		t.Fatalf("continuity prompt sin mapa operativo: %s", out.Plan.ContinuityPrompt)
	}
	if !strings.Contains(out.Plan.ContinuityPrompt, "governance_catalog") || !strings.Contains(out.Plan.ContinuityPrompt, "Catálogo efectivo") {
		t.Fatalf("continuity prompt sin gobernanza efectiva: %s", out.Plan.ContinuityPrompt)
	}

	order, err := db.GetRuntimeOrder(orderID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if order == nil || order.Estado != "pendiente" {
		t.Fatalf("runtime order no deberia consumirse tras preparar: %+v", order)
	}

	estadoConsumido := "pendiente"
	toAgente := "Codex1"
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &toAgente,
		ProyectoID: &proyecto.ID,
		Estado:     &estadoConsumido,
	})
	if err != nil {
		t.Fatalf("listar mailbox pendiente: %v", err)
	}
	if len(mailbox) != len(out.Bootstrap.Mailbox) {
		t.Fatalf("mailbox pendiente inesperado: got=%d want=%d %+v", len(mailbox), len(out.Bootstrap.Mailbox), mailbox)
	}
}

func TestAPIProyectosLecturaUsaRutaEfectivaAunquePersistaRutaHistorica(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	rutaHistorica := filepath.Join(tmp, "historico", "orquestador")
	rutaActual := filepath.Join(tmp, "actual", "orquesta")
	if err := os.MkdirAll(filepath.Join(rutaActual, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir ruta actual: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rutaActual, "go.mod"), []byte("module orquesta\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: rutaHistorica,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(rutaActual, "cmd"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	recProyecto := httptest.NewRecorder()
	reqProyecto := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador", nil)
	mux.ServeHTTP(recProyecto, reqProyecto)
	if recProyecto.Code != http.StatusOK {
		t.Fatalf("status proyecto inesperado: %d body=%s", recProyecto.Code, recProyecto.Body.String())
	}
	var proyectoResp apiProyectoResponse
	if err := json.Unmarshal(recProyecto.Body.Bytes(), &proyectoResp); err != nil {
		t.Fatalf("decode proyecto: %v", err)
	}
	if proyectoResp.Proyecto == nil || proyectoResp.Proyecto.RutaAbs != rutaActual {
		t.Fatalf("proyecto con ruta inesperada: %+v", proyectoResp.Proyecto)
	}

	recListado := httptest.NewRecorder()
	reqListado := httptest.NewRequest(http.MethodGet, "/api/proyectos", nil)
	mux.ServeHTTP(recListado, reqListado)
	if recListado.Code != http.StatusOK {
		t.Fatalf("status listado inesperado: %d body=%s", recListado.Code, recListado.Body.String())
	}
	var listadoResp apiProyectosResponse
	if err := json.Unmarshal(recListado.Body.Bytes(), &listadoResp); err != nil {
		t.Fatalf("decode listado: %v", err)
	}
	if len(listadoResp.Proyectos) != 1 || listadoResp.Proyectos[0] == nil || listadoResp.Proyectos[0].RutaAbs != rutaActual {
		t.Fatalf("listado con ruta inesperada: %+v", listadoResp.Proyectos)
	}

	recOverview := httptest.NewRecorder()
	reqOverview := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/overview", nil)
	mux.ServeHTTP(recOverview, reqOverview)
	if recOverview.Code != http.StatusOK {
		t.Fatalf("status overview inesperado: %d body=%s", recOverview.Code, recOverview.Body.String())
	}
	var overviewResp apiProyectoOverviewResponse
	if err := json.Unmarshal(recOverview.Body.Bytes(), &overviewResp); err != nil {
		t.Fatalf("decode overview: %v", err)
	}
	if overviewResp.Overview == nil || overviewResp.Overview.Proyecto == nil || overviewResp.Overview.Proyecto.RutaAbs != rutaActual {
		t.Fatalf("overview con ruta inesperada: %+v", overviewResp.Overview)
	}
}

func TestAPIProyectoOverviewSoportaDecisionesLegacy(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`DROP TABLE decisiones_proyecto`); err != nil {
		t.Fatalf("drop decisiones_proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`
		CREATE TABLE decisiones_proyecto (
			id                       INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto                 TEXT    NOT NULL,
			titulo                   TEXT    NOT NULL,
			solucion_elegida         TEXT    NOT NULL DEFAULT '',
			motivo                   TEXT    NOT NULL DEFAULT '',
			alternativas_descartadas TEXT    NOT NULL DEFAULT '',
			impacto                  TEXT    NOT NULL DEFAULT 'medio'
			                              CHECK (impacto IN ('alto','medio','bajo')),
			propuesta_codigo         TEXT    NOT NULL DEFAULT '',
			tarea_id                 INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			registrado_por           TEXT    NOT NULL DEFAULT '',
			created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("create legacy decisiones_proyecto: %v", err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO decisiones_proyecto (proyecto, titulo, solucion_elegida, motivo, alternativas_descartadas, impacto, registrado_por)
		VALUES (?,?,?,?,?,?,?)`,
		"orquestador",
		"ADR legacy",
		"Bridge OpenClaw",
		"Compatibilidad con BD recuperada",
		"Sin alternativa",
		"medio",
		"orquesta",
	); err != nil {
		t.Fatalf("insert legacy decision: %v", err)
	}

	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/proyectos/orquestador/overview", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status overview legacy inesperado: %d body=%s", rec.Code, rec.Body.String())
	}
	var resp apiProyectoOverviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode overview legacy: %v", err)
	}
	if resp.Overview == nil || resp.Overview.Proyecto == nil || resp.Overview.Proyecto.ID != proyectoID {
		t.Fatalf("overview legacy sin proyecto esperado: %+v", resp.Overview)
	}
	if len(resp.Overview.Decisiones) != 1 || resp.Overview.Decisiones[0] == nil || resp.Overview.Decisiones[0].Titulo != "ADR legacy" {
		t.Fatalf("overview legacy sin decisiones esperadas: %+v", resp.Overview)
	}
}
