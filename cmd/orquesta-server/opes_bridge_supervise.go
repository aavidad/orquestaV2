package main

import (
	"context"
	"net/http"
	"strings"
)

func opesBridgeSuperviseSubmittedRunV0(
	ctx context.Context,
	client *http.Client,
	config opesDrainConfigV0,
	summary *opesDrainSummaryV0,
	result *opesDrainJobResultV0,
) {
	if result == nil ||
		summary == nil ||
		client == nil ||
		config.DryRun ||
		strings.TrimSpace(result.RunRef) == "" ||
		(strings.TrimSpace(result.Status) != "submitted" &&
			strings.TrimSpace(result.Status) != "already_submitted" &&
			strings.TrimSpace(result.Status) != "recovery_required") {
		return
	}
	if opesBridgeResultUsesGoalFirstV0(*result) {
		opesBridgeObserveSubmittedGoalV0(ctx, client, config, result)
		return
	}
	if config.SuperviseSubmitted &&
		opesBridgeHydrateAlreadySubmittedGoalFirstFromStatsV0(ctx, client, config, result) {
		opesBridgeObserveSubmittedGoalV0(ctx, client, config, result)
		return
	}
	if !config.SuperviseSubmitted {
		if config.ResidentDispatchWait > 0 {
			supervision, err := observeOPESBridgeResidentDispatchUntilV0(
				ctx,
				client,
				config,
				result.RunRef,
			)
			if err != nil {
				result.SupervisionStatus = "retry_pending"
				result.SupervisionStopReason = "resident_dispatch_observation_unavailable"
				return
			}
			if opesBridgeSupervisionUsesGoalFirstV0(supervision) {
				opesBridgeApplySupervisionGoalFirstMetadataV0(result, supervision)
				opesBridgePersistSubmittedGoalFirstMetadataV0(ctx, config, result)
				opesBridgeObserveSubmittedGoalV0(ctx, client, config, result)
				return
			}
			result.SupervisionStatus = supervision.Status
			result.SupervisionStopReason = supervision.StopReason
			result.SupervisionProcessRef = supervision.ProcessRef
			result.SupervisionEvidenceRef = supervision.EvidenceRef
			return
		}
		result.SupervisionStatus = "resident_director_pending"
		return
	}
	supervision, err := superviseOPESExternalWorkRunV0(
		ctx,
		client,
		config.OrquestaBaseURL,
		result.RunRef,
		config.CorrelationID,
	)
	if err != nil {
		result.SupervisionStatus = "retry_pending"
		result.SupervisionStopReason = "supervision_unavailable"
		return
	}
	if opesBridgeSupervisionUsesGoalFirstV0(supervision) {
		opesBridgeApplySupervisionGoalFirstMetadataV0(result, supervision)
		opesBridgePersistSubmittedGoalFirstMetadataV0(ctx, config, result)
		opesBridgeObserveSubmittedGoalV0(ctx, client, config, result)
		return
	}
	if opesBridgePendingJobNeedsRunRecoveryV0(supervision) {
		terminalPending := opesBridgePendingJobTerminalSupervisionV0(supervision)
		recovery, err := recoverStoppedOPESExternalWorkRunV0(
			ctx,
			client,
			config.OrquestaBaseURL,
			result.RunRef,
			result.JobRef,
			config.CorrelationID,
		)
		if err != nil {
			result.SupervisionStatus = supervision.Status
			result.SupervisionStopReason = supervision.StopReason
			result.SupervisionProcessRef = supervision.ProcessRef
			result.SupervisionEvidenceRef = supervision.EvidenceRef
			result.RunRecoveryStatus = "retry_pending"
			if terminalPending {
				result.SupervisionStatus = "retry_pending"
				result.SupervisionStopReason = "pending_opes_job_terminal_supervision"
			}
			return
		}
		result.RunRecoveryStatus = recovery.Status
		result.RunRecoveryEvidenceRef = recovery.EvidenceRef
		supervision, err = superviseOPESExternalWorkRunV0(
			ctx,
			client,
			config.OrquestaBaseURL,
			result.RunRef,
			config.CorrelationID,
		)
		if err != nil {
			result.SupervisionStatus = "retry_pending"
			result.SupervisionStopReason = "supervision_after_recovery_error"
			return
		}
		if opesBridgePendingJobTerminalSupervisionV0(supervision) {
			result.SupervisionStatus = "retry_pending"
			result.SupervisionStopReason = "pending_opes_job_terminal_after_recovery"
			result.SupervisionProcessRef = supervision.ProcessRef
			result.SupervisionEvidenceRef = supervision.EvidenceRef
			return
		}
	}
	result.SupervisionStatus = supervision.Status
	result.SupervisionStopReason = supervision.StopReason
	result.SupervisionProcessRef = supervision.ProcessRef
	result.SupervisionEvidenceRef = supervision.EvidenceRef
}

