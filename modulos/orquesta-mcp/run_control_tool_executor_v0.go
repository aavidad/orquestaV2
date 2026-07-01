package orquestamcp

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type MCPRunControlToolExecutorV0 struct {
	Port              orquestaruncontrol.RunControlWriterPortV0
	ExternalJobSource MCPDirectorExternalJobStatsSourcePortV0
	GoalBackendState  MCPTransportDirectorStatsExecutorV0
}

func NewMCPRunControlToolExecutorV0(
	port orquestaruncontrol.RunControlWriterPortV0,
) MCPRunControlToolExecutorV0 {
	return MCPRunControlToolExecutorV0{Port: port}
}

func (executor MCPRunControlToolExecutorV0) Execute(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var issues []MCPValidationIssueV0
	input, issues = normalizeMCPRunControlIdentityV0(input, "", "")
	if len(issues) > 0 {
		return newMCPRunControlErrorV0(input, issues[0].Code, issues[0].Field), nil
	}
	resolved, ok, err := executor.resolveRunControlInputV0(ctx, input)
	if err != nil {
		return newMCPRunControlErrorV0(input, "external_job_run_ref_error", "external_job_ref"), nil
	}
	if !ok || strings.TrimSpace(resolved.RunRef) == "" {
		return newMCPRunControlErrorV0(input, "run_ref_requerido", "run_ref"), nil
	}
	if executor.Port == nil {
		return newMCPRunControlErrorV0(resolved, "run_control_port_no_disponible", "port"), nil
	}
	if !isMCPRunControlActionSupportedV0(input.Action) {
		return newMCPRunControlErrorV0(resolved, "action_no_soportada", "action"), nil
	}
	beforeLocal := executor.readRunControlStateIfAvailableV0(ctx, resolved.RunRef)
	beforeGoal := executor.observeRunControlGoalBackendV0(ctx, resolved)
	state, err := executor.executeActionV0(ctx, resolved)
	if err != nil {
		return MCPRunControlToolResultV0{}, err
	}
	afterGoal := executor.observeRunControlGoalBackendV0(ctx, resolved)
	result := newMCPRunControlResultV0(resolved, state)
	result = executor.enrichRunControlGoalBackendResultV0(result, resolved, beforeLocal, beforeGoal, afterGoal)
	return result, nil
}

func (executor MCPRunControlToolExecutorV0) resolveRunControlInputV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (MCPRunControlToolInputV0, bool, error) {
	input.RunRef = strings.TrimSpace(input.RunRef)
	if input.RunRef != "" {
		return input, true, nil
	}
	if strings.TrimSpace(input.ExternalJobRef) == "" {
		return input, false, nil
	}
	if executor.ExternalJobSource == nil {
		return input, false, nil
	}
	stats, ok, err := executor.ExternalJobSource.ResolveDirectorExternalJobStatsV0(
		ctx,
		MCPDirectorExternalJobStatsRequestV0{
			AppRef:         strings.TrimSpace(input.AppRef),
			ExternalJobRef: strings.TrimSpace(input.ExternalJobRef),
			CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		},
	)
	if err != nil || !ok || strings.TrimSpace(stats.RunRef) == "" {
		return input, ok, err
	}
	input.RunRef = strings.TrimSpace(stats.RunRef)
	input.EvidenceRefs = compactStringsMCPV0(append(input.EvidenceRefs, stats.JobRef, stats.TaskRef, stats.AgentRef))
	return input, true, nil
}

func (executor MCPRunControlToolExecutorV0) readRunControlStateIfAvailableV0(
	ctx context.Context,
	runRef string,
) *orquestaruncontrol.RunControlStateV0 {
	reader, ok := executor.Port.(orquestaruncontrol.RunControlReaderPortV0)
	if !ok || reader == nil {
		return nil
	}
	state, err := reader.ReadRunControlStateV0(
		ctx,
		orquestaruncontrol.RunControlReadRequestV0{RunRef: strings.TrimSpace(runRef)},
	)
	if err != nil {
		return nil
	}
	normalized := state
	normalized.Status = orquestaruncontrol.NormalizeRunControlStatusV0(state.Status)
	return &normalized
}

