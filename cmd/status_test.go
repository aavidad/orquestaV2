/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func setServerHTTPClientForTest(t *testing.T, transport roundTripFunc) {
	t.Helper()
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{Transport: transport}
	resetServerDiscovery()
	t.Cleanup(func() {
		serverHTTPClient = prevClient
		resetServerDiscovery()
	})
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("cerrando stdout capturado: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("leyendo stdout capturado: %v", err)
	}
	return buf.String()
}

func TestShouldBypassLocalDBConServidorLocalDescubierto(t *testing.T) {
	setServerHTTPClientForTest(t, func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "127.0.0.1:16543" && req.URL.Path == "/api/server" {
			return newJSONResponse(http.StatusOK, `{"name":"orquesta"}`), nil
		}
		return nil, fmt.Errorf("sin servidor en %s", req.URL.String())
	})

	if !shouldBypassLocalDB([]string{"status"}) {
		t.Fatalf("status deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "ver"}) {
		t.Fatalf("config ver deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "agente-nuevo", "Codex9", "programador"}) {
		t.Fatalf("config agente-nuevo deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "agente-retirar", "Codex9"}) {
		t.Fatalf("config agente-retirar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "agente-rehabilitar", "Codex9"}) {
		t.Fatalf("config agente-rehabilitar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"sesion", "inicio", "Codex2"}) {
		t.Fatalf("sesion inicio deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"sesion", "fin", "Codex2"}) {
		t.Fatalf("sesion fin deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"sesion", "listar"}) {
		t.Fatalf("sesion listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"sesion", "nuevo-codex"}) {
		t.Fatalf("sesion nuevo-codex deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"tarea", "listar"}) {
		t.Fatalf("tarea listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"tarea", "ver", "7"}) {
		t.Fatalf("tarea ver deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"tarea", "nueva", "--titulo", "demo"}) {
		t.Fatalf("tarea nueva deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"tarea", "bloquear", "12", "Codex2", "esperando"}) {
		t.Fatalf("tarea bloquear deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"tarea", "desbloquear", "12", "Codex2", "resuelto"}) {
		t.Fatalf("tarea desbloquear deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"tarea", "nota", "12", "Codex2", "nota"}) {
		t.Fatalf("tarea nota deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"propuesta", "listar"}) {
		t.Fatalf("propuesta listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"propuesta", "ver", "OP-080"}) {
		t.Fatalf("propuesta ver deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"propuesta", "nueva", "--titulo", "demo"}) {
		t.Fatalf("propuesta nueva deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"propuesta", "cerrar", "OP-080", "consenso"}) {
		t.Fatalf("propuesta cerrar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"votar", "OP-080", "acuerdo", "--agente", "Codex2"}) {
		t.Fatalf("votar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"proyecto", "fusionar", "orquesta", "orquestador"}) {
		t.Fatalf("proyecto fusionar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "set", "clave", "valor"}) {
		t.Fatalf("config set deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "agente-nuevo", "Codex9", "programador"}) {
		t.Fatalf("config agente-nuevo deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "agente-retirar", "Codex9"}) {
		t.Fatalf("config agente-retirar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"config", "agente-rehabilitar", "Codex9"}) {
		t.Fatalf("config agente-rehabilitar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"exportar", "estado"}) {
		t.Fatalf("exportar estado deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"exportar", "audit", "10"}) {
		t.Fatalf("exportar audit deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"exportar", "diagnostico"}) {
		t.Fatalf("exportar diagnostico deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"runtime", "listar"}) {
		t.Fatalf("runtime listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"runtime", "mailbox-enviar", "Codex1", "Codex2", "handoff"}) {
		t.Fatalf("runtime mailbox-enviar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"runtime", "purgar-handles", "--agente", "Codex6"}) {
		t.Fatalf("runtime purgar-handles deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"runtime", "purgar-ordenes", "--agente", "Codex6"}) {
		t.Fatalf("runtime purgar-ordenes deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"progreso", "fase", "listar", "orquestador"}) {
		t.Fatalf("progreso fase listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"pool", "listar"}) {
		t.Fatalf("pool listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"politica-modelo", "listar"}) {
		t.Fatalf("politica-modelo listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"modelo", "resolver", "--perfil", "programador"}) {
		t.Fatalf("modelo resolver deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"lenguaje", "politica", "ver"}) {
		t.Fatalf("lenguaje politica ver deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"reglas", "listar"}) {
		t.Fatalf("reglas listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"skills", "listar"}) {
		t.Fatalf("skills listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"skills", "detectar-carencia", "--rol", "programador"}) {
		t.Fatalf("skills detectar-carencia deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"skills", "importar", "--rol", "programador", "--repo", "openai/skills", "--skill", "openai-docs"}) {
		t.Fatalf("skills importar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"workflows", "listar"}) {
		t.Fatalf("workflows listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"permisos", "listar"}) {
		t.Fatalf("permisos listar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"agente", "control", "arrancar", "Codex1"}) {
		t.Fatalf("agente control arrancar deberia saltarse la BD local con servidor local descubierto")
	}
	if !shouldBypassLocalDB([]string{"sesion", "presupuesto", "ver", "--agente", "Codex1"}) {
		t.Fatalf("sesion presupuesto ver deberia saltarse la BD local con servidor local descubierto")
	}
	if shouldBypassLocalDB(nil) {
		t.Fatalf("sin comando no deberia saltarse la BD local")
	}
	if got := activeServerURL(); got != defaultServerURL {
		t.Fatalf("URL activa inesperada: %s", got)
	}
}

func TestShouldBypassLocalDBCaeADefaultSiEnvNoResponde(t *testing.T) {
	t.Setenv(serverURLVar, "http://remote.invalid:9999")
	setServerHTTPClientForTest(t, func(req *http.Request) (*http.Response, error) {
		switch {
		case req.URL.Host == "remote.invalid:9999":
			return nil, fmt.Errorf("sin respuesta remota")
		case req.URL.Host == "127.0.0.1:16543" && req.URL.Path == "/api/server":
			return newJSONResponse(http.StatusOK, `{"name":"orquesta"}`), nil
		default:
			return nil, fmt.Errorf("sin servidor en %s", req.URL.String())
		}
	})

	if !shouldBypassLocalDB([]string{"status"}) {
		t.Fatalf("deberia usar el servidor local por fallback cuando la URL configurada falla")
	}
	if got := activeServerURL(); got != defaultServerURL {
		t.Fatalf("URL activa inesperada tras fallback: %s", got)
	}
}

func TestShouldBypassLocalDBMantieneFallbackLocalSinServidor(t *testing.T) {
	t.Setenv(serverURLVar, "http://remote.invalid:9999")
	setServerHTTPClientForTest(t, func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("sin servidor en %s", req.URL.String())
	})

	if shouldBypassLocalDB([]string{"status"}) {
		t.Fatalf("no deberia saltarse la BD local si no hay servidor utilizable")
	}
	if got := activeServerURL(); got != "" {
		t.Fatalf("no deberia descubrir servidor, obtuvo %s", got)
	}
}

func TestStatusExigeServidorSalvoRecuperacionLocal(t *testing.T) {
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()

	err := statusCmd.RunE(statusCmd, nil)
	if err == nil {
		t.Fatalf("status deberia exigir servidor o recuperacion local explicita")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "servidor") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestStatusPermiteRecuperacionConForceLocal(t *testing.T) {
	prepararDBTemporalCmd(t)
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL", "1")()
	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()

	out := captureOutput(t, func() {
		if err := statusCmd.RunE(statusCmd, nil); err != nil {
			t.Fatalf("status en recuperacion local: %v", err)
		}
	})
	if !strings.Contains(out, "ORQUESTA") {
		t.Fatalf("salida status inesperada en recuperacion local:\n%s", out)
	}
}

func TestStatusFuncionaEnRecuperacionLocalDB(t *testing.T) {
	prepararDBTemporalCmd(t)

	defer cambiarEnv(t, "ORQUESTA_SERVER_URL", "")()
	defer cambiarEnv(t, "ORQUESTA_DISABLE_SERVER_CLIENT", "1")()
	defer cambiarEnv(t, "ORQUESTA_FORCE_LOCAL_DB", "1")()

	out := captureOutput(t, func() {
		if err := statusCmd.RunE(statusCmd, nil); err != nil {
			t.Fatalf("status local recovery: %v", err)
		}
	})
	if !strings.Contains(out, "ORQUESTA") {
		t.Fatalf("salida status inesperada: %s", out)
	}
}

func TestStatusRecuperacionLocalExplicitaRenderizaResumen(t *testing.T) {
	prepararDBTemporalCmd(t)

	out := captureOutput(t, func() {
		if err := statusCmd.RunE(statusCmd, nil); err != nil {
			t.Fatalf("status local recovery: %v", err)
		}
	})
	if !strings.Contains(out, "ORQUESTA — ESTADO DEL PROYECTO") {
		t.Fatalf("salida inesperada en recuperacion local: %s", out)
	}
}

func TestFetchServerInfoAndStatus(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/server":
				return newJSONResponse(http.StatusOK, `{"name":"orquesta","version":"v1","storageMode":"single-process","storageDriver":"sqlite","sqlPlaceholder":"?","bootstrapSchema":true,"queryRebinding":true,"capabilities":["status","api"]}`), nil
			case "/api/status":
				return newJSONResponse(http.StatusOK, `{"generado":"2026-03-22T18:00:00Z","tareasPorEstado":{"completada":2,"en_progreso":1},"propuestasAbiertas":[{"codigo":"OP-080","titulo":"Servidor unico","acuerdo":3,"pendiente":1}],"tareasActivas":[{"id":7,"titulo":"Mover status a cliente HTTP","agente":"Codex2"}]}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	info, err := fetchServerInfo("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerInfo: %v", err)
	}
	if info.Name != "orquesta" || info.StorageMode != "single-process" {
		t.Fatalf("serverInfo inesperado: %+v", info)
	}
	if info.StorageDriver != "sqlite" || info.SQLPlaceholder != "?" || !info.BootstrapSchema || !info.QueryRebinding {
		t.Fatalf("serverInfo con metadatos inesperados: %+v", info)
	}
	if got := serverInfoLines(info); len(got) != 3 || got[0] != "🖥️  Backend activo: orquesta v1" || !strings.Contains(got[1], "modo single-process") || !strings.Contains(got[1], "driver sqlite") {
		t.Fatalf("lineas de backend inesperadas: %+v", got)
	}

	status, err := fetchServerStatus("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerStatus: %v", err)
	}
	if status.Generado != "2026-03-22T18:00:00Z" {
		t.Fatalf("generado inesperado: %+v", status)
	}
	if len(status.PropuestasAbiertas) != 1 || status.PropuestasAbiertas[0].Codigo != "OP-080" {
		t.Fatalf("propuestas inesperadas: %+v", status.PropuestasAbiertas)
	}
}

func TestLoadStatusSummaryIgnoraServerInfoInvalido(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/server":
				return newJSONResponse(http.StatusOK, `{"name":"orquesta"`), nil
			case "/api/status":
				return newJSONResponse(http.StatusOK, `{"generado":"2026-03-22T18:00:00Z","tareasPorEstado":{"completada":1}}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
		resetServerDiscovery()
	}()

	ctx, err := loadStatusSummary()
	if err != nil {
		t.Fatalf("loadStatusSummary: %v", err)
	}
	if ctx == nil || ctx.resumen == nil {
		t.Fatalf("contexto de estado inesperado: %+v", ctx)
	}
	if ctx.backend != nil {
		t.Fatalf("backend no deberia parsearse con JSON invalido: %+v", ctx.backend)
	}
}

func TestFetchServerStatusNoRecuperaAgentesCompatSiAgentesActivosVieneVacioPeroPresente(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/status":
				return newJSONResponse(http.StatusOK, `{"generado":"2026-04-01T08:45:00Z","agentesActivos":[],"agentes":[{"Nombre":"Codex3","Activo":false,"EstadoCuota":"activo"}],"tareasPorEstado":{"completada":1}}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	status, err := fetchServerStatus("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerStatus: %v", err)
	}
	if len(status.AgentesActivos) != 0 {
		t.Fatalf("agentes activos inesperados: %+v", status.AgentesActivos)
	}
}

func TestRenderStatusSummaryMuestraBackendActivo(t *testing.T) {
	out := captureOutput(t, func() {
		renderStatusSummary(&statusContext{
			resumen: &estadoResumen{
				TareasPorEstado: map[string]int{},
			},
			backend: &serverInfo{
				Name:         "orquesta",
				Version:      "v1",
				StorageMode:  "single-process",
				Capabilities: []string{"status", "api"},
			},
		})
	})
	if !strings.Contains(out, "Backend activo: orquesta v1") {
		t.Fatalf("salida sin backend activo: %s", out)
	}
	if !strings.Contains(out, "capacidades: status, api") {
		t.Fatalf("salida sin capacidades: %s", out)
	}
}

func TestRenderStatusSummaryMuestraCuotaAgente(t *testing.T) {
	pct := 18
	credits := 3.5
	sessionPct := 42
	dailyPct := 77
	weeklyPct := 12
	resetAt := time.Date(2026, 4, 1, 2, 0, 0, 0, time.UTC)
	sessionResetAt := time.Date(2026, 3, 31, 21, 0, 0, 0, time.UTC)
	weeklyResetAt := time.Date(2026, 4, 6, 2, 0, 0, 0, time.UTC)
	out := captureOutput(t, func() {
		renderStatusSummary(&statusContext{
			resumen: &estadoResumen{
				AgentesActivos: []*db.Agente{
					{
						Nombre:                    "Codex2",
						Rol:                       "programador",
						CuentaEmail:               "codex2@example.com",
						CuentaUsuario:             "codex2_user",
						CuotaRestantePct:          &pct,
						RemainingCredits:          &credits,
						PresupuestoEstado:         "handoff_preventivo",
						PresupuestoVentana:        "weekly",
						PresupuestoResetAt:        &resetAt,
						PresupuestoSesionPct:      &sessionPct,
						PresupuestoSesionResetAt:  &sessionResetAt,
						PresupuestoDiarioPct:      &dailyPct,
						PresupuestoDiarioResetAt:  &resetAt,
						PresupuestoSemanalPct:     &weeklyPct,
						PresupuestoSemanalResetAt: &weeklyResetAt,
					},
				},
				AgentesTrabajando: []*db.Agente{
					{
						Nombre: "Codex2",
					},
				},
				TareasPorEstado: map[string]int{},
			},
		})
	})
	if !strings.Contains(out, "Agentes: 1 conectados · 1 con trabajo activo") {
		t.Fatalf("salida sin resumen conectado/trabajando: %s", out)
	}
	if !strings.Contains(out, "cuenta codex2@example.com") {
		t.Fatalf("salida sin cuenta visible: %s", out)
	}
	if !strings.Contains(out, "usuario codex2_user") {
		t.Fatalf("salida sin usuario visible: %s", out)
	}
	if !strings.Contains(out, "efectivo 18%") {
		t.Fatalf("salida sin porcentaje de cuota: %s", out)
	}
	if !strings.Contains(out, "cred 3.50") {
		t.Fatalf("salida sin créditos restantes: %s", out)
	}
	if !strings.Contains(out, "ventana weekly") {
		t.Fatalf("salida sin ventana efectiva: %s", out)
	}
	if !strings.Contains(out, "reset 2026-04-01 04:00") && !strings.Contains(out, "reset 2026-04-01 02:00") {
		t.Fatalf("salida sin reset visible: %s", out)
	}
	if !strings.Contains(out, "sesión 42%") {
		t.Fatalf("salida sin porcentaje de sesión: %s", out)
	}
	if !strings.Contains(out, "diario 77%") {
		t.Fatalf("salida sin porcentaje diario: %s", out)
	}
	if !strings.Contains(out, "semanal 12%") {
		t.Fatalf("salida sin porcentaje semanal: %s", out)
	}
}

func TestRenderStatusSummaryMuestraAgentesEnEnfriamiento(t *testing.T) {
	resetAt := time.Date(2026, 4, 1, 1, 31, 33, 0, time.UTC)
	pct := 0
	out := captureOutput(t, func() {
		renderStatusSummary(&statusContext{
			resumen: &estadoResumen{
				Agentes: []*db.Agente{
					{
						Nombre:              "Codex5",
						Rol:                 "programador",
						Activo:              false,
						EstadoCuota:         "agotado",
						ReanimarAt:          &resetAt,
						MotivoPausa:         "Presupuesto agotado observado",
						CuotaRestantePct:    &pct,
						PresupuestoVentana:  "5h",
						PresupuestoResetAt:  &resetAt,
						CuentaEmail:         "maritere@avidad.com",
						PresupuestoEstado:   "agotado",
						PresupuestoDiarioPct: &pct,
					},
				},
				TareasPorEstado: map[string]int{},
			},
		})
	})
	if !strings.Contains(out, "En enfriamiento/cuota") {
		t.Fatalf("salida sin bloque de agentes pausados: %s", out)
	}
	if !strings.Contains(out, "Codex5") || !strings.Contains(out, "cuenta maritere@avidad.com") {
		t.Fatalf("salida sin detalle de agente pausado: %s", out)
	}
	if !strings.Contains(out, "cooldown hasta") {
		t.Fatalf("salida sin cooldown visible: %s", out)
	}
	if !strings.Contains(out, "cuota:agotado") {
		t.Fatalf("salida sin estado de cuota del agente pausado: %s", out)
	}
}

func TestRenderStatusSummaryMuestraTareasRetenidasPorCuota(t *testing.T) {
	out := captureOutput(t, func() {
		renderStatusSummary(&statusContext{
			resumen: &estadoResumen{
				Agentes: []*db.Agente{
					{
						Nombre:      "Codex1",
						Rol:         "programador",
						Activo:      false,
						EstadoCuota: "enfriamiento",
					},
				},
				TareasActivas: []tareaLite{
					{ID: 416, Titulo: "Integrar eventos del control plane", Agente: "Codex1", Estado: db.TareaEnProgreso},
				},
				TareasPorEstado: map[string]int{},
			},
		})
	})
	if !strings.Contains(out, "Retenidas por cuota") {
		t.Fatalf("salida sin bloque de tareas retenidas: %s", out)
	}
	if !strings.Contains(out, "[416]") || !strings.Contains(out, "Codex1") {
		t.Fatalf("salida sin tarea retenida visible: %s", out)
	}
}

func TestFetchStatusNoCuentaComoConectadoAgenteConSemanalObservadaAgotada(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
			t.Fatalf("registrando Codex5: %v", err)
		}
		sesionID, err := db.IniciarSesion("Codex5")
		if err != nil {
			t.Fatalf("IniciarSesion Codex5: %v", err)
		}
		now := time.Now().UTC()
		resetPrimary := now.Add(2 * time.Hour)
		resetWeekly := now.Add(4 * 24 * time.Hour)
		raw := `{"rate_limits":{"primary":{"used_percent":5,"window_minutes":300,"resets_at":` + strconv.FormatInt(resetPrimary.Unix(), 10) + `},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":` + strconv.FormatInt(resetWeekly.Unix(), 10) + `}}}`
		if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
			SesionID:        sesionID,
			WindowKind:      "5h",
			BudgetSource:    "codex_token_count_observed",
			RawSnapshotJSON: raw,
			CheckedAt:       now.Add(-6 * time.Hour),
		}); err != nil {
			t.Fatalf("RegistrarPresupuestoSesion: %v", err)
		}

		resp, err := dbStatusService{}.FetchStatus()
		if err != nil {
			t.Fatalf("FetchStatus: %v", err)
		}
		if len(resp.AgentesActivos) != 0 {
			t.Fatalf("Codex5 no deberia contarse como conectado: %+v", resp.AgentesActivos)
		}
		var codex5 *db.Agente
		for _, agente := range resp.Agentes {
			if agente != nil && agente.Nombre == "Codex5" {
				codex5 = agente
				break
			}
		}
		if codex5 == nil || codex5.EstadoCuota != "agotado" || codex5.Activo {
			t.Fatalf("estado visible inesperado para Codex5: %+v", codex5)
		}
	})
}

func TestFetchServerConfig(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path + "?" + req.URL.RawQuery {
			case "/api/config?":
				return newJSONResponse(http.StatusOK, `{"items":{"model_policy_default_reasoning":"high","pool_default_budget_source":"manual"}}`), nil
			case "/api/config?clave=model_policy_default_reasoning":
				return newJSONResponse(http.StatusOK, `{"clave":"model_policy_default_reasoning","valor":"high"}`), nil
			case "/api/config?clave=pool_default_budget_source":
				return newJSONResponse(http.StatusOK, `{"clave":"pool_default_budget_source","valor":"manual"}`), nil
			default:
				if req.URL.Path == "/api/config" && req.Method == http.MethodPost {
					return newJSONResponse(http.StatusOK, `{"clave":"pool_default_budget_source","valor":"manual"}`), nil
				}
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	items, err := fetchServerConfigAll("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerConfigAll: %v", err)
	}
	if items["pool_default_budget_source"] != "manual" {
		t.Fatalf("config inesperada: %+v", items)
	}

	valor, err := fetchServerConfigValue("http://orquesta.local", "model_policy_default_reasoning")
	if err != nil {
		t.Fatalf("fetchServerConfigValue: %v", err)
	}
	if valor != "high" {
		t.Fatalf("valor inesperado: %s", valor)
	}

	if err := submitServerConfigValue("http://orquesta.local", "pool_default_budget_source", "manual"); err != nil {
		t.Fatalf("submitServerConfigValue: %v", err)
	}
}

func TestFetchServerConfigAceptaSobreActual(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path + "?" + req.URL.RawQuery {
			case "/api/config?":
				return newJSONResponse(http.StatusOK, `{"config":{"model_policy_default_reasoning":"high","pool_default_budget_source":"manual"}}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	items, err := fetchServerConfigAll("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerConfigAll shape actual: %v", err)
	}
	if items["model_policy_default_reasoning"] != "high" || items["pool_default_budget_source"] != "manual" {
		t.Fatalf("config actual inesperada: %+v", items)
	}
}

func TestFetchServerText(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/export/estado":
				return newJSONResponse(http.StatusOK, "# export\n"), nil
			case "/api/export/audit":
				return newJSONResponse(http.StatusOK, "# audit\n"), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	body, err := fetchServerText("http://orquesta.local/api/export/estado")
	if err != nil {
		t.Fatalf("fetchServerText estado: %v", err)
	}
	if body != "# export\n" {
		t.Fatalf("texto inesperado: %q", body)
	}
	body, err = fetchServerText("http://orquesta.local/api/export/audit")
	if err != nil {
		t.Fatalf("fetchServerText audit: %v", err)
	}
	if body != "# audit\n" {
		t.Fatalf("texto inesperado: %q", body)
	}
}

func TestSubmitServerSessionStartAndFinish(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/api/sesiones/inicio" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"agente":"Codex2","sesion_id":7,"rol":"programador","propuestas_pendientes":[{"Codigo":"OP-081","Titulo":"RAEX"}],"reglas":[{"Categoria":"calidad","Titulo":"No romper tests","Descripcion":"Mantener verde"}],"skills":[{"Nombre":"rg","CuandoUsar":"buscar rapido"}],"workflow_pasos":["1. votar","2. tomar tarea"]}`), nil
			case req.URL.Path == "/api/sesiones/fin" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"agente":"Codex2"}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	res, err := submitServerSessionStart("http://orquesta.local", "Codex2", false)
	if err != nil {
		t.Fatalf("submitServerSessionStart: %v", err)
	}
	if res == nil || res.Agente != "Codex2" || res.SesionID != 7 || len(res.WorkflowPasos) != 2 {
		t.Fatalf("sesion inicio inesperada: %+v", res)
	}
	if err := submitServerSessionFinish("http://orquesta.local", "Codex2"); err != nil {
		t.Fatalf("submitServerSessionFinish: %v", err)
	}
}

func TestFetchServerAgents(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/api/agentes" && req.Method == http.MethodGet:
				return newJSONResponse(http.StatusOK, `{"items":[{"Nombre":"Codex2","Rol":"programador","Activo":true},{"Nombre":"antigravity","Rol":"documentador","Activo":false}]}`), nil
			case req.URL.Path == "/api/agentes" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusCreated, `{"ok":true,"nombre":"Codex9","rol":"programador"}`), nil
			case req.URL.Path == "/api/agentes/Codex9/retirar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"nombre":"Codex9"}`), nil
			case req.URL.Path == "/api/agentes/Codex9/rehabilitar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"nombre":"Codex9"}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	agentes, err := fetchServerAgents("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerAgents: %v", err)
	}
	if len(agentes) != 2 || agentes[0].Nombre != "Codex2" {
		t.Fatalf("agentes inesperados: %+v", agentes)
	}
	if err := submitServerCreateAgent("http://orquesta.local", "Codex9", "programador"); err != nil {
		t.Fatalf("submitServerCreateAgent: %v", err)
	}
	if err := submitServerAgentAction("http://orquesta.local", "Codex9", "retirar"); err != nil {
		t.Fatalf("submitServerAgentAction(retirar): %v", err)
	}
	if err := submitServerAgentAction("http://orquesta.local", "Codex9", "rehabilitar"); err != nil {
		t.Fatalf("submitServerAgentAction(rehabilitar): %v", err)
	}
}

func TestSubmitServerVoteAndTaskAction(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/api/votar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"codigo":"OP-080","agente":"Codex2","posicion":"acuerdo","comentario":"ok","consensoAlcanzado":false,"conteo":{"acuerdo":3,"desacuerdo":0,"abstencion":0,"pendiente":1}}`), nil
			case req.URL.Path == "/api/tareas" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusCreated, `{"id":55}`), nil
			case req.URL.Path == "/api/tareas/12/tomar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"id":12,"accion":"tomar"}`), nil
			case req.URL.Path == "/api/tareas/12/iniciar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"id":12,"accion":"iniciar"}`), nil
			case req.URL.Path == "/api/tareas/12/completar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"id":12,"accion":"completar"}`), nil
			case req.URL.Path == "/api/tareas/12/bloquear" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"id":12,"accion":"bloquear"}`), nil
			case req.URL.Path == "/api/tareas/12/desbloquear" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"id":12,"accion":"desbloquear"}`), nil
			case req.URL.Path == "/api/tareas/12/nota" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"id":12,"accion":"nota"}`), nil
			case req.URL.Path == "/api/propuestas" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusCreated, `{"id":81,"codigo":"OP-081"}`), nil
			case req.URL.Path == "/api/propuestas/OP-081/cerrar" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusOK, `{"ok":true,"codigo":"OP-081","estado":"consenso"}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	voto, err := submitServerVote("http://orquesta.local", "OP-080", "Codex2", db.VotoAcuerdo, "ok")
	if err != nil {
		t.Fatalf("submitServerVote: %v", err)
	}
	if voto == nil || voto.Codigo != "OP-080" || voto.Conteo["acuerdo"] != 3 {
		t.Fatalf("voto inesperado: %+v", voto)
	}

	for _, accion := range []string{"tomar", "iniciar", "completar", "bloquear", "desbloquear", "nota"} {
		if err := submitServerTaskAction("http://orquesta.local", 12, accion, map[string]any{"agente": "Codex2"}); err != nil {
			t.Fatalf("submitServerTaskAction(%s): %v", accion, err)
		}
	}

	taskID, err := submitServerCreateTask("http://orquesta.local", map[string]any{"titulo": "demo"})
	if err != nil {
		t.Fatalf("submitServerCreateTask: %v", err)
	}
	if taskID != 55 {
		t.Fatalf("taskID inesperado: %d", taskID)
	}

	proposalID, codigo, err := submitServerCreateProposal("http://orquesta.local", map[string]any{"titulo": "demo"})
	if err != nil {
		t.Fatalf("submitServerCreateProposal: %v", err)
	}
	if proposalID != 81 || codigo != "OP-081" {
		t.Fatalf("propuesta inesperada: id=%d codigo=%s", proposalID, codigo)
	}

	if err := submitServerCloseProposal("http://orquesta.local", "OP-081", "consenso", "alberto"); err != nil {
		t.Fatalf("submitServerCloseProposal: %v", err)
	}
}

func TestFetchServerTasksAndProposals(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/api/tareas":
				return newJSONResponse(http.StatusOK, `{"items":[{"ID":12,"Titulo":"Tarea remota","Estado":"asignada","Modulo":"core","Prioridad":"media"}]}`), nil
			case "/api/tareas/12":
				return newJSONResponse(http.StatusOK, `{"item":{"ID":12,"Titulo":"Tarea remota","Estado":"asignada","Modulo":"core","Prioridad":"media","CreadoPor":"alberto"}}`), nil
			case "/api/propuestas":
				return newJSONResponse(http.StatusOK, `{"items":[{"ID":80,"Codigo":"OP-080","Titulo":"Servidor unico","Estado":"abierta","Tipo":"arquitectura","PropuestoPor":"Codex1"}]}`), nil
			case "/api/propuestas/OP-080":
				return newJSONResponse(http.StatusOK, `{"Proposal":{"ID":80,"Codigo":"OP-080","Titulo":"Servidor unico","Estado":"abierta","Tipo":"arquitectura","PropuestoPor":"Codex1"},"Votes":[{"Agente":"Codex1","Posicion":"acuerdo"}]}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	tareas, err := fetchServerTasks("http://orquesta.local", nil)
	if err != nil {
		t.Fatalf("fetchServerTasks: %v", err)
	}
	if len(tareas) != 1 || tareas[0].ID != 12 {
		t.Fatalf("tareas inesperadas: %+v", tareas)
	}

	propuestas, err := fetchServerProposals("http://orquesta.local", nil)
	if err != nil {
		t.Fatalf("fetchServerProposals: %v", err)
	}
	if len(propuestas) != 1 || propuestas[0].Codigo != "OP-080" {
		t.Fatalf("propuestas inesperadas: %+v", propuestas)
	}

	tarea, err := fetchServerTaskDetail("http://orquesta.local", 12)
	if err != nil {
		t.Fatalf("fetchServerTaskDetail: %v", err)
	}
	if tarea == nil || tarea.ID != 12 {
		t.Fatalf("tarea detalle inesperada: %+v", tarea)
	}

	propuesta, err := fetchServerProposalDetail("http://orquesta.local", "OP-080")
	if err != nil {
		t.Fatalf("fetchServerProposalDetail: %v", err)
	}
	if propuesta == nil || propuesta.Proposal == nil || propuesta.Proposal.Codigo != "OP-080" {
		t.Fatalf("propuesta detalle inesperada: %+v", propuesta)
	}
}

func TestFetchServerRecursosAceptaSobresActuales(t *testing.T) {
	prevClient := serverHTTPClient
	serverHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/api/agentes" && req.Method == http.MethodGet:
				return newJSONResponse(http.StatusOK, `{"agentes":[{"Nombre":"Codex2","Rol":"programador","Activo":true}]}`), nil
			case req.URL.Path == "/api/tareas" && req.Method == http.MethodGet:
				return newJSONResponse(http.StatusOK, `{"tareas":[{"ID":12,"Titulo":"Tarea remota","Estado":"asignada","Modulo":"core","Prioridad":"media"}]}`), nil
			case req.URL.Path == "/api/tareas/12" && req.Method == http.MethodGet:
				return newJSONResponse(http.StatusOK, `{"tarea":{"ID":12,"Titulo":"Tarea remota","Estado":"asignada","Modulo":"core","Prioridad":"media","CreadoPor":"alberto"}}`), nil
			case req.URL.Path == "/api/propuestas" && req.Method == http.MethodGet:
				return newJSONResponse(http.StatusOK, `{"propuestas":[{"ID":80,"Codigo":"OP-080","Titulo":"Servidor unico","Estado":"abierta","Tipo":"arquitectura","PropuestoPor":"Codex1"}]}`), nil
			case req.URL.Path == "/api/propuestas/OP-080" && req.Method == http.MethodGet:
				return newJSONResponse(http.StatusOK, `{"propuesta":{"ID":80,"Codigo":"OP-080","Titulo":"Servidor unico","Estado":"abierta","Tipo":"arquitectura","PropuestoPor":"Codex1","Votos":[{"Agente":"Codex1","Posicion":"acuerdo"}]}}`), nil
			case req.URL.Path == "/api/tareas" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusCreated, `{"ok":true,"tarea":{"ID":55,"Titulo":"demo"}}`), nil
			case req.URL.Path == "/api/propuestas" && req.Method == http.MethodPost:
				return newJSONResponse(http.StatusCreated, `{"ok":true,"id":81,"propuesta":{"ID":81,"Codigo":"OP-081","Titulo":"demo"}}`), nil
			default:
				return newJSONResponse(http.StatusNotFound, `{"error":"not found"}`), nil
			}
		}),
	}
	defer func() {
		serverHTTPClient = prevClient
	}()

	agentes, err := fetchServerAgents("http://orquesta.local")
	if err != nil {
		t.Fatalf("fetchServerAgents actual: %v", err)
	}
	if len(agentes) != 1 || agentes[0].Nombre != "Codex2" {
		t.Fatalf("agentes actuales inesperados: %+v", agentes)
	}

	taskID, err := submitServerCreateTask("http://orquesta.local", map[string]any{"titulo": "demo"})
	if err != nil {
		t.Fatalf("submitServerCreateTask actual: %v", err)
	}
	if taskID != 55 {
		t.Fatalf("taskID actual inesperado: %d", taskID)
	}

	proposalID, codigo, err := submitServerCreateProposal("http://orquesta.local", map[string]any{"titulo": "demo"})
	if err != nil {
		t.Fatalf("submitServerCreateProposal actual: %v", err)
	}
	if proposalID != 81 || codigo != "OP-081" {
		t.Fatalf("propuesta actual inesperada: id=%d codigo=%s", proposalID, codigo)
	}

	tareas, err := fetchServerTasks("http://orquesta.local", nil)
	if err != nil {
		t.Fatalf("fetchServerTasks actual: %v", err)
	}
	if len(tareas) != 1 || tareas[0].ID != 12 {
		t.Fatalf("tareas actuales inesperadas: %+v", tareas)
	}

	tarea, err := fetchServerTaskDetail("http://orquesta.local", 12)
	if err != nil {
		t.Fatalf("fetchServerTaskDetail actual: %v", err)
	}
	if tarea == nil || tarea.ID != 12 {
		t.Fatalf("tarea detalle actual inesperada: %+v", tarea)
	}

	propuestas, err := fetchServerProposals("http://orquesta.local", nil)
	if err != nil {
		t.Fatalf("fetchServerProposals actual: %v", err)
	}
	if len(propuestas) != 1 || propuestas[0].Codigo != "OP-080" {
		t.Fatalf("propuestas actuales inesperadas: %+v", propuestas)
	}

	propuesta, err := fetchServerProposalDetail("http://orquesta.local", "OP-080")
	if err != nil {
		t.Fatalf("fetchServerProposalDetail actual: %v", err)
	}
	if propuesta == nil || propuesta.Proposal == nil || propuesta.Proposal.Codigo != "OP-080" || len(propuesta.Votes) != 1 {
		t.Fatalf("propuesta detalle actual inesperada: %+v", propuesta)
	}
}
