package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type ListPendingChangesRequest struct {
	Limit int
}

type ListPendingChangesResult struct {
	Changes []PendingChange
}

func (orchestrator *Orchestrator) ListPendingChanges(ctx context.Context, access Access,
	request ListPendingChangesRequest,
) (ListPendingChangesResult, error) {
	if orchestrator == nil || request.Limit <= 0 {
		return ListPendingChangesResult{}, errors.New("application.pending_changes_request_invalid")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return ListPendingChangesResult{}, err
	}
	authority, err := orchestrator.authorizeRead(ctx, access, identity.PermissionGoalsList, projectRef.String())
	if err != nil {
		return ListPendingChangesResult{}, err
	}
	repositoryRef, err := orchestrator.state.ProjectRepository(ctx, projectRef)
	if err != nil {
		return ListPendingChangesResult{}, err
	}
	query := PendingChangeQuery{ProjectRef: projectRef, RepositoryRef: repositoryRef, Limit: request.Limit}
	if authority.Decision().Role() == identity.RoleContributor {
		query.ActorRef = principal.ActorRef
	}
	changes, err := orchestrator.state.ListPendingChanges(ctx, query)
	if err != nil {
		return ListPendingChangesResult{}, err
	}
	for _, pending := range changes {
		if pending.ChangeSet.ProjectRef != projectRef || pending.ChangeSet.RepositoryRef != repositoryRef ||
			(query.ActorRef.String() != "" && pending.ChangeSet.ActorRef != query.ActorRef) {
			return ListPendingChangesResult{}, &StateError{Code: StateConflict}
		}
	}
	return ListPendingChangesResult{Changes: changes}, nil
}

type IntegrateChangeRequest struct {
	RequestRef        string
	GoalRef           goal.GoalRef
	ChangeRef         ports.ChangeSetRef
	ExpectedTargetOID string
}

type IntegrateChangeResult struct {
	Action  ActionRecord
	Created bool
}

func (orchestrator *Orchestrator) IntegrateChange(ctx context.Context, access Access,
	request IntegrateChangeRequest,
) (IntegrateChangeResult, error) {
	expectedFormat := ports.GitObjectFormatSHA1
	if len(request.ExpectedTargetOID) == 2*sha256.Size {
		expectedFormat = ports.GitObjectFormatSHA256
	}
	if orchestrator == nil || !validApplicationRef(request.RequestRef) || request.GoalRef.String() == "" ||
		request.ChangeRef.String() == "" || ports.ValidateGitOID(request.ExpectedTargetOID, expectedFormat) != nil {
		return IntegrateChangeResult{}, errors.New("application.integrate_change_request_invalid")
	}
	principal, projectRef, err := access.values()
	if err != nil {
		return IntegrateChangeResult{}, err
	}
	now := orchestrator.clock.Now().UTC()
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(
		ctx, access, identity.PermissionChangesIntegrate, request.GoalRef.String(), now,
		"authorization-request:integrate:"+request.RequestRef,
	)
	if err != nil {
		return IntegrateChangeResult{}, err
	}
	now = authorizationCausalFloor(now, authorization)
	fingerprint := fingerprintFields(
		"orquesta.integrate-change.v1", principal.Ref.String(), projectRef.String(), request.RequestRef,
		request.GoalRef.String(), request.ChangeRef.String(), request.ExpectedTargetOID,
	)
	record, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return IntegrateChangeResult{}, err
	}
	change, found := changeSetByRef(record, request.ChangeRef)
	if !found || record.Goal.Project() != projectRef || change.ProjectRef != projectRef {
		return IntegrateChangeResult{}, &StateError{Code: StateNotFound}
	}
	if change.ObjectFormat != expectedFormat {
		return IntegrateChangeResult{}, errors.New("application.integrate_change_request_invalid")
	}
	item, itemFound := record.Goal.WorkItem(change.WorkItemRef)
	execution, executionFound := executionByRef(record.Executions, change.ExecutionRef)
	if !itemFound || !executionFound {
		return IntegrateChangeResult{}, &StateError{Code: StateConflict}
	}
	if previous, found, replayErr := integrationIntentForRequest(record, request.RequestRef); replayErr != nil {
		return IntegrateChangeResult{}, replayErr
	} else if found {
		action, replayErr := integrationReplayAction(
			record, item, execution, change, previous, request, principal.Ref, fingerprint,
		)
		if replayErr != nil {
			return IntegrateChangeResult{}, replayErr
		}
		return orchestrator.persistIntegrationAdmission(
			ctx, request, record, item, authorization, principal.Ref, projectRef, fingerprint, action, now, true,
		)
	}
	if execution.State != ExecutionAwaitingIntegration || item.State() != goal.WorkItemStateRunning {
		return IntegrateChangeResult{}, &StateError{Code: StateConflict}
	}
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return IntegrateChangeResult{}, err
	}
	action, err := orchestrator.integrateChangeAction(
		policy, record.Goal, item, execution, change, request.ExpectedTargetOID,
		principal.Ref, authorization, request.RequestRef, fingerprint, now,
	)
	if err != nil {
		return IntegrateChangeResult{}, err
	}
	return orchestrator.persistIntegrationAdmission(
		ctx, request, record, item, authorization, principal.Ref, projectRef, fingerprint, action, now, false,
	)
}

