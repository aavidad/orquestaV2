package orquestamcp

import (
	"context"
	"errors"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	mcpAutoprogrammingActionMaterialProgressContinueV0 = "material_progress_continue"
	mcpAutoprogrammingActionMaterialProgressWarningV0  = "material_progress_warning"
	mcpAutoprogrammingActionMaterialProgressReplanV0   = "material_progress_replan_required"
	mcpAutoprogrammingActionMaterialProgressHardStopV0 = "material_progress_hard_stop_required"
)

type mcpMaterialProgressStateV0 struct {
	State          orquestaautoprogramming.MaterialProgressStateV0
	Valid          bool
	LegacyFallback bool
	DiagnosticCode string
}

type mcpMaterialProgressStatusProjectionV0 map[string]mcpMaterialProgressStateV0

func mcpMaterialProgressProjectionForStatusV0(
	ctx context.Context,
	reader orquestaautoprogramming.MaterialProgressStateReaderPortV0,
	goalStates []orquestagoal.GoalWorkStateV0,
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) mcpMaterialProgressStatusProjectionV0 {
	projection := mcpMaterialProgressStatusProjectionV0{}
	for _, goalState := range goalStates {
		runRef := strings.TrimSpace(goalState.RunRef)
		goalRef := strings.TrimSpace(goalState.GoalRef)
		if runRef == "" || goalRef == "" {
			continue
		}
		projection[runRef] = mcpMaterialProgressStateForRunGoalV0(ctx, reader, runRef, goalRef)
	}
	for runRef, observed := range observedByRunRef {
		if _, exists := projection[runRef]; exists || observed == nil || observed.Goal == nil {
			continue
		}
		goalRef := strings.TrimSpace(observed.Goal.GoalRef)
		if strings.TrimSpace(runRef) == "" || goalRef == "" {
			continue
		}
		projection[runRef] = mcpMaterialProgressStateForRunGoalV0(ctx, reader, runRef, goalRef)
	}
	return projection
}

func mcpMaterialProgressStateForRunGoalV0(
	ctx context.Context,
	reader orquestaautoprogramming.MaterialProgressStateReaderPortV0,
	runRef string,
	goalRef string,
) mcpMaterialProgressStateV0 {
	runRef = strings.TrimSpace(runRef)
	goalRef = strings.TrimSpace(goalRef)
	if reader == nil {
		return mcpMaterialProgressStateV0{LegacyFallback: true, DiagnosticCode: "material_progress_state_reader_unbound"}
	}
	state, err := reader.LoadMaterialProgressStateV0(ctx, runRef, goalRef)
	if err != nil {
		if mcpMaterialProgressStateNotFoundV0(err) {
			return mcpMaterialProgressStateV0{LegacyFallback: true, DiagnosticCode: "material_progress_state_not_found"}
		}
		return mcpMaterialProgressStateV0{DiagnosticCode: "material_progress_state_unavailable"}
	}
	validation := orquestaautoprogramming.ValidateMaterialProgressStateV0(state)
	if !validation.Accepted || validation.State.RunRef != runRef || validation.State.GoalRef != goalRef {
		return mcpMaterialProgressStateV0{DiagnosticCode: "material_progress_state_invalid"}
	}
	return mcpMaterialProgressStateV0{State: validation.State, Valid: true}
}

func mcpMaterialProgressStateNotFoundV0(err error) bool {
	var notFound orquestaautoprogramming.MaterialProgressStateNotFoundErrorV0
	return errors.As(err, &notFound)
}

func (projection mcpMaterialProgressStatusProjectionV0) statusDiagnostics() []MCPAutoprogrammingDiagnosticV0 {
	codes := map[string]bool{}
	for _, progress := range projection {
		if progress.DiagnosticCode == "" {
			continue
		}
		codes[progress.DiagnosticCode] = true
	}
	diagnostics := make([]MCPAutoprogrammingDiagnosticV0, 0, len(codes))
	for _, code := range []string{
		"material_progress_state_reader_unbound",
		"material_progress_state_not_found",
		"material_progress_state_unavailable",
		"material_progress_state_invalid",
	} {
		if !codes[code] {
			continue
		}
		diagnostics = append(diagnostics, mcpAutoprogrammingDiagnosticV0(
			code,
			"material_progress",
			"material progress projection unavailable",
		))
	}
	return diagnostics
}

func (projection mcpMaterialProgressStatusProjectionV0) legacySuppressedRuns() map[string]bool {
	decisions := make(map[string]bool, len(projection))
	for runRef, progress := range projection {
		if !progress.LegacyFallback {
			decisions[runRef] = true
		}
	}
	return decisions
}

