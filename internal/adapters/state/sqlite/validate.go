package sqlite

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

const maxSQLiteInteger = uint64(1<<63 - 1)

func validateCreateState(state application.CreateGoalState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.RequestedBy.String() == "" || state.AuthorizationReceipt.Ref() == "" {
		return errors.New("sqlite.request_identity_invalid")
	}
	snapshot := state.Goal.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil {
		return err
	}
	principal := state.AuthorizationReceipt.Decision().Request().Principal()
	if identity.ValidatePrincipal(principal) != nil || principal.Ref != state.RequestedBy ||
		principal.ActorRef != state.Goal.Actor() || principal.ActorRef != state.Goal.AppSpec().ConfirmedBy() {
		return errors.New("sqlite.create_principal_binding_invalid")
	}
	if len(snapshot.WorkItems) == 0 || snapshot.State != goal.GoalStateRunning {
		return errors.New("sqlite.create_lifecycle_invalid")
	}
	if snapshot.AppSpec.Generation != 1 || snapshot.AppSpec.ParentRef != "" || snapshot.AppSpec.ParentHash != "" {
		return errors.New("sqlite.create_app_spec_invalid")
	}
	items := make(map[string]goal.WorkItemSnapshot, len(snapshot.WorkItems))
	for _, item := range snapshot.WorkItems {
		items[item.Ref] = item
	}
	ready := readyWorkItemRefs(state.Goal)
	if len(state.Executions) == 0 || len(state.Executions) != len(state.Actions) ||
		len(state.Executions) != len(ready) || len(state.Events) == 0 {
		return errors.New("sqlite.create_schedule_invalid")
	}
	actions := make(map[goal.ExecutionRef]application.ActionRecord, len(state.Actions))
	for _, action := range state.Actions {
		if err := validateAction(action); err != nil || action.Kind != application.ActionLaunchAgent {
			return errors.New("sqlite.create_action_invalid")
		}
		if _, duplicate := actions[action.ExecutionRef]; duplicate {
			return errors.New("sqlite.create_action_duplicate")
		}
		actions[action.ExecutionRef] = action
	}
	executionRefs := make(map[goal.ExecutionRef]struct{}, len(state.Executions))
	executionItems := make(map[goal.WorkItemRef]struct{}, len(state.Executions))
	for _, execution := range state.Executions {
		if err := validateExecution(execution); err != nil {
			return err
		}
		item, ok := items[execution.WorkItemRef.String()]
		action, actionOK := actions[execution.Ref]
		_, isReady := ready[execution.WorkItemRef]
		if _, duplicate := executionRefs[execution.Ref]; duplicate {
			return errors.New("sqlite.create_execution_duplicate")
		}
		if _, duplicate := executionItems[execution.WorkItemRef]; duplicate {
			return errors.New("sqlite.create_execution_item_duplicate")
		}
		if execution.State != application.ExecutionQueued || !ok || !isReady || item.State != goal.WorkItemStatePending ||
			execution.GoalRef.String() != snapshot.Ref || !actionOK ||
			execution.AttemptNo != 1 || execution.PlanGeneration != state.Goal.PlanGeneration() ||
			execution.AppSpecGeneration != state.Goal.AppSpec().Generation() || execution.SpecHash != state.Goal.SpecHash() ||
			action.PlanGeneration != state.Goal.PlanGeneration() || action.WorkItemGeneration != item.Revision ||
			!actionMatches(action, snapshot.Ref, item.Ref, execution.Ref.String()) {
			return errors.New("sqlite.create_execution_scope_invalid")
		}
		executionRefs[execution.Ref] = struct{}{}
		executionItems[execution.WorkItemRef] = struct{}{}
	}
	for _, event := range state.Events {
		if err := validateEvent(event); err != nil || event.GoalRef.String() != snapshot.Ref {
			return errors.New("sqlite.create_event_invalid")
		}
	}
	return nil
}

func validateAmendState(state application.AmendGoalState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) ||
		state.RequestedBy.String() == "" || state.AuthorizationReceipt.Ref() == "" ||
		state.ProjectRef.String() == "" ||
		state.SourceGoalRef.String() == "" || state.ExpectedSourceRevision == 0 ||
		uint64(state.ExpectedSourceRevision) > maxSQLiteInteger || !validCanonicalHash(state.ExpectedSourceSpecHash) {
		return errors.New("sqlite.amend_request_invalid")
	}
	snapshot := state.Successor.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil {
		return err
	}
	principal := state.AuthorizationReceipt.Decision().Request().Principal()
	if identity.ValidatePrincipal(principal) != nil || principal.Ref != state.RequestedBy ||
		principal.ActorRef != state.Successor.AppSpec().ConfirmedBy() {
		return errors.New("sqlite.amend_principal_binding_invalid")
	}
	parentRef, hasParent := state.Successor.AppSpec().ParentRef()
	if state.Successor.Ref() == state.SourceGoalRef || state.Successor.Project() != state.ProjectRef ||
		state.Successor.State() != goal.GoalStatePending ||
		state.Successor.Revision() != 1 || state.Successor.PlanGeneration() != 0 ||
		state.Successor.WorkItemCount() != 0 || len(snapshot.Phases) != 0 ||
		!hasParent || parentRef.String() == "" ||
		state.Successor.AppSpec().ParentHash() != state.ExpectedSourceSpecHash ||
		uint64(state.Successor.AppSpec().Generation()) > maxSQLiteInteger {
		return errors.New("sqlite.amend_successor_invalid")
	}
	if len(state.Events) == 0 {
		return errors.New("sqlite.amend_events_required")
	}
	seen := make(map[string]struct{}, len(state.Events))
	for _, event := range state.Events {
		if err := validateEvent(event); err != nil || event.GoalRef != state.Successor.Ref() ||
			event.WorkItemRef.String() != "" || event.ExecutionRef.String() != "" {
			return errors.New("sqlite.amend_event_invalid")
		}
		if _, duplicate := seen[event.Ref]; duplicate {
			return errors.New("sqlite.amend_event_duplicate")
		}
		seen[event.Ref] = struct{}{}
	}
	return nil
}

