package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestNewClonesAgentCapabilitySlices(t *testing.T) {
	clock := &mutableClock{now: time.Date(2026, 7, 14, 19, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	capabilities := ports.AgentCapabilities{
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
		RoleKeys: []string{"role:worker"}, SkillRefs: []string{"skill:test"},
		ToolRefs: []string{"tool:test"}, CapabilityRefs: []string{"capability:test"},
	}
	orchestrator, err := New(Dependencies{
		State: repository, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: newMemoryArtifactStore(),
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20, MaxExecutionAttempts: 3,
		ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: capabilities,
	})
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	capabilities.RoleKeys[0] = "role:mutated"
	capabilities.SkillRefs[0] = "skill:mutated"
	capabilities.ToolRefs[0] = "tool:mutated"
	capabilities.CapabilityRefs[0] = "capability:mutated"
	want := ports.AgentCapabilities{
		ProviderRef: "provider:test", ModelRef: "model:test", AgentRef: "agent:test",
		RoleKeys: []string{"role:worker"}, SkillRefs: []string{"skill:test"},
		ToolRefs: []string{"tool:test"}, CapabilityRefs: []string{"capability:test"},
	}
	if !reflect.DeepEqual(orchestrator.agentCapabilities, want) {
		t.Fatalf("caller mutated scheduler capabilities after New: got=%+v want=%+v", orchestrator.agentCapabilities, want)
	}
}

func TestOrchestratorOwnsOneDurableLifecycleWriter(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 20, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	launchEntered := make(chan struct{}, 1)
	launchRelease := make(chan struct{})
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "Text/Plain; charset=utf-8",
		Content: []byte("resultado acreditado"),
	}}, launchEntered: launchEntered, launchRelease: launchRelease}
	orchestrator, artifacts := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)

	first, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:one",
		Statement:  "produce un resultado pequeño", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if !first.Created || first.Record.Goal.State() != goal.GoalStateRunning {
		t.Fatalf("unexpected creation: created=%v state=%s", first.Created, first.Record.Goal.State())
	}
	duplicate, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:one",
		Statement:  "produce un resultado pequeño", Confirm: true,
	})
	if err != nil {
		t.Fatalf("idempotent submit: %v", err)
	}
	if duplicate.Created || duplicate.Record.Goal.Ref() != first.Record.Goal.Ref() {
		t.Fatalf("submission was duplicated")
	}

	firstWorker := make(chan error, 1)
	go func() {
		_, processErr := orchestrator.ProcessNext(ctx, "worker:one")
		firstWorker <- processErr
	}()
	<-launchEntered
	secondResult, secondErr := orchestrator.ProcessNext(ctx, "worker:two")
	if secondErr != nil || secondResult.Processed {
		t.Fatalf("leased action was claimed twice: result=%+v err=%v", secondResult, secondErr)
	}
	close(launchRelease)
	if err := <-firstWorker; err != nil {
		t.Fatalf("first worker: %v", err)
	}
	if agent.launches != 1 {
		t.Fatalf("single writer claim failed: launches=%d", agent.launches)
	}

	clock.Advance(time.Second)
	result, err := orchestrator.ProcessNext(ctx, "worker:one")
	if err != nil || !result.Processed || result.Action != ActionObserveAgent {
		t.Fatalf("observe action: result=%+v err=%v", result, err)
	}
	record, err := orchestrator.GetGoal(ctx, access, first.Record.Goal.Ref())
	if err != nil {
		t.Fatalf("get closed goal: %v", err)
	}
	if record.Goal.State() != goal.GoalStateSucceeded || len(record.Artifacts) != 1 || len(record.Attestations) != 1 {
		t.Fatalf("closure not accredited: state=%s artifacts=%d attestations=%d", record.Goal.State(), len(record.Artifacts), len(record.Attestations))
	}
	content, err := orchestrator.GetArtifact(ctx, access, record.Goal.Ref(), record.Artifacts[0].Stored.Ref)
	if err != nil || string(content.Content) != "resultado acreditado" {
		t.Fatalf("artifact mismatch: %q err=%v", content.Content, err)
	}
	if len(artifacts.content) != 1 {
		t.Fatalf("artifact duplicated: %d", len(artifacts.content))
	}
	result, err = orchestrator.ProcessNext(ctx, "worker:one")
	if err != nil || result.Processed {
		t.Fatalf("terminal work replayed: result=%+v err=%v", result, err)
	}
}

