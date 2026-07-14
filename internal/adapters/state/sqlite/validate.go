package sqlite

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

const maxSQLiteInteger = uint64(1<<63 - 1)

func validateCreateState(state application.CreateGoalState) error {
	if !validText(state.RequestRef) || !validText(state.RequestFingerprint) {
		return errors.New("sqlite.request_identity_invalid")
	}
	snapshot := state.Goal.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil {
		return err
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
		state.ActorRef.String() == "" || state.ProjectRef.String() == "" ||
		state.SourceGoalRef.String() == "" || state.ExpectedSourceRevision == 0 ||
		uint64(state.ExpectedSourceRevision) > maxSQLiteInteger || !validCanonicalHash(state.ExpectedSourceSpecHash) {
		return errors.New("sqlite.amend_request_invalid")
	}
	snapshot := state.Successor.Snapshot()
	if _, err := goal.RestoreGoal(snapshot); err != nil {
		return err
	}
	parentRef, hasParent := state.Successor.AppSpec().ParentRef()
	if state.Successor.Ref() == state.SourceGoalRef || state.Successor.Actor() != state.ActorRef ||
		state.Successor.Project() != state.ProjectRef || state.Successor.State() != goal.GoalStatePending ||
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
	if !validText(execution.ArtifactMediaType) || !validText(execution.IdempotencyKey) ||
		execution.MaxOutputBytes <= 0 || execution.MaxAttempts == 0 || execution.MaxAttempts > maxSQLiteInteger {
		return errors.New("sqlite.execution_policy_invalid")
	}
	if execution.CreatedAt.IsZero() {
		return errors.New("sqlite.execution_time_invalid")
	}
	if !optionalText(execution.ProviderRef) || !optionalText(execution.ExternalRef) ||
		!optionalText(execution.FailureCode) {
		return errors.New("sqlite.execution_text_invalid")
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
		if execution.ProviderRef != "" || execution.ExternalRef != "" || !execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
			!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() ||
			!execution.ProviderObservedAt.IsZero() || !execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_queued_fields_invalid")
		}
	case application.ExecutionRunning:
		if !validText(execution.ProviderRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero() ||
			!execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_running_fields_invalid")
		}
	case application.ExecutionDispatching:
		if execution.ProviderRef != "" || execution.ExternalRef != "" || !execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
			!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() ||
			!execution.ProviderObservedAt.IsZero() || !execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_dispatching_fields_invalid")
		}
	case application.ExecutionSucceeded:
		if !validText(execution.ProviderRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero() ||
			execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_succeeded_fields_invalid")
		}
	case application.ExecutionFailed:
		providerAccepted := execution.ProviderRef != ""
		if execution.FinishedAt.IsZero() || !validText(execution.FailureCode) ||
			(providerAccepted != (execution.ExternalRef != "")) ||
			(providerAccepted && (execution.StartedAt.IsZero() || execution.DeadlineAt.IsZero() || execution.ProviderAcceptedAt.IsZero())) ||
			(!providerAccepted && (!execution.StartedAt.IsZero() || !execution.DeadlineAt.IsZero() ||
				!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() || !execution.ProviderObservedAt.IsZero())) {
			return errors.New("sqlite.execution_failed_fields_invalid")
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
	if aggregate.Ref().String() != expectedGoalRef {
		return errors.New("sqlite.goal_record_ref_invalid")
	}
	items := make(map[goal.WorkItemRef]goal.WorkItem, aggregate.WorkItemCount())
	for _, item := range aggregate.WorkItems() {
		items[item.Ref()] = item
	}
	executions := make(map[goal.ExecutionRef]application.ExecutionRecord, len(record.Executions))
	for _, execution := range record.Executions {
		if err := validateExecution(execution); err != nil {
			return err
		}
		item, found := items[execution.WorkItemRef]
		if !found || execution.GoalRef != aggregate.Ref() {
			return errors.New("sqlite.goal_record_execution_scope_invalid")
		}
		if _, duplicate := executions[execution.Ref]; duplicate {
			return errors.New("sqlite.goal_record_execution_duplicate")
		}
		if !workItemExecutionStateMatches(item, execution) {
			return errors.New("sqlite.goal_record_execution_binding_invalid")
		}
		executions[execution.Ref] = execution
	}
	for _, item := range items {
		if executionRef, hasBinding := item.Execution(); hasBinding {
			execution, found := executions[executionRef]
			if !found || execution.WorkItemRef != item.Ref() {
				return errors.New("sqlite.goal_record_item_execution_invalid")
			}
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

func workItemExecutionStateMatches(item goal.WorkItem, execution application.ExecutionRecord) bool {
	bound, hasBinding := item.Execution()
	switch item.State() {
	case goal.WorkItemStatePending:
		return !hasBinding && execution.State == application.ExecutionQueued
	case goal.WorkItemStateRunning:
		return hasBinding && bound == execution.Ref &&
			(execution.State == application.ExecutionDispatching || execution.State == application.ExecutionRunning)
	case goal.WorkItemStateSucceeded:
		return hasBinding && bound == execution.Ref && execution.State == application.ExecutionSucceeded
	case goal.WorkItemStateFailed:
		return hasBinding && bound == execution.Ref && execution.State == application.ExecutionFailed
	case goal.WorkItemStateSkipped:
		return false
	default:
		return false
	}
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
		action.ExecutionRef.String() == "" || action.AvailableAt.IsZero() {
		return errors.New("sqlite.action_invalid")
	}
	switch action.Kind {
	case application.ActionLaunchAgent, application.ActionObserveAgent:
		return nil
	default:
		return errors.New("sqlite.action_kind_invalid")
	}
}

func validateClaim(claim application.ActionClaim) error {
	if !validText(claim.Token) || !validText(claim.WorkerRef) || claim.Attempt == 0 ||
		claim.Attempt > maxSQLiteInteger || claim.LeaseUntil.IsZero() {
		return errors.New("sqlite.claim_invalid")
	}
	if !validText(claim.Action.Ref) || claim.Action.GoalRef.String() == "" ||
		claim.Action.WorkItemRef.String() == "" || claim.Action.ExecutionRef.String() == "" ||
		claim.Action.AvailableAt.IsZero() {
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
	preparedItem, found := state.Goal.WorkItem(state.Claim.Action.WorkItemRef)
	if !found || preparedItem.Revision() <= 1 {
		return errors.New("sqlite.launch_prepare_item_invalid")
	}
	item, err := validateGoalMutation(
		state.Claim,
		state.ExpectedGoalRevision,
		preparedItem.Revision()-1,
		state.Goal,
		state.Execution,
	)
	if err != nil {
		return err
	}
	if state.Claim.Action.Kind != application.ActionLaunchAgent ||
		state.Goal.State() != goal.GoalStateRunning || item.State() != goal.WorkItemStateRunning ||
		state.Execution.State != application.ExecutionDispatching ||
		state.Execution.Ref != state.Claim.Action.ExecutionRef || state.Execution.WorkItemRef != item.Ref() {
		return errors.New("sqlite.launch_prepare_invalid")
	}
	if err := validateEvent(state.Event); err != nil ||
		!eventMatches(state.Event, state.Goal.Ref().String(), item.Ref().String(), state.Execution.Ref.String()) {
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
		state.Execution.GoalRef != state.Claim.Action.GoalRef || state.Execution.WorkItemRef != state.Claim.Action.WorkItemRef {
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
	if err := validateEvent(state.Event); err != nil {
		return err
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
	if err := validateEvents(state.Events, state.Goal.Ref(), item.Ref(), state.Execution.Ref); err != nil {
		return goal.WorkItem{}, err
	}
	if err := validateScheduled(state.Goal, state.NewExecutions, state.NewActions); err != nil {
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
	if err := validateEvents(state.Events, state.Goal.Ref(), item.Ref(), state.Execution.Ref); err != nil {
		return goal.WorkItem{}, err
	}
	if err := validateScheduled(state.Goal, state.NewExecutions, state.NewActions); err != nil {
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
		execution.WorkItemRef != item.Ref() || aggregate.Ref() != claim.Action.GoalRef {
		return goal.WorkItem{}, errors.New("sqlite.mutation_scope_invalid")
	}
	return item, nil
}

func validateEvents(events []application.EventRecord, goalRef goal.GoalRef, _ goal.WorkItemRef, _ goal.ExecutionRef) error {
	if len(events) == 0 {
		return errors.New("sqlite.events_required")
	}
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		if err := validateEvent(event); err != nil {
			return err
		}
		if event.GoalRef != goalRef {
			return errors.New("sqlite.event_scope_mismatch")
		}
		if _, duplicate := seen[event.Ref]; duplicate {
			return errors.New("sqlite.event_duplicate")
		}
		seen[event.Ref] = struct{}{}
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
