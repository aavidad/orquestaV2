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
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/propuestasapp"
	"orquesta/reviewapp"
)

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
		resourceText, _ := contents[0]["text"].(string)
		if !strings.Contains(resourceText, "ready_for_review") {
			t.Fatalf("recurso de revisión sin signal esperado: %s", resourceText)
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
		if _, ok := structured["next_safe_action"]; !ok {
			t.Fatalf("falta next_safe_action: %#v", structured)
		}
		if queue := reflect.ValueOf(structured["safe_action_queue"]); !queue.IsValid() {
			t.Fatalf("falta safe_action_queue: %#v", structured)
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

func TestMCPToolSupervisorConsumeGuidanceDurable(t *testing.T) {
	withTempOrquestaDB(t, func() {
		prev := statusService
		defer func() { statusService = prev }()

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
		if found == nil || found.Estado != "consumido" {
			t.Fatalf("mailbox no consumida: %+v", found)
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
		defer func() { statusService = prev }()

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

		statusService = stubStatusService{response: apiStatusResponse{
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
		}}

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
