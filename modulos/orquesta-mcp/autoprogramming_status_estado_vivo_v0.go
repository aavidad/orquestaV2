package orquestamcp

import (
	"context"
	"strings"
	"time"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const (
	mcpAutoprogrammingEstadoVivoEvidenceLimitV0 = 64
	mcpAutoprogrammingEstadoVivoHuerfanoV0      = time.Hour

	mcpAutoprogrammingActionEstadoVivoConflictoV0   = "estado_vivo_conflicto"
	mcpAutoprogrammingActionEstadoVivoHuerfanoV0    = "estado_vivo_huerfano"
	mcpAutoprogrammingActionEstadoVivoDesconocidoV0 = "estado_vivo_desconocido"

	mcpAutoprogrammingActionEstadoVivoReconcileGoalStateV0 = "estado_vivo_reconcile_goal_state"

	mcpAutoprogrammingEvidenceEstadoVivoConflictoV0   = "evidence-ref-autoprogramming-status-estado-vivo-conflicto"
	mcpAutoprogrammingEvidenceEstadoVivoHuerfanoV0    = "evidence-ref-autoprogramming-status-estado-vivo-huerfano"
	mcpAutoprogrammingEvidenceEstadoVivoDesconocidoV0 = "evidence-ref-autoprogramming-status-estado-vivo-desconocido"

	mcpAutoprogrammingEvidenceEstadoVivoReconcileGoalStateV0 = "evidence-ref-autoprogramming-status-estado-vivo-reconcile-goal-state"
)

func (executor MCPAutoprogrammingStatusToolExecutorV0) estadoVivoProjectionForAutoprogrammingStatusV0(
	ctx context.Context,
	input MCPAutoprogrammingStatusToolInputV0,
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) (*orquestaestadovivo.ProyeccionCicloVidaV0, bool, []MCPAutoprogrammingDiagnosticV0) {
	if executor.EstadoVivoSource == nil {
		return nil, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runRefs := runRefsForEstadoVivoMCPAutoprogrammingV0(input, queue, run, observedRuns...)
	evidencias := []orquestaestadovivo.EvidenciaEstadoV0{}
	if len(runRefs) == 0 {
		listed, err := executor.EstadoVivoSource.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
			Limit: mcpAutoprogrammingEstadoVivoEvidenceLimitV0,
		})
		if err != nil {
			return nil, false, []MCPAutoprogrammingDiagnosticV0{mcpAutoprogrammingDiagnosticV0(
				"estado_vivo_error",
				"estado_vivo",
				"fuente de estado vivo no disponible",
			)}
		}
		evidencias = append(evidencias, listed...)
	} else {
		perRunLimit := mcpAutoprogrammingEstadoVivoEvidenceLimitV0
		if len(runRefs) > 0 {
			perRunLimit = maxIntMCPAutoprogrammingEstadoVivoV0(1, mcpAutoprogrammingEstadoVivoEvidenceLimitV0/len(runRefs))
		}
		for _, runRef := range runRefs {
			listed, err := executor.EstadoVivoSource.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
				RunRef: runRef,
				Limit:  perRunLimit,
			})
			if err != nil {
				return nil, false, []MCPAutoprogrammingDiagnosticV0{mcpAutoprogrammingDiagnosticV0(
					"estado_vivo_error",
					"estado_vivo run:"+runRef,
					"fuente de estado vivo no disponible",
				)}
			}
			evidencias = append(evidencias, listed...)
		}
	}
	projection := orquestaestadovivo.ConstruirProyeccionCicloVidaV0(
		evidencias,
		nowForEstadoVivoMCPAutoprogrammingV0(input.OccurredAt),
		mcpAutoprogrammingEstadoVivoHuerfanoV0,
	)
	return &projection, true, nil
}

func runRefsForEstadoVivoMCPAutoprogrammingV0(
	input MCPAutoprogrammingStatusToolInputV0,
	queue *MCPRunQueuePriorityToolResultV0,
	run *MCPDirectorStatsToolResultV0,
	observedRuns ...*MCPDirectorStatsToolResultV0,
) []string {
	out := []string{input.RunRef}
	if queue != nil {
		for _, candidate := range queue.Ranked {
			out = append(out, candidate.RunRef)
		}
		for _, candidate := range queue.Terminal {
			out = append(out, candidate.RunRef)
		}
	}
	out = append(out, runRefFromDirectorStatsResultMCPAutoprogrammingV0(run))
	for _, observed := range observedRuns {
		out = append(out, runRefFromDirectorStatsResultMCPAutoprogrammingV0(observed))
	}
	return compactStringsMCPV0(out)
}

func runRefFromDirectorStatsResultMCPAutoprogrammingV0(
	run *MCPDirectorStatsToolResultV0,
) string {
	if run == nil {
		return ""
	}
	if run.Stats != nil && strings.TrimSpace(run.Stats.RunRef) != "" {
		return strings.TrimSpace(run.Stats.RunRef)
	}
	return strings.TrimSpace(run.RunRef)
}

