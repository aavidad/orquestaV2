package application

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

const terminalAgentLaunchQuarantineCode = "application.effect_unknown_applied"

type ReconcileTerminalAgentLaunchRequest struct {
	RequestRef         string
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	ExecutionRef       goal.ExecutionRef
	ActionRef          string
	EffectIntentRef    string
	EffectIntentDigest string
	EffectAttemptRef   string
	PlanGeneration     goal.PlanGeneration
	WorkItemGeneration goal.Revision
	ActionFence        uint64
}

type ReconcileTerminalAgentLaunchResult struct {
	Authority TerminalAgentLaunchReconciliationAuthority
	Created   bool
}

// ReconcileTerminalAgentLaunch only authorizes and queues reconciliation. It
// never invokes a provider synchronously and never creates a physical attempt.
func (orchestrator *Orchestrator) ReconcileTerminalAgentLaunch(
	ctx context.Context,
	access Access,
	request ReconcileTerminalAgentLaunchRequest,
) (ReconcileTerminalAgentLaunchResult, error) {
	if orchestrator == nil {
		return ReconcileTerminalAgentLaunchResult{}, errors.New("application.unavailable")
	}
	if err := validateReconcileTerminalAgentLaunchRequest(request); err != nil {
		return ReconcileTerminalAgentLaunchResult{}, err
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return ReconcileTerminalAgentLaunchResult{}, err
	}
	record, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return ReconcileTerminalAgentLaunchResult{}, err
	}
	receiptFingerprint, err := validateTerminalAgentLaunchHistory(record, projectRef, request)
	if err != nil {
		return ReconcileTerminalAgentLaunchResult{}, err
	}
	fingerprint := terminalAgentLaunchReconciliationFingerprint(principal.Ref, projectRef, request)
	now := orchestrator.clock.Now().UTC()
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx,
		access,
		identity.PermissionEffectsApprove,
		"agent-launch-reconciliation:"+request.ActionRef+":"+request.EffectAttemptRef,
		now,
		"authorization-request:agent-launch-reconciliation:"+fingerprint,
	)
	if err != nil {
		return ReconcileTerminalAgentLaunchResult{}, err
	}
	now = authorizationCausalFloor(now, authorization)
	authority := TerminalAgentLaunchReconciliationAuthority{
		Ref:        "agent-launch-reconciliation-authority:" + fingerprint,
		JobRef:     "agent-launch-reconciliation-job:" + fingerprint,
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal.Ref, ProjectRef: projectRef,
		GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef, ExecutionRef: request.ExecutionRef,
		ActionRef: request.ActionRef, EffectIntentRef: request.EffectIntentRef,
		EffectIntentDigest: request.EffectIntentDigest, EffectAttemptRef: request.EffectAttemptRef,
		PlanGeneration: request.PlanGeneration, WorkItemGeneration: request.WorkItemGeneration,
		ActionFence: request.ActionFence, OriginalReceiptFingerprint: receiptFingerprint,
		AuthorizedAt: now,
	}
	persisted, created, err := orchestrator.state.AuthorizeTerminalAgentLaunchReconciliation(
		ctx,
		AuthorizeTerminalAgentLaunchReconciliationState{Authority: authority, AvailableAt: now},
	)
	if err != nil {
		return ReconcileTerminalAgentLaunchResult{}, err
	}
	if persisted != authority {
		return ReconcileTerminalAgentLaunchResult{}, &StateError{Code: StateConflict}
	}
	return ReconcileTerminalAgentLaunchResult{Authority: persisted, Created: created}, nil
}

func validateReconcileTerminalAgentLaunchRequest(request ReconcileTerminalAgentLaunchRequest) error {
	if !validApplicationRef(request.RequestRef) || request.GoalRef.String() == "" ||
		request.WorkItemRef.String() == "" || request.ExecutionRef.String() == "" ||
		request.ActionRef != "action:launch:"+request.ExecutionRef.String() ||
		!validApplicationRef(request.EffectIntentRef) || !validEffectDigest(request.EffectIntentDigest) ||
		!validApplicationRef(request.EffectAttemptRef) || request.PlanGeneration == 0 ||
		request.WorkItemGeneration == 0 || request.ActionFence == 0 {
		return errors.New("application.agent_launch_terminal_reconciliation_invalid")
	}
	return nil
}

