package application

import (
	"time"

	"orquesta/internal/ports"
)

func validateAgentEnvironmentLifecycleSnapshot(snapshot AgentEnvironmentLifecycleSnapshot) error {
	request := ports.AgentEnvironmentInspectRequest{Subject: snapshot.Subject}
	if snapshot.Schema != agentEnvironmentLifecycleSnapshotSchema || snapshot.Revision == 0 ||
		!validApplicationRef(snapshot.LaunchReceiptRef) || snapshot.RecordedAt.IsZero() ||
		ports.ValidateAgentEnvironmentInspectReceipt(request, ports.AgentEnvironmentInspectReceipt{
			Subject: snapshot.Subject, Token: snapshot.Token,
		}) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if snapshot.Effect.IsEmpty() {
		if snapshot.Token.State != ports.AgentEnvironmentActive ||
			snapshot.Preservation != (ports.AgentPreservationBinding{}) || snapshot.PreservationReceiptRef != "" {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	effect := snapshot.Effect
	wantKind, ok := agentEnvironmentLifecycleAction(effect.ExpectedToken.State)
	if !ok || effect.ActionKind != wantKind || !validApplicationRef(effect.ActionRef) ||
		!validApplicationRef(effect.AttemptRef) || effect.ActionFence == 0 ||
		effect.DeliveryAttempt == 0 || effect.WorkItemGeneration == 0 ||
		effect.ClaimToken == "" || effect.WorkerRef == "" ||
		!validApplicationRef(effect.IdempotencyKey) ||
		ports.ValidateAgentEnvironmentInspectReceipt(request, ports.AgentEnvironmentInspectReceipt{
			Subject: snapshot.Subject, Token: effect.ExpectedToken,
		}) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if effect.OutcomeToken == (ports.AgentEnvironmentLifecycleToken{}) {
		if snapshot.Token != effect.ExpectedToken || effect.PhysicalReceipt != "" || effect.EffectReceiptRef != "" {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	} else {
		if snapshot.Token != effect.OutcomeToken ||
			effect.OutcomeToken.Revision == effect.ExpectedToken.Revision ||
			effect.OutcomeToken.Fence != effect.ExpectedToken.Fence {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		_, terminal := agentEnvironmentLifecycleOutcomeStates(effect.ActionKind)
		switch snapshot.Token.State {
		case terminal:
			if !validApplicationRef(effect.PhysicalReceipt) || !validApplicationRef(effect.EffectReceiptRef) {
				return errAgentEnvironmentLifecycleStateInvalid
			}
		default:
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	if agentEnvironmentLifecycleRequiresPreservation(snapshot.Token.State) {
		if !validAgentEnvironmentPreservation(snapshot.Preservation) ||
			!validApplicationRef(snapshot.PreservationReceiptRef) {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	} else if snapshot.Preservation != (ports.AgentPreservationBinding{}) || snapshot.PreservationReceiptRef != "" {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentLifecycleClaim(
	snapshot AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	wantKind ActionKind,
	at time.Time,
) error {
	intent := claim.Action.EffectIntent
	if at.IsZero() || claim.Disposition != ActionClaimDispositionNormal || claim.RecoveryEffectAttemptRef != "" ||
		claim.Action.Kind != wantKind || claim.Action.GoalRef != snapshot.Subject.GoalRef ||
		claim.Action.WorkItemRef != snapshot.Subject.WorkItemRef ||
		claim.Action.ExecutionRef != snapshot.Subject.ExecutionRef ||
		claim.Action.PlanGeneration != snapshot.Subject.PlanGeneration || claim.Action.WorkItemGeneration == 0 ||
		claim.Action.ChangeRef.String() != "" || claim.DeliveryAttempt == 0 || claim.Fence == 0 ||
		claim.Token == "" || claim.WorkerRef == "" || !claim.LeaseUntil.After(at) ||
		validateClaimedEffect(claim, at) != nil || !effectActionKindMatches(intent.Kind, wantKind) ||
		intent.Subject.GoalRef != snapshot.Subject.GoalRef ||
		intent.Subject.WorkItemRef != snapshot.Subject.WorkItemRef ||
		intent.Subject.ExecutionRef != snapshot.Subject.ExecutionRef ||
		intent.Subject.PlanGeneration != snapshot.Subject.PlanGeneration ||
		intent.Subject.AppSpecGeneration != snapshot.Subject.AppSpecGeneration ||
		intent.Subject.SpecHash != snapshot.Subject.SpecHash {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentLifecyclePreEffect(state AgentEnvironmentLifecyclePreEffectState) error {
	snapshot := state.Snapshot
	authoritySnapshot := snapshot
	authoritySnapshot.Token = snapshot.Effect.ExpectedToken
	wantReceiptRef := ""
	if state.Claim.Action.Kind == ActionCloseAgentEnvironment {
		wantReceiptRef = snapshot.Preservation.ApplicationReceiptRef
	}
	if state.ExpectedRevision == 0 || snapshot.Revision != state.ExpectedRevision+1 ||
		state.RequiredApplicationReceiptRef != wantReceiptRef || !snapshot.RecordedAt.Equal(state.OperationAt) ||
		snapshot.Token != snapshot.Effect.ExpectedToken || !snapshot.Effect.NeedsReconciliation() ||
		snapshot.Effect.OutcomeToken != (ports.AgentEnvironmentLifecycleToken{}) ||
		snapshot.Effect.ActionRef != state.Claim.Action.Ref || snapshot.Effect.ActionKind != state.Claim.Action.Kind ||
		snapshot.Effect.AttemptRef != state.Attempt.Ref || snapshot.Effect.ActionFence != state.Attempt.ActionFence ||
		snapshot.Effect.DeliveryAttempt != state.Claim.DeliveryAttempt ||
		snapshot.Effect.WorkItemGeneration != state.Claim.Action.WorkItemGeneration ||
		snapshot.Effect.ClaimToken != state.Claim.Token || snapshot.Effect.WorkerRef != state.Claim.WorkerRef ||
		snapshot.Effect.IdempotencyKey != state.Attempt.IdempotencyKey ||
		validateAgentEnvironmentLifecycleClaim(
			authoritySnapshot, state.Claim, state.Claim.Action.Kind, state.Attempt.StartedAt,
		) != nil ||
		validateEffectAttempt(state.Claim, state.Attempt) != nil ||
		validateAgentEnvironmentLifecycleSnapshot(snapshot) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentPendingReconciliationAuthority(
	pending AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	attempt EffectAttempt,
	inspection ports.AgentEnvironmentInspectReceipt,
	operationAt time.Time,
) error {
	effect := pending.Effect
	_, terminalState := agentEnvironmentLifecycleOutcomeStates(effect.ActionKind)
	original := pending
	original.Token = effect.ExpectedToken
	if validateAgentEnvironmentLifecycleSnapshot(pending) != nil || !effect.NeedsReconciliation() ||
		operationAt.IsZero() || operationAt.Before(pending.RecordedAt) ||
		ports.ValidateAgentEnvironmentInspectReceipt(
			ports.AgentEnvironmentInspectRequest{Subject: pending.Subject}, inspection,
		) != nil || inspection.Token.State != terminalState || inspection.Token.Fence != pending.Token.Fence ||
		inspection.Token.PhysicalToken != pending.Token.PhysicalToken ||
		inspection.Token.Revision == pending.Token.Revision ||
		validateAgentEnvironmentLifecycleClaim(
			original, claim, effect.ActionKind, attempt.StartedAt,
		) != nil || validateClaimedEffect(claim, attempt.StartedAt) != nil ||
		validateEffectAttempt(claim, attempt) != nil || claim.Action.Ref != effect.ActionRef ||
		attempt.Ref != effect.AttemptRef || attempt.ActionFence != effect.ActionFence ||
		claim.DeliveryAttempt != effect.DeliveryAttempt ||
		claim.Action.WorkItemGeneration != effect.WorkItemGeneration ||
		claim.Token != effect.ClaimToken || claim.WorkerRef != effect.WorkerRef ||
		attempt.IdempotencyKey != effect.IdempotencyKey {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentLifecyclePostEffect(state AgentEnvironmentLifecyclePostEffectState) error {
	authoritySnapshot := state.Snapshot
	authoritySnapshot.Token = state.Snapshot.Effect.ExpectedToken
	if state.ExpectedRevision == 0 || state.Snapshot.Revision != state.ExpectedRevision+1 ||
		state.Snapshot.Effect.AttemptRef != state.Attempt.Ref ||
		state.Snapshot.Effect.OutcomeToken != state.Snapshot.Token ||
		validateAgentEnvironmentLifecycleClaim(
			authoritySnapshot, state.Claim, state.Claim.Action.Kind, state.Attempt.StartedAt,
		) != nil ||
		validateAgentEnvironmentLifecycleSnapshot(state.Snapshot) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if state.Snapshot.Effect.NeedsReconciliation() {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if state.EffectReceipt == nil || state.Snapshot.Effect.EffectReceiptRef != state.EffectReceipt.Ref ||
		validateEffectReceipt(state.Claim, state.Attempt, *state.EffectReceipt) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	wantConsumption := consumptionReceipt(state.Claim, ActionConsumedCompleted, "", state.OperationAt)
	wantConsumption.EffectReceiptRef = state.EffectReceipt.Ref
	if state.Snapshot.Effect.DeliveryAttempt != state.Claim.DeliveryAttempt ||
		state.Snapshot.Effect.WorkItemGeneration != state.Claim.Action.WorkItemGeneration ||
		state.Snapshot.Effect.ClaimToken != state.Claim.Token ||
		state.Snapshot.Effect.WorkerRef != state.Claim.WorkerRef ||
		state.ConsumptionReceipt != wantConsumption ||
		!state.Snapshot.RecordedAt.Equal(state.ConsumptionReceipt.ConsumedAt) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if state.Claim.Action.Kind == ActionPreserveAgentEnvironment {
		if state.PreservationFact == nil ||
			state.Snapshot.Preservation.ApplicationReceiptRef != state.PreservationFact.Ref {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	} else if state.PreservationFact != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validAgentEnvironmentPreservation(binding ports.AgentPreservationBinding) bool {
	return validApplicationRef(binding.ApplicationReceiptRef) &&
		validApplicationRef(binding.PhysicalManifest.ManifestRef) &&
		validEffectDigest(binding.PhysicalManifest.ManifestSHA256)
}

func agentEnvironmentLifecycleAction(state ports.AgentEnvironmentLifecycleState) (ActionKind, bool) {
	switch state {
	case ports.AgentEnvironmentActive:
		return ActionQuiesceAgent, true
	case ports.AgentEnvironmentQuiesced:
		return ActionPreserveAgentEnvironment, true
	case ports.AgentEnvironmentPreserved:
		return ActionCloseAgentEnvironment, true
	default:
		return "", false
	}
}

func agentEnvironmentLifecyclePendingState(kind ActionKind) ports.AgentEnvironmentLifecycleState {
	pending, _ := agentEnvironmentLifecycleOutcomeStates(kind)
	return pending
}

func agentEnvironmentLifecycleOutcomeStates(kind ActionKind) (
	ports.AgentEnvironmentLifecycleState,
	ports.AgentEnvironmentLifecycleState,
) {
	switch kind {
	case ActionQuiesceAgent:
		return ports.AgentEnvironmentQuiescing, ports.AgentEnvironmentQuiesced
	case ActionPreserveAgentEnvironment:
		return ports.AgentEnvironmentPreserving, ports.AgentEnvironmentPreserved
	case ActionCloseAgentEnvironment:
		return ports.AgentEnvironmentClosing, ports.AgentEnvironmentClosed
	default:
		return "", ""
	}
}

func agentEnvironmentLifecycleEffectStatus(kind ActionKind) (EffectStatus, bool) {
	switch kind {
	case ActionQuiesceAgent:
		return EffectStatusQuiesced, true
	case ActionPreserveAgentEnvironment:
		return EffectStatusPreserved, true
	case ActionCloseAgentEnvironment:
		return EffectStatusClosed, true
	default:
		return "", false
	}
}

func agentEnvironmentLifecycleRequiresPreservation(state ports.AgentEnvironmentLifecycleState) bool {
	return state == ports.AgentEnvironmentPreserved || state == ports.AgentEnvironmentClosing ||
		state == ports.AgentEnvironmentClosed
}