func nowForEstadoVivoMCPAutoprogrammingV0(occurredAt string) time.Time {
	occurredAt = strings.TrimSpace(occurredAt)
	if occurredAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, occurredAt); err == nil {
			return parsed.UTC()
		}
		if parsed, err := time.Parse(time.RFC3339, occurredAt); err == nil {
			return parsed.UTC()
		}
	}
	return time.Now().UTC()
}

func applyEstadoVivoToQueueHealthMCPAutoprogrammingV0(
	health *MCPAutoprogrammingQueueHealthV0,
	projection *orquestaestadovivo.ProyeccionCicloVidaV0,
	queue *MCPRunQueuePriorityToolResultV0,
	goalStatesByRunRef map[string]orquestagoal.GoalWorkStateV0,
	goalRunMarkersByRunRef map[string]orquestagoal.GoalWorkRunMarkerV0,
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) *MCPAutoprogrammingQueueHealthV0 {
	if projection == nil || len(projection.Nodos) == 0 {
		return health
	}
	if health == nil {
		health = &MCPAutoprogrammingQueueHealthV0{}
	}
	seen := map[string]bool{}
	for _, node := range projection.Nodos {
		ref := refNodoEstadoVivoMCPAutoprogrammingV0(node)
		if ref != "" {
			if seen[ref] {
				continue
			}
			seen[ref] = true
		}
		if candidate, ok := queueCandidateForRunMCPAutoprogrammingHealthV0(queue, node.RunRef); ok {
			applyMCPAutoprogrammingHealthClassV0(
				health,
				classifyMCPAutoprogrammingQueuedRunHealthV0(candidate, goalStatesByRunRef, goalRunMarkersByRunRef),
				-1,
			)
		}
		if observed := observedByRunRef[strings.TrimSpace(node.RunRef)]; observed != nil && observed.Stats != nil {
			for _, class := range healthClassesFromRunStatsMCPAutoprogrammingV0(*observed.Stats) {
				applyMCPAutoprogrammingHealthClassV0(health, class, -1)
			}
		}
		applyMCPAutoprogrammingHealthClassV0(health, healthClassFromEstadoVivoMCPAutoprogrammingV0(node.Fase), 1)
	}
	if len(seen) > health.ObservedRuns {
		health.ObservedRuns = len(seen)
	}
	return health
}

func healthClassFromEstadoVivoMCPAutoprogrammingV0(
	fase orquestaestadovivo.FaseCicloVidaV0,
) string {
	switch fase {
	case orquestaestadovivo.FaseSolicitadoV0, orquestaestadovivo.FaseLanzadoV0:
		return mcpAutoprogrammingHealthQueuedV0
	case orquestaestadovivo.FaseProcesoVivoV0:
		return mcpAutoprogrammingHealthRunningLiveV0
	case orquestaestadovivo.FaseEntregadoParcialV0,
		orquestaestadovivo.FaseBloqueadoV0,
		orquestaestadovivo.FaseTerminalReworkV0,
		orquestaestadovivo.FaseConflictoV0:
		return mcpAutoprogrammingHealthBlockedV0
	case orquestaestadovivo.FaseTerminalAceptadoV0:
		return mcpAutoprogrammingHealthCompletedV0
	case orquestaestadovivo.FaseHuerfanoV0:
		return mcpAutoprogrammingHealthRunningStaleNoProcessV0
	default:
		return mcpAutoprogrammingHealthUnclassifiedV0
	}
}

func healthClassesFromRunStatsMCPAutoprogrammingV0(
	stats orquestacionnucleoapp.DirectorRunStatsV0,
) []string {
	classes := make([]string, 0, 3)
	liveness := mcpAutoprogrammingRunLivenessV0(stats)
	if stats.Counts.AgentsFailed > 0 {
		classes = append(classes, mcpAutoprogrammingHealthFailedV0)
	}
	if stats.Counts.AgentsLost > 0 {
		classes = append(classes, mcpAutoprogrammingHealthLostV0)
	}
	if stats.Closure.Blocked {
		classes = append(classes, mcpAutoprogrammingHealthBlockedV0)
	}
	if liveness.Class == orquestaruncoordinator.RunLivenessClassRunningLiveV0 {
		return append(classes, mcpAutoprogrammingHealthRunningLiveV0)
	}
	if stats.Counts.AgentsInFlight > 0 {
		switch liveness.Class {
		case orquestaruncoordinator.RunLivenessClassRunningWithoutRecentStatsV0:
			return append(classes, mcpAutoprogrammingHealthRunningWithoutRecentStatsV0)
		case orquestaruncoordinator.RunLivenessClassRunningStaleNoProcessV0:
			return append(classes, mcpAutoprogrammingHealthRunningStaleNoProcessV0)
		default:
			return append(classes, mcpAutoprogrammingHealthRunningWithoutRecentStatsV0)
		}
	}
	if stats.Closure.Closed || mcpAutoprogrammingRunStatusCompletedV0(stats.Status) {
		return append(classes, mcpAutoprogrammingHealthCompletedV0)
	}
	if stats.Counts.AgentsFailed == 0 && stats.Counts.AgentsLost == 0 && !stats.Closure.Blocked {
		classes = append(classes, mcpAutoprogrammingHealthUnclassifiedV0)
	}
	return classes
}

