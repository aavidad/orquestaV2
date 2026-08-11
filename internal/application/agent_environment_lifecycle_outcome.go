package application

import (
	"time"

	"orquesta/internal/ports"
)

func RecordAgentEnvironmentQuiesceOutcome(
	prepared AgentEnvironmentLifecyclePreEffectState,
	receipt ports.AgentQuiesceReceipt,
	operationAt time.Time,
) (AgentEnvironmentLifecycleOutcome, error) {
	request := ports.AgentQuiesceRequest{
		Subject: prepared.Snapshot.Subject, ExpectedToken: prepared.Snapshot.Effect.ExpectedToken,
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}
	if ports.ValidateAgentQuiesceReceipt(request, receipt) != nil {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return recordAgentEnvironmentLifecycleOutcome(prepared, receipt.NextToken,
		receipt.ReceiptRef, receipt.ConfirmedAt, nil, ports.AgentPhysicalPreservationBinding{}, operationAt)
}

func RecordAgentEnvironmentPreserveOutcome(
	prepared AgentEnvironmentLifecyclePreEffectState,
	receipt ports.AgentPreserveReceipt,
	preservation *ComprobantePreservacionEntornoAgente,
	record GoalRecord,
	operationAt time.Time,
) (AgentEnvironmentLifecycleOutcome, error) {
	request := ports.AgentPreserveRequest{
		Subject: prepared.Snapshot.Subject, ExpectedToken: prepared.Snapshot.Effect.ExpectedToken,
		IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}
	if ports.ValidateAgentPreserveReceipt(request, receipt) != nil {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	var preservationFact *ComprobantePreservacionEntornoAgente
	if receipt.NextToken.State == ports.AgentEnvironmentPreserved {
		if preservation == nil ||
			validateAgentEnvironmentPreservationFact(*preservation, receipt, record, operationAt) != nil {
			return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
		}
		copy := *preservation
		preservationFact = &copy
	} else if preservation != nil {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return recordAgentEnvironmentLifecycleOutcome(prepared, receipt.NextToken,
		receipt.ReceiptRef, receipt.ConfirmedAt, preservationFact,
		ports.AgentPhysicalPreservationBinding{
			ManifestRef: receipt.Manifest.Ref, ManifestSHA256: receipt.Manifest.SHA256,
		}, operationAt)
}

func RecordAgentEnvironmentCloseOutcome(
	prepared AgentEnvironmentLifecyclePreEffectState,
	receipt ports.AgentCloseReceipt,
	operationAt time.Time,
) (AgentEnvironmentLifecycleOutcome, error) {
	request := ports.AgentCloseRequest{
		Subject: prepared.Snapshot.Subject, ExpectedToken: prepared.Snapshot.Effect.ExpectedToken,
		Preservation: prepared.Snapshot.Preservation, IdempotencyKey: prepared.Attempt.IdempotencyKey,
	}
	if !validAgentEnvironmentPreservation(prepared.Snapshot.Preservation) ||
		ports.ValidateAgentCloseReceipt(request, receipt) != nil {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return recordAgentEnvironmentLifecycleOutcome(prepared, receipt.NextToken,
		receipt.ReceiptRef, receipt.ConfirmedAt, nil, ports.AgentPhysicalPreservationBinding{}, operationAt)
}

// RecordAgentEnvironmentQuiesceReconciliationOutcome resolves the durable
// attempted frontier after an absent response or non-persisted pending reply.
// The typed receipt remains bound to the original ExpectedToken; Inspect only
// corroborates the now-terminal token.
func RecordAgentEnvironmentQuiesceReconciliationOutcome(
	pending AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	attempt EffectAttempt,
	inspection ports.AgentEnvironmentInspectReceipt,
	receipt ports.AgentQuiesceReceipt,
	operationAt time.Time,
) (AgentEnvironmentLifecyclePostEffectState, error) {
	request := ports.AgentQuiesceRequest{
		Subject: pending.Subject, ExpectedToken: pending.Effect.ExpectedToken,
		IdempotencyKey: attempt.IdempotencyKey,
	}
	if ports.ValidateAgentQuiesceReceipt(request, receipt) != nil ||
		receipt.NextToken != inspection.Token || receipt.NextToken.State != ports.AgentEnvironmentQuiesced {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return recordAgentEnvironmentLifecycleReconciliationOutcome(
		pending, claim, attempt, inspection, receipt.ReceiptRef, receipt.ConfirmedAt,
		nil, ports.AgentPhysicalPreservationBinding{}, operationAt,
	)
}

func RecordAgentEnvironmentPreserveReconciliationOutcome(
	pending AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	attempt EffectAttempt,
	inspection ports.AgentEnvironmentInspectReceipt,
	receipt ports.AgentPreserveReceipt,
	preservation *ComprobantePreservacionEntornoAgente,
	record GoalRecord,
	operationAt time.Time,
) (AgentEnvironmentLifecyclePostEffectState, error) {
	request := ports.AgentPreserveRequest{
		Subject: pending.Subject, ExpectedToken: pending.Effect.ExpectedToken,
		IdempotencyKey: attempt.IdempotencyKey,
	}
	if ports.ValidateAgentPreserveReceipt(request, receipt) != nil ||
		receipt.NextToken != inspection.Token || receipt.NextToken.State != ports.AgentEnvironmentPreserved ||
		preservation == nil ||
		validateAgentEnvironmentPreservationFact(*preservation, receipt, record, operationAt) != nil {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	copy := *preservation
	return recordAgentEnvironmentLifecycleReconciliationOutcome(
		pending, claim, attempt, inspection, receipt.ReceiptRef, receipt.ConfirmedAt,
		&copy, ports.AgentPhysicalPreservationBinding{
			ManifestRef: receipt.Manifest.Ref, ManifestSHA256: receipt.Manifest.SHA256,
		}, operationAt,
	)
}

func RecordAgentEnvironmentCloseReconciliationOutcome(
	pending AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	attempt EffectAttempt,
	inspection ports.AgentEnvironmentInspectReceipt,
	receipt ports.AgentCloseReceipt,
	operationAt time.Time,
) (AgentEnvironmentLifecyclePostEffectState, error) {
	request := ports.AgentCloseRequest{
		Subject: pending.Subject, ExpectedToken: pending.Effect.ExpectedToken,
		Preservation: pending.Preservation, IdempotencyKey: attempt.IdempotencyKey,
	}
	if !validAgentEnvironmentPreservation(pending.Preservation) ||
		ports.ValidateAgentCloseReceipt(request, receipt) != nil ||
		receipt.NextToken != inspection.Token || receipt.NextToken.State != ports.AgentEnvironmentClosed {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return recordAgentEnvironmentLifecycleReconciliationOutcome(
		pending, claim, attempt, inspection, receipt.ReceiptRef, receipt.ConfirmedAt,
		nil, ports.AgentPhysicalPreservationBinding{}, operationAt,
	)
}

func recordAgentEnvironmentLifecycleReconciliationOutcome(
	pending AgentEnvironmentLifecycleSnapshot,
	claim ActionClaim,
	attempt EffectAttempt,
	inspection ports.AgentEnvironmentInspectReceipt,
	physicalReceiptRef string,
	confirmedAt time.Time,
	preservation *ComprobantePreservacionEntornoAgente,
	physicalManifest ports.AgentPhysicalPreservationBinding,
	operationAt time.Time,
) (AgentEnvironmentLifecyclePostEffectState, error) {
	if validateAgentEnvironmentPendingReconciliationAuthority(
		pending, claim, attempt, inspection, operationAt,
	) != nil || operationAt.Before(confirmedAt) {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	status, ok := agentEnvironmentLifecycleEffectStatus(claim.Action.Kind)
	if !ok {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	effectReceipt, err := effectReceipt(
		claim, attempt, physicalReceiptRef, status, unknownUsage(), confirmedAt,
	)
	if err != nil {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	next := pending
	next.Revision++
	next.Token = inspection.Token
	next.Effect.OutcomeToken = inspection.Token
	next.Effect.PhysicalReceipt = physicalReceiptRef
	next.Effect.EffectReceiptRef = effectReceipt.Ref
	next.RecordedAt = operationAt.UTC()
	state := AgentEnvironmentLifecyclePostEffectState{
		Claim: claim, ExpectedRevision: pending.Revision, Attempt: attempt,
		Snapshot: next, EffectReceipt: &effectReceipt, OperationAt: operationAt.UTC(),
	}
	state.ConsumptionReceipt = consumptionReceipt(claim, ActionConsumedCompleted, "", operationAt)
	state.ConsumptionReceipt.EffectReceiptRef = effectReceipt.Ref
	if preservation != nil {
		state.Snapshot.Preservation = ports.AgentPreservationBinding{
			ApplicationReceiptRef: preservation.Ref,
			PhysicalManifest:      physicalManifest,
		}
		state.Snapshot.PreservationReceiptRef = physicalReceiptRef
		state.PreservationFact = preservation
	} else if physicalManifest != (ports.AgentPhysicalPreservationBinding{}) {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	if validateAgentEnvironmentLifecyclePostEffect(state) != nil {
		return AgentEnvironmentLifecyclePostEffectState{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return state, nil
}

func recordAgentEnvironmentLifecycleOutcome(
	prepared AgentEnvironmentLifecyclePreEffectState,
	nextToken ports.AgentEnvironmentLifecycleToken,
	physicalReceiptRef string,
	confirmedAt time.Time,
	preservation *ComprobantePreservacionEntornoAgente,
	physicalManifest ports.AgentPhysicalPreservationBinding,
	operationAt time.Time,
) (AgentEnvironmentLifecycleOutcome, error) {
	if validateAgentEnvironmentLifecyclePreEffect(prepared) != nil || operationAt.IsZero() ||
		operationAt.Before(prepared.OperationAt) {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	pending, terminal := agentEnvironmentLifecycleOutcomeStates(prepared.Claim.Action.Kind)
	if nextToken.State != pending && nextToken.State != terminal {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	if nextToken.State == pending {
		if physicalReceiptRef != "" || !confirmedAt.IsZero() || preservation != nil ||
			physicalManifest != (ports.AgentPhysicalPreservationBinding{}) {
			return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
		}
		reconciliation, err := AgentEnvironmentLifecycleReconciliationRequest(prepared.Snapshot)
		if err != nil {
			return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
		}
		return AgentEnvironmentLifecycleOutcome{Reconciliation: &reconciliation}, nil
	}
	next := prepared.Snapshot
	next.Revision++
	next.Token = nextToken
	next.Effect.OutcomeToken = nextToken
	next.RecordedAt = operationAt.UTC()
	state := AgentEnvironmentLifecyclePostEffectState{
		Claim: prepared.Claim, ExpectedRevision: prepared.Snapshot.Revision,
		Attempt: prepared.Attempt, Snapshot: next, OperationAt: operationAt.UTC(),
	}
	status, ok := agentEnvironmentLifecycleEffectStatus(prepared.Claim.Action.Kind)
	if !ok || operationAt.Before(confirmedAt) {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	effectReceipt, err := effectReceipt(
		prepared.Claim, prepared.Attempt, physicalReceiptRef, status, unknownUsage(), confirmedAt,
	)
	if err != nil {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	next.Effect.PhysicalReceipt = physicalReceiptRef
	next.Effect.EffectReceiptRef = effectReceipt.Ref
	state.Snapshot = next
	state.EffectReceipt = &effectReceipt
	state.ConsumptionReceipt = consumptionReceipt(prepared.Claim, ActionConsumedCompleted, "", operationAt)
	state.ConsumptionReceipt.EffectReceiptRef = effectReceipt.Ref
	if preservation != nil {
		state.Snapshot.Preservation = ports.AgentPreservationBinding{
			ApplicationReceiptRef: preservation.Ref,
			PhysicalManifest:      physicalManifest,
		}
		state.Snapshot.PreservationReceiptRef = physicalReceiptRef
		state.PreservationFact = preservation
	} else if physicalManifest != (ports.AgentPhysicalPreservationBinding{}) {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	if validateAgentEnvironmentLifecyclePostEffect(state) != nil {
		return AgentEnvironmentLifecycleOutcome{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return AgentEnvironmentLifecycleOutcome{Terminal: &state}, nil
}

func validateAgentEnvironmentPreservationFact(
	preservation ComprobantePreservacionEntornoAgente,
	receipt ports.AgentPreserveReceipt,
	record GoalRecord,
	operationAt time.Time,
) error {
	if !agentEnvironmentPreservationMatchesSubject(preservation, receipt.Subject) ||
		ValidarCausalidadPreservacionEntornoAgente(preservation, record) != nil ||
		preservation.Resultado.ComprobanteRef != receipt.ReceiptRef ||
		preservation.ManifiestoFisicoRef != receipt.Manifest.Ref ||
		preservation.ManifiestoFisicoDigest != receipt.Manifest.SHA256 ||
		preservation.Resultado.SelladoEn != receipt.Manifest.SealedAt ||
		preservation.Resultado.PreservadoEn != receipt.ConfirmedAt ||
		operationAt.Before(preservation.RegistradoEn) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func agentEnvironmentPreservationMatchesSubject(
	preservation ComprobantePreservacionEntornoAgente,
	subject ports.AgentEnvironmentLifecycleSubject,
) bool {
	return preservation.ObjetivoRef == subject.GoalRef && preservation.ItemRef == subject.WorkItemRef &&
		preservation.EjecucionRef == subject.ExecutionRef &&
		preservation.Resultado.EjecucionRef == subject.ExecutionRef &&
		preservation.Resultado.IntentoEjecucion == subject.ExecutionAttempt &&
		preservation.Resultado.IdentidadExterna == subject.ExternalRef
}
