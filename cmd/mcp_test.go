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
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitestadisticasapp"
	"orquesta/lenguajeapp"
	"orquesta/microprogramacionapp"
	"orquesta/propuestasapp"
	"orquesta/reviewapp"
	"orquesta/runtimesapp"
)

type gitStatsCollectorStub struct {
	gotCWD   string
	gotSince time.Time
	response *gitestadisticasapp.Stats
	err      error
}

func (s *gitStatsCollectorStub) Collect(cwd string, since time.Time) (*gitestadisticasapp.Stats, error) {
	s.gotCWD = cwd
	s.gotSince = since
	return s.response, s.err
}

type mcpRuntimeTraceFixture struct {
	agente   string
	proyecto string
	handleID int64
}

type mcpRuntimeTreeFixture struct {
	agenteRaiz string
	agenteHijo string
	proyecto   string
	rootID     int64
	childID    int64
	closedID   int64
}

func setupMCPRuntimeTraceFixture(t *testing.T) mcpRuntimeTraceFixture {
	t.Helper()

	agente := "CodexTraceMCP"
	proyecto := "orquestador"
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, proyecto)
	if err := db.RegistrarAgente(agente, "programador"); err != nil {
		t.Fatalf("registrando %s: %v", agente, err)
	}
	proyectoID := insertTestProyecto(t, proyecto, proyecto, projectDir)
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      agente,
		ProyectoID:  &proyectoID,
		CWD:         projectDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("runtime handle esperado, got=%+v err=%v", handle, err)
	}
	traceDir := filepath.Join(baseDir, ".orquesta-runtime", strings.ToLower(agente), "20260329-120000-000000001")
	if err := os.MkdirAll(traceDir, 0o700); err != nil {
		t.Fatalf("mkdir trace dir: %v", err)
	}
	logPath := filepath.Join(traceDir, "pty.log")
	manifestPath := filepath.Join(traceDir, "runtime.json")
	if err := os.WriteFile(logPath, []byte("inicio\navance\nfin\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	metaJSON, _ := json.Marshal(map[string]any{
		"trace_dir":      traceDir,
		"trace_manifest": manifestPath,
		"log_path":       logPath,
		"working_dir":    projectDir,
		"driver":         "process_pty_cli",
	})
	if _, err := db.DB.Exec(`UPDATE runtime_handles SET metadata_json = ? WHERE id = ?`, string(metaJSON), handle.ID); err != nil {
		t.Fatalf("update runtime handle metadata: %v", err)
	}
	return mcpRuntimeTraceFixture{agente: agente, proyecto: proyecto, handleID: handle.ID}
}

func setupMCPRuntimeTreeFixture(t *testing.T) mcpRuntimeTreeFixture {
	t.Helper()

	baseDir := t.TempDir()
	proyecto := "orquestador"
	projectDir := filepath.Join(baseDir, proyecto)
	agenteRaiz := "CodexRuntimeRoot"
	agenteHijo := "CodexRuntimeChild"
	agenteCerrado := "CodexRuntimeClosed"
	for _, agente := range []string{agenteRaiz, agenteHijo, agenteCerrado} {
		if err := db.RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrando %s: %v", agente, err)
		}
	}
	proyectoID := insertTestProyecto(t, proyecto, proyecto, projectDir)
	sesion, err := db.IniciarSesionContexto(db.SesionInicio{
		Agente:      agenteRaiz,
		ProyectoID:  &proyectoID,
		CWD:         projectDir,
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion raiz: %v", err)
	}
	rootRuntime, err := db.GetRuntimeBySesionID(sesion.ID)
	if err != nil || rootRuntime == nil {
		t.Fatalf("runtime raíz esperado, got=%+v err=%v", rootRuntime, err)
	}
	childID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:            agenteHijo,
		ProyectoID:        &proyectoID,
		ParentRuntimeID:   &rootRuntime.ID,
		Provider:          "openai",
		Connector:         "cli",
		ExternalSessionID: "mcp-child-runtime",
		LogicalState:      "activo",
		ProcessState:      "vivo",
		CWD:               projectDir,
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("registrar runtime hijo: %v", err)
	}
	closedID, err := db.RegistrarRuntimeInstance(&db.RuntimeInstance{
		Agente:            agenteCerrado,
		ProyectoID:        &proyectoID,
		Provider:          "openai",
		Connector:         "cli",
		ExternalSessionID: "mcp-closed-runtime",
		LogicalState:      "cerrado",
		ProcessState:      "finalizado",
		CWD:               projectDir,
		Branch:            "main",
	})
	if err != nil {
		t.Fatalf("registrar runtime cerrado: %v", err)
	}
	return mcpRuntimeTreeFixture{
		agenteRaiz: agenteRaiz,
		agenteHijo: agenteHijo,
		proyecto:   proyecto,
		rootID:     rootRuntime.ID,
		childID:    childID,
		closedID:   closedID,
	}
}

func TestMCPHandleInitializeNegociaVersion(t *testing.T) {
	t.Helper()

	srv := &mcpServer{}
	params := mustJSON(t, map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
	})
	result, rpcErr := srv.handleRequest(mcpRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  params,
	})
	if rpcErr != nil {
		t.Fatalf("initialize devolvio error: %+v", rpcErr)
	}

	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("resultado inesperado: %#v", result)
	}
	if payload["protocolVersion"] != "2025-06-18" {
		t.Fatalf("version inesperada: %#v", payload["protocolVersion"])
	}
	if !srv.initialized {
		t.Fatalf("el servidor no quedo inicializado")
	}
}

func TestAPIMCPDescribeEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/mcp", nil)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["endpoint"] != "/api/mcp" {
		t.Fatalf("endpoint inesperado: %#v", payload)
	}
	if payload["transport"] != "http_jsonrpc" {
		t.Fatalf("transport inesperado: %#v", payload)
	}
}

func TestAPIMCPCallToolStateless(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		body := mustJSON(t, map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "orquesta.proyectos.listar",
				"arguments": map[string]any{"tipo": "repo", "activo": true},
			},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/mcp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux := http.NewServeMux()
		registerAPIRoutes(mux)

		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
		}

		var resp mcpResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if resp.Error != nil {
			t.Fatalf("mcp error inesperado: %+v", resp.Error)
		}
		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("resultado inesperado: %#v", resp.Result)
		}
		content, ok := result["content"].([]any)
		if !ok || len(content) == 0 {
			t.Fatalf("content MCP inesperado: %#v", result)
		}
		item, _ := content[0].(map[string]any)
		text, _ := item["text"].(string)
		if !strings.Contains(text, `"slug": "orquestador"`) {
			t.Fatalf("resultado MCP sin proyecto esperado: %s", text)
		}
	})
}

func TestAPIMCPCallServerRearmStateless(t *testing.T) {
	prevApply := serverOperationalApplyNextActionFn
	defer func() {
		serverOperationalApplyNextActionFn = prevApply
	}()

	serverOperationalApplyNextActionFn = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"queue_kind": "safe",
		}, nil
	}

	body := mustJSON(t, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "orquesta.server.rearm",
			"arguments": map[string]any{"supervisor": "OpenClaw"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/mcp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	registerAPIRoutes(mux)

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status inesperado: %d body=%s", rec.Code, rec.Body.String())
	}

	var resp mcpResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("mcp error inesperado: %+v", resp.Error)
	}
	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("resultado inesperado: %#v", resp.Result)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content MCP inesperado: %#v", result)
	}
	item, _ := content[0].(map[string]any)
	text, _ := item["text"].(string)
	for _, token := range []string{`"ok": true`, `"supervisor": "OpenClaw"`, `"queue_kind": "safe"`} {
		if !strings.Contains(text, token) {
			t.Fatalf("resultado MCP sin token %q: %s", token, text)
		}
	}
}

func TestMCPToolsIncluyeLenguaje(t *testing.T) {
	tools := listMCPTools()
	for _, name := range []string{
		"orquesta.lenguaje.politica.ver",
		"orquesta.lenguaje.politica.fijar",
		"orquesta.lenguaje.matriz.listar",
		"orquesta.lenguaje.matriz.fijar",
		"orquesta.lenguaje.matriz.borrar",
		"orquesta.lenguaje.resolver",
	} {
		if !mcpToolListed(tools, name) {
			t.Fatalf("tool MCP no registrada: %s", name)
		}
	}
}

func TestMCPToolsIncluyeLifecycleAgentes(t *testing.T) {
	tools := listMCPTools()
	for _, name := range []string{
		"orquesta.agentes.lanzar",
		"orquesta.agentes.start",
		"orquesta.agentes.pause",
		"orquesta.agentes.resume",
		"orquesta.agentes.stop",
	} {
		if !mcpToolListed(tools, name) {
			t.Fatalf("tool MCP no registrada: %s", name)
		}
	}
}

func TestMCPToolLenguajeParidadBasica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		policyResult, err := callMCPTool("orquesta.lenguaje.politica.fijar", map[string]any{
			"default_language":               "en",
			"documentation_multilang":        true,
			"apps_multilang":                 true,
			"documentation_default_language": "en",
			"apps_default_language":          "en",
			"allowed_languages":              []any{"es", "en", "fr"},
			"notes":                          "demo mcp",
			"updated_by":                     "CodexMCP",
		})
		if err != nil {
			t.Fatalf("fijar politica via MCP: %v", err)
		}
		policy, ok := policyResult["structuredContent"].(*lenguajeapp.LanguagePolicy)
		if !ok || policy == nil {
			t.Fatalf("structuredContent politica inesperado: %#v", policyResult["structuredContent"])
		}
		if policy.DefaultLanguage != "en" || policy.AppsDefaultLang != "en" {
			t.Fatalf("politica MCP inesperada: %+v", policy)
		}

		matrixSetResult, err := callMCPTool("orquesta.lenguaje.matriz.fijar", map[string]any{
			"scope":      "project",
			"selector":   "orquestador",
			"contexto":   "apps",
			"language":   "fr",
			"reason":     "demo",
			"updated_by": "CodexMCP",
		})
		if err != nil {
			t.Fatalf("fijar matriz via MCP: %v", err)
		}
		entry, ok := matrixSetResult["structuredContent"].(*lenguajeapp.LanguageMatrixEntry)
		if !ok || entry == nil {
			t.Fatalf("structuredContent entrada inesperado: %#v", matrixSetResult["structuredContent"])
		}
		if entry.Language != "fr" || entry.Context != "apps" {
			t.Fatalf("entrada MCP inesperada: %+v", entry)
		}

		matrixListResult, err := callMCPTool("orquesta.lenguaje.matriz.listar", nil)
		if err != nil {
			t.Fatalf("listar matriz via MCP: %v", err)
		}
		entries, ok := matrixListResult["structuredContent"].([]*lenguajeapp.LanguageMatrixEntry)
		if !ok {
			t.Fatalf("structuredContent matriz inesperado: %#v", matrixListResult["structuredContent"])
		}
		if len(entries) != 1 || entries[0].Language != "fr" {
			t.Fatalf("matriz MCP inesperada: %+v", entries)
		}

		resolveResult, err := callMCPTool("orquesta.lenguaje.resolver", map[string]any{
			"proyecto": "orquestador",
			"contexto": "apps",
		})
		if err != nil {
			t.Fatalf("resolver lenguaje via MCP: %v", err)
		}
		resolution, ok := resolveResult["structuredContent"].(*lenguajeapp.LanguageResolution)
		if !ok || resolution == nil {
			t.Fatalf("structuredContent resolucion inesperado: %#v", resolveResult["structuredContent"])
		}
		if resolution.Idioma != "fr" || !strings.Contains(resolution.Origen, "project.orquestador.apps") {
			t.Fatalf("resolucion MCP inesperada: %+v", resolution)
		}

		deleteResult, err := callMCPTool("orquesta.lenguaje.matriz.borrar", map[string]any{
			"scope":    "project",
			"selector": "orquestador",
			"contexto": "apps",
		})
		if err != nil {
			t.Fatalf("borrar matriz via MCP: %v", err)
		}
		deleted, ok := deleteResult["structuredContent"].(map[string]any)
		if !ok || deleted["ok"] != true {
			t.Fatalf("resultado borrado inesperado: %#v", deleteResult["structuredContent"])
		}

		resolveFallbackResult, err := callMCPTool("orquesta.lenguaje.resolver", map[string]any{
			"proyecto": "orquestador",
			"contexto": "apps",
		})
		if err != nil {
			t.Fatalf("resolver fallback via MCP: %v", err)
		}
		fallback, ok := resolveFallbackResult["structuredContent"].(*lenguajeapp.LanguageResolution)
		if !ok || fallback == nil {
			t.Fatalf("structuredContent fallback inesperado: %#v", resolveFallbackResult["structuredContent"])
		}
		if fallback.Idioma != "en" || !strings.Contains(fallback.Origen, "policy") || fallback.Entrada != nil {
			t.Fatalf("fallback MCP inesperado: %+v", fallback)
		}
	})
}

func TestMCPResourceReadDetallePropuesta(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		insertTestPropuesta(t, "OP-049", "Arquitectura objetivo", "MCP como adaptador de entrada")

		contents, err := readMCPResource("orquesta://propuestas/OP-049")
		if err != nil {
			t.Fatalf("readMCPResource: %v", err)
		}
		if len(contents) != 1 {
			t.Fatalf("contents inesperado: %#v", contents)
		}
		text, _ := contents[0]["text"].(string)
		if !strings.Contains(text, "OP-049") {
			t.Fatalf("el recurso no contiene la propuesta: %s", text)
		}
		if !strings.Contains(text, "MCP como adaptador de entrada") {
			t.Fatalf("el recurso no contiene la descripcion: %s", text)
		}
	})
}

func mcpToolListed(tools []mcpTool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func TestMCPPromptBriefingIncluyeReglasYPropuestasPendientes(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		propuestaID := insertTestPropuesta(t, "OP-049", "Arquitectura objetivo", "Descripcion")
		insertTestVotoPendiente(t, propuestaID, "Codex2")

		result, err := getMCPPrompt("orquesta.briefing.agente", map[string]any{"agente": "Codex2"})
		if err != nil {
			t.Fatalf("getMCPPrompt: %v", err)
		}
		messages, ok := result["messages"].([]mcpPromptMessage)
		if !ok || len(messages) != 1 {
			t.Fatalf("mensajes inesperados: %#v", result["messages"])
		}
		content, _ := messages[0].Content.(map[string]any)
		text, _ := content["text"].(string)
		if !strings.Contains(text, "Propuestas pendientes de voto") {
			t.Fatalf("faltan propuestas pendientes en el briefing: %s", text)
		}
		if !strings.Contains(text, "Reglas activas") {
			t.Fatalf("faltan reglas activas en el briefing: %s", text)
		}
		if !strings.Contains(text, "docs/BIBLIA_APP_ORQUESTA.md") {
			t.Fatalf("falta la doctrina canonica en el briefing: %s", text)
		}
		if !strings.Contains(text, "orquesta.skills.detectar-carencia") {
			t.Fatalf("falta el preflight de skills en el briefing: %s", text)
		}
	})
}

func TestMCPPromptGuidanceAgenteExponeContratoCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		result, err := getMCPPrompt("orquesta.guidance.agente", map[string]any{"agente": "Codex2"})
		if err != nil {
			t.Fatalf("getMCPPrompt guidance agente: %v", err)
		}
		messages, ok := result["messages"].([]mcpPromptMessage)
		if !ok || len(messages) != 1 {
			t.Fatalf("mensajes inesperados: %#v", result["messages"])
		}
		content, _ := messages[0].Content.(map[string]any)
		text, _ := content["text"].(string)
		for _, token := range []string{
			"Guidance canónica de Codex2",
			"## Role & Intent",
			"## Operating Principles",
			"## Execution Protocol",
			"## Constraints & Safety",
			"## Verification & Completion",
			"## Recovery & Lifecycle",
			"docs/BIBLIA_APP_ORQUESTA.md",
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("falta %q en guidance agente: %s", token, text)
			}
		}
	})
}

func TestMCPPromptBriefingSupervisorIncluyeFlotaYRetenidasPorCuota(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
			t.Fatalf("registrando Codex5: %v", err)
		}
		if _, err := db.IniciarSesion("Codex3"); err != nil {
			t.Fatalf("iniciar sesion Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE agentes SET estado_sesion='esperando', estado_cuota='activo' WHERE nombre='Codex3'`); err != nil {
			t.Fatalf("activar Codex3: %v", err)
		}
		reset := time.Now().UTC().Add(2 * time.Hour)
		if _, err := db.DB.Exec(`UPDATE agentes SET activo=0, estado_cuota='enfriamiento', reanimar_at=? WHERE nombre='Codex5'`, reset); err != nil {
			t.Fatalf("enfriar Codex5: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Retenida por cuota",
			Descripcion: "Debe aparecer en el briefing de supervisor",
			ProyectoID:  &proyectoID,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente='Codex5' WHERE id=?`, tareaID); err != nil {
			t.Fatalf("asignar tarea retenida: %v", err)
		}
		gateID, err := db.CrearReviewGate(&db.ReviewGate{
			ProyectoID:     &proyectoID,
			TareaID:        &tareaID,
			RequestedBy:    "orquesta",
			ReviewerAgente: "Codex4",
			Estado:         db.ReviewGateEnRevision,
			SeverityMax:    "high",
		})
		if err != nil {
			t.Fatalf("crear review gate: %v", err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex3",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion runtime: %v", err)
		}
		runtime, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtime, err)
		}
		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtime.ID,
			Kind:        "approval_request",
			Level:       "warning",
			Message:     "Necesito confirmacion para integrar el frente API tras la ultima revision",
			PayloadJSON: `{"classification":"approval_request"}`,
		}); err != nil {
			t.Fatalf("registrar runtime event: %v", err)
		}

		result, err := getMCPPrompt("orquesta.briefing.supervisor", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("getMCPPrompt supervisor: %v", err)
		}
		messages, ok := result["messages"].([]mcpPromptMessage)
		if !ok || len(messages) != 1 {
			t.Fatalf("mensajes inesperados: %#v", result["messages"])
		}
		content, _ := messages[0].Content.(map[string]any)
		text, _ := content["text"].(string)
		for _, token := range []string{
			"Briefing de supervisor: OpenClaw",
			"Workers disponibles",
			"Codex3",
			"Retenida por cuota",
			"Review gates abiertos",
			"#" + itoa(gateID),
			"Codex4",
			"Señales recientes de revisión e integración",
			"approval_request",
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("falta %q en el briefing de supervisor: %s", token, text)
			}
		}
		if !strings.Contains(text, "Codex5") {
			t.Fatalf("falta Codex5 en el briefing de supervisor: %s", text)
		}
	})
}

func TestMCPPromptGuidanceSupervisorExponeContratoCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		if _, err := db.IniciarSesion("Codex3"); err != nil {
			t.Fatalf("iniciar sesion Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE agentes SET estado_sesion='esperando', estado_cuota='activo' WHERE nombre='Codex3'`); err != nil {
			t.Fatalf("activar Codex3: %v", err)
		}
		result, err := getMCPPrompt("orquesta.guidance.supervisor", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("getMCPPrompt guidance supervisor: %v", err)
		}
		messages, ok := result["messages"].([]mcpPromptMessage)
		if !ok || len(messages) != 1 {
			t.Fatalf("mensajes inesperados: %#v", result["messages"])
		}
		content, _ := messages[0].Content.(map[string]any)
		text, _ := content["text"].(string)
		for _, token := range []string{
			"Guidance canónica del supervisor: OpenClaw",
			"## Role & Intent",
			"## Operating Principles",
			"## Execution Protocol",
			"## Constraints & Safety",
			"## Verification & Completion",
			"## Recovery & Lifecycle",
			"Flota conectada ahora",
			"docs/BIBLIA_APP_ORQUESTA.md",
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("falta %q en guidance supervisor: %s", token, text)
			}
		}
	})
}

func TestSupervisorBriefingYGuidanceUsanContadoresVisiblesDeWorkersYSupervisor(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex2", Rol: "supervisor", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex2", Rol: "supervisor", Activo: true, EstadoCuota: "activo"},
			},
			WorkersConectados:   1,
			WorkersTrabajando:   1,
			SupervisoresActivos: 1,
		}}

		briefing, err := buildSupervisorBriefing("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorBriefing: %v", err)
		}
		if !strings.Contains(briefing, "- Conectados y disponibles: 1") || !strings.Contains(briefing, "- Supervisores activos: 1") || !strings.Contains(briefing, "- Con trabajo activo: 1") {
			t.Fatalf("briefing sin contadores visibles: %s", briefing)
		}

		guidance, err := buildSupervisorGuidance("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorGuidance: %v", err)
		}
		for _, token := range []string{
			"- Flota conectada ahora: 1.",
			"- Flota trabajando ahora: 1.",
			"- Supervisores activos ahora: 1.",
		} {
			if !strings.Contains(guidance, token) {
				t.Fatalf("guidance sin contador visible %q: %s", token, guidance)
			}
		}
	})
}

func TestSupervisorBriefingYGuidanceIncluyenAutonomySurfaceCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			WorkersConectados:   1,
			WorkersTrabajando:   1,
			SupervisoresActivos: 1,
		}}

		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Integrar handoff",
			ProyectoID:  &proyectoID,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "slice",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.RegistrarAutonomyEvent(&db.AutonomyEvent{
			Kind:      "handoff_completed",
			Actor:     "orquesta",
			ProjectID: &proyectoID,
			TaskID:    &tareaID,
			Source:    "control_plane",
			Reason:    "worker_recovered",
			StateDelta: map[string]any{
				"agente_destino": "Codex1",
			},
		}); err != nil {
			t.Fatalf("registrar autonomy event: %v", err)
		}

		briefing, err := buildSupervisorBriefing("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorBriefing: %v", err)
		}
		for _, token := range []string{
			"## Autonomía reciente",
			"handoff_completed=1",
			"Focos canónicos:",
			"[orquestador] handoff_completed destino=Codex1 · worker_recovered",
		} {
			if !strings.Contains(briefing, token) {
				t.Fatalf("briefing sin autonomy surface %q: %s", token, briefing)
			}
		}

		guidance, err := buildSupervisorGuidance("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorGuidance: %v", err)
		}
		if !strings.Contains(guidance, "- Autonomía reciente: 1 evento(s)") || !strings.Contains(guidance, "handoff_completed=1") || !strings.Contains(guidance, "- Focos canónicos de autonomía:") {
			t.Fatalf("guidance sin autonomy surface canónica: %s", guidance)
		}
	})
}

func TestSupervisorBriefingYGuidanceReutilizanAutonomySurfaceCanonicaDelStatus(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		now := time.Now().UTC()
		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			WorkersConectados:   1,
			WorkersTrabajando:   1,
			SupervisoresActivos: 1,
			AutonomySurface: &autonomySurface{
				Events: 1,
				Highlights: []string{
					"handoff_completed=1",
				},
				Recent: []autonomySurfaceRecentItem{{
					Project: "infra",
					autonomyEventSummary: autonomyEventSummary{
						Kind:        "handoff_completed",
						TargetAgent: "Codex1",
						Reason:      "worker_recovered",
						CreatedAt:   now,
					},
				}},
			},
		}}

		briefing, err := buildSupervisorBriefing("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorBriefing: %v", err)
		}
		for _, token := range []string{
			"## Autonomía reciente",
			"handoff_completed=1",
			"[infra] handoff_completed destino=Codex1 · worker_recovered",
		} {
			if !strings.Contains(briefing, token) {
				t.Fatalf("briefing no reutiliza autonomy surface de status %q: %s", token, briefing)
			}
		}

		guidance, err := buildSupervisorGuidance("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorGuidance: %v", err)
		}
		for _, token := range []string{
			"- Autonomía reciente: 1 evento(s)",
			"handoff_completed=1",
			"- Focos canónicos de autonomía:",
		} {
			if !strings.Contains(guidance, token) {
				t.Fatalf("guidance no reutiliza autonomy surface de status %q: %s", token, guidance)
			}
		}
	})
}

func TestSupervisorBriefingYGuidanceIncluyenRiesgoGlobalCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
		}()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			WorkersConectados:   1,
			WorkersTrabajando:   1,
			SupervisoresActivos: 1,
		}}
		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{
				{"slug": "infra"},
				{"slug": "orquestador"},
			}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			switch strings.TrimSpace(slug) {
			case "infra":
				return &apiProyectoCockpit{
					Proyecto:                &db.Proyecto{Slug: "infra"},
					TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas:     1,
					RuntimeOrdersAbiertas:   1,
					RuntimeMailboxPendiente: 1,
					PropuestasAbiertas:      1,
				}, nil
			case "orquestador":
				return &apiProyectoCockpit{
					Proyecto:        &db.Proyecto{Slug: "orquestador"},
					TareasPorEstado: map[string]int{string(db.TareaEnProgreso): 1},
				}, nil
			default:
				return nil, nil
			}
		}

		briefing, err := buildSupervisorBriefing("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorBriefing: %v", err)
		}
		for _, token := range []string{
			"## Riesgo global de integración",
			"Proyecto crítico: infra · integracion_bloqueada=15",
			"riesgo=critico",
			"review_gates=1",
			"runtime_orders=1",
			"## Cola canónica del supervisor",
			"queue_kind=safe",
			"next_safe_action=",
		} {
			if !strings.Contains(briefing, token) {
				t.Fatalf("briefing sin riesgo global %q: %s", token, briefing)
			}
		}

		guidance, err := buildSupervisorGuidance("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorGuidance: %v", err)
		}
		for _, token := range []string{
			"Proyecto crítico por riesgo: infra · integracion_bloqueada=15.",
			"Señales canónicas del proyecto crítico:",
			"riesgo=critico",
			"mailbox_rt=1",
			"Cola canónica del supervisor: queue_kind=safe",
			"next_safe_action=",
			"`next_safe_action`, `safe_action_queue` y `eventos_normalizados`",
		} {
			if !strings.Contains(guidance, token) {
				t.Fatalf("guidance sin riesgo global %q: %s", token, guidance)
			}
		}
	})
}

func TestSupervisorBriefingYGuidanceDestacanFrenteCriticoGlobal(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevRows := supervisorPanelRowsBuilder
		defer func() {
			statusService = prev
			supervisorPanelRowsBuilder = prevRows
		}()
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Integrar API release",
			ProyectoID:  &proyectoID,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "frente critico global",
		})
		if err != nil {
			t.Fatalf("crear tarea critica: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			WorkersConectados: 1,
			WorkersTrabajando: 1,
			TareasActivas: []tareaLite{{
				ID:        tareaID,
				Estado:    db.TareaBloqueada,
				Agente:    "Codex4",
				Titulo:    "Integrar API release",
				Modulo:    "api",
				Prioridad: db.PrioridadAlta,
			}},
			Autonomia: autonomiaResumen{
				Recent: []autonomyEventSummary{{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &tareaID,
					TargetAgent: "Codex4",
					Reason:      "review gate y deploy pendientes",
					Artifacts:   []string{"patch:release-api", "test:go test ./cmd"},
				}},
			},
		}}
		supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
			return []agentesapp.Row{{
				Agente:          &db.Agente{Nombre: "Codex4"},
				EstadoOperativo: "atascado",
			}}, nil
		}

		briefing, err := buildSupervisorBriefing("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorBriefing: %v", err)
		}
		expectedTarget := fmt.Sprintf("tarea:%d -> resolver_followup_bloqueado", tareaID)
		if !strings.Contains(briefing, "## Frente crítico actual") || !strings.Contains(briefing, "proyecto=orquestador") || !strings.Contains(briefing, expectedTarget) || !strings.Contains(briefing, "Bloquea integración/progreso global") {
			t.Fatalf("briefing sin frente critico global: %s", briefing)
		}

		guidance, err := buildSupervisorGuidance("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorGuidance: %v", err)
		}
		if !strings.Contains(guidance, "Frente crítico actual: proyecto=orquestador · "+expectedTarget) || !strings.Contains(guidance, "Bloquea integración/progreso global") {
			t.Fatalf("guidance sin frente critico global: %s", guidance)
		}
	})
}

func TestMCPPromptRevisionSupervisorIncluyeGatesYSignals(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrando Codex4: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Integrar API",
			Descripcion: "Debe aparecer en la cola de revisión",
			ProyectoID:  &proyectoID,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		tareaID2, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Ajustar contrato API",
			Descripcion: "Debe aparecer como colisión de módulo",
			ProyectoID:  &proyectoID,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear segunda tarea: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex3' WHERE id=?`, tareaID); err != nil {
			t.Fatalf("activar tarea Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex4' WHERE id=?`, tareaID2); err != nil {
			t.Fatalf("activar tarea Codex4: %v", err)
		}
		if _, err := db.CrearReviewGate(&db.ReviewGate{
			ProyectoID:     &proyectoID,
			TareaID:        &tareaID,
			RequestedBy:    "orquesta",
			ReviewerAgente: "Codex4",
			Estado:         db.ReviewGateEnRevision,
		}); err != nil {
			t.Fatalf("crear gate: %v", err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex3",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		runtime, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtime, err)
		}
		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtime.ID,
			Kind:        "ready_for_review",
			Level:       "info",
			Message:     "Listo para revisión tras cerrar los tests dirigidos",
			PayloadJSON: `{"classification":"ready_for_review"}`,
		}); err != nil {
			t.Fatalf("registrar runtime event: %v", err)
		}
		if _, err := db.GuardarGitMerge(&db.GitMerge{
			ProyectoID:   proyectoID,
			SourceBranch: "orq/orquestador/Codex3/api",
			TargetBranch: "master",
			RequestedBy:  "OpenClaw",
			Estado:       "aprobado",
			Notas:        "Solicitud creada tras review gate aprobado",
			MetadataJSON: `{"source":"review_gate_approved"}`,
		}); err != nil {
			t.Fatalf("guardar merge: %v", err)
		}

		result, err := getMCPPrompt("orquesta.supervision.revision", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("getMCPPrompt revision supervisor: %v", err)
		}
		messages := result["messages"].([]mcpPromptMessage)
		content := messages[0].Content.(map[string]any)
		text := content["text"].(string)
		for _, token := range []string{
			"Cola de revisión del supervisor: OpenClaw",
			"Review gates abiertos",
			"Integrar API",
			"Señales recientes de revisión e integración",
			"ready_for_review",
			"Solicitudes de merge vivas",
			"orq/orquestador/Codex3/api->master",
			"Riesgos de colisión detectados",
			"Ajustar contrato API",
			"Cola canónica del supervisor",
			"queue_kind=safe",
			"next_safe_action=",
			"critical_project_risk=",
			"Criterio de decisión",
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("falta %q en la cola de revisión: %s", token, text)
			}
		}

		contents, err := readMCPResource("orquesta://supervision/OpenClaw/revision")
		if err != nil {
			t.Fatalf("readMCPResource revision supervisor: %v", err)
		}
		if mimeType, _ := contents[0]["mimeType"].(string); mimeType != "application/json" {
			t.Fatalf("mimeType de revision supervisor = %q, want application/json", mimeType)
		}
		resourceText, _ := contents[0]["text"].(string)
		for _, token := range []string{
			`"queue_kind": "safe"`,
			`"next_safe_action"`,
			`"safe_action_queue"`,
			`"critical_project_risk"`,
			`"ready_for_review"`,
		} {
			if !strings.Contains(resourceText, token) {
				t.Fatalf("recurso de revisión sin contrato canónico %q: %s", token, resourceText)
			}
		}
		foundTemplate := false
		for _, item := range mcpResourceTemplates() {
			if item.URITemplate != "orquesta://supervision/{supervisor}/revision" {
				continue
			}
			foundTemplate = true
			if item.MIMEType != "application/json" {
				t.Fatalf("template revision supervisor mimeType = %q, want application/json", item.MIMEType)
			}
			break
		}
		if !foundTemplate {
			t.Fatal("template MCP de revision supervisor no listado")
		}
	})
}

func TestMCPToolRevisionSupervisorDevuelveJSONEstructurado(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrando Codex4: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tarea1, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Integrar API",
			Descripcion: "Revisión estructurada",
			ProyectoID:  &proyectoID,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea1: %v", err)
		}
		tarea2, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Revisar API",
			Descripcion: "Segunda tarea mismo módulo",
			ProyectoID:  &proyectoID,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea2: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex3' WHERE id=?`, tarea1); err != nil {
			t.Fatalf("activar tarea1: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex4' WHERE id=?`, tarea2); err != nil {
			t.Fatalf("activar tarea2: %v", err)
		}
		if _, err := db.CrearReviewGate(&db.ReviewGate{
			ProyectoID:     &proyectoID,
			TareaID:        &tarea1,
			RequestedBy:    "orquesta",
			ReviewerAgente: "Codex4",
			Estado:         db.ReviewGateEnRevision,
		}); err != nil {
			t.Fatalf("crear gate: %v", err)
		}
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex3",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		runtime, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtime, err)
		}
		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtime.ID,
			Kind:        "approval_request",
			Level:       "warning",
			Message:     "Necesito arbitraje del supervisor",
			PayloadJSON: `{"classification":"approval_request"}`,
		}); err != nil {
			t.Fatalf("registrar event: %v", err)
		}

		result, err := callMCPTool("orquesta.supervision.revision", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("callMCPTool revision supervisor: %v", err)
		}
		structured, _ := result["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent vacío: %#v", result)
		}
		if got, _ := structured["supervisor"].(string); got != "OpenClaw" {
			t.Fatalf("supervisor inesperado: %#v", structured["supervisor"])
		}
		if gates := reflect.ValueOf(structured["review_gates"]); !gates.IsValid() || gates.Len() == 0 {
			t.Fatalf("review_gates vacío: %#v", structured)
		}
		if signals := reflect.ValueOf(structured["signals"]); !signals.IsValid() || signals.Len() == 0 {
			t.Fatalf("signals vacío: %#v", structured)
		}
		if conflicts := reflect.ValueOf(structured["module_conflicts"]); !conflicts.IsValid() || conflicts.Len() == 0 {
			t.Fatalf("module_conflicts vacío: %#v", structured)
		}
		if events := reflect.ValueOf(structured["normalized_events"]); !events.IsValid() || events.Len() == 0 {
			t.Fatalf("normalized_events vacío: %#v", structured)
		}
		if actions := reflect.ValueOf(structured["recommended_actions"]); !actions.IsValid() || actions.Len() == 0 {
			t.Fatalf("recommended_actions vacío: %#v", structured)
		}
		if queue := reflect.ValueOf(structured["action_queue"]); !queue.IsValid() || queue.Len() == 0 {
			t.Fatalf("action_queue vacío: %#v", structured)
		}
		if next := structured["next_action"]; next == nil {
			t.Fatalf("next_action vacío: %#v", structured)
		}
		if queueKind, _ := structured["queue_kind"].(string); queueKind != "safe" {
			t.Fatalf("queue_kind inesperado: %#v", structured["queue_kind"])
		}
		if _, ok := structured["next_safe_action"]; !ok {
			t.Fatalf("falta next_safe_action: %#v", structured)
		}
		if queue := reflect.ValueOf(structured["safe_action_queue"]); !queue.IsValid() {
			t.Fatalf("falta safe_action_queue: %#v", structured)
		}
		if _, ok := structured["critical_project_risk"]; !ok {
			t.Fatalf("falta critical_project_risk: %#v", structured)
		}
		switch queueSummary := structured["queue_summary"].(type) {
		case map[string]any:
			if queueSummary == nil {
				t.Fatalf("falta queue_summary: %#v", structured)
			}
			if _, ok := queueSummary["safe_by_kind"]; !ok {
				t.Fatalf("queue_summary sin safe_by_kind: %#v", queueSummary)
			}
		case apiOpenClawQueueSummary:
			if queueSummary.SafeByKind == nil {
				t.Fatalf("queue_summary sin safe_by_kind: %#v", queueSummary)
			}
		default:
			t.Fatalf("queue_summary con tipo inesperado: %#v", structured["queue_summary"])
		}
		if _, ok := structured["capacity_summary"]; !ok {
			t.Fatalf("falta capacity_summary: %#v", structured)
		}
		if _, ok := structured["saturated_agents"]; !ok {
			t.Fatalf("falta saturated_agents: %#v", structured)
		}
		action0 := reflect.ValueOf(structured["recommended_actions"]).Index(0).Interface()
		raw, err := json.Marshal(action0)
		if err != nil {
			t.Fatalf("marshal recommended action: %v", err)
		}
		if !strings.Contains(string(raw), "Action") && !strings.Contains(string(raw), "action") {
			t.Fatalf("recommended action sin action: %s", string(raw))
		}
	})
}

