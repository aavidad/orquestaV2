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
	if len(snapshot.WorkItems) != 1 {
		return errors.New("sqlite.one_work_item_required")
	}
	item := snapshot.WorkItems[0]
	if snapshot.State != goal.GoalStateRunning || item.State != goal.WorkItemStatePending {
		return errors.New("sqlite.create_lifecycle_invalid")
	}
	intent := state.Intent.Snapshot()
	if !sameIntentSnapshot(intent, snapshot.Intent) {
		return errors.New("sqlite.intent_snapshot_mismatch")
	}
	if err := validateExecution(state.Execution); err != nil {
		return err
	}
	if state.Execution.State != application.ExecutionQueued {
		return errors.New("sqlite.create_execution_state_invalid")
	}
	if state.Execution.GoalRef.String() != snapshot.Ref || state.Execution.WorkItemRef.String() != item.Ref {
		return errors.New("sqlite.execution_scope_mismatch")
	}
	if err := validateAction(state.Action); err != nil {
		return err
	}
	if state.Action.Kind != application.ActionLaunchAgent {
		return errors.New("sqlite.create_action_kind_invalid")
	}
	if !actionMatches(state.Action, snapshot.Ref, item.Ref, state.Execution.Ref.String()) {
		return errors.New("sqlite.action_scope_mismatch")
	}
	if err := validateEvent(state.Event); err != nil {
		return err
	}
	if !eventMatches(state.Event, snapshot.Ref, item.Ref, state.Execution.Ref.String()) {
		return errors.New("sqlite.event_scope_mismatch")
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
	if execution.CreatedAt.IsZero() || execution.DeadlineAt.IsZero() ||
		!execution.DeadlineAt.After(execution.CreatedAt) {
		return errors.New("sqlite.execution_time_invalid")
	}
	if !optionalText(execution.ProviderRef) || !optionalText(execution.ExternalRef) ||
		!optionalText(execution.FailureCode) {
		return errors.New("sqlite.execution_text_invalid")
	}
	if !execution.StartedAt.IsZero() && execution.StartedAt.Before(execution.CreatedAt) {
		return errors.New("sqlite.execution_started_at_invalid")
	}
	if !execution.LastObservedAt.IsZero() &&
		(execution.StartedAt.IsZero() || execution.LastObservedAt.Before(execution.StartedAt)) {
		return errors.New("sqlite.execution_observed_at_invalid")
	}
	if !execution.FinishedAt.IsZero() &&
		(execution.StartedAt.IsZero() || execution.FinishedAt.Before(execution.StartedAt)) {
		return errors.New("sqlite.execution_finished_at_invalid")
	}
	if !execution.LastObservedAt.IsZero() && !execution.FinishedAt.IsZero() &&
		execution.LastObservedAt.After(execution.FinishedAt) {
		return errors.New("sqlite.execution_observed_after_finish")
	}
	switch execution.State {
	case application.ExecutionQueued:
		if execution.ProviderRef != "" || execution.ExternalRef != "" || !execution.StartedAt.IsZero() ||
			!execution.ProviderAcceptedAt.IsZero() || !execution.LastObservedAt.IsZero() ||
			!execution.ProviderObservedAt.IsZero() || !execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_queued_fields_invalid")
		}
	case application.ExecutionRunning:
		if !validText(execution.ProviderRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			!execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_running_fields_invalid")
		}
	case application.ExecutionSucceeded:
		if !validText(execution.ProviderRef) || !validText(execution.ExternalRef) || execution.StartedAt.IsZero() ||
			execution.FinishedAt.IsZero() || execution.FailureCode != "" {
			return errors.New("sqlite.execution_succeeded_fields_invalid")
		}
	case application.ExecutionFailed:
		if execution.StartedAt.IsZero() || execution.FinishedAt.IsZero() || !validText(execution.FailureCode) ||
			((execution.ProviderRef == "") != (execution.ExternalRef == "")) {
			return errors.New("sqlite.execution_failed_fields_invalid")
		}
	default:
		return errors.New("sqlite.execution_state_invalid")
	}
	return nil
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
	if !validText(event.Ref) || !validText(event.Kind) || event.GoalRef.String() == "" ||
		event.WorkItemRef.String() == "" || event.ExecutionRef.String() == "" || event.OccurredAt.IsZero() {
		return errors.New("sqlite.event_invalid")
	}
	return nil
}

