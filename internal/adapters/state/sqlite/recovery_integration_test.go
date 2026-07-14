package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestArtifactPersistenceCrossingLeaseCannotCommitBackdatedSuccess(t *testing.T) {
	clock := &restartClock{now: time.Date(2026, 7, 14, 18, 0, 0, 0, time.UTC)}
	path := t.TempDir() + "/state/orquesta.sqlite"
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("open lease-fenced repository: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	agent := &leaseCompletionAgent{clock: clock}
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Launcher: agent, Observer: agent,
		Artifacts: leaseAdvancingArtifacts{clock: clock, advance: 2 * time.Second},
		Clock:     clock, IDs: &restartIDs{}, MaxOutputBytes: 4096, MaxActionAttempts: 3,
		ClaimLease: time.Second, ObservationDelay: time.Millisecond, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new lease-fenced orchestrator: %v", err)
	}
	actor, _ := goal.NewActorRef("actor:lease-fence")
	project, _ := goal.NewProjectRef("project:lease-fence")
	submitted, err := orchestrator.Submit(context.Background(), application.SubmitRequest{
		RequestRef: "request:lease-fence", ActorRef: actor, ProjectRef: project,
		Statement: "persist completion within the claim lease", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit lease-fenced Goal: %v", err)
	}
	if result, err := orchestrator.ProcessNext(context.Background(), "worker:launch"); err != nil || !result.Processed {
		t.Fatalf("launch before lease test = %+v err=%v", result, err)
	}
	result, err := orchestrator.ProcessNext(context.Background(), "worker:observe")
	if !result.Processed || !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("backdated success crossed lease = %+v err=%v", result, err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatalf("get after rejected success: %v", err)
	}
	items := record.Goal.WorkItems()
	if record.Goal.State() != goal.GoalStateRunning || len(items) != 1 || items[0].State() != goal.WorkItemStateRunning ||
		len(record.Artifacts) != 0 || len(record.Attestations) != 0 || len(record.Executions) != 1 ||
		record.Executions[0].State != application.ExecutionRunning {
		t.Fatalf("expired claim partially committed success: %+v", record)
	}
	var completedAt sql.NullInt64
	if err := repository.db.QueryRow(
		"SELECT completed_at FROM outbox WHERE ref = ?", "action:observe:"+record.Executions[0].Ref.String(),
	).Scan(&completedAt); err != nil || completedAt.Valid {
		t.Fatalf("expired observation action completed=%+v err=%v", completedAt, err)
	}
}

func TestDispatchingLaunchRecoversAcrossRestartWithSameIdempotency(t *testing.T) {
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 14, 16, 0, 0, 0, time.UTC)}
	ids := &restartIDs{}
	agent := &restartAgent{clock: clock, temporaryFirst: true}
	orchestrator := newRestartOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:restart")
	project, _ := goal.NewProjectRef("project:restart")
	submitted, err := orchestrator.Submit(context.Background(), application.SubmitRequest{
		RequestRef: "request:dispatch-restart", ActorRef: actor, ProjectRef: project,
		Statement: "recover durable dispatch", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	result, err := orchestrator.ProcessNext(context.Background(), "worker:first")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("first launch = %+v err=%v", result, err)
	}
	dispatching, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || len(dispatching.Executions) != 1 || dispatching.Executions[0].State != application.ExecutionDispatching ||
		!dispatching.Executions[0].StartedAt.IsZero() || !dispatching.Executions[0].DeadlineAt.IsZero() {
		t.Fatalf("durable dispatching state = %+v err=%v", dispatching, err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close before restart: %v", err)
	}

	clock.Advance(2 * time.Second)
	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8, Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	orchestrator = newRestartOrchestrator(t, repository, clock, ids, agent)
	result, err = orchestrator.ProcessNext(context.Background(), "worker:second")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("recovered launch = %+v err=%v", result, err)
	}
	running, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil || len(running.Executions) != 1 || running.Executions[0].State != application.ExecutionRunning ||
		!running.Executions[0].StartedAt.Equal(clock.Now()) ||
		!running.Executions[0].DeadlineAt.Equal(clock.Now().Add(time.Hour)) {
		t.Fatalf("accepted recovered execution = %+v err=%v", running, err)
	}
	requests := agent.launchRequests()
	if len(requests) != 2 || !reflect.DeepEqual(requests[0], requests[1]) ||
		requests[0].IdempotencyKey != running.Executions[0].IdempotencyKey {
		t.Fatalf("recovery changed causal launch request: %+v", requests)
	}
}