func validateExecution(execution application.ExecutionRecord) error {
	if execution.Ref.String() == "" || execution.GoalRef.String() == "" || execution.WorkItemRef.String() == "" {
		return errors.New("sqlite.execution_ref_invalid")
	}
	if execution.AttemptNo == 0 || execution.AttemptNo > maxSQLiteInteger ||
		execution.MaxExecutionAttempts == 0 || execution.MaxExecutionAttempts > maxSQLiteInteger ||
		execution.AttemptNo > execution.MaxExecutionAttempts || execution.PlanGeneration == 0 ||
		uint64(execution.PlanGeneration) > maxSQLiteInteger || execution.AppSpecGeneration == 0 ||
		uint64(execution.AppSpecGeneration) > maxSQLiteInteger || !validCanonicalHash(execution.SpecHash) {
		return errors.New("sqlite.execution_generation_invalid")
	}
	if (execution.AttemptNo == 1 && execution.ReplacesExecutionRef.String() != "") ||
		(execution.AttemptNo > 1 && execution.ReplacesExecutionRef.String() == "") {
		return errors.New("sqlite.execution_replacement_invalid")
	}
	if !validText(execution.ArtifactMediaType) || !validText(execution.IdempotencyKey) ||
		execution.MaxOutputBytes <= 0 {
		return errors.New("sqlite.execution_policy_invalid")
	}
	if execution.CreatedAt.IsZero() {
		return errors.New("sqlite.execution_time_invalid")
	}
	if !optionalText(execution.ProviderRef) || !optionalText(execution.ModelRef) ||
		!optionalText(execution.AgentRef) || !optionalText(execution.ExternalRef) ||
		!optionalText(execution.FailureCode) {
		return errors.New("sqlite.execution_text_invalid")
	}
	if execution.RecipientMailboxRetired &&
		execution.State != application.ExecutionSucceeded && execution.State != application.ExecutionFailed &&
		execution.State != application.ExecutionStopped &&
		execution.State != application.ExecutionCanceled {
		return errors.New("sqlite.execution_mailbox_retirement_state_invalid")
	}
	if !execution.StartedAt.IsZero() && execution.StartedAt.Before(execution.CreatedAt) {
		return errors.New("sqlite.execution_started_at_invalid")
	}
	if !execution.DeadlineAt.IsZero() &&
		(execution.StartedAt.IsZero() || !execution.DeadlineAt.After(execution.StartedAt)) {
		return errors.New("sqlite.execution_deadline_invalid")
	}
	if !execution.LastObservedAt.IsZero() &&
		(execution.StartedAt.IsZero() || execution.LastObservedAt.Before(execution.StartedAt)) {
		return errors.New("sqlite.execution_observed_at_invalid")
	}
	if !execution.FinishedAt.IsZero() &&
		(execution.FinishedAt.Before(execution.CreatedAt) ||
			(!execution.StartedAt.IsZero() && execution.FinishedAt.Before(execution.StartedAt))) {
		return errors.New("sqlite.execution_finished_at_invalid")
	}
	if !execution.LastObservedAt.IsZero() && !execution.FinishedAt.IsZero() &&
		execution.LastObservedAt.After(execution.FinishedAt) {
		return errors.New("sqlite.execution_observed_after_finish")
	}
	switch execution.State {
	case application.ExecutionQueued:
		if execution.ProviderRef != "" || execution.ModelRef != "" || execution.AgentRef != "" ||
			execution.ExternalRef != "" || !execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
			!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() ||
			!execution.ProviderObservedAt.IsZero() || !execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_queued_fields_invalid")
		}
	case application.ExecutionRunning:
		if !validText(execution.ProviderRef) || !validText(execution.ModelRef) ||
			!validText(execution.AgentRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero() ||
			!execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_running_fields_invalid")
		}
	case application.ExecutionDispatching:
		if execution.ProviderRef != "" || execution.ModelRef != "" || execution.AgentRef != "" ||
			execution.ExternalRef != "" || !execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
			!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() ||
			!execution.ProviderObservedAt.IsZero() || !execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_dispatching_fields_invalid")
		}
	case application.ExecutionSucceeded:
		if !validText(execution.ProviderRef) || !validText(execution.ModelRef) ||
			!validText(execution.AgentRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero() ||
			execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_succeeded_fields_invalid")
		}
	case application.ExecutionFailed:
		providerAccepted := execution.ProviderRef != ""
		if execution.FinishedAt.IsZero() || !validText(execution.FailureCode) ||
			(providerAccepted != (execution.ModelRef != "")) ||
			(providerAccepted != (execution.AgentRef != "")) ||
			(providerAccepted != (execution.ExternalRef != "")) ||
			(providerAccepted && (execution.StartedAt.IsZero() || execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero())) ||
			(!providerAccepted && (!execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
				!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() || !execution.ProviderObservedAt.IsZero())) {
			return errors.New("sqlite.execution_failed_fields_invalid")
		}
	case application.ExecutionCanceled:
		if execution.ProviderRef != "" || execution.ModelRef != "" || execution.AgentRef != "" ||
			execution.ExternalRef != "" || !execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
			!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() ||
			!execution.ProviderObservedAt.IsZero() || execution.FinishedAt.IsZero() {
			return errors.New("sqlite.execution_canceled_fields_invalid")
		}
	case application.ExecutionStopped:
		if !validText(execution.ProviderRef) || !validText(execution.ModelRef) ||
			!validText(execution.AgentRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero() || execution.FinishedAt.IsZero() ||
			!validText(execution.FailureCode) {
			return errors.New("sqlite.execution_stopped_fields_invalid")
		}
	default:
		return errors.New("sqlite.execution_state_invalid")
	}
	return nil
}

func validateArtifactRecord(artifact application.ArtifactRecord) error {
	if artifact.Stored.Ref.String() == "" || !validText(artifact.Stored.Digest) ||
		!validText(artifact.Stored.MediaType) || artifact.Stored.Size < 0 || artifact.CreatedAt.IsZero() ||
		artifact.GoalRef.String() == "" || artifact.WorkItemRef.String() == "" {
		return errors.New("sqlite.artifact_invalid")
	}
	return nil
}

func validateAttestationRecord(attestation application.AttestationRecord) error {
	if attestation.Ref.String() == "" || attestation.GoalRef.String() == "" ||
		attestation.WorkItemRef.String() == "" || attestation.ExecutionRef.String() == "" ||
		attestation.ArtifactRef.String() == "" || !validText(attestation.Policy) ||
		attestation.AcceptedAt.IsZero() {
		return errors.New("sqlite.attestation_invalid")
	}
	return nil
}

