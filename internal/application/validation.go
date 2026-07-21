package application

import (
	"errors"
	"mime"
	"reflect"
	"slices"
	"strings"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func validatePersistedCandidate(candidate goal.Goal, executions []ExecutionRecord, record GoalRecord) error {
	if !reflect.DeepEqual(record.Goal.Snapshot(), candidate.Snapshot()) ||
		!slices.Equal(record.Executions, executions) ||
		len(record.Artifacts) != 0 || len(record.Attestations) != 0 || len(record.Controls) != 0 ||
		len(record.WorkspaceBindings) != 0 || len(record.ChangeSets) != 0 ||
		len(record.MergeObservations) != 0 || len(record.IntegrationReceipts) != 0 ||
		len(record.ConsumptionReceipts) != 0 {
		return &StateError{Code: StateConflict}
	}
	return nil
}

func validateClaimedRecord(claim ActionClaim, record GoalRecord, kind ActionKind) error {
	execution, ok := executionForAction(record, claim.Action)
	intent := record.Goal.AppSpec().Intent()
	if claim.Token == "" || claim.WorkerRef == "" || claim.Action.Kind != kind ||
		claim.DeliveryAttempt == 0 || claim.Fence == 0 || claim.LeaseUntil.IsZero() ||
		claim.Action.GoalRef != record.Goal.Ref() ||
		claim.Action.PlanGeneration == 0 || claim.Action.PlanGeneration > record.Goal.PlanGeneration() ||
		claim.Action.WorkItemGeneration == 0 ||
		!ok || claim.Action.WorkItemRef != execution.WorkItemRef ||
		claim.Action.ExecutionRef != execution.Ref || execution.GoalRef != record.Goal.Ref() ||
		intent.Ref() != record.Goal.Intent() || intent.Hash() != record.Goal.IntentHash() ||
		intent.Actor() != record.Goal.Actor() || intent.Project() != record.Goal.Project() ||
		record.Goal.AppSpec().Hash() != record.Goal.SpecHash() ||
		record.Goal.WorkItemCount() == 0 || record.Goal.State() != goal.GoalStateRunning ||
		record.RequestFingerprint == "" {
		return errors.New("application.claim_record_mismatch")
	}
	item, ok := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !ok || item.Goal() != record.Goal.Ref() || item.Actor() != record.Goal.Actor() ||
		item.Project() != record.Goal.Project() || item.Ref() != execution.WorkItemRef {
		return errors.New("application.claim_record_mismatch")
	}
	if execution.MaxOutputBytes <= 0 || execution.AttemptNo == 0 ||
		execution.MaxExecutionAttempts == 0 || execution.AttemptNo > execution.MaxExecutionAttempts ||
		execution.PlanGeneration == 0 || execution.PlanGeneration != claim.Action.PlanGeneration ||
		execution.AppSpecGeneration != record.Goal.AppSpec().Generation() || execution.SpecHash != record.Goal.SpecHash() ||
		strings.TrimSpace(execution.ArtifactMediaType) == "" ||
		strings.TrimSpace(execution.IdempotencyKey) == "" || execution.CreatedAt.IsZero() ||
		execution.Ref.String() == "" {
		return errors.New("application.execution_record_invalid")
	}
	if execution.ExecutionWorkspaceRef.String() != "" && execution.RepositoryRef.String() == "" {
		return errors.New("application.execution_workspace_invalid")
	}
	switch kind {
	case ActionPrepareWorkspace:
		if claim.Action.Ref != "action:prepare-workspace:"+execution.Ref.String() ||
			claim.Action.WorkItemGeneration > item.Revision() || len(item.WriteSet()) == 0 ||
			execution.RepositoryRef.String() == "" || execution.ExecutionWorkspaceRef.String() == "" ||
			!validWorkspacePrepareClaimState(item, execution) {
			return errors.New("application.workspace_prepare_state_invalid")
		}
	case ActionLaunchAgent:
		if claim.Action.Ref != "action:launch:"+execution.Ref.String() ||
			claim.Action.WorkItemGeneration > item.Revision() ||
			!validLaunchClaimState(item, execution) || !execution.StartedAt.IsZero() ||
			!execution.DeadlineAt.IsZero() || execution.ProviderRef != "" || execution.ModelRef != "" ||
			execution.AgentRef != "" || execution.ExternalRef != "" {
			return errors.New("application.launch_state_invalid")
		}
	case ActionObserveAgent:
		if claim.Action.Ref != "action:observe:"+execution.Ref.String() ||
			claim.Action.WorkItemGeneration > item.Revision() ||
			item.State() != goal.WorkItemStateRunning || execution.State != ExecutionRunning ||
			execution.ProviderRef == "" || execution.ModelRef == "" || execution.AgentRef == "" ||
			execution.ExternalRef == "" || execution.StartedAt.IsZero() ||
			execution.ProviderAcceptedAt.IsZero() || !execution.DeadlineAt.After(execution.StartedAt) {
			return errors.New("application.observe_state_invalid")
		}
	case ActionCommitChange:
		if claim.Action.Ref != "action:commit-change:"+execution.Ref.String() ||
			claim.Action.WorkItemGeneration > item.Revision() || item.State() != goal.WorkItemStateRunning ||
			execution.State != ExecutionAwaitingCommit || execution.ExecutionWorkspaceRef.String() == "" {
			return errors.New("application.commit_change_state_invalid")
		}
	case ActionIntegrateChange:
		if !strings.HasPrefix(claim.Action.Ref, "action:integrate-change:") ||
			claim.Action.WorkItemGeneration > item.Revision() || item.State() != goal.WorkItemStateRunning ||
			execution.State != ExecutionAwaitingIntegration || execution.ExecutionWorkspaceRef.String() == "" {
			return errors.New("application.integrate_change_state_invalid")
		}
	}
	return nil
}

func validWorkspacePrepareClaimState(item goal.WorkItem, execution ExecutionRecord) bool {
	return (item.State() == goal.WorkItemStatePending || item.State() == goal.WorkItemStateRunning) &&
		execution.State == ExecutionQueued
}

func validateCreatedRecord(
	request SubmitRequest,
	fingerprint string,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	record GoalRecord,
) error {
	appSpec := record.Goal.AppSpec()
	intent := appSpec.Intent()
	if record.RequestRef != request.RequestRef || record.RequestFingerprint != fingerprint ||
		record.RequestedBy != principal.Ref ||
		intent.Actor() != principal.ActorRef || intent.Project() != projectRef ||
		intent.Statement() != request.Statement || record.Goal.Actor() != principal.ActorRef ||
		record.Goal.Project() != projectRef || record.Goal.WorkItemCount() == 0 ||
		appSpec.Generation() != 1 || appSpec.Objective() != normalizedObjective(request.Statement, request.NormalizedObjective) ||
		appSpec.Reason() != initialAppSpecReason || appSpec.ConfirmedBy() != principal.ActorRef {
		return &StateError{Code: StateConflict}
	}
	if _, hasParent := appSpec.ParentRef(); hasParent {
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

func validateAmendedRecord(
	request AmendRequest,
	fingerprint string,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	source goal.Goal,
	record GoalRecord,
) error {
	appSpec := record.Goal.AppSpec()
	intent := appSpec.Intent()
	parentRef, hasParent := appSpec.ParentRef()
	if record.RequestRef != request.RequestRef || record.RequestFingerprint != fingerprint ||
		record.RequestedBy != principal.Ref ||
		record.Goal.Actor() != source.Actor() || record.Goal.Project() != projectRef ||
		record.Goal.State() != goal.GoalStatePending || record.Goal.WorkItemCount() != 0 ||
		len(record.Executions) != 0 || len(record.Artifacts) != 0 || len(record.Attestations) != 0 ||
		len(record.WorkspaceBindings) != 0 || len(record.ChangeSets) != 0 ||
		len(record.MergeObservations) != 0 || len(record.IntegrationReceipts) != 0 ||
		len(record.Controls) != 0 || len(record.ConsumptionReceipts) != 0 ||
		intent.Actor() != source.Actor() || intent.Project() != projectRef ||
		intent.Statement() != request.Statement ||
		appSpec.Generation() != source.AppSpec().Generation()+1 || !hasParent ||
		parentRef != source.AppSpec().Ref() || appSpec.ParentHash() != source.SpecHash() ||
		appSpec.Objective() != normalizedObjective(request.Statement, request.NormalizedObjective) ||
		appSpec.Reason() != request.Reason || appSpec.ConfirmedBy() != principal.ActorRef {
		return &StateError{Code: StateConflict}
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
		(item.State() == goal.WorkItemStateRunning && execution.State == ExecutionQueued) ||
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
	case strings.TrimSpace(request.Statement) == "":
		return errors.New("application.statement_required")
	default:
		return nil
	}
}

func validateAmendRequest(request AmendRequest) error {
	switch {
	case strings.TrimSpace(request.RequestRef) == "" || strings.TrimSpace(request.RequestRef) != request.RequestRef:
		return errors.New("application.request_ref_invalid")
	case request.SourceGoalRef.String() == "":
		return errors.New("application.source_goal_ref_required")
	case request.ExpectedSourceRevision == 0:
		return errors.New("application.source_revision_required")
	case !goal.IsCanonicalAppSpecHash(request.ExpectedSourceSpecHash):
		return errors.New("application.source_spec_hash_invalid")
	case strings.TrimSpace(request.Statement) == "":
		return errors.New("application.statement_required")
	case strings.TrimSpace(request.Reason) == "":
		return errors.New("application.amendment_reason_required")
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
