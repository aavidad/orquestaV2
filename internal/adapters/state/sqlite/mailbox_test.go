package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestMailboxHandoffRequirementSurvivesRestartAndGatesAdmission(t *testing.T) {
	for _, test := range []struct {
		name            string
		handoffRequired bool
	}{
		{name: "non_contractual", handoffRequired: false},
		{name: "contractual", handoffRequired: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			clock := &sqliteMembershipClock{now: time.Date(2026, 7, 16, 9, 0, 0, 0, time.UTC)}
			path := filepath.Join(t.TempDir(), "mailbox-handoff-policy", "orquesta.sqlite")
			repository := openMailboxTestRepository(t, path, clock)
			defer func() { _ = repository.Close() }()

			fixture := newMailboxGoalFixtureWithHandoff(t, clock.Now(), test.handoffRequired)
			created, fresh, err := createLegacyGoal(t, repository, fixture.state)
			if err != nil || !fresh {
				t.Fatalf("create mailbox policy Goal fresh=%v err=%v", fresh, err)
			}
			source := readLegacyProjectOwner(t, repository, created.Goal.Project())
			recipient := testPrincipal(
				t, "principal:mailbox-policy-recipient", "actor:mailbox-policy-recipient",
				identity.PrincipalKindHuman,
			)
			grantTestMembership(
				t, repository, source, recipient, created.Goal.Project(), identity.RoleOperator,
				"membership:mailbox-policy-recipient", clock.Now(),
			)

			for index := 0; index < 2; index++ {
				clock.Advance(time.Second)
				claim := mustMailboxSchedulerClaim(t, repository, "policy-launch", index, clock.Now())
				startMailboxExecution(t, repository, claim, clock.Now())
			}
			clock.Advance(time.Second)
			childObserve := mustMailboxSchedulerClaim(t, repository, "policy-observe-child", 0, clock.Now())
			if childObserve.Action.WorkItemRef != fixture.childRef ||
				childObserve.Action.Kind != application.ActionObserveAgent {
				t.Fatalf("mailbox policy observe action=%+v, want child", childObserve.Action)
			}
			childArtifact := succeedMailboxChild(t, repository, childObserve, clock.Now())

			beforeRestart, err := repository.GetGoal(ctx, fixture.state.Goal.Ref())
			sqliteTestNoError(t, err)
			beforeParent, _ := beforeRestart.Goal.WorkItem(fixture.parentRef)
			beforeChild, _ := beforeRestart.Goal.WorkItem(fixture.childRef)
			if beforeParent.HandoffRequired() || beforeChild.HandoffRequired() != test.handoffRequired {
				t.Fatalf(
					"pre-restart handoff parent=%v child=%v want child=%v",
					beforeParent.HandoffRequired(), beforeChild.HandoffRequired(), test.handoffRequired,
				)
			}
			parentExecution, parentBound := beforeParent.Execution()
			childExecution, childBound := beforeChild.Execution()
			if !parentBound || !childBound {
				t.Fatalf("mailbox policy executions parent=%v child=%v", parentBound, childBound)
			}

			repository = restartMailboxTestRepository(t, repository, path, clock)
			restarted, err := repository.GetGoal(ctx, fixture.state.Goal.Ref())
			sqliteTestNoError(t, err)
			restartedParent, _ := restarted.Goal.WorkItem(fixture.parentRef)
			restartedChild, _ := restarted.Goal.WorkItem(fixture.childRef)
			if restartedParent.HandoffRequired() || restartedChild.HandoffRequired() != test.handoffRequired {
				t.Fatalf(
					"restart handoff parent=%v child=%v want child=%v",
					restartedParent.HandoffRequired(), restartedChild.HandoffRequired(), test.handoffRequired,
				)
			}

			authorization := authorizeTest(
				t, repository, source, restarted.Goal.Project(), identity.PermissionGoalsDirect,
				restarted.Goal.Ref().String(), "authorization:mailbox-policy:"+test.name, clock.Now(),
			)
			messageRef := mustRef(
				t, "message:mailbox-policy:"+test.name, application.NewMailboxMessageRef,
			)
			envelope := application.MailboxEnvelope{
				Ref: messageRef, RequestRef: "request:mailbox-policy:" + test.name,
				ProjectRef: restarted.Goal.Project(), GoalRef: restarted.Goal.Ref(),
				TargetPlanGeneration: restarted.Goal.PlanGeneration(), Kind: application.MailboxKindChildDelivery,
				ParentWorkItemRef: fixture.parentRef, ChildWorkItemRef: fixture.childRef,
				Source: application.MailboxEndpoint{
					PrincipalRef: source.Ref, WorkItemRef: fixture.childRef, ExecutionRef: childExecution,
				},
				Recipient: application.MailboxEndpoint{
					PrincipalRef: recipient.Ref, WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution,
				},
				Summary: "handoff policy result", ArtifactRefs: []goal.ArtifactRef{childArtifact},
				AdmittedAt: clock.Now(),
			}
			envelope.RequestFingerprint = application.MailboxAdmissionFingerprint(envelope)
			envelope.ContentHash = application.MailboxEnvelopeContentHash(envelope)
			admission := application.MailboxAdmissionReceipt{
				Ref: "receipt:mailbox-policy:" + test.name, MessageRef: messageRef,
				RequestRef: envelope.RequestRef, RequestFingerprint: envelope.RequestFingerprint,
				PrincipalRef: source.Ref, AuthorizationReceipt: authorization, AdmittedAt: clock.Now(),
			}
			state := application.AdmitMailboxState{
				RequestRef: envelope.RequestRef, RequestFingerprint: envelope.RequestFingerprint,
				AuthorizationReceipt: authorization, Envelope: envelope, Admission: admission,
				Action: application.ActionRecord{
					Ref: "action:mailbox:" + messageRef.String(), Kind: application.ActionDeliverMailbox,
					GoalRef: envelope.GoalRef, WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution,
					PlanGeneration:     envelope.TargetPlanGeneration,
					WorkItemGeneration: restartedParent.Revision(), AvailableAt: clock.Now(),
				},
				Event: application.EventRecord{
					Ref: "event:mailbox-policy:" + test.name, Kind: "mailbox.admitted",
					GoalRef: envelope.GoalRef, WorkItemRef: fixture.parentRef,
					ExecutionRef: parentExecution, OccurredAt: clock.Now(),
				},
				OperationAt: clock.Now(),
			}
			record, changed, err := repository.AdmitMailbox(ctx, state)
			if test.handoffRequired {
				if err != nil || !changed || record.State != application.MailboxStateAdmitted {
					t.Fatalf("contractual mailbox admission changed=%v record=%+v err=%v", changed, record, err)
				}
				return
			}
			if !application.IsStateError(err, application.StateConflict) || changed {
				t.Fatalf("non-contractual mailbox admission changed=%v record=%+v err=%v", changed, record, err)
			}
			var envelopes, actions int
			if err := repository.db.QueryRow(
				`SELECT COUNT(*) FROM mailbox_envelopes WHERE goal_ref = ?`, envelope.GoalRef.String(),
			).Scan(&envelopes); err != nil {
				t.Fatal(err)
			}
			if err := repository.db.QueryRow(
				`SELECT COUNT(*) FROM outbox WHERE mailbox_message_ref = ?`, messageRef.String(),
			).Scan(&actions); err != nil {
				t.Fatal(err)
			}
			if envelopes != 0 || actions != 0 {
				t.Fatalf("non-contractual admission leaked envelope=%d action=%d", envelopes, actions)
			}
		})
	}
}

