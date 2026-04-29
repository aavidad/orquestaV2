package cmd

import (
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestMCPToolsResourcesAndPromptsIncluyenOpenClawOperator(t *testing.T) {
	withTempOrquestaDB(t, func() {
		tools := listMCPTools()
		if !mcpToolListed(tools, "orquesta.openclaw.operator") {
			t.Fatal("tool MCP openclaw operator no registrada")
		}

		resources, err := listMCPResources()
		if err != nil {
			t.Fatalf("listMCPResources: %v", err)
		}
		foundResource := false
		for _, item := range resources {
			if item.URI == "orquesta://openclaw/operator" {
				foundResource = true
				break
			}
		}
		if !foundResource {
			t.Fatal("resource MCP openclaw operator no listada")
		}

		foundTemplate := false
		for _, item := range mcpResourceTemplates() {
			if item.URITemplate == "orquesta://openclaw/operator{?supervisor}" {
				foundTemplate = true
				break
			}
		}
		if !foundTemplate {
			t.Fatal("template MCP openclaw operator no listada")
		}

		foundPrompt := false
		for _, item := range listMCPPrompts() {
			if item.Name == "orquesta.openclaw.operator" {
				foundPrompt = true
				break
			}
		}
		if !foundPrompt {
			t.Fatal("prompt MCP openclaw operator no registrado")
		}
	})
}

func TestMCPOpenClawOperatorExponeSnapshotCanonico(t *testing.T) {
	prevFetcher := controlPlaneOperationalInfoFetcher
	defer func() {
		controlPlaneOperationalInfoFetcher = prevFetcher
	}()

	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex4", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		tareaID, err := db.CrearTarea(&db.Tarea{
			Titulo:     "Cerrar paridad MCP OpenClaw",
			ProyectoID: &proyectoID,
			Estado:     db.TareaEnProgreso,
			Prioridad:  db.PrioridadAlta,
			CreadoPor:  "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea: %v", err)
		}
		snapshot := apiStatusResponse{
			Generado: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
			Agentes: []*db.Agente{
				{Nombre: "Codex4"},
			},
			AgentesActivos: []*db.Agente{
				{Nombre: "Codex4"},
			},
			AgentesTrabajando: []*db.Agente{
				{Nombre: "Codex4"},
			},
			TareasPorEstado: map[string]int{
				string(db.TareaEnProgreso): 1,
			},
			TareasActivas: []tareaLite{{
				ID:        tareaID,
				Titulo:    "Cerrar paridad MCP OpenClaw",
				Estado:    db.TareaEnProgreso,
				Agente:    "Codex4",
				Prioridad: db.PrioridadAlta,
			}},
			TareasEnProgreso: []tareaLite{{
				ID:        tareaID,
				Titulo:    "Cerrar paridad MCP OpenClaw",
				Estado:    db.TareaEnProgreso,
				Agente:    "Codex4",
				Prioridad: db.PrioridadAlta,
			}},
			Autonomia: autonomiaResumen{ByKind: map[string]int{}},
		}
		storeStatusSnapshot(snapshot, time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))
		controlPlaneOperationalInfoFetcher = func() (serverOperationalInfo, error) {
			return serverOperationalInfo{
				State:            "ready",
				Operational:      true,
				Reason:           "",
				ActiveAgents:     1,
				WorkingAgents:    1,
				ConnectedWorkers: 1,
				WorkingWorkers:   1,
				TasksInProgress:  1,
				Generated:        "2026-04-29T12:00:05Z",
			}, nil
		}

		result, err := callMCPTool("orquesta.openclaw.operator", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("tool openclaw operator: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("tool openclaw operator marcada como error: %#v", result)
		}
		payload, _ := result["structuredContent"].(map[string]any)
		if payload == nil {
			t.Fatalf("structuredContent inválido: %#v", result["structuredContent"])
		}
		operational, _ := payload["server_operational"].(serverOperationalInfo)
		if operational.State != "ready" || !operational.Operational {
			t.Fatalf("server_operational inesperado: %#v", payload["server_operational"])
		}
		status, _ := payload["status"].(apiOpenClawStatusLite)
		if len(status.AgentesActivos) != 1 || len(status.TareasActivas) != 1 {
			t.Fatalf("status openclaw incompleto: %#v", payload["status"])
		}
		if _, ok := payload["pipeline_state"]; !ok {
			t.Fatalf("snapshot sin pipeline_state: %#v", payload)
		}
		if _, ok := payload["session_candidates"]; !ok {
			t.Fatalf("snapshot sin session_candidates: %#v", payload)
		}

		contents, err := readMCPResource("orquesta://openclaw/operator?supervisor=OpenClaw")
		if err != nil {
			t.Fatalf("resource openclaw operator: %v", err)
		}
		if len(contents) == 0 {
			t.Fatal("resource openclaw operator sin contenido")
		}
		resourceText, _ := contents[0]["text"].(string)
		for _, token := range []string{`"server_operational"`, `"status"`, `"pipeline_state"`} {
			if !strings.Contains(resourceText, token) {
				t.Fatalf("resource openclaw operator sin %q: %s", token, resourceText)
			}
		}

		prompt, err := getMCPPrompt("orquesta.openclaw.operator", map[string]any{"supervisor": "OpenClaw"})
		if err != nil {
			t.Fatalf("prompt openclaw operator: %v", err)
		}
		messages, _ := prompt["messages"].([]mcpPromptMessage)
		if len(messages) == 0 {
			t.Fatalf("prompt openclaw operator sin mensajes: %#v", prompt)
		}
		contentMap, _ := messages[0].Content.(map[string]any)
		content, _ := contentMap["text"].(string)
		for _, token := range []string{"Snapshot operador OpenClaw: OpenClaw", "Operación:", "queue_safe=0"} {
			if !strings.Contains(content, token) {
				t.Fatalf("prompt openclaw operator sin %q: %s", token, content)
			}
		}
	})
}