func (orchestrator *Orchestrator) persistIntegrationAdmission(ctx context.Context, request IntegrateChangeRequest,
	record GoalRecord, item goal.WorkItem, authorization identity.AuthorizationReceipt,
	principal identity.PrincipalRef, projectRef goal.ProjectRef, fingerprint string, action ActionRecord,
	at time.Time, replay bool,
) (IntegrateChangeResult, error) {
	persisted, created, err := orchestrator.state.AdmitIntegration(ctx, AdmitIntegrationState{
		RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
		AuthorizationReceipt: authorization, PrincipalRef: principal,
		ProjectRef: projectRef, GoalRef: request.GoalRef, ChangeRef: request.ChangeRef,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Action: action, OperationAt: at,
	})
	if err != nil {
		return IntegrateChangeResult{}, err
	}
	if (replay && created) || persisted.Ref != action.Ref || persisted.ChangeRef != request.ChangeRef ||
		persisted.ExpectedTargetOID != request.ExpectedTargetOID {
		return IntegrateChangeResult{}, &StateError{Code: StateConflict}
	}
	return IntegrateChangeResult{Action: persisted, Created: created}, nil
}

func integrationIntentForRequest(record GoalRecord, requestRef string) (EffectIntent, bool, error) {
	var result EffectIntent
	found := false
	for _, intent := range record.EffectIntents {
		if intent.ActionKind != ActionIntegrateChange || intent.RequestRef != requestRef {
			continue
		}
		if found {
			return EffectIntent{}, false, &StateError{Code: StateConflict}
		}
		result, found = intent, true
	}
	return result, found, nil
}

func integrationReplayAction(record GoalRecord, item goal.WorkItem, execution ExecutionRecord,
	change ChangeSet, intent EffectIntent, request IntegrateChangeRequest,
	principal identity.PrincipalRef, fingerprint string,
) (ActionRecord, error) {
	if intent.RequestFingerprint != fingerprint || intent.ProposedBy != principal ||
		intent.ActionRef != integrationActionRef(principal, record.Goal.Project(), request.RequestRef) ||
		intent.Kind != EffectKindIntegrateChange || intent.Permission != identity.PermissionChangesIntegrate ||
		intent.Subject.ProjectRef != record.Goal.Project() || intent.Subject.GoalRef != record.Goal.Ref() ||
		intent.Subject.WorkItemRef != change.WorkItemRef || intent.Subject.ExecutionRef != change.ExecutionRef ||
		intent.Subject.PlanGeneration != execution.PlanGeneration ||
		intent.TargetDigest != integrationTargetDigest(change, request.ExpectedTargetOID) {
		return ActionRecord{}, &StateError{Code: StateConflict}
	}
	approval, found := effectApprovalForIntent(record.EffectApprovals, intent.Ref)
	if !found || ValidateEffectApproval(intent, approval) != nil {
		return ActionRecord{}, &StateError{Code: StateConflict}
	}
	return ActionRecord{
		Ref: intent.ActionRef, Kind: ActionIntegrateChange,
		GoalRef: record.Goal.Ref(), WorkItemRef: change.WorkItemRef, ExecutionRef: change.ExecutionRef,
		ChangeRef: change.Ref, ExpectedTargetOID: request.ExpectedTargetOID,
		EffectIntentRef: intent.Ref, EffectIntent: intent, EffectApproval: &approval,
		PlanGeneration: intent.Subject.PlanGeneration, WorkItemGeneration: item.Revision(),
		AvailableAt: intent.CreatedAt,
	}, nil
}

func effectApprovalForIntent(approvals []EffectApproval, intentRef string) (EffectApproval, bool) {
	for _, approval := range approvals {
		if approval.IntentRef == intentRef {
			return approval, true
		}
	}
	return EffectApproval{}, false
}
