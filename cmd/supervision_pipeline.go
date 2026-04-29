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
	return listSupervisorPipelineSnapshot(supervisor, proyectoSlug, limit)
}

func buildSupervisorPipelineReadSnapshot(supervisor, proyectoSlug string, limit int) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	snapshot, err := listSupervisorPipelineSnapshot(supervisor, proyectoSlug, limit)
	if err != nil {
		return nil, err
	}
	if items, _ := snapshot["pipelines"].([]*db.SupervisorPipelineState); len(items) > 0 {
		go func(supervisor, proyectoSlug string) {
			_, _ = reconcileSupervisorPipelineState(supervisor, proyectoSlug)
		}(supervisor, proyectoSlug)
		return snapshot, nil
	}
	return buildSupervisorPipelineSnapshot(supervisor, proyectoSlug, limit)
}

func buildSupervisorPipelineSnapshotFromInputs(supervisor, proyectoSlug string, limit int, status apiStatusResponse, gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	if _, err := reconcileSupervisorPipelineStateFromInputs(supervisor, strings.TrimSpace(proyectoSlug), status, gates, signals, merges, conflicts, mailboxPendiente); err != nil {
		return nil, err
	}
	return listSupervisorPipelineSnapshot(supervisor, proyectoSlug, limit)
}

