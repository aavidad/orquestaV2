package orquestamcp

import (
	"context"
	"strings"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	mcpObserveAppDirectorGoalEstadoVivoEvidenceLimitV0  = 32
	mcpObserveAppDirectorGoalActionReconcileGoalStateV0 = "reconcile_goal_state"
)

// WithMCPObserveAppDirectorGoalCausalVerdictV0 permite a composiciones que no
// pasan por el executor MCP (p.ej. el stack) aplicar el mismo contrato de
// veredicto causal sobre un resultado ya construido.
func WithMCPObserveAppDirectorGoalCausalVerdictV0(
	ctx context.Context,
	source orquestaestadovivo.FuenteEvidenciaEstadoPortV0,
	input MCPObserveAppDirectorGoalToolInputV0,
	result MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	return MCPObserveAppDirectorGoalToolExecutorV0{EstadoVivoSource: source}.withCausalVerdictV0(ctx, input, result)
}

func (executor MCPObserveAppDirectorGoalToolExecutorV0) withCausalVerdictV0(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
	result MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	if executor.EstadoVivoSource == nil {
		return result
	}
	runRef := strings.TrimSpace(firstNonEmptyMCPV0(result.RunRef, input.RunRef))
	if runRef == "" {
		return result
	}
	if ctx == nil {
		ctx = context.Background()
	}
	evidencias, err := executor.EstadoVivoSource.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: runRef,
		Limit:  mcpObserveAppDirectorGoalEstadoVivoEvidenceLimitV0,
	})
	if err != nil || len(evidencias) == 0 {
		return result
	}
	return applyMCPObserveAppDirectorGoalCausalVerdictV0(result, orquestaestadovivo.DerivarVeredictoCausalV0(evidencias))
}

func applyMCPObserveAppDirectorGoalCausalVerdictV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	veredicto orquestaestadovivo.VeredictoCausalV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	result.CausalVerdict = string(veredicto.Clase)
	result.CausalReasonCode = strings.TrimSpace(veredicto.ReasonCode)
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, veredicto.EvidenceRefs...))
	if strings.TrimSpace(result.GoalStatus) != orquestagoal.GoalStatusRunningV0 {
		return result
	}
	switch veredicto.Clase {
	case orquestaestadovivo.VeredictoTerminalByArtifactV0, orquestaestadovivo.VeredictoProcessDeadStateStaleV0:
	default:
		return result
	}
	result.GoalStatus = string(veredicto.Clase)
	if mcpEstadoVivoStatusPublicaRunningV0(result.RunStatus) {
		result.RunStatus = string(veredicto.Clase)
	}
	result.RecommendedAction = mcpObserveAppDirectorGoalActionReconcileGoalStateV0
	return result
}

func (executor MCPObserveAppDirectorGoalToolExecutorV0) withEstadoVivoSnapshotV0(
	ctx context.Context,
	input MCPObserveAppDirectorGoalToolInputV0,
	result MCPObserveAppDirectorGoalToolResultV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	if executor.EstadoVivoSource == nil {
		return result
	}
	runRef := strings.TrimSpace(firstNonEmptyMCPV0(result.RunRef, input.RunRef))
	if runRef == "" {
		return result
	}
	if ctx == nil {
		ctx = context.Background()
	}
	evidencias, err := executor.EstadoVivoSource.ListarEvidenciasEstadoV0(ctx, orquestaestadovivo.FiltroEvidenciaEstadoV0{
		RunRef: runRef,
		Limit:  mcpObserveAppDirectorGoalEstadoVivoEvidenceLimitV0,
	})
	if err != nil {
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpDirectorStatsEstadoVivoErrorV0,
			"estado_vivo",
			"fuente de estado vivo no disponible",
			[]string{mcpDirectorStatsEvidenceEstadoVivoErrorV0},
			"observe_estado_vivo",
		)
	}
	projection := orquestaestadovivo.ConstruirProyeccionCicloVidaV0(
		evidencias,
		nowForEstadoVivoMCPAutoprogrammingV0(input.OccurredAt),
		mcpAutoprogrammingEstadoVivoHuerfanoV0,
	)
	node, ok := mcpDirectorStatsEstadoVivoNodeForRunV0(projection, runRef)
	if !ok {
		return result
	}
	return applyMCPObserveAppDirectorGoalEstadoVivoNodeV0(result, node)
}