func opesBridgeHydrateAlreadySubmittedGoalFirstFromStatsV0(
	ctx context.Context,
	client *http.Client,
	config opesDrainConfigV0,
	result *opesDrainJobResultV0,
) bool {
	if result == nil ||
		client == nil ||
		strings.TrimSpace(result.RunRef) == "" {
		return false
	}
	status := strings.TrimSpace(result.Status)
	if status != "already_submitted" && status != "recovery_required" {
		return false
	}
	decoded, err := loadOPESExternalWorkRunStatsResponseV0(
		ctx,
		client,
		config.OrquestaBaseURL,
		result.RunRef,
		config.CorrelationID,
	)
	if err != nil {
		return false
	}
	metadata, ok := opesBridgeGoalMetadataFromDirectorStatsV0(decoded)
	if !ok {
		return false
	}
	opesBridgeApplyRunMetadataV0(result, metadata)
	opesBridgePersistSubmittedGoalFirstMetadataV0(ctx, config, result)
	return true
}

func opesBridgeObserveSubmittedGoalV0(
	ctx context.Context,
	client *http.Client,
	config opesDrainConfigV0,
	result *opesDrainJobResultV0,
) {
	if result == nil {
		return
	}
	if !config.SuperviseSubmitted && config.ResidentDispatchWait <= 0 {
		result.SupervisionStatus = "goal_first_observe_pending"
		result.SupervisionStopReason = "observe_goal_required"
		return
	}
	supervision, err := observeOPESExternalWorkGoalV0(
		ctx,
		client,
		config.OrquestaBaseURL,
		result.RunRef,
		config.CorrelationID,
	)
	if err != nil {
		result.SupervisionStatus = "retry_pending"
		result.SupervisionStopReason = opesBridgeGoalObservationStopReasonV0(err)
		return
	}
	result.SupervisionStatus = supervision.Status
	result.SupervisionStopReason = supervision.StopReason
	result.SupervisionProcessRef = supervision.ProcessRef
	result.SupervisionEvidenceRef = supervision.EvidenceRef
}

func opesBridgeGoalObservationStopReasonV0(err error) string {
	const fallback = "goal_observation_unavailable"
	if err == nil {
		return fallback
	}
	code := compactCommandHTTPErrorCodeV0(err.Error())
	if code == "" {
		return fallback
	}
	return fallback + ":" + code
}

func opesBridgePersistSubmittedGoalFirstMetadataV0(
	ctx context.Context,
	config opesDrainConfigV0,
	result *opesDrainJobResultV0,
) {
	if result == nil ||
		config.InputLedger == nil ||
		strings.TrimSpace(result.JobRef) == "" ||
		strings.TrimSpace(result.RunRef) == "" {
		return
	}
	_ = externalBridgeRecordSubmittedInputV0(
		ctx,
		config.InputLedger,
		opesBridgeExternalSystemV0,
		result.JobRef,
		result.RunRef,
		result.ChangeRef,
		opesBridgeRunMetadataFromDrainResultV0(*result),
	)
}

func opesBridgeSupervisionUsesGoalFirstV0(
	supervision opesExternalWorkRunSupervisionV0,
) bool {
	if strings.TrimSpace(supervision.RoutePolicy) == opesBridgeRoutePolicyGoalFirstV0 ||
		strings.TrimSpace(supervision.DirectorExecutionMode) == opesBridgeDirectorExecutionModeGoalFirstV0 ||
		strings.TrimSpace(supervision.GoalRef) != "" ||
		strings.TrimSpace(supervision.ExternalGoalRef) != "" ||
		strings.TrimSpace(supervision.StopReason) == "goal_first_observe_required" {
		return true
	}
	for _, action := range supervision.NextActions {
		if strings.TrimSpace(action) == opesBridgeNextActionObserveGoalV0 {
			return true
		}
	}
	return false
}

func opesBridgeApplySupervisionGoalFirstMetadataV0(
	result *opesDrainJobResultV0,
	supervision opesExternalWorkRunSupervisionV0,
) {
	if result == nil {
		return
	}
	opesBridgeApplyRunMetadataV0(result, externalBridgeInputRunMetadataV0{
		RoutePolicy:           opesBridgeRoutePolicyGoalFirstV0,
		DirectorExecutionMode: firstNonEmptyEnvlessV0(supervision.DirectorExecutionMode, opesBridgeDirectorExecutionModeGoalFirstV0),
		GoalRef:               strings.TrimSpace(supervision.GoalRef),
		ExternalGoalRef:       strings.TrimSpace(supervision.ExternalGoalRef),
		NextActions: uniqueOPESBridgeGoalFirstNextActionsV0(append(
			[]string{opesBridgeNextActionObserveGoalV0},
			supervision.NextActions...,
		)),
	})
}

func uniqueOPESBridgeGoalFirstNextActionsV0(
	values []string,
) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func opesBridgePendingJobNeedsRunRecoveryV0(
	supervision opesExternalWorkRunSupervisionV0,
) bool {
	if strings.ToLower(strings.TrimSpace(supervision.Status)) == "stopped" {
		return true
	}
	return opesBridgePendingJobTerminalSupervisionV0(supervision)
}

func opesBridgePendingJobTerminalSupervisionV0(
	supervision opesExternalWorkRunSupervisionV0,
) bool {
	status := strings.ToLower(strings.TrimSpace(supervision.Status))
	stopReason := strings.ToLower(strings.TrimSpace(supervision.StopReason))
	switch status {
	case "done", "complete", "completed", "quiescent", "closed":
		return true
	}
	switch stopReason {
	case "done", "quiescent", "stop_quiescent":
		return true
	default:
		return false
	}
}
