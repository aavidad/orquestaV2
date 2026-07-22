package application

import (
	"errors"
	"strconv"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func workItemAuthorities(
	items []goal.WorkItem,
	principal identity.PrincipalRef,
	permission identity.Permission,
	source EffectApprovalSource,
	receipt identity.AuthorizationReceipt,
	at time.Time,
) []WorkItemAuthority {
	authorities := make([]WorkItemAuthority, len(items))
	for index, item := range items {
		authorities[index] = WorkItemAuthority{
			WorkItemRef: item.Ref(), PrincipalRef: principal, Permission: permission,
			Source: source, AuthorizationReceipt: receipt, RecordedAt: at.UTC(),
		}
	}
	return authorities
}

func workItemAuthorityFor(authorities []WorkItemAuthority, ref goal.WorkItemRef) (WorkItemAuthority, bool) {
	for _, authority := range authorities {
		if authority.WorkItemRef == ref {
			return authority, true
		}
	}
	return WorkItemAuthority{}, false
}

func (orchestrator *Orchestrator) launchAction(
	policy effectPolicySnapshot,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	authority WorkItemAuthority,
	at time.Time,
	availableAt time.Time,
) (ActionRecord, error) {
	actionRef := "action:launch:" + execution.Ref.String()
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: effectAdmissionFingerprint(actionRef, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionLaunchAgent, Kind: EffectKindAgentLaunch,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: authority.PrincipalRef,
		Permission: authority.Permission, Authority: authority.AuthorizationReceipt,
		Demand:              item.BudgetDemand(),
		SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash:      policy.PolicyHash,
		PolicyRevision:  policy.PolicyRevision,
		QuotaRetryDelay: policy.QuotaRetryDelay,
		ApprovalTTL:     policy.ApprovalTTL,
		TargetDigest:    launchTargetDigest(launchEffectTargetRequest(aggregate, item, execution)),
		IdempotencyKey:  execution.IdempotencyKey,
		CreatedAt:       at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionLaunchAgent, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef:   execution.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: availableAt,
	}, authority.Source, at)
}

func (orchestrator *Orchestrator) prepareWorkspaceAction(
	policy effectPolicySnapshot,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	authority WorkItemAuthority,
	at time.Time,
	availableAt time.Time,
) (ActionRecord, error) {
	actionRef := "action:prepare-workspace:" + execution.Ref.String()
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: effectAdmissionFingerprint(actionRef, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionPrepareWorkspace, Kind: EffectKindPrepareWorkspace,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: authority.PrincipalRef,
		Permission: authority.Permission, Authority: authority.AuthorizationReceipt,
		Demand:              governance.BudgetDemand{Ref: "budget-demand:" + actionRef},
		SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.PolicyRevision,
		QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.ApprovalTTL,
		TargetDigest:   workspacePrepareTargetDigest(aggregate, item, execution),
		IdempotencyKey: "workspace:" + execution.IdempotencyKey, CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionPrepareWorkspace, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AvailableAt: availableAt,
	}, authority.Source, at)
}

func (orchestrator *Orchestrator) commitChangeAction(
	policy effectPolicySnapshot,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	binding WorkspaceBinding,
	changeRef ports.ChangeSetRef,
	authority WorkItemAuthority,
	at time.Time,
) (ActionRecord, error) {
	actionRef := "action:commit-change:" + execution.Ref.String()
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: effectAdmissionFingerprint(actionRef, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionCommitChange, Kind: EffectKindCommitChange,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: authority.PrincipalRef,
		Permission: authority.Permission, Authority: authority.AuthorizationReceipt,
		Demand:              governance.BudgetDemand{Ref: "budget-demand:" + actionRef},
		SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.PolicyRevision,
		QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.ApprovalTTL,
		TargetDigest:   commitChangeTargetDigest(binding, changeRef, execution),
		IdempotencyKey: "commit:" + execution.IdempotencyKey, CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionCommitChange, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ChangeRef: changeRef, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AvailableAt: at,
	}, authority.Source, at)
}

func (orchestrator *Orchestrator) attestTestAction(
	policy effectPolicySnapshot,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	binding WorkspaceBinding,
	change ChangeSet,
	authority WorkItemAuthority,
	at time.Time,
) (ActionRecord, error) {
	subject, err := buildTestSubject(aggregate, item, execution, binding, change, orchestrator.testAttestationPolicy)
	if err != nil {
		return ActionRecord{}, err
	}
	actionRef := "action:attest-test:" + execution.Ref.String()
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: effectAdmissionFingerprint(actionRef, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionAttestTest, Kind: EffectKindAttestTest,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: authority.PrincipalRef,
		Permission: authority.Permission, Authority: authority.AuthorizationReceipt,
		Demand:              governance.BudgetDemand{Ref: "budget-demand:" + actionRef},
		SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.PolicyRevision,
		QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.ApprovalTTL,
		TargetDigest: ports.TestSubjectDigest(subject), IdempotencyKey: "attest:" + change.Ref.String(),
		CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionAttestTest, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ChangeRef: change.Ref, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AvailableAt: at.UTC(),
	}, authority.Source, at)
}

