package application

import (
	"errors"
	"strconv"
	"time"

	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func (orchestrator *Orchestrator) autoCouncilOpenState(record GoalRecord, item goal.WorkItem,
	author ExecutionRecord, change ChangeSet, at time.Time,
) (*OpenCouncilRoundState, error) {
	policy, present := item.CouncilPolicy()
	if !present || policy != council.PolicyAuto {
		return nil, nil
	}
	if author.State != ExecutionAwaitingIntegration {
		return nil, errors.New("council.subject_invalid")
	}
	subject, err := councilSubject(record, item, author, change, policy, orchestrator.testAttestationPolicy)
	if err != nil {
		return nil, err
	}
	digest, ok := councilDigest(subject.Digest())
	if !ok {
		return nil, errors.New("council.subject_invalid")
	}
	if _, exists := councilRoundFor(record, digest); exists {
		return nil, nil
	}
	for _, skip := range record.CouncilSkips {
		if skip.SubjectDigest == digest {
			return nil, nil
		}
	}
	authority, found := workItemAuthorityFor(record.WorkItemAuthorities, item.Ref())
	if !found {
		return nil, errors.New("application.work_item_authority_missing")
	}
	requestRef := authority.AuthorizationReceipt.Decision().Request().RequestRef()
	fingerprint := effectAdmissionFingerprint("orquesta.council.auto-open.v1", requestRef, string(digest))
	return orchestrator.councilOpenState(record, item, author, change, subject, digest, authority,
		CouncilRoundOpenerAuto, requestRef, fingerprint, "", 0, at)
}

func councilExecution(aggregate goal.Goal, item goal.WorkItem, author ExecutionRecord, digest CouncilSubjectDigest,
	role council.Role, at time.Time, maxAttempts uint64, maxOutputBytes int64,
) (ExecutionRecord, error) {
	purpose, ok := councilPurpose(role)
	if !ok {
		return ExecutionRecord{}, errors.New("council.role_invalid")
	}
	ref, err := goal.NewExecutionRef("execution:council:" + fingerprintFields("orquesta.council-execution.v1",
		aggregate.Ref().String(), item.Ref().String(), string(digest), string(role), "1"))
	if err != nil {
		return ExecutionRecord{}, err
	}
	return ExecutionRecord{Ref: ref, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), AttemptNo: 1,
		MaxExecutionAttempts: maxAttempts, PlanGeneration: author.PlanGeneration, AppSpecGeneration: author.AppSpecGeneration,
		SpecHash: author.SpecHash, State: ExecutionQueued, Purpose: purpose, CouncilSubjectDigest: digest,
		RepositoryRef: author.RepositoryRef, ExecutionWorkspaceRef: author.ExecutionWorkspaceRef,
		ArtifactMediaType: council.ContributionMediaType, IdempotencyKey: "execution:" + ref.String(),
		MaxOutputBytes: maxOutputBytes, CreatedAt: at.UTC()}, nil
}

func (orchestrator *Orchestrator) councilLaunchAction(policy effectPolicySnapshot, record GoalRecord,
	item goal.WorkItem, execution ExecutionRecord, authority WorkItemAuthority, subject council.Subject, at, availableAt time.Time,
) (ActionRecord, error) {
	phase, found := phaseForWorkItem(record.Goal, item)
	if !found {
		return ActionRecord{}, errors.New("application.phase_missing")
	}
	request, err := councilAgentLaunchRequestForSubject(record, item, execution, phase, subject)
	if err != nil {
		return ActionRecord{}, err
	}
	actionRef := "action:launch:" + execution.Ref.String()
	intent := EffectIntent{Ref: "effect-intent:" + actionRef,
		RequestRef:         authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: effectAdmissionFingerprint(actionRef, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionLaunchAgent, Kind: EffectKindAgentLaunch,
		Subject: effectSubject(record.Goal, item, execution), ProposedBy: authority.PrincipalRef,
		Permission: authority.Permission, Authority: authority.AuthorizationReceipt, Demand: item.BudgetDemand(),
		SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.PolicyRevision, QuotaRetryDelay: policy.QuotaRetryDelay,
		ApprovalTTL: policy.ApprovalTTL, TargetDigest: reviewerLaunchTargetDigest(request),
		IdempotencyKey: execution.IdempotencyKey, CreatedAt: at.UTC()}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{Ref: actionRef, Kind: ActionLaunchAgent,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: availableAt},
		authority.Source, at)
}

func councilRetryRef(execution ExecutionRecord, role council.Role) (goal.ExecutionRef, error) {
	return goal.NewExecutionRef("execution:council:" + fingerprintFields("orquesta.council-execution.v1",
		execution.GoalRef.String(), execution.WorkItemRef.String(), string(execution.CouncilSubjectDigest), string(role),
		strconv.FormatUint(execution.AttemptNo+1, 10)))
}
