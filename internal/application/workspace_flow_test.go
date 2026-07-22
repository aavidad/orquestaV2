package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestIntegrateChangeReplaysExactAdmissionBeforeAndAfterCompletion(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("workspace output"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:workspace-integration-replay", Statement: "commit then explicitly integrate", Confirm: true,
		Plan: workspaceWritePlan(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []ActionKind{
		ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent, ActionCommitChange, ActionAttestTest,
	} {
		processed, processErr := orchestrator.ProcessNext(ctx, "worker:integration-replay")
		if processErr != nil || processed.Action != want {
			t.Fatalf("process=%+v want=%s err=%v", processed, want, processErr)
		}
	}
	for _, want := range []ActionKind{ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent} {
		processed, processErr := orchestrator.ProcessNext(ctx, "worker:integration-replay")
		if processErr != nil || processed.Action != want {
			t.Fatalf("review process=%+v want=%s err=%v", processed, want, processErr)
		}
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if record.Goal.State() != goal.GoalStateRunning || len(record.ChangeSets) != 1 ||
		record.Executions[0].State != ExecutionAwaitingIntegration {
		t.Fatalf("agent output closed lifecycle early: goal=%s changes=%d execution=%s",
			record.Goal.State(), len(record.ChangeSets), record.Executions[0].State)
	}
	request := IntegrateChangeRequest{
		RequestRef: "request:integrate-replay", GoalRef: record.Goal.Ref(),
		ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
	}
	created, err := orchestrator.IntegrateChange(ctx, access, request)
	if err != nil || !created.Created {
		t.Fatalf("admit integration=%+v err=%v", created, err)
	}
	replay, err := orchestrator.IntegrateChange(ctx, access, request)
	if err != nil || replay.Created || !reflect.DeepEqual(replay.Action, created.Action) {
		t.Fatalf("pending replay=%+v err=%v want=%+v", replay, err, created)
	}
	processed, err := orchestrator.ProcessNext(ctx, "worker:integration-replay")
	if err != nil || processed.Action != ActionIntegrateChange {
		t.Fatalf("integrate process=%+v err=%v", processed, err)
	}
	replay, err = orchestrator.IntegrateChange(ctx, access, request)
	if err != nil || replay.Created || !reflect.DeepEqual(replay.Action, created.Action) {
		t.Fatalf("completed replay=%+v err=%v want=%+v", replay, err, created)
	}
	closed, err := repository.GetGoal(ctx, record.Goal.Ref())
	if err != nil || closed.Goal.State() != goal.GoalStateSucceeded ||
		len(closed.IntegrationReceipts) != 1 || closed.IntegrationReceipts[0].Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("closed lifecycle=%+v err=%v", closed, err)
	}
}

func TestIntegrateChangeRejectsMalformedTargetBeforeAuthorizationOrAdmission(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 21, 10, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	accessRepository := newMemoryAccessRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("workspace output"),
	}}}
	orchestrator, _ := newTestOrchestratorWithAccess(t, repository, accessRepository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:workspace-invalid-target", Statement: "commit before invalid integration", Confirm: true,
		Plan: workspaceWritePlan(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []ActionKind{ActionPrepareWorkspace, ActionLaunchAgent, ActionObserveAgent, ActionCommitChange} {
		if processed, processErr := orchestrator.ProcessNext(ctx, "worker:invalid-target"); processErr != nil || processed.Action != want {
			t.Fatalf("process=%+v want=%s err=%v", processed, want, processErr)
		}
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	accessRepository.mu.Lock()
	authorizationsBefore := len(accessRepository.authorizations)
	accessRepository.mu.Unlock()
	repository.mu.Lock()
	admissionsBefore, actionsBefore := len(repository.integrationAdmits), len(repository.actions)
	repository.mu.Unlock()

	_, err = orchestrator.IntegrateChange(ctx, access, IntegrateChangeRequest{
		RequestRef: "request:invalid-target", GoalRef: record.Goal.Ref(),
		ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: "not-an-object-id",
	})
	if err == nil || err.Error() != "application.integrate_change_request_invalid" {
		t.Fatalf("malformed target error=%v", err)
	}
	accessRepository.mu.Lock()
	authorizationsAfter := len(accessRepository.authorizations)
	accessRepository.mu.Unlock()
	repository.mu.Lock()
	admissionsAfter, actionsAfter := len(repository.integrationAdmits), len(repository.actions)
	repository.mu.Unlock()
	if authorizationsAfter != authorizationsBefore || admissionsAfter != admissionsBefore || actionsAfter != actionsBefore {
		t.Fatalf("malformed target mutated state: auth=%d/%d admissions=%d/%d actions=%d/%d",
			authorizationsBefore, authorizationsAfter, admissionsBefore, admissionsAfter, actionsBefore, actionsAfter)
	}
}

func TestLaunchUsesExactOpaqueWorkspaceBinding(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	workspace := &scriptedWorkspaceManager{}
	orchestrator.workspaceManager = workspace
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:workspace-launch", Statement: "write in isolated workspace", Confirm: true,
		Plan: workspaceWritePlan(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := orchestrator.ProcessNext(ctx, "worker:workspace-prepare"); err != nil || result.Action != ActionPrepareWorkspace {
		t.Fatalf("prepare result=%+v err=%v", result, err)
	}
	if result, err := orchestrator.ProcessNext(ctx, "worker:workspace-launch"); err != nil || result.Action != ActionLaunchAgent {
		t.Fatalf("launch result=%+v err=%v", result, err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	execution := onlyExecution(t, record)
	if len(record.WorkspaceBindings) != 1 || record.WorkspaceBindings[0].Ref != execution.ExecutionWorkspaceRef {
		t.Fatalf("binding/execution mismatch: bindings=%+v execution=%+v", record.WorkspaceBindings, execution)
	}
	agent.mu.Lock()
	requests := append([]ports.AgentLaunchRequest(nil), agent.launchRequests...)
	agent.mu.Unlock()
	if len(requests) != 1 || requests[0].ExecutionWorkspaceRef != execution.ExecutionWorkspaceRef ||
		requests[0].ExecutionWorkspaceRef.String() != record.WorkspaceBindings[0].Ref.String() {
		t.Fatalf("launch did not receive exact opaque binding: %+v binding=%+v", requests, record.WorkspaceBindings[0])
	}
	workspace.mu.Lock()
	prepared := append([]ports.WorkspacePrepareRequest(nil), workspace.requests...)
	workspace.mu.Unlock()
	if len(prepared) != 1 || prepared[0].WorkspaceRef != execution.ExecutionWorkspaceRef {
		t.Fatalf("prepare did not receive exact workspace ref: %+v", prepared)
	}
}

func TestReplacementExecutionGetsDistinctWorkspace(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 21, 9, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, launchErr: definitelyUnappliedPermanentError{"replace"},
		launchErrorHook: func() { clock.Advance(time.Nanosecond) }}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	orchestrator.workspaceManager = &scriptedWorkspaceManager{}
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, accessForScope(t, actor, project), SubmitRequest{
		RequestRef: "request:workspace-replace", Statement: "replace isolated writer", Confirm: true,
		Plan: workspaceWritePlan(),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []ActionKind{ActionPrepareWorkspace, ActionLaunchAgent} {
		result, processErr := orchestrator.ProcessNext(ctx, "worker:workspace-replace")
		if processErr != nil && want != ActionLaunchAgent {
			t.Fatalf("%s err=%v", want, processErr)
		}
		if result.Action != want {
			t.Fatalf("action=%s want=%s err=%v", result.Action, want, processErr)
		}
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Executions) != 2 || len(record.WorkspaceBindings) != 1 {
		t.Fatalf("replacement facts executions=%d bindings=%d", len(record.Executions), len(record.WorkspaceBindings))
	}
	first, replacement := record.Executions[0], record.Executions[1]
	if replacement.ReplacesExecutionRef != first.Ref || replacement.ExecutionWorkspaceRef.String() == "" ||
		replacement.ExecutionWorkspaceRef == first.ExecutionWorkspaceRef {
		t.Fatalf("replacement reused workspace: first=%+v replacement=%+v", first, replacement)
	}
}

func workspaceWritePlan() *PlanSpec {
	return &PlanSpec{Phases: []PhaseSpec{{Ref: "phase-instance:workspace", Key: "phase:workspace", TemplateRef: "phase-template:workspace"}},
		WorkItems: []WorkItemSpec{{Key: "writer", Objective: "isolated writer", Phase: "phase:workspace", Role: "role:worker",
			WriteSet: []string{"internal/workspace"}, RequiredTests: requiredTestSpecs("required-test:workspace"), OutputContract: goal.OutputContractEvidenceBundle}}}
}
