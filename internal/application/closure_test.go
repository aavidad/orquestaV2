package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

type temporaryAgentTestError struct{}

func (temporaryAgentTestError) Error() string   { return "test.agent_busy" }
func (temporaryAgentTestError) Temporary() bool { return true }

type invalidArtifactStore struct{}

func (invalidArtifactStore) Put(context.Context, ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	return ports.StoredArtifact{MediaType: "text/plain", Size: 1}, nil
}

func (invalidArtifactStore) Get(context.Context, goal.ArtifactRef, int64) (ports.ArtifactContent, error) {
	return ports.ArtifactContent{}, errors.New("test.not_used")
}

func TestExplicitAgentFailureCreatesReplacementBeforeClosure(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 21, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentFailed, ErrorCode: "provider.execution_failed",
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:failure", Statement: "tarea fallida", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("launch: %v", err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("observe failure: %v", err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if record.Goal.State() != goal.GoalStateRunning || len(record.Executions) != 2 ||
		record.Executions[0].State != ExecutionFailed || record.Executions[1].State != ExecutionQueued {
		t.Fatalf("failure did not create replacement: goal=%s executions=%+v", record.Goal.State(), record.Executions)
	}
	if record.Executions[0].FailureCode != "provider.execution_failed" ||
		record.Executions[1].AttemptNo != 2 || record.Executions[1].ReplacesExecutionRef != record.Executions[0].Ref {
		t.Fatalf("replacement chain lost: %+v", record.Executions)
	}
	if len(record.Artifacts) != 0 || len(record.Attestations) != 0 {
		t.Fatalf("failed work received false evidence")
	}
	agent.mu.Lock()
	agent.observations = append(agent.observations, ports.AgentObservation{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("replacement succeeded"),
	})
	agent.mu.Unlock()
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("replacement launch: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("replacement observe: %v", err)
	}
	record, err = repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateSucceeded || record.Executions[1].State != ExecutionSucceeded {
		t.Fatalf("replacement did not close once: record=%+v err=%v", record, err)
	}
}

func TestControlsExecutionExhaustionInterruptsWithoutClosingGoal(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 21, 15, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{
		{Status: ports.AgentFailed, ErrorCode: "provider.failed.1"},
		{Status: ports.AgentFailed, ErrorCode: "provider.failed.2"},
		{Status: ports.AgentFailed, ErrorCode: "provider.failed.3"},
	}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:attempt-exhaustion",
		Statement:  "bounded provider retries", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("launch attempt 1: %v", err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
			t.Fatalf("observe attempt %d: %v", attempt, err)
		}
		if attempt < 3 {
			clock.Advance(time.Duration(1<<(attempt-1)) * time.Second)
			if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
				t.Fatalf("launch attempt %d: %v", attempt+1, err)
			}
		}
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	items := record.Goal.WorkItems()
	if len(items) != 1 {
		t.Fatalf("attempt exhaustion item count=%d", len(items))
	}
	item := items[0]
	cause, interrupted := item.InterruptCause()
	if err != nil || record.Goal.State() != goal.GoalStateRunning || len(record.Executions) != 3 ||
		item.State() != goal.WorkItemStateInterrupted || !interrupted ||
		cause != goal.WorkItemInterruptExecutionFailed {
		t.Fatalf("attempt exhaustion did not remain replanable: record=%+v err=%v", record, err)
	}
	for index, execution := range record.Executions {
		if execution.AttemptNo != uint64(index+1) || execution.State != ExecutionFailed {
			t.Fatalf("attempt chain[%d] = %+v", index, execution)
		}
	}
	if record.Executions[2].FailureCode != "provider.failed.3" || len(record.Artifacts) != 0 || len(record.Attestations) != 0 {
		t.Fatalf("exhaustion evidence invalid: %+v", record)
	}
}

