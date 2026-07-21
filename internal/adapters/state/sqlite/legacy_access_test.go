package sqlite

import (
	"context"
	"strconv"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

// authorizeLegacyCreateState keeps pre-V10 state-adapter fixtures focused on
// their original lifecycle assertions while exercising the real V10 access
// path. New V10 tests construct identity explicitly instead.
func authorizeLegacyCreateState(
	t *testing.T,
	repository *Repository,
	state application.CreateGoalState,
) application.CreateGoalState {
	t.Helper()
	principal := legacyTestPrincipal(t, state.RequestedBy, state.Goal.Actor())
	ensureLegacyProjectAccess(t, repository, principal, state.Goal.Project(), state.Goal.CreatedAt())
	state.RequestedBy = principal.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, principal, state.Goal.Project(), identity.PermissionGoalsCreate,
		state.Goal.Project().String(), legacyAuthorizationRequestRef("create", principal.Ref, state.Goal.Project(), state.RequestRef),
		state.Goal.CreatedAt(),
	)
	return state
}

// governLegacyCreateState upgrades low-level pre-V15 fixtures at their shared
// constructor boundary. Tests that explicitly build a pre-010 database still
// persist legacy rows because those columns/tables do not exist there.
func governLegacyCreateState(t *testing.T, state application.CreateGoalState) application.CreateGoalState {
	t.Helper()
	policy := sqliteTestBudgetPolicy(state.Goal.CreatedAt())
	snapshot := state.Goal.Snapshot()
	for index := range snapshot.WorkItems {
		if snapshot.WorkItems[index].BudgetDemand.Resources == (governance.ResourceVector{}) {
			snapshot.WorkItems[index].BudgetDemand.Resources = policy.DefaultWorkItemDemand
		}
	}
	var err error
	state.Goal, err = goal.RestoreGoal(snapshot)
	if err != nil {
		t.Fatalf("govern legacy Goal: %v", err)
	}
	state.WorkItemAuthorities = make([]application.WorkItemAuthority, len(snapshot.WorkItems))
	for index, item := range state.Goal.WorkItems() {
		state.WorkItemAuthorities[index] = application.WorkItemAuthority{
			WorkItemRef: item.Ref(), PrincipalRef: state.RequestedBy, Permission: identity.PermissionGoalsCreate,
			Source:               application.EffectApprovalSourceGoalConfirmation,
			AuthorizationReceipt: state.AuthorizationReceipt, RecordedAt: state.Goal.CreatedAt(),
		}
	}
	deployment, project, goalEnvelope := policy.DeploymentEnvelope, policy.ProjectEnvelopeTemplate, policy.GoalEnvelopeTemplate
	project.Ref, project.SubjectRef = "budget-envelope:project:"+state.Goal.Project().String()+":"+policy.PolicyHash, state.Goal.Project().String()
	goalEnvelope.Ref, goalEnvelope.SubjectRef, goalEnvelope.CreatedAt = "budget-envelope:goal:"+state.Goal.Ref().String()+":"+policy.PolicyHash, state.Goal.Ref().String(), state.Goal.CreatedAt()
	state.BudgetEnvelopes = []governance.BudgetEnvelope{deployment, project, goalEnvelope}
	for index, action := range state.Actions {
		if action.Kind != application.ActionLaunchAgent {
			continue
		}
		item, itemFound := state.Goal.WorkItem(action.WorkItemRef)
		execution, executionFound := legacyExecutionByRef(state.Executions, action.ExecutionRef)
		if !itemFound || !executionFound {
			t.Fatalf("govern legacy action scope missing: %s", action.Ref)
		}
		intent := application.EffectIntent{
			Ref: "effect-intent:" + action.Ref, RequestRef: state.AuthorizationReceipt.Decision().Request().RequestRef(),
			RequestFingerprint: canonicalFingerprint("sqlite.test.effect.v15", action.Ref, state.AuthorizationReceipt.Ref(), policy.PolicyHash),
			ActionRef:          action.Ref, ActionKind: action.Kind, Kind: application.EffectKindAgentLaunch,
			Subject: application.EffectSubject{ProjectRef: state.Goal.Project(), GoalRef: state.Goal.Ref(), WorkItemRef: item.Ref(),
				ExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration, AppSpecGeneration: execution.AppSpecGeneration,
				SpecHash: execution.SpecHash, ActorRef: state.Goal.Actor()},
			ProposedBy: state.RequestedBy, Permission: identity.PermissionGoalsCreate, Authority: state.AuthorizationReceipt,
			Demand: item.BudgetDemand(), SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
			PolicyHash: policy.PolicyHash, PolicyRevision: policy.GoalEnvelopeTemplate.Revision,
			QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.EffectApprovalTTL,
			TargetDigest: canonicalFingerprint(
				"orquesta.effect.admission.v1", "target:launch:v1", state.Goal.Project().String(), state.Goal.Ref().String(),
				item.Ref().String(), execution.Ref.String(), strconv.FormatUint(uint64(execution.PlanGeneration), 10),
				strconv.FormatUint(uint64(execution.AppSpecGeneration), 10), strconv.FormatUint(execution.AttemptNo, 10),
				execution.SpecHash, state.Goal.Actor().String(), execution.IdempotencyKey,
			),
			IdempotencyKey: execution.IdempotencyKey, CreatedAt: state.Goal.CreatedAt(),
		}
		intent.Digest = application.EffectIntentDigest(intent)
		approval := application.EffectApproval{
			Ref: "effect-approval:auto:" + intent.Ref, RequestRef: intent.RequestRef, RequestFingerprint: intent.Digest,
			IntentRef: intent.Ref, IntentDigest: intent.Digest, Subject: intent.Subject, ProposedBy: intent.ProposedBy,
			DecidedBy: intent.ProposedBy, Decision: application.EffectApproved,
			Source: application.EffectApprovalSourceGoalConfirmation, SecurityCriticality: intent.SecurityCriticality,
			PolicyHash: intent.PolicyHash, PolicyRevision: intent.PolicyRevision,
			TargetDigest: intent.TargetDigest,
			Reason:       "sqlite test causal authority", IdempotencyKey: intent.IdempotencyKey,
			AuthorizationReceipt: intent.Authority, DecidedAt: intent.CreatedAt,
		}
		state.Actions[index].EffectIntentRef, state.Actions[index].EffectIntent, state.Actions[index].EffectApproval = intent.Ref, intent, &approval
	}
	return state
}