func staleRunningFromEstadoVivoMCPAutoprogrammingV0(
	projection *orquestaestadovivo.ProyeccionCicloVidaV0,
) []MCPAutoprogrammingActionableRunV0 {
	if projection == nil {
		return nil
	}
	out := make([]MCPAutoprogrammingActionableRunV0, 0)
	for _, node := range projection.Nodos {
		action, ok := actionableRunFromEstadoVivoMCPAutoprogrammingV0(node)
		if ok {
			out = append(out, action)
		}
	}
	return out
}

func actionableRunFromEstadoVivoMCPAutoprogrammingV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) (MCPAutoprogrammingActionableRunV0, bool) {
	action, ok := actionableRunFromEstadoVivoNodeSinVeredictoMCPAutoprogrammingV0(node)
	if !ok {
		return action, false
	}
	action.CausalVerdict = string(node.Veredicto.Clase)
	action.CausalReasonCode = strings.TrimSpace(node.Veredicto.ReasonCode)
	return action, true
}

// mcpAutoprogrammingEstadoVivoRunningContradichoV0 detecta un veredicto causal
// que contradice un state running (terminal durable o proceso muerto con state
// stale) mientras la fase proyectada aun no publica ese terminal. La
// superficie no publica running en ese caso; pide reconciliar.
func mcpAutoprogrammingEstadoVivoRunningContradichoV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) bool {
	if node.Veredicto.EstadoPersistido == "" {
		return false
	}
	switch node.Fase {
	case orquestaestadovivo.FaseSolicitadoV0, orquestaestadovivo.FaseLanzadoV0,
		orquestaestadovivo.FaseDesconocidoV0, orquestaestadovivo.FaseHuerfanoV0:
	default:
		return false
	}
	switch node.Veredicto.Clase {
	case orquestaestadovivo.VeredictoTerminalByArtifactV0, orquestaestadovivo.VeredictoProcessDeadStateStaleV0:
		return true
	default:
		return false
	}
}

func actionableRunFromEstadoVivoNodeSinVeredictoMCPAutoprogrammingV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) (MCPAutoprogrammingActionableRunV0, bool) {
	if mcpAutoprogrammingEstadoVivoRunningContradichoV0(node) {
		return MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionEstadoVivoReconcileGoalStateV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(node.RunRef),
			GoalRef:           strings.TrimSpace(node.GoalRef),
			ExternalGoalRef:   strings.TrimSpace(node.ExternalGoalRef),
			Status:            string(node.Fase),
			Reason:            "veredicto causal contradice running persistido: reconciliar estado del goal",
			RecommendedAction: mcpObserveAppDirectorGoalActionReconcileGoalStateV0,
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceEstadoVivoReconcileGoalStateV0},
				evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...,
			)),
		}, true
	}
	switch node.Fase {
	case orquestaestadovivo.FaseEntregadoParcialV0:
		return MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionPartialArtifactsWrittenV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(node.RunRef),
			GoalRef:           strings.TrimSpace(node.GoalRef),
			ExternalGoalRef:   strings.TrimSpace(node.ExternalGoalRef),
			Status:            string(node.Fase),
			Reason:            "estado vivo con entrega parcial pendiente de revision y cierre",
			RecommendedAction: MCPGoalFirstReviewPartialArtifactsActionV0,
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidencePartialArtifactsWrittenV0},
				evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...,
			)),
		}, true
	case orquestaestadovivo.FaseConflictoV0:
		return MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionEstadoVivoConflictoV0,
			Severity:          "blocked",
			RunRef:            strings.TrimSpace(node.RunRef),
			GoalRef:           strings.TrimSpace(node.GoalRef),
			ExternalGoalRef:   strings.TrimSpace(node.ExternalGoalRef),
			Status:            string(node.Fase),
			Reason:            "estado vivo incompatible: hay evidencia terminal y proceso vivo para el mismo run",
			RecommendedAction: "reconcile_estado_vivo",
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceEstadoVivoConflictoV0},
				evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...,
			)),
		}, true
	case orquestaestadovivo.FaseHuerfanoV0:
		return MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionEstadoVivoHuerfanoV0,
			Severity:          "warning",
			RunRef:            strings.TrimSpace(node.RunRef),
			GoalRef:           strings.TrimSpace(node.GoalRef),
			ExternalGoalRef:   strings.TrimSpace(node.ExternalGoalRef),
			Status:            string(node.Fase),
			Reason:            "estado vivo huerfano: marcador sin proceso ni terminal reciente",
			RecommendedAction: "observe_or_reconcile_estado_vivo",
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceEstadoVivoHuerfanoV0},
				evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...,
			)),
		}, true
	case orquestaestadovivo.FaseDesconocidoV0:
		return MCPAutoprogrammingActionableRunV0{
			Code:              mcpAutoprogrammingActionEstadoVivoDesconocidoV0,
			Severity:          "warning",
			RunRef:            strings.TrimSpace(node.RunRef),
			GoalRef:           strings.TrimSpace(node.GoalRef),
			ExternalGoalRef:   strings.TrimSpace(node.ExternalGoalRef),
			Status:            string(node.Fase),
			Reason:            "estado vivo desconocido: falta evidencia suficiente para cierre verde",
			RecommendedAction: "observe_estado_vivo",
			EvidenceRefs: compactStringsMCPV0(append(
				[]string{mcpAutoprogrammingEvidenceEstadoVivoDesconocidoV0},
				evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...,
			)),
		}, true
	default:
		return MCPAutoprogrammingActionableRunV0{}, false
	}
}

