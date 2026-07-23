package application

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type OpenCouncilRoundRequest struct {
	RequestRef           string
	GoalRef              goal.GoalRef
	ChangeRef            ports.ChangeSetRef
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	LeaseToken           string
	LeaseFence           uint64
}

type OpenCouncilRoundResult struct {
	Round   CouncilRoundRecord
	Created bool
}

func (orchestrator *Orchestrator) OpenCouncilRound(ctx context.Context, access Access, request OpenCouncilRoundRequest) (OpenCouncilRoundResult, error) {
	if orchestrator == nil || !validApplicationRef(request.RequestRef) || request.GoalRef.String() == "" || request.ChangeRef.String() == "" || request.LeaseToken == "" || request.LeaseFence == 0 {
		return OpenCouncilRoundResult{}, errors.New("council.open_request_invalid")
	}
	principal, project, err := access.values()
	if err != nil {
		return OpenCouncilRoundResult{}, err
	}
	now := orchestrator.clock.Now().UTC()
	fingerprint := fingerprintFields("orquesta.council.open.v1", principal.Ref.String(), project.String(), request.RequestRef, request.GoalRef.String(), request.ChangeRef.String(), request.LeaseToken, strconv.FormatUint(request.LeaseFence, 10))
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(ctx, access, identity.PermissionGoalsDirect, request.GoalRef.String(), now, "authorization-request:council-open:"+request.RequestRef)
	if err != nil {
		return OpenCouncilRoundResult{}, err
	}
	now = authorization.RecordedAt().UTC()
	record, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return OpenCouncilRoundResult{}, err
	}
	if record.Goal.Project() != project || record.Goal.Revision() != request.ExpectedGoalRevision {
		return OpenCouncilRoundResult{}, &StateError{Code: StateConflict}
	}
	change, found := changeSetByRef(record, request.ChangeRef)
	if !found {
		return OpenCouncilRoundResult{}, &StateError{Code: StateNotFound}
	}
	item, found := record.Goal.WorkItem(change.WorkItemRef)
	author, authorFound := executionByRef(record.Executions, change.ExecutionRef)
	policy, hasPolicy := item.CouncilPolicy()
	if !found || !authorFound || item.Revision() != request.ExpectedItemRevision || policy != council.PolicyRequired || !hasPolicy {
		return OpenCouncilRoundResult{}, &StateError{Code: StateConflict}
	}
	subject, err := councilSubject(record, item, author, change, policy, orchestrator.testAttestationPolicy)
	if err != nil {
		return OpenCouncilRoundResult{}, &StateError{Code: StateConflict, Cause: err}
	}
	digest, ok := councilDigest(subject.Digest())
	if !ok {
		return OpenCouncilRoundResult{}, errors.New("council.subject_invalid")
	}
	state, err := orchestrator.councilOpenState(record, item, author, change, subject, digest, WorkItemAuthority{WorkItemRef: item.Ref(), PrincipalRef: principal.Ref, Permission: identity.PermissionGoalsDirect, Source: EffectApprovalSourceDirectorDecision, AuthorizationReceipt: authorization, RecordedAt: now}, CouncilRoundOpenerDirector, request.RequestRef, fingerprint, request.LeaseToken, request.LeaseFence, now)
	if err != nil {
		return OpenCouncilRoundResult{}, err
	}
	state.ExpectedGoalRevision, state.ExpectedItemRevision = request.ExpectedGoalRevision, request.ExpectedItemRevision
	round, created, err := orchestrator.state.OpenCouncilRound(ctx, *state)
	if err != nil {
		return OpenCouncilRoundResult{}, err
	}
	return OpenCouncilRoundResult{Round: round, Created: created}, nil
}

type SkipCouncilRequest struct {
	RequestRef           string
	GoalRef              goal.GoalRef
	ChangeRef            ports.ChangeSetRef
	ExpectedGoalRevision goal.Revision
	ExpectedItemRevision goal.Revision
	Reason               string
}
type SkipCouncilResult struct {
	Skip    CouncilSkipRecord
	Created bool
}