func TestSupervisorReadOnlySnapshotParsedPreservesCriticalProjectRiskAndSafeQueue(t *testing.T) {
	snapshot := map[string]any{
		"queue_kind": "safe",
		"next_safe_action": map[string]any{
			"kind":           "autonomy_event",
			"target":         "tarea:24",
			"action":         "inspeccionar_handoff_fallido",
			"reason":         "bloqueo real",
			"priority":       "alta",
			"assignee":       "Codex4",
			"auto_aplicable": true,
		},
		"critical_project_risk": map[string]any{
			"project":    "infra",
			"events":     2,
			"blocking":   15,
			"highlights": []any{"riesgo=critico", "integracion_bloqueada=15", "runtime_orders=1"},
		},
	}

	risk := supervisorCriticalProjectRiskFromSnapshot(snapshot)
	if risk == nil || risk.Project != "infra" || risk.Blocking != 15 {
		t.Fatalf("critical_project_risk no parseado desde snapshot read-only: %#v", risk)
	}
	if !reflect.DeepEqual(risk.Highlights, []string{"riesgo=critico", "integracion_bloqueada=15", "runtime_orders=1"}) {
		t.Fatalf("highlights inesperados: %#v", risk.Highlights)
	}
	nextSafeAction := supervisorNextSafeActionFromSnapshot(snapshot)
	if nextSafeAction == nil || nextSafeAction.Target != "tarea:24" || nextSafeAction.Action != "inspeccionar_handoff_fallido" {
		t.Fatalf("next_safe_action no parseado desde snapshot read-only: %#v", nextSafeAction)
	}
	line := formatSupervisorQueueContextLine(snapshot)
	for _, token := range []string{"queue_kind=safe", "next_safe_action=tarea:24 -> inspeccionar_handoff_fallido"} {
		if !strings.Contains(line, token) {
			t.Fatalf("contexto de cola read-only sin token %q: %s", token, line)
		}
	}
}

func TestMCPToolsReviewGatesYGitMergesOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrando Codex4: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Cerrar review MCP",
			Descripcion: "Debe resolverse por tool MCP",
			ProyectoID:  &proyectoID,
			Modulo:      "mcp",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		gateID, err := db.CrearReviewGate(&db.ReviewGate{
			ProyectoID:     &proyectoID,
			TareaID:        &tareaID,
			RequestedBy:    "orquesta",
			ReviewerAgente: "Codex4",
			Estado:         db.ReviewGateEnRevision,
		})
		if err != nil {
			t.Fatalf("crear gate: %v", err)
		}

		listResult, err := callMCPTool("orquesta.review_gates.listar", map[string]any{
			"proyecto": "orquestador",
			"estado":   "en_revision",
		})
		if err != nil {
			t.Fatalf("listar review gates: %v", err)
		}
		listStructured, _ := listResult["structuredContent"].([]*reviewapp.Gate)
		if len(listStructured) == 0 {
			t.Fatalf("review gates vacíos: %#v", listResult)
		}

		resolveResult, err := callMCPTool("orquesta.review_gates.resolver", map[string]any{
			"id":              gateID,
			"estado":          "aprobado",
			"reviewer_agente": "Codex4",
			"findings_json":   `[{"summary":"ok"}]`,
		})
		if err != nil {
			t.Fatalf("resolver review gate: %v", err)
		}
		if resolveResult["isError"] != false {
			t.Fatalf("resolver review gate marcado como error: %#v", resolveResult)
		}
		gate, err := db.GetReviewGate(gateID)
		if err != nil {
			t.Fatalf("get review gate: %v", err)
		}
		if gate == nil || gate.Estado != db.ReviewGateAprobado {
			t.Fatalf("gate no resuelto: %+v", gate)
		}

		saveMergeResult, err := callMCPTool("orquesta.git.merges.guardar", map[string]any{
			"proyecto":      "orquestador",
			"source_branch": "orq/orquestador/Codex4/mcp",
			"target_branch": "master",
			"requested_by":  "OpenClaw",
			"estado":        "aprobado",
			"notas":         "Creado por tool MCP",
		})
		if err != nil {
			t.Fatalf("guardar merge: %v", err)
		}
		if saveMergeResult["isError"] != false {
			t.Fatalf("guardar merge marcado como error: %#v", saveMergeResult)
		}

		listMergeResult, err := callMCPTool("orquesta.git.merges.listar", map[string]any{
			"proyecto": "orquestador",
			"estado":   "aprobado",
		})
		if err != nil {
			t.Fatalf("listar merges: %v", err)
		}
		mergeText := listMergeResult["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(mergeText, "orq/orquestador/Codex4/mcp") {
			t.Fatalf("merge no visible por MCP: %s", mergeText)
		}
	})
}

func TestBuildEstadoResumenLigeroPropagaAuthManual(t *testing.T) {
	prev := statusService
	defer func() { statusService = prev }()

	statusService = stubStatusService{response: apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
		},
		AgentesAuthManual: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador"},
		},
		TareasPorEstado: map[string]int{},
	}}

	resumen, err := buildEstadoResumenLigero()
	if err != nil {
		t.Fatalf("buildEstadoResumenLigero: %v", err)
	}
	if len(resumen.AgentesAuthManual) != 1 || resumen.AgentesAuthManual[0].Nombre != "Codex1" {
		t.Fatalf("agentes auth manual inesperados: %+v", resumen.AgentesAuthManual)
	}
}

func TestMCPGitStatsSurfaceSeLista(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		tools := listMCPTools()
		if !mcpToolListed(tools, "orquesta.git.stats") {
			t.Fatal("tool MCP orquesta.git.stats no registrada")
		}

		resources, err := listMCPResources()
		if err != nil {
			t.Fatalf("listMCPResources: %v", err)
		}
		foundResource := false
		for _, item := range resources {
			if item.URI == "orquesta://git/stats?proyecto=orquestador" {
				foundResource = true
				break
			}
		}
		if !foundResource {
			t.Fatal("resource MCP de git stats por proyecto no listada")
		}

		templates := mcpResourceTemplates()
		foundTemplate := false
		for _, item := range templates {
			if item.URITemplate == "orquesta://git/stats{?proyecto,path,cwd,desde}" {
				foundTemplate = true
				break
			}
		}
		if !foundTemplate {
			t.Fatal("template MCP de git stats no listada")
		}
	})
}

func TestMCPResourceReadGitStatsPorProyecto(t *testing.T) {
	prevService := apiGitStatsService
	prevLookup := apiGitStatsProjectLookupFn
	defer func() {
		apiGitStatsService = prevService
		apiGitStatsProjectLookupFn = prevLookup
	}()

	collector := &gitStatsCollectorStub{
		response: &gitestadisticasapp.Stats{
			RepoRoot:            "/tmp/orquestador",
			CWD:                 "/tmp/orquestador",
			Branch:              "main",
			PendingFiles:        []string{"cmd/mcp.go"},
			RecentCommitFiles:   []string{"cmd/api_git_stats.go"},
			TouchedFiles:        []string{"cmd/api_git_stats.go", "cmd/mcp.go"},
			PendingAddedLines:   7,
			PendingDeletedLines: 2,
			PendingShortStat:    "1 file changed, 7 insertions(+), 2 deletions(-)",
			RecentCommitCount:   3,
		},
	}
	apiGitStatsService = gitOperationalStatsService{collector: collector}
	apiGitStatsProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		return &db.Proyecto{Slug: "orquestador", RutaAbs: "/tmp/orquestador"}, nil
	}

	contents, err := readMCPResource("orquesta://git/stats?proyecto=orquestador&desde=2026-04-24T13:00:00Z")
	if err != nil {
		t.Fatalf("readMCPResource git/stats: %v", err)
	}
	if collector.gotCWD != "/tmp/orquestador" {
		t.Fatalf("cwd inesperado en collector: %q", collector.gotCWD)
	}
	if got := collector.gotSince.UTC().Format(time.RFC3339); got != "2026-04-24T13:00:00Z" {
		t.Fatalf("since inesperado en collector: %s", got)
	}
	text, _ := contents[0]["text"].(string)
	for _, token := range []string{`"scope": "proyecto"`, `"project": "orquestador"`, `"pending_shortstat": "1 file changed, 7 insertions(+), 2 deletions(-)"`, `"files_changed": 2`} {
		if !strings.Contains(text, token) {
			t.Fatalf("resource git/stats sin %q: %s", token, text)
		}
	}
}

func TestMCPToolGitStatsAceptaCwdCanonico(t *testing.T) {
	prevService := apiGitStatsService
	prevLookup := apiGitStatsProjectLookupFn
	defer func() {
		apiGitStatsService = prevService
		apiGitStatsProjectLookupFn = prevLookup
	}()

	collector := &gitStatsCollectorStub{
		response: &gitestadisticasapp.Stats{
			RepoRoot:              "/tmp/repo",
			CWD:                   "/tmp/repo/subdir",
			Branch:                "feature/mcp",
			PendingFiles:          []string{"cmd/mcp.go"},
			RecentCommitFiles:     []string{"cmd/mcp_test.go"},
			TouchedFiles:          []string{"cmd/mcp.go", "cmd/mcp_test.go"},
			PendingAddedLines:     5,
			PendingDeletedLines:   1,
			CommittedAddedLines:   4,
			CommittedDeletedLines: 2,
			RecentCommitCount:     2,
		},
	}
	apiGitStatsService = gitOperationalStatsService{collector: collector}
	apiGitStatsProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		t.Fatalf("project lookup no deberia usarse para cwd, ref=%s", ref)
		return nil, nil
	}

	result, err := callMCPTool("orquesta.git.stats", map[string]any{
		"cwd":   "/tmp/repo/subdir",
		"desde": "2026-04-24T13:00:00Z",
	})
	if err != nil {
		t.Fatalf("callMCPTool git.stats: %v", err)
	}
	if collector.gotCWD != "/tmp/repo/subdir" {
		t.Fatalf("cwd inesperado en collector: %q", collector.gotCWD)
	}
	text := result["content"].([]map[string]any)[0]["text"].(string)
	for _, token := range []string{`"scope": "path"`, `"path": "/tmp/repo/subdir"`, `"files_changed": 2`, `"lines_net": 6`} {
		if !strings.Contains(text, token) {
			t.Fatalf("tool git.stats sin %q: %s", token, text)
		}
	}
	structured, ok := result["structuredContent"].(apiGitStatsResponse)
	if !ok {
		t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
	}
	if structured.Scope != "path" || structured.Path != "/tmp/repo/subdir" {
		t.Fatalf("scope/path inesperados: %+v", structured)
	}
	if structured.Stats == nil || structured.Stats.FilesChanged != 2 || structured.Stats.LinesNet != 6 {
		t.Fatalf("stats inesperados: %+v", structured.Stats)
	}
}

func TestMCPResourceReadGitStatsNormalizaPathCanonico(t *testing.T) {
	prevService := apiGitStatsService
	prevLookup := apiGitStatsProjectLookupFn
	defer func() {
		apiGitStatsService = prevService
		apiGitStatsProjectLookupFn = prevLookup
	}()

	collector := &gitStatsCollectorStub{
		response: &gitestadisticasapp.Stats{
			RepoRoot: "/tmp/repo",
			CWD:      "/tmp/repo",
		},
	}
	apiGitStatsService = gitOperationalStatsService{collector: collector}
	apiGitStatsProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		t.Fatalf("project lookup no deberia usarse para path, ref=%s", ref)
		return nil, nil
	}

	contents, err := readMCPResource("orquesta://git/stats?path=/tmp/repo/./subdir/..")
	if err != nil {
		t.Fatalf("readMCPResource git/stats?path=: %v", err)
	}
	if collector.gotCWD != "/tmp/repo" {
		t.Fatalf("path normalizado inesperado en collector: %q", collector.gotCWD)
	}
	text, _ := contents[0]["text"].(string)
	if !strings.Contains(text, `"path": "/tmp/repo"`) {
		t.Fatalf("resource git/stats sin path canónico: %s", text)
	}
}

func TestMCPToolGitStatsNormalizaCwdCanonico(t *testing.T) {
	prevService := apiGitStatsService
	prevLookup := apiGitStatsProjectLookupFn
	defer func() {
		apiGitStatsService = prevService
		apiGitStatsProjectLookupFn = prevLookup
	}()

	collector := &gitStatsCollectorStub{
		response: &gitestadisticasapp.Stats{
			RepoRoot: "/tmp/repo",
			CWD:      "/tmp/repo",
		},
	}
	apiGitStatsService = gitOperationalStatsService{collector: collector}
	apiGitStatsProjectLookupFn = func(ref string) (*db.Proyecto, error) {
		t.Fatalf("project lookup no deberia usarse para cwd, ref=%s", ref)
		return nil, nil
	}

	result, err := callMCPTool("orquesta.git.stats", map[string]any{
		"cwd": "/tmp/repo/./subdir/..",
	})
	if err != nil {
		t.Fatalf("callMCPTool git.stats: %v", err)
	}
	if collector.gotCWD != "/tmp/repo" {
		t.Fatalf("cwd normalizado inesperado en collector: %q", collector.gotCWD)
	}
	structured, ok := result["structuredContent"].(apiGitStatsResponse)
	if !ok {
		t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
	}
	if structured.Scope != "path" || structured.Path != "/tmp/repo" {
		t.Fatalf("scope/path inesperados tras normalizacion: %+v", structured)
	}
}

func TestMCPRuntimeTraceSurfaceCanonicSeLista(t *testing.T) {
	tools := listMCPTools()
	if !mcpToolListed(tools, "orquesta.runtime.trace") {
		t.Fatal("tool MCP orquesta.runtime.trace no registrada")
	}

	templates := mcpResourceTemplates()
	foundTemplate := false
	for _, item := range templates {
		if item.URITemplate == "orquesta://runtime/trace{?agente,proyecto,handle_id,bytes}" {
			foundTemplate = true
			break
		}
	}
	if !foundTemplate {
		t.Fatal("template MCP de runtime trace no listada")
	}
}

func TestMCPResourceRuntimeTraceCanonicoAceptaAgenteProyectoYBytes(t *testing.T) {
	withTempOrquestaDB(t, func() {
		fixture := setupMCPRuntimeTraceFixture(t)

		contents, err := readMCPResource("orquesta://runtime/trace?agente=" + fixture.agente + "&proyecto=" + fixture.proyecto + "&bytes=10")
		if err != nil {
			t.Fatalf("readMCPResource runtime trace: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("contenido runtime trace vacío")
		}
		text, _ := contents[0]["text"].(string)
		for _, token := range []string{
			`"agente": "CodexTraceMCP"`,
			`"proyecto": "orquestador"`,
			`"truncated": true`,
			`"raw_tail": "vance\nfin\n"`,
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("recurso runtime trace sin %q: %s", token, text)
			}
		}
	})
}

func TestMCPToolRuntimeTraceCanonicoAceptaHandleID(t *testing.T) {
	withTempOrquestaDB(t, func() {
		fixture := setupMCPRuntimeTraceFixture(t)

		result, err := callMCPTool("orquesta.runtime.trace", map[string]any{
			"handle_id": fixture.handleID,
			"bytes":     10,
		})
		if err != nil {
			t.Fatalf("callMCPTool runtime.trace: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		for _, token := range []string{
			fmt.Sprintf(`"handle_id": %d`, fixture.handleID),
			`"agente": "CodexTraceMCP"`,
			`"truncated": true`,
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("tool runtime.trace sin %q: %s", token, text)
			}
		}
		structured, ok := result["structuredContent"].(apiRuntimeTraceResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
		}
		if structured.Trace == nil || structured.Trace.HandleID != fixture.handleID {
			t.Fatalf("trace estructurada inesperada: %+v", structured.Trace)
		}
		if structured.Trace.Agente != fixture.agente || !structured.Trace.Truncated || !strings.Contains(structured.Trace.RawTail, "vance\nfin\n") {
			t.Fatalf("payload trace inesperado: %+v", structured.Trace)
		}
	})
}

func TestMCPRuntimesTreeSurfaceCanonicaSeLista(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		tools := listMCPTools()
		if !mcpToolListed(tools, "orquesta.runtimes.tree") {
			t.Fatal("tool MCP orquesta.runtimes.tree no registrada")
		}

		resources, err := listMCPResources()
		if err != nil {
			t.Fatalf("listMCPResources: %v", err)
		}
		foundResource := false
		for _, item := range resources {
			if item.URI == "orquesta://runtimes/tree" {
				foundResource = true
				break
			}
		}
		if !foundResource {
			t.Fatal("resource MCP de runtimes tree no listada")
		}

		templates := mcpResourceTemplates()
		foundTemplate := false
		for _, item := range templates {
			if item.URITemplate == "orquesta://runtimes/tree{?agente,proyecto,activos}" {
				foundTemplate = true
				break
			}
		}
		if !foundTemplate {
			t.Fatal("template MCP de runtimes tree no listada")
		}
	})
}

func TestMCPResourceRuntimesTreeCanonicoAceptaProyectoYActivos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		fixture := setupMCPRuntimeTreeFixture(t)

		contents, err := readMCPResource("orquesta://runtimes/tree?proyecto=" + fixture.proyecto + "&activos=true")
		if err != nil {
			t.Fatalf("readMCPResource runtimes tree: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("contenido runtimes tree vacío")
		}
		text, _ := contents[0]["text"].(string)
		for _, token := range []string{
			fmt.Sprintf(`"id": %d`, fixture.rootID),
			fmt.Sprintf(`"id": %d`, fixture.childID),
			`"agente": "CodexRuntimeRoot"`,
			`"agente": "CodexRuntimeChild"`,
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("recurso runtimes tree sin %q: %s", token, text)
			}
		}
		if strings.Contains(text, fmt.Sprintf(`"id": %d`, fixture.closedID)) {
			t.Fatalf("recurso runtimes tree no deberia incluir runtime cerrado: %s", text)
		}
	})
}

func TestMCPResourceRuntimesTreeCanonicoFiltraPorAgenteProyectoYActivos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		fixture := setupMCPRuntimeTreeFixture(t)

		contents, err := readMCPResource("orquesta://runtimes/tree?agente=" + fixture.agenteHijo + "&proyecto=" + fixture.proyecto + "&activos=true")
		if err != nil {
			t.Fatalf("readMCPResource runtimes tree filtered: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("contenido runtimes tree filtrado vacío")
		}
		text, _ := contents[0]["text"].(string)
		for _, token := range []string{
			fmt.Sprintf(`"id": %d`, fixture.childID),
			`"agente": "CodexRuntimeChild"`,
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("recurso runtimes tree filtrado sin %q: %s", token, text)
			}
		}
		for _, token := range []string{
			fmt.Sprintf(`"id": %d`, fixture.rootID),
			fmt.Sprintf(`"id": %d`, fixture.closedID),
			`"agente": "CodexRuntimeRoot"`,
			`"agente": "CodexRuntimeClosed"`,
		} {
			if strings.Contains(text, token) {
				t.Fatalf("recurso runtimes tree filtrado no deberia incluir %q: %s", token, text)
			}
		}
	})
}

func TestMCPToolRuntimesTreeCanonicoDevuelveJerarquia(t *testing.T) {
	withTempOrquestaDB(t, func() {
		fixture := setupMCPRuntimeTreeFixture(t)

		result, err := callMCPTool("orquesta.runtimes.tree", map[string]any{
			"proyecto": fixture.proyecto,
			"activos":  true,
		})
		if err != nil {
			t.Fatalf("callMCPTool runtimes.tree: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		for _, token := range []string{
			`"agente": "CodexRuntimeRoot"`,
			`"agente": "CodexRuntimeChild"`,
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("tool runtimes.tree sin %q: %s", token, text)
			}
		}
		structured, ok := result["structuredContent"].(apiRuntimeTreeResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
		}
		if len(structured.Runtimes) != 1 || structured.Runtimes[0] == nil || structured.Runtimes[0].Runtime == nil {
			t.Fatalf("árbol runtime inesperado: %#v", structured.Runtimes)
		}
		if structured.Runtimes[0].Runtime.ID != fixture.rootID {
			t.Fatalf("runtime raíz inesperado: %+v", structured.Runtimes[0].Runtime)
		}
		if len(structured.Runtimes[0].Hijos) != 1 || structured.Runtimes[0].Hijos[0] == nil || structured.Runtimes[0].Hijos[0].Runtime == nil {
			t.Fatalf("hijos runtime inesperados: %#v", structured.Runtimes[0].Hijos)
		}
		if structured.Runtimes[0].Hijos[0].Runtime.ID != fixture.childID {
			t.Fatalf("runtime hijo inesperado: %+v", structured.Runtimes[0].Hijos[0].Runtime)
		}
	})
}

func TestMCPToolRuntimesTreeCanonicoFiltraActivosFalse(t *testing.T) {
	withTempOrquestaDB(t, func() {
		fixture := setupMCPRuntimeTreeFixture(t)

		result, err := callMCPTool("orquesta.runtimes.tree", map[string]any{
			"proyecto": fixture.proyecto,
			"agente":   "CodexRuntimeClosed",
			"activos":  false,
		})
		if err != nil {
			t.Fatalf("callMCPTool runtimes.tree closed: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		for _, token := range []string{
			fmt.Sprintf(`"id": %d`, fixture.closedID),
			`"agente": "CodexRuntimeClosed"`,
			`"logical_state": "cerrado"`,
		} {
			if !strings.Contains(text, token) {
				t.Fatalf("tool runtimes.tree cerrado sin %q: %s", token, text)
			}
		}
		for _, token := range []string{
			fmt.Sprintf(`"id": %d`, fixture.rootID),
			fmt.Sprintf(`"id": %d`, fixture.childID),
			`"agente": "CodexRuntimeRoot"`,
			`"agente": "CodexRuntimeChild"`,
		} {
			if strings.Contains(text, token) {
				t.Fatalf("tool runtimes.tree cerrado no deberia incluir %q: %s", token, text)
			}
		}
		structured, ok := result["structuredContent"].(apiRuntimeTreeResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
		}
		if len(structured.Runtimes) != 1 || structured.Runtimes[0] == nil || structured.Runtimes[0].Runtime == nil {
			t.Fatalf("árbol cerrado inesperado: %#v", structured.Runtimes)
		}
		if structured.Runtimes[0].Runtime.ID != fixture.closedID || structured.Runtimes[0].Runtime.LogicalState != "cerrado" {
			t.Fatalf("runtime cerrado inesperado: %+v", structured.Runtimes[0].Runtime)
		}
	})
}

func TestMCPToolsTareasYNudgeOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		createResult, err := callMCPTool("orquesta.tareas.crear", map[string]any{
			"titulo":      "Cerrar MCP operativo",
			"descripcion": "Debe nacer por tool MCP",
			"proyecto":    "orquestador",
			"modulo":      "mcp",
			"prioridad":   "alta",
			"creado_por":  "OpenClaw",
			"agente":      "Codex3",
		})
		if err != nil {
			t.Fatalf("crear tarea MCP: %v", err)
		}
		if createResult["isError"] != false {
			t.Fatalf("crear tarea marcado como error: %#v", createResult)
		}
		tarea, _ := createResult["structuredContent"].(*db.Tarea)
		if tarea == nil || tarea.ID <= 0 {
			t.Fatalf("tarea estructurada inválida: %#v", createResult)
		}

		startResult, err := callMCPTool("orquesta.tareas.accion", map[string]any{
			"id":     tarea.ID,
			"accion": "iniciar",
			"agente": "Codex3",
		})
		if err != nil {
			t.Fatalf("iniciar tarea MCP: %v", err)
		}
		if startResult["isError"] != false {
			t.Fatalf("iniciar tarea marcado como error: %#v", startResult)
		}

		nudgeResult, err := callMCPTool("orquesta.runtime.nudge", map[string]any{
			"to_agente":   "Codex3",
			"from_agente": "OpenClaw",
			"proyecto":    "orquestador",
			"texto":       "continua con el cierre del frente MCP",
		})
		if err != nil {
			t.Fatalf("runtime nudge MCP: %v", err)
		}
		if nudgeResult["isError"] != false {
			t.Fatalf("runtime nudge marcado como error: %#v", nudgeResult)
		}
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: strPtr("Codex3"), Limit: 10})
		if err != nil {
			t.Fatalf("listar runtime orders: %v", err)
		}
		foundNudge := false
		for _, order := range orders {
			if order != nil && order.Tipo == "nudge" {
				foundNudge = true
				break
			}
		}
		if !foundNudge {
			t.Fatalf("no apareció nudge en runtime orders: %+v", orders)
		}
	})
}

func TestMCPToolsRuntimeYTickOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:     "Codex3",
			ProyectoID: &proyectoID,
		})
		if err != nil {
			t.Fatalf("iniciar sesion contexto: %v", err)
		}
		credits := 50.0
		started := time.Now().UTC().Add(-10 * time.Minute)
		reset := started.Add(5 * time.Hour)
		if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
			SesionID:         sesion.ID,
			WindowKind:       "5h",
			WindowStartedAt:  &started,
			ResetAt:          &reset,
			RemainingCredits: &credits,
			BudgetSource:     "codex_token_count_observed",
			CheckedAt:        time.Now().UTC(),
		}); err != nil {
			t.Fatalf("registrar presupuesto: %v", err)
		}

		if _, err := callMCPTool("orquesta.runtime.nudge", map[string]any{
			"to_agente":   "Codex3",
			"from_agente": "OpenClaw",
			"proyecto":    "orquestador",
			"texto":       "inspecciona el runtime",
		}); err != nil {
			t.Fatalf("runtime nudge MCP: %v", err)
		}
		if _, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "orquesta",
			ToAgente:    "Codex3",
			Kind:        "governance_refresh",
			PayloadJSON: `{"motivo":"refresh"}`,
		}); err != nil {
			t.Fatalf("crear runtime mailbox: %v", err)
		}

		ordersResult, err := callMCPTool("orquesta.runtime.ordenes.listar", map[string]any{
			"agente": "Codex3",
			"limit":  10,
		})
		if err != nil {
			t.Fatalf("listar runtime orders MCP: %v", err)
		}
		if ordersResult["isError"] != false {
			t.Fatalf("runtime orders marcado como error: %#v", ordersResult)
		}
		orders, _ := ordersResult["structuredContent"].([]*db.RuntimeOrder)
		if len(orders) == 0 || orders[0].Tipo == "" {
			t.Fatalf("runtime orders inesperadas: %#v", ordersResult["structuredContent"])
		}

		mailboxResult, err := callMCPTool("orquesta.runtime.mailbox.listar", map[string]any{
			"to_agente": "Codex3",
			"estado":    "pendiente",
		})
		if err != nil {
			t.Fatalf("listar runtime mailbox MCP: %v", err)
		}
		if mailboxResult["isError"] != false {
			t.Fatalf("runtime mailbox marcado como error: %#v", mailboxResult)
		}
		mailbox, _ := mailboxResult["structuredContent"].([]*db.RuntimeMailboxMessage)
		if len(mailbox) == 0 || mailbox[0].Kind == "" {
			t.Fatalf("runtime mailbox inesperada: %#v", mailboxResult["structuredContent"])
		}

		tickResult, err := callMCPTool("orquesta.agentes.tick", map[string]any{
			"agente":    "Codex3",
			"proyecto":  "orquestador",
			"host":      "test-host",
			"pid":       1234,
			"cuota_pct": 42,
		})
		if err != nil {
			t.Fatalf("agente tick MCP: %v", err)
		}
		if tickResult["isError"] != false {
			t.Fatalf("agente tick marcado como error: %#v", tickResult)
		}
		tickOut, _ := tickResult["structuredContent"].(*agentesapp.TickOutput)
		if tickOut == nil || tickOut.Agente != "Codex3" || strings.TrimSpace(tickOut.Proyecto.Slug) != "orquestador" {
			t.Fatalf("tick output inesperado: %#v", tickResult["structuredContent"])
		}
	})
}

func TestMCPToolsRuntimeMailboxGestionanCicloCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrando Codex4: %v", err)
		}
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		sendResult, err := callMCPTool("orquesta.runtime.mailbox.enviar", map[string]any{
			"from_agente":  "OpenClaw",
			"to_agente":    "Codex4",
			"kind":         "governance_refresh",
			"proyecto":     "orquestador",
			"payload_json": `{"motivo":"review"}`,
		})
		if err != nil {
			t.Fatalf("mailbox enviar MCP: %v", err)
		}
		if sendResult["isError"] != false {
			t.Fatalf("mailbox enviar marcado como error: %#v", sendResult)
		}
		sent, _ := sendResult["structuredContent"].(map[string]any)
		id, _ := sent["id"].(int64)
		if id <= 0 {
			t.Fatalf("id mailbox invalido: %#v", sendResult["structuredContent"])
		}

		deliverResult, err := callMCPTool("orquesta.runtime.mailbox.entregar", map[string]any{"id": id})
		if err != nil {
			t.Fatalf("mailbox entregar MCP: %v", err)
		}
		if deliverResult["isError"] != false {
			t.Fatalf("mailbox entregar marcado como error: %#v", deliverResult)
		}

		consumeResult, err := callMCPTool("orquesta.runtime.mailbox.consumir", map[string]any{"id": id})
		if err != nil {
			t.Fatalf("mailbox consumir MCP: %v", err)
		}
		if consumeResult["isError"] != false {
			t.Fatalf("mailbox consumir marcado como error: %#v", consumeResult)
		}

		estado := "consumido"
		mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
			ToAgente: strPtr("Codex4"),
			Estado:   &estado,
		})
		if err != nil {
			t.Fatalf("listar runtime mailbox: %v", err)
		}
		if len(mailbox) == 0 || mailbox[0].ID != id {
			t.Fatalf("mailbox no quedo consumida: %+v", mailbox)
		}
	})
}

func TestMCPToolsRuntimeRecoveryProcesanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevOrders := apiRuntimeProcessOrdersExecutor
		prevMailbox := apiRuntimeProcessMailboxExecutor
		prevAutonomia := runtimeProcessAutonomiaBatch
		prevDegradados := runtimeProcessDegradadosBatchDetailed
		prevHygiene := runtimeProcessHygieneBatch
		prevReanimations := runtimeProcessReanimationsBatchFn
		t.Cleanup(func() {
			apiRuntimeProcessOrdersExecutor = prevOrders
			apiRuntimeProcessMailboxExecutor = prevMailbox
			runtimeProcessAutonomiaBatch = prevAutonomia
			runtimeProcessDegradadosBatchDetailed = prevDegradados
			runtimeProcessHygieneBatch = prevHygiene
			runtimeProcessReanimationsBatchFn = prevReanimations
			runtimeProcessAutonomiaEnCurso.Store(false)
			runtimeProcessDegradadosEnCurso.Store(false)
			runtimeProcessReanimacionesEnCurso.Store(false)
		})

		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		apiRuntimeProcessOrdersExecutor = func() (int, error) { return 7, nil }
		apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
			if filter.ToAgente == nil || *filter.ToAgente != "CodexProcess" {
				t.Fatalf("filtro mailbox inesperado: %+v", filter)
			}
			return 5, nil
		}
		runtimeProcessAutonomiaBatch = func() (int, error) { return 3, nil }
		runtimeProcessDegradadosBatchDetailed = func() (runtimeProcessDegradadosSummary, error) {
			return runtimeProcessDegradadosSummary{
				Count:                     4,
				GhostAssignmentsCompacted: 1,
				ReactivatedWithoutRuntime: 2,
				IdleAutoassigned:          1,
			}, nil
		}
		runtimeProcessHygieneBatch = func() (int, error) { return 6, nil }
		runtimeProcessReanimationsBatchFn = func() apiRuntimeProcessReanimationsResponse {
			return apiRuntimeProcessReanimationsResponse{
				OK:                true,
				Count:             2,
				Candidates:        3,
				Reactivated:       2,
				CooldownSustained: 1,
			}
		}

		ordersResult, err := callMCPTool("orquesta.runtime.process.orders", map[string]any{})
		if err != nil {
			t.Fatalf("runtime process orders MCP: %v", err)
		}
		if ordersResult["isError"] != false {
			t.Fatalf("runtime process orders marcado como error: %#v", ordersResult)
		}
		ordersResp, _ := ordersResult["structuredContent"].(apiRuntimeProcessOrdersResponse)
		if !ordersResp.OK || ordersResp.Count != 7 {
			t.Fatalf("runtime process orders inesperado: %#v", ordersResult["structuredContent"])
		}

		mailboxResult, err := callMCPTool("orquesta.runtime.process.mailbox", map[string]any{
			"to_agente": "CodexProcess",
			"proyecto":  "orquestador",
		})
		if err != nil {
			t.Fatalf("runtime process mailbox MCP: %v", err)
		}
		if mailboxResult["isError"] != false {
			t.Fatalf("runtime process mailbox marcado como error: %#v", mailboxResult)
		}
		mailboxResp, _ := mailboxResult["structuredContent"].(apiRuntimeProcessMailboxResponse)
		if !mailboxResp.OK || mailboxResp.Count != 5 {
			t.Fatalf("runtime process mailbox inesperado: %#v", mailboxResult["structuredContent"])
		}

		autonomiaResult, err := callMCPTool("orquesta.runtime.process.autonomia", map[string]any{"wait": true})
		if err != nil {
			t.Fatalf("runtime process autonomia MCP: %v", err)
		}
		autonomiaResp, _ := autonomiaResult["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !autonomiaResp.OK || autonomiaResp.Count != 3 {
			t.Fatalf("runtime process autonomia inesperado: %#v", autonomiaResult["structuredContent"])
		}

		degradadosResult, err := callMCPTool("orquesta.runtime.process.degradados", map[string]any{"wait": true})
		if err != nil {
			t.Fatalf("runtime process degradados MCP: %v", err)
		}
		degradadosResp, _ := degradadosResult["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !degradadosResp.OK || degradadosResp.Count != 4 || degradadosResp.GhostAssignmentsCompacted != 1 || degradadosResp.ReactivatedWithoutRuntime != 2 || degradadosResp.IdleAutoassigned != 1 {
			t.Fatalf("runtime process degradados inesperado: %#v", degradadosResult["structuredContent"])
		}

		hygieneResult, err := callMCPTool("orquesta.runtime.process.hygiene", map[string]any{"wait": true})
		if err != nil {
			t.Fatalf("runtime process hygiene MCP: %v", err)
		}
		hygieneResp, _ := hygieneResult["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !hygieneResp.OK || hygieneResp.Count != 6 {
			t.Fatalf("runtime process hygiene inesperado: %#v", hygieneResult["structuredContent"])
		}

		reanimationsResult, err := callMCPTool("orquesta.runtime.process.reanimaciones", map[string]any{"wait": true})
		if err != nil {
			t.Fatalf("runtime process reanimaciones MCP: %v", err)
		}
		reanimationsResp, _ := reanimationsResult["structuredContent"].(apiRuntimeProcessReanimationsResponse)
		if !reanimationsResp.OK || reanimationsResp.Count != 2 || reanimationsResp.Reactivated != 2 || reanimationsResp.CooldownSustained != 1 {
			t.Fatalf("runtime process reanimaciones inesperado: %#v", reanimationsResult["structuredContent"])
		}

		wakeResult, err := callMCPTool("orquesta.runtime.wake", map[string]any{
			"orders":  true,
			"mailbox": true,
			"warm":    true,
		})
		if err != nil {
			t.Fatalf("runtime wake MCP: %v", err)
		}
		if wakeResult["isError"] != false {
			t.Fatalf("runtime wake marcado como error: %#v", wakeResult)
		}
		wakeResp, _ := wakeResult["structuredContent"].(apiRuntimeWakeResponse)
		if !wakeResp.Orders || !wakeResp.Mailbox {
			t.Fatalf("runtime wake inesperado: %#v", wakeResult["structuredContent"])
		}
	})
}

func TestMCPToolsRuntimeTranscriptOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevList := apiListRuntimeTranscriptFn
		prevTranscriptBatch := runtimeProcessTranscriptBatch
		prevPurge := apiRuntimePurgeTranscriptNoiseExecutor
		prevPurgeAll := apiRuntimePurgeTranscriptNoiseAllExecutor
		t.Cleanup(func() {
			apiListRuntimeTranscriptFn = prevList
			runtimeProcessTranscriptBatch = prevTranscriptBatch
			apiRuntimePurgeTranscriptNoiseExecutor = prevPurge
			apiRuntimePurgeTranscriptNoiseAllExecutor = prevPurgeAll
			runtimeProcessTranscriptEnCurso.Store(false)
		})

		if err := db.RegistrarAgente("CodexTranscript", "programador"); err != nil {
			t.Fatalf("registrando CodexTranscript: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "CodexTranscript",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
		if err != nil || handle == nil {
			t.Fatalf("runtime handle esperado, got=%+v err=%v", handle, err)
		}

		var got db.FiltroRuntimeTranscript
		apiListRuntimeTranscriptFn = func(filter db.FiltroRuntimeTranscript) ([]*db.RuntimeTranscriptEntry, error) {
			got = filter
			return []*db.RuntimeTranscriptEntry{{
				ID:             77,
				RuntimeID:      55,
				HandleID:       &handle.ID,
				Agente:         "CodexTranscript",
				ProyectoID:     &proyectoID,
				ProyectoSlug:   "orquestador",
				Stream:         "pty_out",
				Text:           "Need approval",
				NormalizedText: "need approval",
				Classification: "approval_request",
				CreatedAt:      time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC),
			}}, nil
		}
		transcriptResult, err := callMCPTool("orquesta.runtime.transcript.listar", map[string]any{
			"agente":          "CodexTranscript",
			"proyecto":        "orquestador",
			"stream":          "pty_out",
			"classification":  "approval_request",
			"q":               "approval",
			"desde":           "2026-04-29T10:00:00Z",
			"signals_pending": true,
			"limit":           10,
		})
		if err != nil {
			t.Fatalf("runtime transcript listar MCP: %v", err)
		}
		if transcriptResult["isError"] != false {
			t.Fatalf("runtime transcript listar marcado como error: %#v", transcriptResult)
		}
		if got.Agente == nil || *got.Agente != "CodexTranscript" || got.ProyectoID == nil || *got.ProyectoID != proyectoID {
			t.Fatalf("filtro transcript inesperado: %+v", got)
		}
		if got.Stream == nil || *got.Stream != "pty_out" || got.Classification == nil || *got.Classification != "approval_request" {
			t.Fatalf("filtro transcript stream/classification inesperado: %+v", got)
		}
		if got.Query == nil || *got.Query != "approval" || got.Desde == nil || got.Desde.UTC().Format(time.RFC3339) != "2026-04-29T10:00:00Z" || !got.SoloSenalesPend || got.Limit != 10 {
			t.Fatalf("filtro transcript restante inesperado: %+v", got)
		}

		transcriptByHandle, err := callMCPTool("orquesta.runtime.process.transcript", map[string]any{"handle_id": handle.ID})
		if err != nil {
			t.Fatalf("runtime process transcript by handle MCP: %v", err)
		}
		handleResp, _ := transcriptByHandle["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !handleResp.OK || handleResp.Count != 0 {
			t.Fatalf("runtime process transcript by handle inesperado: %#v", transcriptByHandle["structuredContent"])
		}

		transcriptByAgent, err := callMCPTool("orquesta.runtime.process.transcript", map[string]any{
			"agente":   "CodexTranscript",
			"proyecto": "orquestador",
		})
		if err != nil {
			t.Fatalf("runtime process transcript by agent MCP: %v", err)
		}
		agentResp, _ := transcriptByAgent["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !agentResp.OK || agentResp.Count != 0 {
			t.Fatalf("runtime process transcript by agent inesperado: %#v", transcriptByAgent["structuredContent"])
		}

		runtimeProcessTranscriptBatch = func() (int, error) { return 4, nil }
		transcriptWait, err := callMCPTool("orquesta.runtime.process.transcript", map[string]any{"wait": true})
		if err != nil {
			t.Fatalf("runtime process transcript wait MCP: %v", err)
		}
		waitResp, _ := transcriptWait["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !waitResp.OK || waitResp.Count != 4 {
			t.Fatalf("runtime process transcript wait inesperado: %#v", transcriptWait["structuredContent"])
		}

		purgeCalls := 0
		apiRuntimePurgeTranscriptNoiseExecutor = func() (int, error) {
			purgeCalls++
			if purgeCalls == 1 {
				return 2, nil
			}
			return 0, nil
		}
		purgeResult, err := callMCPTool("orquesta.runtime.transcript.purgar_ruido", map[string]any{})
		if err != nil {
			t.Fatalf("runtime transcript purgar ruido MCP: %v", err)
		}
		purgeResp, _ := purgeResult["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !purgeResp.OK || purgeResp.Count != 2 || purgeCalls != 1 {
			t.Fatalf("runtime transcript purgar ruido inesperado: resp=%#v calls=%d", purgeResult["structuredContent"], purgeCalls)
		}

		apiRuntimePurgeTranscriptNoiseAllExecutor = func() (int, error) { return 5, nil }
		purgeAllResult, err := callMCPTool("orquesta.runtime.transcript.purgar_ruido", map[string]any{"all": true})
		if err != nil {
			t.Fatalf("runtime transcript purgar ruido all MCP: %v", err)
		}
		purgeAllResp, _ := purgeAllResult["structuredContent"].(apiRuntimeProcessAutonomiaResponse)
		if !purgeAllResp.OK || purgeAllResp.Count != 5 {
			t.Fatalf("runtime transcript purgar ruido all inesperado: %#v", purgeAllResult["structuredContent"])
		}
	})
}

func TestMCPToolsRuntimeAdministrativeControlsOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("CodexRuntime", "programador"); err != nil {
			t.Fatalf("registrando CodexRuntime: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "CodexRuntime",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtimeInst == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtimeInst, err)
		}

		orderID, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
			Agente:      "CodexRuntime",
			ProyectoID:  &proyectoID,
			Tipo:        "send_instruction",
			PayloadJSON: `{"task_id":41}`,
		})
		if err != nil {
			t.Fatalf("crear runtime order: %v", err)
		}
		if _, err := callMCPTool("orquesta.runtime.ordenes.cancelar", map[string]any{
			"id":     orderID,
			"actor":  "OpenClaw",
			"motivo": "limpieza operativa",
		}); err != nil {
			t.Fatalf("runtime orden cancelar MCP: %v", err)
		}

		order, err := runtimesService.GetRuntimeOrder(orderID)
		if err != nil {
			t.Fatalf("leer runtime order: %v", err)
		}
		if order == nil || strings.TrimSpace(order.Estado) != "cancelada" {
			t.Fatalf("runtime order no quedó cancelada: %+v", order)
		}

		purgeResult, err := callMCPTool("orquesta.runtime.ordenes.purgar", map[string]any{
			"agente":   "CodexRuntime",
			"proyecto": "orquestador",
			"actor":    "OpenClaw",
			"estados":  []any{"cancelada"},
		})
		if err != nil {
			t.Fatalf("runtime ordenes purgar MCP: %v", err)
		}
		purgeResp, _ := purgeResult["structuredContent"].(apiRuntimeOrdersPurgeResponse)
		if !purgeResp.OK || purgeResp.Deleted < 1 {
			t.Fatalf("runtime ordenes purgar inesperado: %#v", purgeResult["structuredContent"])
		}

		sendResult, err := callMCPTool("orquesta.runtime.mailbox.enviar", map[string]any{
			"from_agente":  "OpenClaw",
			"to_agente":    "CodexRuntime",
			"kind":         "governance_refresh",
			"proyecto":     "orquestador",
			"payload_json": `{"motivo":"cleanup"}`,
		})
		if err != nil {
			t.Fatalf("runtime mailbox enviar MCP: %v", err)
		}
		sent, _ := sendResult["structuredContent"].(map[string]any)
		mailboxID, _ := sent["id"].(int64)
		if mailboxID <= 0 {
			t.Fatalf("mailbox id invalido: %#v", sendResult["structuredContent"])
		}

		clearResult, err := callMCPTool("orquesta.runtime.mailbox.limpiar", map[string]any{
			"to_agente": "CodexRuntime",
			"proyecto":  "orquestador",
			"estados":   []any{"pendiente"},
			"kinds":     []any{"governance_refresh"},
		})
		if err != nil {
			t.Fatalf("runtime mailbox limpiar MCP: %v", err)
		}
		clearResp, _ := clearResult["structuredContent"].(apiRuntimeMailboxClearResponse)
		if !clearResp.OK || clearResp.Cleared < 1 {
			t.Fatalf("runtime mailbox limpiar inesperado: %#v", clearResult["structuredContent"])
		}

		checkpointCreateResult, err := callMCPTool("orquesta.runtime.checkpoints.crear", map[string]any{
			"agente":          "CodexRuntime",
			"proyecto":        "orquestador",
			"sesion_id":       sesion.ID,
			"runtime_id":      runtimeInst.ID,
			"checkpoint_kind": "manual_override",
			"resumen":         "checkpoint desde MCP",
			"branch":          "candidato_refactor",
			"cwd":             "/tmp/orquestador",
			"payload":         `{"step":"manual_override"}`,
			"resume_strategy": "session_resume",
			"source":          "OpenClaw",
		})
		if err != nil {
			t.Fatalf("runtime checkpoint crear MCP: %v", err)
		}
		createPayload, _ := checkpointCreateResult["structuredContent"].(map[string]any)
		checkpointID, _ := createPayload["id"].(int64)
		if checkpointID <= 0 {
			t.Fatalf("checkpoint id invalido: %#v", checkpointCreateResult["structuredContent"])
		}

		listResult, err := callMCPTool("orquesta.runtime.checkpoints.listar", map[string]any{
			"agente":   "CodexRuntime",
			"proyecto": "orquestador",
			"kind":     "manual_override",
			"source":   "OpenClaw",
			"limit":    5,
		})
		if err != nil {
			t.Fatalf("runtime checkpoints listar MCP: %v", err)
		}
		checkpoints, _ := listResult["structuredContent"].([]*db.RuntimeCheckpoint)
		if len(checkpoints) == 0 || checkpoints[0] == nil || checkpoints[0].ID != checkpointID {
			t.Fatalf("runtime checkpoints inesperados: %#v", listResult["structuredContent"])
		}

		latestResult, err := callMCPTool("orquesta.runtime.checkpoints.latest", map[string]any{
			"agente":   "CodexRuntime",
			"proyecto": "orquestador",
		})
		if err != nil {
			t.Fatalf("runtime checkpoint latest MCP: %v", err)
		}
		latest, _ := latestResult["structuredContent"].(*db.RuntimeCheckpoint)
		if latest == nil || latest.ID != checkpointID || strings.TrimSpace(latest.Source) != "OpenClaw" {
			t.Fatalf("runtime checkpoint latest inesperado: %#v", latestResult["structuredContent"])
		}

		if _, err := callMCPTool("orquesta.runtime.handles.cerrar", map[string]any{
			"agente":   "CodexRuntime",
			"proyecto": "orquestador",
			"actor":    "OpenClaw",
		}); err != nil {
			t.Fatalf("runtime handles cerrar MCP: %v", err)
		}
		if _, err := callMCPTool("orquesta.runtime.handles.purgar", map[string]any{
			"agente":   "CodexRuntime",
			"proyecto": "orquestador",
			"actor":    "OpenClaw",
			"estados":  []any{"cerrado"},
		}); err != nil {
			t.Fatalf("runtime handles purgar MCP: %v", err)
		}
		if _, err := callMCPTool("orquesta.runtimes.cerrar", map[string]any{
			"agente":   "CodexRuntime",
			"proyecto": "orquestador",
			"actor":    "OpenClaw",
		}); err != nil {
			t.Fatalf("runtimes cerrar MCP: %v", err)
		}
	})
}

func TestMCPToolsAgentesListarYPausarOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex5", "programador"); err != nil {
			t.Fatalf("registrando Codex5: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE agentes SET activo=1 WHERE nombre='Codex5'`); err != nil {
			t.Fatalf("activar Codex5: %v", err)
		}

		listResult, err := callMCPTool("orquesta.agentes.listar", map[string]any{})
		if err != nil {
			t.Fatalf("agentes listar MCP: %v", err)
		}
		if listResult["isError"] != false {
			t.Fatalf("agentes listar marcado como error: %#v", listResult)
		}
		agentes, _ := listResult["structuredContent"].([]*db.Agente)
		found := false
		for _, agente := range agentes {
			if agente != nil && strings.TrimSpace(agente.Nombre) == "Codex5" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("agentes listados inesperados: %#v", listResult["structuredContent"])
		}

		pauseResult, err := callMCPTool("orquesta.agentes.pausar", map[string]any{
			"agente":  "Codex5",
			"minutos": 15,
			"motivo":  "prueba MCP",
			"accion":  "openclaw_pause",
			"entidad": "agente",
			"detalle": "Pausa de prueba desde MCP",
		})
		if err != nil {
			t.Fatalf("agentes pausar MCP: %v", err)
		}
		if pauseResult["isError"] != false {
			t.Fatalf("agentes pausar marcado como error: %#v", pauseResult)
		}
		agente, _ := pauseResult["structuredContent"].(*db.Agente)
		if agente == nil || strings.TrimSpace(agente.EstadoCuota) != "enfriamiento" {
			t.Fatalf("agente no quedo pausado: %#v", pauseResult["structuredContent"])
		}
	})
}

func TestMCPToolsAgentesAccionOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
			t.Fatalf("registrando Codex6: %v", err)
		}

		retireResult, err := callMCPTool("orquesta.agentes.accion", map[string]any{
			"agente": "Codex6",
			"accion": "retirar",
		})
		if err != nil {
			t.Fatalf("agentes accion retirar MCP: %v", err)
		}
		if retireResult["isError"] != false {
			t.Fatalf("agentes accion retirar marcado como error: %#v", retireResult)
		}
		agente, _ := retireResult["structuredContent"].(*db.Agente)
		if agente == nil || agente.Habilitado {
			t.Fatalf("agente no quedó retirado: %#v", retireResult["structuredContent"])
		}

		rehabResult, err := callMCPTool("orquesta.agentes.accion", map[string]any{
			"agente": "Codex6",
			"accion": "rehabilitar",
		})
		if err != nil {
			t.Fatalf("agentes accion rehabilitar MCP: %v", err)
		}
		if rehabResult["isError"] != false {
			t.Fatalf("agentes accion rehabilitar marcado como error: %#v", rehabResult)
		}
		agente, _ = rehabResult["structuredContent"].(*db.Agente)
		if agente == nil || !agente.Habilitado {
			t.Fatalf("agente no quedó rehabilitado: %#v", rehabResult["structuredContent"])
		}
	})
}

func TestMCPToolsAgentesRegistrarOperaPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		result, err := callMCPTool("orquesta.agentes.registrar", map[string]any{
			"nombre": "Codex16",
		})
		if err != nil {
			t.Fatalf("agentes registrar MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes registrar marcado como error: %#v", result)
		}
		resp, _ := result["structuredContent"].(map[string]any)
		if resp["ok"] != true || resp["nombre"] != "Codex16" || resp["rol"] != "programador" {
			t.Fatalf("respuesta registrar inesperada: %#v", result["structuredContent"])
		}
		agente, err := agentesService.GetAgent("Codex16")
		if err != nil {
			t.Fatalf("get agent Codex16: %v", err)
		}
		if agente == nil || agente.Nombre != "Codex16" || strings.TrimSpace(agente.Rol) != "programador" {
			t.Fatalf("agente registrado inesperado: %+v", agente)
		}
	})
}

func TestMCPToolsAgentesRegistrarExigeNombreOProveedor(t *testing.T) {
	withTempOrquestaDB(t, func() {
		result, err := callMCPTool("orquesta.agentes.registrar", map[string]any{})
		if err != nil {
			t.Fatalf("agentes registrar sin args MCP: %v", err)
		}
		if result["isError"] != true {
			t.Fatalf("agentes registrar sin args deberia marcar error: %#v", result)
		}
	})
}

func TestMCPToolsAgentesControlOperaPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := agentControlEnqueueFunc
		defer func() { agentControlEnqueueFunc = prev }()

		var seen apiAgenteControlRequest
		agentControlEnqueueFunc = func(req apiAgenteControlRequest) (int64, string, error) {
			seen = req
			return 77, strings.TrimSpace(req.Accion), nil
		}

		result, err := callMCPTool("orquesta.agentes.control", map[string]any{
			"agente":       "Codex9",
			"accion":       "start",
			"proyecto":     "orquestador",
			"conector":     "codex-cli",
			"modelo":       "gpt-5.5",
			"razonamiento": "high",
			"perfil":       "implementacion",
			"motivo":       "launch from mcp",
			"por":          "OpenClaw",
			"tarea_id":     int64(41),
		})
		if err != nil {
			t.Fatalf("agentes control MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes control marcado como error: %#v", result)
		}
		resp, _ := result["structuredContent"].(apiAgenteControlResponse)
		if resp.ID != 77 || resp.Accion != "start" || resp.Agente != "Codex9" {
			t.Fatalf("respuesta control inesperada: %#v", result["structuredContent"])
		}
		if seen.Agente != "Codex9" || seen.Proyecto != "orquestador" || seen.Conector != "codex-cli" || seen.Modelo != "gpt-5.5" {
			t.Fatalf("request control inesperada: %+v", seen)
		}
		if seen.TareaID == nil || *seen.TareaID != 41 {
			t.Fatalf("request control sin tarea esperada: %+v", seen)
		}
	})
}

func TestMCPToolsAgentesLanzarOperaPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		tmp := t.TempDir()
		if err := db.RegistrarAgente("CodexLaunchMCP", "programador"); err != nil {
			t.Fatalf("registrando CodexLaunchMCP: %v", err)
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

		result, err := callMCPTool("orquesta.agentes.lanzar", map[string]any{
			"agente":   "CodexLaunchMCP",
			"proyecto": "orquestador",
			"motivo":   "launch from mcp canonical tool",
			"por":      "OpenClaw",
		})
		if err != nil {
			t.Fatalf("agentes lanzar MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes lanzar marcado como error: %#v", result)
		}
		resp, _ := result["structuredContent"].(apiAgenteLanzarResponse)
		if !resp.OK || resp.OrderID <= 0 {
			t.Fatalf("respuesta lanzar inesperada: %#v", result["structuredContent"])
		}
		if resp.Agente != "CodexLaunchMCP" || resp.Proyecto.Slug != "orquestador" || resp.Conector.Slug != "codex-cli" {
			t.Fatalf("payload lanzar inesperado: %+v", resp)
		}

		order, err := db.GetRuntimeOrder(resp.OrderID)
		if err != nil {
			t.Fatalf("get runtime order: %v", err)
		}
		if order == nil || order.Agente != "CodexLaunchMCP" || order.Tipo != "start" {
			t.Fatalf("runtime order inesperada: %+v", order)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(order.PayloadJSON), &payload); err != nil {
			t.Fatalf("decode runtime order payload: %v", err)
		}
		if strings.TrimSpace(fmt.Sprint(payload["conector"])) != "codex-cli" {
			t.Fatalf("payload sin conector canonico: %+v", payload)
		}
		if strings.TrimSpace(fmt.Sprint(payload["por"])) != "OpenClaw" {
			t.Fatalf("payload sin actor esperado: %+v", payload)
		}
	})
}

func TestMCPToolsAgentesLifecycleOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := agentControlEnqueueFunc
		defer func() { agentControlEnqueueFunc = prev }()

		cases := []struct {
			name       string
			tool       string
			args       map[string]any
			wantAction string
		}{
			{
				name: "start",
				tool: "orquesta.agentes.start",
				args: map[string]any{
					"agente":       "Codex10",
					"proyecto":     "orquestador",
					"conector":     "codex-cli",
					"modelo":       "gpt-5.5",
					"razonamiento": "high",
					"perfil":       "implementacion",
					"motivo":       "launch from lifecycle tool",
					"por":          "OpenClaw",
					"tarea_id":     int64(101),
				},
				wantAction: "start",
			},
			{
				name: "pause",
				tool: "orquesta.agentes.pause",
				args: map[string]any{
					"agente":   "Codex11",
					"proyecto": "orquestador",
					"motivo":   "pause from lifecycle tool",
					"por":      "OpenClaw",
				},
				wantAction: "pause",
			},
			{
				name: "resume",
				tool: "orquesta.agentes.resume",
				args: map[string]any{
					"agente":   "Codex12",
					"proyecto": "orquestador",
					"motivo":   "resume from lifecycle tool",
					"por":      "OpenClaw",
				},
				wantAction: "resume",
			},
			{
				name: "stop",
				tool: "orquesta.agentes.stop",
				args: map[string]any{
					"agente":   "Codex13",
					"proyecto": "orquestador",
					"motivo":   "stop from lifecycle tool",
					"por":      "OpenClaw",
				},
				wantAction: "stop",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var seen apiAgenteControlRequest
				agentControlEnqueueFunc = func(req apiAgenteControlRequest) (int64, string, error) {
					seen = req
					return 88, strings.TrimSpace(req.Accion), nil
				}

				result, err := callMCPTool(tc.tool, tc.args)
				if err != nil {
					t.Fatalf("%s MCP: %v", tc.tool, err)
				}
				if result["isError"] != false {
					t.Fatalf("%s marcado como error: %#v", tc.tool, result)
				}
				resp, _ := result["structuredContent"].(apiAgenteControlResponse)
				if resp.ID != 88 || resp.Accion != tc.wantAction || resp.Agente != strings.TrimSpace(seen.Agente) {
					t.Fatalf("respuesta lifecycle inesperada: %#v", result["structuredContent"])
				}
				if seen.Accion != tc.wantAction {
					t.Fatalf("accion lifecycle inesperada: %+v", seen)
				}
				if seen.Agente != tc.args["agente"] {
					t.Fatalf("agente lifecycle inesperado: %+v", seen)
				}
				if proyecto, _ := tc.args["proyecto"].(string); proyecto != "" && seen.Proyecto != proyecto {
					t.Fatalf("proyecto lifecycle inesperado: %+v", seen)
				}
				if tc.wantAction == "start" {
					if seen.Conector != "codex-cli" || seen.Modelo != "gpt-5.5" || seen.Razonamiento != "high" || seen.Perfil != "implementacion" {
						t.Fatalf("payload start lifecycle inesperado: %+v", seen)
					}
					if seen.TareaID == nil || *seen.TareaID != 101 {
						t.Fatalf("start lifecycle sin tarea esperada: %+v", seen)
					}
				}
			})
		}
	})
}

func TestMCPToolsAgentesHandoffOperaPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
			t.Fatalf("registrando Codex1: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando Codex2: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:     "Handoff MCP",
			ProyectoID: &proyectoID,
			Prioridad:  db.PrioridadAlta,
			CreadoPor:  "OpenClaw",
			Agente:     strPtr("Codex1"),
			Estado:     db.TareaEnProgreso,
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if _, err := db.IniciarSesionContexto(db.SesionInicio{Agente: "Codex1", ProyectoID: &proyectoID}); err != nil {
			t.Fatalf("iniciar sesion origen: %v", err)
		}

		result, err := callMCPTool("orquesta.agentes.handoff", map[string]any{
			"agente_origen":       "Codex1",
			"agente_destino":      "Codex2",
			"tarea_id":            tareaID,
			"motivo":              "presupuesto",
			"resumen":             "continua desde MCP",
			"external_session_id": "sess-mcp-handoff",
		})
		if err != nil {
			t.Fatalf("agentes handoff MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes handoff marcado como error: %#v", result)
		}
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: strPtr("Codex2"), Limit: 10})
		if err != nil {
			t.Fatalf("listar orders destino: %v", err)
		}
		found := false
		for _, order := range orders {
			if order != nil && order.Tipo == "handoff" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no apareció handoff en runtime orders: %+v", orders)
		}
	})
}

func TestMCPToolsAgentesPrepararEInvestigarOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		prepareResult, err := callMCPTool("orquesta.agentes.preparar", map[string]any{
			"agente":   "Codex3",
			"proyecto": "orquestador",
		})
		if err != nil {
			t.Fatalf("agentes preparar MCP: %v", err)
		}
		if prepareResult["isError"] != false {
			t.Fatalf("agentes preparar marcado como error: %#v", prepareResult)
		}
		prepareOut, _ := prepareResult["structuredContent"].(*agentesapp.PrepareOutput)
		if prepareOut == nil || prepareOut.Agente != "Codex3" {
			t.Fatalf("prepare inesperado: %#v", prepareResult["structuredContent"])
		}
		if prepareOut.Proyecto.Slug != "orquestador" {
			t.Fatalf("prepare sin proyecto esperado: %#v", prepareOut.Proyecto)
		}

		investResult, err := callMCPTool("orquesta.agentes.investigar", map[string]any{
			"query":    "router",
			"proyecto": "orquestador",
			"limit":    10,
		})
		if err != nil {
			t.Fatalf("agentes investigar MCP: %v", err)
		}
		if investResult["isError"] != false {
			t.Fatalf("agentes investigar marcado como error: %#v", investResult)
		}
		report, _ := investResult["structuredContent"].(*agentesapp.InvestigationReport)
		if report == nil || report.Query != "router" {
			t.Fatalf("investigation report inesperado: %#v", investResult["structuredContent"])
		}
		if report.Proyecto == nil || report.Proyecto.Slug != "orquestador" {
			t.Fatalf("investigation report sin proyecto esperado: %#v", report.Proyecto)
		}
	})
}

func TestMCPToolsAgentesOverviewYActividadOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex7", "programador"); err != nil {
			t.Fatalf("registrando Codex7: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		if _, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex7",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		}); err != nil {
			t.Fatalf("iniciar sesion Codex7: %v", err)
		}
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:     "MCP agent activity",
			ProyectoID: &proyectoID,
			Modulo:     "cmd",
			Prioridad:  db.PrioridadAlta,
			CreadoPor:  "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.TomarTarea(tareaID, "Codex7"); err != nil {
			t.Fatalf("tomar tarea: %v", err)
		}
		if err := db.IniciarTarea(tareaID, "Codex7"); err != nil {
			t.Fatalf("iniciar tarea: %v", err)
		}

		overviewResult, err := callMCPTool("orquesta.agentes.overview", map[string]any{"agente": "Codex7"})
		if err != nil {
			t.Fatalf("agentes overview MCP: %v", err)
		}
		if overviewResult["isError"] != false {
			t.Fatalf("agentes overview marcado como error: %#v", overviewResult)
		}
		detail, _ := overviewResult["structuredContent"].(*agentesapp.Detail)
		if detail == nil || detail.Entity == nil || detail.Entity.CurrentTask == nil || detail.Entity.CurrentTask.TaskID != tareaID {
			t.Fatalf("overview sin foco de tarea esperado: %#v", overviewResult["structuredContent"])
		}

		activityResult, err := callMCPTool("orquesta.agentes.actividad", map[string]any{
			"agente":   "Codex7",
			"proyecto": "orquestador",
			"desde":    "2h",
		})
		if err != nil {
			t.Fatalf("agentes actividad MCP: %v", err)
		}
		if activityResult["isError"] != false {
			t.Fatalf("agentes actividad marcado como error: %#v", activityResult)
		}
		report, _ := activityResult["structuredContent"].(*agentActivityReport)
		if report == nil || report.Agent != "Codex7" {
			t.Fatalf("activity report inesperado: %#v", activityResult["structuredContent"])
		}
		if len(report.Summary.CurrentTasks) == 0 || !strings.Contains(report.Summary.CurrentTasks[0], "MCP agent activity") {
			t.Fatalf("activity report sin tarea actual esperada: %#v", report.Summary.CurrentTasks)
		}
	})
}

func TestMCPToolsAgentesPanelOperaPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		resetAgentPanelSnapshotCache()
		defer resetAgentPanelSnapshotCache()

		now := time.Now().UTC()
		storeAgentPanelSnapshot([]agentesapp.Row{{
			Agente:           &db.Agente{Nombre: "CodexPanel", Rol: "programador", Activo: true, Habilitado: true},
			EstadoOperativo:  "trabajando",
			DetalleOperativo: "worker ready",
			WorkerAlive:      true,
			WorkerState:      "running",
			MailboxPending:   1,
			OpenTasks:        2,
			CurrentTask:      &agentesapp.TaskFocus{TaskID: 40, Title: "panel canonico", State: db.TareaEnProgreso},
		}}, now)

		result, err := callMCPTool("orquesta.agentes.panel", map[string]any{})
		if err != nil {
			t.Fatalf("agentes panel MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes panel marcado como error: %#v", result)
		}
		resp, _ := result["structuredContent"].(apiAgentesPanelCanonicalResponse)
		if len(resp.Agents) != 1 || resp.Agents[0] == nil || resp.Agents[0].Entity == nil {
			t.Fatalf("panel canónico inesperado: %#v", result["structuredContent"])
		}
		if resp.Agents[0].Entity.Name != "CodexPanel" || resp.Agents[0].OpenTasks != 2 {
			t.Fatalf("panel canónico inesperado: %+v", resp.Agents[0])
		}
	})
}

func TestMCPResourceActividadAgenteAceptaDesde(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex8", "programador"); err != nil {
			t.Fatalf("registrando Codex8: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		if _, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex8",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		}); err != nil {
			t.Fatalf("iniciar sesion Codex8: %v", err)
		}

		contents, err := readMCPResource("orquesta://agentes/Codex8/actividad?desde=2h&proyecto=orquestador")
		if err != nil {
			t.Fatalf("readMCPResource actividad agente: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("contenido actividad agente vacío")
		}
		text, _ := contents[0]["text"].(string)
		for _, token := range []string{`"agent": "Codex8"`, `"project": "orquestador"`} {
			if !strings.Contains(text, token) {
				t.Fatalf("recurso actividad sin %q: %s", token, text)
			}
		}
	})
}

func TestMCPResourceRuntimeTranscriptAceptaFiltros(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("CodexTranscriptResource", "programador"); err != nil {
			t.Fatalf("registrando CodexTranscriptResource: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "CodexTranscriptResource",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtimeInst == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtimeInst, err)
		}
		handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
		if err != nil || handle == nil {
			t.Fatalf("handle esperado, got=%+v err=%v", handle, err)
		}
		byteOffset := int64(64)
		if _, err := db.RegistrarRuntimeTranscript(&db.RuntimeTranscriptEntry{
			RuntimeID:      runtimeInst.ID,
			HandleID:       &handle.ID,
			Agente:         "CodexTranscriptResource",
			ProyectoID:     &proyectoID,
			Stream:         "pty_out",
			ByteOffset:     &byteOffset,
			Text:           "Need approval for deploy",
			NormalizedText: "need approval for deploy",
			Classification: "approval_request",
		}); err != nil {
			t.Fatalf("registrar transcript: %v", err)
		}

		contents, err := readMCPResource("orquesta://runtime/transcript/CodexTranscriptResource?proyecto=orquestador&classification=approval_request&stream=pty_out&desde=2026-04-01T00:00:00Z&limit=5")
		if err != nil {
			t.Fatalf("readMCPResource runtime transcript: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("contenido runtime transcript vacío")
		}
		text, _ := contents[0]["text"].(string)
		for _, token := range []string{`"agente": "CodexTranscriptResource"`, `"classification": "approval_request"`, `"stream": "pty_out"`} {
			if !strings.Contains(text, token) {
				t.Fatalf("recurso transcript sin %q: %s", token, text)
			}
		}
	})
}

