package application

import (
	"time"

	"orquesta/internal/ports"
)

type agentEnvironmentHistoricalEffect struct {
	intent      EffectIntent
	approval    EffectApproval
	attempt     EffectAttempt
	receipt     *EffectReceipt
	consumption *ActionConsumptionReceipt
	frontier    time.Time
}

func validateAgentEnvironmentLifecycleRecord(
	snapshot AgentEnvironmentLifecycleSnapshot,
	launchReceipt ports.AgentLaunchReceipt,
	record GoalRecord,
	preservation *ComprobantePreservacionEntornoAgente,
) error {
	execution, err := exactAgentEnvironmentExecution(record, snapshot)
	if err != nil {
		return err
	}
	if validateAgentEnvironmentEffectFactClosure(record, execution) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	launch, err := exactAgentEnvironmentLaunchHistory(record, execution, snapshot, launchReceipt)
	if err != nil || validateAgentEnvironmentLifecycleHistory(record, snapshot, launch) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if snapshot.Preservation == (ports.AgentPreservationBinding{}) {
		if preservation != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	if preservation == nil || preservation.Ref != snapshot.Preservation.ApplicationReceiptRef ||
		preservation.Resultado.ComprobanteRef != snapshot.PreservationReceiptRef ||
		preservation.ManifiestoFisicoRef != snapshot.Preservation.PhysicalManifest.ManifestRef ||
		preservation.ManifiestoFisicoDigest != snapshot.Preservation.PhysicalManifest.ManifestSHA256 ||
		!agentEnvironmentPreservationMatchesSubject(*preservation, snapshot.Subject) ||
		ValidarCausalidadPreservacionEntornoAgente(*preservation, record) != nil ||
		validateAgentEnvironmentPreservationHistory(snapshot, record, *preservation) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentPreservationHistory(
	snapshot AgentEnvironmentLifecycleSnapshot,
	record GoalRecord,
	preservation ComprobantePreservacionEntornoAgente,
) error {
	var selected EffectReceipt
	matches := 0
	for _, receipt := range record.EffectReceipts {
		intent, found := exactAgentEnvironmentIntentByRef(record.EffectIntents, receipt.IntentRef)
		if !found || intent.Kind != EffectKindAgentPreserve ||
			intent.Subject.ExecutionRef != snapshot.Subject.ExecutionRef {
			continue
		}
		selected = receipt
		matches++
	}
	if matches != 1 || selected.ExternalRef != snapshot.PreservationReceiptRef ||
		!selected.ConfirmedAt.Equal(preservation.Resultado.PreservadoEn) ||
		preservation.RegistradoEn.Before(selected.ConfirmedAt) ||
		preservation.Resultado.SelladoEn.After(selected.ConfirmedAt) ||
		snapshot.RecordedAt.Before(preservation.RegistradoEn) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

// Every effect fact scoped to this Execution must retain its canonical parent.
// Unrelated valid effects are allowed; orphan attempts and receipts are not.
func validateAgentEnvironmentEffectFactClosure(record GoalRecord, execution ExecutionRecord) error {
	for _, intent := range record.EffectIntents {
		if intent.Subject.ExecutionRef == execution.Ref && ValidateEffectIntent(intent) != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	for _, approval := range record.EffectApprovals {
		if approval.Subject.ExecutionRef != execution.Ref {
			continue
		}
		intent, found := exactAgentEnvironmentIntentByRef(record.EffectIntents, approval.IntentRef)
		if !found || ValidateEffectApproval(intent, approval) != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	for _, attempt := range record.EffectAttempts {
		if attempt.Subject.ExecutionRef != execution.Ref {
			continue
		}
		intent, intentFound := exactAgentEnvironmentIntentByRef(record.EffectIntents, attempt.IntentRef)
		approval, approvalFound := exactHistoricalApproval(record.EffectApprovals, attempt.ApprovalRef)
		if !intentFound || !approvalFound ||
			validateAgentEnvironmentHistoricalAuthority(intent, approval, attempt) != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	for _, receipt := range record.EffectReceipts {
		attempt, attemptFound := exactAgentEnvironmentAttemptByRef(record.EffectAttempts, receipt.AttemptRef)
		related := receipt.Subject.ExecutionRef == execution.Ref ||
			(attemptFound && attempt.Subject.ExecutionRef == execution.Ref)
		if !related {
			continue
		}
		intent, intentFound := exactAgentEnvironmentIntentByRef(record.EffectIntents, receipt.IntentRef)
		approval, approvalFound := exactHistoricalApproval(record.EffectApprovals, receipt.ApprovalRef)
		if !attemptFound || !intentFound || !approvalFound ||
			validateAgentEnvironmentHistoricalAuthority(intent, approval, attempt) != nil ||
			validateAgentEnvironmentHistoricalReceipt(intent, approval, attempt, receipt) != nil {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	return nil
}

func exactAgentEnvironmentIntentByRef(records []EffectIntent, ref string) (EffectIntent, bool) {
	var selected EffectIntent
	matches := 0
	for _, record := range records {
		if record.Ref == ref {
			selected = record
			matches++
		}
	}
	return selected, matches == 1
}

func exactAgentEnvironmentAttemptByRef(records []EffectAttempt, ref string) (EffectAttempt, bool) {
	var selected EffectAttempt
	matches := 0
	for _, record := range records {
		if record.Ref == ref {
			selected = record
			matches++
		}
	}
	return selected, matches == 1
}

func exactAgentEnvironmentExecution(
	record GoalRecord,
	snapshot AgentEnvironmentLifecycleSnapshot,
) (ExecutionRecord, error) {
	if record.Goal.Ref() != snapshot.Subject.GoalRef || record.Goal.Project().String() == "" ||
		record.Goal.Actor().String() == "" || record.Goal.SpecHash() != snapshot.Subject.SpecHash {
		return ExecutionRecord{}, errAgentEnvironmentLifecycleStateInvalid
	}
	var selected ExecutionRecord
	matches := 0
	for _, execution := range record.Executions {
		if execution.Ref == snapshot.Subject.ExecutionRef {
			selected = execution
			matches++
		}
	}
	if matches != 1 || selected.GoalRef != snapshot.Subject.GoalRef ||
		selected.WorkItemRef != snapshot.Subject.WorkItemRef ||
		selected.AttemptNo != snapshot.Subject.ExecutionAttempt ||
		selected.PlanGeneration != snapshot.Subject.PlanGeneration ||
		selected.AppSpecGeneration != snapshot.Subject.AppSpecGeneration ||
		selected.SpecHash != snapshot.Subject.SpecHash || selected.ProviderRef != snapshot.Subject.ProviderRef ||
		selected.ModelRef != snapshot.Subject.ModelRef || selected.AgentRef != snapshot.Subject.AgentRef ||
		selected.ExternalRef != snapshot.Subject.ExternalRef || !validApplicationRef(selected.EffectIntentRef) ||
		!validApplicationRef(selected.LaunchReceiptRef) || selected.StartedAt.IsZero() {
		return ExecutionRecord{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return selected, nil
}

func exactAgentEnvironmentLaunchHistory(
	record GoalRecord,
	execution ExecutionRecord,
	snapshot AgentEnvironmentLifecycleSnapshot,
	physical ports.AgentLaunchReceipt,
) (agentEnvironmentHistoricalEffect, error) {
	intents := make([]EffectIntent, 0, 1)
	for _, intent := range record.EffectIntents {
		isLaunch := intent.Kind == EffectKindAgentLaunch || intent.ActionKind == ActionLaunchAgent
		if intent.Ref == execution.EffectIntentRef ||
			(isLaunch && intent.Subject.ExecutionRef == execution.Ref) {
			intents = append(intents, intent)
		}
	}
	if len(intents) != 1 || intents[0].Ref != execution.EffectIntentRef ||
		intents[0].Kind != EffectKindAgentLaunch || intents[0].ActionKind != ActionLaunchAgent ||
		!agentEnvironmentLaunchIntentMatchesExecution(intents[0], execution, record) {
		return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
	}
	history, err := exactAgentEnvironmentEffectHistory(record, intents[0], true, execution.LaunchReceiptRef)
	if err != nil || history.receipt == nil || history.receipt.Status != EffectStatusAccepted ||
		history.receipt.ExternalRef != physical.ReceiptRef || physical.ReceiptRef != snapshot.LaunchReceiptRef ||
		physical.AcceptedAt.Before(history.attempt.StartedAt) ||
		history.receipt.ConfirmedAt.Before(physical.AcceptedAt) ||
		!execution.ProviderAcceptedAt.Equal(physical.AcceptedAt) ||
		execution.IdempotencyKey != physical.IdempotencyKey ||
		execution.RequierePreservacionEntorno != physical.RequierePreservacionEntorno ||
		execution.ProviderRef != physical.ProviderRef || execution.ModelRef != physical.ModelRef ||
		execution.AgentRef != physical.AgentRef || execution.ExternalRef != physical.ExternalRef {
		return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
	}
	history.frontier = latestAgentEnvironmentLifecycleTime(
		history.receipt.ConfirmedAt,
		physical.AcceptedAt,
		execution.StartedAt,
	)
	return history, nil
}

func latestAgentEnvironmentLifecycleTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest
}

func agentEnvironmentLaunchIntentMatchesExecution(
	intent EffectIntent,
	execution ExecutionRecord,
	record GoalRecord,
) bool {
	return intent.Subject.ProjectRef == record.Goal.Project() &&
		intent.Subject.GoalRef == execution.GoalRef &&
		intent.Subject.WorkItemRef == execution.WorkItemRef &&
		intent.Subject.ExecutionRef == execution.Ref &&
		intent.Subject.PlanGeneration == execution.PlanGeneration &&
		intent.Subject.AppSpecGeneration == execution.AppSpecGeneration &&
		intent.Subject.SpecHash == execution.SpecHash &&
		intent.Subject.ActorRef == record.Goal.Actor() &&
		intent.IdempotencyKey == execution.IdempotencyKey &&
		record.Goal.AppSpec().Generation() == execution.AppSpecGeneration
}

func validateAgentEnvironmentLifecycleHistory(
	record GoalRecord,
	snapshot AgentEnvironmentLifecycleSnapshot,
	launch agentEnvironmentHistoricalEffect,
) error {
	wantKinds := expectedAgentEnvironmentLifecycleKinds(snapshot)
	intents := make(map[ActionKind][]EffectIntent, len(wantKinds)+1)
	intentCount := 0
	for _, intent := range record.EffectIntents {
		isLifecycle := agentEnvironmentLifecycleEffectKind(intent.Kind) ||
			agentEnvironmentLifecycleActionKind(intent.ActionKind)
		related := intent.Subject.ExecutionRef == snapshot.Subject.ExecutionRef ||
			(!snapshot.Effect.IsEmpty() && intent.ActionRef == snapshot.Effect.ActionRef)
		if !isLifecycle || !related {
			continue
		}
		intents[intent.ActionKind] = append(intents[intent.ActionKind], intent)
		intentCount++
	}
	if validateAgentEnvironmentLifecyclePhysicalFrontier(record, snapshot, wantKinds) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	previousAt := launch.frontier
	var current agentEnvironmentHistoricalEffect
	for index, kind := range wantKinds {
		candidates := intents[kind]
		if len(candidates) != 1 || candidates[0].Kind != lifecycleEffectKindForAction(kind) ||
			candidates[0].Subject != launch.intent.Subject {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		receiptRequired := index < len(wantKinds)-1 || !snapshot.Effect.NeedsReconciliation()
		history, err := exactAgentEnvironmentEffectHistory(record, candidates[0], receiptRequired, "")
		if err != nil {
			return err
		}
		if history.attempt.StartedAt.Before(previousAt) {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		consumption, err := exactAgentEnvironmentLifecycleConsumption(record, history, receiptRequired)
		if err != nil {
			return err
		}
		history.consumption = consumption
		if consumption != nil {
			previousAt = consumption.ConsumedAt
		}
		if index == len(wantKinds)-1 {
			current = history
		}
	}
	allowedIntentCount := len(wantKinds)
	for _, plannedKind := range agentEnvironmentLifecycleKindsAfter(wantKinds) {
		planned := intents[plannedKind]
		switch len(planned) {
		case 0:
		case 1:
			allowedIntentCount++
			if planned[0].Kind != lifecycleEffectKindForAction(plannedKind) ||
				planned[0].Subject != launch.intent.Subject ||
				validateAgentEnvironmentPlannedEffect(record, planned[0]) != nil {
				return errAgentEnvironmentLifecycleStateInvalid
			}
		default:
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	if intentCount != allowedIntentCount {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if len(wantKinds) == 0 {
		if snapshot.Revision != 1 || snapshot.RecordedAt.Before(previousAt) {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	return validateAgentEnvironmentCurrentFrontier(snapshot, current, len(wantKinds))
}

func exactAgentEnvironmentLifecycleConsumption(
	record GoalRecord,
	history agentEnvironmentHistoricalEffect,
	required bool,
) (*ActionConsumptionReceipt, error) {
	candidates := make([]ActionConsumptionReceipt, 0, 1)
	for _, consumption := range record.ConsumptionReceipts {
		if consumption.ActionRef == history.intent.ActionRef ||
			(history.receipt != nil && consumption.EffectReceiptRef == history.receipt.Ref) {
			candidates = append(candidates, consumption)
		}
	}
	if !required {
		if len(candidates) != 0 {
			return nil, errAgentEnvironmentLifecycleStateInvalid
		}
		return nil, nil
	}
	if history.receipt == nil || len(candidates) != 1 {
		return nil, errAgentEnvironmentLifecycleStateInvalid
	}
	consumption := candidates[0]
	item, itemFound := record.Goal.WorkItem(history.intent.Subject.WorkItemRef)
	if !itemFound || consumption.ActionRef != history.intent.ActionRef || consumption.Kind != history.intent.ActionKind ||
		consumption.GoalRef != history.intent.Subject.GoalRef ||
		consumption.WorkItemRef != history.intent.Subject.WorkItemRef ||
		consumption.ExecutionRef != history.intent.Subject.ExecutionRef ||
		consumption.PlanGeneration != history.intent.Subject.PlanGeneration ||
		consumption.WorkItemGeneration != item.Revision() || consumption.ChangeRef.String() != "" ||
		consumption.MailboxMessageRef.String() != "" || consumption.Fence != history.attempt.ActionFence ||
		consumption.DeliveryAttempt == 0 ||
		consumption.ClaimToken == "" || consumption.WorkerRef != history.attempt.WorkerRef ||
		consumption.Outcome != ActionConsumedCompleted || consumption.ErrorCode != "" ||
		consumption.EffectReceiptRef != history.receipt.Ref ||
		history.attempt.Ref != "effect-attempt:"+history.intent.ActionRef+":"+consumption.ClaimToken ||
		consumption.ConsumedAt.Before(history.receipt.ConfirmedAt) {
		return nil, errAgentEnvironmentLifecycleStateInvalid
	}
	return &consumption, nil
}

func validateAgentEnvironmentLifecyclePhysicalFrontier(
	record GoalRecord,
	snapshot AgentEnvironmentLifecycleSnapshot,
	wantKinds []ActionKind,
) error {
	attemptCount := 0
	for _, attempt := range record.EffectAttempts {
		intent, found := exactAgentEnvironmentIntentByRef(record.EffectIntents, attempt.IntentRef)
		if !found || (!agentEnvironmentLifecycleEffectKind(intent.Kind) &&
			!agentEnvironmentLifecycleActionKind(intent.ActionKind)) {
			continue
		}
		if attempt.Subject.ExecutionRef == snapshot.Subject.ExecutionRef ||
			(!snapshot.Effect.IsEmpty() && attempt.ActionRef == snapshot.Effect.ActionRef) {
			attemptCount++
		}
	}
	wantReceiptCount := len(wantKinds)
	if snapshot.Effect.NeedsReconciliation() {
		wantReceiptCount--
	}
	receiptCount := 0
	for _, receipt := range record.EffectReceipts {
		intent, found := exactAgentEnvironmentIntentByRef(record.EffectIntents, receipt.IntentRef)
		if !found || (!agentEnvironmentLifecycleEffectKind(intent.Kind) &&
			!agentEnvironmentLifecycleActionKind(intent.ActionKind)) {
			continue
		}
		if receipt.Subject.ExecutionRef == snapshot.Subject.ExecutionRef ||
			(!snapshot.Effect.IsEmpty() && receipt.ActionRef == snapshot.Effect.ActionRef) {
			receiptCount++
		}
	}
	consumptionCount := 0
	for _, consumption := range record.ConsumptionReceipts {
		if !agentEnvironmentLifecycleActionKind(consumption.Kind) {
			continue
		}
		if consumption.ExecutionRef == snapshot.Subject.ExecutionRef ||
			(!snapshot.Effect.IsEmpty() && consumption.ActionRef == snapshot.Effect.ActionRef) {
			consumptionCount++
		}
	}
	if attemptCount != len(wantKinds) || receiptCount != wantReceiptCount ||
		consumptionCount != wantReceiptCount {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentPlannedEffect(record GoalRecord, intent EffectIntent) error {
	if ValidateEffectIntent(intent) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	approvals := make([]EffectApproval, 0, 1)
	for _, approval := range record.EffectApprovals {
		if approval.IntentRef == intent.Ref {
			approvals = append(approvals, approval)
		}
	}
	if len(approvals) != 1 || ValidateEffectApproval(intent, approvals[0]) != nil ||
		approvals[0].Decision != EffectApproved {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	for _, attempt := range record.EffectAttempts {
		if attempt.IntentRef == intent.Ref || attempt.ActionRef == intent.ActionRef {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	for _, receipt := range record.EffectReceipts {
		if receipt.IntentRef == intent.Ref || receipt.ActionRef == intent.ActionRef {
			return errAgentEnvironmentLifecycleStateInvalid
		}
	}
	return nil
}

func agentEnvironmentLifecycleKindsAfter(physical []ActionKind) []ActionKind {
	all := []ActionKind{
		ActionQuiesceAgent,
		ActionPreserveAgentEnvironment,
		ActionCloseAgentEnvironment,
	}
	if len(physical) >= len(all) {
		return nil
	}
	return all[len(physical):]
}

func exactAgentEnvironmentEffectHistory(
	record GoalRecord,
	intent EffectIntent,
	receiptRequired bool,
	receiptRef string,
) (agentEnvironmentHistoricalEffect, error) {
	if ValidateEffectIntent(intent) != nil {
		return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
	}
	attempts := make([]EffectAttempt, 0, 1)
	for _, attempt := range record.EffectAttempts {
		if attempt.IntentRef == intent.Ref || attempt.ActionRef == intent.ActionRef {
			attempts = append(attempts, attempt)
		}
	}
	if len(attempts) != 1 {
		return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
	}
	attempt := attempts[0]
	approvals := make([]EffectApproval, 0, 1)
	for _, approval := range record.EffectApprovals {
		if approval.IntentRef == intent.Ref || approval.Ref == attempt.ApprovalRef {
			approvals = append(approvals, approval)
		}
	}
	if len(approvals) != 1 || validateAgentEnvironmentHistoricalAuthority(intent, approvals[0], attempt) != nil {
		return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
	}
	history := agentEnvironmentHistoricalEffect{intent: intent, approval: approvals[0], attempt: attempt}
	receipts := make([]EffectReceipt, 0, 1)
	for _, receipt := range record.EffectReceipts {
		if receipt.IntentRef == intent.Ref || receipt.ActionRef == intent.ActionRef ||
			receipt.AttemptRef == attempt.Ref || (receiptRef != "" && receipt.Ref == receiptRef) {
			receipts = append(receipts, receipt)
		}
	}
	if receiptRequired {
		if len(receipts) != 1 || (receiptRef != "" && receipts[0].Ref != receiptRef) ||
			validateAgentEnvironmentHistoricalReceipt(intent, approvals[0], attempt, receipts[0]) != nil {
			return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
		}
		receipt := receipts[0]
		history.receipt = &receipt
	} else if len(receipts) != 0 {
		return agentEnvironmentHistoricalEffect{}, errAgentEnvironmentLifecycleStateInvalid
	}
	return history, nil
}

func validateAgentEnvironmentHistoricalAuthority(
	intent EffectIntent,
	approval EffectApproval,
	attempt EffectAttempt,
) error {
	if ValidateEffectApproval(intent, approval) != nil || approval.Decision != EffectApproved ||
		validateHistoricalLaunchAttemptIdentity(intent, attempt) != nil ||
		attempt.ApprovalRef != approval.Ref || attempt.StartedAt.Before(intent.CreatedAt) ||
		attempt.StartedAt.Before(approval.DecidedAt) ||
		(approval.Source == EffectApprovalSourceExplicitDecision && !approval.ExpiresAt.After(attempt.StartedAt)) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentHistoricalReceipt(
	intent EffectIntent,
	approval EffectApproval,
	attempt EffectAttempt,
	receipt EffectReceipt,
) error {
	claim := ActionClaim{
		Action: ActionRecord{
			Ref: intent.ActionRef, Kind: intent.ActionKind,
			EffectIntentRef: intent.Ref, EffectIntent: intent,
		},
		EffectApproval: approval,
		Fence:          attempt.ActionFence,
	}
	if receipt.Ref != "effect-receipt:"+intent.Ref || validateEffectReceipt(claim, attempt, receipt) != nil {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func validateAgentEnvironmentCurrentFrontier(
	snapshot AgentEnvironmentLifecycleSnapshot,
	current agentEnvironmentHistoricalEffect,
	stageCount int,
) error {
	effect := snapshot.Effect
	wantRevision := uint64(2 * stageCount)
	if effect.OutcomeToken != (ports.AgentEnvironmentLifecycleToken{}) {
		wantRevision++
	}
	if snapshot.Revision != wantRevision || effect.ActionRef != current.intent.ActionRef ||
		effect.ActionKind != current.intent.ActionKind || effect.AttemptRef != current.attempt.Ref ||
		effect.ActionFence != current.attempt.ActionFence || effect.IdempotencyKey != current.attempt.IdempotencyKey ||
		effect.WorkerRef != current.attempt.WorkerRef ||
		effect.AttemptRef != "effect-attempt:"+effect.ActionRef+":"+effect.ClaimToken ||
		snapshot.RecordedAt.Before(current.attempt.StartedAt) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if current.receipt == nil {
		if current.consumption != nil || !snapshot.RecordedAt.Equal(current.attempt.StartedAt) {
			return errAgentEnvironmentLifecycleStateInvalid
		}
		return nil
	}
	if current.consumption == nil || effect.EffectReceiptRef != current.receipt.Ref ||
		effect.PhysicalReceipt != current.receipt.ExternalRef ||
		effect.DeliveryAttempt != current.consumption.DeliveryAttempt ||
		effect.WorkItemGeneration != current.consumption.WorkItemGeneration ||
		effect.ClaimToken != current.consumption.ClaimToken ||
		effect.WorkerRef != current.consumption.WorkerRef ||
		!snapshot.RecordedAt.Equal(current.consumption.ConsumedAt) {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	if effect.ActionKind == ActionPreserveAgentEnvironment &&
		snapshot.PreservationReceiptRef != current.receipt.ExternalRef {
		return errAgentEnvironmentLifecycleStateInvalid
	}
	return nil
}

func expectedAgentEnvironmentLifecycleKinds(snapshot AgentEnvironmentLifecycleSnapshot) []ActionKind {
	if snapshot.Effect.IsEmpty() {
		return nil
	}
	switch snapshot.Effect.ActionKind {
	case ActionQuiesceAgent:
		return []ActionKind{ActionQuiesceAgent}
	case ActionPreserveAgentEnvironment:
		return []ActionKind{ActionQuiesceAgent, ActionPreserveAgentEnvironment}
	case ActionCloseAgentEnvironment:
		return []ActionKind{ActionQuiesceAgent, ActionPreserveAgentEnvironment, ActionCloseAgentEnvironment}
	default:
		return nil
	}
}

func agentEnvironmentLifecycleEffectKind(kind EffectKind) bool {
	return kind == EffectKindAgentQuiesce || kind == EffectKindAgentPreserve || kind == EffectKindAgentClose
}

func agentEnvironmentLifecycleActionKind(kind ActionKind) bool {
	return kind == ActionQuiesceAgent || kind == ActionPreserveAgentEnvironment || kind == ActionCloseAgentEnvironment
}

func lifecycleEffectKindForAction(kind ActionKind) EffectKind {
	switch kind {
	case ActionQuiesceAgent:
		return EffectKindAgentQuiesce
	case ActionPreserveAgentEnvironment:
		return EffectKindAgentPreserve
	case ActionCloseAgentEnvironment:
		return EffectKindAgentClose
	default:
		return ""
	}
}
