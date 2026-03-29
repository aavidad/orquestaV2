package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func BuildProjectContextSummary(agente string, proyecto *Proyecto) (map[string]any, string) {
	if proyecto == nil {
		return nil, ""
	}
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

	if worktree := getActiveWorktreeSummary(strings.TrimSpace(agente), proyecto.ID); worktree != nil {
		contexto["worktree_activa"] = worktree
		if branch, _ := worktree["branch"].(string); strings.TrimSpace(branch) != "" {
			resumen = append(resumen, "Worktree activa en "+strings.TrimSpace(branch))
		}
	}

	if tareas := getActiveTaskSummaries(strings.TrimSpace(agente), proyecto.ID); len(tareas) > 0 {
		contexto["tareas_activas"] = tareas
		resumen = append(resumen, fmt.Sprintf("%d tarea(s) activas del agente", len(tareas)))
	}

	if propuestas := getOpenProposalSummaries(proyecto.ID); len(propuestas) > 0 {
		contexto["propuestas_abiertas"] = propuestas
		resumen = append(resumen, fmt.Sprintf("%d propuesta(s) abiertas", len(propuestas)))
	}

	return contexto, strings.Join(resumen, ". ")
}

func AppendProjectContextPayload(prev string, contexto map[string]any) string {
	if len(contexto) == 0 {
		return strings.TrimSpace(prev)
	}
	return MergeResumePayloadEnvelope(prev, map[string]any{
		"project_context": contexto,
	})
}

func getActiveWorktreeSummary(agente string, proyectoID int64) map[string]any {
	if strings.TrimSpace(agente) == "" || proyectoID == 0 {
		return nil
	}
	row := DB.QueryRow(`
		SELECT id, nombre, ruta_abs, branch, base_ref, motivo
		FROM worktrees
		WHERE proyecto_id = ? AND agente = ? AND estado = 'activa'
		ORDER BY id DESC
		LIMIT 1`,
		proyectoID, strings.TrimSpace(agente),
	)
	var (
		id      int64
		nombre  sql.NullString
		ruta    sql.NullString
		branch  sql.NullString
		baseRef sql.NullString
		motivo  sql.NullString
	)
	if err := row.Scan(&id, &nombre, &ruta, &branch, &baseRef, &motivo); err != nil {
		return nil
	}
	if !WorktreeActivaCoherente(proyectoID, ruta.String) {
		return nil
	}
	return map[string]any{
		"id":       id,
		"nombre":   strings.TrimSpace(nombre.String),
		"ruta":     strings.TrimSpace(ruta.String),
		"branch":   strings.TrimSpace(branch.String),
		"base_ref": strings.TrimSpace(baseRef.String),
		"motivo":   strings.TrimSpace(motivo.String),
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