func validateTerminalAgentLaunchHistory(
	record GoalRecord,
	projectRef goal.ProjectRef,
	request ReconcileTerminalAgentLaunchRequest,
) (string, error) {
	if record.Goal.Project() != projectRef || record.Goal.Ref() != request.GoalRef ||
		record.Goal.State() != goal.GoalStateRunning {
		return "", &StateError{Code: StateConflict}
	}
	execution, found := executionByRef(record.Executions, request.ExecutionRef)
	if !found || execution.State != ExecutionDispatching || execution.GoalRef != request.GoalRef ||
		execution.WorkItemRef != request.WorkItemRef || execution.PlanGeneration != request.PlanGeneration ||
		execution.EffectIntentRef != request.EffectIntentRef || execution.LaunchReceiptRef != "" {
		return "", &StateError{Code: StateConflict}
	}
	item, found := record.Goal.WorkItem(request.WorkItemRef)
	if !found || item.Revision() < request.WorkItemGeneration {
		return "", &StateError{Code: StateConflict}
	}
	intent, found := effectIntentByRef(record.EffectIntents, request.EffectIntentRef)
	if !found || ValidateEffectIntent(intent) != nil || intent.Digest != request.EffectIntentDigest ||
		intent.ActionRef != request.ActionRef || intent.ActionKind != ActionLaunchAgent ||
		intent.Kind != EffectKindAgentLaunch || intent.Subject.GoalRef != request.GoalRef ||
		intent.Subject.WorkItemRef != request.WorkItemRef || intent.Subject.ExecutionRef != request.ExecutionRef ||
		intent.Subject.PlanGeneration != request.PlanGeneration {
		return "", &StateError{Code: StateConflict}
	}
	attempt, found := terminalEffectAttemptByRef(record.EffectAttempts, request.EffectAttemptRef)
	if !found || attempt.ActionRef != request.ActionRef || attempt.IntentRef != request.EffectIntentRef ||
		attempt.IntentDigest != request.EffectIntentDigest || attempt.ActionFence != request.ActionFence ||
		effectAttemptHasReceipt(record.EffectReceipts, attempt) || effectAttemptDefinitelyUnapplied(record, attempt) {
		return "", &StateError{Code: StateConflict}
	}
	var quarantine ActionConsumptionReceipt
	matches := 0
	for _, receipt := range record.ConsumptionReceipts {
		if receipt.ActionRef == request.ActionRef {
			quarantine = receipt
			matches++
		}
	}
	if matches != 1 || quarantine.Kind != ActionLaunchAgent || quarantine.GoalRef != request.GoalRef ||
		quarantine.WorkItemRef != request.WorkItemRef || quarantine.ExecutionRef != request.ExecutionRef ||
		quarantine.PlanGeneration != request.PlanGeneration ||
		quarantine.WorkItemGeneration != request.WorkItemGeneration || quarantine.Fence != request.ActionFence ||
		quarantine.Outcome != ActionConsumedQuarantined || quarantine.ErrorCode != terminalAgentLaunchQuarantineCode ||
		quarantine.EffectReceiptRef != "" {
		return "", &StateError{Code: StateConflict}
	}
	if len(record.EffectReceipts) != 0 && recoveryReceiptsRelated(record, request.ActionRef, request.EffectIntentRef) {
		return "", &StateError{Code: StateConflict}
	}
	return TerminalAgentLaunchReceiptFingerprint(quarantine), nil
}

func terminalAgentLaunchReconciliationFingerprint(
	principal identity.PrincipalRef,
	projectRef goal.ProjectRef,
	request ReconcileTerminalAgentLaunchRequest,
) string {
	return fingerprintFields(
		"orquesta.agent-launch-terminal-reconciliation.v1",
		principal.String(), projectRef.String(), request.RequestRef, request.GoalRef.String(),
		request.WorkItemRef.String(), request.ExecutionRef.String(), request.ActionRef,
		request.EffectIntentRef, request.EffectIntentDigest, request.EffectAttemptRef,
		strconv.FormatUint(uint64(request.PlanGeneration), 10),
		strconv.FormatUint(uint64(request.WorkItemGeneration), 10),
		strconv.FormatUint(request.ActionFence, 10),
	)
}

// TerminalAgentLaunchReceiptFingerprint binds the complete immutable
// quarantine receipt so a store cannot authorize against crossed history.
func TerminalAgentLaunchReceiptFingerprint(receipt ActionConsumptionReceipt) string {
	return fingerprintFields(
		"orquesta.agent-launch-terminal-quarantine-receipt.v1",
		receipt.ActionRef, string(receipt.Kind), receipt.GoalRef.String(), receipt.WorkItemRef.String(),
		receipt.ExecutionRef.String(), strconv.FormatUint(uint64(receipt.PlanGeneration), 10),
		strconv.FormatUint(uint64(receipt.WorkItemGeneration), 10), strconv.FormatUint(receipt.Fence, 10),
		strconv.FormatUint(receipt.DeliveryAttempt, 10), receipt.ClaimToken, receipt.WorkerRef,
		string(receipt.Outcome), strings.TrimSpace(receipt.ErrorCode),
		strconv.FormatInt(receipt.ConsumedAt.UTC().UnixNano(), 10), receipt.EffectReceiptRef,
	)
}

func terminalEffectAttemptByRef(attempts []EffectAttempt, ref string) (EffectAttempt, bool) {
	for _, attempt := range attempts {
		if attempt.Ref == ref {
			return attempt, true
		}
	}
	return EffectAttempt{}, false
}
