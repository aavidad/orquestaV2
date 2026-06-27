package orquestaappcodexstack

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

const (
	codexStackExternalWorkAgentRequestedNotStartedDiagnosticV0 = "external_work_agent_requested_not_started"
	codexStackExternalWorkStoppedNoDeliveryDiagnosticV0        = "external_work_accepted_stopped_without_delivery"
	codexStackExternalWorkNoAgentMaterializedDiagnosticV0      = "external_work_accepted_no_agent_materialized"
	codexStackStopPendingDispatchInProgressDiagnosticV0        = "stop_pending_but_dispatch_in_progress"
	codexStackCodexRuntimeStreamFDWarningDiagnosticV0          = "codex_runtime_stream_fd_warning"
	codexStackCodexProviderQuotaExhaustedDiagnosticV0          = "codex_provider_quota_exhausted"
	codexStackCodexProviderCapacityLimitedDiagnosticV0         = "codex_provider_capacity_limited"
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
	evidenceDiagnostics := codexStackRunSupervisorEvidenceDiagnosticsMCPV0(partial.Last)
	reviewDiagnostics := codexStackRunSupervisorReviewResultErrorDiagnosticsMCPV0(result.RunRef, partial.Last, err)
	result.NextActions = compactStringsV0(append(
		codexStackRunSupervisorLiveErrorNextActionsMCPV0(partial),
		codexStackRunSupervisorReviewResultErrorNextActionsMCPV0(err)...,
	))
	if diagnostics := codexStackRunSupervisorDiagnosticsMCPV0(partial.Last.Diagnostics); len(diagnostics) > 0 {
		diagnostics = codexStackRunSupervisorDiagnosticsWithFallbackErrorV0(diagnostics, err)
		result.Diagnostics = append(append(append(liveDiagnostics, evidenceDiagnostics...), reviewDiagnostics...), diagnostics...)
		return result
	}
	if len(liveDiagnostics) > 0 || len(evidenceDiagnostics) > 0 || len(reviewDiagnostics) > 0 {
		result.Diagnostics = append(append(liveDiagnostics, evidenceDiagnostics...), reviewDiagnostics...)
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

func codexStackRunSupervisorEvidenceDiagnosticsMCPV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	evidenceRefs := compactStringsV0(snapshot.EvidenceRefs)
	if len(evidenceRefs) == 0 {
		return nil
	}
	scope := codexStackRunSupervisorEvidenceScopeMCPV0(snapshot)
	out := []orquestamcp.MCPAutoprogrammingDiagnosticV0{}
	if refs := codexStackRunSupervisorStreamFDWarningEvidenceRefsV0(evidenceRefs); len(refs) > 0 {
		out = append(out, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         codexStackCodexRuntimeStreamFDWarningDiagnosticV0,
			Scope:        scope,
			Message:      "warning=stream_fd action=continue_observing_do_not_relaunch_by_itself",
			EvidenceRefs: refs,
		})
	}
	if refs := codexStackRunSupervisorQuotaEvidenceRefsV0(evidenceRefs); len(refs) > 0 {
		out = append(out, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         codexStackCodexProviderQuotaExhaustedDiagnosticV0,
			Scope:        scope,
			Message:      "provider_quota_exhausted action=wait_for_quota_or_requeue_capacity_limited",
			EvidenceRefs: refs,
		})
	}
	if refs := codexStackRunSupervisorCapacityEvidenceRefsV0(evidenceRefs); len(refs) > 0 {
		out = append(out, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         codexStackCodexProviderCapacityLimitedDiagnosticV0,
			Scope:        scope,
			Message:      "provider_capacity_limited action=wait_for_capacity_or_requeue_capacity_limited",
			EvidenceRefs: refs,
		})
	}
	return out
}

func codexStackRunSupervisorEvidenceScopeMCPV0(
	snapshot CodexSupervisorRuntimeSnapshotV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"run:" + strings.TrimSpace(snapshot.SessionRef),
		"agent:" + strings.TrimSpace(snapshot.AgentRef),
		"process:" + strings.TrimSpace(snapshot.ProcessRef),
	}), " ")
}

