package cmd

import (
	"strconv"
	"strings"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestMCPOpenClawOperatorGobiernaRuntimePorMCPSinCLI(t *testing.T) {
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
		runtimeInst, err := db.GetRuntimeBySesionID(sesion.ID)
		if err != nil || runtimeInst == nil {
			t.Fatalf("runtime esperado, got=%+v err=%v", runtimeInst, err)
		}

		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:     "Recuperar runtime bloqueado desde MCP",
			ProyectoID: &proyectoID,
			Modulo:     "cmd",
			Prioridad:  db.PrioridadAlta,
			CreadoPor:  "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		if err := db.TomarTarea(tareaID, "Codex3"); err != nil {
			t.Fatalf("tomar tarea: %v", err)
		}
		if err := db.IniciarTarea(tareaID, "Codex3"); err != nil {
			t.Fatalf("iniciar tarea: %v", err)
		}

		if _, err := db.RegistrarRuntimeEvent(&db.RuntimeEvent{
			RuntimeID:   runtimeInst.ID,
			Kind:        "approval_request",
			Level:       "warning",
			Message:     "Necesito arbitraje para retomar el runtime",
			PayloadJSON: `{"classification":"approval_request","suggested_action":"revisar_runtime"}`,
		}); err != nil {
			t.Fatalf("registrar runtime event: %v", err)
		}

		supervisionResult, err := callMCPTool("orquesta.supervision.eventos", map[string]any{
			"supervisor": "OpenClaw",
			"limit":      5,
		})
		if err != nil {
			t.Fatalf("supervision eventos MCP: %v", err)
		}
		if supervisionResult["isError"] != false {
			t.Fatalf("supervision eventos marcado como error: %#v", supervisionResult)
		}
		supervisionPayload, _ := supervisionResult["structuredContent"].(map[string]any)
		normalized, _ := supervisionPayload["normalized_events"].([]openClawNormalizedEvent)
		if !containsNormalizedEvent(normalized, "supervisor.approval_required", "Codex3", "orquestador") {
			t.Fatalf("cola supervisada sin approval_request normalizado: %+v", normalized)
		}

		overviewResult, err := callMCPTool("orquesta.agentes.overview", map[string]any{"agente": "Codex3"})
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

		handlesResult, err := callMCPTool("orquesta.runtime.handles.listar", map[string]any{"agente": "Codex3"})
		if err != nil {
			t.Fatalf("runtime handles MCP: %v", err)
		}
		if handlesResult["isError"] != false {
			t.Fatalf("runtime handles marcado como error: %#v", handlesResult)
		}
		handles, _ := handlesResult["structuredContent"].([]*db.RuntimeHandle)
		if len(handles) == 0 || handles[0] == nil || handles[0].ID != handle.ID {
			t.Fatalf("runtime handles inesperados: %#v", handlesResult["structuredContent"])
		}

		eventsResult, err := callMCPTool("orquesta.runtime.events.listar", map[string]any{
			"agente":   "Codex3",
			"proyecto": "orquestador",
			"kind":     "approval_request",
			"limit":    5,
		})
		if err != nil {
			t.Fatalf("runtime events MCP: %v", err)
		}
		if eventsResult["isError"] != false {
			t.Fatalf("runtime events marcado como error: %#v", eventsResult)
		}
		events, _ := eventsResult["structuredContent"].([]*db.RuntimeEvent)
		if len(events) == 0 || events[0] == nil || events[0].RuntimeID != runtimeInst.ID {
			t.Fatalf("runtime events inesperados: %#v", eventsResult["structuredContent"])
		}

		sendResult, err := callMCPTool("orquesta.runtime.mailbox.enviar", map[string]any{
			"from_agente":  "OpenClaw",
			"to_agente":    "Codex3",
			"proyecto":     "orquestador",
			"kind":         "governance_refresh",
			"payload_json": `{"accion":"revisar_runtime","motivo":"approval_request","task_id":` + int64ToString(tareaID) + `}`,
		})
		if err != nil {
			t.Fatalf("runtime mailbox enviar MCP: %v", err)
		}
		if sendResult["isError"] != false {
			t.Fatalf("runtime mailbox enviar marcado como error: %#v", sendResult)
		}
		sent, _ := sendResult["structuredContent"].(map[string]any)
		mailboxID, _ := sent["id"].(int64)
		if mailboxID <= 0 {
			t.Fatalf("mailbox id invalido: %#v", sendResult["structuredContent"])
		}

		mailboxPendingResult, err := callMCPTool("orquesta.runtime.mailbox.listar", map[string]any{
			"to_agente": "Codex3",
			"proyecto":  "orquestador",
			"estado":    "pendiente",
		})
		if err != nil {
			t.Fatalf("runtime mailbox listar pendiente MCP: %v", err)
		}
		if mailboxPendingResult["isError"] != false {
			t.Fatalf("runtime mailbox pendiente marcado como error: %#v", mailboxPendingResult)
		}
		pending, _ := mailboxPendingResult["structuredContent"].([]*db.RuntimeMailboxMessage)
		if !containsMailboxMessage(pending, mailboxID, "governance_refresh") {
			t.Fatalf("mailbox pendiente sin mensaje esperado: %#v", mailboxPendingResult["structuredContent"])
		}

		deliverResult, err := callMCPTool("orquesta.runtime.mailbox.entregar", map[string]any{"id": mailboxID})
		if err != nil {
			t.Fatalf("runtime mailbox entregar MCP: %v", err)
		}
		if deliverResult["isError"] != false {
			t.Fatalf("runtime mailbox entregar marcado como error: %#v", deliverResult)
		}

		consumeResult, err := callMCPTool("orquesta.runtime.mailbox.consumir", map[string]any{"id": mailboxID})
		if err != nil {
			t.Fatalf("runtime mailbox consumir MCP: %v", err)
		}
		if consumeResult["isError"] != false {
			t.Fatalf("runtime mailbox consumir marcado como error: %#v", consumeResult)
		}

		mailboxConsumedResult, err := callMCPTool("orquesta.runtime.mailbox.listar", map[string]any{
			"to_agente": "Codex3",
			"proyecto":  "orquestador",
			"estado":    "consumido",
		})
		if err != nil {
			t.Fatalf("runtime mailbox listar consumido MCP: %v", err)
		}
		if mailboxConsumedResult["isError"] != false {
			t.Fatalf("runtime mailbox consumido marcado como error: %#v", mailboxConsumedResult)
		}
		consumed, _ := mailboxConsumedResult["structuredContent"].([]*db.RuntimeMailboxMessage)
		if !containsMailboxMessage(consumed, mailboxID, "governance_refresh") {
			t.Fatalf("mailbox consumida sin mensaje esperado: %#v", mailboxConsumedResult["structuredContent"])
		}

		contents, err := readMCPResource("orquesta://supervision/OpenClaw/eventos?limit=5")
		if err != nil {
			t.Fatalf("readMCPResource supervision eventos: %v", err)
		}
		if len(contents) == 0 {
			t.Fatalf("recurso supervision eventos sin contenido")
		}
		resourceText, _ := contents[0]["text"].(string)
		for _, token := range []string{`"supervisor": "OpenClaw"`, `"normalized_events"`, "supervisor.approval_required"} {
			if !strings.Contains(resourceText, token) {
				t.Fatalf("recurso supervision eventos sin %q: %s", token, resourceText)
			}
		}
	})
}

func containsNormalizedEvent(items []openClawNormalizedEvent, event, agent, project string) bool {
	for _, item := range items {
		if strings.TrimSpace(item.NormalizedEvent) == strings.TrimSpace(event) &&
			strings.TrimSpace(item.Agent) == strings.TrimSpace(agent) &&
			strings.TrimSpace(item.Project) == strings.TrimSpace(project) {
			return true
		}
	}
	return false
}

func containsMailboxMessage(items []*db.RuntimeMailboxMessage, id int64, kind string) bool {
	for _, item := range items {
		if item != nil && item.ID == id && strings.TrimSpace(item.Kind) == strings.TrimSpace(kind) {
			return true
		}
	}
	return false
}

func int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
