package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const (
	codexStackExternalWorkAgentRequestedNotStartedDiagnosticV0 = "external_work_agent_requested_not_started"
	codexStackExternalWorkStoppedNoDeliveryDiagnosticV0        = "external_work_accepted_stopped_without_delivery"
)

func codexStackRunSupervisorErrorResultMCPV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
	partial CodexSupervisorResultV0,
	err error,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	var drainErr DrainObservationApplyErrorV0
	if errors.As(err, &drainErr) {
		result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
			input,
			"delivery_ack_ingestion_failed",
			firstNonEmptyQueuedSourceV0(drainErr.Field, "drain_observation"),
			codexStackDrainObservationPublicMessageV0(drainErr),
		)
		result.RunRef = firstNonEmptyQueuedSourceV0(drainErr.RunRef, input.RunRef)
		result.Diagnostics = codexStackDrainObservationDiagnosticsMCPV0(drainErr)
		result.EvidenceRefs = compactStringsV0(append(
			codexStackDrainObservationEvidenceRefsV0(drainErr),
			partial.Last.EvidenceRefs...,
		))
		result.NextActions = compactStringsV0([]string{drainErr.NextActionV0()})
		return result
	}
	result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
		input,
		"run_supervisor_execute_error",
		"executor",
		"run_supervisor_execute_error",
	)
	result.RunRef = firstNonEmptyQueuedSourceV0(partial.Last.SessionRef, input.RunRef)
	result.StopReason = string(partial.StopReason)
	result.Ticks = partial.Ticks
	result.Last = codexStackRunSupervisorSnapshotMCPV0(partial.Last)
	result.History = codexStackRunSupervisorHistoryMCPV0(partial.History)
	result.EvidenceRefs = compactStringsV0(partial.Last.EvidenceRefs)
	liveDiagnostics := codexStackRunSupervisorLiveErrorDiagnosticsMCPV0(result.RunRef, partial, err)
	result.NextActions = codexStackRunSupervisorLiveErrorNextActionsMCPV0(partial)
	if diagnostics := codexStackRunSupervisorDiagnosticsMCPV0(partial.Last.Diagnostics); len(diagnostics) > 0 {
		diagnostics = codexStackRunSupervisorDiagnosticsWithFallbackErrorV0(diagnostics, err)
		result.Diagnostics = append(liveDiagnostics, diagnostics...)
		return result
	}
	if len(liveDiagnostics) > 0 {
		result.Diagnostics = liveDiagnostics
		return result
	}
	if len(result.EvidenceRefs) > 0 {
		result.Diagnostics = []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:         "run_supervisor_partial_snapshot",
			Scope:        "run:" + result.RunRef,
			Message:      "executor fallo con snapshot parcial disponible error=" + codexStackRunSupervisorPublicDiagnosticErrorV0(err.Error()),
			EvidenceRefs: result.EvidenceRefs,
		}}
	}
	return result
}

func codexStackRunSupervisorLiveErrorDiagnosticsMCPV0(
	runRef string,
	partial CodexSupervisorResultV0,
	err error,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if !codexStackRunSupervisorErrorHasLiveAgentV0(partial.Last) {
		return nil
	}
	message := strings.Join(compactStringsV0([]string{
		"last_status=" + strings.TrimSpace(string(partial.Last.Status)),
		"stop_reason=" + strings.TrimSpace(string(partial.StopReason)),
		"action=wait_agents_or_retry_supervise",
		"error=" + codexStackRunSupervisorPublicDiagnosticErrorV0(codexStackRunSupervisorErrorStringV0(err)),
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "supervisor_transition_error_but_agents_live",
		Scope:        codexStackRunSupervisorLiveErrorScopeMCPV0(runRef, partial.Last),
		Message:      message,
		EvidenceRefs: compactStringsV0(append(partial.Last.EvidenceRefs, "evidence-ref-supervisor-transition-error-agents-live")),
	}}
}

func codexStackRunSupervisorErrorHasLiveAgentV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) bool {
	if snapshot.Status == CodexSupervisorRuntimeRunningLiveV0 {
		return true
	}
	for _, ref := range snapshot.EvidenceRefs {
		if strings.TrimSpace(ref) == "evidence-ref-codex-supervisor-process-live" {
			return true
		}
	}
	return false
}

func codexStackRunSupervisorLiveErrorScopeMCPV0(
	runRef string,
	snapshot CodexSupervisorRuntimeSnapshotV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"run:" + firstNonEmptyQueuedSourceV0(runRef, snapshot.SessionRef),
		"agent:" + strings.TrimSpace(snapshot.AgentRef),
		"process:" + strings.TrimSpace(snapshot.ProcessRef),
	}), " ")
}

