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
