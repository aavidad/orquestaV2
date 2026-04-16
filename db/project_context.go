package db

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"orquesta/coordinacion"
)

func BuildProjectContextSummary(agente string, proyecto *Proyecto) (map[string]any, string) {
	if proyecto == nil {
		return nil, ""
	}
	start := time.Now()
	projectContextDebugf("BuildProjectContextSummary start agente=%s proyecto=%s", strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug))
	defer func() {
		projectContextDebugf("BuildProjectContextSummary done agente=%s proyecto=%s duration=%s", strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug), time.Since(start).Round(time.Millisecond))
	}()
	proyecto = ProyectoConRutaEfectiva(proyecto, "")
	contexto := map[string]any{
		"proyecto": map[string]any{
			"id":   proyecto.ID,
			"slug": strings.TrimSpace(proyecto.Slug),
			"ruta": strings.TrimSpace(proyecto.RutaAbs),
			"tipo": strings.TrimSpace(string(proyecto.Tipo)),
		},
	}
	resumen := make([]string, 0, 4)

	stepStart := time.Now()
	if op, err := GetProyectoOperacion(proyecto.ID); err == nil && op != nil {
		contexto["operacion"] = map[string]any{
			"estado":            strings.TrimSpace(string(op.EstadoOperativo)),
			"motivo":            strings.TrimSpace(op.Motivo),
			"objetivo_pct":      op.ObjetivoPct,
			"prioridad":         op.Prioridad,
			"resume_automatico": op.ResumeAutomatico,
			"min_agentes":       op.MinAgentes,
			"max_agentes":       op.MaxAgentes,
		}
		if op.EstadoOperativo != ProyectoOperativoActivo {
			resumen = append(resumen, "Estado operativo: "+string(op.EstadoOperativo))
		}
	}
	projectContextDebugf("BuildProjectContextSummary step=operacion duration=%s", time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	if worktree := getActiveWorktreeSummary(strings.TrimSpace(agente), proyecto); worktree != nil {
		contexto["worktree_activa"] = worktree
		if branch, _ := worktree["branch"].(string); strings.TrimSpace(branch) != "" {
			resumen = append(resumen, "Worktree activa en "+strings.TrimSpace(branch))
		}
	}
	projectContextDebugf("BuildProjectContextSummary step=worktree duration=%s", time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	if tareas := getActiveTaskSummaries(strings.TrimSpace(agente), proyecto.ID); len(tareas) > 0 {
		contexto["tareas_activas"] = tareas
		resumen = append(resumen, fmt.Sprintf("%d tarea(s) activas del agente", len(tareas)))
	}
	projectContextDebugf("BuildProjectContextSummary step=tareas duration=%s", time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	if propuestas := getOpenProposalSummaries(proyecto.ID); len(propuestas) > 0 {
		contexto["propuestas_abiertas"] = propuestas
		resumen = append(resumen, fmt.Sprintf("%d propuesta(s) abiertas", len(propuestas)))
	}
	projectContextDebugf("BuildProjectContextSummary step=propuestas duration=%s", time.Since(stepStart).Round(time.Millisecond))

	return contexto, strings.Join(resumen, ". ")
}

func projectContextDebugf(format string, args ...any) {
	if !preparePromptDebugEnabled() {
		return
	}
	log.Printf("orquesta[prepare-context] "+format, args...)
}

func preparePromptDebugEnabled() bool {
	for _, key := range []string{"ORQUESTA_DEBUG_PREPARE", "ORQUESTA_DEBUG"} {
		value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
		switch value {
		case "1", "true", "yes", "on", "si", "sí":
			return true
		}
	}
	return false
}

func AppendProjectContextPayload(prev string, contexto map[string]any) string {
	if len(contexto) == 0 {
		return strings.TrimSpace(prev)
	}
	return MergeResumePayloadEnvelope(prev, map[string]any{
		"project_context": contexto,
	})
}

func getActiveWorktreeSummary(agente string, proyecto *Proyecto) map[string]any {
	if strings.TrimSpace(agente) == "" || proyecto == nil || proyecto.ID == 0 {
		return nil
	}
	agente = strings.TrimSpace(agente)
	estado := coordinacion.WorktreeActive
	worktrees, err := ListarWorktreesCoord(coordinacion.WorktreeFilter{
		ProjectID: &proyecto.ID,
		Agent:     &agente,
		State:     &estado,
	})
	if err != nil || len(worktrees) == 0 || worktrees[0] == nil {
		return nil
	}
	worktree := worktrees[0]
	return map[string]any{
		"id":       worktree.ID,
		"nombre":   strings.TrimSpace(worktree.Name),
		"ruta":     strings.TrimSpace(worktree.Path),
		"branch":   strings.TrimSpace(worktree.Branch),
		"base_ref": strings.TrimSpace(worktree.BaseRef),
		"motivo":   strings.TrimSpace(worktree.Reason),
	}
}

func getActiveTaskSummaries(agente string, proyectoID int64) []map[string]any {
	tareas, err := ListarTareas(FiltroTareas{
		Agente:     strPtrRuntime(strings.TrimSpace(agente)),
		ProyectoID: &proyectoID,
	})
	if err != nil {
		return nil
	}
	items := make([]map[string]any, 0, 4)
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case TareaAsignada, TareaEnProgreso, TareaBloqueada:
		default:
			continue
		}
		items = append(items, map[string]any{
			"id":     tarea.ID,
			"titulo": strings.TrimSpace(tarea.Titulo),
			"estado": strings.TrimSpace(string(tarea.Estado)),
			"modulo": strings.TrimSpace(tarea.Modulo),
		})
		if len(items) == 4 {
			break
		}
	}
	return items
}

func getOpenProposalSummaries(proyectoID int64) []map[string]any {
	estado := PropuestaAbierta
	propuestas, err := ListarPropuestas(&estado, &proyectoID)
	if err != nil {
		return nil
	}
	items := make([]map[string]any, 0, 4)
	for _, propuesta := range propuestas {
		if propuesta == nil {
			continue
		}
		items = append(items, map[string]any{
			"id":     propuesta.ID,
			"codigo": strings.TrimSpace(propuesta.Codigo),
			"titulo": strings.TrimSpace(propuesta.Titulo),
			"estado": strings.TrimSpace(string(propuesta.Estado)),
		})
		if len(items) == 4 {
			break
		}
	}
	return items
}