func codexStackRunSupervisorLiveErrorNextActionsMCPV0(
	partial CodexSupervisorResultV0,
) []string {
	if !codexStackRunSupervisorErrorHasLiveAgentV0(partial.Last) {
		return nil
	}
	return []string{
		"wait_agents",
		"retry_supervise",
		"do_not_relaunch_same_run_ref_while_process_live",
	}
}

func codexStackRunSupervisorErrorStringV0(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (stack *StackV0) codexStackRunSupervisorRequestedNotStartedDiagnosticsMCPV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	result CodexSupervisorResultV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if stack == nil || stack.Stores.RunStore == nil {
		return nil
	}
	runRef := strings.TrimSpace(firstNonEmptyQueuedSourceV0(input.RunRef, result.Last.SessionRef))
	if runRef == "" {
		return nil
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil ||
		!codexStackRunLooksExternalWorkV0(run.ProjectRef, run.AppSpecRef) ||
		len(stackDrainOpenTaskRefsV0(run)) == 0 {
		return nil
	}
	requested := compactStringsV0(run.Agents)
	started := compactStringsV0(run.StartedAgents)
	inFlight := stackDrainPendingStartedAgentRefsV0(run)
	if len(requested) == 0 || len(started) > 0 || len(inFlight) > 0 {
		return nil
	}
	message := strings.Join(compactStringsV0([]string{
		"requested_agents=" + strconv.Itoa(len(requested)),
		"started_agents=0",
		"in_flight=0",
		"cause=unknown",
		"action=retry_materialization_or_check_capacity_auth_runtime_queue_outbox_policy",
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:    codexStackExternalWorkAgentRequestedNotStartedDiagnosticV0,
		Scope:   "run:" + runRef,
		Message: message,
		EvidenceRefs: compactStringsV0(append(
			result.Last.EvidenceRefs,
			"evidence-ref-external-work-agent-requested-not-started",
		)),
	}}
}

func (stack *StackV0) codexStackRunSupervisorStoppedNoDeliveryDiagnosticsMCPV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	result CodexSupervisorResultV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if stack == nil || stack.Stores.RunStore == nil {
		return nil
	}
	if result.StopReason != CodexSupervisorStopStoppedV0 &&
		result.Last.Status != CodexSupervisorRuntimeStoppedV0 {
		return nil
	}
	runRef := strings.TrimSpace(firstNonEmptyQueuedSourceV0(input.RunRef, result.Last.SessionRef))
	if runRef == "" {
		return nil
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil || !codexStackRunLooksExternalWorkV0(run.ProjectRef, run.AppSpecRef) {
		return nil
	}
	if len(compactStringsV0(run.Agents)) > 0 ||
		len(compactStringsV0(run.StartedAgents)) > 0 ||
		len(stackDrainPendingStartedAgentRefsV0(run)) > 0 ||
		len(compactStringsV0(run.Deliveries)) > 0 ||
		len(compactStringsV0(run.DeliveredAgents)) > 0 {
		return nil
	}
	message := strings.Join(compactStringsV0([]string{
		"agents_requested=0",
		"started_agents=0",
		"in_flight=0",
		"deliveries=0",
		"action=relaunch_or_replan_external_work_with_causal_error",
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:    codexStackExternalWorkStoppedNoDeliveryDiagnosticV0,
		Scope:   "run:" + runRef,
		Message: message,
		EvidenceRefs: compactStringsV0(append(
			result.Last.EvidenceRefs,
			"evidence-ref-external-work-accepted-stopped-without-delivery",
		)),
	}}
}

func codexStackRunSupervisorWithRequestedNotStartedActionsMCPV0(
	output orquestamcp.MCPRunSupervisorToolResultV0,
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	hasRequestedNotStarted := false
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == codexStackExternalWorkAgentRequestedNotStartedDiagnosticV0 {
			hasRequestedNotStarted = true
			break
		}
	}
	if !hasRequestedNotStarted {
		return output
	}
	output.NextActions = compactStringsV0(append(output.NextActions,
		"retry_materialization",
		"check_capacity_auth_runtime_queue_outbox_policy",
		"do_not_mark_completed_without_agent_start_or_delivery",
	))
	return output
}

func codexStackRunSupervisorWithStoppedNoDeliveryActionsMCPV0(
	output orquestamcp.MCPRunSupervisorToolResultV0,
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	hasStoppedNoDelivery := false
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == codexStackExternalWorkStoppedNoDeliveryDiagnosticV0 {
			hasStoppedNoDelivery = true
			break
		}
	}
	if !hasStoppedNoDelivery {
		return output
	}
	output.NextActions = compactStringsV0(append(output.NextActions,
		"relaunch_or_replan_external_work_with_causal_error",
		"inspect_external_work_payload_and_runtime_binding",
		"do_not_mark_completed_without_agent_start_or_delivery",
	))
	return output
}

func codexStackRunSupervisorDiagnosticsWithFallbackErrorV0(
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
	err error,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if err == nil || len(diagnostics) == 0 {
		return diagnostics
	}
	publicError := codexStackRunSupervisorPublicDiagnosticErrorV0(err.Error())
	if publicError == "" {
		return diagnostics
	}
	out := append([]orquestamcp.MCPAutoprogrammingDiagnosticV0(nil), diagnostics...)
	hasError := false
	for index := range out {
		if strings.Contains(out[index].Message, "error=") && !strings.Contains(out[index].Message, "error= issues=") {
			hasError = true
			break
		}
	}
	if !hasError {
		out[0].Message = strings.TrimSpace(out[0].Message + " error=" + publicError)
	}
	return out
}

func codexStackRunSupervisorDiagnosticsMCPV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	out := make([]orquestamcp.MCPAutoprogrammingDiagnosticV0, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		code := strings.TrimSpace(diagnostic.Kind)
		if code == "" {
			code = "run_supervisor_diagnostic"
		}
		out = append(out, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         code,
			Scope:        codexStackRunSupervisorDiagnosticScopeMCPV0(diagnostic),
			Message:      codexStackRunSupervisorDiagnosticMessageMCPV0(diagnostic),
			EvidenceRefs: compactStringsV0(diagnostic.EvidenceRefs),
		})
	}
	return out
}