func applyMCPObserveAppDirectorGoalEstadoVivoNodeV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	node orquestaestadovivo.NodoCicloVidaV0,
) MCPObserveAppDirectorGoalToolResultV0 {
	result.CausalVerdict = string(node.Veredicto.Clase)
	result.CausalReasonCode = strings.TrimSpace(node.Veredicto.ReasonCode)
	evidenceRefs := evidenceRefsFromEstadoVivoNodeMCPAutoprogrammingV0(node)
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, evidenceRefs...))
	if node.Veredicto.Clase == orquestaestadovivo.VeredictoDivergentNeedsRepairV0 {
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpAutoprogrammingActionEstadoVivoConflictoV0,
			"estado_vivo.veredicto."+node.Veredicto.ReasonCode,
			"veredicto causal divergente; requiere reconciliacion por identidad y evidencias",
			compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoConflictoV0}, evidenceRefs...)),
			"reconcile_estado_vivo",
		)
	}
	switch node.Fase {
	case orquestaestadovivo.FaseTerminalAceptadoV0:
		result.RunStatus = "closed"
		result.ClosureStatus = orquestagoal.GoalStatusAcceptedV0
		result.ClosureAccepted = true
		result.ClosureNeedsRework = false
		result.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(result)
		return result
	case orquestaestadovivo.FaseProcesoVivoV0:
		result.RunStatus = firstNonEmptyMCPV0(result.RunStatus, mcpDirectorStatsEstadoVivoProcesoVivoV0)
		if result.ClosureAccepted || strings.TrimSpace(result.ClosureStatus) == orquestagoal.GoalStatusAcceptedV0 {
			result.ClosureStatus = ""
			result.ClosureAccepted = false
			result.ClosureNeedsRework = false
		}
		result.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(result)
		return result
	case orquestaestadovivo.FaseEntregadoParcialV0:
		result = EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0(result, MCPDirectorGoalMaterializedRefsV0{
			EvidenceRefs: compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidencePartialArtifactsWrittenV0}, evidenceRefs...)),
			IssueCodes:   []string{MCPGoalFirstPartialArtifactsWrittenV0},
		})
		result.ClosureStatus = orquestagoal.GoalStatusBlockedV0
		result.ClosureAccepted = false
		result.ClosureNeedsRework = true
		result.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(result)
		return result
	case orquestaestadovivo.FaseConflictoV0:
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpAutoprogrammingActionEstadoVivoConflictoV0,
			"estado_vivo.conflicto",
			"estado vivo incompatible: hay evidencia terminal y proceso vivo para el mismo run",
			compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoConflictoV0}, evidenceRefs...)),
			"reconcile_estado_vivo",
		)
	case orquestaestadovivo.FaseHuerfanoV0:
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpAutoprogrammingActionEstadoVivoHuerfanoV0,
			"estado_vivo.huerfano",
			"estado vivo huerfano: marcador sin proceso ni terminal reciente",
			compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoHuerfanoV0}, evidenceRefs...)),
			"observe_or_reconcile_estado_vivo",
		)
	case orquestaestadovivo.FaseDesconocidoV0:
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpAutoprogrammingActionEstadoVivoDesconocidoV0,
			"estado_vivo.veredicto."+node.Veredicto.ReasonCode,
			"veredicto causal indeterminado; no se confirma running ni cierre",
			compactStringsMCPV0(append([]string{mcpAutoprogrammingEvidenceEstadoVivoDesconocidoV0}, evidenceRefs...)),
			"observe_estado_vivo",
		)
	case orquestaestadovivo.FaseBloqueadoV0:
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpDirectorStatsEstadoVivoBloqueadoV0,
			"estado_vivo.bloqueado",
			"estado vivo bloqueado",
			evidenceRefs,
			"replan",
		)
	case orquestaestadovivo.FaseTerminalReworkV0:
		return mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
			result,
			mcpDirectorStatsEstadoVivoTerminalReworkV0,
			"estado_vivo.terminal_rework",
			"estado vivo terminal requiere rework",
			evidenceRefs,
			"replan",
		)
	case orquestaestadovivo.FaseSolicitadoV0:
		result.RunStatus = firstNonEmptyMCPV0(result.RunStatus, mcpDirectorStatsEstadoVivoSolicitadoV0)
		result.ClosureAccepted = false
		if strings.TrimSpace(result.ClosureStatus) == orquestagoal.GoalStatusAcceptedV0 {
			result.ClosureStatus = ""
		}
		result.RecommendedAction = "observe_later"
		return result
	case orquestaestadovivo.FaseLanzadoV0:
		result.RunStatus = firstNonEmptyMCPV0(result.RunStatus, mcpDirectorStatsEstadoVivoLanzadoV0)
		result.ClosureAccepted = false
		if strings.TrimSpace(result.ClosureStatus) == orquestagoal.GoalStatusAcceptedV0 {
			result.ClosureStatus = ""
		}
		result.RecommendedAction = "observe_later"
		return result
	default:
		result.RecommendedAction = mcpObserveAppDirectorGoalRecommendedActionV0(result)
		return result
	}
}

func mcpObserveAppDirectorGoalBlockWithEstadoVivoIssueV0(
	result MCPObserveAppDirectorGoalToolResultV0,
	code string,
	field string,
	message string,
	evidenceRefs []string,
	recommendedAction string,
) MCPObserveAppDirectorGoalToolResultV0 {
	if mcpEstadoVivoStatusPublicaRunningV0(result.RunStatus) {
		result.RunStatus = strings.TrimSpace(code)
	}
	if mcpEstadoVivoStatusPublicaRunningV0(result.GoalStatus) {
		result.GoalStatus = strings.TrimSpace(code)
	}
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, evidenceRefs...))
	if !mcpObserveAppDirectorGoalHasClosureIssueCodeV0(result.ClosureIssues, code) {
		result.ClosureIssues = append(result.ClosureIssues, MCPValidationIssueV0{
			Code:    strings.TrimSpace(code),
			Field:   strings.TrimSpace(field),
			Message: strings.TrimSpace(message),
		})
	}
	result.ClosureStatus = orquestagoal.GoalStatusBlockedV0
	result.ClosureAccepted = false
	result.ClosureNeedsRework = true
	result.RecommendedAction = firstNonEmptyMCPV0(recommendedAction, mcpObserveAppDirectorGoalRecommendedActionV0(result))
	return result
}

func mcpEstadoVivoStatusPublicaRunningV0(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "active", "in_progress", "proceso_vivo", mcpDirectorStatsEstadoVivoProcesoVivoV0:
		return true
	default:
		return false
	}
}
