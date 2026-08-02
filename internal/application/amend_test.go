package application

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const v04DifferentSpecHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type v04CountingClock struct {
	now   time.Time
	calls int
}

type v04SnapshotSubstitutingRepository struct {
	*memoryRepository
	substituteCreate bool
	substituteAmend  bool
}

func (repository *v04SnapshotSubstitutingRepository) CreateGoal(
	ctx context.Context,
	state CreateGoalState,
) (GoalRecord, bool, error) {
	record, created, err := repository.memoryRepository.CreateGoal(ctx, state)
	if err != nil || !created || !repository.substituteCreate {
		return record, created, err
	}
	snapshot := record.Goal.Snapshot()
	snapshot.StartedAt = snapshot.StartedAt.Add(time.Nanosecond)
	record.Goal, err = goal.RestoreGoal(snapshot)
	return record, created, err
}

func (repository *v04SnapshotSubstitutingRepository) AmendGoal(
	ctx context.Context,
	state AmendGoalState,
) (GoalRecord, bool, error) {
	record, created, err := repository.memoryRepository.AmendGoal(ctx, state)
	if err != nil || !created || !repository.substituteAmend {
		return record, created, err
	}
	snapshot := record.Goal.Snapshot()
	snapshot.CreatedAt = snapshot.CreatedAt.Add(time.Nanosecond)
	record.Goal, err = goal.RestoreGoal(snapshot)
	return record, created, err
}

func (clock *v04CountingClock) Now() time.Time {
	clock.calls++
	return clock.now
}

func TestV04ConfirmationFalseHasNoClockIDOrStateEffect(t *testing.T) {
	ctx := context.Background()
	clock := &v04CountingClock{now: time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC)}
	ids := &sequentialIDs{}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	artifacts := newMemoryArtifactStore()
	orchestrator, err := New(Dependencies{
		State: repository, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: artifacts,
		Clock: clock, IDs: ids, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, AgentCapabilities: testAgentCapabilities(),
		ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(time.Unix(1, 0)),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)

	_, submitErr := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:not-confirmed",
		Statement:  "must not exist", Confirm: false,
	})
	_, amendErr := orchestrator.Amend(ctx, access, AmendRequest{
		RequestRef: "request:amend-not-confirmed",
		Statement:  "must not exist either", Confirm: false,
	})
	if submitErr == nil || submitErr.Error() != "application.confirmation_required" ||
		amendErr == nil || amendErr.Error() != "application.confirmation_required" {
		t.Fatalf("confirmation errors: submit=%v amend=%v", submitErr, amendErr)
	}
	if clock.calls != 0 || ids.next != 0 || len(repository.records) != 0 ||
		len(repository.requests) != 0 || len(repository.actions) != 0 || len(repository.events) != 0 ||
		agent.launches != 0 || len(artifacts.content) != 0 {
		t.Fatalf("confirm=false caused effects: clock=%d ids=%d records=%d requests=%d actions=%d events=%d launches=%d artifacts=%d",
			clock.calls, ids.next, len(repository.records), len(repository.requests), len(repository.actions),
			len(repository.events), agent.launches, len(artifacts.content))
	}
}

func TestV04SubmitCreatesConfirmedRootAppSpecFromExactIntent(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 9, 0, 0, 123, time.FixedZone("source", 2*60*60))}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	request := SubmitRequest{
		RequestRef: "request:root-spec",
		Statement:  "  preserve this exact intent  ", Confirm: true,
	}

	result, err := orchestrator.Submit(ctx, access, request)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	spec := result.Record.Goal.AppSpec()
	intent := spec.Intent()
	parentRef, hasParent := spec.ParentRef()
	if !result.Created || intent.Statement() != request.Statement || spec.Generation() != 1 ||
		spec.Objective() != "preserve this exact intent" || spec.Reason() != initialAppSpecReason ||
		spec.ConfirmedBy() != actor || spec.ConfirmedAt() != clock.now.UTC() || hasParent ||
		parentRef.String() != "" || !goal.IsCanonicalAppSpecHash(spec.Hash()) || result.Record.Goal.SpecHash() != spec.Hash() {
		t.Fatalf("invalid root AppSpec: created=%v intent=%q spec=%+v parent=%s/%v", result.Created, intent.Statement(), spec.Snapshot(), parentRef.String(), hasParent)
	}
	items := result.Record.Goal.WorkItems()
	if len(items) != 1 || items[0].Objective() != spec.Objective() {
		t.Fatalf("default work objective does not derive from AppSpec: %+v", items)
	}
	summaries, err := orchestrator.ListGoals(ctx, access, 10)
	if err != nil || len(summaries) != 1 || summaries[0].AppSpecRef != spec.Ref() ||
		summaries[0].AppSpecGeneration != spec.Generation() || summaries[0].SpecHash != spec.Hash() {
		t.Fatalf("GoalSummary lost AppSpec identity: summaries=%+v err=%v", summaries, err)
	}
	differentExactIntent := request
	differentExactIntent.Statement = strings.TrimSpace(request.Statement)
	if submissionFingerprint(access, request) == submissionFingerprint(access, differentExactIntent) {
		t.Fatal("submission fingerprint lost exact Intent statement")
	}
}