func TestMCPResourceRuntimeCheckpointsAceptaProyectoYLatest(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("CodexCheckpointResource", "programador"); err != nil {
			t.Fatalf("registrando CodexCheckpointResource: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "CodexCheckpointResource",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtimeInst == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtimeInst, err)
		}
		id, err := runtimesService.CreateRuntimeCheckpoint(&db.RuntimeCheckpoint{
			Agente:         "CodexCheckpointResource",
			ProyectoID:     &proyectoID,
			SesionID:       &sesion.ID,
			RuntimeID:      &runtimeInst.ID,
			CheckpointKind: "manual_override",
			Resumen:        "checkpoint resource",
			Branch:         "candidato_refactor",
			CWD:            "/tmp/orquestador",
			PayloadJSON:    `{"step":"resource"}`,
			ResumeStrategy: "session_resume",
			Source:         "OpenClaw",
		})
		if err != nil {
			t.Fatalf("create runtime checkpoint: %v", err)
		}
		if id <= 0 {
			t.Fatalf("checkpoint id inválido: %d", id)
		}

		contents, err := readMCPResource("orquesta://runtime/checkpoints/CodexCheckpointResource?proyecto=orquestador&kind=manual_override&source=OpenClaw&limit=5")
		if err != nil {
			t.Fatalf("readMCPResource runtime checkpoints: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("contenido runtime checkpoints vacío")
		}
		text, _ := contents[0]["text"].(string)
		for _, token := range []string{`"agente": "CodexCheckpointResource"`, `"checkpoint_kind": "manual_override"`, `"source": "OpenClaw"`} {
			if !strings.Contains(text, token) {
				t.Fatalf("recurso checkpoints sin %q: %s", token, text)
			}
		}

		latestContents, err := readMCPResource("orquesta://runtime/checkpoints/CodexCheckpointResource/latest?proyecto=orquestador")
		if err != nil {
			t.Fatalf("readMCPResource runtime checkpoint latest: %v", err)
		}
		if len(latestContents) == 0 {
			t.Fatalf("contenido runtime checkpoint latest vacío")
		}
		latestText, _ := latestContents[0]["text"].(string)
		for _, token := range []string{`"agente": "CodexCheckpointResource"`, `"source": "OpenClaw"`} {
			if !strings.Contains(latestText, token) {
				t.Fatalf("recurso checkpoint latest sin %q: %s", token, latestText)
			}
		}
	})
}

func TestMCPToolsSesionesYObservabilidadRuntimeOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex3",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		handle, err := db.GetRuntimeHandleBySesionID(sesion.ID)
		if err != nil || handle == nil {
			t.Fatalf("runtime handle esperado, got=%+v err=%v", handle, err)
		}
		runtime, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtime, err)
		}
		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtime.ID,
			Kind:        "approval_request",
			Level:       "warning",
			Message:     "Necesito confirmación para integrar",
			PayloadJSON: `{"classification":"approval_request"}`,
		}); err != nil {
			t.Fatalf("registrar runtime event: %v", err)
		}

		handlesResult, err := callMCPTool("orquesta.runtime.handles.listar", map[string]any{"agente": "Codex3"})
		if err != nil {
			t.Fatalf("runtime handles listar MCP: %v", err)
		}
		if handlesResult["isError"] != false {
			t.Fatalf("runtime handles listar marcado como error: %#v", handlesResult)
		}
		handles, _ := handlesResult["structuredContent"].([]*db.RuntimeHandle)
		if len(handles) == 0 || handles[0] == nil {
			t.Fatalf("runtime handles vacío: %#v", handlesResult["structuredContent"])
		}

		eventsResult, err := callMCPTool("orquesta.runtime.events.listar", map[string]any{
			"agente":   "Codex3",
			"proyecto": "orquestador",
			"kind":     "approval_request",
			"limit":    5,
		})
		if err != nil {
			t.Fatalf("runtime events listar MCP: %v", err)
		}
		if eventsResult["isError"] != false {
			t.Fatalf("runtime events listar marcado como error: %#v", eventsResult)
		}
		events, _ := eventsResult["structuredContent"].([]*db.RuntimeEvent)
		if len(events) == 0 || events[0] == nil || events[0].Kind != "approval_request" {
			t.Fatalf("runtime events inesperados: %#v", eventsResult["structuredContent"])
		}

		finishResult, err := callMCPTool("orquesta.sesiones.fin", map[string]any{"agente": "Codex3"})
		if err != nil {
			t.Fatalf("sesion fin MCP: %v", err)
		}
		if finishResult["isError"] != false {
			t.Fatalf("sesion fin marcado como error: %#v", finishResult)
		}
		if sesionActiva, err := db.GetSesionActiva("Codex3", &proyectoID); err == nil && sesionActiva != nil {
			t.Fatalf("la sesión siguió activa tras sesion fin: %#v", sesionActiva)
		}
	})
}

func TestMCPEventosSupervisorOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		sesion, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "Codex3",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		})
		if err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		runtime, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtime == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtime, err)
		}
		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtime.ID,
			Kind:        "ready_for_review",
			Level:       "info",
			Message:     "Listo para revisión desde MCP supervisor",
			PayloadJSON: `{"classification":"ready_for_review"}`,
		}); err != nil {
			t.Fatalf("registrar runtime event: %v", err)
		}

		result, err := callMCPTool("orquesta.supervision.eventos", map[string]any{
			"supervisor": "OpenClaw",
			"limit":      5,
		})
		if err != nil {
			t.Fatalf("callMCPTool supervision eventos: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("supervision eventos marcado como error: %#v", result)
		}
		structured, _ := result["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent vacío: %#v", result)
		}
		if got, _ := structured["supervisor"].(string); got != "OpenClaw" {
			t.Fatalf("supervisor inesperado: %#v", structured["supervisor"])
		}
		events := reflect.ValueOf(structured["normalized_events"])
		if !events.IsValid() || events.Len() == 0 {
			t.Fatalf("normalized_events vacío: %#v", structured)
		}

		contents, err := readMCPResource("orquesta://supervision/OpenClaw/eventos?limit=5")
		if err != nil {
			t.Fatalf("readMCPResource supervision eventos: %v", err)
		}
		resourceText, _ := contents[0]["text"].(string)
		for _, token := range []string{`"supervisor": "OpenClaw"`, `"normalized_events"`, `"review.ready"`} {
			if !strings.Contains(resourceText, token) {
				t.Fatalf("recurso supervision eventos sin %q: %s", token, resourceText)
			}
		}

		prompt, err := getMCPPrompt("orquesta.supervision.eventos", map[string]any{
			"supervisor": "OpenClaw",
			"limit":      5,
		})
		if err != nil {
			t.Fatalf("getMCPPrompt supervision eventos: %v", err)
		}
		text := prompt["messages"].([]mcpPromptMessage)[0].Content.(map[string]any)["text"].(string)
		for _, token := range []string{"Eventos del supervisor: OpenClaw", "review.ready", "Listo para revisión desde MCP supervisor"} {
			if !strings.Contains(text, token) {
				t.Fatalf("prompt supervision eventos sin %q: %s", token, text)
			}
		}
	})
}

func TestMCPThreadsSupervisorOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
			t.Fatalf("registrar codex6: %v", err)
		}
		proyecto, err := db.GetProyecto("orquestador")
		if err != nil {
			t.Fatalf("get proyecto: %v", err)
		}
		if _, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:            "Codex6",
			ProyectoID:        &proyecto.ID,
			CWD:               "/tmp/orquestador",
			Herramienta:       "claude-rust",
			ExternalSessionID: "claude-sess-1",
			Host:              "host-claude",
		}); err != nil {
			t.Fatalf("iniciar sesion codex6: %v", err)
		}
		registerResult, err := callMCPTool("orquesta.supervision.threads.registrar", map[string]any{
			"supervisor": "OpenClaw",
			"proyecto":   "orquestador",
			"session_id": "sess-openclaw-1",
			"thread_id":  "leader-1",
			"kind":       "leader",
			"mode":       "review",
			"source":     "mcp_tool",
			"turn_id":    "turn-1",
		})
		if err != nil {
			t.Fatalf("registrar thread supervisor MCP: %v", err)
		}
		if registerResult["isError"] != false {
			t.Fatalf("registrar thread marcado como error: %#v", registerResult)
		}
		if _, err := callMCPTool("orquesta.supervision.threads.registrar", map[string]any{
			"supervisor": "OpenClaw",
			"proyecto":   "orquestador",
			"session_id": "sess-openclaw-1",
			"thread_id":  "sub-1",
			"kind":       "subagent",
			"mode":       "integration",
			"source":     "mcp_tool",
			"turn_id":    "turn-2",
		}); err != nil {
			t.Fatalf("registrar subagent thread MCP: %v", err)
		}
		listResult, err := callMCPTool("orquesta.supervision.threads.listar", map[string]any{
			"supervisor": "OpenClaw",
			"session_id": "sess-openclaw-1",
		})
		if err != nil {
			t.Fatalf("listar threads supervisor MCP: %v", err)
		}
		if listResult["isError"] != false {
			t.Fatalf("listar threads marcado como error: %#v", listResult)
		}
		structured, _ := listResult["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent inesperado: %#v", listResult["structuredContent"])
		}
		sessions := reflect.ValueOf(structured["sessions"])
		if !sessions.IsValid() || sessions.Len() == 0 {
			t.Fatalf("sessions vacío: %#v", structured)
		}
		observed := reflect.ValueOf(structured["observed_agent_sessions"])
		if !observed.IsValid() || observed.Len() == 0 {
			t.Fatalf("observed_agent_sessions vacío: %#v", structured)
		}
		contents, err := readMCPResource("orquesta://supervision/OpenClaw/threads")
		if err != nil {
			t.Fatalf("readMCPResource threads supervisor: %v", err)
		}
		if len(contents) == 0 || !strings.Contains(fmt.Sprintf("%v", contents[0]["text"]), "sess-openclaw-1") {
			t.Fatalf("resource threads inesperado: %#v", contents)
		}
		prompt, err := getMCPPrompt("orquesta.supervision.threads", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("getMCPPrompt threads supervisor: %v", err)
		}
		if !strings.Contains(fmt.Sprintf("%v", prompt), "Threads y subagentes del supervisor") {
			t.Fatalf("prompt threads sin contenido esperado: %#v", prompt)
		}
	})
}

func TestMCPPipelineSupervisorOperaPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		updateResult, err := callMCPTool("orquesta.supervision.pipeline.actualizar", map[string]any{
			"supervisor":    "OpenClaw",
			"proyecto":      "orquestador",
			"pipeline_name": "autopilot",
			"current_phase": "review",
			"status":        "active",
			"metadata_json": `{"mode":"review"}`,
		})
		if err != nil {
			t.Fatalf("actualizar pipeline supervisor MCP: %v", err)
		}
		if updateResult["isError"] != false {
			t.Fatalf("actualizar pipeline marcado como error: %#v", updateResult)
		}
		listResult, err := callMCPTool("orquesta.supervision.pipeline.listar", map[string]any{
			"supervisor": "OpenClaw",
			"proyecto":   "orquestador",
		})
		if err != nil {
			t.Fatalf("listar pipeline supervisor MCP: %v", err)
		}
		if listResult["isError"] != false {
			t.Fatalf("listar pipeline marcado como error: %#v", listResult)
		}
		structured, _ := listResult["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent pipeline inesperado: %#v", listResult["structuredContent"])
		}
		pipelines := reflect.ValueOf(structured["pipelines"])
		if !pipelines.IsValid() || pipelines.Len() == 0 {
			t.Fatalf("pipelines vacío: %#v", structured)
		}
		contents, err := readMCPResource("orquesta://supervision/OpenClaw/pipeline")
		if err != nil {
			t.Fatalf("readMCPResource pipeline supervisor: %v", err)
		}
		if len(contents) == 0 || !strings.Contains(fmt.Sprintf("%v", contents[0]["text"]), "autopilot") {
			t.Fatalf("resource pipeline inesperado: %#v", contents)
		}
		prompt, err := getMCPPrompt("orquesta.supervision.pipeline", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("getMCPPrompt pipeline supervisor: %v", err)
		}
		if !strings.Contains(fmt.Sprintf("%v", prompt), "Pipeline del supervisor") {
			t.Fatalf("prompt pipeline sin contenido esperado: %#v", prompt)
		}
	})
}

func TestMCPSubagentesSupervisorOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		registerResult, err := callMCPTool("orquesta.supervision.subagentes.registrar", map[string]any{
			"supervisor":       "OpenClaw",
			"proyecto":         "orquestador",
			"session_id":       "sess-openclaw-1",
			"parent_thread_id": "leader-1",
			"thread_id":        "sub-exp-1",
			"subagent_name":    "OpenClaw-Explore-1",
			"subagent_type":    "Explore",
			"manifest_path":    "/tmp/sub-exp-1/manifest.json",
			"output_path":      "/tmp/sub-exp-1/output.md",
		})
		if err != nil {
			t.Fatalf("registrar subagente supervisor MCP: %v", err)
		}
		if registerResult["isError"] != false {
			t.Fatalf("registrar subagente marcado como error: %#v", registerResult)
		}
		listResult, err := callMCPTool("orquesta.supervision.subagentes.listar", map[string]any{
			"supervisor": "OpenClaw",
			"proyecto":   "orquestador",
		})
		if err != nil {
			t.Fatalf("listar subagentes supervisor MCP: %v", err)
		}
		if listResult["isError"] != false {
			t.Fatalf("listar subagentes marcado como error: %#v", listResult)
		}
		structured, _ := listResult["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent subagentes inesperado: %#v", listResult["structuredContent"])
		}
		subagents := reflect.ValueOf(structured["subagents"])
		if !subagents.IsValid() || subagents.Len() == 0 {
			t.Fatalf("subagents vacío: %#v", structured)
		}
		contents, err := readMCPResource("orquesta://supervision/OpenClaw/subagents")
		if err != nil {
			t.Fatalf("readMCPResource subagents supervisor: %v", err)
		}
		if len(contents) == 0 || !strings.Contains(fmt.Sprintf("%v", contents[0]["text"]), "sub-exp-1") {
			t.Fatalf("resource subagents inesperado: %#v", contents)
		}
		prompt, err := getMCPPrompt("orquesta.supervision.subagentes", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("getMCPPrompt subagentes supervisor: %v", err)
		}
		if !strings.Contains(fmt.Sprintf("%v", prompt), "Subagentes del supervisor") {
			t.Fatalf("prompt subagentes sin contenido esperado: %#v", prompt)
		}
	})
}

func TestMCPRevisionSupervisorPromueveSubagentesTerminales(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		failed, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:     "OpenClaw",
			ProyectoSlug:   "orquestador",
			SessionID:      "sess-openclaw-1",
			ParentThreadID: "leader-1",
			ThreadID:       "sub-fail-1",
			SubagentName:   "OpenClaw-Verify-1",
			SubagentType:   "verification",
			Status:         "failed",
			ErrorMessage:   "panic",
		})
		if err != nil {
			t.Fatalf("crear subagente fallido: %v", err)
		}
		if _, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:     "OpenClaw",
			ProyectoSlug:   "orquestador",
			SessionID:      "sess-openclaw-1",
			ParentThreadID: "leader-1",
			ThreadID:       "sub-done-1",
			SubagentName:   "OpenClaw-Explore-1",
			SubagentType:   "explore",
			Status:         "completed",
			OutputPath:     "/tmp/sub-done-1/output.md",
		}); err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}
		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		var found bool
		for _, item := range queue {
			if item.Action == "revisar_subagente_fallido" && item.Target == fmt.Sprintf("subagente:%d", failed.ID) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("cola sin subagente fallido: %+v", queue)
		}
		result, err := applySupervisorRecommendedAction("OpenClaw", "revisar_subagente_fallido", fmt.Sprintf("subagente:%d", failed.ID), "OpenClaw")
		if err != nil {
			t.Fatalf("aplicar revisar_subagente_fallido: %v", err)
		}
		subagente, _ := result["subagente"].(*db.SupervisorSubagent)
		if subagente == nil || subagente.ID != failed.ID {
			t.Fatalf("resultado de subagente inesperado: %#v", result["subagente"])
		}
	})
}

func TestMCPSubagentesSupervisorRefrescaStoreClaude(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		storeDir := t.TempDir()
		t.Setenv("CLAWD_AGENT_STORE", storeDir)
		manifestPath := filepath.Join(storeDir, "agent-1.json")
		outputPath := filepath.Join(storeDir, "agent-1.md")
		if err := os.WriteFile(outputPath, []byte("# output\n"), 0o644); err != nil {
			t.Fatalf("write output: %v", err)
		}
		manifest := map[string]any{
			"agentId":      "agent-1",
			"name":         "Claude Explore 1",
			"description":  "explora el repo",
			"subagentType": "Explore",
			"model":        "claude-opus-4-6",
			"status":       "completed",
			"outputFile":   outputPath,
			"manifestFile": manifestPath,
			"createdAt":    time.Now().UTC().Format(time.RFC3339),
			"startedAt":    time.Now().UTC().Format(time.RFC3339),
			"completedAt":  time.Now().UTC().Format(time.RFC3339),
		}
		raw, _ := json.Marshal(manifest)
		if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}

		result, err := callMCPTool("orquesta.supervision.subagentes.refrescar_store", map[string]any{
			"supervisor": "OpenClaw",
			"proyecto":   "orquestador",
		})
		if err != nil {
			t.Fatalf("refrescar store subagentes: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("refrescar store marcado como error: %#v", result)
		}
		items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			Limit:        10,
		})
		if err != nil {
			t.Fatalf("listar subagentes tras refresh: %v", err)
		}
		if len(items) != 1 || items[0].ThreadID != "agent-1" {
			t.Fatalf("subagentes importados inesperados: %+v", items)
		}
		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		var found bool
		for _, item := range queue {
			if item.Action == "recoger_resultado_subagente" && item.Target == "subagente:"+fmt.Sprintf("%d", items[0].ID) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("cola sin recoger_resultado_subagente: %+v", queue)
		}
	})
}

func TestMCPSubagentesSupervisorLanzaExternoYSincronizaStore(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		storeDir := t.TempDir()
		t.Setenv("CLAWD_AGENT_STORE", storeDir)
		launcherPath := filepath.Join(t.TempDir(), "launcher.sh")
		script := `#!/usr/bin/env bash
set -euo pipefail
id="agent-test-001"
manifest="${ORQUESTA_SUBAGENT_STORE}/${id}.json"
output="${ORQUESTA_SUBAGENT_STORE}/${id}.md"
mkdir -p "${ORQUESTA_SUBAGENT_STORE}"
printf '# result\n' > "${output}"
cat > "${manifest}" <<JSON
{"agentId":"${id}","name":"${ORQUESTA_SUBAGENT_NAME}","description":"${ORQUESTA_SUBAGENT_DESCRIPTION}","subagentType":"${ORQUESTA_SUBAGENT_TYPE}","model":"${ORQUESTA_SUBAGENT_MODEL}","status":"running","outputFile":"${output}","manifestFile":"${manifest}","createdAt":"2026-04-02T12:00:00Z","startedAt":"2026-04-02T12:00:00Z"}
JSON
printf 'ok\n'
`
		if err := os.WriteFile(launcherPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write launcher: %v", err)
		}
		t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", launcherPath)

		result, err := callMCPTool("orquesta.supervision.subagentes.lanzar_externo", map[string]any{
			"supervisor":    "OpenClaw",
			"proyecto":      "orquestador",
			"name":          "Claude Explore 1",
			"description":   "explora modulo openclaw",
			"prompt":        "revisa el modulo openclaw y resume riesgos",
			"subagent_type": "explore",
			"model":         "claude-opus-4-6",
		})
		if err != nil {
			t.Fatalf("lanzar subagente externo: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("lanzar externo marcado como error: %#v", result)
		}
		items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			Limit:        10,
		})
		if err != nil {
			t.Fatalf("listar subagentes tras launch: %v", err)
		}
		if len(items) != 1 || items[0].ThreadID != "agent-test-001" || items[0].Status != "running" {
			t.Fatalf("subagentes inesperados tras launch: %+v", items)
		}
	})
}

func TestMCPSubagentesSupervisorRecogeEntregaGitDesdeWorktree(t *testing.T) {
	withTempOrquestaDB(t, func() {
		repoDir := t.TempDir()
		runGitCmdTest(t, repoDir, "init", "-b", "main")
		runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
		runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
		if err := os.MkdirAll(filepath.Join(repoDir, "modulo"), 0o755); err != nil {
			t.Fatalf("mkdir modulo: %v", err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "modulo", "worker.go"), []byte("package modulo\n\nfunc Worker() string { return \"old\" }\n"), 0o644); err != nil {
			t.Fatalf("write worker: %v", err)
		}
		runGitCmdTest(t, repoDir, "add", ".")
		runGitCmdTest(t, repoDir, "commit", "-m", "init")

		projectID := insertTestProyecto(t, "orquestador", "orquestador", repoDir)
		agente := "OpenClaw-Implementa-1"
		if err := agentesService.RegisterAgent(agente, "programador"); err != nil {
			t.Fatalf("registrar agente: %v", err)
		}
		worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
			ProjectRef: "orquestador",
			Agent:      agente,
			Name:       "wt-openclaw-impl-1",
			Branch:     "orq/orquestador/openclaw-impl-1",
			BaseRef:    "main",
			Reason:     "test_subagente_git",
		})
		if err != nil {
			t.Fatalf("prepare worktree: %v", err)
		}
		specID, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
			ProyectoID:        &projectID,
			Titulo:            "modulo/worker.go::Worker",
			ArchivoObjetivo:   "modulo/worker.go",
			SimboloObjetivo:   "Worker",
			Descripcion:       "Actualiza Worker por git",
			WriteSet:          []string{"modulo/worker.go"},
			TestsObligatorios: []string{"go test ./... -count=1"},
			FormatoSalida:     "git_worktree+evidencia",
			CreadoPor:         "alberto",
		})
		if err != nil {
			t.Fatalf("crear especificacion: %v", err)
		}
		despacho, err := runtimesService.DispatchMicroprogramacionInstruction(runtimesapp.MicroprogramacionDispatchRequest{
			AgenteDestino:     agente,
			ProyectoID:        &projectID,
			Mensaje:           "Implementa Worker por git",
			EspecificacionID:  specID,
			ArchivoObjetivo:   "modulo/worker.go",
			SimboloObjetivo:   "Worker",
			WriteSet:          []string{"modulo/worker.go"},
			TestsObligatorios: []string{"go test ./... -count=1"},
			FormatoSalida:     "git_worktree+evidencia",
		})
		if err != nil {
			t.Fatalf("dispatch microprogramacion: %v", err)
		}
		if despacho.RuntimeOrderID <= 0 {
			t.Fatalf("runtime order invalida: %+v", despacho)
		}

		if err := os.WriteFile(filepath.Join(worktree.Path, "modulo", "worker.go"), []byte("package modulo\n\nfunc Worker() string { return \"new\" }\n"), 0o644); err != nil {
			t.Fatalf("write worker in worktree: %v", err)
		}

		subagente, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-openclaw-1",
			ThreadID:     "sub-done-git-1",
			SubagentName: agente,
			SubagentType: "general-purpose",
			Status:       "completed",
			OutputPath:   filepath.Join(worktree.Path, "resultado.md"),
		})
		if err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}

		result, err := applySupervisorRecommendedAction("OpenClaw", "recoger_resultado_subagente", fmt.Sprintf("subagente:%d", subagente.ID), "OpenClaw")
		if err != nil {
			t.Fatalf("aplicar recoger_resultado_subagente: %v", err)
		}
		entregaGit, _ := result["entrega_git"].(map[string]any)
		if entregaGit == nil {
			t.Fatalf("entrega git inesperada: %#v", result["entrega_git"])
		}
		mergeID, ok := entregaGit["git_merge_id"].(int64)
		if !ok || mergeID <= 0 {
			t.Fatalf("entrega git inesperada: %#v", result["entrega_git"])
		}
		order, err := runtimesService.GetRuntimeOrder(despacho.RuntimeOrderID)
		if err != nil || order == nil {
			t.Fatalf("get runtime order: %+v err=%v", order, err)
		}
		if order.Estado != "completada" || !strings.Contains(order.ResultadoJSON, `"receipt_source":"git_worktree"`) {
			t.Fatalf("runtime order sin cierre git: %+v", order)
		}
		merges, err := db.ListarGitMerges(&projectID, "pendiente")
		if err != nil || len(merges) == 0 {
			t.Fatalf("listar merges: %+v err=%v", merges, err)
		}
		if merges[0].SourceBranch != worktree.Branch || merges[0].TargetBranch != "main" {
			t.Fatalf("merge inesperado: %+v", merges[0])
		}
	})
}

func TestBuildSupervisorSubagentActionsPriorizaSlicePipelineParalela(t *testing.T) {
	withTempOrquestaDB(t, func() {
		item, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-pipeline-slice-1",
			ThreadID:     "slice-done-1",
			SubagentName: "OpenClaw-orquestador-implementacion-slice-1",
			SubagentType: "general-purpose",
			Status:       "completed",
			MetadataJSON: `{"source":"pipeline_local_parallel","slice_index":1,"slice_total":2,"task_title":"Frente amplio del control plane","write_set_slice":["cmd/controlplane_support.go","db/controlplane_entities.go"]}`,
		})
		if err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}
		actions := buildSupervisorSubagentActions(map[string]any{
			"subagents": []*db.SupervisorSubagent{item},
			"store":     supervisorSubagentStoreSummary{},
		})
		if len(actions) != 1 {
			t.Fatalf("actions inesperadas: %+v", actions)
		}
		if actions[0].Priority != "alta" {
			t.Fatalf("priority inesperada: %+v", actions[0])
		}
		if !strings.Contains(actions[0].Reason, "slice=1/2") || !strings.Contains(actions[0].Reason, "Frente amplio del control plane") {
			t.Fatalf("reason sin contexto de slice: %+v", actions[0])
		}
	})
}

func TestBuildSupervisorSubagentsSnapshotExponeFollowupsCompactos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		_, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-subagents-followup",
			ThreadID:     "slice-subagents-followup-1",
			SubagentName: "OpenClaw-orquestador-implementacion-slice-followup",
			SubagentType: "general-purpose",
			Status:       "completed",
			MetadataJSON: `{"source":"pipeline_local_parallel","task_id":530,"task_title":"Frente amplio del control plane","slice_index":1,"slice_total":2,"pipeline_parent_followup_dispatched":true,"pipeline_parent_followup_phase":"revision","pipeline_parent_followup_action":"avanzar_fase","pipeline_parent_followup_git_merge_id":91}`,
		})
		if err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}
		snapshot, err := buildSupervisorSubagentsSnapshot("OpenClaw", "orquestador", "", 20)
		if err != nil {
			t.Fatalf("buildSupervisorSubagentsSnapshot: %v", err)
		}
		followups, _ := snapshot["followups"].([]map[string]any)
		if len(followups) != 1 {
			t.Fatalf("followups inesperados: %#v", snapshot["followups"])
		}
		if followups[0]["source"] != "pipeline_local_parallel" || followups[0]["followup_phase"] != "revision" {
			t.Fatalf("followup compacta inesperada: %#v", followups[0])
		}
	})
}

func TestApplySupervisorRecommendedActionDevuelveMetadataSlicePipeline(t *testing.T) {
	withTempOrquestaDB(t, func() {
		subagente, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-pipeline-slice-2",
			ThreadID:     "slice-done-2",
			SubagentName: "OpenClaw-orquestador-implementacion-slice-2",
			SubagentType: "general-purpose",
			Status:       "completed",
			MetadataJSON: `{"source":"pipeline_local_parallel","slice_index":2,"slice_total":2,"task_title":"Frente amplio del control plane","write_set_slice":["cmd/controlplane_support_test.go","db/controlplane_entities_test.go"]}`,
		})
		if err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}
		result, err := applySupervisorRecommendedAction("OpenClaw", "recoger_resultado_subagente", fmt.Sprintf("subagente:%d", subagente.ID), "OpenClaw")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		slice, _ := result["pipeline_slice"].(map[string]any)
		if slice == nil {
			t.Fatalf("pipeline_slice ausente: %#v", result)
		}
		if got := int64SupervisorSubagente(slice["slice_index"]); got != 2 {
			t.Fatalf("slice_index inesperado: %+v", slice)
		}
		if got := stringSliceSupervisorSubagente(slice["write_set_slice"]); len(got) != 2 || got[0] != "cmd/controlplane_support_test.go" {
			t.Fatalf("write_set_slice inesperado: %+v", slice)
		}
	})
}

func TestApplySupervisorRecommendedActionPromuevePipelineTrasEntregaSidecar(t *testing.T) {
	withTempOrquestaDB(t, func() {
		subagente, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-pipeline-slice-followup",
			ThreadID:     "slice-followup-1",
			SubagentName: "OpenClaw-orquestador-implementacion-slice-followup",
			SubagentType: "general-purpose",
			Status:       "completed",
			MetadataJSON: `{"source":"pipeline_local_parallel","slice_index":1,"slice_total":2,"task_title":"Frente amplio del control plane","write_set_slice":["cmd/controlplane_support.go"]}`,
		})
		if err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}

		oldRecoger := mcpRecogerResultadoGitSubagenteFn
		oldDispatch := mcpPipelineDispatchFn
		defer func() {
			mcpRecogerResultadoGitSubagenteFn = oldRecoger
			mcpPipelineDispatchFn = oldDispatch
		}()

		mcpRecogerResultadoGitSubagenteFn = func(item *db.SupervisorSubagent) (map[string]any, error) {
			return map[string]any{
				"git_merge_id":     int64(91),
				"runtime_order_id": int64(77),
			}, nil
		}

		var proyectoLlamado string
		mcpPipelineDispatchFn = func(proyectoSlug string) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, error) {
			proyectoLlamado = strings.TrimSpace(proyectoSlug)
			return &capacidadapp.ResultadoEjecucionPasoPipelineLocal{
				Paso: &capacidadapp.PasoPipelineLocalDeterminista{
					ProyectoSlug: proyectoSlug,
					FaseObjetivo: "revision",
				},
			}, nil
		}

		result, err := applySupervisorRecommendedAction("OpenClaw", "recoger_resultado_subagente", fmt.Sprintf("subagente:%d", subagente.ID), "OpenClaw")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		if proyectoLlamado != "orquestador" {
			t.Fatalf("pipeline dispatch no llamado con proyecto esperado: %q", proyectoLlamado)
		}
		followup, _ := result["pipeline_followup"].(*capacidadapp.ResultadoEjecucionPasoPipelineLocal)
		if followup == nil || followup.Paso == nil || followup.Paso.FaseObjetivo != "revision" {
			t.Fatalf("pipeline_followup inesperado: %#v", result["pipeline_followup"])
		}
	})
}

func TestApplySupervisorRecommendedActionNoDuplicaFollowupPipelineSidecar(t *testing.T) {
	withTempOrquestaDB(t, func() {
		subagente, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			SessionID:    "sess-pipeline-slice-followup-once",
			ThreadID:     "slice-followup-once-1",
			SubagentName: "OpenClaw-orquestador-implementacion-slice-once",
			SubagentType: "general-purpose",
			Status:       "completed",
			MetadataJSON: `{"source":"pipeline_local_parallel","slice_index":1,"slice_total":2,"task_title":"Frente amplio del control plane","write_set_slice":["cmd/controlplane_support.go"]}`,
		})
		if err != nil {
			t.Fatalf("crear subagente completado: %v", err)
		}

		oldRecoger := mcpRecogerResultadoGitSubagenteFn
		oldDispatch := mcpPipelineDispatchFn
		defer func() {
			mcpRecogerResultadoGitSubagenteFn = oldRecoger
			mcpPipelineDispatchFn = oldDispatch
		}()

		mcpRecogerResultadoGitSubagenteFn = func(item *db.SupervisorSubagent) (map[string]any, error) {
			return map[string]any{"git_merge_id": int64(91)}, nil
		}

		calls := 0
		mcpPipelineDispatchFn = func(proyectoSlug string) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, error) {
			calls++
			return &capacidadapp.ResultadoEjecucionPasoPipelineLocal{
				Paso: &capacidadapp.PasoPipelineLocalDeterminista{
					ProyectoSlug: proyectoSlug,
					FaseObjetivo: "revision",
				},
			}, nil
		}

		for i := 0; i < 2; i++ {
			result, err := applySupervisorRecommendedAction("OpenClaw", "recoger_resultado_subagente", fmt.Sprintf("subagente:%d", subagente.ID), "OpenClaw")
			if err != nil {
				t.Fatalf("applySupervisorRecommendedAction iter=%d: %v", i, err)
			}
			if i == 0 {
				if result["pipeline_followup"] == nil {
					t.Fatalf("faltó pipeline_followup en primera recogida: %#v", result)
				}
			} else {
				if result["pipeline_followup"] != nil {
					t.Fatalf("no debería repetir pipeline_followup: %#v", result["pipeline_followup"])
				}
			}
		}
		if calls != 1 {
			t.Fatalf("dispatch duplicado, calls=%d", calls)
		}

		refrescado, err := db.GetSupervisorSubagentByID(subagente.ID)
		if err != nil || refrescado == nil {
			t.Fatalf("GetSupervisorSubagentByID: %+v err=%v", refrescado, err)
		}
		if !strings.Contains(refrescado.MetadataJSON, `"pipeline_parent_followup_dispatched":true`) {
			t.Fatalf("metadata sin marca durable de followup: %s", refrescado.MetadataJSON)
		}
	})
}

func TestLaunchClaudeSubagentExternalPreparaWorktreeYExponeEntornoGit(t *testing.T) {
	withTempOrquestaDB(t, func() {
		repoDir := t.TempDir()
		runGitCmdTest(t, repoDir, "init", "-b", "main")
		runGitCmdTest(t, repoDir, "config", "user.email", "test@example.com")
		runGitCmdTest(t, repoDir, "config", "user.name", "Test User")
		if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# repo\n"), 0o644); err != nil {
			t.Fatalf("write readme: %v", err)
		}
		runGitCmdTest(t, repoDir, "add", ".")
		runGitCmdTest(t, repoDir, "commit", "-m", "init")

		insertTestProyecto(t, "orquestador", "orquestador", repoDir)
		storeDir := t.TempDir()
		t.Setenv("CLAWD_AGENT_STORE", storeDir)
		launcherPath := filepath.Join(t.TempDir(), "launcher.sh")
		script := `#!/usr/bin/env bash
set -euo pipefail
id="agent-test-git-001"
manifest="${ORQUESTA_SUBAGENT_STORE}/${id}.json"
output="${ORQUESTA_SUBAGENT_STORE}/${id}.md"
mkdir -p "${ORQUESTA_SUBAGENT_STORE}"
printf 'cwd=%s\nbranch=%s\nworktree=%s\n' "$PWD" "${ORQUESTA_SUBAGENT_BRANCH}" "${ORQUESTA_SUBAGENT_WORKTREE}" > "${output}"
cat > "${manifest}" <<JSON
{"agentId":"${id}","name":"${ORQUESTA_SUBAGENT_NAME}","description":"${ORQUESTA_SUBAGENT_DESCRIPTION}","subagentType":"${ORQUESTA_SUBAGENT_TYPE}","model":"${ORQUESTA_SUBAGENT_MODEL}","status":"running","outputFile":"${output}","manifestFile":"${manifest}","createdAt":"2026-04-02T12:00:00Z","startedAt":"2026-04-02T12:00:00Z"}
JSON
printf 'ok\n'
`
		if err := os.WriteFile(launcherPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write launcher: %v", err)
		}
		t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", launcherPath)

		resultado, err := launchClaudeSubagentExternal(supervisorSubagentLaunchRequest{
			Supervisor:   "OpenClaw",
			Proyecto:     "orquestador",
			Name:         "OpenClaw-Explore-Git-1",
			Description:  "explora modulo",
			Prompt:       "revisa el modulo y entrega por git",
			SubagentType: "explore",
			Model:        "claude-opus-4-6",
		})
		if err != nil {
			t.Fatalf("launchClaudeSubagentExternal: %v", err)
		}
		if resultado == nil || resultado.Worktree == nil {
			t.Fatalf("resultado sin worktree: %+v", resultado)
		}
		rutaWorktree, _ := resultado.Worktree["path"].(string)
		branch, _ := resultado.Worktree["branch"].(string)
		if strings.TrimSpace(rutaWorktree) == "" || strings.TrimSpace(branch) == "" {
			t.Fatalf("worktree incompleta: %+v", resultado.Worktree)
		}
		if _, err := os.Stat(rutaWorktree); err != nil {
			t.Fatalf("worktree no creada: %v", err)
		}
		outputPath := filepath.Join(storeDir, "agent-test-git-001.md")
		raw, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		texto := string(raw)
		if !strings.Contains(texto, "cwd="+rutaWorktree) || !strings.Contains(texto, "worktree="+rutaWorktree) {
			t.Fatalf("launcher no uso worktree esperada:\n%s", texto)
		}
		if !strings.Contains(texto, "branch="+branch) {
			t.Fatalf("launcher sin branch esperada:\n%s", texto)
		}
		items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			Limit:        10,
		})
		if err != nil {
			t.Fatalf("listar subagentes: %v", err)
		}
		if len(items) != 1 || items[0].SubagentName != "OpenClaw-Explore-Git-1" {
			t.Fatalf("subagentes inesperados: %+v", items)
		}
		if !strings.Contains(items[0].MetadataJSON, `"worktree_id"`) || !strings.Contains(items[0].MetadataJSON, rutaWorktree) {
			t.Fatalf("metadata del subagente sin worktree persistida: %s", items[0].MetadataJSON)
		}
	})
}