func legacyExecutionByRef(records []application.ExecutionRecord, ref goal.ExecutionRef) (application.ExecutionRecord, bool) {
	for _, record := range records {
		if record.Ref == ref {
			return record, true
		}
	}
	return application.ExecutionRecord{}, false
}

func authorizeLegacyAmendState(
	t *testing.T,
	repository *Repository,
	state application.AmendGoalState,
) application.AmendGoalState {
	t.Helper()
	confirmationActor := state.Successor.AppSpec().ConfirmedBy()
	if confirmationActor != state.Successor.Actor() {
		state.RequestedBy = identity.PrincipalRef{}
	}
	principal := legacyTestPrincipal(t, state.RequestedBy, confirmationActor)
	ensureLegacyProjectAccess(t, repository, principal, state.ProjectRef, state.Successor.CreatedAt())
	state.RequestedBy = principal.Ref
	state.AuthorizationReceipt = authorizeTest(
		t, repository, principal, state.ProjectRef, identity.PermissionGoalsAmend,
		state.SourceGoalRef.String(), legacyAuthorizationRequestRef("amend", principal.Ref, state.ProjectRef, state.RequestRef),
		state.Successor.CreatedAt(),
	)
	return state
}

func ensureLegacyProjectAccess(
	t *testing.T,
	repository *Repository,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	at time.Time,
) {
	t.Helper()
	err := repository.ProvisionLocalAccess(
		context.Background(), principal, testHierarchy(t, projectRef), identity.RoleProjectOwner, at,
	)
	if err == nil {
		return
	}
	if !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("provision legacy access: %v", err)
	}
	current, membershipErr := repository.Membership(context.Background(), principal.Ref, projectRef)
	if membershipErr == nil && current.IsActive() && identity.RoleAllows(current.Role(), identity.PermissionGoalsCreate) {
		return
	}
	if membershipErr != nil && !application.IsStateError(membershipErr, application.StateNotFound) {
		t.Fatalf("read legacy membership: %v", membershipErr)
	}
	owner := readLegacyProjectOwner(t, repository, projectRef)
	authorization := authorizeTest(
		t, repository, owner, projectRef, identity.PermissionProjectMembershipManage,
		principal.Ref.String(), legacyAuthorizationRequestRef("membership", owner.Ref, projectRef, principal.Ref.String()), at,
	)
	expected := identity.MembershipRevision(0)
	if membershipErr == nil {
		expected = current.Revision()
	}
	request, requestErr := identity.NewMembershipGrantRequest(identity.MembershipGrantRequestInput{
		RequestRef: legacyAuthorizationRequestRef("membership-grant", owner.Ref, projectRef, principal.Ref.String()),
		Actor:      owner, TargetRef: principal.Ref, ProjectRef: projectRef,
		Role: identity.RoleProjectOwner, ExpectedRevision: expected, RequestedAt: at,
	})
	if requestErr != nil {
		t.Fatal(requestErr)
	}
	if _, _, changed, grantErr := repository.GrantMembership(context.Background(), application.MembershipGrantState{
		AuthorizationReceipt: authorization, Request: request, Target: principal,
	}); grantErr != nil || !changed {
		t.Fatalf("grant legacy project access changed=%v err=%v", changed, grantErr)
	}
}