func TestV04InitialAppSpecReasonMatchesAcceptanceContract(t *testing.T) {
	if initialAppSpecReason != "operator.initial_confirmation" {
		t.Fatalf("initial AppSpec reason = %q", initialAppSpecReason)
	}
}

func TestV04SubmitRejectsCreatedSnapshotSubstitution(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 9, 30, 0, 0, time.UTC)}
	repository := &v04SnapshotSubstitutingRepository{
		memoryRepository: newMemoryRepository(), substituteCreate: true,
	}
	agent := &scriptedAgent{now: clock.Now}
	orchestrator := v04NewOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	_, err := orchestrator.Submit(ctx, access, SubmitRequest{
		RequestRef: "request:substituted-create",
		Statement:  "exact candidate only", Confirm: true,
	})
	if !IsStateError(err, StateConflict) {
		t.Fatalf("created Goal snapshot substitution accepted: %v", err)
	}
}

func TestV04AmendCreatesCausalPendingSuccessorAndReplayIsIdempotent(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("source evidence"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	source := v04SubmitAndClose(t, ctx, orchestrator, clock, access, SubmitRequest{
		RequestRef: "request:source",
		Statement:  "source exact", NormalizedObjective: "source normalized", Confirm: true,
	})
	sourceBefore := source.Goal.Snapshot()
	request := AmendRequest{
		RequestRef:    "request:amend",
		SourceGoalRef: source.Goal.Ref(), ExpectedSourceRevision: source.Goal.Revision(),
		ExpectedSourceSpecHash: source.Goal.SpecHash(), Statement: "  amended exact  ",
		NormalizedObjective: " amended normalized ", Reason: "operator clarified scope", Confirm: true,
	}

	amended, err := orchestrator.Amend(ctx, access, request)
	if err != nil {
		t.Fatalf("amend: %v", err)
	}
	spec := amended.Record.Goal.AppSpec()
	parentRef, hasParent := spec.ParentRef()
	if !amended.Created || amended.Record.Goal.State() != goal.GoalStatePending ||
		amended.Record.Goal.WorkItemCount() != 0 || len(amended.Record.Executions) != 0 ||
		len(amended.Record.Artifacts) != 0 || len(amended.Record.Attestations) != 0 ||
		spec.Generation() != source.Goal.AppSpec().Generation()+1 || !hasParent ||
		parentRef != source.Goal.AppSpec().Ref() || spec.ParentHash() != source.Goal.SpecHash() ||
		spec.Intent().Statement() != request.Statement || spec.Objective() != "amended normalized" ||
		spec.Reason() != request.Reason || spec.ConfirmedBy() != actor {
		t.Fatalf("invalid successor: created=%v goal=%+v spec=%+v", amended.Created, amended.Record.Goal.Snapshot(), spec.Snapshot())
	}
	sourceAfter, err := repository.GetGoal(ctx, source.Goal.Ref())
	if err != nil || !reflect.DeepEqual(sourceAfter.Goal.Snapshot(), sourceBefore) ||
		len(sourceAfter.Artifacts) != len(source.Artifacts) || len(sourceAfter.Attestations) != len(source.Attestations) {
		t.Fatalf("source history changed: before=%+v after=%+v err=%v", sourceBefore, sourceAfter, err)
	}

	replayed, err := orchestrator.Amend(ctx, access, request)
	if err != nil || replayed.Created || replayed.Record.Goal.Ref() != amended.Record.Goal.Ref() || len(repository.records) != 2 {
		t.Fatalf("amend replay: created=%v ref=%s records=%d err=%v", replayed.Created, replayed.Record.Goal.Ref().String(), len(repository.records), err)
	}
	conflict := request
	conflict.Statement = "different exact amendment"
	if _, err := orchestrator.Amend(ctx, access, conflict); !IsStateError(err, StateConflict) || len(repository.records) != 2 {
		t.Fatalf("amend semantic conflict: records=%d err=%v", len(repository.records), err)
	}
	secondSuccessor := request
	secondSuccessor.RequestRef = "request:amend-branch"
	if _, err := orchestrator.Amend(ctx, access, secondSuccessor); !IsStateError(err, StateConflict) ||
		len(repository.records) != 2 || len(repository.successors) != 1 {
		t.Fatalf("second successor accepted: records=%d successors=%d err=%v", len(repository.records), len(repository.successors), err)
	}
}

func TestV04AmendRejectsCreatedSnapshotSubstitution(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 10, 30, 0, 0, time.UTC)}
	repository := &v04SnapshotSubstitutingRepository{
		memoryRepository: newMemoryRepository(), substituteAmend: true,
	}
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("closed source"),
	}}}
	orchestrator := v04NewOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	source := v04SubmitAndClose(t, ctx, orchestrator, clock, access, SubmitRequest{
		RequestRef: "request:substitution-source",
		Statement:  "source", Confirm: true,
	})
	_, err := orchestrator.Amend(ctx, access, v04AmendRequest(
		source.Goal, "request:substituted-amend",
	))
	if !IsStateError(err, StateConflict) {
		t.Fatalf("created successor snapshot substitution accepted: %v", err)
	}
}