func (executor MCPRunControlToolExecutorV0) observeRunControlGoalBackendV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) *MCPDirectorStatsToolResultV0 {
	if executor.GoalBackendState == nil || strings.TrimSpace(input.RunRef) == "" {
		return nil
	}
	stats, err := executor.GoalBackendState.Execute(ctx, MCPDirectorStatsToolInputV0{
		RequestID:            input.RequestID,
		CorrelationID:        firstNonEmptyMCPV0(input.CorrelationID, input.RequestID),
		RunRef:               strings.TrimSpace(input.RunRef),
		AppRef:               strings.TrimSpace(input.AppRef),
		OccurredAt:           "",
		IncludeProcessRefs:   true,
		IncludeAgentProgress: true,
		IncludeAgentUsage:    true,
	})
	if err != nil || stats.Estado != MCPDirectorStatsEstadoOKV0 || stats.Goal == nil {
		return nil
	}
	return &stats
}

func (executor MCPRunControlToolExecutorV0) enrichRunControlGoalBackendResultV0(
	result MCPRunControlToolResultV0,
	input MCPRunControlToolInputV0,
	beforeLocal *orquestaruncontrol.RunControlStateV0,
	beforeGoal *MCPDirectorStatsToolResultV0,
	afterGoal *MCPDirectorStatsToolResultV0,
) MCPRunControlToolResultV0 {
	action := normalizeMCPRunControlActionV0(input.Action)
	if action != "stop" && action != "cancel" {
		return result
	}
	if beforeLocal != nil {
		result.PreviousStatus = string(orquestaruncontrol.NormalizeRunControlStatusV0(beforeLocal.Status))
	}
	result.GoalStatusBefore = mcpRunControlGoalStatusFromStatsV0(beforeGoal)
	result.GoalStatusAfter = mcpRunControlGoalStatusFromStatsV0(afterGoal)
	result.GoalRef = firstNonEmptyMCPV0(
		mcpRunControlGoalRefFromStatsV0(afterGoal),
		mcpRunControlGoalRefFromStatsV0(beforeGoal),
	)
	result.ExternalGoalRef = firstNonEmptyMCPV0(
		mcpRunControlExternalGoalRefFromStatsV0(afterGoal),
		mcpRunControlExternalGoalRefFromStatsV0(beforeGoal),
	)
	result.GoalControlSignalConfirmed = mcpRunControlGoalBackendTerminalV0(afterGoal)
	if !mcpRunControlGoalBackendActiveV0(afterGoal) {
		return result
	}
	result.GoalControlSignalSent = false
	result.GoalControlSignalConfirmed = false
	result.RecommendedAction = "observe_goal_backend_before_declaring_stopped"
	result.Diagnostics = append(result.Diagnostics, MCPRunControlDiagnosticV0{
		Code:    "control_not_propagated_to_goal_backend",
		Scope:   "run:" + strings.TrimSpace(result.RunRef),
		Message: "run control local no confirma stop/cancel del backend goal-first; no publicar stopped como terminal",
		EvidenceRefs: compactStringsMCPV0([]string{
			"evidence-ref-run-control-goal-backend-active",
			result.GoalRef,
			result.ExternalGoalRef,
		}),
	})
	result.Errores = append(result.Errores, MCPValidationIssueV0{
		Code:    "control_not_propagated_to_goal_backend",
		Field:   "goal_backend",
		Message: "goal backend sigue activo tras control local",
	})
	result.Estado = MCPRunControlEstadoErrorV0
	result.Status = mcpRunControlRequestedStatusForActionV0(action)
	result.FinalStatus = result.Status
	return result
}

func mcpRunControlGoalStatusFromStatsV0(stats *MCPDirectorStatsToolResultV0) string {
	if stats == nil || stats.Goal == nil {
		return ""
	}
	return strings.TrimSpace(stats.Goal.Status)
}