func TestPreparedLaunchWithRetainedClaimRecoversAfterCrashAndLeaseExpiry(t *testing.T) {
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 14, 17, 0, 0, 0, time.UTC)}
	ids := &restartIDs{}
	agent := &restartAgent{clock: clock}
	orchestrator := newRestartOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:crash-restart")
	project, _ := goal.NewProjectRef("project:crash-restart")
	submitted, err := orchestrator.Submit(context.Background(), application.SubmitRequest{
		RequestRef: "request:prepared-crash", ActorRef: actor, ProjectRef: project,
		Statement: "recover retained claim", Confirm: true,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	claim, found, err := repository.ClaimNextAction(context.Background(), application.ClaimRequest{
		WorkerRef: "worker:crashed", Token: "claim:crashed", Now: clock.Now(), LeaseDuration: time.Second,
	})
	if err != nil || !found || claim.Attempt != 1 {
		t.Fatalf("first claim = %+v found=%v err=%v", claim, found, err)
	}
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatalf("get queued Goal: %v", err)
	}
	item := record.Goal.WorkItems()[0]
	preparedGoal, err := record.Goal.StartWorkItem(
		record.Goal.Revision(), item.Revision(), item.Ref(), record.Executions[0].Ref, clock.Now(),
	)
	if err != nil {
		t.Fatalf("start prepared WorkItem: %v", err)
	}
	preparedExecution := record.Executions[0]
	preparedExecution.State = application.ExecutionDispatching
	if err := repository.RecordLaunchPrepared(context.Background(), application.LaunchPreparedState{
		Claim: claim, ExpectedGoalRevision: record.Goal.Revision(), Goal: preparedGoal,
		Execution: preparedExecution, OperationAt: clock.Now(),
		Event: application.EventRecord{
			Ref: "event:execution-dispatching:prepared-crash", Kind: "execution.dispatching",
			GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(), ExecutionRef: preparedExecution.Ref,
			OccurredAt: clock.Now(),
		},
	}); err != nil {
		t.Fatalf("record prepared before crash: %v", err)
	}
	expectedRequest := ports.AgentLaunchRequest{
		ExecutionRef: preparedExecution.Ref, GoalRef: preparedGoal.Ref(), WorkItemRef: item.Ref(),
		SpecHash: preparedGoal.SpecHash(),
		ActorRef: preparedGoal.Actor(), ProjectRef: preparedGoal.Project(), Objective: item.Objective(),
		PhaseRef: preparedGoal.Phases()[0].Ref().String(), PhaseKey: item.Phase().String(),
		PhaseTemplateRef: preparedGoal.Phases()[0].TemplateRef().String(),
		RoleKey:          item.Role().String(), WriteSet: []string{},
		OutputContract: string(item.OutputContract().Kind()), ArtifactMediaType: preparedExecution.ArtifactMediaType,
		IdempotencyKey: preparedExecution.IdempotencyKey, MaxOutputBytes: preparedExecution.MaxOutputBytes,
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("simulate crash close: %v", err)
	}

	clock.Advance(2 * time.Second)
	repository, err = Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8, Now: clock.Now,
	})
	if err != nil {
		t.Fatalf("reopen after crash: %v", err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	orchestrator = newRestartOrchestrator(t, repository, clock, ids, agent)
	result, err := orchestrator.ProcessNext(context.Background(), "worker:recovery")
	if err != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("recover retained launch = %+v err=%v", result, err)
	}
	requests := agent.launchRequests()
	if len(requests) != 1 || !reflect.DeepEqual(requests[0], expectedRequest) {
		t.Fatalf("recovered request = %+v want %+v", requests, expectedRequest)
	}
	var attempt int
	var completedAt sql.NullInt64
	if err := repository.db.QueryRow(
		"SELECT attempt, completed_at FROM outbox WHERE ref = ?", claim.Action.Ref,
	).Scan(&attempt, &completedAt); err != nil || attempt != 2 || !completedAt.Valid {
		t.Fatalf("recovered claim = attempt:%d completed:%+v err:%v", attempt, completedAt, err)
	}
}