func TestV04AmendRejectsNonterminalStaleAndForeignSource(t *testing.T) {
	t.Run("nonterminal", func(t *testing.T) {
		ctx := context.Background()
		clock := &v04CountingClock{now: time.Date(2026, 7, 14, 11, 0, 0, 0, time.UTC)}
		ids := &sequentialIDs{}
		repository := newMemoryRepository()
		agent := &scriptedAgent{now: clock.Now}
		orchestrator, err := New(Dependencies{
			State: repository, Access: newMemoryAccessRepository(),
			Launcher: agent, Observer: agent, Artifacts: newMemoryArtifactStore(),
			Clock: clock, IDs: ids, MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
			MaxExecutionAttempts: 3, AgentCapabilities: testAgentCapabilities(),
			ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
			MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
			ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		})
		if err != nil {
			t.Fatalf("new: %v", err)
		}
		actor, project := testScope(t)
		access := accessForScope(t, actor, project)
		source, err := orchestrator.Submit(ctx, access, SubmitRequest{
			RequestRef: "request:running",
			Statement:  "still running", Confirm: true,
		})
		if err != nil {
			t.Fatalf("submit: %v", err)
		}
		clockCalls, idCalls := clock.calls, ids.next
		records, requests := len(repository.records), len(repository.requests)
		actions, events := len(repository.actions), len(repository.events)
		_, err = orchestrator.Amend(ctx, access, v04AmendRequest(source.Record.Goal, "request:running-amend"))
		if goal.ErrorCodeOf(err) != goal.ErrorInvalidTransition || len(repository.records) != records ||
			len(repository.requests) != requests || len(repository.actions) != actions || len(repository.events) != events ||
			len(repository.successors) != 0 || clock.calls != clockCalls+1 || ids.next != idCalls+1 {
			t.Fatalf("nonterminal amendment caused effects: err=%v records=%d/%d requests=%d/%d actions=%d/%d events=%d/%d successors=%d clock=%d/%d ids=%d/%d",
				err, len(repository.records), records, len(repository.requests), requests,
				len(repository.actions), actions, len(repository.events), events, len(repository.successors),
				clock.calls, clockCalls, ids.next, idCalls)
		}
	})

	t.Run("stale revision and hash", func(t *testing.T) {
		ctx := context.Background()
		clock := &mutableClock{now: time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)}
		repository := newMemoryRepository()
		agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
			Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("closed"),
		}}}
		orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
		actor, project := testScope(t)
		access := accessForScope(t, actor, project)
		source := v04SubmitAndClose(t, ctx, orchestrator, clock, access, SubmitRequest{
			RequestRef: "request:stale-source",
			Statement:  "closed", Confirm: true,
		})
		staleRevision := v04AmendRequest(source.Goal, "request:stale-revision")
		staleRevision.ExpectedSourceRevision++
		if _, err := orchestrator.Amend(ctx, access, staleRevision); !IsStateError(err, StateConflict) {
			t.Fatalf("stale revision accepted: %v", err)
		}
		staleHash := v04AmendRequest(source.Goal, "request:stale-hash")
		staleHash.ExpectedSourceSpecHash = v04DifferentSpecHash
		if _, err := orchestrator.Amend(ctx, access, staleHash); !IsStateError(err, StateConflict) {
			t.Fatalf("stale spec hash accepted: %v", err)
		}
		foreignActor, _ := goal.NewActorRef("actor:foreign")
		foreignProject, _ := goal.NewProjectRef("project:foreign")
		foreignAccess := accessForScope(t, foreignActor, foreignProject)
		foreign := v04AmendRequest(source.Goal, "request:foreign")
		if _, err := orchestrator.Amend(ctx, foreignAccess, foreign); !IsStateError(err, StateNotFound) {
			t.Fatalf("foreign scope leaked source: %v", err)
		}
		if len(repository.records) != 1 {
			t.Fatalf("rejected amendments wrote successors: %d", len(repository.records))
		}
	})
}