func TestCodexChildDeliveryUsesSameExecutionServicePrincipalAfterArtifactPersistence(t *testing.T) {
	ctx := context.Background()
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 23, 14, 0, 0, 0, time.UTC)}
	path := filepath.Join(t.TempDir(), "post-artifact-mailbox", "orquesta.sqlite")
	repository := openMailboxTestRepository(t, path, clock)
	defer func() { _ = repository.Close() }()

	fixture := newMailboxGoalFixture(t, clock.Now())
	created, fresh, err := createLegacyGoal(t, repository, fixture.state)
	if err != nil || !fresh {
		t.Fatalf("create Goal fresh=%v err=%v", fresh, err)
	}
	for index := 0; index < 2; index++ {
		clock.Advance(time.Second)
		claim := mustMailboxSchedulerClaim(t, repository, "post-artifact-launch", index, clock.Now())
		startMailboxExecution(t, repository, claim, clock.Now())
	}
	clock.Advance(time.Second)
	childObserve := mustMailboxSchedulerClaim(t, repository, "post-artifact-observe", 0, clock.Now())
	if childObserve.Action.WorkItemRef != fixture.childRef {
		t.Fatalf("claimed work item=%s want child=%s", childObserve.Action.WorkItemRef, fixture.childRef)
	}
	sourceBefore, err := repository.ExecutionSessionAuthority(
		ctx, childObserve.Action.ExecutionRef, "execution_token",
	)
	sqliteTestNoError(t, err)
	childArtifact := succeedMailboxChildWithPostArtifact(
		t, repository, childObserve, clock.Now(), true,
	)

	repository = restartMailboxTestRepository(t, repository, path, clock)
	persisted, err := repository.GetGoal(ctx, created.Goal.Ref())
	sqliteTestNoError(t, err)
	sourceAfter, err := repository.ExecutionSessionAuthority(
		ctx, childObserve.Action.ExecutionRef, "execution_token",
	)
	if err != nil || !application.SameExecutionSessionAuthority(sourceBefore, sourceAfter) {
		t.Fatalf("post-artifact authority=%+v before=%+v err=%v", sourceAfter, sourceBefore, err)
	}
	parent, _ := persisted.Goal.WorkItem(fixture.parentRef)
	parentExecutionRef, _ := parent.Execution()
	parentExecution := findMailboxExecution(t, persisted.Executions, parentExecutionRef)
	recipient, err := application.DeriveExecutionSessionAuthority(
		application.ExecutionSessionRequest(persisted.Goal, parentExecution), "execution_token",
	)
	sqliteTestNoError(t, err)

	authorizationRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:post-artifact-mailbox",
		Principal:  sourceAfter.ServicePrincipal, ProjectRef: persisted.Goal.Project(),
		Permission: identity.PermissionGoalsDirect, ResourceRef: persisted.Goal.Ref().String(),
		RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	authorization, err := repository.Authorize(ctx, authorizationRequest)
	if err != nil || authorization.Decision().Role() != identity.RoleExecutionService {
		t.Fatalf("execution authorization=%+v err=%v", authorization, err)
	}
	if _, err := repository.Membership(
		ctx, sourceAfter.ServicePrincipal.Ref, persisted.Goal.Project(),
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("execution principal acquired membership err=%v", err)
	}
	artifactAuthorizationRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:post-artifact-undelivered",
		Principal:  sourceAfter.ServicePrincipal, ProjectRef: persisted.Goal.Project(),
		Permission: identity.PermissionArtifactsRead, ResourceRef: childArtifact.String(),
		RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	if _, err := repository.Authorize(ctx, artifactAuthorizationRequest); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("source execution read undelivered artifact err=%v", err)
	}
	listRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:post-artifact-list",
		Principal:  sourceAfter.ServicePrincipal, ProjectRef: persisted.Goal.Project(),
		Permission: identity.PermissionGoalsList, ResourceRef: persisted.Goal.Ref().String(),
		RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	if _, err := repository.Authorize(ctx, listRequest); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("execution principal listed goals err=%v", err)
	}
	siblingArtifact, _ := goal.NewArtifactRef("artifact:sha256:" + strings.Repeat("f", 64))
	_, err = repository.db.ExecContext(ctx, `
INSERT INTO artifacts(ref,goal_ref,work_item_ref,digest,media_type,size,created_at)
SELECT ?,goal_ref,work_item_ref,?,'text/plain',7,created_at
FROM artifacts WHERE goal_ref=? AND ref=?`,
		siblingArtifact.String(), strings.Repeat("f", 64),
		persisted.Goal.Ref().String(), childArtifact.String(),
	)
	sqliteTestNoError(t, err)
	_, err = repository.db.ExecContext(ctx, `
INSERT INTO artifact_occurrences(
 occurrence_ref,kind,goal_ref,work_item_ref,execution_ref,artifact_ref,
 execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,created_at)
SELECT ?,kind,goal_ref,work_item_ref,execution_ref,?,
 execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,created_at
FROM artifact_occurrences WHERE goal_ref=? AND artifact_ref=?`,
		"artifact-occurrence:post-artifact-sibling", siblingArtifact.String(),
		persisted.Goal.Ref().String(), childArtifact.String(),
	)
	sqliteTestNoError(t, err)
	_, err = repository.db.ExecContext(ctx, `
INSERT INTO attestations(
 ref,kind,verdict,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,
 work_item_generation,app_spec_generation,spec_hash,artifact_ref,subject_digest,
 workspace_binding_digest,change_set_ref,change_set_digest,manifest_artifact_ref,report_artifact_ref,
 attestor_ref,receipt_ref,policy_ref,required_tests_digest,policy_digest,effect_intent_ref,
 effect_attempt_ref,effect_fence,effect_receipt_ref,started_at,finished_at,policy,accepted_at)
SELECT ?,kind,verdict,goal_ref,work_item_ref,execution_ref,execution_attempt,plan_generation,
 work_item_generation,app_spec_generation,spec_hash,?,subject_digest,
 workspace_binding_digest,change_set_ref,change_set_digest,manifest_artifact_ref,report_artifact_ref,
 attestor_ref,receipt_ref,policy_ref,required_tests_digest,policy_digest,effect_intent_ref,
 effect_attempt_ref,effect_fence,effect_receipt_ref,started_at,finished_at,policy,accepted_at
FROM attestations WHERE goal_ref=? AND artifact_ref=?`,
		"attestation:post-artifact-sibling", siblingArtifact.String(),
		persisted.Goal.Ref().String(), childArtifact.String(),
	)
	sqliteTestNoError(t, err)

	messageRef := mustRef(t, "message:post-artifact-mailbox", application.NewMailboxMessageRef)
	envelope := application.MailboxEnvelope{
		Ref: messageRef, RequestRef: "request:post-artifact-mailbox",
		ProjectRef: persisted.Goal.Project(), GoalRef: persisted.Goal.Ref(),
		TargetPlanGeneration: persisted.Goal.PlanGeneration(), Kind: application.MailboxKindChildDelivery,
		ParentWorkItemRef: fixture.parentRef, ChildWorkItemRef: fixture.childRef,
		Source: application.MailboxEndpoint{
			PrincipalRef: sourceAfter.ServicePrincipal.Ref, WorkItemRef: fixture.childRef,
			ExecutionRef: childObserve.Action.ExecutionRef,
		},
		Recipient: application.MailboxEndpoint{
			PrincipalRef: recipient.ServicePrincipal.Ref, WorkItemRef: fixture.parentRef,
			ExecutionRef: parentExecutionRef,
		},
		Summary: "mailbox child", ArtifactRefs: []goal.ArtifactRef{childArtifact},
		AdmittedAt: clock.Now(),
	}
	envelope.RequestFingerprint = application.MailboxAdmissionFingerprint(envelope)
	envelope.ContentHash = application.MailboxEnvelopeContentHash(envelope)
	admission := application.MailboxAdmissionReceipt{
		Ref: "receipt:post-artifact-mailbox", MessageRef: messageRef,
		RequestRef: envelope.RequestRef, RequestFingerprint: envelope.RequestFingerprint,
		PrincipalRef: sourceAfter.ServicePrincipal.Ref, AuthorizationReceipt: authorization,
		AdmittedAt: clock.Now(),
	}
	admitState := application.AdmitMailboxState{
		RequestRef: envelope.RequestRef, RequestFingerprint: envelope.RequestFingerprint,
		AuthorizationReceipt: authorization, Envelope: envelope, Admission: admission,
		Action: application.ActionRecord{
			Ref: "action:mailbox:" + messageRef.String(), Kind: application.ActionDeliverMailbox,
			GoalRef: envelope.GoalRef, WorkItemRef: fixture.parentRef, ExecutionRef: parentExecutionRef,
			PlanGeneration: envelope.TargetPlanGeneration, WorkItemGeneration: parent.Revision(),
			AvailableAt: clock.Now(),
		},
		Event: application.EventRecord{
			Ref: "event:post-artifact-mailbox", Kind: "mailbox.admitted",
			GoalRef: envelope.GoalRef, WorkItemRef: fixture.parentRef,
			ExecutionRef: parentExecutionRef, OccurredAt: clock.Now(),
		},
		OperationAt: clock.Now(),
	}
	record, changed, err := repository.AdmitMailbox(ctx, admitState)
	if err != nil || !changed || record.Admission.PrincipalRef != sourceBefore.ServicePrincipal.Ref {
		t.Fatalf("mailbox changed=%v record=%+v err=%s", changed, record, sqliteTestErrorChain(err))
	}
	replayed, changed, err := repository.AdmitMailbox(ctx, admitState)
	if err != nil || changed || replayed.Envelope.Ref != record.Envelope.Ref ||
		replayed.Admission.Ref != record.Admission.Ref {
		t.Fatalf("mailbox replay changed=%v record=%+v err=%s", changed, replayed, sqliteTestErrorChain(err))
	}
	deliveredArtifactRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:post-artifact-delivered",
		Principal:  recipient.ServicePrincipal, ProjectRef: persisted.Goal.Project(),
		Permission: identity.PermissionArtifactsRead, ResourceRef: childArtifact.String(),
		RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	if receipt, err := repository.Authorize(ctx, deliveredArtifactRequest); err != nil ||
		receipt.Decision().Role() != identity.RoleExecutionService {
		t.Fatalf("delivered artifact authorization=%+v err=%v", receipt, err)
	}
	siblingArtifactRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:post-artifact-sibling",
		Principal:  recipient.ServicePrincipal, ProjectRef: persisted.Goal.Project(),
		Permission: identity.PermissionArtifactsRead, ResourceRef: siblingArtifact.String(),
		RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	if _, err := repository.Authorize(ctx, siblingArtifactRequest); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("recipient execution read sibling artifact err=%v", err)
	}
	sourceMailboxRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization:post-artifact-source-envelope",
		Principal:  sourceAfter.ServicePrincipal, ProjectRef: persisted.Goal.Project(),
		Permission: identity.PermissionGoalsGet, ResourceRef: messageRef.String(),
		RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	sourceMailboxAuthorization, err := repository.Authorize(ctx, sourceMailboxRequest)
	if err != nil || sourceMailboxAuthorization.Decision().Role() != identity.RoleExecutionService {
		t.Fatalf("source envelope authorization=%+v err=%v", sourceMailboxAuthorization, err)
	}

	// Keep the parent's observation leased so the internal admission action is
	// the next claimable action without mutating scheduler priority.
	parentObserve := mustMailboxSchedulerClaim(t, repository, "post-artifact-parent-observe", 0, clock.Now())
	admitClaim := mustMailboxSchedulerClaim(t, repository, "post-artifact-admit", 0, clock.Now())
	if admitClaim.Action.Kind != application.ActionAdmitMailbox ||
		admitClaim.Action.ExecutionRef != childObserve.Action.ExecutionRef {
		t.Fatalf("post-artifact claim=%+v", admitClaim)
	}
	clock.Advance(time.Second)
	sqliteTestNoError(t, repository.RecordPostArtifactMailboxAdmitted(
		ctx, application.PostArtifactMailboxAdmittedState{
			Claim: admitClaim, MessageRef: messageRef, AdmissionRef: admission.Ref,
			OperationAt: clock.Now(),
		},
	))
	repository = restartMailboxTestRepository(t, repository, path, clock)
	if _, err := repository.ExecutionSessionAuthority(
		ctx, childObserve.Action.ExecutionRef, "execution_token",
	); !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("completed delivery retained execution authority err=%v", err)
	}
	if _, err := repository.Authorize(ctx, authorizationRequest); !errors.Is(err, application.ErrForbidden) {
		t.Fatalf("authorization replay after logical revocation err=%v", err)
	}
	sourceClaimFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationClaim, sourceAfter.ServicePrincipal.Ref,
		envelope.ProjectRef, envelope.GoalRef, messageRef, envelope.ParentWorkItemRef,
		childObserve.Action.ExecutionRef, "", 0, "",
	)
	if _, changed, err := repository.ClaimMailbox(ctx, application.ClaimMailboxState{
		RequestRef:            "request:post-artifact-source-cannot-claim",
		RequestFingerprint:    sourceClaimFingerprint,
		AuthorizationReceipt:  sourceMailboxAuthorization,
		PrincipalRef:          sourceAfter.ServicePrincipal.Ref,
		ProjectRef:            envelope.ProjectRef,
		GoalRef:               envelope.GoalRef,
		MessageRef:            messageRef,
		RecipientExecutionRef: childObserve.Action.ExecutionRef,
		Token:                 "token:post-artifact-source-cannot-claim",
		LeaseDuration:         time.Minute,
		RequestedAt:           clock.Now(),
	}); err == nil || changed {
		t.Fatalf("revoked source claimed recipient mailbox changed=%v err=%v", changed, err)
	}

	t.Run("completed admission survives succeeded Goal recovery", func(t *testing.T) {
		completePostArtifactMailbox(
			t, repository, envelope, record.Action, recipient.ServicePrincipal,
			parentExecutionRef, clock,
		)
		clock.Advance(time.Second)
		succeedMailboxWorkItem(
			t, repository, parentObserve, clock.Now(), "post-artifact-parent", "b", true, false,
		)
		repository = restartMailboxTestRepository(t, repository, path, clock)
		terminal, err := repository.GetGoal(ctx, envelope.GoalRef)
		sqliteTestNoError(t, err)
		if terminal.Goal.State() != goal.GoalStateSucceeded {
			t.Fatalf("recovered Goal state=%s want succeeded", terminal.Goal.State())
		}
		var completedAt sql.NullInt64
		if err := repository.db.QueryRowContext(ctx, `
SELECT completed_at
FROM outbox
WHERE ref=? AND kind='admit_mailbox'`,
			admitClaim.Action.Ref,
		).Scan(&completedAt); err != nil || !completedAt.Valid {
			t.Fatalf("recovered post-artifact admission completed_at=%+v err=%v", completedAt, err)
		}
	})
}