func validateLaunchAccepted(state application.LaunchAcceptedState) (goal.WorkItem, error) {
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
	if state.Claim.Action.Kind != application.ActionLaunchAgent || state.NextAction.Kind != application.ActionObserveAgent {
		return goal.WorkItem{}, errors.New("sqlite.launch_action_kind_invalid")
	}
	if state.Goal.State() != goal.GoalStateRunning || item.State() != goal.WorkItemStateRunning ||
		state.Execution.State != application.ExecutionRunning {
		return goal.WorkItem{}, errors.New("sqlite.launch_lifecycle_invalid")
	}
	if err := validateAction(state.NextAction); err != nil {
		return goal.WorkItem{}, err
	}
	if !actionMatches(
		state.NextAction,
		state.Goal.Ref().String(),
		item.Ref().String(),
		state.Execution.Ref.String(),
	) {
		return goal.WorkItem{}, errors.New("sqlite.next_action_scope_mismatch")
	}
	if err := validateEvent(state.Event); err != nil {
		return goal.WorkItem{}, err
	}
	if !eventMatches(state.Event, state.Goal.Ref().String(), item.Ref().String(), state.Execution.Ref.String()) {
		return goal.WorkItem{}, errors.New("sqlite.event_scope_mismatch")
	}
	return item, nil
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
		if state.Execution.State != application.ExecutionQueued {
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
	if state.Goal.State() != goal.GoalStateSucceeded || state.Execution.State != application.ExecutionSucceeded {
		return goal.WorkItem{}, errors.New("sqlite.succeeded_state_invalid")
	}
	if item.State() != goal.WorkItemStateSucceeded {
		return goal.WorkItem{}, errors.New("sqlite.succeeded_item_state_invalid")
	}
	if state.Execution.FinishedAt.IsZero() {
		return goal.WorkItem{}, errors.New("sqlite.execution_finished_at_required")
	}
	if state.Artifact.Stored.Ref.String() == "" || !validText(state.Artifact.Stored.Digest) ||
		!validText(state.Artifact.Stored.MediaType) || state.Artifact.Stored.Size < 0 || state.Artifact.CreatedAt.IsZero() ||
		state.Artifact.GoalRef != state.Goal.Ref() || state.Artifact.WorkItemRef != item.Ref() {
		return goal.WorkItem{}, errors.New("sqlite.artifact_invalid")
	}
	if state.Attestation.Ref.String() == "" || state.Attestation.GoalRef != state.Goal.Ref() ||
		state.Attestation.WorkItemRef != item.Ref() || state.Attestation.ExecutionRef != state.Execution.Ref ||
		state.Attestation.ArtifactRef != state.Artifact.Stored.Ref || !validText(state.Attestation.Policy) ||
		state.Attestation.AcceptedAt.IsZero() {
		return goal.WorkItem{}, errors.New("sqlite.attestation_invalid")
	}
	if err := validateEvents(state.Events, state.Goal.Ref(), item.Ref(), state.Execution.Ref); err != nil {
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
	if state.Goal.State() != goal.GoalStateFailed || state.Execution.State != application.ExecutionFailed ||
		!validText(state.Execution.FailureCode) || state.Execution.FinishedAt.IsZero() {
		return goal.WorkItem{}, errors.New("sqlite.failed_state_invalid")
	}
	if item.State() != goal.WorkItemStateFailed {
		return goal.WorkItem{}, errors.New("sqlite.failed_item_state_invalid")
	}
	if err := validateEvents(state.Events, state.Goal.Ref(), item.Ref(), state.Execution.Ref); err != nil {
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
	if len(snapshot.WorkItems) != 1 || aggregate.Revision() <= expectedGoal ||
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

func validateEvents(events []application.EventRecord, goalRef goal.GoalRef, itemRef goal.WorkItemRef, executionRef goal.ExecutionRef) error {
	if len(events) == 0 {
		return errors.New("sqlite.events_required")
	}
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		if err := validateEvent(event); err != nil {
			return err
		}
		if !eventMatches(event, goalRef.String(), itemRef.String(), executionRef.String()) {
			return errors.New("sqlite.event_scope_mismatch")
		}
		if _, duplicate := seen[event.Ref]; duplicate {
			return errors.New("sqlite.event_duplicate")
		}
		seen[event.Ref] = struct{}{}
	}
	return nil
}

func sameIntentSnapshot(left, right goal.IntentManifestSnapshot) bool {
	return left.Ref == right.Ref && left.ActorRef == right.ActorRef && left.ProjectRef == right.ProjectRef &&
		left.Statement == right.Statement && left.SubmittedAt.Equal(right.SubmittedAt) && left.Hash == right.Hash
}

func actionMatches(action application.ActionRecord, goalRef, itemRef, executionRef string) bool {
	return action.GoalRef.String() == goalRef && action.WorkItemRef.String() == itemRef &&
		action.ExecutionRef.String() == executionRef
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