func codexStackRunSupervisorStreamFDWarningEvidenceRefsV0(
	evidenceRefs []string,
) []string {
	out := []string{}
	for _, ref := range evidenceRefs {
		ref = strings.TrimSpace(ref)
		if ref == "evidence-ref-warning-stream-fd" {
			out = append(out, ref)
		}
	}
	return compactStringsV0(out)
}

func codexStackRunSupervisorQuotaEvidenceRefsV0(
	evidenceRefs []string,
) []string {
	out := []string{}
	for _, ref := range evidenceRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		switch {
		case ref == "evidence-ref-provider-quota-exhausted",
			ref == "evidence-ref-provider-usage-limit-retry-after",
			ref == "evidence-ref-codex-usage-quota-exhausted",
			ref == "evidence-ref-codex-usage-quota-limited":
			out = append(out, ref)
		}
	}
	return compactStringsV0(out)
}

func codexStackRunSupervisorCapacityEvidenceRefsV0(
	evidenceRefs []string,
) []string {
	out := []string{}
	for _, ref := range evidenceRefs {
		ref = strings.TrimSpace(ref)
		if ref == "evidence-ref-capacity-limited" || ref == "evidence-ref-capacity-warning" {
			out = append(out, ref)
		}
	}
	return compactStringsV0(out)
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

func (stack *StackV0) codexStackRunSupervisorStopPendingDispatchDiagnosticsMCPV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	result CodexSupervisorResultV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if stack == nil || stack.Stores.RunControl == nil || stack.Stores.RunStore == nil {
		return nil
	}
	runRef := strings.TrimSpace(firstNonEmptyQueuedSourceV0(input.RunRef, result.Last.SessionRef))
	if runRef == "" {
		return nil
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(ctx, orquestaruncontrol.RunControlReadRequestV0{RunRef: runRef})
	if err != nil || !codexStackRunControlStopPendingV0(state.Status) {
		return nil
	}
	run, err := stack.Stores.RunStore.LoadRunV0(ctx, runRef)
	if err != nil {
		return nil
	}
	started := compactStringsV0(run.StartedAgents)
	inFlight := stackDrainPendingStartedAgentRefsV0(run)
	if len(started) == 0 &&
		len(inFlight) == 0 &&
		result.StopReason != CodexSupervisorStopDispatchV0 &&
		result.Last.Status != CodexSupervisorRuntimeRunningLiveV0 {
		return nil
	}
	message := strings.Join(compactStringsV0([]string{
		"control_status=" + strings.TrimSpace(string(state.Status)),
		"started_agents=" + strconv.Itoa(len(started)),
		"in_flight=" + strconv.Itoa(len(inFlight)),
		"last_status=" + strings.TrimSpace(string(result.Last.Status)),
		"action=wait_agents_or_confirm_stop_do_not_kill_useful_delivery",
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:    codexStackStopPendingDispatchInProgressDiagnosticV0,
		Scope:   codexStackRunSupervisorLiveErrorScopeMCPV0(runRef, result.Last),
		Message: message,
		EvidenceRefs: compactStringsV0(append(
			append([]string{}, result.Last.EvidenceRefs...),
			append(state.EvidenceRefs, "evidence-ref-stop-pending-dispatch-in-progress")...,
		)),
	}}
}

func codexStackRunControlStopPendingV0(status orquestaruncontrol.RunControlStatusV0) bool {
	switch orquestaruncontrol.NormalizeRunControlStatusV0(status) {
	case orquestaruncontrol.RunControlStatusStopRequestedV0,
		orquestaruncontrol.RunControlStatusCancelRequestedV0:
		return true
	default:
		return false
	}
}

func codexStackRunSupervisorReviewResultErrorDiagnosticsMCPV0(
	runRef string,
	snapshot CodexSupervisorRuntimeSnapshotV0,
	err error,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	var reviewErr orquestacoreworkflow.ReviewResultErrorV0
	if !errors.As(err, &reviewErr) {
		return nil
	}
	code := strings.TrimSpace(reviewErr.Code)
	if code == "" {
		code = orquestacoreworkflow.ErrReviewResultPayloadInvalidoV0
	}
	message := strings.Join(compactStringsV0([]string{
		"review_result_error=" + codexStackRunSupervisorPublicDiagnosticErrorV0(code),
		"field=" + codexStackRunSupervisorPublicDiagnosticErrorV0(reviewErr.Field),
		"action=retry_review_with_compact_payload",
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "review_result_payload_invalid_after_delivery",
		Scope:        codexStackRunSupervisorReviewResultErrorScopeMCPV0(runRef, snapshot),
		Message:      message,
		EvidenceRefs: compactStringsV0(append(snapshot.EvidenceRefs, "evidence-ref-review-result-payload-invalid")),
	}}
}

func codexStackRunSupervisorReviewResultErrorScopeMCPV0(
	runRef string,
	snapshot CodexSupervisorRuntimeSnapshotV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"run:" + firstNonEmptyQueuedSourceV0(runRef, snapshot.SessionRef),
		"agent:" + strings.TrimSpace(snapshot.AgentRef),
		"process:" + strings.TrimSpace(snapshot.ProcessRef),
	}), " ")
}