func readLegacyProjectOwner(
	t *testing.T,
	repository *Repository,
	projectRef goal.ProjectRef,
) identity.Principal {
	t.Helper()
	var principalValue, actorValue, kind, method string
	err := repository.db.QueryRow(`
SELECT p.ref, p.actor_ref, p.kind, p.authentication_method
FROM project_memberships m
JOIN principals p ON p.ref = m.principal_ref
WHERE m.project_ref = ? AND m.status = 'active'
  AND m.role IN ('platform_admin', 'project_owner')
ORDER BY CASE m.role WHEN 'platform_admin' THEN 0 ELSE 1 END, p.ref
LIMIT 1`, projectRef.String()).Scan(&principalValue, &actorValue, &kind, &method)
	if err != nil {
		t.Fatalf("read legacy project owner: %v", err)
	}
	principalRef, err := identity.NewPrincipalRef(principalValue)
	if err != nil {
		t.Fatal(err)
	}
	actorRef, err := goal.NewActorRef(actorValue)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := identity.NewPrincipal(principalRef, actorRef, identity.PrincipalKind(kind), method)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func legacyTestPrincipal(
	t *testing.T,
	principalRef identity.PrincipalRef,
	actorRef goal.ActorRef,
) identity.Principal {
	t.Helper()
	var err error
	if principalRef.String() == "" {
		principalRef, err = identity.NewPrincipalRef(actorRef.String())
		if err != nil {
			t.Fatal(err)
		}
	}
	principal, err := identity.NewPrincipal(
		principalRef, actorRef, identity.PrincipalKindHuman, "sqlite_legacy_test",
	)
	if err != nil {
		t.Fatal(err)
	}
	return principal
}

func legacyAuthorizationRequestRef(
	kind string,
	principalRef identity.PrincipalRef,
	projectRef goal.ProjectRef,
	businessRequestRef string,
) string {
	return "legacy-authorization-" + canonicalFingerprint(
		"legacy-authorization.v1", kind, principalRef.String(), projectRef.String(), businessRequestRef,
	)
}

func createLegacyGoal(
	t *testing.T,
	repository *Repository,
	state application.CreateGoalState,
) (application.GoalRecord, bool, error) {
	t.Helper()
	state = authorizeLegacyCreateState(t, repository, state)
	state = governLegacyWorkspaceCreateState(t, state)
	return repository.CreateGoal(context.Background(), state)
}

// governLegacyWorkspaceCreateState upgrades only fixtures that already declare
// a WriteSet. It preserves their domain snapshot while making the V16
// prepare-before-launch boundary explicit and causally governed.
func governLegacyWorkspaceCreateState(t *testing.T, state application.CreateGoalState) application.CreateGoalState {
	t.Helper()
	hasWorkspace := false
	repositoryRef, err := identity.NewRepositoryRef("repository:" + state.Goal.Project().String())
	if err != nil {
		t.Fatal(err)
	}
	for index := range state.Executions {
		execution := &state.Executions[index]
		item, found := state.Goal.WorkItem(execution.WorkItemRef)
		if !found || len(item.WriteSet()) == 0 {
			continue
		}
		hasWorkspace = true
		workspaceRef, refErr := ports.NewExecutionWorkspaceRef("execution-workspace:" + execution.Ref.String())
		if refErr != nil {
			t.Fatal(refErr)
		}
		execution.RepositoryRef, execution.ExecutionWorkspaceRef = repositoryRef, workspaceRef
		for actionIndex := range state.Actions {
			action := &state.Actions[actionIndex]
			if action.ExecutionRef == execution.Ref {
				action.Ref = "action:prepare-workspace:" + execution.Ref.String()
				action.Kind = application.ActionPrepareWorkspace
			}
		}
	}
	if !hasWorkspace {
		return state
	}
	policy := sqliteTestBudgetPolicy(state.Goal.CreatedAt())
	state.WorkItemAuthorities = make([]application.WorkItemAuthority, 0, state.Goal.WorkItemCount())
	for _, item := range state.Goal.WorkItems() {
		state.WorkItemAuthorities = append(state.WorkItemAuthorities, application.WorkItemAuthority{
			WorkItemRef: item.Ref(), PrincipalRef: state.RequestedBy,
			Permission: identity.PermissionGoalsCreate, Source: application.EffectApprovalSourceGoalConfirmation,
			AuthorizationReceipt: state.AuthorizationReceipt, RecordedAt: state.Goal.CreatedAt(),
		})
	}
	deployment, project, goalEnvelope := policy.DeploymentEnvelope, policy.ProjectEnvelopeTemplate, policy.GoalEnvelopeTemplate
	project.Ref, project.SubjectRef = "budget-envelope:project:"+state.Goal.Project().String()+":"+policy.PolicyHash, state.Goal.Project().String()
	goalEnvelope.Ref, goalEnvelope.SubjectRef, goalEnvelope.CreatedAt = "budget-envelope:goal:"+state.Goal.Ref().String()+":"+policy.PolicyHash, state.Goal.Ref().String(), state.Goal.CreatedAt()
	state.BudgetEnvelopes = []governance.BudgetEnvelope{deployment, project, goalEnvelope}
	for index := range state.Actions {
		action := state.Actions[index]
		item, itemFound := state.Goal.WorkItem(action.WorkItemRef)
		execution, executionFound := legacyExecutionByRef(state.Executions, action.ExecutionRef)
		if !itemFound || !executionFound {
			t.Fatalf("govern legacy workspace scope missing: %s", action.Ref)
		}
		authority := state.WorkItemAuthorities[0]
		for _, candidate := range state.WorkItemAuthorities {
			if candidate.WorkItemRef == item.Ref() {
				authority = candidate
				break
			}
		}
		state.Actions[index] = legacyGovernedExecutionAction(t, state.Goal, item, execution, action, authority, policy)
	}
	return state
}

func legacyGovernedExecutionAction(
	t *testing.T,
	aggregate goal.Goal,
	item goal.WorkItem,
	execution application.ExecutionRecord,
	action application.ActionRecord,
	authority application.WorkItemAuthority,
	policy application.BudgetPolicy,
) application.ActionRecord {
	t.Helper()
	effectKind := application.EffectKindAgentLaunch
	demand, idempotency := item.BudgetDemand(), execution.IdempotencyKey
	targetFields := []string{
		"target:launch:v1", aggregate.Project().String(), aggregate.Ref().String(), item.Ref().String(),
		execution.Ref.String(), strconv.FormatUint(uint64(execution.PlanGeneration), 10),
		strconv.FormatUint(uint64(execution.AppSpecGeneration), 10), strconv.FormatUint(execution.AttemptNo, 10),
		execution.SpecHash, aggregate.Actor().String(), execution.ExecutionWorkspaceRef.String(), execution.IdempotencyKey,
	}
	if action.Kind == application.ActionPrepareWorkspace {
		effectKind = application.EffectKindPrepareWorkspace
		demand = governance.BudgetDemand{Ref: "budget-demand:" + action.Ref}
		idempotency = "workspace:" + execution.IdempotencyKey
		targetFields = []string{
			"target:workspace-prepare:v1", execution.RepositoryRef.String(), aggregate.Project().String(),
			aggregate.Ref().String(), item.Ref().String(), execution.Ref.String(), execution.ExecutionWorkspaceRef.String(),
			strconv.FormatUint(uint64(execution.PlanGeneration), 10),
			strconv.FormatUint(uint64(execution.AppSpecGeneration), 10), strconv.FormatUint(execution.AttemptNo, 10),
			execution.SpecHash,
		}
		for _, scope := range item.WriteSet() {
			targetFields = append(targetFields, scope.String())
		}
	}
	intent := application.EffectIntent{
		Ref:                "effect-intent:" + action.Ref,
		RequestRef:         authority.AuthorizationReceipt.Decision().Request().RequestRef(),
		RequestFingerprint: canonicalFingerprint("orquesta.effect.admission.v1", action.Ref, authority.AuthorizationReceipt.Ref(), policy.PolicyHash),
		ActionRef:          action.Ref, ActionKind: action.Kind, Kind: effectKind,
		Subject: application.EffectSubject{ProjectRef: aggregate.Project(), GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
			ExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration,
			AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash, ActorRef: aggregate.Actor()},
		ProposedBy: authority.PrincipalRef, Permission: authority.Permission, Authority: authority.AuthorizationReceipt,
		Demand: demand, SecurityCriticality: item.SecurityCriticality(), ReasoningEffort: item.ReasoningEffort(),
		PolicyHash: policy.PolicyHash, PolicyRevision: policy.GoalEnvelopeTemplate.Revision,
		QuotaRetryDelay: policy.QuotaRetryDelay, ApprovalTTL: policy.EffectApprovalTTL,
		TargetDigest:   canonicalFingerprint(append([]string{"orquesta.effect.admission.v1"}, targetFields...)...),
		IdempotencyKey: idempotency, CreatedAt: action.AvailableAt,
	}
	intent.Digest = application.EffectIntentDigest(intent)
	approval := application.EffectApproval{
		Ref: "effect-approval:auto:" + intent.Ref, RequestRef: intent.RequestRef, RequestFingerprint: intent.Digest,
		IntentRef: intent.Ref, IntentDigest: intent.Digest, Subject: intent.Subject,
		ProposedBy: intent.ProposedBy, DecidedBy: intent.ProposedBy, Decision: application.EffectApproved,
		Source: authority.Source, SecurityCriticality: intent.SecurityCriticality,
		PolicyHash: intent.PolicyHash, PolicyRevision: intent.PolicyRevision, TargetDigest: intent.TargetDigest,
		Reason: "sqlite test causal authority", IdempotencyKey: intent.IdempotencyKey,
		AuthorizationReceipt: intent.Authority, DecidedAt: intent.CreatedAt,
	}
	action.EffectIntentRef, action.EffectIntent, action.EffectApproval = intent.Ref, intent, &approval
	return action
}

func amendLegacyGoal(
	t *testing.T,
	repository *Repository,
	state application.AmendGoalState,
) (application.GoalRecord, bool, error) {
	t.Helper()
	return repository.AmendGoal(context.Background(), authorizeLegacyAmendState(t, repository, state))
}