func TestLaunchClaudeSubagentExternalPersisteMetadataAdicional(t *testing.T) {
	withTempOrquestaDB(t, func() {
		tmp := t.TempDir()
		repo := filepath.Join(tmp, "repo-launch-meta")
		runGitCmdAPITest(t, tmp, "init", repo)
		runGitCmdAPITest(t, repo, "config", "user.email", "repo@test")
		runGitCmdAPITest(t, repo, "config", "user.name", "Repo Test")
		if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hola\n"), 0o644); err != nil {
			t.Fatalf("write readme: %v", err)
		}
		runGitCmdAPITest(t, repo, "add", "README.md")
		runGitCmdAPITest(t, repo, "commit", "-m", "init")
		if _, err := db.UpsertProyecto(&db.Proyecto{
			Slug:    "orquestador",
			Nombre:  "Orquestador",
			RutaAbs: repo,
			Tipo:    db.ProyectoRepo,
			Activo:  true,
		}); err != nil {
			t.Fatalf("upsert proyecto: %v", err)
		}

		storeDir := filepath.Join(tmp, ".clawd-agents")
		t.Setenv("CLAWD_AGENT_STORE", storeDir)
		launcherPath := filepath.Join(t.TempDir(), "launcher-meta.sh")
		script := `#!/usr/bin/env bash
set -euo pipefail
id="agent-test-meta-001"
manifest="${ORQUESTA_SUBAGENT_STORE}/${id}.json"
output="${ORQUESTA_SUBAGENT_STORE}/${id}.md"
mkdir -p "${ORQUESTA_SUBAGENT_STORE}"
printf 'ok\n' > "${output}"
cat > "${manifest}" <<JSON
{"agentId":"${id}","name":"${ORQUESTA_SUBAGENT_NAME}","description":"${ORQUESTA_SUBAGENT_DESCRIPTION}","subagentType":"${ORQUESTA_SUBAGENT_TYPE}","model":"${ORQUESTA_SUBAGENT_MODEL}","status":"running","outputFile":"${output}","manifestFile":"${manifest}","createdAt":"2026-04-02T12:00:00Z","startedAt":"2026-04-02T12:00:00Z"}
JSON
printf 'ok\n'
`
		if err := os.WriteFile(launcherPath, []byte(script), 0o755); err != nil {
			t.Fatalf("write launcher: %v", err)
		}
		t.Setenv("ORQUESTA_CLAUDE_SUBAGENT_LAUNCHER", launcherPath)

		_, err := launchClaudeSubagentExternal(supervisorSubagentLaunchRequest{
			Supervisor:   "OpenClaw",
			Proyecto:     "orquestador",
			Name:         "OpenClaw-Pipeline-Slice-1",
			Description:  "slice 1",
			Prompt:       "trabaja el slice 1",
			SubagentType: "general-purpose",
			Metadata: map[string]any{
				"source":          "pipeline_local_parallel",
				"slice_index":     1,
				"slice_total":     2,
				"write_set_slice": []string{"cmd/controlplane_support.go", "cmd/controlplane_support_test.go"},
			},
		})
		if err != nil {
			t.Fatalf("launchClaudeSubagentExternal: %v", err)
		}
		items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
			Supervisor:   "OpenClaw",
			ProyectoSlug: "orquestador",
			Limit:        10,
		})
		if err != nil {
			t.Fatalf("listar subagentes: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("subagentes inesperados: %+v", items)
		}
		for _, token := range []string{`"source":"pipeline_local_parallel"`, `"slice_index":1`, `"slice_total":2`, `"write_set_slice":["cmd/controlplane_support.go","cmd/controlplane_support_test.go"]`} {
			if !strings.Contains(items[0].MetadataJSON, token) {
				t.Fatalf("metadata sin token %s: %s", token, items[0].MetadataJSON)
			}
		}
	})
}

func TestMCPRevisionSupervisorSugiereDispatchOperativo(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Cerrar backlog operativo",
			Descripcion: "Debe aparecer como siguiente acción del supervisor",
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		if tareaID == 0 {
			t.Fatalf("id tarea libre inválido: %d", tareaID)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		nextAction, _ := snapshot["next_action"].(supervisorRecommendedAction)
		if nextAction.Action != "asignar_tarea_libre" {
			t.Fatalf("next_action inesperada: %+v", nextAction)
		}
		if nextAction.Target != "tarea:"+itoa(tareaID) {
			t.Fatalf("target inesperado: %+v", nextAction)
		}
		if nextAction.Assignee != "Codex3" {
			t.Fatalf("assignee inesperado: %+v", nextAction)
		}
	})
}

func TestMCPToolSupervisorAplicaDispatch(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Dispatch MCP supervisor",
			Descripcion: "Debe pasar a en_progreso por acción canónica",
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar agente: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
		}}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar", map[string]any{
			"supervisor": "OpenClaw",
			"action":     "asignar_tarea_libre",
			"target":     "tarea:" + itoa(tareaID),
			"assignee":   "Codex3",
		})
		if err != nil {
			t.Fatalf("aplicar accion supervisor: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("accion supervisor marcada como error: %#v", result)
		}

		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea.Estado != db.TareaEnProgreso {
			t.Fatalf("estado tarea inesperado: %+v", tarea)
		}
		if tarea.Agente == nil || *tarea.Agente != "Codex3" {
			t.Fatalf("agente tarea inesperado: %+v", tarea)
		}
	})
}

func TestMCPToolSupervisorAplicaSiguiente(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Dispatch siguiente",
			Descripcion: "Debe aplicarse como siguiente acción segura",
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar agente: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
		}}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar_siguiente", map[string]any{
			"supervisor": "OpenClaw",
		})
		if err != nil {
			t.Fatalf("aplicar siguiente supervisor: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("aplicar siguiente marcado como error: %#v", result)
		}

		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea == nil || tarea.Estado != db.TareaEnProgreso || tarea.Agente == nil || *tarea.Agente != "Codex3" {
			t.Fatalf("siguiente acción no aplicada: %+v", tarea)
		}
	})
}

func TestMCPToolsPropuestasCrearYAccionarOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		createResult, err := callMCPTool("orquesta.propuestas.crear", map[string]any{
			"codigo":        "OP-500",
			"titulo":        "Cerrar huecos del MCP",
			"descripcion":   "Debe nacer por MCP",
			"tipo":          "implementacion",
			"proyecto":      "orquestador",
			"propuesto_por": "OpenClaw",
			"distribuidor":  "OpenClaw",
		})
		if err != nil {
			t.Fatalf("propuestas crear MCP: %v", err)
		}
		if createResult["isError"] != false {
			t.Fatalf("propuestas crear marcado como error: %#v", createResult)
		}

		actionResult, err := callMCPTool("orquesta.propuestas.accion", map[string]any{
			"codigo":        "OP-500",
			"accion":        "cerrar",
			"estado_cierre": string(db.PropuestaBacklog),
			"agente":        "OpenClaw",
		})
		if err != nil {
			t.Fatalf("propuestas accion MCP: %v", err)
		}
		if actionResult["isError"] != false {
			t.Fatalf("propuestas accion marcado como error: %#v", actionResult)
		}
		propuesta, _ := actionResult["structuredContent"].(*db.Propuesta)
		if propuesta == nil || propuesta.Codigo != "OP-500" || propuesta.Estado != db.PropuestaBacklog {
			t.Fatalf("propuesta no quedo cerrada: %#v", actionResult["structuredContent"])
		}
	})
}

func TestMCPRevisionSupervisorSugiereAssigneePorMenorCargaEnReplanificacion(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Runtime", Agente: "Codex3"},
				{ID: 411, Estado: db.TareaEnProgreso, Titulo: "Server", Agente: "Codex4"},
				{ID: 414, Estado: db.TareaEnProgreso, Titulo: "Web", Agente: "Codex4"},
				{ID: 416, Estado: db.TareaEnProgreso, Titulo: "Controlplane", Agente: "Codex2"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		nextAction, _ := snapshot["next_action"].(supervisorRecommendedAction)
		if nextAction.Action != "replanificar_por_cuota" {
			t.Fatalf("next_action inesperada: %+v", nextAction)
		}
		if nextAction.Target != "tarea:416" {
			t.Fatalf("target inesperado: %+v", nextAction)
		}
		if nextAction.Assignee != "Codex3" {
			t.Fatalf("assignee por menor carga inesperado: %+v", nextAction)
		}
	})
}

func TestMCPRevisionSupervisorSugiereReservaCuandoNoHayIdle(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		tareaLibreID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Reserva backlog",
			Descripcion: "Debe sugerirse reserva sin start",
			Modulo:      "openclaw",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Runtime", Agente: "Codex3"},
				{ID: 411, Estado: db.TareaEnProgreso, Titulo: "Server", Agente: "Codex4"},
				{ID: 412, Estado: db.TareaEnProgreso, Titulo: "DB", Agente: "Codex3"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "reservar_tarea_libre" && item.Target == "tarea:"+itoa(tareaLibreID) && item.Assignee == "Codex4" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece reserva de backlog libre: %+v", queue)
		}
	})
}

func TestMCPToolSupervisorReservaTareaLibre(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		tareaLibreID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Reserva real",
			Descripcion: "Debe quedar asignada sin start",
			Modulo:      "openclaw",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrar Codex4: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Runtime", Agente: "Codex3"},
				{ID: 411, Estado: db.TareaEnProgreso, Titulo: "Server", Agente: "Codex4"},
				{ID: 412, Estado: db.TareaEnProgreso, Titulo: "DB", Agente: "Codex3"},
			},
		}}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar", map[string]any{
			"supervisor": "OpenClaw",
			"action":     "reservar_tarea_libre",
			"target":     "tarea:" + itoa(tareaLibreID),
			"assignee":   "Codex4",
		})
		if err != nil {
			t.Fatalf("reservar tarea libre: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("reserva marcada como error: %#v", result)
		}
		tarea, err := db.GetTarea(tareaLibreID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea.Estado != db.TareaAsignada || tarea.Agente == nil || *tarea.Agente != "Codex4" {
			t.Fatalf("tarea no quedó reservada: %+v", tarea)
		}
	})
}

func TestMCPRevisionSupervisorSugiereRebalanceoDeReserva(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Runtime", Agente: "Codex3"},
				{ID: 412, Estado: db.TareaEnProgreso, Titulo: "Persistencia", Agente: "Codex3"},
				{ID: 413, Estado: db.TareaEnProgreso, Titulo: "Git", Agente: "Codex4"},
				{ID: 414, Estado: db.TareaEnProgreso, Titulo: "Web", Agente: "Codex4"},
				{ID: 411, Estado: db.TareaAsignada, Titulo: "CLI/API", Agente: "Codex4"},
				{ID: 415, Estado: db.TareaAsignada, Titulo: "OpenClaw", Agente: "Codex4"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "rebalancear_reserva" && item.Target == "tarea:411" && item.Assignee == "Codex3" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece rebalanceo de reserva: %+v", queue)
		}
	})
}

func TestMCPToolSupervisorRebalanceaReserva(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Reserva a rebalancear",
			Descripcion: "Debe cambiar de assignee sin start",
			Modulo:      "openclaw",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrar Codex4: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente='Codex4' WHERE id=?`, tareaID); err != nil {
			t.Fatalf("asignar tarea a Codex4: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Runtime", Agente: "Codex3"},
				{ID: 412, Estado: db.TareaEnProgreso, Titulo: "Persistencia", Agente: "Codex3"},
				{ID: 413, Estado: db.TareaEnProgreso, Titulo: "Git", Agente: "Codex4"},
				{ID: 414, Estado: db.TareaEnProgreso, Titulo: "Web", Agente: "Codex4"},
				{ID: tareaID, Estado: db.TareaAsignada, Titulo: "Reserva", Agente: "Codex4"},
				{ID: 415, Estado: db.TareaAsignada, Titulo: "Reserva 2", Agente: "Codex4"},
			},
		}}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar", map[string]any{
			"supervisor": "OpenClaw",
			"action":     "rebalancear_reserva",
			"target":     "tarea:" + itoa(tareaID),
			"assignee":   "Codex3",
		})
		if err != nil {
			t.Fatalf("rebalancear reserva: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("rebalanceo marcado como error: %#v", result)
		}
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea.Estado != db.TareaAsignada || tarea.Agente == nil || *tarea.Agente != "Codex3" {
			t.Fatalf("tarea no quedó rebalanceada: %+v", tarea)
		}
	})
}

func TestMCPRevisionSupervisorSugiereCerrarPropuestaRechazada(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			PropuestasResumen: []propuestaLite{
				{Codigo: "OP-095", Titulo: "Orquestacion mixta", Estado: db.PropuestaAbierta, Acuerdo: 1, Desacuerdo: 2, Abstencion: 0, Pendiente: 0},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "cerrar_propuesta_rechazada" && item.Target == "propuesta:OP-095" && item.Assignee == "OpenClaw" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece cierre de propuesta rechazada: %+v", queue)
		}
	})
}

func TestMCPToolSupervisorCierraPropuestaRechazada(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
			t.Fatalf("registrar Codex1: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		id, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
			Codigo:       "OP-600",
			Titulo:       "Cerrar propuesta rechazada",
			Descripcion:  "Debe cerrarse desde OpenClaw",
			Tipo:         "implementacion",
			PropuestoPor: "antigravity",
		})
		if err != nil {
			t.Fatalf("crear propuesta: %v", err)
		}
		if _, err := db.Votar(id, "Codex1", db.VotoAcuerdo, "ok"); err != nil {
			t.Fatalf("voto acuerdo: %v", err)
		}
		if _, err := db.Votar(id, "Codex2", db.VotoDesacuerdo, "no"); err != nil {
			t.Fatalf("voto desacuerdo: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.Votar(id, "Codex3", db.VotoDesacuerdo, "no"); err != nil {
			t.Fatalf("segundo voto desacuerdo: %v", err)
		}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar", map[string]any{
			"supervisor": "OpenClaw",
			"action":     "cerrar_propuesta_rechazada",
			"target":     "propuesta:OP-600",
			"assignee":   "OpenClaw",
		})
		if err != nil {
			t.Fatalf("cerrar propuesta rechazada: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("cierre de propuesta marcado como error: %#v", result)
		}
		propuesta, err := db.GetPropuesta("OP-600")
		if err != nil {
			t.Fatalf("get propuesta: %v", err)
		}
		if propuesta == nil || propuesta.Estado != db.PropuestaRechazada {
			t.Fatalf("propuesta no quedó rechazada: %+v", propuesta)
		}
	})
}

func TestMCPToolSupervisorAplicaLoteFiltradoPorProposal(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		if err := db.RegistrarAgente("Codex1", "programador"); err != nil {
			t.Fatalf("registrar Codex1: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrar Codex4: %v", err)
		}

		id, _, err := propuestasService.Create(propuestasapp.CreateProposalInput{
			Codigo:       "OP-602",
			Titulo:       "Cerrar propuesta por lote",
			Descripcion:  "Debe cerrarse sin mezclar dispatch",
			Tipo:         "implementacion",
			PropuestoPor: "antigravity",
		})
		if err != nil {
			t.Fatalf("crear propuesta: %v", err)
		}
		if _, err := db.Votar(id, "Codex1", db.VotoAcuerdo, "ok"); err != nil {
			t.Fatalf("voto acuerdo: %v", err)
		}
		if _, err := db.Votar(id, "Codex2", db.VotoDesacuerdo, "no"); err != nil {
			t.Fatalf("voto desacuerdo: %v", err)
		}
		if _, err := db.Votar(id, "Codex3", db.VotoDesacuerdo, "no"); err != nil {
			t.Fatalf("segundo voto desacuerdo: %v", err)
		}
		agenteCodex3 := "Codex3"
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Reserva para no mezclar",
			Descripcion: "Debe quedar intacta si el lote filtra proposal",
			Modulo:      "cmd",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='asignada', agente=? WHERE id=?`, agenteCodex3, tareaID); err != nil {
			t.Fatalf("preparar tarea asignada: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			PropuestasResumen: []propuestaLite{
				{Codigo: "OP-602", Titulo: "Cerrar propuesta por lote", Estado: db.PropuestaAbierta, Acuerdo: 1, Desacuerdo: 2, Pendiente: 0},
			},
			TareasActivas: []tareaLite{
				{ID: tareaID, Estado: db.TareaAsignada, Titulo: "Reserva para no mezclar", Agente: "Codex3", Modulo: "cmd"},
				{ID: 999, Estado: db.TareaEnProgreso, Titulo: "Carga activa en Codex3", Agente: "Codex3", Modulo: "db"},
				{ID: 1000, Estado: db.TareaEnProgreso, Titulo: "Carga activa en Codex3 2", Agente: "Codex3", Modulo: "web"},
				{ID: 1001, Estado: db.TareaEnProgreso, Titulo: "Carga activa en Codex4", Agente: "Codex4", Modulo: "api"},
			},
		}}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar_lote", map[string]any{
			"supervisor": "OpenClaw",
			"kind":       "proposal",
			"max_items":  5,
		})
		if err != nil {
			t.Fatalf("aplicar lote proposal: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("lote proposal marcado como error: %#v", result)
		}
		structured, _ := result["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent vacío: %#v", result)
		}
		if got, _ := structured["kind"].(string); got != "proposal" {
			t.Fatalf("kind inesperado: %#v", structured)
		}
		propuesta, err := db.GetPropuesta("OP-602")
		if err != nil {
			t.Fatalf("get propuesta: %v", err)
		}
		if propuesta == nil || propuesta.Estado != db.PropuestaRechazada {
			t.Fatalf("propuesta no quedó rechazada: %+v", propuesta)
		}
		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea == nil || tarea.Agente == nil || *tarea.Agente != "Codex3" || tarea.Estado != db.TareaAsignada {
			t.Fatalf("dispatch mezclado en lote proposal: %+v", tarea)
		}
	})
}

func TestMCPRevisionSupervisorExponeTodasLasRetenidasPorCuota(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: 410, Estado: db.TareaEnProgreso, Titulo: "Runtime", Agente: "Codex3"},
				{ID: 411, Estado: db.TareaEnProgreso, Titulo: "Server", Agente: "Codex4"},
				{ID: 416, Estado: db.TareaEnProgreso, Titulo: "Controlplane", Agente: "Codex2"},
				{ID: 421, Estado: db.TareaEnProgreso, Titulo: "Eventos", Agente: "Codex2"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		if len(queue) < 2 {
			t.Fatalf("cola operativa insuficiente: %+v", queue)
		}
		targets := make(map[string]bool, len(queue))
		for _, item := range queue {
			if item.Action != "replanificar_por_cuota" {
				continue
			}
			targets[item.Target] = true
		}
		if !targets["tarea:416"] || !targets["tarea:421"] {
			t.Fatalf("faltan retenidas por cuota en action_queue: %+v", queue)
		}
	})
}

func TestMCPRevisionSupervisorExponeGuidanceDurablePendiente(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    "Codex3",
			Kind:        "autonomia",
			PayloadJSON: `{"texto":"continua"}`,
			Estado:      "pendiente",
		}); err != nil {
			t.Fatalf("crear mailbox pendiente: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "seguir_guidance_durable" && item.Target == "agente:Codex3" && item.Assignee == "Codex3" && item.Priority != "" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece acción por guidance durable pendiente: %+v", queue)
		}
	})
}

func TestMCPRevisionSupervisorExponeSesionObservadaReutilizable(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
			t.Fatalf("registrar Codex6: %v", err)
		}
		sesionID, err := db.IniciarSesion("Codex6")
		if err != nil {
			t.Fatalf("iniciar sesion Codex6: %v", err)
		}
		now := time.Now().UTC()
		rawClaude := `{"account_email":"carlos@avidad.com","account_user":"Carlos Claude","session_usage":{"session_path":"/tmp/.claude/sessions/session-codex6.json","message_count":7,"turns":2,"updated_at":"` + now.Format(time.RFC3339Nano) + `"}}`
		if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
			SesionID:        sesionID,
			WindowKind:      "unknown",
			BudgetSource:    "claude_rust_session_observed",
			RawSnapshotJSON: rawClaude,
			CheckedAt:       now,
		}); err != nil {
			t.Fatalf("registrar presupuesto Claude observado: %v", err)
		}
		if err := db.FinSesion("Codex6"); err != nil {
			t.Fatalf("fin sesion Codex6: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			Agentes: []*db.Agente{
				{Nombre: "Codex6", Rol: "programador", Activo: false, EstadoCuota: "activo"},
			},
		}}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "inspeccionar_sesion_observada" && item.Target == "agente:Codex6" && item.Assignee == "Codex6" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece acción para sesión observada reutilizable: %+v", queue)
		}
	})
}

func TestMCPToolSupervisorInspeccionaSesionObservada(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		if err := db.RegistrarAgente("Codex6", "programador"); err != nil {
			t.Fatalf("registrar Codex6: %v", err)
		}
		sesionID, err := db.IniciarSesion("Codex6")
		if err != nil {
			t.Fatalf("iniciar sesion Codex6: %v", err)
		}
		now := time.Now().UTC()
		rawClaude := `{"account_email":"carlos@avidad.com","account_user":"Carlos Claude","session_usage":{"session_path":"/tmp/.claude/sessions/session-codex6.json","message_count":7,"turns":2,"updated_at":"` + now.Format(time.RFC3339Nano) + `"}}`
		if _, err := db.RegistrarPresupuestoSesion(&db.PresupuestoSesion{
			SesionID:        sesionID,
			WindowKind:      "unknown",
			BudgetSource:    "claude_rust_session_observed",
			RawSnapshotJSON: rawClaude,
			CheckedAt:       now,
		}); err != nil {
			t.Fatalf("registrar presupuesto Claude observado: %v", err)
		}
		if err := db.FinSesion("Codex6"); err != nil {
			t.Fatalf("fin sesion Codex6: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			Agentes: []*db.Agente{
				{Nombre: "Codex6", Rol: "programador", Activo: false, EstadoCuota: "activo"},
			},
		}}

		result, err := applySupervisorRecommendedAction("OpenClaw", "inspeccionar_sesion_observada", "agente:Codex6", "Codex6")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		candidate, _ := result["session_candidate"].(apiOpenClawSessionCandidate)
		if candidate.Agente != "Codex6" {
			t.Fatalf("candidate inesperada: %#v", result["session_candidate"])
		}
		if candidate.ObservedSessionPath != "/tmp/.claude/sessions/session-codex6.json" {
			t.Fatalf("session path inesperado: %#v", candidate)
		}
	})
}

func TestMCPRevisionSupervisorExponeWorktreeDrift(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevDrift := openClawWorktreeDriftBuilder
		defer func() {
			statusService = prev
			openClawWorktreeDriftBuilder = prevDrift
		}()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
			Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		}}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				Branch:       "orq-orquesta-codex3",
				Path:         "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
				CurrentHead:  "c51c0ac9",
				ExpectedHead: "faa85e13",
				Dirty:        true,
			}}, nil
		}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "revisar_worktree_desfasada" && item.Target == "agente:Codex3" && item.Assignee == "Codex3" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece acción para worktree drift: %+v", queue)
		}
	})
}

func TestBuildSupervisorOperationalActionsNoPromueveCheckpointPendienteComoGuidance(t *testing.T) {
	prepararDBTemporalCmd(t)
	status := apiStatusResponse{
		AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
	}
	mailboxPendiente := []apiOpenClawMailboxLite{{
		Agente:               "Codex3",
		Count:                1,
		Kinds:                []string{"autonomia"},
		KindsCSV:             "autonomia",
		SupervisorActions:    []string{"revisar_worktree_desfasada"},
		SupervisorActionsCSV: "revisar_worktree_desfasada",
		OldestAgeMin:         5,
	}}

	actions := buildSupervisorOperationalActions(status, mailboxPendiente)
	for _, item := range actions {
		if item.Action == "seguir_guidance_durable" && item.Target == "agente:Codex3" {
			t.Fatalf("no debería promocionar guidance de checkpoint como drenado: %+v", actions)
		}
	}
}

func TestBuildSupervisorOperationalActionsIncorporaAutonomyEventsRecientes(t *testing.T) {
	prepararDBTemporalCmd(t)
	taskID := int64(412)
	runtimeID := int64(77)
	handleID := int64(15)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "handoff_failed",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
					Reason:      "runtime sin consolidar",
					Artifacts:   []string{"runtime_order:123", "task:412"},
				},
				{
					Kind:      "worker_recovery_paused_external",
					CreatedAt: time.Now().UTC(),
					RuntimeID: &runtimeID,
					HandleID:  &handleID,
					Agent:     "Codex1",
					Reason:    "connector unavailable",
				},
			},
		},
		TareasActivas: []tareaLite{{ID: 413, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if !containsSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:412") {
		t.Fatalf("falta accion por handoff_failed: %+v", actions)
	}
	if !containsSupervisorAction(actions, "revisar_pausa_externa", "agente:Codex1") {
		t.Fatalf("falta accion por worker_recovery_paused_external: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsIncorporaEventosAutonomiaDeSeguimiento(t *testing.T) {
	prepararDBTemporalCmd(t)
	taskID := int64(413)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "runtime_restart_requested",
					CreatedAt:   time.Now().UTC(),
					Agent:       "Codex1",
					TargetAgent: "Codex1",
					Reason:      "tmux session missing",
				},
				{
					Kind:        "task_reassigned",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
					Reason:      "worker degradado",
				},
				{
					Kind:        "handoff_completed",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
					Reason:      "handoff consolidado",
				},
			},
		},
		TareasActivas: []tareaLite{{ID: 413, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if !containsSupervisorAction(actions, "seguir_reinicio_runtime", "agente:Codex1") {
		t.Fatalf("falta accion por runtime_restart_requested: %+v", actions)
	}
	if !containsSupervisorAction(actions, "verificar_handoff_consolidado", "tarea:413") {
		t.Fatalf("falta accion por handoff_completed: %+v", actions)
	}
	if containsSupervisorAction(actions, "seguir_reasignacion", "tarea:413") {
		t.Fatalf("no deberia duplicar reasignacion si ya existe handoff_completed del mismo frente: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsOmiteHandoffConsolidadoSiYaHayProgresoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	now := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			LastAutonomyState:  "work_confirmed",
			WorkerLastProgress: &now,
		}}, nil
	}
	taskID := int64(501)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_completed",
				CreatedAt:   now,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{ID: 501, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "verificar_handoff_consolidado", "tarea:501") {
		t.Fatalf("no deberia mantener followup de handoff consolidado si ya hay progreso reciente: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsDegradaReasignacionSiYaHayProgresoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	now := time.Now().UTC()
	progress := now.Add(-2 * time.Minute)
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(777)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   now,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{ID: 777, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "seguir_reasignacion", "tarea:777")
	if item == nil {
		t.Fatalf("falta accion de reasignacion: %+v", actions)
	}
	if item.Priority != "baja" {
		t.Fatalf("deberia degradar prioridad con progreso reciente: %+v", item)
	}
	if !strings.Contains(item.Reason, "Ya hay progreso reciente") {
		t.Fatalf("deberia explicar degradacion por progreso reciente: %+v", item)
	}
	if !strings.Contains(item.Reason, "progreso sin bloqueo visible") {
		t.Fatalf("deberia usar tambien el estado visible en progreso: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsOmiteReasignacionSiLaTareaYaNoEsVisible(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "trabajando",
		}}, nil
	}
	taskID := int64(888)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas:    []tareaLite{{ID: 999, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
		TareasReservadas: nil,
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "seguir_reasignacion", "tarea:888") {
		t.Fatalf("no deberia mantener reasignacion si la tarea ya no es visible: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsMantieneHandoffFallidoSiElFrenteSigueBloqueado(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "atascado",
		}}, nil
	}
	taskID := int64(901)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{ID: 901, Estado: db.TareaBloqueada, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	for _, item := range actions {
		if item.Action == "inspeccionar_handoff_fallido" && item.Target == "tarea:901" {
			if item.Priority != "alta" {
				t.Fatalf("deberia seguir alta si el frente sigue bloqueado: %+v", item)
			}
			if !strings.Contains(item.Reason, "bloqueado y visible") {
				t.Fatalf("deberia explicar bloqueo visible: %+v", item)
			}
			return
		}
	}
	t.Fatalf("falta handoff_failed bloqueado: %+v", actions)
}

func TestBuildSupervisorOperationalActionsElevaContextoDeHandoffFallidoBloqueadoAltaPrioridad(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "atascado",
		}}, nil
	}
	taskID := int64(905)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{
			ID:        905,
			Estado:    db.TareaBloqueada,
			Agente:    "Codex4",
			Modulo:    "checkout",
			Prioridad: db.PrioridadAlta,
		}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:905")
	if item == nil {
		t.Fatalf("falta handoff_failed alta prioridad: %+v", actions)
	}
	if item.Priority != "alta" {
		t.Fatalf("deberia seguir alta: %+v", item)
	}
	if !strings.Contains(item.Reason, "prioridad alta") || !strings.Contains(item.Reason, "Módulo: checkout") || !strings.Contains(item.Reason, "Bloquea integración real") {
		t.Fatalf("deberia explicar impacto del frente bloqueado: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneHandoffFallidoBloqueadoAunqueHayaProgresoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(9021)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{ID: 9021, Estado: db.TareaBloqueada, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:9021")
	if item == nil {
		t.Fatalf("falta handoff_failed bloqueado con progreso reciente: %+v", actions)
	}
	if item.Priority != "alta" {
		t.Fatalf("deberia mantenerse alta si el frente sigue bloqueado: %+v", item)
	}
	if !strings.Contains(item.Reason, "bloqueado y visible") {
		t.Fatalf("deberia explicar que el frente sigue bloqueado: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneHandoffFallidoAsignadoSinProgresoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "trabajando",
		}}, nil
	}
	taskID := int64(903)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasReservadas: []tareaLite{{ID: 903, Estado: db.TareaAsignada, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	for _, item := range actions {
		if item.Action == "inspeccionar_handoff_fallido" && item.Target == "tarea:903" {
			if item.Priority != "media" {
				t.Fatalf("deberia mantener prioridad media si el frente sigue asignado: %+v", item)
			}
			if !strings.Contains(item.Reason, "retenido y visible") {
				t.Fatalf("deberia explicar que el frente sigue retenido: %+v", item)
			}
			return
		}
	}
	t.Fatalf("falta handoff_failed asignado: %+v", actions)
}

func TestBuildSupervisorOperationalActionsMantieneReasignacionAsignadaSinProgreso(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "listo",
		}}, nil
	}
	taskID := int64(9031)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasReservadas: []tareaLite{{ID: 9031, Estado: db.TareaAsignada, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "seguir_reasignacion", "tarea:9031")
	if item == nil {
		t.Fatalf("no deberia desaparecer la reasignacion retenida: %+v", actions)
	}
	if item.Priority == "baja" {
		t.Fatalf("una tarea asignada sin progreso no deberia caer a prioridad trivial: %+v", item)
	}
	if !strings.Contains(item.Reason, "retenido y visible") {
		t.Fatalf("deberia explicar que el frente sigue retenido: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsDegradaReinicioRuntimeSiFrenteVisibleYaProgresa(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	now := time.Now().UTC()
	progress := now.Add(-time.Minute)
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex1"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "runtime_restart_requested",
				CreatedAt:   now,
				Agent:       "Codex1",
				TargetAgent: "Codex1",
				Reason:      "tmux session missing",
			}},
		},
		TareasActivas: []tareaLite{{ID: 9041, Estado: db.TareaEnProgreso, Agente: "Codex1"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "seguir_reinicio_runtime", "agente:Codex1")
	if item == nil {
		t.Fatalf("falta accion de reinicio runtime: %+v", actions)
	}
	if item.Priority != "baja" {
		t.Fatalf("deberia degradar fuerte si el frente ya progresa: %+v", item)
	}
	if !strings.Contains(item.Reason, "Ya hay progreso reciente") || !strings.Contains(item.Reason, "progreso sin bloqueo visible") || !strings.Contains(item.Reason, "No parece bloquear integración/progreso global") || !strings.Contains(item.Reason, "Frente local o no bloqueante") {
		t.Fatalf("deberia explicar progreso reciente y frente visible en curso: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsElevaFollowupBloqueadoQueBloqueaIntegracionReal(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "atascado",
		}}, nil
	}
	taskID := int64(9045)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "post_remediation_blocked_escalated",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Reason:      "review y merge pendientes",
				Artifacts:   []string{"patch:api-integration", "test:go test ./cmd"},
			}},
		},
		TareasActivas: []tareaLite{{
			ID:        9045,
			Estado:    db.TareaBloqueada,
			Agente:    "Codex4",
			Titulo:    "Integrar API MCP",
			Modulo:    "mcp",
			Prioridad: db.PrioridadAlta,
		}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "resolver_followup_bloqueado", "tarea:9045")
	if item == nil {
		t.Fatalf("falta followup bloqueado escalado: %+v", actions)
	}
	if item.Priority != "alta" {
		t.Fatalf("deberia quedar alto si bloquea integracion real: %+v", item)
	}
	if !strings.Contains(item.Reason, "bloqueado") || !strings.Contains(item.Reason, "Bloquea integración real") || !strings.Contains(item.Reason, "Bloquea integración/progreso global") || !strings.Contains(item.Reason, "Frente crítico del proyecto/workspace") {
		t.Fatalf("deberia explicitar bloqueo de integracion real: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsDegradaFollowupBloqueadoSiElFrenteYaProgresaSano(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	now := time.Now().UTC()
	progress := now.Add(-2 * time.Minute)
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(9046)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "post_remediation_blocked_escalated",
				CreatedAt:   now,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{
			ID:        9046,
			Estado:    db.TareaEnProgreso,
			Agente:    "Codex4",
			Titulo:    "Revisar front",
			Modulo:    "web",
			Prioridad: db.PrioridadMedia,
		}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "resolver_followup_bloqueado", "tarea:9046")
	if item == nil {
		t.Fatalf("falta followup degradado con frente sano: %+v", actions)
	}
	if item.Priority != "baja" {
		t.Fatalf("deberia bajar fuerte si el frente ya progresa sano: %+v", item)
	}
	if !strings.Contains(item.Reason, "No parece bloquear integración/progreso global") {
		t.Fatalf("deberia explicitar que ya no bloquea integracion real: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneHandoffFallidoConEvidenciaFuerteAunqueYaNoHayaFrenteVisible(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(9047)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Reason:      "runtime recovery still under review",
				Artifacts:   []string{"runtime_order:123", "checkpoint:resume-safe"},
			}},
		},
		TareasActivas: []tareaLite{{ID: 999, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:9047")
	if item == nil {
		t.Fatalf("deberia mantener handoff_failed con evidencia fuerte para cerrarlo bien: %+v", actions)
	}
	if item.Priority != "baja" {
		t.Fatalf("si ya progresa sin bloqueo global, deberia degradarse: %+v", item)
	}
	if !strings.Contains(item.Reason, "Evidencia fuerte") || !strings.Contains(item.Reason, "No parece bloquear integración/progreso global") {
		t.Fatalf("deberia explicitar evidencia fuerte mitigada y absorcion sana: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsDegradaHandoffFuerteMitigadoYSubeSiguienteRiesgoReal(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevStatus := statusService
	prevListProjects := workspaceControlListProjects
	prevCockpitBuilder := workspaceControlCockpitBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		statusService = prevStatus
		workspaceControlListProjects = prevListProjects
		workspaceControlCockpitBuilder = prevCockpitBuilder
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()

	now := time.Now().UTC()
	progress := now.Add(-time.Minute)
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{
			{
				Agente:             &db.Agente{Nombre: "Codex4"},
				EstadoOperativo:    "trabajando",
				WorkerLastProgress: &progress,
			},
			{
				Agente:          &db.Agente{Nombre: "Codex3"},
				EstadoOperativo: "atascado",
			},
		}, nil
	}
	statusService = stubStatusService{response: apiStatusResponse{}}

	projectMitigado := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
	projectPendiente := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
	taskMitigada, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Runtime recovery mitigado",
		ProyectoID:  &projectMitigado,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "worker",
	})
	if err != nil {
		t.Fatalf("crear tarea mitigada: %v", err)
	}
	taskPendiente, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Merge web bloqueado",
		ProyectoID:  &projectPendiente,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "web",
	})
	if err != nil {
		t.Fatalf("crear tarea pendiente: %v", err)
	}

	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "infra"}, {"slug": "webapp"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		switch strings.TrimSpace(slug) {
		case "infra":
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "infra"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		case "webapp":
			return &apiProyectoCockpit{
				Proyecto:            &db.Proyecto{Slug: "webapp"},
				TareasPorEstado:     map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas: 1,
			}, nil
		default:
			return nil, nil
		}
	}

	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "handoff_failed",
					CreatedAt:   now,
					TaskID:      &taskMitigada,
					TargetAgent: "Codex4",
					Reason:      "runtime recovery still under review",
					Artifacts:   []string{"runtime_order:123", "checkpoint:resume-safe"},
				},
				{
					Kind:        "handoff_failed",
					CreatedAt:   now,
					TaskID:      &taskPendiente,
					TargetAgent: "Codex3",
					Reason:      "merge bloqueado",
					Artifacts:   []string{"patch:web-merge", "test:go test ./cmd"},
				},
			},
		},
		TareasActivas: []tareaLite{
			{ID: taskMitigada, Estado: db.TareaEnProgreso, Agente: "Codex4", Titulo: "Runtime recovery mitigado", Modulo: "worker", Prioridad: db.PrioridadAlta},
			{ID: taskPendiente, Estado: db.TareaBloqueada, Agente: "Codex3", Titulo: "Merge web bloqueado", Modulo: "web", Prioridad: db.PrioridadAlta},
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if len(actions) == 0 {
		t.Fatalf("acciones vacias: %+v", actions)
	}
	if actions[0].Target != fmt.Sprintf("tarea:%d", taskPendiente) || actions[0].Action != "inspeccionar_handoff_fallido" {
		t.Fatalf("deberia subir primero el siguiente riesgo real, got=%+v", actions[0])
	}
	mitigated := findSupervisorAction(actions, "inspeccionar_handoff_fallido", fmt.Sprintf("tarea:%d", taskMitigada))
	if mitigated == nil {
		t.Fatalf("falta followup mitigado: %+v", actions)
	}
	if mitigated.Priority != "baja" {
		t.Fatalf("deberia quedar degradado tras mitigacion sana: %+v", mitigated)
	}
	if !strings.Contains(mitigated.Reason, "No parece bloquear integración/progreso global") {
		t.Fatalf("deberia explicitar mitigacion sana: %+v", mitigated)
	}
}