func TestV04ConcurrentAmendAllowsExactlyOneSuccessorPerSourceSpec(t *testing.T) {
	ctx := context.Background()
	clock := &mutableClock{now: time.Date(2026, 7, 14, 12, 30, 0, 0, time.UTC)}
	repository := newMemoryRepository()
	agent := &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
		Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("closed"),
	}}}
	orchestrator, _ := newTestOrchestrator(t, repository, clock, agent)
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	source := v04SubmitAndClose(t, ctx, orchestrator, clock, access, SubmitRequest{
		RequestRef: "request:concurrent-source",
		Statement:  "closed source", Confirm: true,
	})
	requests := []AmendRequest{
		v04AmendRequest(source.Goal, "request:successor-a"),
		v04AmendRequest(source.Goal, "request:successor-b"),
	}
	start := make(chan struct{})
	type outcome struct {
		result AmendResult
		err    error
	}
	outcomes := make(chan outcome, len(requests))
	for _, request := range requests {
		request := request
		go func() {
			<-start
			result, err := orchestrator.Amend(ctx, access, request)
			outcomes <- outcome{result: result, err: err}
		}()
	}
	close(start)
	created, conflicts := 0, 0
	for range requests {
		outcome := <-outcomes
		switch {
		case outcome.err == nil && outcome.result.Created:
			created++
		case IsStateError(outcome.err, StateConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent amendment result: %+v err=%v", outcome.result, outcome.err)
		}
	}
	if created != 1 || conflicts != 1 || len(repository.records) != 2 || len(repository.successors) != 1 {
		t.Fatalf("successor uniqueness failed: created=%d conflicts=%d records=%d successors=%d",
			created, conflicts, len(repository.records), len(repository.successors))
	}
}

