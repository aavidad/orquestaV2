package cmd

import (
	"testing"

	"orquesta/db"
)

func TestMCPToolsAgentesOverviewCanonicalExponeTaskDebt(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("CodexOverviewDebt", "programador"); err != nil {
			t.Fatalf("registrando CodexOverviewDebt: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		tareaA, err := db.CrearTarea(&db.Tarea{
			Titulo:      "frente A",
			Descripcion: "multitarea canonical",
			ProyectoID:  &proyectoID,
			Modulo:      "cmd",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea A: %v", err)
		}
		tareaB, err := db.CrearTarea(&db.Tarea{
			Titulo:      "frente B",
			Descripcion: "worker gap canonical",
			ProyectoID:  &proyectoID,
			Modulo:      "cmd",
			Prioridad:   db.PrioridadAlta,
			CreadoPor:   "OpenClaw",
		})
		if err != nil {
			t.Fatalf("crear tarea B: %v", err)
		}
		if err := db.TomarTarea(tareaA, "CodexOverviewDebt"); err != nil {
			t.Fatalf("tomar tarea A: %v", err)
		}
		if err := db.TomarTarea(tareaB, "CodexOverviewDebt"); err != nil {
			t.Fatalf("tomar tarea B: %v", err)
		}
		if err := db.IniciarTarea(tareaA, "CodexOverviewDebt"); err != nil {
			t.Fatalf("iniciar tarea A: %v", err)
		}
		if err := db.IniciarTarea(tareaB, "CodexOverviewDebt"); err != nil {
			t.Fatalf("iniciar tarea B: %v", err)
		}

		result, err := callMCPTool("orquesta.agentes.overview", map[string]any{
			"agente": "CodexOverviewDebt",
			"schema": "canonical",
		})
		if err != nil {
			t.Fatalf("agentes overview canonical MCP: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("agentes overview canonical marcado como error: %#v", result)
		}
		resp, _ := result["structuredContent"].(apiAgenteOverviewResponse)
		if resp.Agent == nil || resp.Agent.Entity == nil {
			t.Fatalf("agent canónico inesperado: %#v", result["structuredContent"])
		}
		if resp.Agent.OpenTasks != 2 {
			t.Fatalf("open_tasks inesperadas: %+v", resp.Agent)
		}
		if resp.Agent.Entity.MultitaskDebt != 1 || resp.Agent.Entity.WorkerGapCount != 2 {
			t.Fatalf("task debt canónica inesperada: %+v", resp.Agent.Entity)
		}
		if resp.Agent.Entity.TaskWorkerHint == "" || resp.Agent.Entity.TaskWorkerHint != "2 tarea(s) abiertas sin worker vivo u operativo; este agente puede no aparecer en /api/status.agentesActivos" {
			t.Fatalf("task worker hint canónico inesperado: %+v", resp.Agent.Entity)
		}
	})
}