func TestSubmitIdempotencyRejectsSemanticConflict(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 20, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	if _, err := orchestrator.Submit(ctx, access, SubmitRequest{RequestRef: "request:same", Statement: "uno", Confirm: true}); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	_, err := orchestrator.Submit(ctx, access, SubmitRequest{RequestRef: "request:same", Statement: "dos", Confirm: true})
	if !IsStateError(err, StateConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestArtifactReadRejectsCorruptAdapterContentAfterScopedLookup(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 20, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("trusted"),
	}}}
	orchestrator, artifacts := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:corrupt-read", Statement: "read safely", Confirm: true,
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
	if err != nil || len(record.Artifacts) != 1 {
		t.Fatalf("closed record: %+v err=%v", record, err)
	}
	ref := record.Artifacts[0].Stored.Ref
	artifacts.mu.Lock()
	corrupt := artifacts.content[ref]
	corrupt.Content = []byte("substituted")
	artifacts.content[ref] = corrupt
	artifacts.mu.Unlock()
	_, err = orchestrator.GetArtifact(ctx, access, record.Goal.Ref(), ref)
	if err == nil || err.Error() != "artifact.content_contract_invalid" {
		t.Fatalf("corrupt adapter content accepted: %v", err)
	}
}

func TestProviderClockSkewCannotDriveLifecycle(t *testing.T) {
	ctx := context.Background()
	logicalNow := time.Date(2026, 7, 14, 23, 0, 0, 0, time.UTC)
	clock := &mutableClock{now: logicalNow}
	repository := newMemoryRepository()
	providerNow := func() time.Time { return logicalNow.Add(-24 * time.Hour) }
	agent := &scriptedAgent{now: providerNow, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("clock-safe"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:clock-skew", Statement: "clock", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("launch with skew: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil {
		t.Fatalf("observe with skew: %v", err)
	}
	record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || record.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("skew controlled lifecycle: state=%s err=%v", record.Goal.State(), err)
	}
	if !onlyExecution(t, record).ProviderAcceptedAt.Equal(providerNow()) || !onlyExecution(t, record).StartedAt.Equal(logicalNow) {
		t.Fatalf("provider and lifecycle clocks were not separated: %+v", onlyExecution(t, record))
	}
}

func TestCompletedObservationWithUnexpectedMediaTypeFailsWithoutEvidence(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 23, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "application/json", Content: []byte(`{"unexpected":true}`),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:media-mismatch", Statement: "plain text", Confirm: true,
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
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if record.Goal.State() != goal.GoalStateFailed || len(record.Artifacts) != 0 || len(record.Attestations) != 0 ||
		onlyExecution(t, record).FailureCode != "agent.observation_media_type_mismatch" {
		t.Fatalf("unexpected mismatch closure: %+v", record)
	}
}

func TestClaimRecordMismatchIsQuarantinedBeforeAgentEffect(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	if _, err := orchestrator.Submit(ctx, access, SubmitRequest{RequestRef: "request:bad-action", Statement: "safe", Confirm: true}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	repository.mu.Lock()
	for ref, action := range repository.actions {
		action.record.ExecutionRef, _ = goal.NewExecutionRef("execution:crossed-row")
		repository.actions[ref] = action
	}
	repository.mu.Unlock()
	if result, err := orchestrator.ProcessNext(ctx, "worker:test"); !result.Processed || err == nil {
		t.Fatalf("mismatch was not reported: result=%+v err=%v", result, err)
	}
	if agent.launches != 0 {
		t.Fatalf("agent ran before causal validation")
	}
	if result, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil || result.Processed {
		t.Fatalf("quarantined action replayed: result=%+v err=%v", result, err)
	}
}