func (orchestrator *Orchestrator) integrateChangeAction(
	policy effectPolicySnapshot,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	change ChangeSet,
	expectedTargetOID string,
	principal identity.PrincipalRef,
	authority identity.AuthorizationReceipt,
	requestRef string,
	requestFingerprint string,
	at time.Time,
) (ActionRecord, error) {
	actionRef := integrationActionRef(principal, aggregate.Project(), requestRef)
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: requestRef,
		RequestFingerprint: requestFingerprint,
		ActionRef:          actionRef, ActionKind: ActionIntegrateChange, Kind: EffectKindIntegrateChange,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: principal,
		Permission: identity.PermissionChangesIntegrate, Authority: authority,
		Demand:              governance.BudgetDemand{Ref: "budget-demand:" + actionRef},
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortLow,
		PolicyHash:          policy.PolicyHash, PolicyRevision: policy.PolicyRevision,
		QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.ApprovalTTL,
		TargetDigest:   integrationTargetDigest(change, expectedTargetOID),
		IdempotencyKey: "integration:" + requestRef, CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionIntegrateChange, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ChangeRef: change.Ref, ExpectedTargetOID: expectedTargetOID,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: at,
	}, EffectApprovalSourceIntegrationDecision, at)
}

func integrationActionRef(principal identity.PrincipalRef, project goal.ProjectRef, requestRef string) string {
	return "action:integrate-change:" + fingerprintFields(
		"orquesta.integrate-change.action.v1", principal.String(), project.String(), requestRef,
	)
}