// validateGoalRecordConsistency checks only relationships owned by the state
// adapter. Aggregate lifecycle, phase, WorkItem and AppSpec rules remain in
// goal.RestoreGoal, invoked by readGoalRecord before this function.
func validateGoalRecordConsistency(record application.GoalRecord, expectedGoalRef string) error {
	aggregate := record.Goal
	if record.RequestedBy.String() == "" || aggregate.Ref().String() != expectedGoalRef {
		return errors.New("sqlite.goal_record_ref_invalid")
	}
	items := make(map[goal.WorkItemRef]goal.WorkItem, aggregate.WorkItemCount())
	for _, item := range aggregate.WorkItems() {
		items[item.Ref()] = item
	}
	executions := make(map[goal.ExecutionRef]application.ExecutionRecord, len(record.Executions))
	byItem := make(map[goal.WorkItemRef][]application.ExecutionRecord, len(items))
	for _, execution := range record.Executions {
		if err := validateExecution(execution); err != nil {
			return err
		}
		_, found := items[execution.WorkItemRef]
		if !found || execution.GoalRef != aggregate.Ref() {
			return errors.New("sqlite.goal_record_execution_scope_invalid")
		}
		if execution.PlanGeneration > aggregate.PlanGeneration() ||
			execution.AppSpecGeneration != aggregate.AppSpec().Generation() ||
			execution.SpecHash != aggregate.SpecHash() {
			return errors.New("sqlite.goal_record_execution_generation_invalid")
		}
		if _, duplicate := executions[execution.Ref]; duplicate {
			return errors.New("sqlite.goal_record_execution_duplicate")
		}
		executions[execution.Ref] = execution
		byItem[execution.WorkItemRef] = append(byItem[execution.WorkItemRef], execution)
	}
	for _, item := range items {
		if err := validateWorkItemExecutionChain(item, byItem[item.Ref()]); err != nil {
			return err
		}
	}

	artifacts := make(map[goal.ArtifactRef]application.ArtifactRecord, len(record.Artifacts))
	for _, artifact := range record.Artifacts {
		if err := validateArtifactRecord(artifact); err != nil {
			return err
		}
		item, found := items[artifact.WorkItemRef]
		if !found || artifact.GoalRef != aggregate.Ref() || !workItemHasArtifact(item, artifact.Stored.Ref) {
			return errors.New("sqlite.goal_record_artifact_scope_invalid")
		}
		if _, duplicate := artifacts[artifact.Stored.Ref]; duplicate {
			return errors.New("sqlite.goal_record_artifact_duplicate")
		}
		artifacts[artifact.Stored.Ref] = artifact
	}

	attestations := make(map[goal.AttestationRef]application.AttestationRecord, len(record.Attestations))
	for _, attestation := range record.Attestations {
		if err := validateAttestationRecord(attestation); err != nil {
			return err
		}
		item, itemFound := items[attestation.WorkItemRef]
		execution, executionFound := executions[attestation.ExecutionRef]
		artifact, artifactFound := artifacts[attestation.ArtifactRef]
		boundExecution, hasBinding := item.Execution()
		if !itemFound || !executionFound || !artifactFound || !hasBinding ||
			attestation.GoalRef != aggregate.Ref() || boundExecution != attestation.ExecutionRef ||
			execution.WorkItemRef != attestation.WorkItemRef || artifact.WorkItemRef != attestation.WorkItemRef ||
			!workItemHasAttestation(item, attestation.Ref) {
			return errors.New("sqlite.goal_record_attestation_scope_invalid")
		}
		if _, duplicate := attestations[attestation.Ref]; duplicate {
			return errors.New("sqlite.goal_record_attestation_duplicate")
		}
		attestations[attestation.Ref] = attestation
	}
	for _, item := range items {
		for _, ref := range item.Artifacts() {
			artifact, found := artifacts[ref]
			if !found || artifact.WorkItemRef != item.Ref() {
				return errors.New("sqlite.goal_record_item_artifact_invalid")
			}
		}
		for _, ref := range item.Attestations() {
			attestation, found := attestations[ref]
			if !found || attestation.WorkItemRef != item.Ref() {
				return errors.New("sqlite.goal_record_item_attestation_invalid")
			}
		}
	}

	seenActions := make(map[string]struct{}, len(record.ConsumptionReceipts))
	seenTokens := make(map[string]struct{}, len(record.ConsumptionReceipts))
	seenFences := make(map[string]struct{}, len(record.ConsumptionReceipts))
	for _, receipt := range record.ConsumptionReceipts {
		if err := validateConsumptionReceipt(receipt); err != nil {
			return err
		}
		item, itemFound := items[receipt.WorkItemRef]
		execution, executionFound := executions[receipt.ExecutionRef]
		if !itemFound || !executionFound || receipt.GoalRef != aggregate.Ref() ||
			execution.WorkItemRef != receipt.WorkItemRef ||
			receipt.PlanGeneration != execution.PlanGeneration ||
			receipt.WorkItemGeneration > item.Revision() {
			return errors.New("sqlite.goal_record_receipt_scope_invalid")
		}
		fenceScope := receipt.WorkItemRef.String()
		if receipt.Kind == application.ActionDeliverMailbox {
			fenceScope = receipt.MailboxMessageRef.String()
		}
		fenceKey := string(receipt.Kind) + ":" + fenceScope + ":" + fmt.Sprint(receipt.Fence)
		if _, duplicate := seenActions[receipt.ActionRef]; duplicate {
			return errors.New("sqlite.goal_record_receipt_action_duplicate")
		}
		if _, duplicate := seenTokens[receipt.ClaimToken]; duplicate {
			return errors.New("sqlite.goal_record_receipt_token_duplicate")
		}
		if _, duplicate := seenFences[fenceKey]; duplicate {
			return errors.New("sqlite.goal_record_receipt_fence_duplicate")
		}
		seenActions[receipt.ActionRef] = struct{}{}
		seenTokens[receipt.ClaimToken] = struct{}{}
		seenFences[fenceKey] = struct{}{}
	}
	return nil
}

