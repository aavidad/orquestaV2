package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

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