func (orchestrator *Orchestrator) stopAction(
	policy effectPolicySnapshot,
	control ControlRecord,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution ExecutionRecord,
	at time.Time,
) (ActionRecord, error) {
	actionRef := "action:stop:" + control.Ref + ":" + execution.Ref.String()
	criticality := governance.SecurityCriticalityNormal
	if control.Mode == ports.AgentStopForced {
		criticality = governance.SecurityCriticalitySensitive
	}
	intent := EffectIntent{
		Ref: "effect-intent:" + actionRef, RequestRef: control.RequestRef,
		RequestFingerprint: effectAdmissionFingerprint(actionRef, control.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          actionRef, ActionKind: ActionStopAgent, Kind: EffectKindAgentStop,
		Subject: effectSubject(aggregate, item, execution), ProposedBy: control.PrincipalRef,
		Permission: identity.PermissionGoalsDirect, Authority: control.AuthorizationReceipt,
		Demand:              governance.BudgetDemand{Ref: "budget-demand:" + actionRef},
		SecurityCriticality: criticality, ReasoningEffort: governance.ReasoningEffortLow,
		PolicyHash:      policy.PolicyHash,
		PolicyRevision:  policy.PolicyRevision,
		QuotaRetryDelay: policy.QuotaRetryDelay,
		ApprovalTTL:     policy.ApprovalTTL,
		TargetDigest:    stopTargetDigest(control, stopRequest(control, execution)),
		IdempotencyKey:  "stop:" + control.Ref + ":" + execution.Ref.String(), CreatedAt: at.UTC(),
	}
	return orchestrator.finalizeEffectAction(intent, ActionRecord{
		Ref: actionRef, Kind: ActionStopAgent, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ControlRef: control.Ref,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: at,
	}, EffectApprovalSourceDirectorDecision, at)
}

func (orchestrator *Orchestrator) finalizeEffectAction(intent EffectIntent, action ActionRecord, source EffectApprovalSource, at time.Time) (ActionRecord, error) {
	intent.Digest = EffectIntentDigest(intent)
	action.EffectIntentRef, action.EffectIntent = intent.Ref, intent
	approval, found, err := orchestrator.automaticApproval(intent, source, at)
	if err != nil {
		return ActionRecord{}, err
	}
	if found {
		action.EffectApproval = &approval
	}
	if err := ValidateEffectIntent(intent); err != nil {
		return ActionRecord{}, err
	}
	return action, nil
}

func launchEffectTargetRequest(aggregate goal.Goal, item goal.WorkItem, execution ExecutionRecord) ports.AgentLaunchRequest {
	return ports.AgentLaunchRequest{
		ExecutionRef: execution.Ref, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		ExecutionAttempt: execution.AttemptNo, SpecHash: execution.SpecHash,
		ActorRef: aggregate.Actor(), ProjectRef: aggregate.Project(), IdempotencyKey: execution.IdempotencyKey,
		ExecutionWorkspaceRef: execution.ExecutionWorkspaceRef,
	}
}

func (orchestrator *Orchestrator) automaticApproval(
	intent EffectIntent,
	source EffectApprovalSource,
	at time.Time,
) (EffectApproval, bool, error) {
	if intent.SecurityCriticality != governance.SecurityCriticalityNormal {
		return EffectApproval{}, false, nil
	}
	approval := EffectApproval{
		Ref: "effect-approval:auto:" + intent.Ref, RequestRef: intent.RequestRef,
		RequestFingerprint: intent.Digest, IntentRef: intent.Ref, IntentDigest: intent.Digest,
		Subject: intent.Subject, ProposedBy: intent.ProposedBy, DecidedBy: intent.ProposedBy,
		Decision: EffectApproved, Source: source, SecurityCriticality: intent.SecurityCriticality,
		PolicyHash: intent.PolicyHash, PolicyRevision: intent.PolicyRevision, TargetDigest: intent.TargetDigest,
		Reason: "application.effect_auto_approved_normal", IdempotencyKey: intent.IdempotencyKey,
		AuthorizationReceipt: intent.Authority, DecidedAt: at.UTC(),
	}
	if err := ValidateEffectApproval(intent, approval); err != nil {
		return EffectApproval{}, false, err
	}
	return approval, true, nil
}

func effectSubject(aggregate goal.Goal, item goal.WorkItem, execution ExecutionRecord) EffectSubject {
	return EffectSubject{
		ProjectRef: aggregate.Project(), GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, ActorRef: aggregate.Actor(),
	}
}

func effectAdmissionFingerprint(fields ...string) string {
	return fingerprintFields("orquesta.effect.admission.v1", fields...)
}

func launchTargetDigest(request ports.AgentLaunchRequest) string {
	return effectAdmissionFingerprint(
		"target:launch:v1", request.ProjectRef.String(), request.GoalRef.String(), request.WorkItemRef.String(),
		request.ExecutionRef.String(), strconv.FormatUint(uint64(request.PlanGeneration), 10),
		strconv.FormatUint(uint64(request.AppSpecGeneration), 10), strconv.FormatUint(request.ExecutionAttempt, 10),
		request.SpecHash, request.ActorRef.String(), request.ExecutionWorkspaceRef.String(), request.IdempotencyKey,
	)
}

func stopTargetDigest(control ControlRecord, request ports.AgentStopRequest) string {
	return effectAdmissionFingerprint(
		"target:stop:v1", string(control.Operation), string(control.Target), string(request.Mode),
		request.ExecutionRef.String(), request.GoalRef.String(), request.WorkItemRef.String(),
		strconv.FormatUint(uint64(request.PlanGeneration), 10), strconv.FormatUint(uint64(request.AppSpecGeneration), 10),
		strconv.FormatUint(request.ExecutionAttempt, 10), request.SpecHash, request.IdempotencyKey,
	)
}

func workspacePrepareTargetDigest(aggregate goal.Goal, item goal.WorkItem, execution ExecutionRecord) string {
	fields := []string{
		"target:workspace-prepare:v1", execution.RepositoryRef.String(), aggregate.Project().String(),
		aggregate.Ref().String(), item.Ref().String(), execution.Ref.String(),
		execution.ExecutionWorkspaceRef.String(),
		strconv.FormatUint(uint64(execution.PlanGeneration), 10),
		strconv.FormatUint(uint64(execution.AppSpecGeneration), 10),
		strconv.FormatUint(execution.AttemptNo, 10), execution.SpecHash,
	}
	for _, scope := range item.WriteSet() {
		fields = append(fields, scope.String())
	}
	return effectAdmissionFingerprint(fields...)
}

func commitChangeTargetDigest(binding WorkspaceBinding, changeRef ports.ChangeSetRef, execution ExecutionRecord) string {
	return effectAdmissionFingerprint(
		"target:commit-change:v1", changeRef.String(), binding.Ref.String(), binding.RepositoryRef.String(),
		binding.BaseOID, binding.WriteSetDigest, execution.Ref.String(), execution.SpecHash,
	)
}

func integrationTargetDigest(change ChangeSet, expectedTargetOID string) string {
	return effectAdmissionFingerprint(
		"target:integrate-change:v1", change.Ref.String(), change.RepositoryRef.String(),
		change.HeadOID, change.TreeOID, expectedTargetOID, change.DiffDigest,
	)
}

func validateWorkItemAuthority(aggregate goal.Goal, authority WorkItemAuthority) error {
	request := authority.AuthorizationReceipt.Decision().Request()
	resourceRef := aggregate.Ref().String()
	if authority.Permission == identity.PermissionGoalsCreate {
		resourceRef = aggregate.Project().String()
	}
	if authority.WorkItemRef.String() == "" || authority.PrincipalRef.String() == "" || authority.RecordedAt.IsZero() ||
		!((authority.Source == EffectApprovalSourceGoalConfirmation && authority.Permission == identity.PermissionGoalsCreate) ||
			(authority.Source == EffectApprovalSourceDirectorDecision && authority.Permission == identity.PermissionGoalsDirect)) ||
		request.Principal().Ref != authority.PrincipalRef || request.ProjectRef() != aggregate.Project() ||
		request.Permission() != authority.Permission || request.ResourceRef() != resourceRef ||
		authority.AuthorizationReceipt.Decision().Outcome() != identity.AuthorizationAllowed {
		return errors.New("application.work_item_authority_invalid")
	}
	return nil
}
