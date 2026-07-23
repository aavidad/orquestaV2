package application

import (
	"context"
	"reflect"
	"strings"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

type memoryDirectorMutation struct {
	fingerprint string
	lease       DirectorLeaseRecord
	decision    DirectorDecisionRecord
}

func (repository *memoryRepository) DirectorReplay(
	_ context.Context,
	request DirectorReplayRequest,
) (DirectorReplayRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if request.Kind != DirectorMutationClaim && request.Kind != DirectorMutationRenew &&
		request.Kind != DirectorMutationPlan {
		return DirectorReplayRecord{}, false, &StateError{Code: StateInvalid}
	}
	key := memoryDirectorRequestKey(
		string(request.Kind), request.PrincipalRef, request.ProjectRef, request.RequestRef,
	)
	previous, exists := repository.directorRequests[key]
	if !exists {
		return DirectorReplayRecord{}, false, nil
	}
	if previous.fingerprint != request.RequestFingerprint {
		return DirectorReplayRecord{}, false, &StateError{Code: StateConflict}
	}
	switch request.Kind {
	case DirectorMutationClaim, DirectorMutationRenew:
		if previous.lease.GoalRef != request.GoalRef || previous.lease.PrincipalRef != request.PrincipalRef {
			return DirectorReplayRecord{}, false, &StateError{Code: StateConflict}
		}
		lease, ok := repository.activeDirectorLeaseReplay(previous.lease)
		if !ok {
			return DirectorReplayRecord{}, false, &StateError{Code: StateConflict}
		}
		return DirectorReplayRecord{Lease: lease}, true, nil
	case DirectorMutationPlan:
		if previous.decision.GoalRef != request.GoalRef || previous.decision.PrincipalRef != request.PrincipalRef {
			return DirectorReplayRecord{}, false, &StateError{Code: StateConflict}
		}
		return DirectorReplayRecord{Decision: previous.decision}, true, nil
	}
	return DirectorReplayRecord{}, false, &StateError{Code: StateInvalid}
}

func (repository *memoryRepository) ClaimDirector(
	_ context.Context,
	state ClaimDirectorState,
) (DirectorLeaseRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := memoryDirectorRequestKey("claim", state.PrincipalRef, state.ProjectRef, state.RequestRef)
	if previous, exists := repository.directorRequests[key]; exists {
		if previous.fingerprint != state.RequestFingerprint {
			return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
		}
		lease, ok := repository.activeDirectorLeaseReplay(previous.lease)
		if !ok {
			return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
		}
		return lease, false, nil
	}
	if !memoryWriteAuthorizationValid(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionGoalsDirect, state.GoalRef.String(),
	) || state.Token == "" || state.LeaseDuration <= 0 {
		return DirectorLeaseRecord{}, false, &StateError{Code: StateInvalid}
	}
	record, exists := repository.records[state.GoalRef]
	if !exists || record.Goal.Project() != state.ProjectRef {
		return DirectorLeaseRecord{}, false, &StateError{Code: StateNotFound}
	}
	now := repository.now().UTC()
	current, exists := repository.directorLeases[state.GoalRef]
	if exists && current.Token != "" && current.LeaseUntil.After(now) {
		return DirectorLeaseRecord{}, false, &StateError{Code: StateAlreadyClaimed}
	}
	fence := uint64(1)
	if exists {
		fence = current.Fence + 1
		if fence == 0 {
			return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	lease := DirectorLeaseRecord{
		GoalRef: state.GoalRef, PrincipalRef: state.PrincipalRef, Token: state.Token,
		Fence: fence, LeaseUntil: now.Add(state.LeaseDuration),
	}
	repository.directorLeases[state.GoalRef] = lease
	repository.directorRequests[key] = memoryDirectorMutation{
		fingerprint: state.RequestFingerprint, lease: directorLeaseWithoutToken(lease),
	}
	return lease, true, nil
}

func (repository *memoryRepository) RenewDirector(
	_ context.Context,
	state RenewDirectorState,
) (DirectorLeaseRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := memoryDirectorRequestKey("renew", state.PrincipalRef, state.ProjectRef, state.RequestRef)
	if previous, exists := repository.directorRequests[key]; exists {
		if previous.fingerprint != state.RequestFingerprint {
			return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
		}
		lease, ok := repository.activeDirectorLeaseReplay(previous.lease)
		if !ok {
			return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
		}
		return lease, false, nil
	}
	if !memoryWriteAuthorizationValid(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionGoalsDirect, state.GoalRef.String(),
	) || state.Token == "" || state.Fence == 0 || state.LeaseDuration <= 0 {
		return DirectorLeaseRecord{}, false, &StateError{Code: StateInvalid}
	}
	now := repository.now().UTC()
	current, exists := repository.directorLeases[state.GoalRef]
	if !exists || current.PrincipalRef != state.PrincipalRef || current.Token != state.Token ||
		current.Fence != state.Fence || !current.LeaseUntil.After(now) {
		return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
	}
	nextUntil := now.Add(state.LeaseDuration)
	if !nextUntil.After(current.LeaseUntil) {
		return DirectorLeaseRecord{}, false, &StateError{Code: StateConflict}
	}
	current.LeaseUntil = nextUntil
	repository.directorLeases[state.GoalRef] = current
	repository.directorRequests[key] = memoryDirectorMutation{
		fingerprint: state.RequestFingerprint, lease: directorLeaseWithoutToken(current),
	}
	return current, true, nil
}

func (repository *memoryRepository) ApplyDirectorPlan(
	_ context.Context,
	state ApplyDirectorPlanState,
) (DirectorDecisionRecord, bool, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	key := memoryDirectorRequestKey("plan", state.PrincipalRef, state.ProjectRef, state.RequestRef)
	if previous, exists := repository.directorRequests[key]; exists {
		if previous.fingerprint != state.RequestFingerprint {
			return DirectorDecisionRecord{}, false, &StateError{Code: StateConflict}
		}
		return previous.decision, false, nil
	}
	if !memoryWriteAuthorizationValid(
		state.AuthorizationReceipt, state.PrincipalRef, state.ProjectRef,
		identity.PermissionGoalsDirect, state.GoalRef.String(),
	) {
		return DirectorDecisionRecord{}, false, &StateError{Code: StateInvalid}
	}
	current, exists := repository.records[state.GoalRef]
	if !exists || current.Goal.Project() != state.ProjectRef {
		return DirectorDecisionRecord{}, false, &StateError{Code: StateNotFound}
	}
	now := repository.now().UTC()
	lease, exists := repository.directorLeases[state.GoalRef]
	if !exists || lease.PrincipalRef != state.PrincipalRef || lease.Token != state.LeaseToken ||
		lease.Fence != state.LeaseFence || !lease.LeaseUntil.After(now) {
		return DirectorDecisionRecord{}, false, &StateError{Code: StateConflict}
	}
	if current.Goal.Revision() != state.ExpectedGoalRevision ||
		current.Goal.PlanGeneration() != state.ExpectedPlanGeneration ||
		!memoryDirectorSuccessorValid(current.Goal, state.Goal, state.Decision) ||
		!memoryDirectorDecisionValid(state) ||
		!memoryDirectorScheduleValid(state) {
		return DirectorDecisionRecord{}, false, &StateError{Code: StateConflict}
	}
	if state.Decision.Cause != "" {
		source, found := current.Goal.WorkItem(state.Decision.SourceWorkItemRef)
		execution, executionFound := executionByRef(current.Executions, state.Decision.SourceExecutionRef)
		if !found || source.Revision() != state.ExpectedWorkItemRevision || !executionFound ||
			execution.WorkItemRef != source.Ref() || execution.AttemptNo != state.Decision.SourceExecutionAttempt {
			return DirectorDecisionRecord{}, false, &StateError{Code: StateConflict}
		}
	}
	newActionRefs := make(map[string]struct{}, len(state.NewActions))
	for _, action := range state.NewActions {
		if _, duplicate := repository.actions[action.Ref]; duplicate {
			return DirectorDecisionRecord{}, false, &StateError{Code: StateConflict}
		}
		if _, duplicate := newActionRefs[action.Ref]; duplicate {
			return DirectorDecisionRecord{}, false, &StateError{Code: StateConflict}
		}
		newActionRefs[action.Ref] = struct{}{}
	}
	updated := cloneGoalRecord(current)
	updated.Goal = state.Goal
	for _, execution := range state.UpdatedExecutions {
		updated.Executions = replaceExecution(updated.Executions, execution)
	}
	updated.Executions = append(updated.Executions, state.NewExecutions...)
	updated.WorkItemAuthorities = append(updated.WorkItemAuthorities, state.NewWorkItemAuthorities...)
	for _, action := range state.NewActions {
		if action.EffectIntent.Ref != "" {
			updated.EffectIntents = append(updated.EffectIntents, action.EffectIntent)
		}
		if action.EffectApproval != nil {
			updated.EffectApprovals = append(updated.EffectApprovals, *action.EffectApproval)
		}
		repository.actions[action.Ref] = memoryAction{record: action}
	}
	for _, actionRef := range state.RetireActionRefs {
		if receipt, retired := repository.retireActionLocked(
			actionRef, "director-retire:"+state.Decision.Ref,
			state.PrincipalRef.String(), state.OperationAt,
		); retired {
			updated.ConsumptionReceipts = append(updated.ConsumptionReceipts, receipt)
		}
	}
	repository.records[state.GoalRef] = updated
	repository.events = append(repository.events, state.Events...)
	repository.directorRequests[key] = memoryDirectorMutation{
		fingerprint: state.RequestFingerprint, decision: state.Decision,
	}
	return state.Decision, true, nil
}

func memoryDirectorSuccessorValid(current, candidate goal.Goal, decision DirectorDecisionRecord) bool {
	if candidate.Ref() != current.Ref() || candidate.Actor() != current.Actor() ||
		candidate.Project() != current.Project() || candidate.State() != current.State() ||
		candidate.Revision() != current.Revision()+1 ||
		candidate.PlanGeneration() != current.PlanGeneration()+1 ||
		candidate.SpecHash() != current.SpecHash() {
		return false
	}
	currentPhases, candidatePhases := current.Phases(), candidate.Phases()
	currentItems, candidateItems := current.WorkItems(), candidate.WorkItems()
	if len(candidatePhases) < len(currentPhases) || len(candidateItems) <= len(currentItems) ||
		!reflect.DeepEqual(candidatePhases[:len(currentPhases)], currentPhases) {
		return false
	}
	if decision.Cause == "" {
		return reflect.DeepEqual(candidateItems[:len(currentItems)], currentItems)
	}
	for index, existing := range currentItems {
		if existing.Ref() == decision.SourceWorkItemRef {
			if candidateItems[index].State() != goal.WorkItemStateSuperseded ||
				candidateItems[index].Revision() != existing.Revision()+1 {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(candidateItems[index], existing) {
			return false
		}
	}
	return true
}

func memoryDirectorDecisionValid(state ApplyDirectorPlanState) bool {
	decision := state.Decision
	baseValid := decision.Ref != "" && decision.RequestRef == state.RequestRef &&
		decision.RequestFingerprint == state.RequestFingerprint && decision.GoalRef == state.GoalRef &&
		decision.PrincipalRef == state.PrincipalRef && decision.LeaseFence == state.LeaseFence &&
		decision.SourceGoalRevision == state.ExpectedGoalRevision &&
		decision.SourcePlanGeneration == state.ExpectedPlanGeneration &&
		decision.SourceWorkItemRevision == state.ExpectedWorkItemRevision &&
		decision.AppliedGoalRevision == state.Goal.Revision() &&
		decision.AppliedPlanGeneration == state.Goal.PlanGeneration() &&
		decision.Reason != "" && !decision.DecidedAt.IsZero() &&
		decision.AuthorizationReceipt.Ref() == state.AuthorizationReceipt.Ref()
	if !baseValid {
		return false
	}
	if decision.Cause == "" {
		return decision.SourceWorkItemRef.String() == "" && decision.SourceWorkItemRevision == 0 &&
			decision.SourceExecutionRef.String() == "" && decision.SourceExecutionAttempt == 0 &&
			decision.CouncilSubjectDigest == "" && decision.CouncilDecisionRef == "" && decision.CouncilDecisionDigest == ""
	}
	baseCause := decision.Cause == goal.ReplanCauseSplitPending ||
		decision.Cause == goal.ReplanCauseExecutionStopped ||
		decision.Cause == goal.ReplanCauseExecutionFailed || decision.Cause == goal.ReplanCauseReviewChangesRequested
	if decision.Cause == goal.ReplanCauseGovernanceDecision {
		return decision.SourceWorkItemRef.String() != "" && decision.SourceWorkItemRevision != 0 &&
			decision.SourceExecutionRef.String() != "" && decision.SourceExecutionAttempt != 0 &&
			validCouncilDigest(string(decision.CouncilSubjectDigest)) && validCouncilRef(decision.CouncilDecisionRef) &&
			validCouncilDigest(string(decision.CouncilDecisionDigest))
	}
	return baseCause && decision.CouncilSubjectDigest == "" && decision.CouncilDecisionRef == "" && decision.CouncilDecisionDigest == "" &&
		decision.SourceWorkItemRef.String() != "" && decision.SourceWorkItemRevision != 0 &&
		decision.SourceExecutionRef.String() != "" && decision.SourceExecutionAttempt != 0
}

func memoryDirectorScheduleValid(state ApplyDirectorPlanState) bool {
	if state.Decision.Cause == goal.ReplanCauseSplitPending {
		if len(state.UpdatedExecutions) != 1 || len(state.RetireActionRefs) != 1 ||
			state.UpdatedExecutions[0].Ref != state.Decision.SourceExecutionRef ||
			state.UpdatedExecutions[0].State != ExecutionCanceled {
			return false
		}
	} else if state.Decision.Cause == goal.ReplanCauseGovernanceDecision {
		if len(state.UpdatedExecutions) != 1 || len(state.RetireActionRefs) != 0 ||
			state.UpdatedExecutions[0].Ref != state.Decision.SourceExecutionRef ||
			state.UpdatedExecutions[0].State != ExecutionCanceled {
			return false
		}
	} else if len(state.UpdatedExecutions) != 0 || len(state.RetireActionRefs) != 0 {
		return false
	}
	seenExecutions := make(map[goal.ExecutionRef]struct{}, len(state.NewExecutions))
	for _, execution := range state.NewExecutions {
		if execution.Ref.String() == "" || execution.GoalRef != state.GoalRef ||
			execution.PlanGeneration != state.Goal.PlanGeneration() || execution.State != ExecutionQueued {
			return false
		}
		seenExecutions[execution.Ref] = struct{}{}
	}
	for _, action := range state.NewActions {
		if (action.Kind != ActionLaunchAgent && action.Kind != ActionPrepareWorkspace) || action.GoalRef != state.GoalRef ||
			action.PlanGeneration != state.Goal.PlanGeneration() {
			return false
		}
		if _, exists := seenExecutions[action.ExecutionRef]; !exists {
			return false
		}
	}
	return len(state.NewExecutions) == len(state.NewActions)
}

func memoryDirectorRequestKey(
	kind string,
	principal identity.PrincipalRef,
	project goal.ProjectRef,
	requestRef string,
) string {
	return strings.Join([]string{kind, principal.String(), project.String(), requestRef}, "\x00")
}

func directorLeaseWithoutToken(lease DirectorLeaseRecord) DirectorLeaseRecord {
	lease.Token = ""
	return lease
}

func (repository *memoryRepository) activeDirectorLeaseReplay(
	historical DirectorLeaseRecord,
) (DirectorLeaseRecord, bool) {
	now := repository.now().UTC()
	current, exists := repository.directorLeases[historical.GoalRef]
	if !exists || historical.Token != "" || !historical.LeaseUntil.After(now) ||
		!current.LeaseUntil.After(now) || current.PrincipalRef != historical.PrincipalRef ||
		current.Fence != historical.Fence || current.Token == "" {
		return DirectorLeaseRecord{}, false
	}
	historical.Token = current.Token
	return historical, true
}
