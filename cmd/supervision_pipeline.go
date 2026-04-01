package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"orquesta/db"
)

func buildSupervisorPipelineSnapshot(supervisor, proyectoSlug string, limit int) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	if _, err := reconcileSupervisorPipelineState(supervisor, strings.TrimSpace(proyectoSlug)); err != nil {
		return nil, err
	}
	items, err := db.ListarSupervisorPipelineStates(db.FiltroSupervisorPipelineStates{
		Supervisor:   strings.TrimSpace(supervisor),
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"supervisor":   supervisor,
		"proyecto":     strings.TrimSpace(proyectoSlug),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"pipelines":    items,
	}, nil
}

func buildSupervisorPipelineOverview(supervisor string) (string, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorPipelineSnapshot(supervisor, "", 20)
	if err != nil {
		return "", err
	}
	items, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState)
	var b strings.Builder
	fmt.Fprintf(&b, "# Pipeline del supervisor: %s\n\n", supervisor)
	if len(items) == 0 {
		b.WriteString("- Sin pipeline explícita registrada.\n")
		return b.String(), nil
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		fmt.Fprintf(&b, "- %s", strings.TrimSpace(item.PipelineName))
		if strings.TrimSpace(item.ProyectoSlug) != "" {
			fmt.Fprintf(&b, " proyecto=%s", strings.TrimSpace(item.ProyectoSlug))
		}
		if strings.TrimSpace(item.CurrentPhase) != "" {
			fmt.Fprintf(&b, " fase=%s", strings.TrimSpace(item.CurrentPhase))
		}
		fmt.Fprintf(&b, " estado=%s\n", strings.TrimSpace(item.Status))
	}
	return strings.TrimSpace(b.String()) + "\n", nil
}

func reconcileSupervisorPipelineState(supervisor, proyectoSlug string) (*db.SupervisorPipelineState, error) {
	supervisor = resolveSupervisorName(supervisor)
	proyectoSlug = strings.TrimSpace(proyectoSlug)

	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{Limit: 20})
	if err != nil {
		return nil, err
	}
	openGates := make([]*db.ReviewGate, 0, len(gates))
	for _, gate := range gates {
		if gate == nil || gate.Estado == db.ReviewGateAprobado {
			continue
		}
		openGates = append(openGates, gate)
	}
	signals, err := listarSignalsRevisionSupervisor(12)
	if err != nil {
		return nil, err
	}
	merges, err := listarMergesRevisionSupervisor(12)
	if err != nil {
		return nil, err
	}
	conflicts, err := listarSupervisorModuleConflicts()
	if err != nil {
		return nil, err
	}
	recommended := buildSupervisorRecommendedActions(openGates, signals, merges, conflicts)
	retenidas := tareasRetenidasPorCuota(status.TareasActivas, status.Agentes)
	item := deriveSupervisorPipelineState(supervisor, proyectoSlug, status, recommended, retenidas)
	return db.UpsertSupervisorPipelineState(item)
}

