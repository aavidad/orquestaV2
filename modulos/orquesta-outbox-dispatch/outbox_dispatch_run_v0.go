package orquestaoutboxdispatch

import "strings"

func RunOutboxDispatchOnceV0(
	input RunOutboxDispatchOnceInputV0,
) (RunOutboxDispatchOnceResultV0, error) {
	input = normalizeRunOutboxDispatchOnceInputV0(input)
	result := RunOutboxDispatchOnceResultV0{
		Status:     RunOutboxDispatchOnceInvalidV0,
		RunID:      input.RunID,
		TargetPort: input.TargetPort,
	}
	if issues := validateRunOutboxDispatchOnceInputV0(input); len(issues) > 0 {
		result.Issues = issues
		return result, nil
	}

	pending, issues := input.Reader.ListPendingOutboxV0(PendingOutboxFilterV0{
		RunID:       input.RunID,
		TargetPort:  input.TargetPort,
		MessageType: input.MessageType,
	})
	result.Issues = append(result.Issues, issues...)
	result.Decision = ChooseNextDispatchV0(DispatchSelectionV0{
		RunID:       input.RunID,
		TargetPort:  input.TargetPort,
		MessageType: input.MessageType,
		Pending:     pending,
	})
	result.Issues = append(result.Issues, result.Decision.Issues...)
	if result.Decision.Kind != DispatchDecisionReadyV0 {
		result.Status = runStatusFromDecisionV0(result.Decision.Kind)
		return result, nil
	}

	result.Intent = result.Decision.Intent
	claim, claimIssues := input.Claimer.ClaimOutboxDispatchV0(claimFromIntentV0(result.Intent))
	result.Issues = append(result.Issues, claimIssues...)
	if !claim.Claimed {
		result.Status = RunOutboxDispatchOnceAlreadyClaimedV0
		return result, nil
	}

	execution, err := input.Executor.ExecuteOutboxDispatchV0(result.Intent)
	result.Execution = normalizeExecutionResultV0(execution)
	if err != nil {
		result.Status = RunOutboxDispatchOnceDispatchFailedV0
		return result, err
	}

	result.Ack = ackFromExecutionV0(result.Intent, result.Execution)
	ackIssues := input.Acker.AckOutboxDispatchV0(result.Ack)
	result.Issues = append(result.Issues, ackIssues...)
	if len(ackIssues) > 0 {
		result.Status = RunOutboxDispatchOnceAckFailedV0
		return result, nil
	}
	result.Status = RunOutboxDispatchOnceDispatchedV0
	return result, nil
}

func normalizeRunOutboxDispatchOnceInputV0(
	input RunOutboxDispatchOnceInputV0,
) RunOutboxDispatchOnceInputV0 {
	input.RunID = strings.TrimSpace(input.RunID)
	input.TargetPort = strings.TrimSpace(input.TargetPort)
	input.MessageType = strings.TrimSpace(input.MessageType)
	return input
}

func validateRunOutboxDispatchOnceInputV0(input RunOutboxDispatchOnceInputV0) []DispatchIssueV0 {
	var issues []DispatchIssueV0
	if input.TargetPort == "" {
		issues = append(issues, issueV0(issueInvalidRequestV0, "target_port", "target_port requerido"))
	}
	if input.Reader == nil {
		issues = append(issues, issueV0(issueInvalidRequestV0, "reader", "reader requerido"))
	}
	if input.Claimer == nil {
		issues = append(issues, issueV0(issueInvalidRequestV0, "claimer", "claimer requerido"))
	}
	if input.Executor == nil {
		issues = append(issues, issueV0(issueInvalidRequestV0, "executor", "executor requerido"))
	}
	if input.Acker == nil {
		issues = append(issues, issueV0(issueInvalidRequestV0, "acker", "acker requerido"))
	}
	return issues
}

func runStatusFromDecisionV0(kind DispatchDecisionKindV0) RunOutboxDispatchOnceStatusV0 {
	if kind == DispatchDecisionNoPendingV0 {
		return RunOutboxDispatchOnceNoPendingV0
	}
	return RunOutboxDispatchOnceInvalidV0
}

func claimFromIntentV0(intent DispatchIntentV0) OutboxDispatchClaimV0 {
	return OutboxDispatchClaimV0{
		MessageID:      intent.MessageID,
		RunID:          intent.RunID,
		TargetPort:     intent.TargetPort,
		IdempotencyKey: intent.IdempotencyKey,
	}
}

func normalizeExecutionResultV0(result OutboxDispatchExecutionResultV0) OutboxDispatchExecutionResultV0 {
	return OutboxDispatchExecutionResultV0{
		DispatchRef:  strings.TrimSpace(result.DispatchRef),
		EvidenceRefs: compactStringsV0(result.EvidenceRefs),
	}
}

func ackFromExecutionV0(
	intent DispatchIntentV0,
	execution OutboxDispatchExecutionResultV0,
) OutboxDispatchAckV0 {
	return OutboxDispatchAckV0{
		MessageID:    intent.MessageID,
		RunID:        intent.RunID,
		TargetPort:   intent.TargetPort,
		DispatchRef:  execution.DispatchRef,
		EvidenceRefs: compactStringsV0(execution.EvidenceRefs),
	}
}
