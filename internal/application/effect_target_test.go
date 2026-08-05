package application

import (
	"context"
	"strconv"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestTamperedStopTargetNeverInvokesAdapter(t *testing.T) {
	system := newControlTestSystem(t, nil)
	system.launch(t)
	record := system.record(t)
	item := record.Goal.WorkItems()[0]
	execution := mustBoundExecution(t, record, item.Ref())
	requested, err := system.orchestrator.Control(context.Background(), system.access, system.request(
		t, "control:tampered-stop", ControlStop, ControlTargetExecution, item.Ref(), execution.Ref,
	))
	if err != nil || !requested.Created {
		t.Fatalf("request stop: result=%+v err=%v", requested, err)
	}

	system.repository.mu.Lock()
	stored := system.repository.records[record.Goal.Ref()]
	control, found := controlByRef(stored.Controls, requested.Control.Ref)
	if !found {
		system.repository.mu.Unlock()
		t.Fatal("stop control missing")
	}
	control.Target, control.Mode = ControlTargetGoal, ports.AgentStopForced
	stored.Controls = replaceControl(stored.Controls, control)
	system.repository.records[record.Goal.Ref()] = stored
	system.repository.mu.Unlock()

	processed, err := system.orchestrator.ProcessNext(context.Background(), "worker:tampered-stop")
	if !processed.Processed || processed.Action != ActionStopAgent || err == nil ||
		err.Error() != "application.effect_target_mismatch" || system.stopCount() != 0 {
		t.Fatalf("tampered target crossed adapter: result=%+v stops=%d err=%v", processed, system.stopCount(), err)
	}
}

func TestLaunchTargetDigestBindsExactExecution(t *testing.T) {
	actor, project := testScope(t)
	goalRef, _ := goal.NewGoalRef("goal:target-digest")
	itemRef, _ := goal.NewWorkItemRef("work-item:target-digest")
	executionRef, _ := goal.NewExecutionRef("execution:target-digest")
	request := ports.AgentLaunchRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: itemRef,
		PlanGeneration: 1, AppSpecGeneration: 1, ExecutionAttempt: 1,
		SpecHash: "sha256:" + effectAdmissionFingerprint("target-spec"), ActorRef: actor,
		ProjectRef: project, IdempotencyKey: "launch:target-digest",
	}
	want := authorLaunchTargetDigest(request)
	request.ExecutionAttempt++
	if got := authorLaunchTargetDigest(request); got == want {
		t.Fatal("execution attempt retained launch target digest")
	}
}

func TestDeterministicExecutionSessionRefPreservesPreV22TargetDigest(t *testing.T) {
	actor, project := testScope(t)
	goalRef, _ := goal.NewGoalRef("goal:target-session")
	itemRef, _ := goal.NewWorkItemRef("work-item:target-session")
	executionRef, _ := goal.NewExecutionRef("execution:target-session")
	request := ports.AgentLaunchRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: itemRef,
		PlanGeneration: 1, AppSpecGeneration: 1, ExecutionAttempt: 1,
		SpecHash: testDigest("target-session"), ActorRef: actor, ProjectRef: project,
		IdempotencyKey: "launch:target-session",
	}
	legacy := authorLaunchTargetDigest(request)
	session, _ := ports.NewExecutionSessionRef("execution-session:sha256:" + testDigest("session-ref"))
	request.SessionRef = session
	if got := authorLaunchTargetDigest(request); got != legacy {
		t.Fatalf("deterministic SessionRef invalidated durable target: got=%s want=%s", got, legacy)
	}
}

func TestLaunchTargetDigestPreservesLegacyDomainsAndBindsPresentEgressTuple(t *testing.T) {
	actor, project := testScope(t)
	goalRef, _ := goal.NewGoalRef("goal:target-egress")
	itemRef, _ := goal.NewWorkItemRef("work-item:target-egress")
	executionRef, _ := goal.NewExecutionRef("execution:target-egress")
	request := ports.AgentLaunchRequest{
		ExecutionRef: executionRef, GoalRef: goalRef, WorkItemRef: itemRef,
		PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 4,
		SpecHash: testDigest("target-egress"), ActorRef: actor, ProjectRef: project,
		IdempotencyKey: "launch:target-egress", Objective: "review exact output",
		PhaseRef: "phase-instance:review", PhaseKey: "phase:review", PhaseTemplateRef: "phase-template:review",
		RoleKey: "role:reviewer", OutputContract: string(goal.OutputContractAttestation),
		ArtifactMediaType: "application/json", MaxOutputBytes: 1024,
	}
	wantAuthorLegacy := effectAdmissionFingerprint(
		"target:launch:v1", project.String(), goalRef.String(), itemRef.String(), executionRef.String(),
		"2", "3", "4", request.SpecHash, actor.String(), "", request.IdempotencyKey,
	)
	if got := authorLaunchTargetDigest(request); got != wantAuthorLegacy {
		t.Fatalf("author legacy digest=%s want=%s", got, wantAuthorLegacy)
	}
	reviewerFields := []string{
		"target:launch:v2", project.String(), goalRef.String(), itemRef.String(), executionRef.String(),
		"2", "3", "4", request.SpecHash, actor.String(), "", request.IdempotencyKey,
		request.Objective, request.PhaseRef, request.PhaseKey, request.PhaseTemplateRef, request.RoleKey,
		request.OutputContract, request.ArtifactMediaType, strconv.FormatInt(request.MaxOutputBytes, 10),
		string(request.SecurityCriticality), string(request.ReasoningEffort), request.BudgetDemand.Ref,
	}
	wantReviewerLegacy := effectAdmissionFingerprint(reviewerFields...)
	if got := reviewerLaunchTargetDigest(request); got != wantReviewerLegacy {
		t.Fatalf("reviewer legacy digest=%s want=%s", got, wantReviewerLegacy)
	}

	policy := testEgressPolicyAuthority(t, "egress-policy:target", `{"destinations":["example.org"]}`)
	request.EgressAuthority = ports.AgentLaunchEgressAuthority{
		PolicyRef: policy.PolicyRef.String(), PayloadSHA256: policy.PayloadSHA256, CanonicalPayload: []byte(policy.CanonicalPayload),
	}
	authorWithEgress, reviewerWithEgress := authorLaunchTargetDigest(request), reviewerLaunchTargetDigest(request)
	if authorWithEgress == wantAuthorLegacy || reviewerWithEgress == wantReviewerLegacy ||
		authorWithEgress == reviewerWithEgress {
		t.Fatalf("egress domain not isolated: author=%s reviewer=%s", authorWithEgress, reviewerWithEgress)
	}
	request.EgressAuthority.CanonicalPayload = append(request.EgressAuthority.CanonicalPayload, ' ')
	if authorLaunchTargetDigest(request) == authorWithEgress || reviewerLaunchTargetDigest(request) == reviewerWithEgress {
		t.Fatal("canonical payload not bound into egress launch target")
	}
}
