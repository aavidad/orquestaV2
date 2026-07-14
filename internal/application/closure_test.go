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

func TestClosurePersistsExplicitAgentFailureWithoutFalseEvidence(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 21, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentFailed, ErrorCode: "provider.execution_failed",
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, SubmitRequest{
		RequestRef: "request:failure", ActorRef: actor, ProjectRef: project, Statement: "tarea fallida",
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
	if record.Goal.State() != goal.GoalStateFailed || onlyExecution(t, record).State != ExecutionFailed {
		t.Fatalf("failure not terminal: goal=%s execution=%s", record.Goal.State(), onlyExecution(t, record).State)
	}
	if onlyExecution(t, record).FailureCode != "provider.execution_failed" {
		t.Fatalf("failure code lost: %s", onlyExecution(t, record).FailureCode)
	}
	if len(record.Artifacts) != 0 || len(record.Attestations) != 0 {
		t.Fatalf("failed work received false evidence")
	}
}

func TestInvalidArtifactAdapterCannotAccreditSuccessfulGoal(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 21, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("real content"),
	}}}
	orchestrator, err := New(Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: invalidArtifactStore{},
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1024,
		MaxActionAttempts: 3, ClaimLease: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, SubmitRequest{
		RequestRef: "request:invalid-artifact-adapter", ActorRef: actor, ProjectRef: project, Statement: "must have real evidence",
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

func TestLaunchInfrastructureFailureClosesGoalDeterministically(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 22, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, launchErr: errors.New("missing executable")}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, SubmitRequest{
		RequestRef: "request:launch-failure", ActorRef: actor, ProjectRef: project, Statement: "tarea",
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
	if record.Goal.State() != goal.GoalStateFailed || onlyExecution(t, record).FailureCode != "agent.launch_failed" {
		t.Fatalf("unexpected terminal failure: goal=%s code=%s", record.Goal.State(), onlyExecution(t, record).FailureCode)
	}
}

func TestTemporaryLaunchFailureRequeuesWithoutClosingGoal(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 22, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, launchErr: temporaryAgentTestError{}, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("after capacity"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, SubmitRequest{
		RequestRef: "request:temporary-launch", ActorRef: actor, ProjectRef: project, Statement: "wait safely",
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
	agent := &scriptedAgent{now: clock.Now, launchErr: temporaryAgentTestError{}, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("capacity recovered"),
	}}}
	orchestrator, err := New(Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: newMemoryArtifactStore(),
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1024,
		MaxActionAttempts: 2, ClaimLease: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, SubmitRequest{
		RequestRef: "request:capacity-boundary", ActorRef: actor, ProjectRef: project, Statement: "bounded capacity wait",
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
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{
		{Status: ports.AgentPending}, {Status: ports.AgentRunning},
	}}
	artifacts := newMemoryArtifactStore()
	orchestrator, err := New(Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1024,
		MaxActionAttempts: 2, ClaimLease: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	submitted, err := orchestrator.Submit(ctx, SubmitRequest{
		RequestRef: "request:bounded", ActorRef: actor, ProjectRef: project, Statement: "bounded",
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
	if err != nil || record.Goal.State() != goal.GoalStateFailed || onlyExecution(t, record).FailureCode != "application.execution_expired" {
		t.Fatalf("unbounded execution: state=%s code=%s err=%v", record.Goal.State(), onlyExecution(t, record).FailureCode, err)
	}
}