func TestMailboxRestartPreservesEveryCausalFrontier(t *testing.T) {
	ctx := context.Background()
	clock := &sqliteMembershipClock{now: time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)}
	path := filepath.Join(t.TempDir(), "mailbox", "orquesta.sqlite")
	repository := openMailboxTestRepository(t, path, clock)
	defer func() { _ = repository.Close() }()

	fixture := newMailboxGoalFixture(t, clock.Now())
	created, fresh, err := createLegacyGoal(t, repository, fixture.state)
	if err != nil || !fresh {
		t.Fatalf("create mailbox Goal fresh=%v err=%v", fresh, err)
	}
	source := readLegacyProjectOwner(t, repository, created.Goal.Project())
	recipient := testPrincipal(t, "principal:mailbox-recipient", "actor:mailbox-recipient", identity.PrincipalKindHuman)
	grantTestMembership(
		t, repository, source, recipient, created.Goal.Project(), identity.RoleOperator,
		"membership:mailbox-recipient", clock.Now(),
	)

	for index := 0; index < 2; index++ {
		clock.Advance(time.Second)
		claim := mustMailboxSchedulerClaim(t, repository, "launch", index, clock.Now())
		startMailboxExecution(t, repository, claim, clock.Now())
	}
	clock.Advance(time.Second)
	childObserve := mustMailboxSchedulerClaim(t, repository, "observe-child", 0, clock.Now())
	if childObserve.Action.WorkItemRef != fixture.childRef || childObserve.Action.Kind != application.ActionObserveAgent {
		t.Fatalf("first observe action=%+v, want child", childObserve.Action)
	}
	childArtifact := succeedMailboxChild(t, repository, childObserve, clock.Now())

	running, err := repository.GetGoal(ctx, fixture.state.Goal.Ref())
	sqliteTestNoError(t, err)
	parent, _ := running.Goal.WorkItem(fixture.parentRef)
	child, _ := running.Goal.WorkItem(fixture.childRef)
	parentExecution, parentBound := parent.Execution()
	childExecution, childBound := child.Execution()
	if !parentBound || !childBound || parent.State() != goal.WorkItemStateRunning ||
		child.State() != goal.WorkItemStateSucceeded || parent.HandoffRequired() || !child.HandoffRequired() {
		t.Fatalf("mailbox source lifecycle parent=%s child=%s", parent.State(), child.State())
	}

	admitAuthorization := authorizeTest(
		t, repository, source, running.Goal.Project(), identity.PermissionGoalsDirect,
		running.Goal.Ref().String(), "authorization:mailbox-admit", clock.Now(),
	)
	messageRef := mustRef(t, "message:mailbox-restart", application.NewMailboxMessageRef)
	envelope := application.MailboxEnvelope{
		Ref: messageRef, RequestRef: "request:mailbox-admit",
		ProjectRef: running.Goal.Project(), GoalRef: running.Goal.Ref(),
		TargetPlanGeneration: running.Goal.PlanGeneration(), Kind: application.MailboxKindChildDelivery,
		ParentWorkItemRef: fixture.parentRef, ChildWorkItemRef: fixture.childRef,
		Source: application.MailboxEndpoint{
			PrincipalRef: source.Ref, WorkItemRef: fixture.childRef, ExecutionRef: childExecution,
		},
		Recipient: application.MailboxEndpoint{
			PrincipalRef: recipient.Ref, WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution,
		},
		Summary: "compact child result", ArtifactRefs: []goal.ArtifactRef{childArtifact}, AdmittedAt: clock.Now(),
	}
	envelope.RequestFingerprint = application.MailboxAdmissionFingerprint(envelope)
	envelope.ContentHash = application.MailboxEnvelopeContentHash(envelope)
	admission := application.MailboxAdmissionReceipt{
		Ref: "receipt:mailbox-admission", MessageRef: messageRef,
		RequestRef: envelope.RequestRef, RequestFingerprint: envelope.RequestFingerprint,
		PrincipalRef: source.Ref, AuthorizationReceipt: admitAuthorization, AdmittedAt: clock.Now(),
	}
	action := application.ActionRecord{
		Ref: "action:mailbox:" + messageRef.String(), Kind: application.ActionDeliverMailbox,
		GoalRef: running.Goal.Ref(), WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution,
		PlanGeneration: running.Goal.PlanGeneration(), WorkItemGeneration: parent.Revision(),
		AvailableAt: clock.Now(),
	}
	admitState := application.AdmitMailboxState{
		RequestRef: envelope.RequestRef, RequestFingerprint: envelope.RequestFingerprint,
		AuthorizationReceipt: admitAuthorization, Envelope: envelope, Admission: admission, Action: action,
		Event: application.EventRecord{
			Ref: "event:mailbox-admitted:restart", Kind: "mailbox.admitted", GoalRef: running.Goal.Ref(),
			WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution, OccurredAt: clock.Now(),
		},
		OperationAt: clock.Now(),
	}
	for _, stale := range []struct {
		ref     goal.WorkItemRef
		state   goal.WorkItemState
		restore goal.WorkItemState
	}{
		{ref: fixture.childRef, state: goal.WorkItemStateFailed, restore: goal.WorkItemStateSucceeded},
		{ref: fixture.parentRef, state: goal.WorkItemStateFailed, restore: goal.WorkItemStateRunning},
	} {
		if _, err := repository.db.Exec(`UPDATE work_items SET state = ? WHERE goal_ref = ? AND ref = ?`,
			string(stale.state), envelope.GoalRef.String(), stale.ref.String(),
		); err != nil {
			t.Fatal(err)
		}
		_, changed, err := repository.AdmitMailbox(ctx, admitState)
		if !application.IsStateError(err, application.StateConflict) || changed {
			t.Fatalf("admit from terminal non-deliverable state=%s changed=%v err=%v", stale.state, changed, err)
		}
		if _, err := repository.db.Exec(`UPDATE work_items SET state = ? WHERE goal_ref = ? AND ref = ?`,
			string(stale.restore), envelope.GoalRef.String(), stale.ref.String(),
		); err != nil {
			t.Fatal(err)
		}
	}
	var rejectedAdmissions int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM mailbox_envelopes WHERE request_ref = ?`,
		envelope.RequestRef,
	).Scan(&rejectedAdmissions); err != nil || rejectedAdmissions != 0 {
		t.Fatalf("rejected admission persisted rows=%d err=%v", rejectedAdmissions, err)
	}
	admitted, changed, err := repository.AdmitMailbox(ctx, admitState)
	if err != nil || !changed || admitted.State != application.MailboxStateAdmitted {
		t.Fatalf("admit changed=%v state=%s err=%v cause=%v", changed, admitted.State, err, errors.Unwrap(err))
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	assertMailboxState(t, repository, envelope, application.MailboxStateAdmitted, 0)
	assertMailboxListFIFO(t, repository, clock, envelope)
	restartedGoal, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	restartedChild, _ := restartedGoal.Goal.WorkItem(fixture.childRef)
	if !restartedChild.HandoffRequired() {
		t.Fatal("restart lost required child handoff edge")
	}

	clock.Advance(time.Millisecond)
	duplicateAuthorization := authorizeTest(
		t, repository, source, running.Goal.Project(), identity.PermissionGoalsDirect,
		running.Goal.Ref().String(), "authorization:mailbox-admit-duplicate", clock.Now(),
	)
	duplicateEnvelope := envelope
	duplicateEnvelope.Ref = mustRef(t, "message:mailbox-duplicate", application.NewMailboxMessageRef)
	duplicateEnvelope.RequestRef = "request:mailbox-admit-duplicate"
	duplicateEnvelope.RequestFingerprint = "fingerprint:mailbox-admit-duplicate"
	duplicateEnvelope.AdmittedAt = clock.Now()
	duplicateEnvelope.ContentHash = application.MailboxEnvelopeContentHash(duplicateEnvelope)
	duplicateAdmission := admission
	duplicateAdmission.Ref = "receipt:mailbox-admission-duplicate"
	duplicateAdmission.MessageRef = duplicateEnvelope.Ref
	duplicateAdmission.RequestRef = duplicateEnvelope.RequestRef
	duplicateAdmission.RequestFingerprint = duplicateEnvelope.RequestFingerprint
	duplicateAdmission.AuthorizationReceipt = duplicateAuthorization
	duplicateAdmission.AdmittedAt = clock.Now()
	duplicateAction := action
	duplicateAction.Ref = "action:mailbox:" + duplicateEnvelope.Ref.String()
	duplicateAction.AvailableAt = clock.Now()
	_, changed, err = repository.AdmitMailbox(ctx, application.AdmitMailboxState{
		RequestRef: duplicateEnvelope.RequestRef, RequestFingerprint: duplicateEnvelope.RequestFingerprint,
		AuthorizationReceipt: duplicateAuthorization, Envelope: duplicateEnvelope,
		Admission: duplicateAdmission, Action: duplicateAction,
		Event: application.EventRecord{
			Ref: "event:mailbox-admitted:duplicate", Kind: "mailbox.admitted", GoalRef: running.Goal.Ref(),
			WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution, OccurredAt: clock.Now(),
		},
		OperationAt: clock.Now(),
	})
	if !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("duplicate child delivery changed=%v err=%v cause=%v", changed, err, errors.Unwrap(err))
	}
	var envelopeCount, outboxCount int
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM mailbox_envelopes
WHERE goal_ref = ? AND parent_work_item_ref = ? AND child_work_item_ref = ?
  AND kind = 'child_delivery'`, envelope.GoalRef.String(), envelope.ParentWorkItemRef.String(),
		envelope.ChildWorkItemRef.String()).Scan(&envelopeCount); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox WHERE kind = 'deliver_mailbox' AND goal_ref = ?`,
		envelope.GoalRef.String()).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if envelopeCount != 1 || outboxCount != 1 {
		t.Fatalf("duplicate child delivery leaked envelope=%d outbox=%d", envelopeCount, outboxCount)
	}

	directorAccess, err := application.NewAccess(source, envelope.ProjectRef)
	sqliteTestNoError(t, err)
	director := newSQLiteDirectorOrchestrator(t, repository, clock, &sqliteDirectorIDs{})
	directorLease, err := director.ClaimDirector(ctx, directorAccess, application.ClaimDirectorRequest{
		RequestRef: "director-claim:mailbox-plan-extension", GoalRef: envelope.GoalRef,
	})
	if err != nil || !directorLease.Changed {
		t.Fatalf("claim mailbox Director lease=%+v err=%v", directorLease, err)
	}
	beforeExtension, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	planDecision, err := director.ProposeDirectorPlan(ctx, directorAccess, application.ProposeDirectorPlanRequest{
		RequestRef: "director-plan:mailbox-monotonic-extension", GoalRef: envelope.GoalRef,
		ExpectedGoalRevision:   beforeExtension.Goal.Revision(),
		ExpectedPlanGeneration: beforeExtension.Goal.PlanGeneration(),
		LeaseToken:             directorLease.Lease.Token, LeaseFence: directorLease.Lease.Fence,
		Reason: "prove admitted child delivery survives an append-only plan generation",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "work:mailbox-extension", Objective: "independent later plan work",
			Phase: "phase:mailbox", Role: "role:mailbox-extension",
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err != nil || !planDecision.Created {
		t.Fatalf("append mailbox plan decision=%+v err=%v cause=%v", planDecision, err, errors.Unwrap(err))
	}
	afterExtension, err := repository.GetGoal(ctx, envelope.GoalRef)
	if err != nil || afterExtension.Goal.PlanGeneration() != envelope.TargetPlanGeneration+1 {
		t.Fatalf("mailbox plan extension generation=%d err=%v", afterExtension.Goal.PlanGeneration(), err)
	}
	assertMailboxFrontierRetires(t, repository, clock, envelope, application.MailboxStateAdmitted, "admitted")
	staleClaimAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "claim-stale-parent", clock.Now())
	if _, err := repository.db.Exec(`UPDATE work_items SET state = 'failed' WHERE goal_ref = ? AND ref = ?`,
		envelope.GoalRef.String(), envelope.ParentWorkItemRef.String(),
	); err != nil {
		t.Fatal(err)
	}
	_, changed, err = repository.ClaimMailbox(ctx, application.ClaimMailboxState{
		RequestRef:           "request:mailbox-claim-stale-parent",
		RequestFingerprint:   "fingerprint:mailbox-claim-stale-parent",
		AuthorizationReceipt: staleClaimAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
		RecipientExecutionRef: parentExecution, Token: "token:mailbox-claim-stale-parent",
		LeaseDuration: time.Minute, RequestedAt: clock.Now(),
	})
	if !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("failed recipient claimed mailbox changed=%v err=%v", changed, err)
	}
	if _, err := repository.db.Exec(`UPDATE work_items SET state = 'running' WHERE goal_ref = ? AND ref = ?`,
		envelope.GoalRef.String(), envelope.ParentWorkItemRef.String(),
	); err != nil {
		t.Fatal(err)
	}

	claimAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "claim-one", clock.Now())
	claimFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationClaim, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		messageRef, envelope.ParentWorkItemRef, parentExecution, "", 0, "",
	)
	firstClaim, changed, err := repository.ClaimMailbox(ctx, application.ClaimMailboxState{
		RequestRef: "request:mailbox-claim-one", RequestFingerprint: claimFingerprint,
		AuthorizationReceipt: claimAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
		RecipientExecutionRef: parentExecution, Token: "token:mailbox-claim-one",
		LeaseDuration: time.Minute, RequestedAt: clock.Now(),
	})
	if err != nil || !changed || firstClaim.Attempt.Fence != 1 {
		t.Fatalf("first claim changed=%v claim=%+v err=%v cause=%v", changed, firstClaim, err, errors.Unwrap(err))
	}
	assertMailboxClaimReplayHistorical(
		t, repository, envelope, recipient, "request:mailbox-claim-one", claimFingerprint, firstClaim.Attempt,
	)
	repository = restartMailboxTestRepository(t, repository, path, clock)
	assertMailboxState(t, repository, envelope, application.MailboxStateClaimed, 1)

	clock.Advance(2 * time.Minute)
	expiredClaimReplay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: "request:mailbox-claim-one",
		RequestFingerprint: claimFingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
	})
	if err != nil || !found || expiredClaimReplay.Claim.Record.State != application.MailboxStateClaimed ||
		expiredClaimReplay.Claim.Attempt != firstClaim.Attempt {
		t.Fatalf("expired exact claim replay changed lease/result replay=%+v found=%v err=%v",
			expiredClaimReplay, found, err)
	}
	reclaimAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "claim-two", clock.Now())
	claim, changed, err := repository.ClaimMailbox(ctx, application.ClaimMailboxState{
		RequestRef: "request:mailbox-claim-two", RequestFingerprint: claimFingerprint,
		AuthorizationReceipt: reclaimAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
		RecipientExecutionRef: parentExecution, Token: "token:mailbox-claim-two",
		LeaseDuration: time.Minute, RequestedAt: clock.Now(),
	})
	if err != nil || !changed || claim.Attempt.Fence != 2 {
		t.Fatalf("reclaim changed=%v claim=%+v err=%v cause=%v", changed, claim, err, errors.Unwrap(err))
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	assertMailboxState(t, repository, envelope, application.MailboxStateClaimed, 2)
	assertMailboxFrontierRetires(t, repository, clock, envelope, application.MailboxStateClaimed, "claimed")
	if replay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: "request:mailbox-claim-one",
		RequestFingerprint: claimFingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
	}); !application.IsStateError(err, application.StateConflict) || found {
		t.Fatalf("superseded claim replay=%+v found=%v err=%v", replay, found, err)
	}

	clock.Advance(time.Second)
	deliveryAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "deliver", clock.Now())
	deliveryFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationDeliver, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		messageRef, envelope.ParentWorkItemRef, parentExecution, claim.Attempt.ClaimToken,
		claim.Attempt.Fence, "",
	)
	delivered, changed, err := repository.MarkMailboxDelivered(ctx, application.MarkMailboxDeliveredState{
		RequestRef: "request:mailbox-deliver", RequestFingerprint: deliveryFingerprint,
		AuthorizationReceipt: deliveryAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
		RecipientExecutionRef: parentExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence:       claim.Attempt.Fence,
		DeliveryRef: "receipt:mailbox-delivered", OperationAt: clock.Now(),
	})
	if err != nil || !changed || delivered.State != application.MailboxStateDelivered {
		t.Fatalf("deliver changed=%v state=%s err=%v cause=%v", changed, delivered.State, err, errors.Unwrap(err))
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	assertMailboxState(t, repository, envelope, application.MailboxStateDelivered, 2)
	assertMailboxClaimReplayHistorical(
		t, repository, envelope, recipient, "request:mailbox-claim-two", claimFingerprint, claim.Attempt,
	)
	assertMailboxFrontierRetires(t, repository, clock, envelope, application.MailboxStateDelivered, "delivered")

	clock.Advance(time.Second)
	consumeAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "consume", clock.Now())
	consumeFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationConsume, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		messageRef, envelope.ParentWorkItemRef, parentExecution, claim.Attempt.ClaimToken,
		claim.Attempt.Fence, "",
	)
	receipt := application.ActionConsumptionReceipt{
		ActionRef: action.Ref, Kind: application.ActionDeliverMailbox,
		GoalRef: envelope.GoalRef, WorkItemRef: fixture.parentRef, ExecutionRef: parentExecution,
		MailboxMessageRef: messageRef, PlanGeneration: action.PlanGeneration,
		WorkItemGeneration: action.WorkItemGeneration, Fence: claim.Attempt.Fence,
		DeliveryAttempt: claim.Attempt.Fence, ClaimToken: claim.Attempt.ClaimToken,
		WorkerRef: recipient.Ref.String(), Outcome: application.ActionConsumedCompleted,
		ConsumedAt: clock.Now(),
	}
	consumed, changed, err := repository.ConsumeMailbox(ctx, application.ConsumeMailboxState{
		RequestRef: "request:mailbox-consume", RequestFingerprint: consumeFingerprint,
		AuthorizationReceipt: consumeAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
		RecipientExecutionRef: parentExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence:          claim.Attempt.Fence,
		ConsumptionRef: "receipt:mailbox-consumed", ConsumptionReceipt: receipt, OperationAt: clock.Now(),
	})
	if err != nil || !changed || consumed.State != application.MailboxStateConsumed {
		t.Fatalf("consume changed=%v state=%s err=%v cause=%v", changed, consumed.State, err, errors.Unwrap(err))
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	assertMailboxState(t, repository, envelope, application.MailboxStateConsumed, 2)
	latestClaimReplay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: "request:mailbox-claim-two",
		RequestFingerprint: claimFingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
	})
	if err != nil || !found || latestClaimReplay.Claim.Record.State != application.MailboxStateClaimed ||
		latestClaimReplay.Claim.Attempt != claim.Attempt ||
		!latestClaimReplay.Claim.Attempt.LeaseUntil.Equal(claim.Attempt.LeaseUntil) {
		t.Fatalf("exact expired/consumed claim replay renewed or projected later state: replay=%+v found=%v err=%v",
			latestClaimReplay, found, err)
	}
	recipientCapabilities := sqliteTestCapabilities()
	recipientCapabilities.Unrestricted = false
	recipientCapabilities.RoleKeys = []string{"role:mailbox"}
	genericClaim, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:generic-must-not-see-mailbox", Token: "token:generic-must-not-see-mailbox",
		LeaseDuration: time.Minute, Capabilities: recipientCapabilities, BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || (found && genericClaim.Action.Kind == application.ActionDeliverMailbox) {
		t.Fatalf("generic scheduler claimed mailbox claim=%+v found=%v err=%v", genericClaim, found, err)
	}
	assertConsumedMailboxRetirement(
		t, repository, clock, envelope, recipient, claim, genericClaim, found,
		deliveryFingerprint, consumeFingerprint,
	)

	clock.Advance(time.Second)
	ackAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "ack", clock.Now())
	beforeAck, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	ackRef := "receipt:mailbox-acknowledged"
	updated, err := beforeAck.Goal.ResolveChildHandoff(
		beforeAck.Goal.Revision(), fixture.parentRef, fixture.childRef,
		messageRef.String(), goal.ChildHandoffAcknowledged, ackRef, clock.Now(),
	)
	sqliteTestNoError(t, err)
	ack := application.MailboxAcknowledgement{
		Ref: ackRef, MessageRef: messageRef, ActionRef: action.Ref,
		RequestRef: "request:mailbox-ack",
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef,
		TargetPlanGeneration: envelope.TargetPlanGeneration,
		ParentWorkItemRef:    fixture.parentRef, ChildWorkItemRef: fixture.childRef,
		Recipient: envelope.Recipient,
		Fence:     claim.Attempt.Fence, Outcome: application.MailboxOutcomeAcknowledged,
		EffectOrReworkRef: "effect:mailbox-child-applied", AcknowledgedAt: clock.Now(),
		AuthorizationReceipt: ackAuthorization,
	}
	ack.RequestFingerprint = application.MailboxMutationFingerprint(
		application.MailboxMutationAcknowledge, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		messageRef, envelope.ParentWorkItemRef, parentExecution, claim.Attempt.ClaimToken,
		claim.Attempt.Fence,
		strconv.FormatUint(uint64(beforeAck.Goal.Revision()), 10)+"\x00"+
			strconv.FormatUint(uint64(beforeAck.Goal.PlanGeneration()), 10)+"\x00"+
			ack.EffectOrReworkRef,
	)
	persistedAck, changed, err := repository.AcknowledgeMailbox(ctx, application.ResolveMailboxState{
		RequestRef: ack.RequestRef, RequestFingerprint: ack.RequestFingerprint,
		AuthorizationReceipt: ackAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
		RecipientExecutionRef: parentExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence:                  claim.Attempt.Fence,
		ExpectedGoalRevision:   beforeAck.Goal.Revision(),
		ExpectedPlanGeneration: beforeAck.Goal.PlanGeneration(), Goal: updated,
		Acknowledgement: ack, Events: []application.EventRecord{{
			Ref: "event:mailbox-acknowledged:restart", Kind: "mailbox.acknowledged",
			GoalRef: envelope.GoalRef, WorkItemRef: fixture.parentRef,
			ExecutionRef: parentExecution, OccurredAt: clock.Now(),
		}}, OperationAt: clock.Now(),
	})
	if err != nil || !changed || persistedAck.Ref != ackRef {
		t.Fatalf("ack changed=%v ack=%+v err=%v cause=%v", changed, persistedAck, err, errors.Unwrap(err))
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	final := assertMailboxState(t, repository, envelope, application.MailboxStateAcknowledged, 2)
	finalGoal, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	if final.Acknowledgement == nil || len(finalGoal.Goal.ChildHandoffResolutions()) != 1 ||
		finalGoal.Goal.ChildHandoffResolutions()[0].ReceiptRef() != ackRef ||
		finalGoal.Goal.PlanGeneration() != envelope.TargetPlanGeneration+1 ||
		len(finalGoal.Mailboxes) != 1 ||
		finalGoal.Mailboxes[0].Admission.Ref != admission.Ref ||
		finalGoal.Mailboxes[0].Attempts[1].ConsumptionRef != "receipt:mailbox-consumed" ||
		finalGoal.Mailboxes[0].Acknowledgement == nil ||
		finalGoal.Mailboxes[0].Acknowledgement.Ref != ackRef {
		t.Fatalf("restart lost terminal mailbox/Goal fact: mailbox=%+v goal=%+v", final, finalGoal.Goal.Snapshot())
	}
	if replay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: "request:mailbox-claim-two",
		RequestFingerprint: claimFingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
	}); !application.IsStateError(err, application.StateConflict) || found {
		t.Fatalf("terminal ACK replayed claim replay=%+v found=%v err=%v", replay, found, err)
	}
	admissionReplay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationAdmit, RequestRef: envelope.RequestRef,
		RequestFingerprint: envelope.RequestFingerprint, PrincipalRef: source.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef,
	})
	if err != nil || !found || admissionReplay.Record.State != application.MailboxStateAcknowledged ||
		len(admissionReplay.Record.Attempts) != 2 || admissionReplay.Record.Acknowledgement == nil ||
		!reflect.DeepEqual(admissionReplay.Record.Envelope, envelope) ||
		!reflect.DeepEqual(admissionReplay.Record.Admission, admission) {
		t.Fatalf("admission replay lost immutable/current facts replay=%+v found=%v err=%v", admissionReplay, found, err)
	}

	replay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationDeliver, RequestRef: "request:mailbox-deliver",
		RequestFingerprint: deliveryFingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: messageRef,
	})
	if err != nil || !found || replay.Record.State != application.MailboxStateDelivered ||
		replay.Record.Attempts[len(replay.Record.Attempts)-1].ConsumptionRef != "" {
		t.Fatalf("historical delivery replay=%+v found=%v err=%v", replay, found, err)
	}
	assertAcknowledgedMailboxAllowsBirthGenerationReplacement(
		t, repository, clock, envelope, genericClaim, found,
	)

	recovery, _, _ := newV09TestRecovery(t, repository, clock.Now().Add(time.Minute), nil)
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup V13 mailbox: %v cause=%v", err, errors.Unwrap(err))
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:mailbox-v13")
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, targetRef); err != nil {
		t.Fatalf("restore V13 mailbox: %v", err)
	}
	targetPath, err := recovery.TargetPath(targetRef)
	sqliteTestNoError(t, err)
	restored := openMailboxTestRepository(t, targetPath, clock)
	defer restored.Close()
	restoredMailbox := assertMailboxState(
		t, restored, envelope, application.MailboxStateAcknowledged, 2,
	)
	restoredGoal, err := restored.GetGoal(ctx, envelope.GoalRef)
	if err != nil || restoredMailbox.Acknowledgement == nil ||
		len(restoredGoal.Goal.ChildHandoffResolutions()) != 1 {
		t.Fatalf("recovery lost mailbox causal facts mailbox=%+v goal=%+v err=%v", restoredMailbox, restoredGoal, err)
	}
	t.Run("record_rejects_successor_after_consumption", func(t *testing.T) {
		invalid := restoredMailbox
		invalid.Attempts = append([]application.MailboxDeliveryAttempt(nil), restoredMailbox.Attempts...)
		invalid.Attempts[0].DeliveryRef = "receipt:tampered-first-delivery"
		invalid.Attempts[0].DeliveredAt = invalid.Attempts[0].ClaimedAt.Add(time.Second)
		invalid.Attempts[0].ConsumptionRef = "receipt:tampered-first-consumption"
		invalid.Attempts[0].ConsumedAt = invalid.Attempts[0].ClaimedAt.Add(2 * time.Second)
		if err := validatePersistedMailboxRecord(invalid); err == nil {
			t.Fatal("mailbox record accepted a successor after consumed attempt")
		}
	})
	for _, tamper := range []mailboxRecoveryTamper{
		{
			name: "summary", trigger: "mailbox_envelopes_immutable_update",
			tamperSQL:   `UPDATE mailbox_envelopes SET summary = '   ' WHERE ref = ?`,
			tamperArgs:  []any{messageRef.String()},
			restoreSQL:  `UPDATE mailbox_envelopes SET summary = ? WHERE ref = ?`,
			restoreArgs: []any{envelope.Summary, messageRef.String()},
		},
		{
			name: "content_hash", trigger: "mailbox_envelopes_immutable_update",
			tamperSQL:   `UPDATE mailbox_envelopes SET content_hash = ? WHERE ref = ?`,
			tamperArgs:  []any{strings.Repeat("b", 64), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_envelopes SET content_hash = ? WHERE ref = ?`,
			restoreArgs: []any{envelope.ContentHash, messageRef.String()},
		},
		{
			name: "admission_fingerprint", trigger: "mailbox_admission_receipts_immutable_update",
			tamperSQL:   `UPDATE mailbox_admission_receipts SET request_fingerprint = ? WHERE mailbox_message_ref = ?`,
			tamperArgs:  []any{strings.Repeat("a", 64), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_admission_receipts SET request_fingerprint = ? WHERE mailbox_message_ref = ?`,
			restoreArgs: []any{envelope.RequestFingerprint, messageRef.String()},
		},
		{
			name: "claim_fingerprint", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL:   `UPDATE mailbox_delivery_attempts SET claim_request_fingerprint = ? WHERE mailbox_message_ref = ? AND fence = 2`,
			tamperArgs:  []any{strings.Repeat("b", 64), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_attempts SET claim_request_fingerprint = ? WHERE mailbox_message_ref = ? AND fence = 2`,
			restoreArgs: []any{claimFingerprint, messageRef.String()},
		},
		{
			name: "delivery_fingerprint", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL:   `UPDATE mailbox_delivery_attempts SET delivery_request_fingerprint = ? WHERE mailbox_message_ref = ? AND fence = 2`,
			tamperArgs:  []any{strings.Repeat("c", 64), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_attempts SET delivery_request_fingerprint = ? WHERE mailbox_message_ref = ? AND fence = 2`,
			restoreArgs: []any{deliveryFingerprint, messageRef.String()},
		},
		{
			name: "consumption_fingerprint", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL:   `UPDATE mailbox_delivery_attempts SET consumption_request_fingerprint = ? WHERE mailbox_message_ref = ? AND fence = 2`,
			tamperArgs:  []any{strings.Repeat("d", 64), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_attempts SET consumption_request_fingerprint = ? WHERE mailbox_message_ref = ? AND fence = 2`,
			restoreArgs: []any{consumeFingerprint, messageRef.String()},
		},
		{
			name: "ack_fingerprint", trigger: "mailbox_delivery_acks_immutable_update",
			tamperSQL:   `UPDATE mailbox_delivery_acks SET request_fingerprint = ? WHERE mailbox_message_ref = ?`,
			tamperArgs:  []any{strings.Repeat("e", 64), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_acks SET request_fingerprint = ? WHERE mailbox_message_ref = ?`,
			restoreArgs: []any{ack.RequestFingerprint, messageRef.String()},
		},
		{
			name: "admission_authorization", trigger: "mailbox_admission_receipts_immutable_update",
			tamperSQL:   `UPDATE mailbox_admission_receipts SET authorization_receipt_ref = ? WHERE mailbox_message_ref = ?`,
			tamperArgs:  []any{ackAuthorization.Ref(), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_admission_receipts SET authorization_receipt_ref = ? WHERE mailbox_message_ref = ?`,
			restoreArgs: []any{admitAuthorization.Ref(), messageRef.String()},
		},
		{
			name: "claim_authorization", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL: `UPDATE mailbox_delivery_attempts SET claim_authorization_receipt_ref = ?
WHERE mailbox_message_ref = ? AND fence = 2`,
			tamperArgs: []any{admitAuthorization.Ref(), messageRef.String()},
			restoreSQL: `UPDATE mailbox_delivery_attempts SET claim_authorization_receipt_ref = ?
WHERE mailbox_message_ref = ? AND fence = 2`,
			restoreArgs: []any{reclaimAuthorization.Ref(), messageRef.String()},
		},
		{
			name: "delivery_authorization", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL: `UPDATE mailbox_delivery_attempts SET delivery_authorization_receipt_ref = ?
WHERE mailbox_message_ref = ? AND fence = 2`,
			tamperArgs: []any{admitAuthorization.Ref(), messageRef.String()},
			restoreSQL: `UPDATE mailbox_delivery_attempts SET delivery_authorization_receipt_ref = ?
WHERE mailbox_message_ref = ? AND fence = 2`,
			restoreArgs: []any{deliveryAuthorization.Ref(), messageRef.String()},
		},
		{
			name: "delivery_authorization_time", trigger: "authorization_receipts_immutable_update",
			tamperSQL:   `UPDATE authorization_receipts SET recorded_at = ? WHERE ref = ?`,
			tamperArgs:  []any{delivered.Attempts[len(delivered.Attempts)-1].DeliveredAt.Add(time.Second).UnixNano(), deliveryAuthorization.Ref()},
			restoreSQL:  `UPDATE authorization_receipts SET recorded_at = ? WHERE ref = ?`,
			restoreArgs: []any{deliveryAuthorization.RecordedAt().UnixNano(), deliveryAuthorization.Ref()},
		},
		{
			name: "consumption_authorization", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL: `UPDATE mailbox_delivery_attempts SET consumption_authorization_receipt_ref = ?
WHERE mailbox_message_ref = ? AND fence = 2`,
			tamperArgs: []any{admitAuthorization.Ref(), messageRef.String()},
			restoreSQL: `UPDATE mailbox_delivery_attempts SET consumption_authorization_receipt_ref = ?
WHERE mailbox_message_ref = ? AND fence = 2`,
			restoreArgs: []any{consumeAuthorization.Ref(), messageRef.String()},
		},
		{
			name: "attempt_fence", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL:   `UPDATE mailbox_delivery_attempts SET fence = 9 WHERE mailbox_message_ref = ? AND fence = 1`,
			tamperArgs:  []any{messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_attempts SET fence = 1 WHERE mailbox_message_ref = ? AND fence = 9`,
			restoreArgs: []any{messageRef.String()},
		},
		{
			name: "attempt_overlap", trigger: "mailbox_delivery_attempt_progress_guard",
			tamperSQL: `UPDATE mailbox_delivery_attempts SET lease_until = ?
WHERE mailbox_message_ref = ? AND fence = 1`,
			tamperArgs: []any{claim.Attempt.ClaimedAt.Add(time.Second).UnixNano(), messageRef.String()},
			restoreSQL: `UPDATE mailbox_delivery_attempts SET lease_until = ?
WHERE mailbox_message_ref = ? AND fence = 1`,
			restoreArgs: []any{firstClaim.Attempt.LeaseUntil.UnixNano(), messageRef.String()},
		},
		{
			name: "ack_plan_binding", trigger: "mailbox_delivery_acks_immutable_update",
			tamperSQL:   `UPDATE mailbox_delivery_acks SET expected_plan_generation = ? WHERE mailbox_message_ref = ?`,
			tamperArgs:  []any{int64(beforeAck.Goal.PlanGeneration() + 1), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_acks SET expected_plan_generation = ? WHERE mailbox_message_ref = ?`,
			restoreArgs: []any{int64(beforeAck.Goal.PlanGeneration()), messageRef.String()},
		},
		{
			name: "ack_recipient_scope", trigger: "mailbox_delivery_acks_immutable_update",
			tamperSQL:   `UPDATE mailbox_delivery_acks SET recipient_principal_ref = ? WHERE mailbox_message_ref = ?`,
			tamperArgs:  []any{envelope.Source.PrincipalRef.String(), messageRef.String()},
			restoreSQL:  `UPDATE mailbox_delivery_acks SET recipient_principal_ref = ? WHERE mailbox_message_ref = ?`,
			restoreArgs: []any{recipient.Ref.String(), messageRef.String()},
		},
	} {
		t.Run("recovery_rejects_"+tamper.name, func(t *testing.T) {
			assertMailboxRecoveryTamperRejected(t, restored, envelope, tamper)
		})
	}
}

func assertConsumedMailboxRetirement(
	t *testing.T,
	source *Repository,
	clock *sqliteMembershipClock,
	envelope application.MailboxEnvelope,
	recipient identity.Principal,
	mailboxClaim application.MailboxClaim,
	initialActionClaim application.ActionClaim,
	initialActionFound bool,
	deliveryFingerprint string,
	consumeFingerprint string,
) {
	ctx := context.Background()
	recovery, _, _ := newV09TestRecovery(t, source, clock.Now(), nil)
	if _, _, recoveryErr := validateRecoveryDatabase(ctx, source.db); recoveryErr != nil {
		t.Fatalf("pre-backup mailbox FIFO recovery: %v", recoveryErr)
	}
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup consumed mailbox branch: %v", err)
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:mailbox-retirement")
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, targetRef); err != nil {
		t.Fatalf("restore consumed mailbox branch: %v", err)
	}
	path, err := recovery.TargetPath(targetRef)
	sqliteTestNoError(t, err)
	repository := openMailboxTestRepository(t, path, clock)
	defer repository.Close()

	actionClaim := initialActionClaim
	if !initialActionFound || actionClaim.Action.ExecutionRef != envelope.Recipient.ExecutionRef {
		actionClaim, initialActionFound, err = repository.ClaimNextAction(ctx, application.ClaimRequest{
			WorkerRef: "worker:mailbox-retirement", Token: "token:mailbox-retirement",
			LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
		})
	}
	if err != nil || !initialActionFound || actionClaim.Action.ExecutionRef != envelope.Recipient.ExecutionRef ||
		actionClaim.Action.Kind != application.ActionObserveAgent {
		t.Fatalf("claim recipient observe for retirement claim=%+v found=%v err=%v", actionClaim, initialActionFound, err)
	}
	record, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(envelope.ParentWorkItemRef)
	if !found {
		t.Fatal("retirement recipient WorkItem missing")
	}
	var execution application.ExecutionRecord
	for _, candidate := range record.Executions {
		if candidate.Ref == envelope.Recipient.ExecutionRef {
			execution = candidate
			break
		}
	}
	if execution.Ref.String() == "" {
		t.Fatal("retirement recipient execution missing")
	}
	expectedGoalRevision, expectedItemRevision := record.Goal.Revision(), item.Revision()
	failedAt := clock.Now()
	replacementState := buildMailboxReplacementState(
		t, record, item, execution, actionClaim, failedAt,
		"mailbox-replacement-must-block", "agent.retry_must_retire_mailbox",
	)
	if err := repository.RecordExecutionReplaced(ctx, replacementState); !application.IsStateError(err, application.StateRecipientMailboxActive) {
		t.Fatalf("replacement did not lose mailbox race: %v cause=%v", err, errors.Unwrap(err))
	}
	afterRejectedReplacement, err := repository.GetGoal(ctx, envelope.GoalRef)
	if err != nil || len(afterRejectedReplacement.Executions) != len(record.Executions) {
		t.Fatalf("mailbox replacement rollback record=%+v err=%v", afterRejectedReplacement, err)
	}
	failedGoal, err := record.Goal.FailWorkItem(
		expectedGoalRevision, expectedItemRevision, item.Ref(), failedAt,
	)
	sqliteTestNoError(t, err)
	execution.State = application.ExecutionFailed
	execution.FinishedAt = failedAt
	execution.FailureCode = "agent.recipient_failed"
	events := []application.EventRecord{{
		Ref: "event:mailbox-recipient-failed", Kind: "work_item.failed", GoalRef: envelope.GoalRef,
		WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: failedAt,
	}}
	if failedGoal.State() == goal.GoalStateFailed {
		events = append(events, application.EventRecord{
			Ref: "event:mailbox-goal-failed", Kind: "goal.failed", GoalRef: envelope.GoalRef,
			OccurredAt: failedAt,
		})
	}
	failedState := application.GoalFailedState{
		Claim: actionClaim, ExpectedGoalRevision: expectedGoalRevision,
		ExpectedItemRevision: expectedItemRevision, Goal: failedGoal, Execution: execution,
		Events: events, BudgetSettlement: sqliteMailboxSettlement(t, record, execution, failedAt),
		OperationAt: failedAt,
	}
	seeded := events[0]
	if _, err := repository.db.Exec(`
INSERT INTO events(ref, kind, goal_ref, work_item_ref, execution_ref, occurred_at)
VALUES (?, ?, ?, ?, ?, ?)`, seeded.Ref, seeded.Kind, seeded.GoalRef.String(),
		seeded.WorkItemRef.String(), seeded.ExecutionRef.String(), requiredTime(seeded.OccurredAt)); err != nil {
		t.Fatal(err)
	}
	if err := repository.RecordGoalFailed(ctx, failedState); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("late failure event did not roll back retirement: %v", err)
	}
	var retirementCount int
	var rollbackRetiredAt sql.NullInt64
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM mailbox_retirements WHERE mailbox_message_ref = ?`,
		envelope.Ref.String()).Scan(&retirementCount); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT retired_at FROM outbox WHERE mailbox_message_ref = ?`,
		envelope.Ref.String()).Scan(&rollbackRetiredAt); err != nil {
		t.Fatal(err)
	}
	if retirementCount != 0 || rollbackRetiredAt.Valid {
		t.Fatalf("failed Goal mutation leaked retirement count=%d outbox=%v", retirementCount, rollbackRetiredAt)
	}
	if _, err := repository.db.Exec(`DELETE FROM events WHERE ref = ?`, seeded.Ref); err != nil {
		t.Fatal(err)
	}
	type mutationResult struct {
		kind string
		err  error
	}
	start := make(chan struct{})
	results := make(chan mutationResult, 2)
	go func() {
		<-start
		results <- mutationResult{kind: "replace", err: repository.RecordExecutionReplaced(ctx, replacementState)}
	}()
	go func() {
		<-start
		results <- mutationResult{kind: "fail", err: repository.RecordGoalFailed(ctx, failedState)}
	}()
	close(start)
	var replacementErr, failureErr error
	for range 2 {
		result := <-results
		if result.kind == "replace" {
			replacementErr = result.err
		} else {
			failureErr = result.err
		}
	}
	if failureErr != nil {
		t.Fatalf("retire mailbox with recipient failure race: %v cause=%v", failureErr, errors.Unwrap(failureErr))
	}
	if !application.IsStateError(replacementErr, application.StateRecipientMailboxActive) &&
		!application.IsStateError(replacementErr, application.StateConflict) {
		t.Fatalf("replacement escaped failure/retirement race: %v", replacementErr)
	}
	retired := assertMailboxState(t, repository, envelope, application.MailboxStateRetired, len(mailboxClaim.Record.Attempts))
	if retired.Retirement == nil || retired.Acknowledgement != nil ||
		retired.Retirement.RecipientExecutionRef != envelope.Recipient.ExecutionRef ||
		retired.Retirement.FailureCode != execution.FailureCode ||
		!retired.Retirement.RetiredAt.Equal(failedAt) ||
		retired.Attempts[len(retired.Attempts)-1].ConsumptionRef == "" {
		t.Fatalf("retirement lost cause or consumed receipt: %+v", retired)
	}
	var completedAt, retiredAt sql.NullInt64
	var lastError string
	if err := repository.db.QueryRow(`
SELECT completed_at, retired_at, last_error_code FROM outbox WHERE ref = ?`, retired.Action.Ref,
	).Scan(&completedAt, &retiredAt, &lastError); err != nil || !completedAt.Valid || !retiredAt.Valid ||
		lastError != "" {
		t.Fatalf("retired outbox completion=%v retirement=%v last_error=%q err=%v", completedAt, retiredAt, lastError, err)
	}
	claimAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "claim-after-retirement", clock.Now())
	claimFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationClaim, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		envelope.Ref, envelope.ParentWorkItemRef, envelope.Recipient.ExecutionRef, "", 0, "",
	)
	if _, changed, err := repository.ClaimMailbox(ctx, application.ClaimMailboxState{
		RequestRef: "request:mailbox-claim-after-retirement", RequestFingerprint: claimFingerprint,
		AuthorizationReceipt: claimAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: envelope.Recipient.ExecutionRef, Token: "token:claim-after-retirement",
		LeaseDuration: time.Minute, RequestedAt: clock.Now(),
	}); !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("retired mailbox accepted new claim changed=%v err=%v", changed, err)
	}
	if replay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: "request:mailbox-claim-two",
		RequestFingerprint: claimFingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
	}); !application.IsStateError(err, application.StateConflict) || found {
		t.Fatalf("retired terminal replayed claim replay=%+v found=%v err=%v", replay, found, err)
	}
	staleDeliveryAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "deliver-after-retirement", clock.Now())
	if _, changed, err := repository.MarkMailboxDelivered(ctx, application.MarkMailboxDeliveredState{
		RequestRef: "request:mailbox-deliver-after-retirement", RequestFingerprint: deliveryFingerprint,
		AuthorizationReceipt: staleDeliveryAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: envelope.Recipient.ExecutionRef,
		ClaimToken:            mailboxClaim.Attempt.ClaimToken, Fence: mailboxClaim.Attempt.Fence,
		DeliveryRef: "receipt:mailbox-deliver-after-retirement", OperationAt: clock.Now(),
	}); !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("retired mailbox accepted new delivery changed=%v err=%v", changed, err)
	}
	staleConsumeAuthorization := authorizeMailboxTest(t, repository, recipient, envelope, "consume-after-retirement", clock.Now())
	staleReceipt := application.ActionConsumptionReceipt{
		ActionRef: retired.Action.Ref, Kind: application.ActionDeliverMailbox,
		GoalRef: envelope.GoalRef, WorkItemRef: envelope.ParentWorkItemRef,
		ExecutionRef: envelope.Recipient.ExecutionRef, MailboxMessageRef: envelope.Ref,
		PlanGeneration: retired.Action.PlanGeneration, WorkItemGeneration: retired.Action.WorkItemGeneration,
		Fence: mailboxClaim.Attempt.Fence, DeliveryAttempt: mailboxClaim.Attempt.Fence,
		ClaimToken: mailboxClaim.Attempt.ClaimToken, WorkerRef: recipient.Ref.String(),
		Outcome: application.ActionConsumedCompleted, ConsumedAt: clock.Now(),
	}
	if _, changed, err := repository.ConsumeMailbox(ctx, application.ConsumeMailboxState{
		RequestRef: "request:mailbox-consume-after-retirement", RequestFingerprint: consumeFingerprint,
		AuthorizationReceipt: staleConsumeAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: envelope.Recipient.ExecutionRef,
		ClaimToken:            mailboxClaim.Attempt.ClaimToken, Fence: mailboxClaim.Attempt.Fence,
		ConsumptionRef:     "receipt:mailbox-consume-after-retirement",
		ConsumptionReceipt: staleReceipt, OperationAt: clock.Now(),
	}); !application.IsStateError(err, application.StateConflict) || changed {
		t.Fatalf("retired mailbox accepted new consumption changed=%v err=%v", changed, err)
	}
	for _, replayRequest := range []struct {
		kind        application.MailboxMutationKind
		requestRef  string
		fingerprint string
		wantState   application.MailboxState
	}{
		{application.MailboxMutationDeliver, "request:mailbox-deliver", deliveryFingerprint, application.MailboxStateDelivered},
		{application.MailboxMutationConsume, "request:mailbox-consume", consumeFingerprint, application.MailboxStateConsumed},
	} {
		replay, found, err := repository.MailboxReplay(ctx, application.MailboxReplayRequest{
			Kind: replayRequest.kind, RequestRef: replayRequest.requestRef,
			RequestFingerprint: replayRequest.fingerprint, PrincipalRef: recipient.Ref,
			ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		})
		if err != nil || !found || replay.Record.State != replayRequest.wantState || replay.Record.Retirement != nil {
			t.Fatalf("retirement historical replay kind=%s replay=%+v found=%v err=%v", replayRequest.kind, replay, found, err)
		}
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	retired = assertMailboxState(t, repository, envelope, application.MailboxStateRetired, len(mailboxClaim.Record.Attempts))
	if retired.Retirement == nil {
		t.Fatal("restart lost mailbox retirement")
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("recovery rejected canonical mailbox retirement: %v", err)
	}
	assertMailboxRecoveryTamperRejected(t, repository, envelope, mailboxRecoveryTamper{
		name: "retirement_failure_code", trigger: "mailbox_retirements_immutable_update",
		tamperSQL: `UPDATE mailbox_retirements SET failure_code = 'agent.tampered'
WHERE mailbox_message_ref = ?`, tamperArgs: []any{envelope.Ref.String()},
		restoreSQL:  `UPDATE mailbox_retirements SET failure_code = ? WHERE mailbox_message_ref = ?`,
		restoreArgs: []any{execution.FailureCode, envelope.Ref.String()},
	})
}

