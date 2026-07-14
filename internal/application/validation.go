package application

import (
	"errors"
	"mime"
	"strings"
	"time"

	"orquesta/internal/goal"
)

func validateClaimedRecord(claim ActionClaim, record GoalRecord, kind ActionKind) error {
	if claim.Token == "" || claim.WorkerRef == "" || claim.Action.Kind != kind ||
		claim.Attempt == 0 || claim.LeaseUntil.IsZero() ||
		claim.Action.GoalRef != record.Goal.Ref() ||
		claim.Action.WorkItemRef != record.Execution.WorkItemRef ||
		claim.Action.ExecutionRef != record.Execution.Ref ||
		record.Execution.GoalRef != record.Goal.Ref() ||
		record.Intent.Ref() != record.Goal.Intent() ||
		record.Intent.Hash() != record.Goal.IntentHash() ||
		record.Intent.Actor() != record.Goal.Actor() ||
		record.Intent.Project() != record.Goal.Project() ||
		record.Goal.WorkItemCount() != 1 || record.Goal.State() != goal.GoalStateRunning ||
		record.RequestFingerprint == "" {
		return errors.New("application.claim_record_mismatch")
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.Goal() != record.Goal.Ref() || item.Actor() != record.Goal.Actor() ||
		item.Project() != record.Goal.Project() || item.Ref() != record.Execution.WorkItemRef {
		return errors.New("application.claim_record_mismatch")
	}
	if record.Execution.MaxOutputBytes <= 0 || record.Execution.MaxAttempts == 0 ||
		strings.TrimSpace(record.Execution.ArtifactMediaType) == "" ||
		strings.TrimSpace(record.Execution.IdempotencyKey) == "" ||
		record.Execution.CreatedAt.IsZero() ||
		!record.Execution.DeadlineAt.After(record.Execution.CreatedAt) {
		return errors.New("application.execution_record_invalid")
	}
	switch kind {
	case ActionLaunchAgent:
		if claim.Action.Ref != "action:launch:"+record.Execution.Ref.String() ||
			item.State() != goal.WorkItemStatePending || record.Execution.State != ExecutionQueued {
			return errors.New("application.launch_state_invalid")
		}
	case ActionObserveAgent:
		if claim.Action.Ref != "action:observe:"+record.Execution.Ref.String() ||
			item.State() != goal.WorkItemStateRunning || record.Execution.State != ExecutionRunning ||
			record.Execution.ProviderRef == "" || record.Execution.ExternalRef == "" ||
			record.Execution.StartedAt.IsZero() {
			return errors.New("application.observe_state_invalid")
		}
	}
	return nil
}

func validateCreatedRecord(request SubmitRequest, fingerprint string, record GoalRecord) error {
	if record.RequestRef != request.RequestRef || record.RequestFingerprint != fingerprint ||
		record.Intent.Actor() != request.ActorRef || record.Intent.Project() != request.ProjectRef ||
		record.Intent.Statement() != request.Statement || record.Goal.Actor() != request.ActorRef ||
		record.Goal.Project() != request.ProjectRef || record.Goal.WorkItemCount() != 1 ||
		record.Execution.GoalRef != record.Goal.Ref() {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func lifecycleTime(now time.Time, aggregate goal.Goal, item goal.WorkItem) time.Time {
	now = now.UTC()
	floor := aggregate.CreatedAt()
	if started, ok := aggregate.StartedAt(); ok && started.After(floor) {
		floor = started
	}
	if item.CreatedAt().After(floor) {
		floor = item.CreatedAt()
	}
	if started, ok := item.StartedAt(); ok && started.After(floor) {
		floor = started
	}
	if now.Before(floor) {
		return floor
	}
	return now
}

func validateSubmitRequest(request SubmitRequest) error {
	switch {
	case strings.TrimSpace(request.RequestRef) == "" || strings.TrimSpace(request.RequestRef) != request.RequestRef:
		return errors.New("application.request_ref_invalid")
	case request.ActorRef.String() == "":
		return errors.New("application.actor_ref_required")
	case request.ProjectRef.String() == "":
		return errors.New("application.project_ref_required")
	case strings.TrimSpace(request.Statement) == "":
		return errors.New("application.statement_required")
	default:
		return nil
	}
}

func stableFailureCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "application.execution_failed"
	}
	if len(code) > 160 {
		return code[:160]
	}
	return code
}

func compatibleMediaType(expected, observed string) bool {
	expectedType, _, expectedErr := mime.ParseMediaType(expected)
	observedType, _, observedErr := mime.ParseMediaType(observed)
	return expectedErr == nil && observedErr == nil && expectedType == observedType
}