func TestV04ProviderSpecHashMismatchCreatesNoEvidenceOrClosure(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		agent     func(*mutableClock) *scriptedAgent
		wantError string
	}{
		{
			name: "receipt required",
			agent: func(clock *mutableClock) *scriptedAgent {
				return &scriptedAgent{now: clock.Now, preserveEmptyReceiptSpecHash: true}
			},
			wantError: "agent.receipt_spec_hash_required",
		},
		{
			name: "receipt invalid",
			agent: func(clock *mutableClock) *scriptedAgent {
				return &scriptedAgent{now: clock.Now, receiptSpecHashOverride: strings.Repeat("A", 64)}
			},
			wantError: "agent.receipt_spec_hash_invalid",
		},
		{
			name: "receipt",
			agent: func(clock *mutableClock) *scriptedAgent {
				return &scriptedAgent{now: clock.Now, receiptSpecHashOverride: v04DifferentSpecHash}
			},
			wantError: "agent.receipt_spec_hash_mismatch",
		},
		{
			name: "observation required",
			agent: func(clock *mutableClock) *scriptedAgent {
				return &scriptedAgent{now: clock.Now, preserveEmptyObservationSpecHash: true, observations: []ports.AgentObservation{{
					Status: ports.AgentCompleted, MediaType: "text/plain", Content: []byte("must not persist"),
				}}}
			},
			wantError: "agent.observation_spec_hash_required",
		},
		{
			name: "observation invalid",
			agent: func(clock *mutableClock) *scriptedAgent {
				return &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
					SpecHash: strings.Repeat("A", 64), Status: ports.AgentCompleted,
					MediaType: "text/plain", Content: []byte("must not persist"),
				}}}
			},
			wantError: "agent.observation_spec_hash_invalid",
		},
		{
			name: "observation",
			agent: func(clock *mutableClock) *scriptedAgent {
				return &scriptedAgent{now: clock.Now, observations: []ports.AgentObservation{{
					SpecHash: v04DifferentSpecHash, Status: ports.AgentCompleted,
					MediaType: "text/plain", Content: []byte("must not persist"),
				}}}
			},
			wantError: "agent.observation_spec_hash_mismatch",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			clock := &mutableClock{now: time.Date(2026, 7, 14, 13, 0, 0, 0, time.UTC)}
			repository := newMemoryRepository()
			agent := testCase.agent(clock)
			orchestrator, artifacts := newTestOrchestrator(t, repository, clock, agent)
			actor, project := testScope(t)
			access := accessForScope(t, actor, project)
			submitted, err := orchestrator.Submit(ctx, access, SubmitRequest{
				RequestRef: "request:mismatch-" + testCase.name,
				Statement:  "reject crossed evidence", Confirm: true,
			})
			if err != nil {
				t.Fatalf("submit: %v", err)
			}
			result, processErr := orchestrator.ProcessNext(ctx, "worker:test")
			receiptCase := strings.HasPrefix(testCase.name, "receipt")
			if !receiptCase {
				if processErr != nil {
					t.Fatalf("launch: %v", processErr)
				}
				clock.Advance(time.Second)
				result, processErr = orchestrator.ProcessNext(ctx, "worker:test")
			}
			wantProcessError := testCase.wantError
			if receiptCase {
				wantProcessError = effectUnknownAppliedCode
			}
			if !result.Processed || processErr == nil || processErr.Error() != wantProcessError {
				t.Fatalf("spec-hash fence result=%+v err=%v, want %s", result, processErr, testCase.wantError)
			}
			record, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
			status, statusErr := repository.Status(ctx, project)
			wantQuarantined, wantPending := int64(1), int64(0)
			if err != nil || statusErr != nil || record.Goal.IsTerminal() || record.Goal.State() != goal.GoalStateRunning ||
				onlyExecution(t, record).FailureCode != "" || status.QuarantinedActions != wantQuarantined || status.PendingActions != wantPending ||
				len(record.Artifacts) != 0 || len(record.Attestations) != 0 || len(artifacts.content) != 0 {
				t.Fatalf("mismatched evidence escaped governed state: record=%+v status=%+v stored=%d err=%v/%v",
					record, status, len(artifacts.content), err, statusErr)
			}
			if replay, err := orchestrator.ProcessNext(ctx, "worker:test"); err != nil || replay.Processed {
				t.Fatalf("quarantined action replayed: result=%+v err=%v", replay, err)
			}
		})
	}
}

func v04SubmitAndClose(
	t *testing.T,
	ctx context.Context,
	orchestrator *Orchestrator,
	clock *mutableClock,
	access Access,
	request SubmitRequest,
) GoalRecord {
	t.Helper()
	submitted, err := orchestrator.Submit(ctx, access, request)
	if err != nil {
		t.Fatalf("submit source: %v", err)
	}
	if _, err := orchestrator.ProcessNext(ctx, "worker:v04"); err != nil {
		t.Fatalf("launch source: %v", err)
	}
	clock.Advance(time.Second)
	if _, err := orchestrator.ProcessNext(ctx, "worker:v04"); err != nil {
		t.Fatalf("observe source: %v", err)
	}
	record, err := orchestrator.GetGoal(ctx, access, submitted.Record.Goal.Ref())
	if err != nil || !record.Goal.IsTerminal() {
		t.Fatalf("source not terminal: state=%s err=%v", record.Goal.State(), err)
	}
	return record
}

func v04AmendRequest(source goal.Goal, requestRef string) AmendRequest {
	return AmendRequest{
		RequestRef:    requestRef,
		SourceGoalRef: source.Ref(), ExpectedSourceRevision: source.Revision(),
		ExpectedSourceSpecHash: source.SpecHash(), Statement: "successor",
		NormalizedObjective: "successor objective", Reason: "test amendment", Confirm: true,
	}
}

func v04NewOrchestrator(
	t *testing.T,
	state StateRepository,
	clock Clock,
	agent *scriptedAgent,
) *Orchestrator {
	t.Helper()
	orchestrator, err := New(Dependencies{
		State: state, Access: newMemoryAccessRepository(),
		Launcher: agent, Observer: agent, Artifacts: newMemoryArtifactStore(),
		Clock: clock, IDs: &sequentialIDs{}, MaxOutputBytes: 1 << 20, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, AgentCapabilities: testAgentCapabilities(), ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute,
		MaxChildrenPerParent:  6, EffectApprovalTTL: time.Hour, BudgetPolicy: testBudgetPolicy(clock.Now()),
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		CapacitySources: fuentesCapacidadPruebaEstaticas(), CapacityObservationWait: time.Second,
	})
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator
}