func TestBuildSupervisorOperationalActionsMantieneFollowupBloqueadoConEvidenciaFuerteAunqueYaProgrese(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	now := time.Now().UTC()
	progress := now.Add(-time.Minute)
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(9048)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "post_remediation_blocked_escalated",
				CreatedAt:   now,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Reason:      "tests and checkpoint still under review",
				Artifacts:   []string{"patch:api-contract", "test:go test ./cmd"},
			}},
		},
		TareasActivas: []tareaLite{{
			ID:        9048,
			Estado:    db.TareaEnProgreso,
			Agente:    "Codex4",
			Titulo:    "Integrar API",
			Modulo:    "api",
			Prioridad: db.PrioridadAlta,
		}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "resolver_followup_bloqueado", "tarea:9048")
	if item == nil {
		t.Fatalf("deberia mantener followup bloqueado con evidencia fuerte: %+v", actions)
	}
	if item.Priority != "media" {
		t.Fatalf("deberia sostener prioridad media con evidencia fuerte: %+v", item)
	}
	if !strings.Contains(item.Reason, "Evidencia fuerte") {
		t.Fatalf("deberia explicitar la evidencia fuerte: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneFollowupGlobalConEvidenciaFuerteAunqueYaProgrese(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	now := time.Now().UTC()
	progress := now.Add(-time.Minute)
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(90481)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "post_remediation_blocked_escalated",
				CreatedAt:   now,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Reason:      "review gate and release integration still open",
				Artifacts:   []string{"patch:release-api", "test:go test ./cmd"},
			}},
		},
		TareasActivas: []tareaLite{{
			ID:        90481,
			Estado:    db.TareaEnProgreso,
			Agente:    "Codex4",
			Titulo:    "Integrar release API",
			Modulo:    "api",
			Prioridad: db.PrioridadAlta,
		}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "resolver_followup_bloqueado", "tarea:90481")
	if item == nil {
		t.Fatalf("deberia mantener followup global con evidencia fuerte: %+v", actions)
	}
	if item.Priority != "media" {
		t.Fatalf("deberia sostener prioridad media cuando sigue afectando al proyecto/workspace: %+v", item)
	}
	if !strings.Contains(item.Reason, "Aun así, sigue afectando integración/progreso global") || !strings.Contains(item.Reason, "Frente crítico del proyecto/workspace") {
		t.Fatalf("deberia explicitar impacto global aun con progreso: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsOmiteHandoffFallidoSinEvidenciaFuerteSiYaHayProgresoVisible(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(9049)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Reason:      "noise only",
			}},
		},
		TareasActivas: []tareaLite{{ID: 999, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if item := findSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:9049"); item != nil {
		t.Fatalf("no deberia mantener handoff_failed sin evidencia fuerte si ya hay progreso visible: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsOmiteHandoffFallidoHistoricoSiYaNoHayFrenteYHayProgreso(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(902)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{ID: 999, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:902") {
		t.Fatalf("no deberia mantener handoff_failed historico si ya no hay frente visible y el destino progresa: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsOmiteHandoffFallidoSiElFrenteYaProgresaVisible(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(904)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasActivas: []tareaLite{{ID: 904, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:904") {
		t.Fatalf("no deberia mantener handoff_failed si el frente ya progresa visible: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsColapsaReasignacionYHandoffConsolidadoPorTarea(t *testing.T) {
	prepararDBTemporalCmd(t)
	taskID := int64(915)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "task_reassigned",
					CreatedAt:   time.Now().UTC().Add(-time.Minute),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
				},
				{
					Kind:        "handoff_completed",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
				},
			},
		},
		TareasActivas: []tareaLite{{ID: 915, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "seguir_reasignacion", "tarea:915") {
		t.Fatalf("no deberia mantener reasignacion si ya hay handoff consolidado del mismo frente: %+v", actions)
	}
	if !containsSupervisorAction(actions, "verificar_handoff_consolidado", "tarea:915") {
		t.Fatalf("deberia conservar handoff consolidado: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsDaPrioridadAHandoffFallidoSobreFollowups(t *testing.T) {
	prepararDBTemporalCmd(t)
	taskID := int64(916)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "task_reassigned",
					CreatedAt:   time.Now().UTC().Add(-2 * time.Minute),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
				},
				{
					Kind:        "handoff_completed",
					CreatedAt:   time.Now().UTC().Add(-time.Minute),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
				},
				{
					Kind:        "handoff_failed",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskID,
					TargetAgent: "Codex4",
				},
			},
		},
		TareasActivas: []tareaLite{{ID: 916, Estado: db.TareaBloqueada, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if containsSupervisorAction(actions, "seguir_reasignacion", "tarea:916") || containsSupervisorAction(actions, "verificar_handoff_consolidado", "tarea:916") {
		t.Fatalf("no deberia mantener followups si ya hay handoff_failed del mismo frente: %+v", actions)
	}
	if !containsSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:916") {
		t.Fatalf("deberia conservar handoff_failed: %+v", actions)
	}
}

func TestBuildSupervisorOperationalActionsPriorizaPrimeroElFrenteConImpactoGlobal(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "atascado",
		}}, nil
	}
	taskGlobal := int64(7001)
	taskLocal := int64(7002)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskLocal,
					TargetAgent: "Codex4",
					Reason:      "seguimiento local",
					Artifacts:   []string{"patch:local-ui"},
				},
				{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskGlobal,
					TargetAgent: "Codex4",
					Reason:      "review y merge bloquean release",
					Artifacts:   []string{"patch:release-api", "test:go test ./cmd"},
				},
			},
		},
		TareasActivas: []tareaLite{
			{ID: 7001, Estado: db.TareaBloqueada, Agente: "Codex4", Titulo: "Integrar API release", Modulo: "api", Prioridad: db.PrioridadAlta},
			{ID: 7002, Estado: db.TareaBloqueada, Agente: "Codex4", Titulo: "Ajuste local de UI", Modulo: "web", Prioridad: db.PrioridadMedia},
		},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	if len(actions) == 0 {
		t.Fatalf("acciones vacias: %+v", actions)
	}
	if actions[0].Target != "tarea:7001" || actions[0].Action != "resolver_followup_bloqueado" {
		t.Fatalf("deberia priorizar primero el frente global, got=%+v", actions[0])
	}
	if !strings.Contains(actions[0].Reason, "Bloquea integración/progreso global") {
		t.Fatalf("la accion prioritaria deberia explicitar impacto global: %+v", actions[0])
	}
}

func TestBuildSupervisorOperationalActionsPriorizaProyectoConMayorRiesgoCanonico(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevStatus := statusService
	prevListProjects := workspaceControlListProjects
	prevCockpitBuilder := workspaceControlCockpitBuilder
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		statusService = prevStatus
		workspaceControlListProjects = prevListProjects
		workspaceControlCockpitBuilder = prevCockpitBuilder
	}()

	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "atascado",
		}}, nil
	}
	statusService = stubStatusService{response: apiStatusResponse{}}

	projectLow := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
	projectHigh := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
	taskLow, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Ajuste local UI",
		ProyectoID:  &projectLow,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "web",
	})
	if err != nil {
		t.Fatalf("crear tarea low risk: %v", err)
	}
	taskHigh, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Integrar release API",
		ProyectoID:  &projectHigh,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "api",
	})
	if err != nil {
		t.Fatalf("crear tarea high risk: %v", err)
	}

	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "webapp"},
			{"slug": "infra"},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		switch strings.TrimSpace(slug) {
		case "infra":
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "infra"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		case "webapp":
			return &apiProyectoCockpit{
				Proyecto:        &db.Proyecto{Slug: "webapp"},
				TareasPorEstado: map[string]int{string(db.TareaBloqueada): 1},
			}, nil
		default:
			return nil, nil
		}
	}

	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskLow,
					TargetAgent: "Codex4",
					Reason:      "seguimiento local",
					Artifacts:   []string{"patch:web-ui", "test:go test ./cmd"},
				},
				{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   time.Now().UTC(),
					TaskID:      &taskHigh,
					TargetAgent: "Codex4",
					Reason:      "release e integración bloqueadas",
					Artifacts:   []string{"patch:release-api", "test:go test ./cmd"},
				},
			},
		},
		TareasActivas: []tareaLite{
			{ID: taskLow, Estado: db.TareaBloqueada, Agente: "Codex4", Titulo: "Ajuste local UI", Modulo: "web", Prioridad: db.PrioridadAlta},
			{ID: taskHigh, Estado: db.TareaBloqueada, Agente: "Codex4", Titulo: "Integrar release API", Modulo: "api", Prioridad: db.PrioridadAlta},
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if len(actions) == 0 {
		t.Fatalf("acciones vacias: %+v", actions)
	}
	if actions[0].Target != fmt.Sprintf("tarea:%d", taskHigh) {
		t.Fatalf("deberia priorizar el proyecto con mayor riesgo canonico, got=%+v", actions[0])
	}
}

func TestBuildSupervisorOperationalActionsDegradaProyectoAbsorbidoYSubeSiguienteRiesgoReal(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevStatus := statusService
	prevListProjects := workspaceControlListProjects
	prevCockpitBuilder := workspaceControlCockpitBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		statusService = prevStatus
		workspaceControlListProjects = prevListProjects
		workspaceControlCockpitBuilder = prevCockpitBuilder
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()

	now := time.Now().UTC()
	progress := now.Add(-2 * time.Minute)
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{
			{
				Agente:             &db.Agente{Nombre: "Codex4"},
				EstadoOperativo:    "trabajando",
				WorkerLastProgress: &progress,
			},
			{
				Agente:          &db.Agente{Nombre: "Codex3"},
				EstadoOperativo: "atascado",
			},
		}, nil
	}
	statusService = stubStatusService{response: apiStatusResponse{}}

	projectAbsorbido := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
	projectPendiente := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
	taskAbsorbida, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Ajuste local ya absorbido",
		ProyectoID:  &projectAbsorbido,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "worker",
	})
	if err != nil {
		t.Fatalf("crear tarea absorbida: %v", err)
	}
	taskPendiente, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Merge web bloqueado",
		ProyectoID:  &projectPendiente,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "web",
	})
	if err != nil {
		t.Fatalf("crear tarea pendiente: %v", err)
	}

	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "infra"},
			{"slug": "webapp"},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		switch strings.TrimSpace(slug) {
		case "infra":
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "infra"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		case "webapp":
			return &apiProyectoCockpit{
				Proyecto:            &db.Proyecto{Slug: "webapp"},
				TareasPorEstado:     map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas: 1,
			}, nil
		default:
			return nil, nil
		}
	}

	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   now,
					TaskID:      &taskAbsorbida,
					TargetAgent: "Codex4",
					Reason:      "seguimiento local estable",
					Artifacts:   []string{"patch:ajuste-local", "test:go test ./ui"},
				},
				{
					Kind:        "handoff_failed",
					CreatedAt:   now,
					TaskID:      &taskPendiente,
					TargetAgent: "Codex3",
					Reason:      "merge bloqueado",
					Artifacts:   []string{"patch:web-merge", "test:go test ./cmd"},
				},
			},
		},
		TareasActivas: []tareaLite{
			{ID: taskAbsorbida, Estado: db.TareaEnProgreso, Agente: "Codex4", Titulo: "Ajuste local ya absorbido", Modulo: "worker", Prioridad: db.PrioridadAlta},
			{ID: taskPendiente, Estado: db.TareaBloqueada, Agente: "Codex3", Titulo: "Merge web bloqueado", Modulo: "web", Prioridad: db.PrioridadAlta},
		},
	}

	actions := buildSupervisorOperationalActions(status, nil)
	if len(actions) == 0 {
		t.Fatalf("acciones vacias: %+v", actions)
	}
	if actions[0].Target != fmt.Sprintf("tarea:%d", taskPendiente) || actions[0].Action != "inspeccionar_handoff_fallido" {
		t.Fatalf("deberia subir primero el siguiente riesgo real, got=%+v", actions[0])
	}
	absorbed := findSupervisorAction(actions, "resolver_followup_bloqueado", fmt.Sprintf("tarea:%d", taskAbsorbida))
	if absorbed == nil {
		t.Fatalf("falta followup absorbido: %+v", actions)
	}
	if absorbed.Priority != "media" && absorbed.Priority != "baja" {
		t.Fatalf("deberia quedar degradado tras absorcion sana: %+v", absorbed)
	}
	if !strings.Contains(absorbed.Reason, "Ya hay progreso reciente") || !strings.Contains(absorbed.Reason, "No parece bloquear integración/progreso global") {
		t.Fatalf("deberia explicitar absorcion sana: %+v", absorbed)
	}
}

func TestBuildSupervisorCriticalAutonomyFocusOmiteFrenteAbsorbidoSano(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevStatus := statusService
	prevListProjects := workspaceControlListProjects
	prevCockpitBuilder := workspaceControlCockpitBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		statusService = prevStatus
		workspaceControlListProjects = prevListProjects
		workspaceControlCockpitBuilder = prevCockpitBuilder
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()

	now := time.Now().UTC()
	progress := now.Add(-time.Minute)
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{
			{
				Agente:             &db.Agente{Nombre: "Codex4"},
				EstadoOperativo:    "trabajando",
				WorkerLastProgress: &progress,
			},
			{
				Agente:          &db.Agente{Nombre: "Codex3"},
				EstadoOperativo: "atascado",
			},
		}, nil
	}
	statusService = stubStatusService{response: apiStatusResponse{}}

	projectAbsorbido := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
	projectPendiente := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
	taskAbsorbida, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Ajuste local ya absorbido",
		ProyectoID:  &projectAbsorbido,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "worker",
	})
	if err != nil {
		t.Fatalf("crear tarea absorbida: %v", err)
	}
	taskPendiente, err := db.CrearTarea(&db.Tarea{
		Titulo:      "Merge web bloqueado",
		ProyectoID:  &projectPendiente,
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Descripcion: "slice",
		Modulo:      "web",
	})
	if err != nil {
		t.Fatalf("crear tarea pendiente: %v", err)
	}

	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{
			{"slug": "infra"},
			{"slug": "webapp"},
		}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		switch strings.TrimSpace(slug) {
		case "infra":
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "infra"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		case "webapp":
			return &apiProyectoCockpit{
				Proyecto:            &db.Proyecto{Slug: "webapp"},
				TareasPorEstado:     map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas: 1,
			}, nil
		default:
			return nil, nil
		}
	}

	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{
				{
					Kind:        "post_remediation_blocked_escalated",
					CreatedAt:   now,
					TaskID:      &taskAbsorbida,
					TargetAgent: "Codex4",
					Reason:      "seguimiento local estable",
					Artifacts:   []string{"patch:ajuste-local", "test:go test ./ui"},
				},
				{
					Kind:        "handoff_failed",
					CreatedAt:   now,
					TaskID:      &taskPendiente,
					TargetAgent: "Codex3",
					Reason:      "merge bloqueado",
					Artifacts:   []string{"patch:web-merge", "test:go test ./cmd"},
				},
			},
		},
		TareasActivas: []tareaLite{
			{ID: taskAbsorbida, Estado: db.TareaEnProgreso, Agente: "Codex4", Titulo: "Ajuste local ya absorbido", Modulo: "worker", Prioridad: db.PrioridadAlta},
			{ID: taskPendiente, Estado: db.TareaBloqueada, Agente: "Codex3", Titulo: "Merge web bloqueado", Modulo: "web", Prioridad: db.PrioridadAlta},
		},
	}

	focus := buildSupervisorCriticalAutonomyFocus(status, nil, nil)
	if focus == nil {
		t.Fatalf("falta frente critico")
	}
	if focus.Target != fmt.Sprintf("tarea:%d", taskPendiente) || focus.Action != "inspeccionar_handoff_fallido" {
		t.Fatalf("deberia saltar el frente absorbido sano y subir el riesgo real: %+v", focus)
	}
}

func TestBuildSupervisorReviewSnapshotDegradaProyectoCriticoAbsorbidoYExponeSiguienteRiesgoReal(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevRows := supervisorPanelRowsBuilder
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevNow := statusNowFunc
		prevThreshold := supervisorSemanticProgressRecentThreshold
		defer func() {
			statusService = prev
			supervisorPanelRowsBuilder = prevRows
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			statusNowFunc = prevNow
			supervisorSemanticProgressRecentThreshold = prevThreshold
		}()

		now := time.Now().UTC()
		progress := now.Add(-2 * time.Minute)
		statusNowFunc = func() time.Time { return now }
		supervisorSemanticProgressRecentThreshold = 10 * time.Minute
		supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
			return []agentesapp.Row{
				{
					Agente:             &db.Agente{Nombre: "Codex4"},
					EstadoOperativo:    "trabajando",
					WorkerLastProgress: &progress,
				},
				{
					Agente:          &db.Agente{Nombre: "Codex3"},
					EstadoOperativo: "atascado",
				},
			}, nil
		}

		projectAbsorbido := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
		projectPendiente := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
		taskAbsorbida, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Ajuste local ya absorbido",
			ProyectoID:  &projectAbsorbido,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "slice",
			Modulo:      "worker",
		})
		if err != nil {
			t.Fatalf("crear tarea absorbida: %v", err)
		}
		taskPendiente, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Merge web bloqueado",
			ProyectoID:  &projectPendiente,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "slice",
			Modulo:      "web",
		})
		if err != nil {
			t.Fatalf("crear tarea pendiente: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex4' WHERE id=?`, taskAbsorbida); err != nil {
			t.Fatalf("preparar tarea absorbida: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='bloqueada', agente='Codex3' WHERE id=?`, taskPendiente); err != nil {
			t.Fatalf("preparar tarea pendiente: %v", err)
		}

		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
				{Nombre: "Codex4", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			WorkersConectados:   2,
			WorkersTrabajando:   1,
			SupervisoresActivos: 1,
			Autonomia: autonomiaResumen{
				Recent: []autonomyEventSummary{
					{
						Kind:        "post_remediation_blocked_escalated",
						CreatedAt:   now,
						TaskID:      &taskAbsorbida,
						TargetAgent: "Codex4",
						Reason:      "seguimiento local estable",
						Artifacts:   []string{"patch:ajuste-local", "test:go test ./ui"},
					},
					{
						Kind:        "handoff_failed",
						CreatedAt:   now,
						TaskID:      &taskPendiente,
						TargetAgent: "Codex3",
						Reason:      "merge bloqueado",
						Artifacts:   []string{"patch:web-merge", "test:go test ./cmd"},
					},
				},
			},
			TareasActivas: []tareaLite{
				{ID: taskAbsorbida, Estado: db.TareaEnProgreso, Agente: "Codex4", Titulo: "Ajuste local ya absorbido", Modulo: "worker", Prioridad: db.PrioridadAlta},
				{ID: taskPendiente, Estado: db.TareaBloqueada, Agente: "Codex3", Titulo: "Merge web bloqueado", Modulo: "web", Prioridad: db.PrioridadAlta},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)

		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{
				{"slug": "infra"},
				{"slug": "webapp"},
			}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			switch strings.TrimSpace(slug) {
			case "infra":
				return &apiProyectoCockpit{
					Proyecto:                &db.Proyecto{Slug: "infra"},
					TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas:     1,
					RuntimeOrdersAbiertas:   1,
					RuntimeMailboxPendiente: 1,
					PropuestasAbiertas:      1,
				}, nil
			case "webapp":
				return &apiProyectoCockpit{
					Proyecto:            &db.Proyecto{Slug: "webapp"},
					TareasPorEstado:     map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas: 1,
				}, nil
			default:
				return nil, nil
			}
		}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		risk, _ := snapshot["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || risk.Project != "webapp" || risk.Blocking != 9 {
			t.Fatalf("critical_project_risk deberia subir el siguiente riesgo real, got=%#v", snapshot["critical_project_risk"])
		}
		nextAction, _ := snapshot["next_action"].(supervisorRecommendedAction)
		if nextAction.Target != fmt.Sprintf("tarea:%d", taskPendiente) || nextAction.Action != "inspeccionar_handoff_fallido" {
			t.Fatalf("next_action deberia abrir con el siguiente riesgo real, got=%+v", nextAction)
		}
	})
}

func TestBuildSupervisorReviewSnapshotExponeYRrespetaProyectoCriticoPorRiesgo(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevRows := supervisorPanelRowsBuilder
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			supervisorPanelRowsBuilder = prevRows
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			statusNowFunc = prevNow
		}()

		supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
			return []agentesapp.Row{{
				Agente:          &db.Agente{Nombre: "Codex4"},
				EstadoOperativo: "atascado",
			}}, nil
		}

		projectLow := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
		projectHigh := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
		taskLow, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Ajuste local UI",
			ProyectoID:  &projectLow,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "slice",
			Modulo:      "web",
		})
		if err != nil {
			t.Fatalf("crear tarea low risk: %v", err)
		}
		taskHigh, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Integrar release API",
			ProyectoID:  &projectHigh,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "slice",
			Modulo:      "api",
		})
		if err != nil {
			t.Fatalf("crear tarea high risk: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			Autonomia: autonomiaResumen{
				Recent: []autonomyEventSummary{
					{
						Kind:        "post_remediation_blocked_escalated",
						CreatedAt:   time.Now().UTC(),
						TaskID:      &taskLow,
						TargetAgent: "Codex4",
						Reason:      "seguimiento local",
						Artifacts:   []string{"patch:web-ui", "test:go test ./cmd"},
					},
					{
						Kind:        "post_remediation_blocked_escalated",
						CreatedAt:   time.Now().UTC(),
						TaskID:      &taskHigh,
						TargetAgent: "Codex4",
						Reason:      "release e integración bloqueadas",
						Artifacts:   []string{"patch:release-api", "test:go test ./cmd"},
					},
				},
			},
			TareasActivas: []tareaLite{
				{ID: taskLow, Estado: db.TareaBloqueada, Agente: "Codex4", Titulo: "Ajuste local UI", Modulo: "web", Prioridad: db.PrioridadAlta},
				{ID: taskHigh, Estado: db.TareaBloqueada, Agente: "Codex4", Titulo: "Integrar release API", Modulo: "api", Prioridad: db.PrioridadAlta},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)
		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{
				{"slug": "webapp"},
				{"slug": "infra"},
			}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			switch strings.TrimSpace(slug) {
			case "infra":
				return &apiProyectoCockpit{
					Proyecto:                &db.Proyecto{Slug: "infra"},
					TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas:     1,
					RuntimeOrdersAbiertas:   1,
					RuntimeMailboxPendiente: 1,
					PropuestasAbiertas:      1,
				}, nil
			case "webapp":
				return &apiProyectoCockpit{
					Proyecto:        &db.Proyecto{Slug: "webapp"},
					TareasPorEstado: map[string]int{string(db.TareaBloqueada): 1},
				}, nil
			default:
				return nil, nil
			}
		}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		nextAction, _ := snapshot["next_action"].(supervisorRecommendedAction)
		if nextAction.Target != fmt.Sprintf("tarea:%d", taskHigh) {
			t.Fatalf("next_action deberia priorizar proyecto critico, got=%+v", nextAction)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		if len(queue) == 0 || queue[0].Target != fmt.Sprintf("tarea:%d", taskHigh) {
			t.Fatalf("action_queue deberia abrir con el proyecto critico, got=%+v", queue)
		}
		risk, _ := snapshot["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || risk.Project != "infra" || risk.Blocking != 15 {
			t.Fatalf("critical_project_risk inesperado: %#v", snapshot["critical_project_risk"])
		}
	})
}

func TestBuildSupervisorReviewSnapshotSafeActionQueueRespetaProyectoCriticoPorRiesgo(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevDrift := openClawWorktreeDriftBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			openClawWorktreeDriftBuilder = prevDrift
			statusNowFunc = prevNow
		}()

		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		projectLow := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
		projectHigh := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
		taskLow, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Frente local retenido",
			ProyectoID:  &projectLow,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "quota",
			Modulo:      "web",
		})
		if err != nil {
			t.Fatalf("crear tarea low risk retenida: %v", err)
		}
		taskHigh, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Release API retenida",
			ProyectoID:  &projectHigh,
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "orquesta",
			Descripcion: "quota",
			Modulo:      "api",
		})
		if err != nil {
			t.Fatalf("crear tarea high risk retenida: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex5", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: taskLow, Estado: db.TareaEnProgreso, Titulo: "Frente local retenido", Agente: "Codex2", Modulo: "web", Prioridad: db.PrioridadAlta},
				{ID: taskHigh, Estado: db.TareaEnProgreso, Titulo: "Release API retenida", Agente: "Codex5", Modulo: "api", Prioridad: db.PrioridadAlta},
				{ID: 9991, Estado: db.TareaEnProgreso, Titulo: "Trabajo activo", Agente: "Codex3", Modulo: "runtime", Prioridad: db.PrioridadMedia},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)

		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{
				{"slug": "webapp"},
				{"slug": "infra"},
			}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			switch strings.TrimSpace(slug) {
			case "infra":
				return &apiProyectoCockpit{
					Proyecto:                &db.Proyecto{Slug: "infra"},
					TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas:     1,
					RuntimeOrdersAbiertas:   1,
					RuntimeMailboxPendiente: 1,
				}, nil
			case "webapp":
				return &apiProyectoCockpit{
					Proyecto:        &db.Proyecto{Slug: "webapp"},
					TareasPorEstado: map[string]int{string(db.TareaBloqueada): 1},
				}, nil
			default:
				return nil, nil
			}
		}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		safeQueue, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
		if len(safeQueue) == 0 {
			t.Fatalf("safe_action_queue vacia: %+v", snapshot)
		}
		if safeQueue[0].Action != "replanificar_por_cuota" || safeQueue[0].Target != fmt.Sprintf("tarea:%d", taskHigh) {
			t.Fatalf("safe_action_queue deberia abrir con el proyecto critico, got=%+v", safeQueue[0])
		}
		nextSafeAction, _ := snapshot["next_safe_action"].(supervisorRecommendedAction)
		if nextSafeAction.Action != "replanificar_por_cuota" || nextSafeAction.Target != fmt.Sprintf("tarea:%d", taskHigh) {
			t.Fatalf("next_safe_action deberia respetar el arbitraje por proyecto, got=%+v", nextSafeAction)
		}
	})
}

func TestBuildSupervisorReviewSnapshotWithStatusToleraSeccionesLentasOMuertas(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevThreads := supervisorReviewThreadsSnapshotBuilder
		prevSubagents := supervisorReviewSubagentsSnapshotBuilder
		prevPipeline := supervisorReviewPipelineSnapshotBuilder
		prevSignals := supervisorReviewSignalsBuilder
		prevMerges := supervisorReviewMergesBuilder
		prevCriticalRisk := supervisorReviewCriticalRiskBuilder
		prevDrift := openClawWorktreeDriftBuilder
		defer func() {
			supervisorReviewThreadsSnapshotBuilder = prevThreads
			supervisorReviewSubagentsSnapshotBuilder = prevSubagents
			supervisorReviewPipelineSnapshotBuilder = prevPipeline
			supervisorReviewSignalsBuilder = prevSignals
			supervisorReviewMergesBuilder = prevMerges
			supervisorReviewCriticalRiskBuilder = prevCriticalRisk
			openClawWorktreeDriftBuilder = prevDrift
		}()

		supervisorReviewThreadsSnapshotBuilder = func(supervisor, sessionID string, limit int) (map[string]any, error) {
			return nil, fmt.Errorf("threads degradadas")
		}
		supervisorReviewSubagentsSnapshotBuilder = func(supervisor, proyectoSlug, sessionID string, limit int) (map[string]any, error) {
			return nil, fmt.Errorf("subagents degradados")
		}
		supervisorReviewPipelineSnapshotBuilder = func(supervisor, proyectoSlug string, limit int, status apiStatusResponse, gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (map[string]any, error) {
			return nil, fmt.Errorf("pipeline degradado")
		}
		supervisorReviewSignalsBuilder = func(limit int) ([]*supervisorReviewSignal, error) {
			return nil, fmt.Errorf("signals degradadas")
		}
		supervisorReviewMergesBuilder = func(limit int) ([]*db.GitMerge, error) {
			return nil, fmt.Errorf("merges degradados")
		}
		supervisorReviewCriticalRiskBuilder = func(actions []supervisorRecommendedAction) (*workspaceAutonomyProjectSummary, error) {
			return nil, fmt.Errorf("critical risk degradado")
		}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		status := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
		}

		snapshot, err := buildSupervisorReviewSnapshotWithStatus("OpenClaw", status)
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshotWithStatus no deberia fallar por secciones degradadas: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		if len(queue) == 0 {
			t.Fatalf("action_queue no deberia quedar vacia: %+v", snapshot)
		}
		if queue[0].Kind != "dispatch" {
			t.Fatalf("action_queue deberia conservar accion operativa minima, got=%+v", queue[0])
		}
		safeQueue, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
		if len(safeQueue) == 0 {
			t.Fatalf("safe_action_queue no deberia quedar vacia: %+v", snapshot)
		}
		nextAction, _ := snapshot["next_action"].(supervisorRecommendedAction)
		if nextAction.Kind != "dispatch" {
			t.Fatalf("next_action inesperada: %+v", nextAction)
		}
		if signals, _ := snapshot["signals"].([]*supervisorReviewSignal); len(signals) != 0 {
			t.Fatalf("signals deberia degradarse a vacio: %+v", signals)
		}
		if merges, _ := snapshot["merges"].([]*db.GitMerge); len(merges) != 0 {
			t.Fatalf("merges deberia degradarse a vacio: %+v", merges)
		}
		if risk, _ := snapshot["critical_project_risk"].(*workspaceAutonomyProjectSummary); risk != nil {
			t.Fatalf("critical_project_risk deberia degradarse a nil: %#v", risk)
		}
	})
}