func (orchestrator *Orchestrator) SkipCouncil(ctx context.Context, access Access, request SkipCouncilRequest) (SkipCouncilResult, error) {
	if orchestrator == nil || !validApplicationRef(request.RequestRef) || request.GoalRef.String() == "" || request.ChangeRef.String() == "" || request.Reason == "" || request.Reason != strings.TrimSpace(request.Reason) {
		return SkipCouncilResult{}, errors.New("council.skip_request_invalid")
	}
	principal, project, err := access.values()
	if err != nil {
		return SkipCouncilResult{}, err
	}
	if principal.Kind != identity.PrincipalKindHuman {
		return SkipCouncilResult{}, errForbidden
	}
	now := orchestrator.clock.Now().UTC()
	fingerprint := fingerprintFields("orquesta.council.skip.v1", principal.Ref.String(), project.String(), request.RequestRef, request.GoalRef.String(), request.ChangeRef.String(), request.Reason)
	authorization, err := orchestrator.authorizeIdempotentWithRequestRef(ctx, access, identity.PermissionCouncilSkip, request.GoalRef.String(), now, "authorization-request:council-skip:"+request.RequestRef)
	if err != nil {
		return SkipCouncilResult{}, err
	}
	now = authorization.RecordedAt().UTC()
	record, err := orchestrator.state.GetGoal(ctx, request.GoalRef)
	if err != nil {
		return SkipCouncilResult{}, err
	}
	change, found := changeSetByRef(record, request.ChangeRef)
	if !found || record.Goal.Project() != project || record.Goal.Revision() != request.ExpectedGoalRevision {
		return SkipCouncilResult{}, &StateError{Code: StateConflict}
	}
	item, found := record.Goal.WorkItem(change.WorkItemRef)
	author, authorFound := executionByRef(record.Executions, change.ExecutionRef)
	policy, present := item.CouncilPolicy()
	if !found || !authorFound || !present || policy != council.PolicySkipByOperator || item.Revision() != request.ExpectedItemRevision {
		return SkipCouncilResult{}, &StateError{Code: StateConflict}
	}
	subject, err := councilSubject(record, item, author, change, policy, orchestrator.testAttestationPolicy)
	if err != nil {
		return SkipCouncilResult{}, &StateError{Code: StateConflict, Cause: err}
	}
	skip, err := council.NewSkip(subject, council.Skip{PrincipalRef: principal.Ref.String(), Reason: request.Reason, SpecHash: subject.SpecHash, IdempotencyKey: "council-skip:" + request.RequestRef, CouncilSubjectDigest: subject.Digest(), RecordedAtUTC: now})
	if err != nil {
		return SkipCouncilResult{}, err
	}
	recordFact := CouncilSkipRecord{Ref: "council-skip:" + string(mustCouncilSubjectDigest(subject)), Subject: subject, SubjectDigest: mustCouncilSubjectDigest(subject), Skip: skip, SkipDigest: CouncilSubjectDigest(skip.Digest()), RecordedAt: now, RequestRef: request.RequestRef, RequestFingerprint: fingerprint, AuthorizationReceiptRef: authorization.Ref()}
	persisted, created, err := orchestrator.state.RecordCouncilSkip(ctx, CouncilSkipState{RequestRef: request.RequestRef, RequestFingerprint: fingerprint, AuthorizationReceipt: authorization, ProjectRef: project, GoalRef: request.GoalRef, ExpectedGoalRevision: request.ExpectedGoalRevision, ExpectedItemRevision: request.ExpectedItemRevision, PrincipalRef: principal.Ref, RoundSubject: subject, Skip: skip, Record: recordFact, OperationAt: now})
	if err != nil {
		return SkipCouncilResult{}, err
	}
	return SkipCouncilResult{Skip: persisted, Created: created}, nil
}

