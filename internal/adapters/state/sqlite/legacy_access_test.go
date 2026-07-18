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
	return repository.CreateGoal(context.Background(), authorizeLegacyCreateState(t, repository, state))
}

func amendLegacyGoal(
	t *testing.T,
	repository *Repository,
	state application.AmendGoalState,
) (application.GoalRecord, bool, error) {
	t.Helper()
	return repository.AmendGoal(context.Background(), authorizeLegacyAmendState(t, repository, state))
}