func TestBuildSupervisorOperationalActionsMantieneReinicioRuntimeAsignadoSinProgresoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "trabajando",
		}}, nil
	}
	taskID := int64(918)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "runtime_restart_requested",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasReservadas: []tareaLite{{ID: 918, Estado: db.TareaAsignada, Agente: "Codex4", Prioridad: db.PrioridadMedia}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "seguir_reinicio_runtime", "tarea:918")
	if item == nil {
		t.Fatalf("falta followup de reinicio runtime: %+v", actions)
	}
	if item.Priority != "media" {
		t.Fatalf("deberia mantener prioridad media si el frente sigue asignado: %+v", item)
	}
	if !strings.Contains(item.Reason, "retenido y visible") {
		t.Fatalf("deberia explicar frente retenido: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneHandoffFallidoConEvidenciaFuerteAunqueElFrenteYaProgresa(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(919)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "handoff_failed",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Artifacts:   []string{"runtime_order:41", "patch:handoff-fix.diff"},
			}},
		},
		TareasActivas: []tareaLite{{ID: 919, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "inspeccionar_handoff_fallido", "tarea:919")
	if item == nil {
		t.Fatalf("deberia mantener handoff_failed con evidencia fuerte para cerrarlo bien: %+v", actions)
	}
	if item.Priority != "baja" {
		t.Fatalf("si ya progresa sin bloqueo global, deberia degradarse: %+v", item)
	}
	if !strings.Contains(strings.ToLower(item.Reason), "evidencia fuerte") || !strings.Contains(item.Reason, "No parece bloquear integración/progreso global") {
		t.Fatalf("deberia explicitar evidencia fuerte mitigada y absorcion sana: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneFollowupBloqueadoConEvidenciaFuerteAunqueElFrenteYaProgresa(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	prevThreshold := supervisorSemanticProgressRecentThreshold
	defer func() {
		supervisorPanelRowsBuilder = prevRows
		supervisorSemanticProgressRecentThreshold = prevThreshold
	}()
	supervisorSemanticProgressRecentThreshold = 10 * time.Minute
	progress := time.Now().UTC()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:             &db.Agente{Nombre: "Codex4"},
			EstadoOperativo:    "trabajando",
			WorkerLastProgress: &progress,
		}}, nil
	}
	taskID := int64(920)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "post_remediation_blocked_escalated",
				CreatedAt:   progress,
				TaskID:      &taskID,
				TargetAgent: "Codex4",
				Artifacts:   []string{"runtime_order:99", "test:integration-red"},
			}},
		},
		TareasActivas: []tareaLite{{ID: 920, Estado: db.TareaEnProgreso, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	item := findSupervisorAction(actions, "resolver_followup_bloqueado", "tarea:920")
	if item == nil {
		t.Fatalf("deberia mantener followup bloqueado con evidencia fuerte: %+v", actions)
	}
	if item.Priority != "media" {
		t.Fatalf("deberia mantenerse en media con evidencia fuerte aunque ya progrese: %+v", item)
	}
	if !strings.Contains(item.Reason, "Evidencia:") {
		t.Fatalf("deberia conservar evidencia en el motivo: %+v", item)
	}
}

func TestBuildSupervisorOperationalActionsMantieneReasignacionAsignadaSinProgresoReciente(t *testing.T) {
	prepararDBTemporalCmd(t)
	prevRows := supervisorPanelRowsBuilder
	defer func() { supervisorPanelRowsBuilder = prevRows }()
	supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
		return []agentesapp.Row{{
			Agente:          &db.Agente{Nombre: "Codex4"},
			EstadoOperativo: "trabajando",
		}}, nil
	}
	taskID := int64(917)
	status := apiStatusResponse{
		Autonomia: autonomiaResumen{
			Recent: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   time.Now().UTC(),
				TaskID:      &taskID,
				TargetAgent: "Codex4",
			}},
		},
		TareasReservadas: []tareaLite{{ID: 917, Estado: db.TareaAsignada, Agente: "Codex4"}},
	}
	actions := buildSupervisorOperationalActions(status, nil)
	for _, item := range actions {
		if item.Action == "seguir_reasignacion" && item.Target == "tarea:917" {
			if item.Priority != "media" {
				t.Fatalf("deberia mantener prioridad media si el frente sigue asignado sin progreso: %+v", item)
			}
			if !strings.Contains(item.Reason, "retenido y visible") {
				t.Fatalf("deberia explicar que el frente sigue retenido: %+v", item)
			}
			return
		}
	}
	t.Fatalf("falta followup de reasignacion asignada: %+v", actions)
}

func TestMCPToolSupervisorInspeccionaWorktreeDrift(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevDrift := openClawWorktreeDriftBuilder
		defer func() {
			statusService = prev
			openClawWorktreeDriftBuilder = prevDrift
		}()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
			Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		}}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				Branch:       "orq-orquesta-codex3",
				Path:         "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
				CurrentHead:  "c51c0ac9",
				ExpectedHead: "faa85e13",
			}}, nil
		}

		result, err := applySupervisorRecommendedAction("OpenClaw", "revisar_worktree_desfasada", "agente:Codex3", "Codex3")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		drift, _ := result["worktree_drift"].(apiOpenClawWorktreeDrift)
		if drift.Agente != "Codex3" || drift.CurrentHead != "c51c0ac9" || drift.ExpectedHead != "faa85e13" {
			t.Fatalf("worktree drift inesperado: %#v", result["worktree_drift"])
		}
	})
}

func TestBuildSupervisorReviewSnapshotToleraWorktreeDriftLento(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevDrift := openClawWorktreeDriftBuilder
		prevTimeout := openClawWorktreeDriftTimeout
		defer func() {
			openClawWorktreeDriftBuilder = prevDrift
			openClawWorktreeDriftTimeout = prevTimeout
		}()

		openClawWorktreeDriftTimeout = 20 * time.Millisecond
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			time.Sleep(200 * time.Millisecond)
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				CurrentHead:  "slowhead1",
				ExpectedHead: "slowhead2",
			}}, nil
		}

		start := time.Now()
		snapshot, err := buildSupervisorReviewSnapshotWithStatus("OpenClaw", apiStatusResponse{
			Agentes:        []*db.Agente{{Nombre: "Codex3", Activo: true, EstadoCuota: "activo"}},
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Activo: true, EstadoCuota: "activo"}},
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshotWithStatus: %v", err)
		}
		if elapsed >= 150*time.Millisecond {
			t.Fatalf("snapshot deberia tolerar drift lento sin bloquear, elapsed=%s", elapsed)
		}
		drift, _ := snapshot["worktree_drift"].([]apiOpenClawWorktreeDrift)
		if len(drift) != 0 {
			t.Fatalf("drift deberia omitirse si excede timeout, got=%+v", drift)
		}
	})
}

func TestBuildSupervisorReviewSnapshotWithStatusUsaCarrilesRapidosParaSafeQueueYOperacional(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prevRisk := supervisorWorkspaceProjectRiskSnapshotBuilder
		prevRows := supervisorPanelRowsBuilder
		prevDrift := openClawWorktreeDriftBuilder
		prevThreads := supervisorReviewThreadsSnapshotBuilder
		prevSubagents := supervisorReviewSubagentsSnapshotBuilder
		prevPipeline := supervisorReviewPipelineSnapshotBuilder
		prevSignals := supervisorReviewSignalsBuilder
		prevMerges := supervisorReviewMergesBuilder
		prevCriticalRisk := supervisorReviewCriticalRiskBuilder
		defer func() {
			supervisorWorkspaceProjectRiskSnapshotBuilder = prevRisk
			supervisorPanelRowsBuilder = prevRows
			openClawWorktreeDriftBuilder = prevDrift
			supervisorReviewThreadsSnapshotBuilder = prevThreads
			supervisorReviewSubagentsSnapshotBuilder = prevSubagents
			supervisorReviewPipelineSnapshotBuilder = prevPipeline
			supervisorReviewSignalsBuilder = prevSignals
			supervisorReviewMergesBuilder = prevMerges
			supervisorReviewCriticalRiskBuilder = prevCriticalRisk
		}()

		supervisorWorkspaceProjectRiskSnapshotBuilder = func() map[string]int {
			time.Sleep(250 * time.Millisecond)
			return map[string]int{"orquestador": 9}
		}
		supervisorPanelRowsBuilder = func() ([]agentesapp.Row, error) {
			time.Sleep(250 * time.Millisecond)
			return []agentesapp.Row{{
				Agente: &db.Agente{Nombre: "Codex10", Rol: "programador"},
			}}, nil
		}
		supervisorReviewThreadsSnapshotBuilder = func(supervisor, sessionID string, limit int) (map[string]any, error) {
			return map[string]any{}, nil
		}
		supervisorReviewSubagentsSnapshotBuilder = func(supervisor, proyectoSlug, sessionID string, limit int) (map[string]any, error) {
			return map[string]any{}, nil
		}
		supervisorReviewPipelineSnapshotBuilder = func(supervisor, proyectoSlug string, limit int, status apiStatusResponse, gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (map[string]any, error) {
			return map[string]any{}, nil
		}
		supervisorReviewSignalsBuilder = func(limit int) ([]*supervisorReviewSignal, error) {
			return nil, nil
		}
		supervisorReviewMergesBuilder = func(limit int) ([]*db.GitMerge, error) {
			return nil, nil
		}
		supervisorReviewCriticalRiskBuilder = func(actions []supervisorRecommendedAction) (*workspaceAutonomyProjectSummary, error) {
			return nil, nil
		}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		start := time.Now()
		snapshot, err := buildSupervisorReviewSnapshotWithStatus("OpenClaw", apiStatusResponse{
			Agentes:           []*db.Agente{{Nombre: "Codex10", Activo: true, EstadoCuota: "activo"}},
			AgentesActivos:    []*db.Agente{{Nombre: "Codex10", Activo: true, EstadoCuota: "activo"}},
			AgentesTrabajando: []*db.Agente{{Nombre: "Codex10", Activo: true, EstadoCuota: "activo"}},
			TareasActivas: []tareaLite{{
				ID: 40, Estado: db.TareaEnProgreso, Agente: "Codex10", Modulo: "controlplane", Titulo: "Micro-refactorización",
			}},
			TareasPorEstado: map[string]int{
				string(db.TareaEnProgreso): 1,
				string(db.TareaLibre):      1,
			},
		})
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshotWithStatus: %v", err)
		}
		if elapsed >= 600*time.Millisecond {
			t.Fatalf("snapshot no deberia bloquearse por risk/panel lentos, elapsed=%s", elapsed)
		}
		safeQueue, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
		if len(safeQueue) == 0 {
			t.Fatalf("safe_action_queue vacia: %+v", snapshot)
		}
	})
}

func containsSupervisorAction(items []supervisorRecommendedAction, action, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item.Action) == strings.TrimSpace(action) && strings.TrimSpace(item.Target) == strings.TrimSpace(target) {
			return true
		}
	}
	return false
}

func findSupervisorAction(items []supervisorRecommendedAction, action, target string) *supervisorRecommendedAction {
	for i := range items {
		if strings.TrimSpace(items[i].Action) == strings.TrimSpace(action) && strings.TrimSpace(items[i].Target) == strings.TrimSpace(target) {
			return &items[i]
		}
	}
	return nil
}

func TestMCPToolSupervisorSolicitaCheckpointPorWorktreeDrift(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevDrift := openClawWorktreeDriftBuilder
		defer func() {
			statusService = prev
			openClawWorktreeDriftBuilder = prevDrift
		}()

		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestAsignacion(t, "Codex3", proyectoID, "frente activo")

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
			Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		}}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				Branch:       "orq-orquesta-codex3",
				Path:         "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
				CurrentHead:  "c51c0ac9",
				ExpectedHead: "ec39312c",
				Dirty:        true,
				DirtySummary: "12 tracked · 6 untracked",
			}}, nil
		}

		result, err := applySupervisorRecommendedAction("OpenClaw", "revisar_worktree_desfasada", "agente:Codex3", "Codex3")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		if result["nudge_enqueued"] != true {
			t.Fatalf("deberia haber encolado guidance durable: %#v", result)
		}
		if result["proyecto"] != "orquestador" {
			t.Fatalf("proyecto inesperado: %#v", result["proyecto"])
		}
		instruction, _ := result["checkpoint_request"].(string)
		for _, token := range []string{"checkpoint", "c51c0ac9", "ec39312c", "12 tracked · 6 untracked"} {
			if !strings.Contains(instruction, token) {
				t.Fatalf("instruction sin %q: %s", token, instruction)
			}
		}
		orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: strPtr("Codex3"), Limit: 10})
		if err != nil {
			t.Fatalf("listar runtime orders: %v", err)
		}
		found := false
		for _, order := range orders {
			if order == nil || order.Tipo != "nudge" {
				continue
			}
			if strings.Contains(order.PayloadJSON, `"supervisor_action":"revisar_worktree_desfasada"`) &&
				strings.Contains(order.PayloadJSON, `"dirty_summary":"12 tracked · 6 untracked"`) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparecio nudge de checkpoint worktree en runtime orders: %+v", orders)
		}
	})
}

func TestMCPRevisionSupervisorNoRepiteWorktreeDriftSiYaHayGuidancePendiente(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevDrift := openClawWorktreeDriftBuilder
		defer func() {
			statusService = prev
			openClawWorktreeDriftBuilder = prevDrift
		}()

		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestAsignacion(t, "Codex3", proyectoID, "frente activo")

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
			Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		}}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				Branch:       "orq-orquesta-codex3",
				Path:         "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
				CurrentHead:  "c51c0ac9",
				ExpectedHead: "ec39312c",
				Dirty:        true,
				DirtySummary: "12 tracked · 6 untracked",
			}}, nil
		}

		if _, err := applySupervisorRecommendedAction("OpenClaw", "revisar_worktree_desfasada", "agente:Codex3", "Codex3"); err != nil {
			t.Fatalf("aplicar revisar_worktree_desfasada: %v", err)
		}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
		for _, item := range queue {
			if item.Action == "revisar_worktree_desfasada" && item.Target == "agente:Codex3" {
				t.Fatalf("no deberia repetir worktree_drift mientras haya guidance pendiente: %+v", queue)
			}
		}
	})
}

func TestMCPRevisionSupervisorExponeRefrescoSeguroDeWorktreeLimpia(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevDrift := openClawWorktreeDriftBuilder
		defer func() {
			statusService = prev
			openClawWorktreeDriftBuilder = prevDrift
		}()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
			Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		}}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				Branch:       "orq-orquesta-codex3",
				Path:         "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
				CurrentHead:  "c51c0ac9",
				ExpectedHead: "faa85e13",
				Dirty:        false,
			}}, nil
		}

		snapshot, err := buildSupervisorReviewSnapshot("OpenClaw")
		if err != nil {
			t.Fatalf("buildSupervisorReviewSnapshot: %v", err)
		}
		queue, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
		found := false
		for _, item := range queue {
			if item.Action == "refrescar_worktree_limpia" && item.Target == "agente:Codex3" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no aparece acción segura para worktree limpia: %+v", queue)
		}
	})
}

func TestMCPToolSupervisorRefrescaWorktreeLimpia(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevDrift := openClawWorktreeDriftBuilder
		prevRefresh := supervisorRefreshCleanWorktreeDrift
		defer func() {
			statusService = prev
			openClawWorktreeDriftBuilder = prevDrift
			supervisorRefreshCleanWorktreeDrift = prevRefresh
		}()

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
			Agentes:        []*db.Agente{{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"}},
		}}
		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return []apiOpenClawWorktreeDrift{{
				Agente:       "Codex3",
				Branch:       "orq-orquesta-codex3",
				Path:         "/tmp/orquesta/.orquesta-worktrees/orquesta-codex3",
				CurrentHead:  "c51c0ac9",
				ExpectedHead: "faa85e13",
				Dirty:        false,
			}}, nil
		}
		supervisorRefreshCleanWorktreeDrift = func(item apiOpenClawWorktreeDrift) (map[string]any, error) {
			return map[string]any{"reopened_worktree_id": int64(99), "path": item.Path}, nil
		}

		result, err := applySupervisorRecommendedAction("OpenClaw", "refrescar_worktree_limpia", "agente:Codex3", "Codex3")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		refresh, _ := result["worktree_refresh"].(map[string]any)
		if refresh["reopened_worktree_id"] != int64(99) {
			t.Fatalf("refresh inesperado: %#v", result["worktree_refresh"])
		}
	})
}

func TestMCPToolSupervisorDrainGuidanceDurableWithoutBlindConsume(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevMailboxExecutor := apiRuntimeProcessMailboxExecutor
		defer func() { statusService = prev }()
		defer func() { apiRuntimeProcessMailboxExecutor = prevMailboxExecutor }()

		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		msgID, err := db.EnviarRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    "Codex3",
			Kind:        "autonomia",
			PayloadJSON: `{"texto":"continua"}`,
			Estado:      "pendiente",
		})
		if err != nil {
			t.Fatalf("crear mailbox pendiente: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
		}}
		var seenFilter db.FiltroRuntimeMailbox
		apiRuntimeProcessMailboxExecutor = func(filter db.FiltroRuntimeMailbox) (int, error) {
			seenFilter = filter
			return 3, nil
		}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar", map[string]any{
			"supervisor": "OpenClaw",
			"action":     "seguir_guidance_durable",
			"target":     "agente:Codex3",
			"assignee":   "Codex3",
		})
		if err != nil {
			t.Fatalf("aplicar guidance durable: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("guidance durable marcada como error: %#v", result)
		}
		if seenFilter.ToAgente == nil || *seenFilter.ToAgente != "Codex3" {
			t.Fatalf("filtro mailbox inesperado: %+v", seenFilter)
		}
		payload, _ := result["structuredContent"].(map[string]any)
		processedCount, _ := payload["processed_count"].(int)
		if got := processedCount; got != 3 {
			t.Fatalf("processed_count inesperado: %#v", payload)
		}
		pendingAfter, _ := payload["mailbox_pending_after"].(int)
		if got := pendingAfter; got != 1 {
			t.Fatalf("mailbox_pending_after inesperado: %#v", payload)
		}
		if resolved, _ := payload["guidance_resolved"].(bool); resolved {
			t.Fatalf("guidance no deberia figurar resuelta sin entrega real: %#v", payload)
		}
		rows, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: strPtr("Codex3")})
		if err != nil {
			t.Fatalf("listar mailbox: %v", err)
		}
		var found *db.RuntimeMailboxMessage
		for _, row := range rows {
			if row != nil && row.ID == msgID {
				found = row
				break
			}
		}
		if found == nil || found.Estado != "pendiente" {
			t.Fatalf("mailbox no deberia consumirse a ciegas: %+v", found)
		}
	})
}

func TestMCPToolSupervisorAplicaReplanificacionPorCuotaConFallbackExplicito(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Replanificar por cuota",
			Descripcion: "Debe poder reasignarse aunque la cola viva no la recomiende ahora",
			Modulo:      "runtime",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaID); err != nil {
			t.Fatalf("dejar tarea en progreso con Codex2: %v", err)
		}

		statusService = stubStatusService{response: apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
		}}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar", map[string]any{
			"supervisor": "OpenClaw",
			"action":     "replanificar_por_cuota",
			"target":     "tarea:" + itoa(tareaID),
			"assignee":   "Codex3",
		})
		if err != nil {
			t.Fatalf("aplicar replanificacion supervisor: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("replanificacion marcada como error: %#v", result)
		}

		tarea, err := db.GetTarea(tareaID)
		if err != nil {
			t.Fatalf("get tarea: %v", err)
		}
		if tarea.Agente == nil || *tarea.Agente != "Codex3" {
			t.Fatalf("agente tarea inesperado tras replanificacion: %+v", tarea)
		}
		if tarea.Estado != db.TareaEnProgreso {
			t.Fatalf("estado tarea inesperado tras replanificacion: %+v", tarea)
		}
	})
}

func TestMCPToolSupervisorAplicaLoteSeguro(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			statusNowFunc = prevNow
		}()

		tareaLibreID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Dispatch en lote",
			Descripcion: "Debe entrar por batch MCP",
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		tareaRetenidaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Retenida por cuota",
			Descripcion: "Debe replanificarse por batch MCP",
			Modulo:      "controlplane",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea retenida: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaRetenidaID); err != nil {
			t.Fatalf("preparar retenida: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
			TareasActivas: []tareaLite{
				{ID: tareaRetenidaID, Estado: db.TareaEnProgreso, Titulo: "Retenida por cuota", Agente: "Codex2"},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)
		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{{"slug": "orquestador"}}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "orquestador"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		}

		result, err := callMCPTool("orquesta.supervision.acciones.aplicar_lote", map[string]any{
			"supervisor": "OpenClaw",
			"max_items":  2,
		})
		if err != nil {
			t.Fatalf("aplicar lote supervisor: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("lote supervisor marcado como error: %#v", result)
		}
		structured, _ := result["structuredContent"].(map[string]any)
		if structured == nil {
			t.Fatalf("structuredContent vacío: %#v", result)
		}
		switch count := structured["count"].(type) {
		case int:
			if count != 2 {
				t.Fatalf("count inesperado: %#v", structured)
			}
		case float64:
			if count != 2 {
				t.Fatalf("count inesperado: %#v", structured)
			}
		default:
			t.Fatalf("count con tipo inesperado: %#v", structured)
		}
		risk, _ := structured["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil {
			t.Fatalf("falta critical_project_risk en batch: %#v", structured)
		}
		if strings.TrimSpace(risk.Project) != "orquestador" || risk.Blocking <= 0 {
			t.Fatalf("critical_project_risk inesperado: %#v", risk)
		}

		tareaLibre, err := db.GetTarea(tareaLibreID)
		if err != nil {
			t.Fatalf("get tarea libre: %v", err)
		}
		if tareaLibre.Estado != db.TareaEnProgreso || tareaLibre.Agente == nil || *tareaLibre.Agente != "Codex3" {
			t.Fatalf("tarea libre no aplicada por lote: %+v", tareaLibre)
		}
		tareaRetenida, err := db.GetTarea(tareaRetenidaID)
		if err != nil {
			t.Fatalf("get tarea retenida: %v", err)
		}
		if tareaRetenida.Estado != db.TareaEnProgreso || tareaRetenida.Agente == nil || *tareaRetenida.Agente != "Codex3" {
			t.Fatalf("tarea retenida no aplicada por lote: %+v", tareaRetenida)
		}
	})
}

func TestApplySupervisorRecommendedActionsBatchUsaSafeQueueYExponeProyectoCritico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevDrift := openClawWorktreeDriftBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			openClawWorktreeDriftBuilder = prevDrift
			statusNowFunc = prevNow
		}()

		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		projectLow := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
		projectHigh := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
		tareaLibreID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Dispatch low risk",
			Descripcion: "No debe ganar si solo cabe una acción segura",
			ProyectoID:  &projectLow,
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		tareaCriticaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Release API retenida",
			Descripcion: "Debe priorizarse por riesgo de proyecto",
			ProyectoID:  &projectHigh,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea crítica: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaCriticaID); err != nil {
			t.Fatalf("preparar tarea crítica retenida: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
			TareasActivas: []tareaLite{
				{ID: tareaCriticaID, Estado: db.TareaEnProgreso, Titulo: "Release API retenida", Agente: "Codex2", Modulo: "api", Prioridad: db.PrioridadAlta},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)

		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{
				{"slug": "webapp"},
				{"slug": "infra"},
			}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			switch strings.TrimSpace(slug) {
			case "infra":
				return &apiProyectoCockpit{
					Proyecto:                &db.Proyecto{Slug: "infra"},
					TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas:     1,
					RuntimeOrdersAbiertas:   1,
					RuntimeMailboxPendiente: 1,
					PropuestasAbiertas:      1,
				}, nil
			case "webapp":
				return &apiProyectoCockpit{
					Proyecto: &db.Proyecto{Slug: "webapp"},
				}, nil
			default:
				return nil, nil
			}
		}

		result, err := applySupervisorRecommendedActionsBatch("OpenClaw", 1, "")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedActionsBatch: %v", err)
		}
		if queueKind, _ := result["queue_kind"].(string); queueKind != "safe" {
			t.Fatalf("queue_kind inesperado: %#v", result["queue_kind"])
		}
		safeQueue, _ := result["safe_action_queue"].([]supervisorRecommendedAction)
		if len(safeQueue) == 0 || safeQueue[0].Target != fmt.Sprintf("tarea:%d", tareaCriticaID) {
			t.Fatalf("safe_action_queue debería abrir con la tarea crítica: %+v", safeQueue)
		}
		nextSafeAction, _ := result["next_safe_action"].(supervisorRecommendedAction)
		if nextSafeAction.Target != fmt.Sprintf("tarea:%d", tareaCriticaID) {
			t.Fatalf("next_safe_action inesperada: %+v", nextSafeAction)
		}
		risk, _ := result["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || risk.Project != "infra" || risk.Blocking != 15 {
			t.Fatalf("critical_project_risk inesperado: %#v", result["critical_project_risk"])
		}
		applied, _ := result["applied"].([]map[string]any)
		if len(applied) != 1 {
			t.Fatalf("applied inesperado: %#v", result["applied"])
		}
		if taskID, _ := applied[0]["task_id"].(int64); taskID != tareaCriticaID {
			t.Fatalf("el batch debería aplicar primero la tarea crítica, got=%#v", applied[0])
		}

		tareaLibre, err := db.GetTarea(tareaLibreID)
		if err != nil {
			t.Fatalf("get tarea libre: %v", err)
		}
		if tareaLibre.Estado != db.TareaLibre {
			t.Fatalf("la tarea libre no debería consumirse en este batch: %+v", tareaLibre)
		}
		tareaCritica, err := db.GetTarea(tareaCriticaID)
		if err != nil {
			t.Fatalf("get tarea crítica: %v", err)
		}
		if tareaCritica.Agente == nil || *tareaCritica.Agente != "Codex3" || tareaCritica.Estado != db.TareaEnProgreso {
			t.Fatalf("la tarea crítica debería quedar reasignada a Codex3: %+v", tareaCritica)
		}
	})
}

func TestApplySupervisorNextActionUsaSafeQueueYExponeProyectoCritico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			statusNowFunc = prevNow
		}()

		tareaLibreID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Dispatch low risk",
			Descripcion: "No debe ejecutarse antes que la crítica",
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		projectID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaCriticaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Retenida crítica",
			Descripcion: "Debe salir primero por next action",
			Modulo:      "controlplane",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
			ProyectoID:  &projectID,
		})
		if err != nil {
			t.Fatalf("crear tarea critica: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaCriticaID); err != nil {
			t.Fatalf("preparar tarea critica: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			AgentesTrabajando: []*db.Agente{},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
			TareasActivas: []tareaLite{
				{ID: tareaCriticaID, Estado: db.TareaEnProgreso, Titulo: "Retenida crítica", Agente: "Codex2"},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)
		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{{"slug": "orquestador"}}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "orquestador"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		}

		result, err := applySupervisorNextAction("OpenClaw")
		if err != nil {
			t.Fatalf("applySupervisorNextAction: %v", err)
		}
		if queueKind, _ := result["queue_kind"].(string); queueKind != "safe" {
			t.Fatalf("queue_kind inesperado: %#v", result["queue_kind"])
		}
		nextSafeAction, _ := result["next_safe_action"].(supervisorRecommendedAction)
		if nextSafeAction.Action != "replanificar_por_cuota" || nextSafeAction.Target != fmt.Sprintf("tarea:%d", tareaCriticaID) {
			t.Fatalf("next_safe_action inesperada: %+v", nextSafeAction)
		}
		risk, _ := result["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || strings.TrimSpace(risk.Project) != "orquestador" || risk.Blocking <= 0 {
			t.Fatalf("critical_project_risk inesperado: %#v", result["critical_project_risk"])
		}
		tareaLibre, err := db.GetTarea(tareaLibreID)
		if err != nil {
			t.Fatalf("get tarea libre: %v", err)
		}
		if tareaLibre.Estado != db.TareaLibre {
			t.Fatalf("la tarea low risk no deberia ejecutarse primero: %+v", tareaLibre)
		}
		tareaCritica, err := db.GetTarea(tareaCriticaID)
		if err != nil {
			t.Fatalf("get tarea critica: %v", err)
		}
		if tareaCritica.Agente == nil || *tareaCritica.Agente != "Codex3" {
			t.Fatalf("la tarea crítica no se aplicó primero: %+v", tareaCritica)
		}
	})
}

func TestApplySupervisorRecommendedActionExponeProyectoCriticoCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevDrift := openClawWorktreeDriftBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			openClawWorktreeDriftBuilder = prevDrift
			statusNowFunc = prevNow
		}()

		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		projectHigh := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
		tareaCriticaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Release API retenida",
			Descripcion: "Debe devolver riesgo canónico al aplicar una acción individual",
			ProyectoID:  &projectHigh,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea crítica: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaCriticaID); err != nil {
			t.Fatalf("preparar tarea crítica retenida: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasActivas: []tareaLite{
				{ID: tareaCriticaID, Estado: db.TareaEnProgreso, Titulo: "Release API retenida", Agente: "Codex2", Modulo: "api", Prioridad: db.PrioridadAlta},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)

		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{{"slug": "infra"}}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			return &apiProyectoCockpit{
				Proyecto:                &db.Proyecto{Slug: "infra"},
				TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
				ReviewGatesAbiertas:     1,
				RuntimeOrdersAbiertas:   1,
				RuntimeMailboxPendiente: 1,
				PropuestasAbiertas:      1,
			}, nil
		}

		result, err := applySupervisorRecommendedAction("OpenClaw", "replanificar_por_cuota", fmt.Sprintf("tarea:%d", tareaCriticaID), "Codex3")
		if err != nil {
			t.Fatalf("applySupervisorRecommendedAction: %v", err)
		}
		if queueKind, _ := result["queue_kind"].(string); queueKind != "safe" {
			t.Fatalf("queue_kind inesperado: %#v", result["queue_kind"])
		}
		nextSafeAction, _ := result["next_safe_action"].(supervisorRecommendedAction)
		if nextSafeAction.Target != fmt.Sprintf("tarea:%d", tareaCriticaID) {
			t.Fatalf("next_safe_action inesperada: %+v", nextSafeAction)
		}
		safeQueue, _ := result["safe_action_queue"].([]supervisorRecommendedAction)
		if len(safeQueue) == 0 || safeQueue[0].Target != fmt.Sprintf("tarea:%d", tareaCriticaID) {
			t.Fatalf("safe_action_queue inesperada: %+v", safeQueue)
		}
		risk, _ := result["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || risk.Project != "infra" || risk.Blocking != 15 {
			t.Fatalf("critical_project_risk inesperado: %#v", result["critical_project_risk"])
		}
		if taskID, _ := result["task_id"].(int64); taskID != tareaCriticaID {
			t.Fatalf("task_id inesperado: %#v", result)
		}
	})
}

func TestApplySupervisorNextActionExponeProyectoCriticoCanonico(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		prevListProjects := workspaceControlListProjects
		prevCockpitBuilder := workspaceControlCockpitBuilder
		prevDrift := openClawWorktreeDriftBuilder
		prevNow := statusNowFunc
		defer func() {
			statusService = prev
			workspaceControlListProjects = prevListProjects
			workspaceControlCockpitBuilder = prevCockpitBuilder
			openClawWorktreeDriftBuilder = prevDrift
			statusNowFunc = prevNow
		}()

		openClawWorktreeDriftBuilder = func(status apiStatusResponse) ([]apiOpenClawWorktreeDrift, error) {
			return nil, nil
		}

		projectLow := insertTestProyecto(t, "webapp", "webapp", "/tmp/webapp")
		projectHigh := insertTestProyecto(t, "infra", "infra", "/tmp/infra")
		tareaLibreID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Dispatch low risk",
			Descripcion: "No debe ganar si el proyecto crítico es otro",
			ProyectoID:  &projectLow,
			Modulo:      "web",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea libre: %v", err)
		}
		tareaCriticaID, err := db.CrearTarea(&db.Tarea{
			Titulo:      "Release API retenida",
			Descripcion: "Debe ejecutarse primero por riesgo canónico",
			ProyectoID:  &projectHigh,
			Modulo:      "api",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "alberto",
		})
		if err != nil {
			t.Fatalf("crear tarea crítica: %v", err)
		}
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrar Codex2: %v", err)
		}
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrar Codex3: %v", err)
		}
		if _, err := db.DB.Exec(`UPDATE tareas SET estado='en_progreso', agente='Codex2' WHERE id=?`, tareaCriticaID); err != nil {
			t.Fatalf("preparar tarea crítica retenida: %v", err)
		}

		now := time.Now().UTC()
		statusNowFunc = func() time.Time { return now }
		snapshotStatus := apiStatusResponse{
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			Agentes: []*db.Agente{
				{Nombre: "Codex2", Rol: "programador", Activo: false, EstadoCuota: "enfriamiento"},
				{Nombre: "Codex3", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaLibre): 1,
			},
			TareasActivas: []tareaLite{
				{ID: tareaCriticaID, Estado: db.TareaEnProgreso, Titulo: "Release API retenida", Agente: "Codex2", Modulo: "api", Prioridad: db.PrioridadAlta},
			},
		}
		statusService = stubStatusService{response: snapshotStatus}
		storeStatusSnapshotWithTTL(snapshotStatus, now, 5*time.Minute)

		workspaceControlListProjects = func() ([]map[string]any, error) {
			return []map[string]any{
				{"slug": "webapp"},
				{"slug": "infra"},
			}, nil
		}
		workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
			switch strings.TrimSpace(slug) {
			case "infra":
				return &apiProyectoCockpit{
					Proyecto:                &db.Proyecto{Slug: "infra"},
					TareasPorEstado:         map[string]int{string(db.TareaBloqueada): 1},
					ReviewGatesAbiertas:     1,
					RuntimeOrdersAbiertas:   1,
					RuntimeMailboxPendiente: 1,
					PropuestasAbiertas:      1,
				}, nil
			case "webapp":
				return &apiProyectoCockpit{
					Proyecto: &db.Proyecto{Slug: "webapp"},
				}, nil
			default:
				return nil, nil
			}
		}

		result, err := applySupervisorNextAction("OpenClaw")
		if err != nil {
			t.Fatalf("applySupervisorNextAction: %v", err)
		}
		if queueKind, _ := result["queue_kind"].(string); queueKind != "safe" {
			t.Fatalf("queue_kind inesperado: %#v", result["queue_kind"])
		}
		nextSafeAction, _ := result["next_safe_action"].(supervisorRecommendedAction)
		if nextSafeAction.Target != fmt.Sprintf("tarea:%d", tareaCriticaID) {
			t.Fatalf("next_safe_action inesperada: %+v", nextSafeAction)
		}
		risk, _ := result["critical_project_risk"].(*workspaceAutonomyProjectSummary)
		if risk == nil || risk.Project != "infra" || risk.Blocking != 15 {
			t.Fatalf("critical_project_risk inesperado: %#v", result["critical_project_risk"])
		}
		if taskID, _ := result["task_id"].(int64); taskID != tareaCriticaID {
			t.Fatalf("la next_action debería aplicar primero la tarea crítica, got=%#v", result)
		}

		tareaLibre, err := db.GetTarea(tareaLibreID)
		if err != nil {
			t.Fatalf("get tarea libre: %v", err)
		}
		if tareaLibre.Estado != db.TareaLibre {
			t.Fatalf("la tarea libre no debería ejecutarse primero: %+v", tareaLibre)
		}
	})
}

func TestMCPToolsCoordinacionOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex3", "programador"); err != nil {
			t.Fatalf("registrando Codex3: %v", err)
		}
		repo := prepararRepoGitMCP(t)
		insertTestProyecto(t, "orquestador", "orquestador", repo)

		assignResult, err := callMCPTool("orquesta.asignaciones.activar", map[string]any{
			"agente":   "Codex3",
			"proyecto": "orquestador",
			"nota":     "frente coordinado por MCP",
		})
		if err != nil {
			t.Fatalf("asignaciones activar MCP: %v", err)
		}
		if assignResult["isError"] != false {
			t.Fatalf("asignaciones activar marcado como error: %#v", assignResult)
		}

		lockResult, err := callMCPTool("orquesta.locks.adquirir", map[string]any{
			"agente":        "Codex3",
			"proyecto":      "orquestador",
			"scope_type":    "branch",
			"scope_key":     "orq-orquestador-codex3",
			"branch":        "orq-orquestador-codex3",
			"motivo":        "prueba MCP",
			"lease_seconds": 120,
		})
		if err != nil {
			t.Fatalf("locks adquirir MCP: %v", err)
		}
		if lockResult["isError"] != false {
			t.Fatalf("locks adquirir marcado como error: %#v", lockResult)
		}
		lock, _ := lockResult["structuredContent"].(*coordinacion.Lock)
		if lock == nil || lock.ID <= 0 || strings.TrimSpace(lock.LeaseToken) == "" {
			t.Fatalf("lock inválido: %#v", lockResult["structuredContent"])
		}

		worktreeResult, err := callMCPTool("orquesta.worktrees.preparar", map[string]any{
			"agente":   "Codex3",
			"proyecto": "orquestador",
			"lock_id":  lock.ID,
			"motivo":   "prueba MCP",
		})
		if err != nil {
			t.Fatalf("worktrees preparar MCP: %v", err)
		}
		if worktreeResult["isError"] != false {
			t.Fatalf("worktrees preparar marcado como error: %#v", worktreeResult)
		}
		worktree, _ := worktreeResult["structuredContent"].(*coordinacion.Worktree)
		if worktree == nil || worktree.ID <= 0 || strings.TrimSpace(worktree.Path) == "" {
			t.Fatalf("worktree inválida: %#v", worktreeResult["structuredContent"])
		}
		if _, err := os.Stat(worktree.Path); err != nil {
			t.Fatalf("worktree no creada en disco: %v", err)
		}

		closeResult, err := callMCPTool("orquesta.worktrees.cerrar", map[string]any{
			"id":       worktree.ID,
			"eliminar": true,
			"motivo":   "fin prueba MCP",
		})
		if err != nil {
			t.Fatalf("worktrees cerrar MCP: %v", err)
		}
		if closeResult["isError"] != false {
			t.Fatalf("worktrees cerrar marcado como error: %#v", closeResult)
		}

		releaseResult, err := callMCPTool("orquesta.locks.liberar", map[string]any{
			"id":          lock.ID,
			"agente":      "Codex3",
			"lease_token": lock.LeaseToken,
			"motivo":      "fin prueba MCP",
		})
		if err != nil {
			t.Fatalf("locks liberar MCP: %v", err)
		}
		if releaseResult["isError"] != false {
			t.Fatalf("locks liberar marcado como error: %#v", releaseResult)
		}
	})
}

func TestMCPToolVotarPropuestaActualizaEstado(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		if err := db.ConfigSet("propuesta_min_votes", "1"); err != nil {
			t.Fatalf("config propuesta_min_votes: %v", err)
		}
		if err := db.ConfigSet("propuesta_min_non_author_votes", "1"); err != nil {
			t.Fatalf("config propuesta_min_non_author_votes: %v", err)
		}
		propuestaID := insertTestPropuesta(t, "OP-049", "Arquitectura objetivo", "Descripcion")
		insertTestVotoPendiente(t, propuestaID, "Codex2")

		result, err := callMCPTool("orquesta.propuestas.votar", map[string]any{
			"codigo":     "OP-049",
			"agente":     "Codex2",
			"posicion":   "acuerdo",
			"comentario": "Coherente y evolutiva",
		})
		if err != nil {
			t.Fatalf("callMCPTool: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("resultado marcado como error: %#v", result)
		}

		p, err := db.GetPropuesta("OP-049")
		if err != nil {
			t.Fatalf("leyendo propuesta: %v", err)
		}
		if p.Estado != db.PropuestaConsenso {
			t.Fatalf("estado inesperado tras votar: %s", p.Estado)
		}
	})
}

func TestMCPResourceReadProyectoIncluyeWorktreesYAsignaciones(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestAsignacion(t, "Codex2", proyectoID, "mcp server y recursos")
		insertTestWorktree(t, proyectoID, "Codex2", "orquestador-codex2", "/tmp/orquestador/.orquesta-worktrees/orquestador-codex2", "orq-orquestador-codex2")

		contents, err := readMCPResource("orquesta://proyectos/orquestador")
		if err != nil {
			t.Fatalf("readMCPResource proyecto: %v", err)
		}
		text, _ := contents[0]["text"].(string)
		if !strings.Contains(text, "orquestador-codex2") {
			t.Fatalf("faltan worktrees en el recurso: %s", text)
		}
		if !strings.Contains(text, "mcp server y recursos") {
			t.Fatalf("faltan asignaciones en el recurso: %s", text)
		}
	})
}

func TestMCPResourcesIncluyenProjectControl(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		resources, err := listMCPResources()
		if err != nil {
			t.Fatalf("listMCPResources: %v", err)
		}
		for _, want := range []string{"orquesta://proyectos/orquestador/control"} {
			found := false
			for _, item := range resources {
				if item.URI == want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("resource %s no listada", want)
			}
		}

		templates := mcpResourceTemplates()
		foundTemplate := false
		for _, item := range templates {
			if item.URITemplate == "orquesta://proyectos/{slug}/control" {
				foundTemplate = true
				break
			}
		}
		if !foundTemplate {
			t.Fatal("template de proyecto control no listada")
		}
	})
}

func TestMCPResourceReadProyectoControlAceptaDesde(t *testing.T) {
	prevBuilder := workspaceControlProjectBuilder
	defer func() {
		workspaceControlProjectBuilder = prevBuilder
	}()

	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project:   &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:     since,
			Generated: time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC),
			Git: projectControlGitAggregate{
				FilesChanged: 3,
				Insertions:   11,
				Deletions:    4,
				LinesNet:     7,
			},
		}, nil
	}

	contents, err := readMCPResource("orquesta://proyectos/orquestador/control?desde=2026-04-24T13:00:00Z")
	if err != nil {
		t.Fatalf("readMCPResource proyecto/control?desde: %v", err)
	}
	text, _ := contents[0]["text"].(string)
	for _, token := range []string{`"since": "2026-04-24T13:00:00Z"`, `"files_changed": 3`, `"lines_net": 7`, `"Slug": "orquestador"`} {
		if !strings.Contains(text, token) {
			t.Fatalf("resource proyecto/control sin %q: %s", token, text)
		}
	}
}

func TestMCPResourceReadProyectoControlUsa24hPorDefecto(t *testing.T) {
	prevBuilder := workspaceControlProjectBuilder
	defer func() {
		workspaceControlProjectBuilder = prevBuilder
	}()

	before := time.Now().UTC()
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:   since,
		}, nil
	}

	contents, err := readMCPResource("orquesta://proyectos/orquestador/control")
	if err != nil {
		t.Fatalf("readMCPResource proyecto/control: %v", err)
	}
	after := time.Now().UTC()
	text, _ := contents[0]["text"].(string)
	var resp apiProyectoControlResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("decode proyecto/control: %v", err)
	}
	if resp.Control == nil {
		t.Fatalf("control vacío: %s", text)
	}
	minWant := before.Add(-24 * time.Hour).Add(-2 * time.Second)
	maxWant := after.Add(-24 * time.Hour).Add(2 * time.Second)
	if resp.Control.Since.Before(minWant) || resp.Control.Since.After(maxWant) {
		t.Fatalf("since MCP proyecto/control por defecto inesperado: got=%s want_between=[%s,%s]", resp.Control.Since, minWant, maxWant)
	}
}

func TestMCPToolProyectoControlExponeGitYVentana(t *testing.T) {
	prevBuilder := workspaceControlProjectBuilder
	defer func() {
		workspaceControlProjectBuilder = prevBuilder
	}()

	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project:   &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:     since,
			Generated: time.Date(2026, 4, 29, 9, 30, 0, 0, time.UTC),
			Git: projectControlGitAggregate{
				FilesChanged: 2,
				Insertions:   9,
				Deletions:    1,
				LinesNet:     8,
			},
		}, nil
	}

	result, err := callMCPTool("orquesta.proyectos.control", map[string]any{
		"slug":  "orquestador",
		"desde": "48h",
	})
	if err != nil {
		t.Fatalf("callMCPTool proyectos.control: %v", err)
	}
	text := result["content"].([]map[string]any)[0]["text"].(string)
	for _, token := range []string{`"files_changed": 2`, `"insertions": 9`, `"lines_net": 8`, `"Slug": "orquestador"`} {
		if !strings.Contains(text, token) {
			t.Fatalf("tool proyectos.control sin %q: %s", token, text)
		}
	}
	structured, ok := result["structuredContent"].(apiProyectoControlResponse)
	if !ok || structured.Control == nil {
		t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
	}
	if structured.Control.Git.FilesChanged != 2 || structured.Control.Git.LinesNet != 8 {
		t.Fatalf("git inesperado en structuredContent: %+v", structured.Control.Git)
	}
}

func TestMCPToolProyectoControlUsa24hPorDefecto(t *testing.T) {
	prevBuilder := workspaceControlProjectBuilder
	defer func() {
		workspaceControlProjectBuilder = prevBuilder
	}()

	before := time.Now().UTC()
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:   since,
		}, nil
	}

	result, err := callMCPTool("orquesta.proyectos.control", map[string]any{
		"slug": "orquestador",
	})
	if err != nil {
		t.Fatalf("callMCPTool proyectos.control: %v", err)
	}
	after := time.Now().UTC()
	structured, ok := result["structuredContent"].(apiProyectoControlResponse)
	if !ok || structured.Control == nil {
		t.Fatalf("structuredContent inesperado: %#v", result["structuredContent"])
	}
	minWant := before.Add(-24 * time.Hour).Add(-2 * time.Second)
	maxWant := after.Add(-24 * time.Hour).Add(2 * time.Second)
	if structured.Control.Since.Before(minWant) || structured.Control.Since.After(maxWant) {
		t.Fatalf("since MCP proyectos.control por defecto inesperado: got=%s want_between=[%s,%s]", structured.Control.Since, minWant, maxWant)
	}
}

func TestMCPResourceReadWorkspaceControlIncluyeResumenGlobal(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
	}()

	statusService = stubStatusService{response: apiStatusResponse{
		TareasPorEstado: map[string]int{"en_progreso": 2},
		DeudaDispatch:   deudaDispatchResumen{Total: 1, Pendientes: 1},
		Autonomia: autonomiaResumen{
			Supervisando: 1,
			Count:        1,
			ByKind:       map[string]int{"task_reassigned": 1},
		},
		WorkersConectados:   2,
		WorkersTrabajando:   1,
		SupervisoresActivos: 1,
	}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "orquestador"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		ts := time.Date(2026, 4, 24, 15, 0, 0, 0, time.UTC)
		return &apiProyectoCockpit{
			Proyecto:        &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			TareasPorEstado: map[string]int{"en_progreso": 2},
			AutonomyEvents:  1,
			AutonomyByKind:  map[string]int{"task_reassigned": 1},
			AutonomyLastAt:  &ts,
			Autonomy: []autonomyEventSummary{{
				Kind:        "task_reassigned",
				CreatedAt:   ts,
				TargetAgent: "Codex4",
			}},
		}, nil
	}

	before := time.Now().UTC()
	contents, err := readMCPResource("orquesta://workspace/control")
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("readMCPResource workspace/control: %v", err)
	}
	text, _ := contents[0]["text"].(string)
	for _, token := range []string{`"active_projects": 1`, `"workers_conectados": 2`, `"task_reassigned"`, `"orquestador"`} {
		if !strings.Contains(text, token) {
			t.Fatalf("falta %q en resource workspace control: %s", token, text)
		}
	}
	var resp apiWorkspaceControlResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("decode workspace control: %v", err)
	}
	if resp.Control == nil {
		t.Fatal("control global vacio")
	}
	minWant := before.Add(-workspaceControlDefaultWindow).Add(-2 * time.Second)
	maxWant := after.Add(-workspaceControlDefaultWindow).Add(2 * time.Second)
	if resp.Control.Since.Before(minWant) || resp.Control.Since.After(maxWant) {
		t.Fatalf("since MCP por defecto inesperado: got=%s want_between=[%s,%s]", resp.Control.Since, minWant, maxWant)
	}
}

func TestMCPResourcesIncluyenServerOperational(t *testing.T) {
	withTempOrquestaDB(t, func() {
		resources, err := listMCPResources()
		if err != nil {
			t.Fatalf("listMCPResources: %v", err)
		}
		for _, want := range []string{"orquesta://server/operational", "orquesta://workspace/control"} {
			found := false
			for _, item := range resources {
				if item.URI == want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("resource %s no listada", want)
			}
		}
	})
}

func TestMCPResourceReadServerOperational(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	now := time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC)
	storeStatusSnapshotWithTTL(apiStatusResponse{
		Agentes: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			{Nombre: "Codex2", Rol: "programador", EstadoCuota: "enfriamiento"},
		},
		AgentesActivos: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		AgentesTrabajando: []*db.Agente{
			{Nombre: "Codex1", Rol: "programador", Activo: true, EstadoCuota: "activo"},
		},
		TareasEnProgreso: []tareaLite{
			{ID: 41, Estado: db.TareaEnProgreso, Agente: "Codex1", Titulo: "vigilar runtime"},
		},
		TareasPorEstado: map[string]int{
			string(db.TareaEnProgreso): 1,
		},
		Generado: now.Format(time.RFC3339),
	}, now, time.Minute)

	contents, err := readMCPResource("orquesta://server/operational")
	if err != nil {
		t.Fatalf("readMCPResource server/operational: %v", err)
	}
	if len(contents) == 0 {
		t.Fatal("resource server/operational vacía")
	}
	text, _ := contents[0]["text"].(string)
	for _, token := range []string{`"state": "ready"`, `"operational": true`, `"activeAgents": 1`, `"tasksInProgress": 1`} {
		if !strings.Contains(text, token) {
			t.Fatalf("falta %q en resource server/operational: %s", token, text)
		}
	}
	var info serverOperationalInfo
	if err := json.Unmarshal([]byte(text), &info); err != nil {
		t.Fatalf("decode server operational: %v", err)
	}
	if info.State != "ready" || !info.Operational {
		t.Fatalf("server operational inesperado: %+v", info)
	}
	if info.ActiveAgents != 1 || info.TasksInProgress != 1 {
		t.Fatalf("contadores server operational inesperados: %+v", info)
	}
}

func TestMCPResourceReadServerOperationalIncluyeRecoveryHint(t *testing.T) {
	resetStatusSnapshotCache()
	defer resetStatusSnapshotCache()

	prevReview := serverOperationalReviewSnapshotFn
	prevFast := statusFastFetcher
	prevUltraLite := apiStatusUltraLiteFetcher
	defer func() {
		serverOperationalReviewSnapshotFn = prevReview
		statusFastFetcher = prevFast
		apiStatusUltraLiteFetcher = prevUltraLite
	}()
	serverOperationalReviewSnapshotFn = func(supervisor string) (map[string]any, error) {
		return map[string]any{
			"supervisor": supervisor,
			"next_safe_action": supervisorRecommendedAction{
				Target: "tarea:51",
				Action: "reanudar_worker",
				Reason: "worker parado con backlog pendiente",
			},
		}, nil
	}
	statusFastFetcher = func() (apiStatusResponse, error) {
		return apiStatusResponse{
			Agentes: []*db.Agente{
				{Nombre: "CodexGap", Rol: "programador", Activo: true, EstadoCuota: "activo"},
			},
			TareasEnProgreso: []tareaLite{
				{ID: 51, Estado: db.TareaEnProgreso, Agente: "CodexGap", Titulo: "reanudar runtime"},
				{ID: 52, Estado: db.TareaEnProgreso, Agente: "CodexGap", Titulo: "cerrar bloqueo"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaEnProgreso): 2,
			},
			Generado: time.Now().UTC().Format(time.RFC3339),
		}, nil
	}
	apiStatusUltraLiteFetcher = func(timeout time.Duration) (apiStatusResponse, bool) {
		return apiStatusResponse{}, false
	}

	contents, err := readMCPResource("orquesta://server/operational")
	if err != nil {
		t.Fatalf("readMCPResource server/operational recovery: %v", err)
	}
	text, _ := contents[0]["text"].(string)
	for _, token := range []string{`"recovery"`, `"kind": "worker_gap"`, `"suggestedAction": "server_rearm"`} {
		if !strings.Contains(text, token) {
			t.Fatalf("falta %q en resource server/operational: %s", token, text)
		}
	}
	var info serverOperationalInfo
	if err := json.Unmarshal([]byte(text), &info); err != nil {
		t.Fatalf("decode server operational recovery: %v", err)
	}
	if info.Recovery == nil || info.Recovery.Kind != "worker_gap" || info.Recovery.MissingWorkers != 2 {
		t.Fatalf("recovery MCP inesperada: %+v", info.Recovery)
	}
}

func TestMCPResourceReadAgentesPanelCanonico(t *testing.T) {
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "CodexPanel", Rol: "programador", Activo: true, Habilitado: true},
		EstadoOperativo: "trabajando",
		WorkerAlive:     true,
		WorkerState:     "running",
		MailboxPending:  2,
		OpenTasks:       1,
	}}, now)

	contents, err := readMCPResource("orquesta://agentes/panel")
	if err != nil {
		t.Fatalf("readMCPResource agentes/panel: %v", err)
	}
	text, _ := contents[0]["text"].(string)
	if !strings.Contains(text, "\"agents\"") || !strings.Contains(text, "\"name\": \"CodexPanel\"") {
		t.Fatalf("resource agentes/panel inesperado: %s", text)
	}
}

func TestMCPResourceReadAgentesActivosCanonical(t *testing.T) {
	resetAgentPanelSnapshotCache()
	defer resetAgentPanelSnapshotCache()

	now := time.Now().UTC()
	storeAgentPanelSnapshot([]agentesapp.Row{
		{
			Agente:          &db.Agente{Nombre: "CodexActive", Rol: "programador", Activo: true, Habilitado: true},
			EstadoOperativo: "trabajando",
			WorkerAlive:     true,
			WorkerState:     "running",
			OpenTasks:       1,
		},
		{
			Agente:          &db.Agente{Nombre: "CodexIdle", Rol: "programador", Activo: false, Habilitado: true},
			EstadoOperativo: "disponible",
		},
	}, now)

	contents, err := readMCPResource("orquesta://agentes/activos?schema=canonical")
	if err != nil {
		t.Fatalf("readMCPResource agentes/activos canonical: %v", err)
	}
	text, _ := contents[0]["text"].(string)
	if !strings.Contains(text, "\"agents\"") || !strings.Contains(text, "\"name\": \"CodexActive\"") {
		t.Fatalf("resource agentes/activos canonical inesperado: %s", text)
	}
	if strings.Contains(text, "\"CodexIdle\"") {
		t.Fatalf("resource agentes/activos canonical no deberia incluir agentes inactivos: %s", text)
	}
}

func TestMCPResourceReadAgentesActivosLegacySigueSiendoArray(t *testing.T) {
	withTempOrquestaDB(t, func() {
		contents, err := readMCPResource("orquesta://agentes/activos")
		if err != nil {
			t.Fatalf("readMCPResource agentes/activos legacy: %v", err)
		}
		text, _ := contents[0]["text"].(string)
		trimmed := strings.TrimSpace(text)
		if trimmed != "null" && !strings.HasPrefix(trimmed, "[") {
			t.Fatalf("legacy agentes/activos no deberia cambiar a objeto canónico: %s", text)
		}
	})
}

func TestMCPToolAgentesOverviewCanonical(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("CodexOverviewCanonical", "programador"); err != nil {
			t.Fatalf("registrando CodexOverviewCanonical: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		if _, err := db.IniciarSesionContexto(db.SesionInicio{
			Agente:      "CodexOverviewCanonical",
			ProyectoID:  &proyectoID,
			CWD:         "/tmp/orquestador",
			Herramienta: "codex-cli",
		}); err != nil {
			t.Fatalf("iniciar sesion: %v", err)
		}
		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:     "canonical overview",
			ProyectoID: &proyectoID,
			Modulo:     "cmd",
			Prioridad:  db.PrioridadAlta,
			CreadoPor:  "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.TomarTarea(tareaID, "CodexOverviewCanonical"); err != nil {
			t.Fatalf("tomar tarea: %v", err)
		}
		if err := db.IniciarTarea(tareaID, "CodexOverviewCanonical"); err != nil {
			t.Fatalf("iniciar tarea: %v", err)
		}

		result, err := callMCPTool("orquesta.agentes.overview", map[string]any{
			"agente": "CodexOverviewCanonical",
			"schema": "canonical",
		})
		if err != nil {
			t.Fatalf("agentes overview canonical MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes overview canonical marcado como error: %#v", result)
		}
		resp, _ := result["structuredContent"].(apiAgenteOverviewResponse)
		if resp.Detail == nil || resp.Detail.Entity == nil || resp.Detail.Entity.Name != "CodexOverviewCanonical" {
			t.Fatalf("detail legacy inesperado: %#v", result["structuredContent"])
		}
		if resp.Agent == nil || resp.Agent.Entity == nil || resp.Agent.Entity.CurrentTask == nil || resp.Agent.Entity.CurrentTask.TaskID != tareaID {
			t.Fatalf("agent canónico sin foco de tarea esperado: %#v", resp.Agent)
		}
	})
}

func TestMCPResourceReadWorkspaceControlAceptaDesde(t *testing.T) {
	prevStatus := statusService
	prevProjects := workspaceControlListProjects
	prevCockpit := workspaceControlCockpitBuilder
	prevProjectControl := workspaceControlProjectBuilder
	defer func() {
		statusService = prevStatus
		workspaceControlListProjects = prevProjects
		workspaceControlCockpitBuilder = prevCockpit
		workspaceControlProjectBuilder = prevProjectControl
	}()

	statusService = stubStatusService{response: apiStatusResponse{}}
	workspaceControlListProjects = func() ([]map[string]any, error) {
		return []map[string]any{{"slug": "orquestador"}}, nil
	}
	workspaceControlCockpitBuilder = func(slug string) (*apiProyectoCockpit, error) {
		return &apiProyectoCockpit{Proyecto: &db.Proyecto{Slug: slug, Nombre: "Orquestador"}}, nil
	}
	workspaceControlProjectBuilder = func(slug string, since time.Time) (*projectControlReport, error) {
		return &projectControlReport{
			Project: &db.Proyecto{Slug: slug, Nombre: "Orquestador"},
			Since:   since,
		}, nil
	}

	contents, err := readMCPResource("orquesta://workspace/control?desde=2026-04-24T13:00:00Z")
	if err != nil {
		t.Fatalf("readMCPResource workspace/control?desde: %v", err)
	}
	text, _ := contents[0]["text"].(string)
	if !strings.Contains(text, `"since": "2026-04-24T13:00:00Z"`) {
		t.Fatalf("resource workspace control sin since propagado: %s", text)
	}
}

func TestMCPResourceReadWorkspaceControlRechazaDesdeInvalido(t *testing.T) {
	contents, err := readMCPResource("orquesta://workspace/control?desde=xxx")
	if err == nil {
		t.Fatalf("readMCPResource workspace/control?desde=xxx deberia fallar, contents=%#v", contents)
	}
	if !strings.Contains(err.Error(), "valor --desde inválido") {
		t.Fatalf("error inesperado para workspace/control?desde=xxx: %v", err)
	}
}

func TestMCPToolListaProyectosFiltrados(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestProyectoConTipoActivo(t, "grupo-pm", "PlataformaMunicipal", "/tmp/PlataformaMunicipal", "grupo", true)
		insertTestProyectoConTipoActivo(t, "repo-inactivo", "repo-inactivo", "/tmp/repo-inactivo", "repo", false)

		result, err := callMCPTool("orquesta.proyectos.listar", map[string]any{
			"tipo":   "repo",
			"activo": true,
		})
		if err != nil {
			t.Fatalf("callMCPTool proyectos.listar: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(text, `"slug": "orquestador"`) {
			t.Fatalf("no aparece el repo esperado: %s", text)
		}
		if strings.Contains(text, "repo-inactivo") {
			t.Fatalf("aparece un proyecto inactivo filtrado: %s", text)
		}
		if strings.Contains(text, "grupo-pm") {
			t.Fatalf("aparece un proyecto de otro tipo: %s", text)
		}
	})
}

func TestMCPPromptContextoProyectoIncluyeBloquesOperativos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestAsignacion(t, "Codex2", proyectoID, "mcp server y recursos")

		result, err := getMCPPrompt("orquesta.contexto.proyecto", map[string]any{"slug": "orquestador"})
		if err != nil {
			t.Fatalf("getMCPPrompt contexto.proyecto: %v", err)
		}
		messages := result["messages"].([]mcpPromptMessage)
		content := messages[0].Content.(map[string]any)
		text := content["text"].(string)
		if !strings.Contains(text, "Asignaciones activas") {
			t.Fatalf("faltan asignaciones en prompt: %s", text)
		}
		if !strings.Contains(text, "siguiente bloque tecnico con mejor retorno") {
			t.Fatalf("falta la guia de salida esperada: %s", text)
		}
	})
}

func TestMCPResourceReadPoolIncluyeModelosYCapacidad(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 1)
		insertTestPoolModelo(t, "codex", "gpt-5.4", 10, 1.5)

		contents, err := readMCPResource("orquesta://pools/codex")
		if err != nil {
			t.Fatalf("readMCPResource pool: %v", err)
		}
		text, _ := contents[0]["text"].(string)
		if !strings.Contains(text, "gpt-5.4") {
			t.Fatalf("faltan modelos en el recurso: %s", text)
		}
		if !strings.Contains(text, "capacidad_disponible") {
			t.Fatalf("falta resumen de capacidad en el recurso: %s", text)
		}
	})
}

func TestMCPToolListaPoolsYModelos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 1)
		insertTestPoolModelo(t, "codex", "gpt-5.4", 10, 1.5)

		poolsResult, err := callMCPTool("orquesta.pools.listar", map[string]any{})
		if err != nil {
			t.Fatalf("callMCPTool pools.listar: %v", err)
		}
		poolsText := poolsResult["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(poolsText, `"slug": "codex"`) {
			t.Fatalf("no aparece el pool esperado: %s", poolsText)
		}

		modelosResult, err := callMCPTool("orquesta.pools.modelos", map[string]any{"pool_slug": "codex"})
		if err != nil {
			t.Fatalf("callMCPTool pools.modelos: %v", err)
		}
		modelosText := modelosResult["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(modelosText, "gpt-5.4") {
			t.Fatalf("no aparece el modelo esperado: %s", modelosText)
		}
	})
}

func TestMCPToolResuelveModeloPorPolitica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 0)
		insertTestPoolModelo(t, "codex", "gpt-5.4", 10, 1.5)
		insertTestPoolModelo(t, "codex", "gpt-5.4-mini", 20, 0.5)
		insertTestPoliticaModelo(t, db.PoliticaModelo{
			ScopeTipo:       "perfil",
			ScopeRef:        "script",
			PerfilTarea:     "script",
			PoolSlug:        "codex",
			ReasoningEffort: "medium",
			Prioridad:       10,
			Activa:          true,
		})

		result, err := callMCPTool("orquesta.modelos.resolver", map[string]any{
			"perfil": "script",
		})
		if err != nil {
			t.Fatalf("callMCPTool modelos.resolver: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(text, `"pool_slug": "codex"`) {
			t.Fatalf("pool no resuelto correctamente: %s", text)
		}
		if !strings.Contains(text, `"model_slug": "gpt-5.4-mini"`) {
			t.Fatalf("modelo economico no resuelto: %s", text)
		}
		if !strings.Contains(text, `"reasoning_effort": "medium"`) {
			t.Fatalf("reasoning no resuelto: %s", text)
		}
	})
}

func TestMCPToolDetectaCarenciaSkillPorAgente(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		if _, err := db.CrearSkill("Codex1", &db.Skill{
			TipoAgente:       "programador",
			Nombre:           "gofmt",
			Descripcion:      "formateo go",
			CuandoUsar:       "formatear codigo go",
			Escenario:        "codigo",
			HerramientasJSON: `["gofmt"]`,
			Activa:           true,
		}); err != nil {
			t.Fatalf("CrearSkill: %v", err)
		}

		result, err := callMCPTool("orquesta.skills.detectar-carencia", map[string]any{
			"agente":            "Codex2",
			"nombre":            "goimports",
			"descripcion":       "ordenar imports y formatear go",
			"cuando_usar":       "corregir imports y formato en codigo go",
			"escenario":         "codigo",
			"herramientas_json": `["goimports"]`,
		})
		if err != nil {
			t.Fatalf("callMCPTool skills.detectar-carencia: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(text, `"falta": true`) {
			t.Fatalf("no marca skill faltante: %s", text)
		}
		if !strings.Contains(text, "$skill-creator") {
			t.Fatalf("no propone skill creator: %s", text)
		}
	})
}

func TestMCPResourceReadPoliticasModelo(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 0)
		insertTestPoliticaModelo(t, db.PoliticaModelo{
			ScopeTipo:       "perfil",
			ScopeRef:        "orquestacion",
			PerfilTarea:     "orquestacion",
			PoolSlug:        "codex",
			ReasoningEffort: "xhigh",
			Prioridad:       10,
			Activa:          true,
		})

		contents, err := readMCPResource("orquesta://politicas-modelo")
		if err != nil {
			t.Fatalf("readMCPResource politicas-modelo: %v", err)
		}
		text := contents[0]["text"].(string)
		if !strings.Contains(text, `"scope_tipo": "perfil"`) {
			t.Fatalf("falta scope_tipo en recurso: %s", text)
		}
		if !strings.Contains(text, `"reasoning_effort": "xhigh"`) {
			t.Fatalf("falta reasoning en recurso: %s", text)
		}
	})
}

func withTempOrquestaDB(t *testing.T, fn func()) {
	t.Helper()
	prepararDBTemporalCmd(t)
	fn()
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func insertTestPropuesta(t *testing.T, codigo, titulo, descripcion string) int64 {
	t.Helper()
	var id int64
	if err := db.DB.QueryRow(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, propuesto_por, distribuidor)
		VALUES (?,?,?,?,?,?)
		RETURNING id`,
		codigo, titulo, descripcion, "arquitectura", "Codex1", "Codex1",
	).Scan(&id); err != nil {
		t.Fatalf("insert propuesta: %v", err)
	}
	return id
}

func insertTestVotoPendiente(t *testing.T, propuestaID int64, agente string) {
	t.Helper()
	if _, err := db.DB.Exec(
		`INSERT INTO votos (propuesta_id, agente, posicion, comentario) VALUES (?,?,?,?)`,
		propuestaID, agente, db.VotoPendiente, "",
	); err != nil {
		t.Fatalf("insert voto pendiente: %v", err)
	}
}

func insertTestProyecto(t *testing.T, slug, nombre, rutaAbs string) int64 {
	t.Helper()
	return insertTestProyectoConTipoActivo(t, slug, nombre, rutaAbs, "repo", true)
}

func insertTestProyectoConTipoActivo(t *testing.T, slug, nombre, rutaAbs, tipo string, activo bool) int64 {
	t.Helper()
	var id int64
	if err := db.DB.QueryRow(`
		INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo)
		VALUES (?,?,?,?,?)
		RETURNING id`,
		slug, nombre, rutaAbs, tipo, activo,
	).Scan(&id); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	return id
}

func insertTestAsignacion(t *testing.T, agente string, proyectoID int64, nota string) {
	t.Helper()
	if _, err := db.DB.Exec(`
		INSERT INTO asignaciones (agente, proyecto_id, estado, nota)
		VALUES (?,?,?,?)`,
		agente, proyectoID, "activa", nota,
	); err != nil {
		t.Fatalf("insert asignacion: %v", err)
	}
}

func insertTestWorktree(t *testing.T, proyectoID int64, agente, nombre, rutaAbs, branch string) {
	t.Helper()
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, estado)
		VALUES (?,?,?,?,?,?)`,
		proyectoID, agente, nombre, rutaAbs, branch, "activa",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
}

func insertTestPool(t *testing.T, slug, proveedor, runtime string, capacidadTotal, capacidadReservada int) {
	t.Helper()
	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:                slug,
		Proveedor:           proveedor,
		Runtime:             runtime,
		Plan:                "default",
		EsDePago:            true,
		CapacidadTotal:      capacidadTotal,
		CapacidadReservada:  capacidadReservada,
		PermiteHijos:        true,
		PermiteModelosMulti: true,
		PermiteSobrecoste:   false,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        "{}",
		Activo:              true,
	}); err != nil {
		t.Fatalf("insert pool: %v", err)
	}
}

func prepararRepoGitMCP(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "orquestador")
	cmdGitMCP(t, "", "init", "-b", "master", repo)
	cmdGitMCP(t, repo, "config", "user.name", "Orquesta Test")
	cmdGitMCP(t, repo, "config", "user.email", "orquesta@example.test")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	cmdGitMCP(t, repo, "add", "README.md")
	cmdGitMCP(t, repo, "commit", "-m", "base")
	return repo
}

func cmdGitMCP(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func insertTestPoolModelo(t *testing.T, poolSlug, modelSlug string, prioridad int, coste float64) {
	t.Helper()
	if _, err := db.GuardarPoolModelo(poolSlug, &db.PoolModelo{
		ModelSlug:          modelSlug,
		Activo:             true,
		Prioridad:          prioridad,
		CosteRelativo:      coste,
		LimiteConocidoJSON: "{}",
	}); err != nil {
		t.Fatalf("insert pool modelo: %v", err)
	}
}

func insertTestPoliticaModelo(t *testing.T, politica db.PoliticaModelo) {
	t.Helper()
	if _, err := db.GuardarPoliticaModelo(&politica); err != nil {
		t.Fatalf("insert politica modelo: %v", err)
	}
}