func newRestartOrchestrator(
	t *testing.T,
	repository application.StateRepository,
	clock *restartClock,
	ids *restartIDs,
	agent *restartAgent,
) *application.Orchestrator {
	t.Helper()
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Launcher: agent, Observer: agent, Artifacts: restartArtifacts{},
		Clock: clock, IDs: ids, MaxOutputBytes: 4096, MaxActionAttempts: 3,
		ClaimLease: time.Minute, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new orchestrator: %v", err)
	}
	return orchestrator
}

type restartClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *restartClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *restartClock) Advance(duration time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(duration)
	clock.mu.Unlock()
}

type restartIDs struct {
	mu   sync.Mutex
	next int
}

func (ids *restartIDs) NewID(ctx context.Context, namespace string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	ids.mu.Lock()
	defer ids.mu.Unlock()
	ids.next++
	return fmt.Sprintf("%s:restart-%d", namespace, ids.next), nil
}

type restartTemporaryError struct{}

func (restartTemporaryError) Error() string   { return "restart_agent.ack_lost" }
func (restartTemporaryError) Temporary() bool { return true }

type restartAgent struct {
	mu             sync.Mutex
	clock          *restartClock
	temporaryFirst bool
	requests       []ports.AgentLaunchRequest
	receipt        ports.AgentLaunchReceipt
}

func (agent *restartAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:restart"}, nil
}

func (agent *restartAgent) Launch(ctx context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.requests = append(agent.requests, request)
	if len(agent.requests) == 1 {
		agent.receipt = ports.AgentLaunchReceipt{
			ExecutionRef: request.ExecutionRef, SpecHash: request.SpecHash, ProviderRef: "provider:restart",
			ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
			AcceptedAt: agent.clock.Now(),
		}
		if agent.temporaryFirst {
			return ports.AgentLaunchReceipt{}, restartTemporaryError{}
		}
	}
	return agent.receipt, nil
}

func (agent *restartAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{}, errors.New("restart_agent.observe_not_used")
}

func (agent *restartAgent) launchRequests() []ports.AgentLaunchRequest {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	result := append([]ports.AgentLaunchRequest(nil), agent.requests...)
	for index := range result {
		result[index].WriteSet = make([]string, len(result[index].WriteSet))
		copy(result[index].WriteSet, agent.requests[index].WriteSet)
	}
	return result
}

type restartArtifacts struct{}

func (restartArtifacts) Put(context.Context, ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	return ports.StoredArtifact{}, errors.New("restart_artifact.put_not_used")
}

func (restartArtifacts) Get(context.Context, goal.ArtifactRef, int64) (ports.ArtifactContent, error) {
	return ports.ArtifactContent{}, errors.New("restart_artifact.get_not_used")
}

type leaseCompletionAgent struct {
	mu       sync.Mutex
	clock    *restartClock
	specHash string
}

func (agent *leaseCompletionAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return ports.AgentCapabilities{ProviderRef: "provider:lease-test"}, nil
}

func (agent *leaseCompletionAgent) Launch(_ context.Context, request ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	agent.specHash = request.SpecHash
	agent.mu.Unlock()
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, SpecHash: request.SpecHash, ProviderRef: "provider:lease-test",
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		AcceptedAt: agent.clock.Now(),
	}, nil
}

func (agent *leaseCompletionAgent) Observe(_ context.Context, executionRef goal.ExecutionRef) (ports.AgentObservation, error) {
	agent.mu.Lock()
	specHash := agent.specHash
	agent.mu.Unlock()
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: specHash, Status: ports.AgentCompleted, MediaType: "text/plain",
		Content: []byte("lease-fenced artifact"), ObservedAt: agent.clock.Now(),
	}, nil
}

type leaseAdvancingArtifacts struct {
	clock   *restartClock
	advance time.Duration
}

func (store leaseAdvancingArtifacts) Put(_ context.Context, request ports.PutArtifactRequest) (ports.StoredArtifact, error) {
	store.clock.Advance(store.advance)
	digest := sha256.Sum256(request.Content)
	digestText := hex.EncodeToString(digest[:])
	ref, err := goal.NewArtifactRef("artifact:sha256:" + digestText)
	if err != nil {
		return ports.StoredArtifact{}, err
	}
	return ports.StoredArtifact{
		Ref: ref, Digest: digestText, MediaType: request.MediaType, Size: int64(len(request.Content)),
	}, nil
}

func (leaseAdvancingArtifacts) Get(context.Context, goal.ArtifactRef, int64) (ports.ArtifactContent, error) {
	return ports.ArtifactContent{}, errors.New("lease_artifact.get_not_used")
}