func validateWorkItemExecutionChain(item goal.WorkItem, records []application.ExecutionRecord) error {
	bound, hasBinding := item.Execution()
	if len(records) == 0 {
		if hasBinding {
			return errors.New("sqlite.goal_record_item_execution_invalid")
		}
		return nil
	}
	byAttempt := make(map[uint64]application.ExecutionRecord, len(records))
	for _, execution := range records {
		if _, duplicate := byAttempt[execution.AttemptNo]; duplicate {
			return errors.New("sqlite.goal_record_execution_attempt_duplicate")
		}
		byAttempt[execution.AttemptNo] = execution
	}
	for attempt := uint64(1); attempt <= uint64(len(records)); attempt++ {
		execution, found := byAttempt[attempt]
		if !found {
			return errors.New("sqlite.goal_record_execution_attempt_gap")
		}
		if attempt == 1 {
			if execution.ReplacesExecutionRef.String() != "" {
				return errors.New("sqlite.goal_record_execution_chain_invalid")
			}
			continue
		}
		previous := byAttempt[attempt-1]
		if execution.ReplacesExecutionRef != previous.Ref ||
			(previous.State != application.ExecutionFailed && previous.State != application.ExecutionStopped) ||
			execution.MaxExecutionAttempts != previous.MaxExecutionAttempts ||
			execution.PlanGeneration != previous.PlanGeneration ||
			execution.AppSpecGeneration != previous.AppSpecGeneration || execution.SpecHash != previous.SpecHash ||
			execution.CreatedAt.Before(previous.FinishedAt) {
			return errors.New("sqlite.goal_record_execution_chain_invalid")
		}
	}
	latest := byAttempt[uint64(len(records))]
	for attempt := uint64(1); attempt < latest.AttemptNo; attempt++ {
		if byAttempt[attempt].State != application.ExecutionFailed && byAttempt[attempt].State != application.ExecutionStopped {
			return errors.New("sqlite.goal_record_historical_execution_active")
		}
	}
	switch item.State() {
	case goal.WorkItemStatePending:
		if hasBinding || latest.State != application.ExecutionQueued {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	case goal.WorkItemStateRunning:
		if !hasBinding || bound != latest.Ref ||
			(latest.State != application.ExecutionQueued && latest.State != application.ExecutionDispatching &&
				latest.State != application.ExecutionRunning) {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	case goal.WorkItemStateSucceeded:
		if !hasBinding || bound != latest.Ref || latest.State != application.ExecutionSucceeded {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	case goal.WorkItemStateFailed:
		if !hasBinding || bound != latest.Ref || latest.State != application.ExecutionFailed {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	case goal.WorkItemStateSkipped:
		return errors.New("sqlite.goal_record_skipped_execution_invalid")
	case goal.WorkItemStateInterrupted:
		if !hasBinding || bound != latest.Ref ||
			(latest.State != application.ExecutionFailed && latest.State != application.ExecutionStopped) {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	case goal.WorkItemStateCanceled:
		if hasBinding && bound != latest.Ref {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
		if latest.State != application.ExecutionCanceled && latest.State != application.ExecutionStopped &&
			latest.State != application.ExecutionSucceeded && latest.State != application.ExecutionFailed {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	case goal.WorkItemStateSuperseded:
		if latest.State != application.ExecutionCanceled && latest.State != application.ExecutionStopped &&
			latest.State != application.ExecutionFailed {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
	default:
		return errors.New("sqlite.goal_record_execution_binding_invalid")
	}
	return nil
}

func validateConsumptionReceipt(receipt application.ActionConsumptionReceipt) error {
	if !validText(receipt.ActionRef) || receipt.GoalRef.String() == "" ||
		receipt.WorkItemRef.String() == "" || receipt.ExecutionRef.String() == "" ||
		receipt.PlanGeneration == 0 || uint64(receipt.PlanGeneration) > maxSQLiteInteger ||
		receipt.WorkItemGeneration == 0 || uint64(receipt.WorkItemGeneration) > maxSQLiteInteger ||
		receipt.Fence == 0 || receipt.Fence > maxSQLiteInteger || receipt.DeliveryAttempt == 0 ||
		receipt.DeliveryAttempt > maxSQLiteInteger || !validText(receipt.ClaimToken) ||
		!validText(receipt.WorkerRef) || !optionalText(receipt.ErrorCode) || receipt.ConsumedAt.IsZero() {
		return errors.New("sqlite.consumption_receipt_invalid")
	}
	switch receipt.Kind {
	case application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionStopAgent:
		if receipt.MailboxMessageRef.String() != "" {
			return errors.New("sqlite.consumption_receipt_mailbox_unexpected")
		}
	case application.ActionDeliverMailbox:
		if receipt.MailboxMessageRef.String() == "" || receipt.Outcome != application.ActionConsumedCompleted || receipt.ErrorCode != "" {
			return errors.New("sqlite.consumption_receipt_mailbox_invalid")
		}
	default:
		return errors.New("sqlite.consumption_receipt_kind_invalid")
	}
	switch receipt.Outcome {
	case application.ActionConsumedCompleted:
	case application.ActionConsumedQuarantined:
		if !validText(receipt.ErrorCode) {
			return errors.New("sqlite.consumption_receipt_error_required")
		}
	default:
		return errors.New("sqlite.consumption_receipt_outcome_invalid")
	}
	hasEffect := receipt.EffectReceiptRef != "" || receipt.EffectStatus != "" || !receipt.EffectConfirmedAt.IsZero()
	if hasEffect {
		if receipt.Kind != application.ActionStopAgent || receipt.Outcome != application.ActionConsumedCompleted ||
			receipt.ErrorCode != "" || !validText(receipt.EffectReceiptRef) ||
			!receipt.EffectConfirmedAt.Equal(receipt.ConsumedAt) ||
			(receipt.EffectStatus != string(ports.AgentStopped) &&
				receipt.EffectStatus != string(ports.AgentStopAlreadyStopped) &&
				receipt.EffectStatus != string(ports.AgentStopAlreadyCompleted) &&
				receipt.EffectStatus != string(ports.AgentStopAlreadyFailed)) {
			return errors.New("sqlite.consumption_receipt_effect_invalid")
		}
	} else if receipt.Kind == application.ActionStopAgent &&
		receipt.Outcome == application.ActionConsumedCompleted && receipt.ErrorCode == "" {
		return errors.New("sqlite.consumption_receipt_effect_missing")
	}
	return nil
}

func workItemHasArtifact(item goal.WorkItem, expected goal.ArtifactRef) bool {
	for _, ref := range item.Artifacts() {
		if ref == expected {
			return true
		}
	}
	return false
}

func workItemHasAttestation(item goal.WorkItem, expected goal.AttestationRef) bool {
	for _, ref := range item.Attestations() {
		if ref == expected {
			return true
		}
	}
	return false
}

func validateAction(action application.ActionRecord) error {
	if !validText(action.Ref) || action.GoalRef.String() == "" || action.WorkItemRef.String() == "" ||
		action.ExecutionRef.String() == "" || action.PlanGeneration == 0 ||
		uint64(action.PlanGeneration) > maxSQLiteInteger || action.WorkItemGeneration == 0 ||
		uint64(action.WorkItemGeneration) > maxSQLiteInteger || action.AvailableAt.IsZero() {
		return errors.New("sqlite.action_invalid")
	}
	switch action.Kind {
	case application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionDeliverMailbox:
		if action.ControlRef != "" {
			return errors.New("sqlite.action_control_unexpected")
		}
		return nil
	case application.ActionStopAgent:
		if !validText(action.ControlRef) {
			return errors.New("sqlite.action_control_required")
		}
		return nil
	default:
		return errors.New("sqlite.action_kind_invalid")
	}
}

func validateClaim(claim application.ActionClaim) error {
	if !validText(claim.Token) || !validText(claim.WorkerRef) || claim.DeliveryAttempt == 0 ||
		claim.DeliveryAttempt > maxSQLiteInteger || claim.Fence == 0 ||
		claim.Fence > maxSQLiteInteger || claim.LeaseUntil.IsZero() {
		return errors.New("sqlite.claim_invalid")
	}
	if err := validateAction(claim.Action); err != nil {
		return errors.New("sqlite.claim_action_invalid")
	}
	return nil
}

func validateEvent(event application.EventRecord) error {
	if !validText(event.Ref) || !validText(event.Kind) || event.GoalRef.String() == "" || event.OccurredAt.IsZero() {
		return errors.New("sqlite.event_invalid")
	}
	if event.ExecutionRef.String() != "" && event.WorkItemRef.String() == "" {
		return errors.New("sqlite.event_execution_without_work_item")
	}
	return nil
}

func validateLaunchPrepared(state application.LaunchPreparedState) error {
	if err := validateClaim(state.Claim); err != nil {
		return err
	}
	if state.ExpectedGoalRevision == 0 || uint64(state.ExpectedGoalRevision) > maxSQLiteInteger {
		return errors.New("sqlite.expected_revision_invalid")
	}
	if _, err := goal.RestoreGoal(state.Goal.Snapshot()); err != nil {
		return err
	}
	preparedItem, found := state.Goal.WorkItem(state.Claim.Action.WorkItemRef)
	bound, hasBinding := preparedItem.Execution()
	if !found || !hasBinding || bound != state.Execution.Ref ||
		state.Goal.State() != goal.GoalStateRunning || preparedItem.State() != goal.WorkItemStateRunning {
		return errors.New("sqlite.launch_prepare_item_invalid")
	}
	initialStart := state.Goal.Revision() == state.ExpectedGoalRevision+1 &&
		preparedItem.Revision() > state.Claim.Action.WorkItemGeneration
	replacementStart := state.Goal.Revision() == state.ExpectedGoalRevision &&
		preparedItem.Revision() >= state.Claim.Action.WorkItemGeneration
	if (!initialStart && !replacementStart) || uint64(state.Goal.Revision()) > maxSQLiteInteger ||
		uint64(preparedItem.Revision()) > maxSQLiteInteger {
		return errors.New("sqlite.launch_prepare_revision_invalid")
	}
	if err := validateExecution(state.Execution); err != nil {
		return err
	}
	if state.Claim.Action.Kind != application.ActionLaunchAgent ||
		state.Execution.State != application.ExecutionDispatching ||
		state.Execution.Ref != state.Claim.Action.ExecutionRef ||
		state.Execution.GoalRef != state.Goal.Ref() ||
		state.Execution.WorkItemRef != preparedItem.Ref() ||
		state.Claim.Action.GoalRef != state.Goal.Ref() ||
		state.Execution.PlanGeneration != state.Claim.Action.PlanGeneration ||
		state.Execution.PlanGeneration == 0 || state.Execution.PlanGeneration > state.Goal.PlanGeneration() ||
		state.Execution.AppSpecGeneration != state.Goal.AppSpec().Generation() ||
		state.Execution.SpecHash != state.Goal.SpecHash() {
		return errors.New("sqlite.launch_prepare_invalid")
	}
	if err := validateEvent(state.Event); err != nil ||
		state.Event.Kind != "execution.dispatching" ||
		!eventMatches(state.Event, state.Goal.Ref().String(), preparedItem.Ref().String(), state.Execution.Ref.String()) {
		return errors.New("sqlite.launch_prepare_event_invalid")
	}
	return nil
}

func validateLaunchAccepted(state application.LaunchAcceptedState) error {
	if err := validateClaim(state.Claim); err != nil {
		return err
	}
	if state.Claim.Action.Kind != application.ActionLaunchAgent || state.NextAction.Kind != application.ActionObserveAgent ||
		state.Execution.State != application.ExecutionRunning || state.Execution.Ref != state.Claim.Action.ExecutionRef ||
		state.Execution.GoalRef != state.Claim.Action.GoalRef || state.Execution.WorkItemRef != state.Claim.Action.WorkItemRef ||
		state.Execution.PlanGeneration != state.Claim.Action.PlanGeneration {
		return errors.New("sqlite.launch_action_kind_invalid")
	}
	if err := validateExecution(state.Execution); err != nil {
		return err
	}
	if err := validateAction(state.NextAction); err != nil {
		return err
	}
	if !actionMatches(
		state.NextAction,
		state.Execution.GoalRef.String(),
		state.Execution.WorkItemRef.String(),
		state.Execution.Ref.String(),
	) {
		return errors.New("sqlite.next_action_scope_mismatch")
	}
	if state.NextAction.PlanGeneration != state.Execution.PlanGeneration ||
		state.NextAction.WorkItemGeneration < state.Claim.Action.WorkItemGeneration {
		return errors.New("sqlite.next_action_generation_mismatch")
	}
	if err := validateEvent(state.Event); err != nil {
		return err
	}
	if state.Event.Kind != "execution.accepted" {
		return errors.New("sqlite.launch_accepted_event_kind_invalid")
	}
	if !eventMatches(state.Event, state.Execution.GoalRef.String(), state.Execution.WorkItemRef.String(), state.Execution.Ref.String()) {
		return errors.New("sqlite.event_scope_mismatch")
	}
	return nil
}

func validateRequeued(state application.ActionRequeuedState) error {
	if err := validateClaim(state.Claim); err != nil {
		return err
	}
	if err := validateExecution(state.Execution); err != nil {
		return err
	}
	if state.Execution.Ref != state.Claim.Action.ExecutionRef ||
		state.Execution.GoalRef != state.Claim.Action.GoalRef ||
		state.Execution.WorkItemRef != state.Claim.Action.WorkItemRef || state.AvailableAt.IsZero() {
		return errors.New("sqlite.requeue_scope_invalid")
	}
	switch state.Claim.Action.Kind {
	case application.ActionLaunchAgent:
		if state.Execution.State != application.ExecutionQueued && state.Execution.State != application.ExecutionDispatching {
			return errors.New("sqlite.requeue_launch_state_invalid")
		}
	case application.ActionObserveAgent:
		if state.Execution.State != application.ExecutionRunning {
			return errors.New("sqlite.requeue_observe_state_invalid")
		}
	case application.ActionStopAgent:
		if state.Execution.State != application.ExecutionDispatching &&
			state.Execution.State != application.ExecutionRunning {
			return errors.New("sqlite.requeue_stop_state_invalid")
		}
	default:
		return errors.New("sqlite.requeue_action_kind_invalid")
	}
	if state.ErrorCode != "" && !validText(state.ErrorCode) {
		return errors.New("sqlite.error_code_invalid")
	}
	return nil
}

func validateQuarantined(state application.ActionQuarantinedState) error {
	if err := validateClaim(state.Claim); err != nil {
		return err
	}
	if !validText(state.ErrorCode) {
		return errors.New("sqlite.quarantine_code_invalid")
	}
	if err := validateEvent(state.Event); err != nil {
		return err
	}
	if state.Event.Kind != "action.quarantined" {
		return errors.New("sqlite.quarantine_event_kind_invalid")
	}
	if !eventMatches(
		state.Event,
		state.Claim.Action.GoalRef.String(),
		state.Claim.Action.WorkItemRef.String(),
		state.Claim.Action.ExecutionRef.String(),
	) {
		return errors.New("sqlite.quarantine_event_scope_mismatch")
	}
	return nil
}

func validateExecutionReplaced(state application.ExecutionReplacedState) error {
	if err := validateClaim(state.Claim); err != nil {
		return err
	}
	if state.ExpectedGoalRevision == 0 || state.ExpectedItemRevision == 0 ||
		uint64(state.ExpectedGoalRevision) > maxSQLiteInteger || uint64(state.ExpectedItemRevision) > maxSQLiteInteger {
		return errors.New("sqlite.expected_revision_invalid")
	}
	if _, err := goal.RestoreGoal(state.Goal.Snapshot()); err != nil {
		return err
	}
	item, found := state.Goal.WorkItem(state.Claim.Action.WorkItemRef)
	bound, hasBinding := item.Execution()
	if !found || !hasBinding || bound != state.ReplacementExecution.Ref ||
		state.Goal.Ref() != state.Claim.Action.GoalRef || state.Goal.Revision() <= state.ExpectedGoalRevision ||
		item.Revision() <= state.ExpectedItemRevision || item.State() != goal.WorkItemStateRunning {
		return errors.New("sqlite.execution_replacement_goal_invalid")
	}
	if err := validateExecution(state.FailedExecution); err != nil {
		return err
	}
	if err := validateExecution(state.ReplacementExecution); err != nil {
		return err
	}
	failed := state.FailedExecution
	replacement := state.ReplacementExecution
	if failed.Ref != state.Claim.Action.ExecutionRef || failed.GoalRef != state.Goal.Ref() ||
		failed.WorkItemRef != item.Ref() || failed.State != application.ExecutionFailed ||
		replacement.GoalRef != failed.GoalRef || replacement.WorkItemRef != failed.WorkItemRef ||
		replacement.State != application.ExecutionQueued || replacement.AttemptNo != failed.AttemptNo+1 ||
		replacement.ReplacesExecutionRef != failed.Ref ||
		replacement.MaxExecutionAttempts != failed.MaxExecutionAttempts ||
		replacement.PlanGeneration != failed.PlanGeneration ||
		replacement.AppSpecGeneration != failed.AppSpecGeneration || replacement.SpecHash != failed.SpecHash ||
		replacement.ArtifactMediaType != failed.ArtifactMediaType ||
		replacement.MaxOutputBytes != failed.MaxOutputBytes ||
		replacement.CreatedAt.Before(failed.FinishedAt) ||
		!validText(state.ErrorCode) || state.ErrorCode != failed.FailureCode || failed.FinishedAt.IsZero() {
		return errors.New("sqlite.execution_replacement_invalid")
	}
	if err := validateAction(state.NextAction); err != nil || state.NextAction.Kind != application.ActionLaunchAgent ||
		!actionMatches(state.NextAction, state.Goal.Ref().String(), item.Ref().String(), replacement.Ref.String()) ||
		state.NextAction.PlanGeneration != replacement.PlanGeneration ||
		state.NextAction.PlanGeneration == 0 ||
		state.NextAction.PlanGeneration > state.Goal.PlanGeneration() ||
		state.NextAction.WorkItemGeneration != item.Revision() {
		return errors.New("sqlite.execution_replacement_action_invalid")
	}
	if err := validateExactMutationEvents(
		state.Events,
		state.Goal,
		[]eventSemantic{
			newEventSemantic("execution.failed", failed.WorkItemRef, failed.Ref),
			newEventSemantic("execution.queued", replacement.WorkItemRef, replacement.Ref),
		},
		false,
	); err != nil {
		return err
	}
	return nil
}

func validateExecutionInterrupted(state application.ExecutionInterruptedState) error {
	item, err := validateGoalMutation(
		state.Claim, state.ExpectedGoalRevision, state.ExpectedItemRevision,
		state.Goal, state.Execution,
	)
	if err != nil {
		return err
	}
	if item.State() != goal.WorkItemStateInterrupted ||
		state.Execution.State != application.ExecutionFailed ||
		!validText(state.Execution.FailureCode) || state.Execution.FinishedAt.IsZero() {
		return errors.New("sqlite.execution_interrupted_invalid")
	}
	if err := validateScheduled(state.Goal, state.NewExecutions, state.NewActions); err != nil {
		return err
	}
	requiredEvents := []eventSemantic{
		newEventSemantic("execution.failed", state.Execution.WorkItemRef, state.Execution.Ref),
		newEventSemantic("work_item.interrupted", state.Execution.WorkItemRef, state.Execution.Ref),
	}
	for _, execution := range state.NewExecutions {
		requiredEvents = append(requiredEvents,
			newEventSemantic("execution.queued", execution.WorkItemRef, execution.Ref))
	}
	return validateExactMutationEvents(state.Events, state.Goal, requiredEvents, false)
}

func validateSucceeded(state application.GoalSucceededState) (goal.WorkItem, error) {
	item, err := validateGoalMutation(
		state.Claim,
		state.ExpectedGoalRevision,
		state.ExpectedItemRevision,
		state.Goal,
		state.Execution,
	)
	if err != nil {
		return goal.WorkItem{}, err
	}
	if (state.Goal.State() != goal.GoalStateRunning && state.Goal.State() != goal.GoalStateSucceeded &&
		state.Goal.State() != goal.GoalStateFailed) ||
		state.Execution.State != application.ExecutionSucceeded {
		return goal.WorkItem{}, errors.New("sqlite.succeeded_state_invalid")
	}
	if item.State() != goal.WorkItemStateSucceeded {
		return goal.WorkItem{}, errors.New("sqlite.succeeded_item_state_invalid")
	}
	if state.Execution.FinishedAt.IsZero() {
		return goal.WorkItem{}, errors.New("sqlite.execution_finished_at_required")
	}
	if err := validateArtifactRecord(state.Artifact); err != nil ||
		state.Artifact.GoalRef != state.Goal.Ref() || state.Artifact.WorkItemRef != item.Ref() {
		return goal.WorkItem{}, errors.New("sqlite.artifact_invalid")
	}
	if err := validateAttestationRecord(state.Attestation); err != nil || state.Attestation.GoalRef != state.Goal.Ref() ||
		state.Attestation.WorkItemRef != item.Ref() || state.Attestation.ExecutionRef != state.Execution.Ref ||
		state.Attestation.ArtifactRef != state.Artifact.Stored.Ref {
		return goal.WorkItem{}, errors.New("sqlite.attestation_invalid")
	}
	if err := validateScheduled(state.Goal, state.NewExecutions, state.NewActions); err != nil {
		return goal.WorkItem{}, err
	}
	requiredEvents := []eventSemantic{
		newEventSemantic("work_item.succeeded", state.Execution.WorkItemRef, state.Execution.Ref),
	}
	for _, execution := range state.NewExecutions {
		requiredEvents = append(requiredEvents, newEventSemantic("execution.queued", execution.WorkItemRef, execution.Ref))
	}
	if terminal, required := terminalGoalEvent(state.Goal); required {
		requiredEvents = append(requiredEvents, terminal)
	}
	if err := validateExactMutationEvents(
		state.Events,
		state.Goal,
		requiredEvents,
		false,
	); err != nil {
		return goal.WorkItem{}, err
	}
	return item, nil
}

func validateFailed(state application.GoalFailedState) (goal.WorkItem, error) {
	item, err := validateGoalMutation(
		state.Claim,
		state.ExpectedGoalRevision,
		state.ExpectedItemRevision,
		state.Goal,
		state.Execution,
	)
	if err != nil {
		return goal.WorkItem{}, err
	}
	if (state.Goal.State() != goal.GoalStateRunning && state.Goal.State() != goal.GoalStateFailed) ||
		state.Execution.State != application.ExecutionFailed ||
		!validText(state.Execution.FailureCode) || state.Execution.FinishedAt.IsZero() {
		return goal.WorkItem{}, errors.New("sqlite.failed_state_invalid")
	}
	if item.State() != goal.WorkItemStateFailed {
		return goal.WorkItem{}, errors.New("sqlite.failed_item_state_invalid")
	}
	if err := validateScheduled(state.Goal, state.NewExecutions, state.NewActions); err != nil {
		return goal.WorkItem{}, err
	}
	requiredEvents := []eventSemantic{
		newEventSemantic("work_item.failed", state.Execution.WorkItemRef, state.Execution.Ref),
	}
	for _, execution := range state.NewExecutions {
		requiredEvents = append(requiredEvents, newEventSemantic("execution.queued", execution.WorkItemRef, execution.Ref))
	}
	if state.Goal.State() == goal.GoalStateFailed {
		requiredEvents = append(requiredEvents, eventSemantic{kind: "goal.failed"})
	}
	if err := validateExactMutationEvents(
		state.Events,
		state.Goal,
		requiredEvents,
		true,
	); err != nil {
		return goal.WorkItem{}, err
	}
	return item, nil
}

func validateGoalMutation(
	claim application.ActionClaim,
	expectedGoal goal.Revision,
	expectedItem goal.Revision,
	aggregate goal.Goal,
	execution application.ExecutionRecord,
) (goal.WorkItem, error) {
	if err := validateClaim(claim); err != nil {
		return goal.WorkItem{}, err
	}
	if expectedGoal == 0 || expectedItem == 0 || uint64(expectedGoal) > maxSQLiteInteger ||
		uint64(expectedItem) > maxSQLiteInteger {
		return goal.WorkItem{}, errors.New("sqlite.expected_revision_invalid")
	}
	snapshot := aggregate.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil {
		return goal.WorkItem{}, err
	}
	if len(snapshot.WorkItems) == 0 || aggregate.Revision() <= expectedGoal ||
		uint64(aggregate.Revision()) > maxSQLiteInteger {
		return goal.WorkItem{}, errors.New("sqlite.goal_revision_invalid")
	}
	item, found := aggregate.WorkItem(claim.Action.WorkItemRef)
	if !found || item.Revision() <= expectedItem || uint64(item.Revision()) > maxSQLiteInteger {
		return goal.WorkItem{}, errors.New("sqlite.work_item_revision_invalid")
	}
	if err := validateExecution(execution); err != nil {
		return goal.WorkItem{}, err
	}
	if execution.Ref != claim.Action.ExecutionRef || execution.GoalRef != aggregate.Ref() ||
		execution.WorkItemRef != item.Ref() || aggregate.Ref() != claim.Action.GoalRef ||
		execution.PlanGeneration == 0 || execution.PlanGeneration > aggregate.PlanGeneration() ||
		claim.Action.PlanGeneration != execution.PlanGeneration ||
		execution.AppSpecGeneration != aggregate.AppSpec().Generation() || execution.SpecHash != aggregate.SpecHash() {
		return goal.WorkItem{}, errors.New("sqlite.mutation_scope_invalid")
	}
	return item, nil
}

type eventSemantic struct {
	kind         string
	workItemRef  string
	executionRef string
}

func newEventSemantic(kind string, workItemRef goal.WorkItemRef, executionRef goal.ExecutionRef) eventSemantic {
	return eventSemantic{kind: kind, workItemRef: workItemRef.String(), executionRef: executionRef.String()}
}

func terminalGoalEvent(aggregate goal.Goal) (eventSemantic, bool) {
	switch aggregate.State() {
	case goal.GoalStateSucceeded:
		return eventSemantic{kind: "goal.succeeded"}, true
	case goal.GoalStateFailed:
		return eventSemantic{kind: "goal.failed"}, true
	case goal.GoalStateCanceled:
		return eventSemantic{kind: "goal.canceled"}, true
	default:
		return eventSemantic{}, false
	}
}

func validateExactMutationEvents(
	events []application.EventRecord,
	aggregate goal.Goal,
	requiredEvents []eventSemantic,
	allowSkipped bool,
) error {
	if len(events) == 0 {
		return errors.New("sqlite.events_required")
	}
	required := make(map[eventSemantic]struct{}, len(requiredEvents))
	for _, expected := range requiredEvents {
		if !validText(expected.kind) {
			return errors.New("sqlite.event_semantic_invalid")
		}
		if _, duplicate := required[expected]; duplicate {
			return errors.New("sqlite.event_semantic_duplicate")
		}
		required[expected] = struct{}{}
	}
	seen := make(map[string]struct{}, len(events))
	seenSemantics := make(map[eventSemantic]struct{}, len(events))
	for _, event := range events {
		if err := validateEvent(event); err != nil {
			return err
		}
		if event.GoalRef != aggregate.Ref() {
			return errors.New("sqlite.event_scope_mismatch")
		}
		if _, duplicate := seen[event.Ref]; duplicate {
			return errors.New("sqlite.event_duplicate")
		}
		seen[event.Ref] = struct{}{}
		semantic := eventSemantic{
			kind: event.Kind, workItemRef: event.WorkItemRef.String(), executionRef: event.ExecutionRef.String(),
		}
		if _, duplicate := seenSemantics[semantic]; duplicate {
			return errors.New("sqlite.event_semantic_duplicate")
		}
		seenSemantics[semantic] = struct{}{}
		if _, expected := required[semantic]; expected {
			delete(required, semantic)
			continue
		}
		if allowSkipped && semantic.kind == "work_item.skipped" && semantic.workItemRef != "" && semantic.executionRef == "" {
			item, found := aggregate.WorkItem(event.WorkItemRef)
			if !found || item.State() != goal.WorkItemStateSkipped {
				return errors.New("sqlite.event_work_item_scope_invalid")
			}
			continue
		}
		return errors.New("sqlite.event_semantic_invalid")
	}
	if len(required) != 0 {
		return errors.New("sqlite.event_semantic_missing")
	}
	return nil
}

func validateScheduled(aggregate goal.Goal, executions []application.ExecutionRecord, actions []application.ActionRecord) error {
	if len(executions) != len(actions) {
		return errors.New("sqlite.scheduled_cardinality_invalid")
	}
	byExecution := make(map[goal.ExecutionRef]application.ActionRecord, len(actions))
	for _, action := range actions {
		if err := validateAction(action); err != nil || action.Kind != application.ActionLaunchAgent {
			return errors.New("sqlite.scheduled_action_invalid")
		}
		if _, duplicate := byExecution[action.ExecutionRef]; duplicate {
			return errors.New("sqlite.scheduled_action_duplicate")
		}
		byExecution[action.ExecutionRef] = action
	}
	ready := readyWorkItemRefs(aggregate)
	seenExecutions := make(map[goal.ExecutionRef]struct{}, len(executions))
	seenItems := make(map[goal.WorkItemRef]struct{}, len(executions))
	for _, execution := range executions {
		item, found := aggregate.WorkItem(execution.WorkItemRef)
		action, actionFound := byExecution[execution.Ref]
		if err := validateExecution(execution); err != nil {
			return err
		}
		_, isReady := ready[execution.WorkItemRef]
		if _, duplicate := seenExecutions[execution.Ref]; duplicate {
			return errors.New("sqlite.scheduled_execution_duplicate")
		}
		if _, duplicate := seenItems[execution.WorkItemRef]; duplicate {
			return errors.New("sqlite.scheduled_execution_item_duplicate")
		}
		if !found || !isReady || item.State() != goal.WorkItemStatePending || execution.State != application.ExecutionQueued ||
			execution.GoalRef != aggregate.Ref() || !actionFound ||
			execution.AttemptNo != 1 || execution.PlanGeneration != aggregate.PlanGeneration() ||
			execution.AppSpecGeneration != aggregate.AppSpec().Generation() || execution.SpecHash != aggregate.SpecHash() ||
			action.PlanGeneration != aggregate.PlanGeneration() || action.WorkItemGeneration != item.Revision() ||
			!actionMatches(action, aggregate.Ref().String(), item.Ref().String(), execution.Ref.String()) {
			return errors.New("sqlite.scheduled_scope_invalid")
		}
		seenExecutions[execution.Ref] = struct{}{}
		seenItems[execution.WorkItemRef] = struct{}{}
	}
	return nil
}

func readyWorkItemRefs(aggregate goal.Goal) map[goal.WorkItemRef]struct{} {
	ready := aggregate.ReadyWorkItems()
	result := make(map[goal.WorkItemRef]struct{}, len(ready))
	for _, item := range ready {
		result[item.Ref()] = struct{}{}
	}
	return result
}

func actionMatches(action application.ActionRecord, goalRef, itemRef, executionRef string) bool {
	return action.GoalRef.String() == goalRef && action.WorkItemRef.String() == itemRef &&
		action.ExecutionRef.String() == executionRef
}

func validCanonicalHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func eventMatches(event application.EventRecord, goalRef, itemRef, executionRef string) bool {
	return event.GoalRef.String() == goalRef && event.WorkItemRef.String() == itemRef &&
		event.ExecutionRef.String() == executionRef
}

func validText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value
}

func optionalText(value string) bool {
	return value == "" || validText(value)
}

func safeLeaseUntil(now time.Time, duration time.Duration) (time.Time, error) {
	if now.IsZero() || duration <= 0 {
		return time.Time{}, errors.New("sqlite.claim_time_invalid")
	}
	leaseUntil := now.Add(duration)
	if !leaseUntil.After(now) {
		return time.Time{}, fmt.Errorf("sqlite.claim_time_overflow")
	}
	return leaseUntil.Round(0).UTC(), nil
}