func TestInvalidArtifactAdapterCannotAccreditSuccessfulGoal(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 21, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("real content"),
	}}}
	orchestrator, err := New(Dependencies{
		State: repository, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: invalidArtifactStore{},
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, AgentCapabilities: testAgentCapabilities(), ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute,
		MaxChildrenPerParent:  6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:invalid-artifact-adapter", Statement: "must have real evidence", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("observe: %v", err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateFailed ||
		onlyExecution(t, record).FailureCode != "artifact.stored_ref_mismatch" ||
		len(record.Artifacts) != 0 || len(record.Attestations) != 0 {
		t.Fatalf("invalid adapter produced evidence: record=%+v err=%v", record, err)
	}
}

func TestLaunchInfrastructureFailureCreatesReplaceableAttempt(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 22, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, launchErr: definitelyUnappliedPermanentError{"missing executable"}, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("recovered"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:launch-failure", Statement: "tarea", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("process launch failure: %v", err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if record.Goal.State() != goal.GoalStateRunning || len(record.Executions) != 2 ||
		record.Executions[0].FailureCode != "agent.launch_failed" || record.Executions[1].AttemptNo != 2 {
		t.Fatalf("unexpected replacement: goal=%s executions=%+v", record.Goal.State(), record.Executions)
	}
	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("replacement launch: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("replacement observe: %v", err)
	}
}

func TestTemporaryLaunchFailureRequeuesWithoutClosingGoal(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 22, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, launchErr: temporaryAgentTestError{}, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("after capacity"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:temporary-launch", Statement: "wait safely", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("temporary launch: %v", err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateRunning || onlyExecution(t, record).State != ExecutionDispatching {
		t.Fatalf("temporary failure changed lifecycle: record=%+v err=%v", record, err)
	}
	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()
	// Queue time does not consume the provider execution timeout. A saturated
	// adapter may recover after the original submission deadline.
	clock.Advance(2 * time.Hour)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("retry launch: %v", err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("observe retry: %v", err)
	}
	record, err = repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateSucceeded || agent.launches != 2 {
		t.Fatalf("retry did not close exactly once: record=%+v launches=%d err=%v", record, agent.launches, err)
	}
}

func TestTemporaryLaunchCapacityWaitDoesNotConsumeExecutionAttemptBudget(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 23, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, launchErr: temporaryAgentTestError{}, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("capacity recovered"),
	}}}
	orchestrator, err := New(Dependencies{
		State: repository, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: newMemoryArtifactStore(),
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, AgentCapabilities: testAgentCapabilities(), ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute,
		MaxChildrenPerParent:  6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:capacity-boundary", Statement: "bounded capacity wait", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("first capacity response: %v", err)
	}
	clock.Advance(time.Second)
	for attempt := 0; attempt < 3; attempt++ {
		if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
			t.Fatalf("capacity response %d: %v", attempt+2, err)
		}
		clock.Advance(time.Second)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateRunning ||
		onlyExecution(t, record).State != ExecutionDispatching || agent.launches != 4 {
		t.Fatalf("capacity wait consumed execution budget: record=%+v launches=%d err=%v", record, agent.launches, err)
	}
	agent.mu.Lock()
	agent.launchErr = nil
	agent.mu.Unlock()
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("launch after capacity recovery: %v", err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("observe after capacity recovery: %v", err)
	}
	record, err = repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateSucceeded || agent.launches != 5 {
		t.Fatalf("capacity recovery did not close: record=%+v launches=%d err=%v", record, agent.launches, err)
	}
}

func TestPendingObservationHasDurableAttemptBoundary(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	repository.now = clock.Now
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{
		{Status: ports.AgentPending}, {Status: ports.AgentRunning},
	}}
	artifacts := newMemoryArtifactStore()
	orchestrator, err := New(Dependencies{
		State: repository, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, AgentCapabilities: testAgentCapabilities(), ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute,
		MaxChildrenPerParent:  6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:bounded", Statement: "bounded", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("first pending: %v", err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("terminal attempt: %v", err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateRunning || len(record.Executions) != 1 ||
		record.Executions[0].State != ExecutionRunning {
		t.Fatalf("delivery retry consumed execution attempt: record=%+v err=%v", record, err)
	}
	clock.Advance(time.Hour)
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("expired execution replacement: %v", err)
	}
	record, err = repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateRunning || len(record.Executions) != 2 ||
		record.Executions[0].FailureCode != "application.execution_expired" || record.Executions[1].AttemptNo != 2 {
		t.Fatalf("deadline did not create replacement: record=%+v err=%v", record, err)
	}
}