func (projection mcpMaterialProgressStatusProjectionV0) actions(
	observedByRunRef map[string]*MCPDirectorStatsToolResultV0,
) []MCPAutoprogrammingActionableRunV0 {
	actions := make([]MCPAutoprogrammingActionableRunV0, 0)
	for runRef, progress := range projection {
		if !progress.Valid {
			continue
		}
		action := mcpMaterialProgressActionV0(progress.State)
		if observed := observedByRunRef[runRef]; observed != nil {
			action.RunStatus = mcpAutoprogrammingObservedRunStatusV0(observedByRunRef, runRef)
			if observed.Goal != nil {
				action.ExternalGoalRef = strings.TrimSpace(observed.Goal.ExternalGoalRef)
				action.GoalStatus = strings.TrimSpace(observed.Goal.Status)
			}
		}
		actions = append(actions, action)
	}
	return actions
}

func mcpMaterialProgressActionV0(
	state orquestaautoprogramming.MaterialProgressStateV0,
) MCPAutoprogrammingActionableRunV0 {
	decision := state.LastDecision
	action := MCPAutoprogrammingActionableRunV0{
		RunRef:       state.RunRef,
		GoalRef:      state.GoalRef,
		TokensUsed:   decision.TokensWithoutMaterial,
		ArtifactRefs: []string{state.LastCheckpointRef},
		EvidenceRefs: compactStringsMCPV0(append(
			append(state.EvidenceRefs, state.LastCheckpoint.EvidenceRefs...),
			state.LastActionIdempotencyKey,
		)),
		Reason:            "persisted material progress decision",
		RecommendedAction: "continue",
		Severity:          "info",
	}
	switch decision.Action {
	case orquestaautoprogramming.MaterialProgressActionWarningV0:
		action.Code = mcpAutoprogrammingActionMaterialProgressWarningV0
		action.Severity = "warning"
		action.RecommendedAction = "observe_goal_backend"
	case orquestaautoprogramming.MaterialProgressActionReplanRequiredV0:
		action.Code = mcpAutoprogrammingActionMaterialProgressReplanV0
		action.Severity = "blocked"
		action.RecommendedAction = "replan_narrow_context"
	case orquestaautoprogramming.MaterialProgressActionHardStopRequiredV0:
		action.Code = mcpAutoprogrammingActionMaterialProgressHardStopV0
		action.Severity = "blocked"
		action.RecommendedAction = "stop_safely"
	default:
		action.Code = mcpAutoprogrammingActionMaterialProgressContinueV0
	}
	return action
}

func (progress mcpMaterialProgressStateV0) forcedTerminalReason() (string, string, bool) {
	if !progress.Valid {
		return "", "", false
	}
	switch progress.State.LastDecision.Action {
	case orquestaautoprogramming.MaterialProgressActionReplanRequiredV0,
		orquestaautoprogramming.MaterialProgressActionHardStopRequiredV0:
		return string(progress.State.LastDecision.Action), progress.State.LastCheckpointRef, true
	default:
		return "", "", false
	}
}

func (progress mcpMaterialProgressStateV0) runControlDiagnostics(runRef string) []MCPRunControlDiagnosticV0 {
	if progress.DiagnosticCode == "" {
		return nil
	}
	return []MCPRunControlDiagnosticV0{{
		Code:    progress.DiagnosticCode,
		Scope:   "run:" + strings.TrimSpace(runRef),
		Message: "material progress projection unavailable",
	}}
}

func filterMCPAutoprogrammingLegacyMaterialProgressActionsV0(
	actions []MCPAutoprogrammingActionableRunV0,
	projectedRuns map[string]bool,
) []MCPAutoprogrammingActionableRunV0 {
	out := make([]MCPAutoprogrammingActionableRunV0, 0, len(actions))
	for _, action := range actions {
		if projectedRuns[strings.TrimSpace(action.RunRef)] && mcpAutoprogrammingLegacyMaterialProgressActionV0(action.Code) {
			continue
		}
		out = append(out, action)
	}
	return out
}

func mcpAutoprogrammingLegacyMaterialProgressActionV0(code string) bool {
	switch strings.TrimSpace(code) {
	case mcpAutoprogrammingActionActiveNoCheckpointYetV0,
		mcpAutoprogrammingActionActiveCheckpointOnlyYetV0,
		mcpAutoprogrammingActionActiveTimeoutCheckpointRecentV0,
		mcpAutoprogrammingActionNoCheckpointConsumptionWarningV0,
		mcpAutoprogrammingActionCheckpointOnlyConsumptionWarningV0,
		mcpAutoprogrammingActionNoCheckpointHighConsumptionV0,
		mcpAutoprogrammingActionCheckpointOnlyHighConsumptionV0:
		return true
	default:
		return false
	}
}
