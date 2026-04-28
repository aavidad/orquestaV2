package db

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"orquesta/coordinacion"
)

var (
	projectContextSummaryTTL               = 5 * time.Second
	projectContextOperacionFn              = GetProyectoOperacionPrepareLite
	projectContextActiveWorktreeSummaryFn  = getActiveWorktreeSummary
	projectContextActiveTaskSummariesFn    = getActiveTaskSummaries
	projectContextOpenProposalSummariesFn  = getOpenProposalSummaries
	projectContextSummaryCache             struct {
		mu    sync.Mutex
		items map[string]cachedProjectContextSummary
	}
)

type cachedProjectContextSummary struct {
	contexto map[string]any
	resumen  string
	expires  time.Time
}

func BuildProjectContextSummary(agente string, proyecto *Proyecto) (map[string]any, string) {
	if proyecto == nil {
		return nil, ""
	}
	cacheKey := projectContextSummaryCacheKey(agente, proyecto)
	if contexto, resumen, ok := getCachedProjectContextSummary(cacheKey); ok {
		return contexto, resumen
	}
	start := time.Now()
	projectContextDebugf("BuildProjectContextSummary start agente=%s proyecto=%s", strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug))
	defer func() {
		projectContextDebugf("BuildProjectContextSummary done agente=%s proyecto=%s duration=%s", strings.TrimSpace(agente), strings.TrimSpace(proyecto.Slug), time.Since(start).Round(time.Millisecond))
	}()
	proyecto = ProyectoPrepareLiteConRutaEfectiva(proyecto, strings.TrimSpace(agente))
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
	if op, err := projectContextOperacionFn(proyecto.ID); err == nil && op != nil {
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
	if worktree := projectContextActiveWorktreeSummaryFn(strings.TrimSpace(agente), proyecto); worktree != nil {
		contexto["worktree_activa"] = worktree
		if branch, _ := worktree["branch"].(string); strings.TrimSpace(branch) != "" {
			resumen = append(resumen, "Worktree activa en "+strings.TrimSpace(branch))
		}
	}
	projectContextDebugf("BuildProjectContextSummary step=worktree duration=%s", time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	if tareas := projectContextActiveTaskSummariesFn(strings.TrimSpace(agente), proyecto.ID); len(tareas) > 0 {
		contexto["tareas_activas"] = tareas
		resumen = append(resumen, fmt.Sprintf("%d tarea(s) activas del agente", len(tareas)))
	}
	projectContextDebugf("BuildProjectContextSummary step=tareas duration=%s", time.Since(stepStart).Round(time.Millisecond))

	stepStart = time.Now()
	if propuestas := projectContextOpenProposalSummariesFn(proyecto.ID); len(propuestas) > 0 {
		contexto["propuestas_abiertas"] = propuestas
		resumen = append(resumen, fmt.Sprintf("%d propuesta(s) abiertas", len(propuestas)))
	}
	projectContextDebugf("BuildProjectContextSummary step=propuestas duration=%s", time.Since(stepStart).Round(time.Millisecond))

	resumenStr := strings.Join(resumen, ". ")
	storeCachedProjectContextSummary(cacheKey, contexto, resumenStr)
	return cloneProjectContextSummaryMap(contexto), resumenStr
}

func projectContextSummaryCacheKey(agente string, proyecto *Proyecto) string {
	if proyecto == nil {
		return ""
	}
	return strings.TrimSpace(agente) + "|" + strconv.FormatInt(proyecto.ID, 10) + "|" + strings.TrimSpace(proyecto.Slug) + "|" + strings.TrimSpace(proyecto.RutaAbs)
}

func getCachedProjectContextSummary(key string) (map[string]any, string, bool) {
	if strings.TrimSpace(key) == "" || projectContextSummaryTTL <= 0 {
		return nil, "", false
	}
	projectContextSummaryCache.mu.Lock()
	defer projectContextSummaryCache.mu.Unlock()
	item, ok := projectContextSummaryCache.items[key]
	if !ok || time.Now().UTC().After(item.expires) {
		return nil, "", false
	}
	return cloneProjectContextSummaryMap(item.contexto), item.resumen, true
}

func storeCachedProjectContextSummary(key string, contexto map[string]any, resumen string) {
	if strings.TrimSpace(key) == "" || projectContextSummaryTTL <= 0 {
		return
	}
	projectContextSummaryCache.mu.Lock()
	defer projectContextSummaryCache.mu.Unlock()
	if projectContextSummaryCache.items == nil {
		projectContextSummaryCache.items = make(map[string]cachedProjectContextSummary)
	}
	projectContextSummaryCache.items[key] = cachedProjectContextSummary{
		contexto: cloneProjectContextSummaryMap(contexto),
		resumen:  strings.TrimSpace(resumen),
		expires:  time.Now().UTC().Add(projectContextSummaryTTL),
	}
}

func resetProjectContextSummaryCache() {
	projectContextSummaryCache.mu.Lock()
	defer projectContextSummaryCache.mu.Unlock()
	projectContextSummaryCache.items = nil
}

func cloneProjectContextSummaryMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	data, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return in
	}
	return out
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
	worktrees, err := ListarWorktreesCoordPrepareLite(coordinacion.WorktreeFilter{
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
	tareas, err := ListarTareasContextPrepareLite(strings.TrimSpace(agente), proyectoID, 8)
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
	propuestas, err := ListarPropuestasAbiertasPrepareLite(proyectoID, 4)
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
