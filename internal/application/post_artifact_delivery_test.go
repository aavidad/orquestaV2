package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type recordingPostArtifactMailbox struct {
	requests []PostArtifactMailboxAdmissionRequest
}

func (mailbox *recordingPostArtifactMailbox) AdmitPostArtifactMailbox(
	_ context.Context,
	request PostArtifactMailboxAdmissionRequest,
) (PostArtifactMailboxAdmissionReceipt, error) {
	mailbox.requests = append(mailbox.requests, request)
	messageRef, err := NewMailboxMessageRef("mailbox-message:post-artifact-test")
	if err != nil {
		return PostArtifactMailboxAdmissionReceipt{}, err
	}
	return PostArtifactMailboxAdmissionReceipt{
		GoalRef: request.GoalRef, MessageRef: messageRef,
		AdmissionRef: "mailbox-admission:post-artifact-test",
	}, nil
}

type postArtifactDeliveryTestSystem struct {
	clock           *mutableClock
	repository      *memoryRepository
	orchestrator    *Orchestrator
	sessionBroker   *testExecutionSessionBroker
	mailbox         *recordingPostArtifactMailbox
	aggregate       goal.Goal
	action          ActionRecord
	parentRef       goal.WorkItemRef
	parentExecution goal.ExecutionRef
	childRef        goal.WorkItemRef
	childExecution  ExecutionRecord
}