func mcpRunControlGoalRefFromStatsV0(stats *MCPDirectorStatsToolResultV0) string {
	if stats == nil || stats.Goal == nil {
		return ""
	}
	return strings.TrimSpace(stats.Goal.GoalRef)
}

func mcpRunControlExternalGoalRefFromStatsV0(stats *MCPDirectorStatsToolResultV0) string {
	if stats == nil || stats.Goal == nil {
		return ""
	}
	return strings.TrimSpace(stats.Goal.ExternalGoalRef)
}

func mcpRunControlGoalBackendActiveV0(stats *MCPDirectorStatsToolResultV0) bool {
	status := strings.ToLower(mcpRunControlGoalStatusFromStatsV0(stats))
	switch status {
	case "active", "running":
		return true
	default:
		return false
	}
}

func mcpRunControlGoalBackendTerminalV0(stats *MCPDirectorStatsToolResultV0) bool {
	status := strings.ToLower(mcpRunControlGoalStatusFromStatsV0(stats))
	switch status {
	case "complete", "completed", "accepted", "canceled", "cancelled", "stopped", "failed":
		return true
	default:
		return false
	}
}

func mcpRunControlRequestedStatusForActionV0(action string) string {
	switch action {
	case "cancel":
		return string(orquestaruncontrol.RunControlStatusCancelRequestedV0)
	default:
		return string(orquestaruncontrol.RunControlStatusStopRequestedV0)
	}
}

func (executor MCPRunControlToolExecutorV0) executeActionV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	switch normalizeMCPRunControlActionV0(input.Action) {
	case "pause":
		return executor.Port.PauseRunV0(ctx, orquestaruncontrol.PauseRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "resume":
		return executor.Port.ResumeRunV0(ctx, orquestaruncontrol.ResumeRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
	case "stop":
		state, err := executor.Port.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			Forced:         input.Forced,
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
		if err != nil {
			return state, err
		}
		return executor.recordStopCheckpointIfAvailableV0(ctx, input, state)
	case "cancel":
		state, err := executor.Port.CancelRunV0(ctx, orquestaruncontrol.CancelRunCommandV0{
			RunRef:         strings.TrimSpace(input.RunRef),
			RequestedBy:    strings.TrimSpace(input.RequestedBy),
			Reason:         strings.TrimSpace(input.Reason),
			Forced:         input.Forced,
			IdempotencyKey: strings.TrimSpace(input.IdempotencyKey),
			EvidenceRefs:   compactStringsMCPV0(input.EvidenceRefs),
		})
		if err != nil {
			return state, err
		}
		return executor.recordStopCheckpointIfAvailableV0(ctx, input, state)
	default:
		return orquestaruncontrol.RunControlStateV0{}, nil
	}
}

func (executor MCPRunControlToolExecutorV0) recordStopCheckpointIfAvailableV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
	state orquestaruncontrol.RunControlStateV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if state.CheckpointRecorded {
		return state, nil
	}
	checkpointWriter, ok := executor.Port.(orquestaruncontrol.RunControlCheckpointWriterPortV0)
	if !ok {
		return state, nil
	}
	action := normalizeMCPRunControlActionV0(input.Action)
	recorded, err := checkpointWriter.RecordRunCheckpointV0(ctx, orquestaruncontrol.RecordRunCheckpointCommandV0{
		RunRef:         strings.TrimSpace(input.RunRef),
		RequestedBy:    firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-run-control"),
		Reason:         firstNonEmptyMCPV0(input.Reason, "checkpoint registrado por control MCP de run"),
		IdempotencyKey: firstNonEmptyMCPV0(input.IdempotencyKey, "idem-mcp-run-control-checkpoint-"+action+"-"+strings.TrimSpace(input.RunRef)),
		EvidenceRefs:   compactStringsMCPV0(append(input.EvidenceRefs, "evidence-ref-mcp-run-control-checkpoint-recorded")),
	})
	if err != nil {
		return state, err
	}
	return recorded, nil
}

func isMCPRunControlActionSupportedV0(action string) bool {
	switch normalizeMCPRunControlActionV0(action) {
	case "pause", "resume", "stop", "cancel":
		return true
	default:
		return false
	}
}