func councilIntegrationResolution(record GoalRecord, item goal.WorkItem, execution ExecutionRecord, change ChangeSet, gateDigest string, testPolicy TestAttestationPolicy) (*CouncilResolution, error) {
	policy, ok := item.CouncilPolicy()
	if !ok {
		return nil, errors.New("council.policy_required")
	}
	subject, err := councilSubject(record, item, execution, change, policy, testPolicy)
	if err != nil || subject.ReviewGateDigest != gateDigest {
		return nil, errors.New("council.subject_invalid")
	}
	digest := mustCouncilSubjectDigest(subject)
	switch policy {
	case council.PolicyAuto, council.PolicyRequired:
		for _, decision := range record.CouncilDecisions {
			if decision.SubjectDigest == digest && decision.Decision.Outcome == council.OutcomeAccepted {
				return &CouncilResolution{SubjectDigest: digest, DecisionRef: decision.Ref, DecisionDigest: decision.DecisionDigest}, nil
			}
		}
	case council.PolicySkipByOperator:
		for _, skip := range record.CouncilSkips {
			if skip.SubjectDigest == digest {
				return &CouncilResolution{SubjectDigest: digest, SkipRef: skip.Ref, SkipDigest: skip.SkipDigest}, nil
			}
		}
	}
	return nil, errors.New("council.resolution_required")
}

func (orchestrator *Orchestrator) councilOpenState(record GoalRecord, item goal.WorkItem, author ExecutionRecord, change ChangeSet, subject council.Subject, digest CouncilSubjectDigest, authority WorkItemAuthority, opener CouncilRoundOpener, requestRef, fingerprint, leaseToken string, leaseFence uint64, at time.Time) (*OpenCouncilRoundState, error) {
	policy, err := historicalEffectPolicy(record)
	if err != nil {
		return nil, err
	}
	round := CouncilRoundRecord{Ref: "council-round:" + fingerprintFields("orquesta.council-round.v1", record.Goal.Ref().String(), item.Ref().String(), string(digest)), GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ChangeSetRef: change.Ref.String(), Subject: subject, SubjectDigest: digest, OpenedBy: authority.PrincipalRef, OpenedAt: at.UTC(), IdempotencyKey: "council-round:" + string(digest), Opener: opener, DirectorFence: leaseFence, RequestRef: requestRef, RequestFingerprint: fingerprint, AuthorizationReceiptRef: authority.AuthorizationReceipt.Ref()}
	executions := make([]ExecutionRecord, 0, 3)
	actions := make([]ActionRecord, 0, 3)
	events := make([]EventRecord, 0, 3)
	for _, role := range council.Roles() {
		execution, execErr := councilExecution(record.Goal, item, author, digest, role, at, orchestrator.maxExecutionAttempts, orchestrator.maxOutputBytes)
		if execErr != nil {
			return nil, execErr
		}
		scheduled := record
		scheduled.Executions = append(append([]ExecutionRecord(nil), record.Executions...), executions...)
		scheduled.Executions = append(scheduled.Executions, execution)
		action, actionErr := orchestrator.councilLaunchAction(policy, scheduled, item, execution, authority, subject, at, at)
		if actionErr != nil {
			return nil, actionErr
		}
		executions, actions = append(executions, execution), append(actions, action)
		events = append(events, EventRecord{Ref: "event:council-queued:" + execution.Ref.String(), Kind: "council.queued", GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at.UTC()})
	}
	return &OpenCouncilRoundState{RequestRef: requestRef, RequestFingerprint: fingerprint, AuthorizationReceipt: authority.AuthorizationReceipt, PrincipalRef: authority.PrincipalRef, ProjectRef: record.Goal.Project(), GoalRef: record.Goal.Ref(), LeaseToken: leaseToken, LeaseFence: leaseFence, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), Round: round, Executions: executions, Actions: actions, Events: events, OperationAt: at.UTC()}, nil
}