func codexStackRunSupervisorReviewResultErrorNextActionsMCPV0(err error) []string {
	var reviewErr orquestacoreworkflow.ReviewResultErrorV0
	if !errors.As(err, &reviewErr) {
		return nil
	}
	return []string{
		"preserve_delivery_evidence",
		"retry_review_with_compact_payload",
		"do_not_relaunch_agent_for_review_payload_error",
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

func (stack *StackV0) codexStackRunSupervisorNoAgentMaterializedDiagnosticsMCPV0(
	ctx context.Context,
	input orquestamcp.MCPRunSupervisorToolInputV0,
	result CodexSupervisorResultV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if stack == nil || stack.Stores.RunStore == nil {
		return nil
	}
	if result.StopReason != CodexSupervisorStopDoneV0 &&
		result.Last.Status != CodexSupervisorRuntimeDoneV0 {
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
		"terminal_status=done",
		"action=relaunch_or_replan_external_work_with_causal_error",
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:    codexStackExternalWorkNoAgentMaterializedDiagnosticV0,
		Scope:   "run:" + runRef,
		Message: message,
		EvidenceRefs: compactStringsV0(append(
			result.Last.EvidenceRefs,
			"evidence-ref-external-work-accepted-no-agent-materialized",
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

func codexStackRunSupervisorWithNoAgentMaterializedActionsMCPV0(
	output orquestamcp.MCPRunSupervisorToolResultV0,
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	hasNoAgentMaterialized := false
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == codexStackExternalWorkNoAgentMaterializedDiagnosticV0 {
			hasNoAgentMaterialized = true
			break
		}
	}
	if !hasNoAgentMaterialized {
		return output
	}
	output.NextActions = compactStringsV0(append(output.NextActions,
		"relaunch_or_replan_external_work_with_causal_error",
		"inspect_external_work_payload_and_runtime_binding",
		"do_not_mark_completed_without_agent_start_or_delivery",
	))
	return output
}

func codexStackRunSupervisorWithStopPendingDispatchActionsMCPV0(
	output orquestamcp.MCPRunSupervisorToolResultV0,
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	hasStopPendingDispatch := false
	for _, diagnostic := range diagnostics {
		if strings.TrimSpace(diagnostic.Code) == codexStackStopPendingDispatchInProgressDiagnosticV0 {
			hasStopPendingDispatch = true
			break
		}
	}
	if !hasStopPendingDispatch {
		return output
	}
	output.NextActions = compactStringsV0(append(output.NextActions,
		"wait_agents_or_confirm_stop",
		"observe_run_control_and_director_stats",
		"do_not_kill_live_agents_with_useful_output",
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