func deriveSupervisorPipelineState(supervisor, proyectoSlug string, status apiStatusResponse, recommended []supervisorRecommendedAction, retenidas []tareaLite) db.UpsertSupervisorPipelineStateInput {
	currentPhase := "idle"
	pipelineStatus := "paused"
	var currentTaskID *int64
	var currentGateID *int64
	var currentMergeID *int64
	metadata := map[string]any{
		"connected_workers": len(status.AgentesActivos),
		"working_workers":   len(status.AgentesTrabajando),
		"retained_by_quota": len(retenidas),
		"recommended_count": len(recommended),
		"generated_by":      "orquesta.supervision.pipeline.derived",
	}
	artifacts := map[string]any{
		"connected_agents":  supervisorAgentNames(status.AgentesActivos),
		"working_agents":    supervisorAgentNames(status.AgentesTrabajando),
		"retained_task_ids": tareaLiteIDs(retenidas),
		"visible_task_ids":  tareaLiteIDs(status.TareasActivas),
		"action_queue":      recommended,
	}

	if len(recommended) > 0 {
		action := recommended[0]
		metadata["next_action"] = action
		currentPhase, pipelineStatus, currentTaskID, currentGateID, currentMergeID = phaseFromRecommendedAction(action)
	}
	if currentPhase == "idle" {
		switch {
		case len(retenidas) > 0 && len(status.AgentesActivos) == 0:
			currentPhase = "blocked_by_quota"
			pipelineStatus = "blocked"
			currentTaskID = int64Ptr(retenidas[0].ID)
		case len(status.AgentesTrabajando) > 0:
			currentPhase = "coordinar_workers"
			pipelineStatus = "active"
			currentTaskID = firstVisibleTaskID(status.TareasActivas, retenidas)
		case len(status.AgentesActivos) > 0 && len(status.TareasActivas) > 0:
			currentPhase = "dispatch"
			pipelineStatus = "active"
			currentTaskID = firstVisibleTaskID(status.TareasActivas, retenidas)
		case len(status.AgentesActivos) > 0:
			currentPhase = "idle"
			pipelineStatus = "active"
		case len(retenidas) > 0:
			currentPhase = "blocked_by_quota"
			pipelineStatus = "blocked"
			currentTaskID = int64Ptr(retenidas[0].ID)
		}
	}

	metadataJSON, _ := json.Marshal(metadata)
	artifactsJSON, _ := json.Marshal(artifacts)
	return db.UpsertSupervisorPipelineStateInput{
		Supervisor:     supervisor,
		ProyectoSlug:   proyectoSlug,
		PipelineName:   "supervisor-loop",
		CurrentPhase:   currentPhase,
		Status:         pipelineStatus,
		CurrentTaskID:  currentTaskID,
		CurrentGateID:  currentGateID,
		CurrentMergeID: currentMergeID,
		ArtifactsJSON:  string(artifactsJSON),
		MetadataJSON:   string(metadataJSON),
	}
}

func phaseFromRecommendedAction(action supervisorRecommendedAction) (string, string, *int64, *int64, *int64) {
	var currentTaskID *int64
	var currentGateID *int64
	var currentMergeID *int64
	phase := "review"
	status := "active"
	switch strings.TrimSpace(action.Kind) {
	case "review_gate":
		phase = "review"
		currentGateID = parseSupervisorActionTargetID("review_gate:", action.Target)
	case "merge":
		phase = "merge"
		currentMergeID = parseSupervisorActionTargetID("merge:", action.Target)
	case "module_conflict":
		phase = "resolver_conflicto"
		currentTaskID = parseSupervisorActionTargetID("tarea:", action.Target)
	case "signal":
		phase = "arbitrar_revision"
	default:
		phase = "coordinar_workers"
	}
	if strings.TrimSpace(action.Priority) == "alta" {
		status = "active"
	}
	return phase, status, currentTaskID, currentGateID, currentMergeID
}

func parseSupervisorActionTargetID(prefix, target string) *int64 {
	target = strings.TrimSpace(target)
	if prefix != "" {
		if !strings.HasPrefix(target, prefix) {
			return nil
		}
		target = strings.TrimPrefix(target, prefix)
	}
	id, err := strconv.ParseInt(strings.TrimSpace(target), 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func firstVisibleTaskID(tareas []tareaLite, retenidas []tareaLite) *int64 {
	retenidasIDs := make(map[int64]struct{}, len(retenidas))
	for _, tarea := range retenidas {
		retenidasIDs[tarea.ID] = struct{}{}
	}
	for _, tarea := range tareas {
		if _, blocked := retenidasIDs[tarea.ID]; blocked {
			continue
		}
		return int64Ptr(tarea.ID)
	}
	if len(retenidas) > 0 {
		return int64Ptr(retenidas[0].ID)
	}
	return nil
}

func tareaLiteIDs(tareas []tareaLite) []int64 {
	out := make([]int64, 0, len(tareas))
	for _, tarea := range tareas {
		if tarea.ID > 0 {
			out = append(out, tarea.ID)
		}
	}
	return out
}

func supervisorAgentNames(agentes []*db.Agente) []string {
	out := make([]string, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre != "" {
			out = append(out, nombre)
		}
	}
	return out
}

func int64Ptr(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}