func assertMailboxListFIFO(
	t *testing.T,
	source *Repository,
	clock *sqliteMembershipClock,
	envelope application.MailboxEnvelope,
) {
	t.Helper()
	ctx := context.Background()
	recovery, _, _ := newV09TestRecovery(t, source, clock.Now(), nil)
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup mailbox FIFO branch: %v cause=%v", err, errors.Unwrap(err))
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:mailbox-fifo")
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, targetRef); err != nil {
		t.Fatalf("restore mailbox FIFO branch: %v", err)
	}
	path, err := recovery.TargetPath(targetRef)
	sqliteTestNoError(t, err)
	repository := openMailboxTestRepository(t, path, clock)
	defer repository.Close()
	first, err := repository.GetMailbox(
		ctx, envelope.ProjectRef, envelope.GoalRef, envelope.Ref, envelope.Recipient,
	)
	sqliteTestNoError(t, err)
	secondChild := mustRef(t, "work-item:mailbox-fifo-child", goal.NewWorkItemRef)
	secondExecution := mustRef(t, "execution:mailbox-fifo-child", goal.NewExecutionRef)
	if _, err := repository.db.Exec(`
INSERT INTO work_items(
    ref, goal_ref, actor_ref, project_ref, objective, phase_key, role_key,
    parent_ref, output_contract, skip_reason, state, revision, position,
    created_at, started_at, finished_at, execution_ref, handoff_required
)
SELECT ?, goal_ref, actor_ref, project_ref, 'mailbox FIFO child', phase_key, role_key,
       parent_ref, output_contract, skip_reason, state, revision,
       (SELECT MAX(position) + 1 FROM work_items WHERE goal_ref = source.goal_ref),
       created_at, started_at, finished_at, ?, 1
FROM work_items source WHERE goal_ref = ? AND ref = ?`,
		secondChild.String(), secondExecution.String(), envelope.GoalRef.String(),
		envelope.ChildWorkItemRef.String(),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec(`
INSERT INTO executions(
    ref, goal_ref, work_item_ref, attempt_no, max_execution_attempts,
    replaces_execution_ref, plan_generation, app_spec_generation, spec_hash,
    state, purpose, artifact_media_type, idempotency_key, max_output_bytes,
    provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
    started_at, provider_accepted_at, last_observed_at, provider_observed_at,
    finished_at, failure_code
)
SELECT ?, goal_ref, ?, attempt_no, max_execution_attempts, NULL, plan_generation,
       app_spec_generation, spec_hash, state, 'work', artifact_media_type, ?, max_output_bytes,
       provider_ref, model_ref, agent_ref, external_ref, created_at, deadline_at,
       started_at, provider_accepted_at, last_observed_at, provider_observed_at,
       finished_at, failure_code
FROM executions WHERE ref = ?`,
		secondExecution.String(), secondChild.String(), "idempotency:"+secondExecution.String(),
		envelope.Source.ExecutionRef.String(),
	); err != nil {
		t.Fatal(err)
	}
	secondRef := mustRef(t, "message:aaa-mailbox-fifo-later", application.NewMailboxMessageRef)
	secondEnvelope := envelope
	secondEnvelope.Ref = secondRef
	secondEnvelope.RequestRef = "request:mailbox-fifo-later"
	secondEnvelope.ChildWorkItemRef = secondChild
	secondEnvelope.Source.WorkItemRef = secondChild
	secondEnvelope.Source.ExecutionRef = secondExecution
	secondEnvelope.ArtifactRefs = nil
	secondEnvelope.Summary = "compact mailbox summary"
	secondEnvelope.AdmittedAt = envelope.AdmittedAt.Add(time.Second)
	secondEnvelope.RequestFingerprint = application.MailboxAdmissionFingerprint(secondEnvelope)
	secondEnvelope.ContentHash = application.MailboxEnvelopeContentHash(secondEnvelope)
	mailboxInsertEnvelope(t, repository.db, mailboxEnvelopeSeed{
		Ref: secondRef.String(), RequestRef: secondEnvelope.RequestRef,
		Fingerprint: secondEnvelope.RequestFingerprint, ProjectRef: envelope.ProjectRef.String(),
		GoalRef: envelope.GoalRef.String(), PlanGeneration: int64(envelope.TargetPlanGeneration),
		Kind: string(envelope.Kind), Hash: secondEnvelope.ContentHash,
		SourcePrincipalRef: envelope.Source.PrincipalRef.String(), ChildWorkItemRef: secondChild.String(),
		SourceExecutionRef: secondExecution.String(), RecipientPrincipalRef: envelope.Recipient.PrincipalRef.String(),
		ParentWorkItemRef:     envelope.ParentWorkItemRef.String(),
		RecipientExecutionRef: envelope.Recipient.ExecutionRef.String(),
		RecipientGeneration:   int64(first.Action.WorkItemGeneration),
		AdmittedAt:            requiredTime(secondEnvelope.AdmittedAt),
	})
	if _, err := repository.db.Exec(`
INSERT INTO mailbox_admission_receipts(
    ref, mailbox_message_ref, request_ref, request_fingerprint,
    authorization_receipt_ref, project_ref, goal_ref, source_principal_ref, admitted_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"receipt:mailbox-fifo-later", secondRef.String(), secondEnvelope.RequestRef,
		secondEnvelope.RequestFingerprint, first.Admission.AuthorizationReceipt.Ref(),
		envelope.ProjectRef.String(), envelope.GoalRef.String(), envelope.Source.PrincipalRef.String(),
		requiredTime(secondEnvelope.AdmittedAt),
	); err != nil {
		t.Fatal(err)
	}
	mailboxInsertDeliveryAction(
		t, repository.db, "action:mailbox:"+secondRef.String(), secondRef.String(),
		envelope.GoalRef.String(), envelope.ParentWorkItemRef.String(),
		envelope.Recipient.ExecutionRef.String(), int64(envelope.TargetPlanGeneration),
		int64(first.Action.WorkItemGeneration), requiredTime(secondEnvelope.AdmittedAt),
	)
	listed, err := repository.ListMailbox(
		ctx, envelope.ProjectRef, envelope.GoalRef, envelope.Recipient, 10,
	)
	if err != nil || len(listed) != 2 || listed[0].Envelope.Ref != envelope.Ref ||
		listed[1].Envelope.Ref != secondRef {
		t.Fatalf("mailbox list not FIFO: records=%+v err=%v cause=%v", listed, err, errors.Unwrap(err))
	}
	limited, err := repository.ListMailbox(
		ctx, envelope.ProjectRef, envelope.GoalRef, envelope.Recipient, 1,
	)
	if err != nil || len(limited) != 1 || limited[0].Envelope.Ref != envelope.Ref {
		t.Fatalf("mailbox FIFO limit skipped oldest: records=%+v err=%v", limited, err)
	}
}

func assertMailboxClaimReplayHistorical(
	t *testing.T,
	repository *Repository,
	envelope application.MailboxEnvelope,
	recipient identity.Principal,
	requestRef string,
	fingerprint string,
	want application.MailboxDeliveryAttempt,
) {
	t.Helper()
	replay, found, err := repository.MailboxReplay(context.Background(), application.MailboxReplayRequest{
		Kind: application.MailboxMutationClaim, RequestRef: requestRef,
		RequestFingerprint: fingerprint, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
	})
	if err != nil || !found || replay.Claim.Record.State != application.MailboxStateClaimed ||
		replay.Claim.Attempt != want || replay.Claim.Record.Acknowledgement != nil ||
		replay.Claim.Record.Retirement != nil || replay.Claim.Attempt.DeliveryRef != "" ||
		replay.Claim.Attempt.ConsumptionRef != "" {
		t.Fatalf("claim replay did not preserve claimed frontier replay=%+v found=%v err=%v", replay, found, err)
	}
}

func assertMailboxFrontierRetires(
	t *testing.T,
	source *Repository,
	clock *sqliteMembershipClock,
	envelope application.MailboxEnvelope,
	wantFrontier application.MailboxState,
	suffix string,
) {
	t.Helper()
	ctx := context.Background()
	before, err := source.GetMailbox(
		ctx, envelope.ProjectRef, envelope.GoalRef, envelope.Ref, envelope.Recipient,
	)
	if err != nil || before.State != wantFrontier {
		t.Fatalf("%s retirement source state=%s err=%v", suffix, before.State, err)
	}
	recovery, _, _ := newV09TestRecovery(t, source, clock.Now(), nil)
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup %s retirement frontier: %v", suffix, err)
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:mailbox-retire-" + suffix)
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, targetRef); err != nil {
		t.Fatalf("restore %s retirement frontier: %v", suffix, err)
	}
	path, err := recovery.TargetPath(targetRef)
	sqliteTestNoError(t, err)
	repository := openMailboxTestRepository(t, path, clock)
	defer repository.Close()
	capabilities := sqliteTestCapabilities()
	capabilities.Unrestricted = false
	capabilities.RoleKeys = []string{"role:mailbox"}
	actionClaim, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:mailbox-retire-" + suffix, Token: "token:mailbox-retire-" + suffix,
		LeaseDuration: time.Minute, Capabilities: capabilities, BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found || actionClaim.Action.Kind != application.ActionObserveAgent ||
		actionClaim.Action.ExecutionRef != envelope.Recipient.ExecutionRef {
		t.Fatalf("claim %s recipient action=%+v found=%v err=%v", suffix, actionClaim, found, err)
	}
	record, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(envelope.ParentWorkItemRef)
	if !found {
		t.Fatal("retirement frontier parent missing")
	}
	var execution application.ExecutionRecord
	for _, candidate := range record.Executions {
		if candidate.Ref == envelope.Recipient.ExecutionRef {
			execution = candidate
			break
		}
	}
	failedAt := clock.Now()
	failedGoal, err := record.Goal.FailWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), failedAt,
	)
	sqliteTestNoError(t, err)
	execution.State = application.ExecutionFailed
	execution.FinishedAt = failedAt
	execution.FailureCode = "agent.frontier_failed_" + suffix
	budgetSettlement := sqliteMailboxSettlement(t, record, execution, failedAt)
	events := []application.EventRecord{{
		Ref: "event:mailbox-frontier-failed:" + suffix, Kind: "work_item.failed",
		GoalRef: envelope.GoalRef, WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		OccurredAt: failedAt,
	}}
	if failedGoal.State() == goal.GoalStateFailed {
		events = append(events, application.EventRecord{
			Ref: "event:mailbox-frontier-goal-failed:" + suffix, Kind: "goal.failed",
			GoalRef: envelope.GoalRef, OccurredAt: failedAt,
		})
	}
	if err := repository.RecordGoalFailed(ctx, application.GoalFailedState{
		Claim: actionClaim, ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedItemRevision: item.Revision(), Goal: failedGoal, Execution: execution,
		Events: events, BudgetSettlement: budgetSettlement, OperationAt: failedAt,
	}); err != nil {
		t.Fatalf("retire %s frontier: %v cause=%v", suffix, err, errors.Unwrap(err))
	}
	retired := assertMailboxState(t, repository, envelope, application.MailboxStateRetired, len(before.Attempts))
	if retired.Retirement == nil || retired.Acknowledgement != nil ||
		!reflect.DeepEqual(retired.Attempts, before.Attempts) {
		t.Fatalf("%s retirement changed frontier evidence before=%+v after=%+v", suffix, before, retired)
	}
	var completedAt, retiredAt sql.NullInt64
	if err := repository.db.QueryRow(`SELECT completed_at, retired_at FROM outbox WHERE ref = ?`,
		retired.Action.Ref).Scan(&completedAt, &retiredAt); err != nil || completedAt.Valid || !retiredAt.Valid {
		t.Fatalf("%s retired outbox completed=%v retired=%v err=%v", suffix, completedAt, retiredAt, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("recovery rejected %s retirement frontier: %v cause=%v", suffix, err, errors.Unwrap(err))
	}
	repository = restartMailboxTestRepository(t, repository, path, clock)
	if restarted := assertMailboxState(
		t, repository, envelope, application.MailboxStateRetired, len(before.Attempts),
	); restarted.Retirement == nil {
		t.Fatalf("restart lost %s retirement", suffix)
	}
}

func buildMailboxReplacementState(
	t *testing.T,
	record application.GoalRecord,
	item goal.WorkItem,
	execution application.ExecutionRecord,
	claim application.ActionClaim,
	at time.Time,
	suffix string,
	failureCode string,
) application.ExecutionReplacedState {
	t.Helper()
	replacementRef := mustRef(t, "execution:"+suffix, goal.NewExecutionRef)
	replacedGoal, err := record.Goal.ReplaceWorkItemExecution(
		record.Goal.Revision(), item.Revision(), item.Ref(), execution.Ref, replacementRef, at,
	)
	sqliteTestNoError(t, err)
	failed := execution
	failed.State = application.ExecutionFailed
	failed.FinishedAt = at
	failed.FailureCode = failureCode
	replacement := application.ExecutionRecord{
		Ref: replacementRef, GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef,
		AttemptNo: execution.AttemptNo + 1, MaxExecutionAttempts: execution.MaxExecutionAttempts,
		ReplacesExecutionRef: execution.Ref, PlanGeneration: execution.PlanGeneration,
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		State: application.ExecutionQueued, ArtifactMediaType: execution.ArtifactMediaType,
		IdempotencyKey: "execution:" + replacementRef.String(), MaxOutputBytes: execution.MaxOutputBytes,
		CreatedAt: at,
	}
	replacedItem, _ := replacedGoal.WorkItem(item.Ref())
	next := application.ActionRecord{
		Ref: "action:launch:" + replacementRef.String(), Kind: application.ActionLaunchAgent,
		GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacementRef,
		PlanGeneration: replacement.PlanGeneration, WorkItemGeneration: replacedItem.Revision(),
		AvailableAt: at,
	}
	if len(replacedItem.WriteSet()) != 0 {
		replacement.RepositoryRef = execution.RepositoryRef
		if replacement.RepositoryRef.String() == "" {
			replacement.RepositoryRef, err = identity.NewRepositoryRef("repository:" + record.Goal.Project().String())
			sqliteTestNoError(t, err)
		}
		replacement.ExecutionWorkspaceRef, err = ports.NewExecutionWorkspaceRef(
			"execution-workspace:" + replacementRef.String(),
		)
		sqliteTestNoError(t, err)
		next.Ref = "action:prepare-workspace:" + replacementRef.String()
		next.Kind = application.ActionPrepareWorkspace
		authority := application.WorkItemAuthority{}
		for _, candidate := range record.WorkItemAuthorities {
			if candidate.WorkItemRef == replacedItem.Ref() {
				authority = candidate
				break
			}
		}
		if authority.PrincipalRef.String() == "" {
			t.Fatal("mailbox replacement WorkItem authority missing")
		}
		next = legacyGovernedExecutionAction(
			t, replacedGoal, replacedItem, replacement, next, authority, sqliteTestBudgetPolicy(at),
		)
	}
	return application.ExecutionReplacedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: replacedGoal, FailedExecution: failed, ReplacementExecution: replacement,
		NextAction: next, ErrorCode: failureCode,
		BudgetSettlement: sqliteMailboxSettlement(t, record, failed, at), OperationAt: at,
		Events: []application.EventRecord{
			{Ref: "event:" + suffix + ":failed", Kind: "execution.failed", GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at},
			{Ref: "event:" + suffix + ":queued", Kind: "execution.queued", GoalRef: record.Goal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: replacementRef, OccurredAt: at},
		},
	}
}

func assertAcknowledgedMailboxAllowsBirthGenerationReplacement(
	t *testing.T,
	source *Repository,
	clock *sqliteMembershipClock,
	envelope application.MailboxEnvelope,
	claim application.ActionClaim,
	claimFound bool,
) {
	t.Helper()
	ctx := context.Background()
	if !claimFound || claim.Action.ExecutionRef != envelope.Recipient.ExecutionRef ||
		claim.Action.PlanGeneration != envelope.TargetPlanGeneration {
		t.Fatalf("birth-generation recipient claim invalid: %+v found=%v", claim, claimFound)
	}
	recovery, _, _ := newV09TestRecovery(t, source, clock.Now(), nil)
	backup, err := recovery.CreateBackup(ctx)
	if err != nil {
		t.Fatalf("backup acknowledged replacement branch: %v", err)
	}
	targetRef, _ := application.NewRecoveryTargetRef("recovery-target:mailbox-ack-replacement")
	if _, err := recovery.RestoreBackup(ctx, backup.Ref, targetRef); err != nil {
		t.Fatalf("restore acknowledged replacement branch: %v", err)
	}
	path, err := recovery.TargetPath(targetRef)
	sqliteTestNoError(t, err)
	repository := openMailboxTestRepository(t, path, clock)
	defer repository.Close()
	record, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	if record.Goal.PlanGeneration() <= claim.Action.PlanGeneration {
		t.Fatalf("test lacks later Goal generation: action=%d goal=%d", claim.Action.PlanGeneration, record.Goal.PlanGeneration())
	}
	item, found := record.Goal.WorkItem(envelope.ParentWorkItemRef)
	if !found {
		t.Fatal("acknowledged replacement WorkItem missing")
	}
	var execution application.ExecutionRecord
	for _, candidate := range record.Executions {
		if candidate.Ref == envelope.Recipient.ExecutionRef {
			execution = candidate
			break
		}
	}
	state := buildMailboxReplacementState(
		t, record, item, execution, claim, clock.Now(),
		"mailbox-ack-birth-generation", "agent.observe_retry",
	)
	if state.NextAction.PlanGeneration != execution.PlanGeneration ||
		state.NextAction.PlanGeneration >= record.Goal.PlanGeneration() {
		t.Fatalf("replacement action lost birth generation: action=%d execution=%d goal=%d",
			state.NextAction.PlanGeneration, execution.PlanGeneration, record.Goal.PlanGeneration())
	}
	if err := repository.RecordExecutionReplaced(ctx, state); err != nil {
		t.Fatalf("acknowledged mailbox blocked valid birth-generation replacement: %v cause=%v", err, errors.Unwrap(err))
	}
	var planGeneration int64
	var retiredAt sql.NullInt64
	if err := repository.db.QueryRow(`
SELECT plan_generation, retired_at FROM outbox WHERE ref = ?`, state.NextAction.Ref,
	).Scan(&planGeneration, &retiredAt); err != nil ||
		planGeneration != int64(execution.PlanGeneration) || retiredAt.Valid {
		t.Fatalf("replacement launch action plan=%d retired=%v err=%v", planGeneration, retiredAt, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("recovery rejected birth-generation replacement: %v cause=%v", err, errors.Unwrap(err))
	}
}

type mailboxRecoveryTamper struct {
	name        string
	trigger     string
	tamperSQL   string
	tamperArgs  []any
	restoreSQL  string
	restoreArgs []any
}

func assertMailboxRecoveryTamperRejected(
	t *testing.T,
	repository *Repository,
	envelope application.MailboxEnvelope,
	tamper mailboxRecoveryTamper,
) {
	t.Helper()
	var triggerSQL string
	if err := repository.db.QueryRow(`SELECT sql FROM sqlite_schema WHERE type = 'trigger' AND name = ?`,
		tamper.trigger,
	).Scan(&triggerSQL); err != nil {
		t.Fatal(err)
	}
	connection, err := repository.db.Conn(context.Background())
	sqliteTestNoError(t, err)
	defer connection.Close()
	apply := func(query string, arguments []any) {
		t.Helper()
		if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys = OFF`); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints = ON`); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(context.Background(), `DROP TRIGGER `+tamper.trigger); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(context.Background(), query, arguments...); err != nil {
			_, _ = connection.ExecContext(context.Background(), triggerSQL)
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(context.Background(), triggerSQL); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints = OFF`); err != nil {
			t.Fatal(err)
		}
		if _, err := connection.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`); err != nil {
			t.Fatal(err)
		}
	}
	apply(tamper.tamperSQL, tamper.tamperArgs)
	if _, err := repository.GetMailbox(
		context.Background(), envelope.ProjectRef, envelope.GoalRef, envelope.Ref, envelope.Recipient,
	); err == nil {
		t.Fatalf("runtime reader accepted %s tamper", tamper.name)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err == nil {
		t.Fatalf("recovery accepted %s tamper", tamper.name)
	}
	apply(tamper.restoreSQL, tamper.restoreArgs)
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("restoring %s did not recover canonical state: %v", tamper.name, err)
	}
}

type mailboxGoalFixture struct {
	state     application.CreateGoalState
	parentRef goal.WorkItemRef
	childRef  goal.WorkItemRef
}

func newMailboxGoalFixture(t *testing.T, at time.Time) mailboxGoalFixture {
	t.Helper()
	return newMailboxGoalFixtureWithHandoff(t, at, true)
}

func newMailboxGoalFixtureWithHandoff(
	t *testing.T,
	at time.Time,
	handoffRequired bool,
) mailboxGoalFixture {
	t.Helper()
	base := newCreateFixture(
		t, "mailbox-runtime", "request:mailbox-goal", "fingerprint:mailbox-goal",
		"actor:mailbox-source", "project:mailbox-runtime",
	)
	pending, err := goal.NewGoal(
		mustRef(t, "goal:mailbox-runtime", goal.NewGoalRef), base.Goal.AppSpec(), at,
	)
	sqliteTestNoError(t, err)
	phaseKey, _ := goal.NewPhaseKey("phase:mailbox")
	phase, _ := goal.NewPhaseInstance(phaseKey)
	role, _ := goal.NewRoleKey("role:mailbox")
	parentRef := mustRef(t, "work-item:mailbox-parent", goal.NewWorkItemRef)
	childRef := mustRef(t, "work-item:mailbox-child", goal.NewWorkItemRef)
	parentScope, _ := goal.NewWriteScope("internal/mailbox/parent")
	childScope, _ := goal.NewWriteScope("internal/mailbox/child")
	parent := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: parentRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "mailbox parent", CreatedAt: at, Phase: phaseKey, Role: role,
		WriteSet: []goal.WriteScope{parentScope}, CouncilPolicy: council.PolicyRequired,
		RequiredTests: sqliteRequiredTests(t, "required-test:sqlite-mailbox-parent"), OutputContract: goal.EvidenceBundleOutputContract(),
	})
	child := mustWorkItem(t, goal.NewWorkItemInput{
		Ref: childRef, Goal: pending.Ref(), Actor: pending.Actor(), Project: pending.Project(),
		Objective: "mailbox child", CreatedAt: at, Phase: phaseKey, Role: role, Parent: parentRef,
		HandoffRequired: handoffRequired,
		WriteSet:        []goal.WriteScope{childScope}, CouncilPolicy: council.PolicyRequired,
		RequiredTests: sqliteRequiredTests(t, "required-test:sqlite-mailbox-child"), OutputContract: goal.EvidenceBundleOutputContract(),
	})
	plan, err := goal.NewPlan(goal.PlanInput{
		Generation: 1, Phases: []goal.PhaseInstance{phase}, WorkItems: []goal.WorkItem{child, parent},
	})
	sqliteTestNoError(t, err)
	aggregate, err := pending.ApplyPlan(pending.Revision(), plan)
	if err == nil {
		aggregate, err = aggregate.Start(aggregate.Revision(), at)
	}
	sqliteTestNoError(t, err)
	state := application.CreateGoalState{
		RequestRef: "request:mailbox-goal", RequestFingerprint: "fingerprint:mailbox-goal", Goal: aggregate,
		Events: []application.EventRecord{{
			Ref: "event:goal-created:mailbox", Kind: "goal.created", GoalRef: aggregate.Ref(), OccurredAt: at,
		}},
	}
	for _, item := range aggregate.ReadyWorkItems() {
		executionRef := mustRef(t, "execution:mailbox:"+item.Ref().String(), goal.NewExecutionRef)
		state.Executions = append(state.Executions, application.ExecutionRecord{
			Ref: executionRef, GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(),
			AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: aggregate.PlanGeneration(),
			AppSpecGeneration: aggregate.AppSpec().Generation(), SpecHash: aggregate.SpecHash(),
			State: application.ExecutionQueued, ArtifactMediaType: "text/plain",
			IdempotencyKey: "idempotency:" + executionRef.String(), MaxOutputBytes: 1024, CreatedAt: at,
		})
		state.Actions = append(state.Actions, application.ActionRecord{
			Ref: "action:launch:mailbox:" + item.Ref().String(), Kind: application.ActionLaunchAgent,
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef,
			PlanGeneration: aggregate.PlanGeneration(), WorkItemGeneration: item.Revision(), AvailableAt: at,
		})
		state.Events = append(state.Events, application.EventRecord{
			Ref: "event:execution-queued:mailbox:" + item.Ref().String(), Kind: "execution.queued",
			GoalRef: aggregate.Ref(), WorkItemRef: item.Ref(), ExecutionRef: executionRef, OccurredAt: at,
		})
	}
	return mailboxGoalFixture{state: state, parentRef: parentRef, childRef: childRef}
}

func mustMailboxSchedulerClaim(
	t *testing.T,
	repository *Repository,
	kind string,
	index int,
	at time.Time,
) application.ActionClaim {
	t.Helper()
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef:     "worker:mailbox:" + kind,
		Token:         "token:scheduler:mailbox:" + kind + ":" + time.Unix(0, int64(index)+1).UTC().Format("150405.000000000"),
		LeaseDuration: time.Minute, Capabilities: sqliteTestCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found {
		t.Fatalf("scheduler claim %s found=%v err=%v at=%s", kind, found, err, at)
	}
	if claim.Action.Kind == application.ActionPrepareWorkspace {
		prepareLegacyWorkspaceClaim(t, repository, claim, at)
		return mustMailboxSchedulerClaim(t, repository, kind+"-launch", index, at)
	}
	return claim
}

func startMailboxExecution(t *testing.T, repository *Repository, claim application.ActionClaim, at time.Time) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	item, found := record.Goal.WorkItem(claim.Action.WorkItemRef)
	if !found {
		t.Fatal("claimed WorkItem missing")
	}
	updated, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), claim.Action.ExecutionRef, at,
	)
	sqliteTestNoError(t, err)
	execution := findMailboxExecution(t, record.Executions, claim.Action.ExecutionRef)
	execution.State = application.ExecutionDispatching
	if claim.Action.EffectIntentRef != "" {
		execution.BudgetReservationRef = claim.BudgetReservationRef
		execution.EffectIntentRef = claim.Action.EffectIntentRef
	}
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: updated,
		Execution: execution, OperationAt: at,
		Event: application.EventRecord{
			Ref: "event:dispatching:" + execution.Ref.String(), Kind: "execution.dispatching",
			GoalRef: updated.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}); err != nil {
		t.Fatal(err)
	}
	var effectReceipt application.EffectReceipt
	if claim.Action.EffectIntentRef != "" {
		attempt := sqliteV15Attempt(claim, at)
		attempt, _, err = repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: at,
		})
		sqliteTestNoError(t, err)
		effectReceipt = sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, at)
		execution.LaunchReceiptRef = effectReceipt.Ref
	}
	execution.State = application.ExecutionRunning
	execution.ProviderRef = "provider:codex"
	execution.ModelRef = "model:codex"
	execution.AgentRef = "agent:codex"
	execution.ExternalRef = "external:" + execution.Ref.String()
	execution.StartedAt = at
	execution.DeadlineAt = at.Add(time.Hour)
	execution.ProviderAcceptedAt = at
	startedItem, _ := updated.WorkItem(item.Ref())
	if err := repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: claim, Execution: execution, OperationAt: at,
		NextAction: application.ActionRecord{
			Ref: "action:observe:mailbox:" + item.Ref().String(), Kind: application.ActionObserveAgent,
			GoalRef: updated.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: updated.PlanGeneration(), WorkItemGeneration: startedItem.Revision(), AvailableAt: at,
		},
		EffectReceipt: effectReceipt, Event: application.EventRecord{
			Ref: "event:accepted:" + execution.Ref.String(), Kind: "execution.accepted",
			GoalRef: updated.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func succeedMailboxChild(
	t *testing.T,
	repository *Repository,
	claim application.ActionClaim,
	at time.Time,
) goal.ArtifactRef {
	return succeedMailboxChildWithPostArtifact(t, repository, claim, at, false)
}

func succeedMailboxChildWithPostArtifact(
	t *testing.T,
	repository *Repository,
	claim application.ActionClaim,
	at time.Time,
	postArtifact bool,
) goal.ArtifactRef {
	return succeedMailboxWorkItem(
		t, repository, claim, at, "mailbox-child", "c", false, postArtifact,
	)
}

func succeedMailboxWorkItem(
	t *testing.T,
	repository *Repository,
	claim application.ActionClaim,
	at time.Time,
	suffix string,
	digestMarker string,
	closeGoal bool,
	postArtifact bool,
) goal.ArtifactRef {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), claim.Action.GoalRef)
	sqliteTestNoError(t, err)
	item, _ := record.Goal.WorkItem(claim.Action.WorkItemRef)
	artifactDigest := strings.Repeat(digestMarker, 64)
	artifactRef := mustRef(t, "artifact:sha256:"+artifactDigest, goal.NewArtifactRef)
	attestationRef := mustRef(t, "attestation:"+suffix, goal.NewAttestationRef)
	updated, err := record.Goal.SucceedWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(),
		[]goal.ArtifactRef{artifactRef}, []goal.AttestationRef{attestationRef}, at,
	)
	sqliteTestNoError(t, err)
	if closeGoal {
		updated, err = updated.Close(updated.Revision(), goal.GoalOutcomeSucceeded, at)
		sqliteTestNoError(t, err)
	}
	execution := findMailboxExecution(t, record.Executions, claim.Action.ExecutionRef)
	execution.State = application.ExecutionSucceeded
	execution.FinishedAt = at
	budgetSettlement := sqliteMailboxSettlement(t, record, execution, at)
	artifact := application.ArtifactRecord{
		OccurrenceRef: "artifact-occurrence:" + suffix + ":" + execution.Ref.String(),
		Kind:          application.ArtifactKindAgentOutput,
		Stored:        ports.StoredArtifact{Ref: artifactRef, Digest: artifactDigest, MediaType: "text/plain", Size: 8},
		GoalRef:       updated.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
		ExecutionAttempt: execution.AttemptNo, PlanGeneration: execution.PlanGeneration,
		WorkItemGeneration: item.Revision(), AppSpecGeneration: execution.AppSpecGeneration,
		SpecHash: execution.SpecHash, CreatedAt: at,
	}
	attestation := application.AttestationRecord{
		Ref: attestationRef, Kind: application.AttestationKindArtifactProvenance,
		Verdict: application.AttestationVerdictObserved, GoalRef: updated.Ref(), WorkItemRef: item.Ref(),
		ExecutionRef: execution.Ref, ExecutionAttempt: execution.AttemptNo,
		PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(),
		AppSpecGeneration: execution.AppSpecGeneration, SpecHash: execution.SpecHash,
		ArtifactRef: artifactRef, SubjectDigest: strings.Repeat(digestMarker, 64),
		PolicyRef: "mailbox." + suffix + ".complete", PolicyDigest: strings.Repeat("e", 64),
		StartedAt: at, FinishedAt: at, Policy: "mailbox." + suffix + ".complete", AcceptedAt: at,
	}
	events := []application.EventRecord{{
		Ref: "event:work-succeeded:" + suffix, Kind: "work_item.succeeded",
		GoalRef: updated.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref, OccurredAt: at,
	}}
	if closeGoal {
		events = append(events, application.EventRecord{
			Ref: "event:goal-succeeded:" + suffix, Kind: "goal.succeeded",
			GoalRef: updated.Ref(), OccurredAt: at,
		})
	}
	state := application.GoalSucceededState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(),
		Goal: updated, Execution: execution, Artifact: artifact, Attestation: attestation,
		Events: events, BudgetSettlement: budgetSettlement, OperationAt: at,
	}
	if postArtifact {
		state.PostArtifactAction = &application.ActionRecord{
			Ref: "action:admit-mailbox:" + execution.Ref.String(), Kind: application.ActionAdmitMailbox,
			GoalRef: updated.Ref(), WorkItemRef: item.Ref(), ExecutionRef: execution.Ref,
			PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: at,
		}
	}
	if err := repository.RecordGoalSucceeded(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	return artifactRef
}

func sqliteMailboxSettlement(
	t *testing.T,
	record application.GoalRecord,
	execution application.ExecutionRecord,
	at time.Time,
) *governance.BudgetSettlement {
	t.Helper()
	if execution.BudgetReservationRef == "" {
		return nil
	}
	for _, reservation := range record.BudgetReservations {
		if reservation.Ref != execution.BudgetReservationRef {
			continue
		}
		settlement, err := governance.Reconcile(
			reservation, governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		)
		sqliteTestNoError(t, err)
		settlement.SettledAt = at
		return &settlement
	}
	t.Fatal("mailbox execution budget reservation missing")
	return nil
}

func findMailboxExecution(
	t *testing.T,
	executions []application.ExecutionRecord,
	ref goal.ExecutionRef,
) application.ExecutionRecord {
	t.Helper()
	for _, execution := range executions {
		if execution.Ref == ref {
			return execution
		}
	}
	t.Fatalf("execution %s missing", ref)
	return application.ExecutionRecord{}
}

func authorizeMailboxTest(
	t *testing.T,
	repository *Repository,
	principal identity.Principal,
	envelope application.MailboxEnvelope,
	suffix string,
	at time.Time,
) identity.AuthorizationReceipt {
	t.Helper()
	return authorizeTest(
		t, repository, principal, envelope.ProjectRef, identity.PermissionGoalsGet,
		envelope.Ref.String(), "authorization:mailbox:"+suffix, at,
	)
}

func completePostArtifactMailbox(
	t *testing.T,
	repository *Repository,
	envelope application.MailboxEnvelope,
	action application.ActionRecord,
	recipient identity.Principal,
	recipientExecution goal.ExecutionRef,
	clock *sqliteMembershipClock,
) {
	t.Helper()
	ctx := context.Background()
	claimAuthorization := authorizeMailboxTest(
		t, repository, recipient, envelope, "post-artifact-terminal-claim", clock.Now(),
	)
	claimFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationClaim, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		envelope.Ref, envelope.ParentWorkItemRef, recipientExecution, "", 0, "",
	)
	claim, changed, err := repository.ClaimMailbox(ctx, application.ClaimMailboxState{
		RequestRef: "request:post-artifact-terminal-claim", RequestFingerprint: claimFingerprint,
		AuthorizationReceipt: claimAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: recipientExecution, Token: "token:post-artifact-terminal-claim",
		LeaseDuration: time.Minute, RequestedAt: clock.Now(),
	})
	if err != nil || !changed {
		t.Fatalf("post-artifact terminal claim changed=%v claim=%+v err=%s",
			changed, claim, sqliteTestErrorChain(err))
	}

	clock.Advance(time.Second)
	deliveryAuthorization := authorizeMailboxTest(
		t, repository, recipient, envelope, "post-artifact-terminal-deliver", clock.Now(),
	)
	deliveryFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationDeliver, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		envelope.Ref, envelope.ParentWorkItemRef, recipientExecution, claim.Attempt.ClaimToken,
		claim.Attempt.Fence, "",
	)
	delivered, changed, err := repository.MarkMailboxDelivered(ctx, application.MarkMailboxDeliveredState{
		RequestRef: "request:post-artifact-terminal-deliver", RequestFingerprint: deliveryFingerprint,
		AuthorizationReceipt: deliveryAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: recipientExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence: claim.Attempt.Fence, DeliveryRef: "receipt:post-artifact-terminal-delivered",
		OperationAt: clock.Now(),
	})
	if err != nil || !changed || delivered.State != application.MailboxStateDelivered {
		t.Fatalf("post-artifact terminal deliver changed=%v state=%s err=%v", changed, delivered.State, err)
	}

	clock.Advance(time.Second)
	consumeAuthorization := authorizeMailboxTest(
		t, repository, recipient, envelope, "post-artifact-terminal-consume", clock.Now(),
	)
	consumeFingerprint := application.MailboxMutationFingerprint(
		application.MailboxMutationConsume, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		envelope.Ref, envelope.ParentWorkItemRef, recipientExecution, claim.Attempt.ClaimToken,
		claim.Attempt.Fence, "",
	)
	consumption := application.ActionConsumptionReceipt{
		ActionRef: action.Ref, Kind: application.ActionDeliverMailbox,
		GoalRef: envelope.GoalRef, WorkItemRef: envelope.ParentWorkItemRef,
		ExecutionRef: recipientExecution, MailboxMessageRef: envelope.Ref,
		PlanGeneration: action.PlanGeneration, WorkItemGeneration: action.WorkItemGeneration,
		Fence: claim.Attempt.Fence, DeliveryAttempt: claim.Attempt.Fence,
		ClaimToken: claim.Attempt.ClaimToken, WorkerRef: recipient.Ref.String(),
		Outcome: application.ActionConsumedCompleted, ConsumedAt: clock.Now(),
	}
	consumed, changed, err := repository.ConsumeMailbox(ctx, application.ConsumeMailboxState{
		RequestRef: "request:post-artifact-terminal-consume", RequestFingerprint: consumeFingerprint,
		AuthorizationReceipt: consumeAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: recipientExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence: claim.Attempt.Fence, ConsumptionRef: "receipt:post-artifact-terminal-consumed",
		ConsumptionReceipt: consumption, OperationAt: clock.Now(),
	})
	if err != nil || !changed || consumed.State != application.MailboxStateConsumed {
		t.Fatalf("post-artifact terminal consume changed=%v state=%s err=%v", changed, consumed.State, err)
	}

	clock.Advance(time.Second)
	ackAuthorization := authorizeMailboxTest(
		t, repository, recipient, envelope, "post-artifact-terminal-ack", clock.Now(),
	)
	beforeAck, err := repository.GetGoal(ctx, envelope.GoalRef)
	sqliteTestNoError(t, err)
	ackRef := "receipt:post-artifact-terminal-acknowledged"
	updated, err := beforeAck.Goal.ResolveChildHandoff(
		beforeAck.Goal.Revision(), envelope.ParentWorkItemRef, envelope.ChildWorkItemRef,
		envelope.Ref.String(), goal.ChildHandoffAcknowledged, ackRef, clock.Now(),
	)
	sqliteTestNoError(t, err)
	ack := application.MailboxAcknowledgement{
		Ref: ackRef, MessageRef: envelope.Ref, ActionRef: action.Ref,
		RequestRef: "request:post-artifact-terminal-ack",
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef,
		TargetPlanGeneration: envelope.TargetPlanGeneration,
		ParentWorkItemRef:    envelope.ParentWorkItemRef, ChildWorkItemRef: envelope.ChildWorkItemRef,
		Recipient: envelope.Recipient, Fence: claim.Attempt.Fence,
		Outcome:           application.MailboxOutcomeAcknowledged,
		EffectOrReworkRef: "effect:post-artifact-terminal-applied", AcknowledgedAt: clock.Now(),
		AuthorizationReceipt: ackAuthorization,
	}
	ack.RequestFingerprint = application.MailboxMutationFingerprint(
		application.MailboxMutationAcknowledge, recipient.Ref, envelope.ProjectRef, envelope.GoalRef,
		envelope.Ref, envelope.ParentWorkItemRef, recipientExecution, claim.Attempt.ClaimToken,
		claim.Attempt.Fence,
		strconv.FormatUint(uint64(beforeAck.Goal.Revision()), 10)+"\x00"+
			strconv.FormatUint(uint64(beforeAck.Goal.PlanGeneration()), 10)+"\x00"+
			ack.EffectOrReworkRef,
	)
	persisted, changed, err := repository.AcknowledgeMailbox(ctx, application.ResolveMailboxState{
		RequestRef: ack.RequestRef, RequestFingerprint: ack.RequestFingerprint,
		AuthorizationReceipt: ackAuthorization, PrincipalRef: recipient.Ref,
		ProjectRef: envelope.ProjectRef, GoalRef: envelope.GoalRef, MessageRef: envelope.Ref,
		RecipientExecutionRef: recipientExecution, ClaimToken: claim.Attempt.ClaimToken,
		Fence: claim.Attempt.Fence, ExpectedGoalRevision: beforeAck.Goal.Revision(),
		ExpectedPlanGeneration: beforeAck.Goal.PlanGeneration(), Goal: updated,
		Acknowledgement: ack, Events: []application.EventRecord{{
			Ref: "event:post-artifact-terminal-acknowledged", Kind: "mailbox.acknowledged",
			GoalRef: envelope.GoalRef, WorkItemRef: envelope.ParentWorkItemRef,
			ExecutionRef: recipientExecution, OccurredAt: clock.Now(),
		}}, OperationAt: clock.Now(),
	})
	if err != nil || !changed || persisted.Ref != ack.Ref {
		t.Fatalf("post-artifact terminal ack changed=%v receipt=%+v err=%v", changed, persisted, err)
	}
}

func openMailboxTestRepository(
	t *testing.T,
	path string,
	clock *sqliteMembershipClock,
) *Repository {
	t.Helper()
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8, Now: clock.Now,
	})
	sqliteTestNoError(t, err)
	return repository
}

func restartMailboxTestRepository(
	t *testing.T,
	repository *Repository,
	path string,
	clock *sqliteMembershipClock,
) *Repository {
	t.Helper()
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	return openMailboxTestRepository(t, path, clock)
}

func assertMailboxState(
	t *testing.T,
	repository *Repository,
	envelope application.MailboxEnvelope,
	want application.MailboxState,
	attempts int,
) application.MailboxRecord {
	t.Helper()
	record, err := repository.GetMailbox(
		context.Background(), envelope.ProjectRef, envelope.GoalRef, envelope.Ref, envelope.Recipient,
	)
	if err != nil || record.State != want || len(record.Attempts) != attempts ||
		!reflect.DeepEqual(record.Envelope.ArtifactRefs, envelope.ArtifactRefs) {
		t.Fatalf("mailbox restart state=%s/%s attempts=%d/%d err=%v", record.State, want, len(record.Attempts), attempts, err)
	}
	return record
}