func applyCausalVerdictToStatusResultMCPAutoprogrammingV0(
	result MCPAutoprogrammingStatusToolResultV0,
	projection *orquestaestadovivo.ProyeccionCicloVidaV0,
	runRef string,
) MCPAutoprogrammingStatusToolResultV0 {
	runRef = strings.TrimSpace(runRef)
	if projection == nil || runRef == "" {
		return result
	}
	node, ok := mcpDirectorStatsEstadoVivoNodeForRunV0(*projection, runRef)
	if !ok {
		return result
	}
	result.CausalVerdict = string(node.Veredicto.Clase)
	result.CausalReasonCode = strings.TrimSpace(node.Veredicto.ReasonCode)
	return result
}

func diagnosticsFromEstadoVivoMCPAutoprogrammingV0(
	projection *orquestaestadovivo.ProyeccionCicloVidaV0,
) []MCPAutoprogrammingDiagnosticV0 {
	if projection == nil {
		return nil
	}
	out := make([]MCPAutoprogrammingDiagnosticV0, 0)
	for _, node := range projection.Nodos {
		switch node.Fase {
		case orquestaestadovivo.FaseConflictoV0:
			out = append(out, MCPAutoprogrammingDiagnosticV0{
				Code:         mcpAutoprogrammingActionEstadoVivoConflictoV0,
				Scope:        scopeEstadoVivoMCPAutoprogrammingV0(node),
				Message:      "estado vivo conflictivo: no publicar verde hasta reconciliar evidencias",
				EvidenceRefs: compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoConflictoV0}, evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...)),
			})
		case orquestaestadovivo.FaseHuerfanoV0:
			out = append(out, MCPAutoprogrammingDiagnosticV0{
				Code:         mcpAutoprogrammingActionEstadoVivoHuerfanoV0,
				Scope:        scopeEstadoVivoMCPAutoprogrammingV0(node),
				Message:      "estado vivo huerfano: requiere observacion o reconciliacion",
				EvidenceRefs: compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoHuerfanoV0}, evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...)),
			})
		case orquestaestadovivo.FaseDesconocidoV0:
			out = append(out, MCPAutoprogrammingDiagnosticV0{
				Code:         mcpAutoprogrammingActionEstadoVivoDesconocidoV0,
				Scope:        scopeEstadoVivoMCPAutoprogrammingV0(node),
				Message:      "estado vivo desconocido: evidencia insuficiente",
				EvidenceRefs: compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoDesconocidoV0}, evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)...)),
			})
		}
	}
	return out
}

func evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) []string {
	out := make([]string, 0)
	for _, evidencia := range node.Evidencias {
		out = append(out, evidencia.EvidenceRefs...)
	}
	return compactStringsMCPV0(out)
}

func refNodoEstadoVivoMCPAutoprogrammingV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) string {
	return firstNonEmptyMCPV0(node.RunRef, node.GoalRef, node.ExternalGoalRef)
}

func scopeEstadoVivoMCPAutoprogrammingV0(
	node orquestaestadovivo.NodoCicloVidaV0,
) string {
	if runRef := strings.TrimSpace(node.RunRef); runRef != "" {
		return "estado_vivo run:" + runRef
	}
	if goalRef := strings.TrimSpace(node.GoalRef); goalRef != "" {
		return "estado_vivo goal:" + goalRef
	}
	return "estado_vivo"
}

func maxIntMCPAutoprogrammingEstadoVivoV0(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