func codexStackRunSupervisorDiagnosticScopeMCPV0(
	diagnostic orquestaruncoordinator.RunDrainDiagnosticV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"run:" + strings.TrimSpace(diagnostic.RunRef),
		"message:" + strings.TrimSpace(diagnostic.MessageID),
	}), " ")
}

func codexStackRunSupervisorDiagnosticMessageMCPV0(
	diagnostic orquestaruncoordinator.RunDrainDiagnosticV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"status=" + strings.TrimSpace(diagnostic.Status),
		"message_type=" + strings.TrimSpace(diagnostic.MessageType),
		codexStackRunSupervisorDiagnosticTargetMCPV0(diagnostic),
		"error=" + codexStackRunSupervisorPublicDiagnosticErrorV0(diagnostic.Error),
		fmt.Sprintf("issues=%d", diagnostic.Issues),
	}), " ")
}

func codexStackRunSupervisorDiagnosticTargetMCPV0(
	diagnostic orquestaruncoordinator.RunDrainDiagnosticV0,
) string {
	targetPort := strings.TrimSpace(diagnostic.TargetPort)
	if targetPort == "" {
		return "target=unknown"
	}
	return "target=" + targetPort
}

func codexStackRunSupervisorPublicDiagnosticErrorV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	redacted, _ := orquestarails.RedactOperationalTextForFieldV0(
		"codex_stack_run_supervisor",
		"diagnostic_error",
		value,
	)
	return strings.TrimSpace(redacted)
}

func codexStackDrainObservationPublicMessageV0(err DrainObservationApplyErrorV0) string {
	parts := compactStringsV0([]string{
		"delivery_ack_ingestion_failed",
		"field=" + err.Field,
		"code=" + err.CommandCode,
		"cause=" + err.Cause,
		"next_action=" + err.NextActionV0(),
	})
	return strings.Join(parts, " ")
}

func codexStackDrainObservationDiagnosticsMCPV0(
	err DrainObservationApplyErrorV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	message := strings.Join(compactStringsV0([]string{
		"field=" + err.Field,
		"code=" + err.CommandCode,
		"cause=" + err.Cause,
		"next_action=" + err.NextActionV0(),
		fmt.Sprintf("agents=%d started=%d failed=%d lost=%d stopped=%d confirmed_stopped=%d deliveries=%d",
			len(compactStringsV0(err.Agents)),
			len(compactStringsV0(err.StartedAgents)),
			len(compactStringsV0(err.FailedAgents)),
			len(compactStringsV0(err.LostAgents)),
			len(compactStringsV0(err.StoppedAgents)),
			len(compactStringsV0(err.ConfirmedStoppedAgents)),
			len(compactStringsV0(err.Deliveries)),
		),
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "drain_observation_apply_failed",
		Scope:        codexStackDrainObservationScopeMCPV0(err),
		Message:      message,
		EvidenceRefs: codexStackDrainObservationEvidenceRefsV0(err),
	}}
}

func codexStackDrainObservationScopeMCPV0(err DrainObservationApplyErrorV0) string {
	parts := compactStringsV0([]string{
		"run:" + err.RunRef,
		"task:" + err.TaskRef,
		"agent:" + err.AgentRef,
	})
	return strings.Join(parts, "/")
}

func codexStackDrainObservationEvidenceRefsV0(err DrainObservationApplyErrorV0) []string {
	return compactStringsV0([]string{
		err.ArtifactRef,
		err.DeliveryRef,
		err.TaskRef,
		err.AgentRef,
		err.PhaseID,
	})
}
