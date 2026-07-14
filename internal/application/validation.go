package application

import (
	"errors"
	"mime"
	"strings"
	"time"

	"orquesta/internal/goal"
)

func validateClaimedRecord(claim ActionClaim, record GoalRecord, kind ActionKind) error {
	execution, ok := executionForAction(record, claim.Action)
	if claim.Token == "" || claim.WorkerRef == "" || claim.Action.Kind != kind ||
		claim.Attempt == 0 || claim.LeaseUntil.IsZero() ||
		claim.Action.GoalRef != record.Goal.Ref() ||
		!ok || claim.Action.WorkItemRef != execution.WorkItemRef ||
		claim.Action.ExecutionRef != execution.Ref || execution.GoalRef != record.Goal.Ref() ||
		record.Intent.Ref() != record.Goal.Intent() ||
		record.Intent.Hash() != record.Goal.IntentHash() ||
		record.Intent.Actor() != record.Goal.Actor() ||
		record.Intent.Project() != record.Goal.Project() ||
		record.Goal.WorkItemCount() == 0 || record.Goal.State() != goal.GoalStateRunning ||
		record.RequestFingerprint == "" {
		return errors.New("application.claim_record_mismatch")
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.Goal() != record.Goal.Ref() || item.Actor() != record.Goal.Actor() ||
		item.Project() != record.Goal.Project() || item.Ref() != execution.WorkItemRef {
		return errors.New("application.claim_record_mismatch")
	}
	if execution.MaxOutputBytes <= 0 || execution.MaxAttempts == 0 ||
		strings.TrimSpace(execution.ArtifactMediaType) == "" ||
		strings.TrimSpace(execution.IdempotencyKey) == "" || execution.CreatedAt.IsZero() ||
		execution.Ref.String() == "" {
		return errors.New("application.execution_record_invalid")
	}
	switch kind {
	case ActionLaunchAgent:
		if claim.Action.Ref != "action:launch:"+execution.Ref.String() ||
			!validLaunchClaimState(item, execution) || !execution.StartedAt.IsZero() ||
			!execution.DeadlineAt.IsZero() || execution.ProviderRef != "" || execution.ExternalRef != "" {
			return errors.New("application.launch_state_invalid")
		}
	case ActionObserveAgent:
		if claim.Action.Ref != "action:observe:"+execution.Ref.String() ||
			item.State() != goal.WorkItemStateRunning || execution.State != ExecutionRunning ||
			execution.ProviderRef == "" || execution.ExternalRef == "" || execution.StartedAt.IsZero() ||
			execution.ProviderAcceptedAt.IsZero() || !execution.DeadlineAt.After(execution.StartedAt) {
			return errors.New("application.observe_state_invalid")
		}
	}
	return nil
}

func validateCreatedRecord(request SubmitRequest, fingerprint string, record GoalRecord) error {
	if record.RequestRef != request.RequestRef || record.RequestFingerprint != fingerprint ||
		record.Intent.Actor() != request.ActorRef || record.Intent.Project() != request.ProjectRef ||
		record.Intent.Statement() != request.Statement || record.Goal.Actor() != request.ActorRef ||
		record.Goal.Project() != request.ProjectRef || record.Goal.WorkItemCount() == 0 {
		return &StateError{Code: StateConflict}
	}
	if len(record.Executions) == 0 {
		return &StateError{Code: StateConflict}
	}
	for _, execution := range record.Executions {
		if execution.GoalRef != record.Goal.Ref() {
			return &StateError{Code: StateConflict}
		}
	}
	return nil
}

func executionForAction(record GoalRecord, action ActionRecord) (ExecutionRecord, bool) {
	for _, execution := range record.Executions {
		if execution.Ref == action.ExecutionRef && execution.WorkItemRef == action.WorkItemRef {
			return execution, true
		}
	}
	return ExecutionRecord{}, false
}

func validLaunchClaimState(item goal.WorkItem, execution ExecutionRecord) bool {
	return (item.State() == goal.WorkItemStatePending && execution.State == ExecutionQueued) ||
		(item.State() == goal.WorkItemStateRunning && execution.State == ExecutionDispatching)
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