func TestPostArtifactMailboxWaitsForPendingParentAndAdmitsExactlyOnce(t *testing.T) {
	ctx := context.Background()
	system := newPostArtifactDeliveryTestSystem(t)
	sourceSessionRef := system.childExecution.ExecutionSessionRef

	first, err := system.orchestrator.ProcessNext(ctx, "worker:post-artifact-pending")
	if err != nil || !first.Processed || first.Action != ActionAdmitMailbox {
		t.Fatalf("first process=%+v err=%v", first, err)
	}
	system.repository.mu.Lock()
	requeued, stillPending := system.repository.actions[system.action.Ref]
	system.repository.mu.Unlock()
	afterRequeue, err := system.repository.GetGoal(ctx, system.aggregate.Ref())
	if err != nil {
		t.Fatal(err)
	}
	sourceAfterRequeue, found := executionByRef(afterRequeue.Executions, system.childExecution.Ref)
	if !stillPending || requeued.token != "" || requeued.deliveryAttempt != 1 || requeued.fence != 1 ||
		len(afterRequeue.ConsumptionReceipts) != 0 || len(system.repository.events) != 0 ||
		len(system.sessionBroker.requests) != 0 || len(system.mailbox.requests) != 0 ||
		!found || !reflect.DeepEqual(sourceAfterRequeue, system.childExecution) ||
		sourceAfterRequeue.ExecutionSessionRef != sourceSessionRef {
		t.Fatalf(
			"pending parent consumed or altered delivery: pending=%v action=%+v receipts=%+v events=%d sessions=%d admissions=%d source=%+v",
			stillPending, requeued, afterRequeue.ConsumptionReceipts, len(system.repository.events),
			len(system.sessionBroker.requests), len(system.mailbox.requests), sourceAfterRequeue,
		)
	}

	system.clock.Advance(time.Second)
	parent, _ := system.aggregate.WorkItem(system.parentRef)
	started, err := system.aggregate.StartWorkItem(
		system.aggregate.Revision(), parent.Revision(), system.parentRef,
		system.parentExecution, system.clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	parentExecution := postArtifactExecution(
		started, system.parentRef, system.parentExecution, ExecutionRunning,
		system.clock.Now(), time.Time{},
	)
	system.aggregate = started
	system.repository.mu.Lock()
	record := system.repository.records[started.Ref()]
	record.Goal = started
	record.Executions = append(record.Executions, parentExecution)
	system.repository.records[started.Ref()] = record
	system.repository.mu.Unlock()

	second, err := system.orchestrator.ProcessNext(ctx, "worker:post-artifact-running")
	if err != nil || !second.Processed || second.Action != ActionAdmitMailbox {
		t.Fatalf("second process=%+v err=%v", second, err)
	}
	afterAdmission, err := system.repository.GetGoal(ctx, started.Ref())
	if err != nil {
		t.Fatal(err)
	}
	system.repository.mu.Lock()
	_, stillPending = system.repository.actions[system.action.Ref]
	system.repository.mu.Unlock()
	if stillPending || len(system.sessionBroker.requests) != 1 || len(system.mailbox.requests) != 1 ||
		len(afterAdmission.ConsumptionReceipts) != 1 {
		t.Fatalf(
			"admission multiplicity: pending=%v sessions=%d admissions=%d receipts=%+v",
			stillPending, len(system.sessionBroker.requests), len(system.mailbox.requests),
			afterAdmission.ConsumptionReceipts,
		)
	}
	receipt := afterAdmission.ConsumptionReceipts[0]
	if receipt.ActionRef != system.action.Ref || receipt.Outcome != ActionConsumedCompleted ||
		receipt.DeliveryAttempt != 2 || receipt.Fence != 2 || receipt.ErrorCode != "" {
		t.Fatalf("admission receipt=%+v", receipt)
	}
	request := system.mailbox.requests[0]
	wantSession := ExecutionSessionRequest(started, system.childExecution)
	if request.Session != wantSession || system.sessionBroker.requests[0] != wantSession ||
		request.ParentWorkItemRef != system.parentRef || request.ChildWorkItemRef != system.childRef ||
		request.RecipientExecutionRef != system.parentExecution ||
		request.GoalRef != started.Ref() || request.ExpectedPlanGeneration != started.PlanGeneration() {
		t.Fatalf("admission request=%+v session=%+v", request, system.sessionBroker.requests[0])
	}
	sourceAfterAdmission, found := executionByRef(afterAdmission.Executions, system.childExecution.Ref)
	if !found || sourceAfterAdmission.ExecutionSessionRef != sourceSessionRef {
		t.Fatalf("source session changed after admission: %+v", sourceAfterAdmission)
	}
	if third, thirdErr := system.orchestrator.ProcessNext(ctx, "worker:post-artifact-replay"); thirdErr != nil ||
		third.Processed || len(system.mailbox.requests) != 1 {
		t.Fatalf("admission replay=%+v admissions=%d err=%v", third, len(system.mailbox.requests), thirdErr)
	}
}

func TestPostArtifactMailboxQuarantinesTerminalParent(t *testing.T) {
	ctx := context.Background()
	system := newPostArtifactDeliveryTestSystem(t)
	parent, _ := system.aggregate.WorkItem(system.parentRef)
	started, err := system.aggregate.StartWorkItem(
		system.aggregate.Revision(), parent.Revision(), system.parentRef,
		system.parentExecution, system.clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	parent, _ = started.WorkItem(system.parentRef)
	failed, err := started.FailWorkItem(
		started.Revision(), parent.Revision(), system.parentRef, system.clock.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}
	parentExecution := postArtifactExecution(
		failed, system.parentRef, system.parentExecution, ExecutionFailed,
		system.clock.Now(), system.clock.Now(),
	)
	parentExecution.FailureCode = "agent.parent_failed"
	system.repository.mu.Lock()
	record := system.repository.records[failed.Ref()]
	record.Goal = failed
	record.Executions = append(record.Executions, parentExecution)
	system.repository.records[failed.Ref()] = record
	system.repository.mu.Unlock()

	result, err := system.orchestrator.ProcessNext(ctx, "worker:post-artifact-terminal")
	if err == nil || err.Error() != "application.post_artifact_mailbox_invalid" ||
		!result.Processed || result.Action != ActionAdmitMailbox {
		t.Fatalf("process terminal parent=%+v err=%v", result, err)
	}
	after, err := system.repository.GetGoal(ctx, failed.Ref())
	if err != nil {
		t.Fatal(err)
	}
	system.repository.mu.Lock()
	_, stillPending := system.repository.actions[system.action.Ref]
	system.repository.mu.Unlock()
	if stillPending || len(system.sessionBroker.requests) != 0 || len(system.mailbox.requests) != 0 ||
		len(after.ConsumptionReceipts) != 1 ||
		after.ConsumptionReceipts[0].Outcome != ActionConsumedQuarantined ||
		after.ConsumptionReceipts[0].ErrorCode != "application.post_artifact_mailbox_invalid" {
		t.Fatalf(
			"terminal parent not quarantined: pending=%v sessions=%d admissions=%d receipts=%+v",
			stillPending, len(system.sessionBroker.requests), len(system.mailbox.requests),
			after.ConsumptionReceipts,
		)
	}
}

func newPostArtifactDeliveryTestSystem(t *testing.T) *postArtifactDeliveryTestSystem {
	t.Helper()
	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: now}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	sessionBroker := &testExecutionSessionBroker{at: now}
	mailbox := &recordingPostArtifactMailbox{}
	orchestrator.executionSessions = sessionBroker
	orchestrator.postArtifactMailbox = mailbox

	actor := mailboxMustRef(t, "actor:post-artifact-test", goal.NewActorRef)
	project := mailboxMustRef(t, "project:post-artifact-test", goal.NewProjectRef)
	requestedBy := mailboxMustRef(t, "principal:post-artifact-test", identity.NewPrincipalRef)
	intent, err := goal.NewIntentManifest(goal.IntentManifestInput{
		Ref:   mailboxMustRef(t, "intent:post-artifact-test", goal.NewIntentRef),
		Actor: actor, Project: project, Statement: "deliver a child artifact to its parent",
		SubmittedAt: now.Add(-5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	spec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
		Ref:    mailboxMustRef(t, "app-spec:post-artifact-test", goal.NewAppSpecRef),
		Intent: intent, Objective: "coordinate the parent after child completion",
		Reason: "operator.confirmed", ConfirmedBy: actor, ConfirmedAt: now.Add(-4 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err := goal.NewGoal(
		mailboxMustRef(t, "goal:post-artifact-test", goal.NewGoalRef),
		spec, now.Add(-3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	phase, err := goal.NewPhaseInstance(goal.DefaultPhaseKey())
	if err != nil {
		t.Fatal(err)
	}
	parentRef := mailboxMustRef(t, "work-item:post-artifact-parent", goal.NewWorkItemRef)
	parent, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: parentRef, Goal: aggregate.Ref(), Actor: actor, Project: project,
		Objective: "integrate the child artifact", Phase: phase.Key(),
		CreatedAt: now.Add(-2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	childRef := mailboxMustRef(t, "work-item:post-artifact-child", goal.NewWorkItemRef)
	child, err := goal.NewWorkItem(goal.NewWorkItemInput{
		Ref: childRef, Goal: aggregate.Ref(), Actor: actor, Project: project,
		Objective: "produce the child artifact", Phase: phase.Key(), Parent: parentRef,
		HandoffRequired: true, CreatedAt: now.Add(-2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := goal.NewPlan(goal.PlanInput{
		Generation: 1, Phases: []goal.PhaseInstance{phase},
		WorkItems: []goal.WorkItem{parent, child},
	})
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = aggregate.ApplyPlan(aggregate.Revision(), plan)
	if err != nil {
		t.Fatal(err)
	}
	aggregate, err = aggregate.Start(aggregate.Revision(), now.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	childExecutionRef := mailboxMustRef(t, "execution:post-artifact-child", goal.NewExecutionRef)
	child, _ = aggregate.WorkItem(childRef)
	aggregate, err = aggregate.StartWorkItem(
		aggregate.Revision(), child.Revision(), childRef,
		childExecutionRef, now.Add(-50*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	artifactRef := mailboxMustRef(t, "artifact:post-artifact-child", goal.NewArtifactRef)
	child, _ = aggregate.WorkItem(childRef)
	aggregate, err = aggregate.SucceedWorkItem(
		aggregate.Revision(), child.Revision(), childRef,
		[]goal.ArtifactRef{artifactRef},
		[]goal.AttestationRef{
			mailboxMustRef(t, "attestation:post-artifact-child", goal.NewAttestationRef),
		},
		now.Add(-40*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	childExecution := postArtifactExecution(
		aggregate, childRef, childExecutionRef, ExecutionSucceeded,
		now.Add(-50*time.Second), now.Add(-40*time.Second),
	)
	sourceAuthority, err := DeriveExecutionSessionAuthority(
		ExecutionSessionRequest(aggregate, childExecution), "execution_token",
	)
	if err != nil {
		t.Fatal(err)
	}
	childExecution.ExecutionSessionRef = sourceAuthority.SessionRef
	child, _ = aggregate.WorkItem(childRef)
	action, err := postArtifactMailboxAction(aggregate, child, childExecution, now)
	if err != nil || action == nil {
		t.Fatalf("post artifact action=%+v err=%v", action, err)
	}
	repository.records[aggregate.Ref()] = GoalRecord{
		RequestRef: "request:post-artifact-test", RequestFingerprint: "fixture",
		RequestedBy: requestedBy, Goal: aggregate, Executions: []ExecutionRecord{childExecution},
	}
	repository.actions[action.Ref] = memoryAction{record: *action}
	return &postArtifactDeliveryTestSystem{
		clock: clock, repository: repository, orchestrator: orchestrator,
		sessionBroker: sessionBroker, mailbox: mailbox, aggregate: aggregate, action: *action,
		parentRef: parentRef,
		parentExecution: mailboxMustRef(
			t, "execution:post-artifact-parent", goal.NewExecutionRef,
		),
		childRef: childRef, childExecution: childExecution,
	}
}

func postArtifactExecution(
	aggregate goal.Goal,
	workItemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
	state ExecutionState,
	startedAt time.Time,
	finishedAt time.Time,
) ExecutionRecord {
	return ExecutionRecord{
		Ref: executionRef, GoalRef: aggregate.Ref(), WorkItemRef: workItemRef,
		AttemptNo: 1, MaxExecutionAttempts: 3, PlanGeneration: aggregate.PlanGeneration(),
		AppSpecGeneration: aggregate.AppSpec().Generation(), SpecHash: aggregate.SpecHash(),
		State: state, ArtifactMediaType: "text/plain",
		IdempotencyKey: "execution:" + executionRef.String(), MaxOutputBytes: 1 << 20,
		ProviderRef: "provider:test", ModelRef: "model:test",
		AgentRef: "agent:test", ExternalRef: "external:" + executionRef.String(),
		CreatedAt: startedAt, StartedAt: startedAt, ProviderAcceptedAt: startedAt,
		DeadlineAt: startedAt.Add(time.Hour), FinishedAt: finishedAt,
	}
}

var _ ports.ExecutionSessionBroker = (*testExecutionSessionBroker)(nil)
var _ PostArtifactMailboxAdmitter = (*recordingPostArtifactMailbox)(nil)