func listSupervisorPipelineSnapshot(supervisor, proyectoSlug string, limit int) (map[string]any, error) {
	items, err := db.ListarSupervisorPipelineStates(db.FiltroSupervisorPipelineStates{
		Supervisor:   strings.TrimSpace(supervisor),
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}
	autonomyHighlights, criticalProjectRisk := supervisorPipelineAutonomySnapshotContext(items)
	return map[string]any{
		"supervisor":            supervisor,
		"proyecto":              strings.TrimSpace(proyectoSlug),
		"generated_at":          time.Now().UTC().Format(time.RFC3339),
		"pipelines":             items,
		"followup":              buildOpenClawPipelineFollowupFromPipelineItems(items),
		"autonomy_highlights":   autonomyHighlights,
		"critical_project_risk": criticalProjectRisk,
	}, nil
}

func buildSupervisorPipelineOverview(supervisor string) (string, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorPipelineReadSnapshot(supervisor, "", 20)
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
	conflicts := listarSupervisorModuleConflictsFromTasks(status.TareasActivas)
	mailboxPendiente, err := buildOpenClawPendingMailbox(status.Agentes)
	if err != nil {
		return nil, err
	}
	return reconcileSupervisorPipelineStateFromInputs(supervisor, proyectoSlug, status, openGates, signals, merges, conflicts, mailboxPendiente)
}

func reconcileSupervisorPipelineStateFromInputs(supervisor, proyectoSlug string, status apiStatusResponse, openGates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict, mailboxPendiente []apiOpenClawMailboxLite) (*db.SupervisorPipelineState, error) {
	recommended := buildSupervisorRecommendedActions(openGates, signals, merges, conflicts)
	recommended = append(recommended, buildSupervisorOperationalActions(status, mailboxPendiente)...)
	sortSupervisorRecommendedActions(recommended)
	retenidas := tareasRetenidasPorCuota(status.TareasActivas, status.Agentes, status.AgentesQuotaBlocked)
	item := deriveSupervisorPipelineState(supervisor, proyectoSlug, status, recommended, retenidas)
	return db.UpsertSupervisorPipelineState(item)
}

func deriveSupervisorPipelineState(supervisor, proyectoSlug string, status apiStatusResponse, recommended []supervisorRecommendedAction, retenidas []tareaLite) db.UpsertSupervisorPipelineStateInput {
	currentPhase := "idle"
	pipelineStatus := "paused"
	var currentTaskID *int64
	var currentGateID *int64
	var currentMergeID *int64
	latestSidecarFollowup, _ := latestPipelineSidecarFollowup(supervisor, proyectoSlug)
	workersActivos := visibleNonSupervisorAgents(status.AgentesActivos)
	workersTrabajando := visibleNonSupervisorAgents(status.AgentesTrabajando)
	autonomySurface, autonomyHighlights, criticalProjectRisk := normalizeStatusAutonomyPayload(status.AutonomySurface, status.AutonomyHighlights, status.CriticalProjectRisk)
	integrationRiskScore, integrationRiskLabel, integrationHighlights, integrationRiskHigh := supervisorPipelineIntegrationRiskContext(autonomyHighlights, criticalProjectRisk)
	metadata := map[string]any{
		"connected_workers": len(workersActivos),
		"working_workers":   len(workersTrabajando),
		"retained_by_quota": len(retenidas),
		"recommended_count": len(recommended),
		"generated_by":      "orquesta.supervision.pipeline.derived",
	}
	artifacts := map[string]any{
		"connected_agents":  supervisorAgentNames(workersActivos),
		"working_agents":    supervisorAgentNames(workersTrabajando),
		"retained_task_ids": tareaLiteIDs(retenidas),
		"visible_task_ids":  tareaLiteIDs(status.TareasActivas),
		"action_queue":      recommended,
	}
	if len(autonomyHighlights) > 0 {
		metadata["autonomy_highlights"] = autonomyHighlights
		artifacts["autonomy_highlights"] = autonomyHighlights
	}
	if criticalProjectRisk != nil {
		metadata["critical_project_risk"] = criticalProjectRisk
		artifacts["critical_project_risk"] = criticalProjectRisk
	}
	if integrationRiskScore > 0 {
		metadata["integration_risk_score"] = integrationRiskScore
	}
	if integrationRiskLabel != "" {
		metadata["integration_risk"] = integrationRiskLabel
	}
	if len(integrationHighlights) > 0 {
		metadata["integration_highlights"] = integrationHighlights
		artifacts["integration_highlights"] = integrationHighlights
	}
	if autonomySurface != nil {
		artifacts["autonomy_surface"] = autonomySurface
	}
	if status.Autonomia.Count > 0 {
		metadata["autonomy_event_count"] = status.Autonomia.Count
	}
	if len(status.Autonomia.ByKind) > 0 {
		metadata["autonomy_by_kind"] = status.Autonomia.ByKind
	}
	if status.Autonomia.LastAt != nil && !status.Autonomia.LastAt.IsZero() {
		metadata["autonomy_last_at"] = status.Autonomia.LastAt.UTC()
	}
	if len(status.Autonomia.Recent) > 0 {
		artifacts["recent_autonomy_events"] = status.Autonomia.Recent
	}
	if latestSidecarFollowup != nil {
		metadata["latest_parallel_sidecar_followup"] = latestSidecarFollowup
		artifacts["latest_parallel_sidecar_followup"] = latestSidecarFollowup
	}

	if len(recommended) > 0 {
		action := recommended[0]
		metadata["next_action"] = action
		currentPhase, pipelineStatus, currentTaskID, currentGateID, currentMergeID = phaseFromRecommendedAction(action)
	}
	if currentPhase == "idle" {
		switch {
		case len(retenidas) > 0 && len(workersActivos) == 0:
			currentPhase = "blocked_by_quota"
			pipelineStatus = "blocked"
			currentTaskID = int64Ptr(retenidas[0].ID)
		case len(workersTrabajando) > 0:
			currentPhase = "coordinar_workers"
			pipelineStatus = "active"
			currentTaskID = firstVisibleTaskID(status.TareasActivas, retenidas)
		case len(workersActivos) > 0 && len(status.TareasActivas) > 0:
			currentPhase = "dispatch"
			pipelineStatus = "active"
			currentTaskID = firstVisibleTaskID(status.TareasActivas, retenidas)
		case len(workersActivos) > 0:
			currentPhase = "idle"
			pipelineStatus = "active"
		case len(retenidas) > 0:
			currentPhase = "blocked_by_quota"
			pipelineStatus = "blocked"
			currentTaskID = int64Ptr(retenidas[0].ID)
		}
		if integrationRiskHigh {
			switch {
			case len(workersActivos) == 0 && pipelineStatus == "paused":
				pipelineStatus = "blocked"
				metadata["risk_watch_mode"] = "blocked"
			case len(workersActivos) > 0 && currentPhase == "idle":
				currentPhase = "coordinar_workers"
				pipelineStatus = "active"
				metadata["risk_watch_mode"] = "active"
			}
		}
	}
	if shouldRefineSupervisorPipelineWithSidecarFollowup(currentPhase, currentGateID, currentMergeID, latestSidecarFollowup) {
		if phase := strings.TrimSpace(stringSupervisorSubagente(latestSidecarFollowup["phase"])); phase != "" {
			currentPhase = phase
			pipelineStatus = "active"
		}
		if taskID := int64SupervisorSubagente(latestSidecarFollowup["task_id"]); taskID > 0 {
			currentTaskID = &taskID
		}
		if mergeID := int64SupervisorSubagente(latestSidecarFollowup["git_merge_id"]); mergeID > 0 {
			currentMergeID = &mergeID
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

func supervisorPipelineIntegrationRiskContext(autonomyHighlights []string, criticalProjectRisk *workspaceAutonomyProjectSummary) (int, string, []string, bool) {
	score := 0
	label := ""
	highlights := []string(nil)
	if criticalProjectRisk != nil {
		if criticalProjectRisk.Blocking > score {
			score = criticalProjectRisk.Blocking
		}
		if len(criticalProjectRisk.Highlights) > 0 {
			highlights = compactProjectControlIntegrationHighlights(criticalProjectRisk.Highlights)
			for _, item := range criticalProjectRisk.Highlights {
				value := strings.TrimSpace(item)
				if strings.HasPrefix(value, "riesgo=") {
					label = strings.TrimSpace(strings.TrimPrefix(value, "riesgo="))
					break
				}
			}
		}
	}
	if score <= 0 {
		for _, item := range autonomyHighlights {
			value := strings.TrimSpace(item)
			if !strings.HasPrefix(value, "integracion_bloqueada=") {
				continue
			}
			parsed, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(value, "integracion_bloqueada=")))
			if err == nil && parsed > score {
				score = parsed
			}
		}
	}
	if label == "" && score > 0 {
		label = workspaceIntegrationRiskLabel(score)
	}
	return score, label, highlights, score >= 5
}

func supervisorPipelineAutonomySnapshotContext(items []*db.SupervisorPipelineState) ([]string, *workspaceAutonomyProjectSummary) {
	for _, item := range items {
		if item == nil || strings.TrimSpace(item.PipelineName) != "supervisor-loop" {
			continue
		}
		meta := mapFromJSON(item.MetadataJSON)
		artifacts := mapFromJSON(item.ArtifactsJSON)
		highlights := stringSliceFromAny(meta["autonomy_highlights"])
		if len(highlights) == 0 {
			highlights = stringSliceFromAny(artifacts["autonomy_highlights"])
		}
		var risk *workspaceAutonomyProjectSummary
		if parsed, ok := parseWorkspaceAutonomyProjectSummarySnapshotValue(meta["critical_project_risk"]); ok {
			risk = parsed
		} else if parsed, ok := parseWorkspaceAutonomyProjectSummarySnapshotValue(artifacts["critical_project_risk"]); ok {
			risk = parsed
		}
		return highlights, risk
	}
	return nil, nil
}

func latestPipelineSidecarFollowup(supervisor, proyectoSlug string) (map[string]any, error) {
	items, err := db.ListarSupervisorSubagents(db.FiltroSupervisorSubagents{
		Supervisor:   strings.TrimSpace(supervisor),
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Status:       "completed",
		Limit:        20,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		metadata := metadataSupervisorSubagente(item)
		if len(metadata) == 0 {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(stringSupervisorSubagente(metadata["source"])), "pipeline_local_parallel") {
			continue
		}
		if !sidecarPipelineFollowupYaDespachado(metadata) {
			continue
		}
		followup := map[string]any{
			"subagent_id": item.ID,
			"thread_id":   strings.TrimSpace(item.ThreadID),
			"updated_at":  item.UpdatedAt.UTC().Format(time.RFC3339),
		}
		for _, key := range []string{
			"task_id",
			"task_title",
			"slice_index",
			"slice_total",
			"write_set_slice",
			"pipeline_parent_followup_phase",
			"pipeline_parent_followup_action",
			"pipeline_parent_followup_git_merge_id",
			"pipeline_parent_followup_dispatched_at",
		} {
			if value, ok := metadata[key]; ok {
				followup[key] = value
			}
		}
		if phase := strings.TrimSpace(stringSupervisorSubagente(metadata["pipeline_parent_followup_phase"])); phase != "" {
			followup["phase"] = phase
		}
		if action := strings.TrimSpace(stringSupervisorSubagente(metadata["pipeline_parent_followup_action"])); action != "" {
			followup["action"] = action
		}
		if mergeID := int64SupervisorSubagente(metadata["pipeline_parent_followup_git_merge_id"]); mergeID > 0 {
			followup["git_merge_id"] = mergeID
		}
		return followup, nil
	}
	return nil, nil
}

func shouldRefineSupervisorPipelineWithSidecarFollowup(currentPhase string, currentGateID, currentMergeID *int64, followup map[string]any) bool {
	if len(followup) == 0 || currentGateID != nil || currentMergeID != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(currentPhase)) {
	case "", "idle", "coordinar_workers", "dispatch":
		return strings.TrimSpace(stringSupervisorSubagente(followup["phase"])) != ""
	default:
		return false
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
	case "quota_hold":
		phase = "blocked_by_quota"
		status = "blocked"
		currentTaskID = parseSupervisorActionTargetID("tarea:", action.Target)
	case "dispatch":
		phase = "dispatch"
		currentTaskID = parseSupervisorActionTargetID("tarea:", action.Target)
	case "autonomy_event":
		currentTaskID = parseSupervisorActionTargetID("tarea:", action.Target)
		switch strings.TrimSpace(action.Action) {
		case "resolver_followup_bloqueado":
			phase = "autonomy_followup"
		case "inspeccionar_handoff_fallido":
			phase = "autonomy_recovery"
		case "revisar_pausa_externa":
			phase = "autonomy_recovery"
		case "seguir_repair_helper":
			phase = "autonomy_recovery"
		case "seguir_reinicio_runtime":
			phase = "autonomy_recovery"
		case "seguir_reasignacion":
			phase = "autonomy_followup"
		case "verificar_handoff_consolidado":
			phase = "autonomy_followup"
		default:
			phase = "coordinar_workers"
		}
	default:
		phase = "coordinar_workers"
	}
	if strings.TrimSpace(action.Priority) == "alta" && status == "active" {
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

func resolveSupervisorActionSubagentID(target string) (int64, error) {
	id := parseSupervisorActionTargetID("subagente:", target)
	if id == nil || *id <= 0 {
		return 0, fmt.Errorf("target de subagente inválido: %s", strings.TrimSpace(target))
	}
	return *id, nil
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
