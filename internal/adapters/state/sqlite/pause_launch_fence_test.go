package sqlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLitePauseGatesQueuedRetryAcrossRestart(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 17, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteMultiControlAgent{clock: clock}
	orchestrator := newSQLiteMultiControlOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:sqlite-pause-retry")
	project, _ := goal.NewProjectRef("project:sqlite-pause-retry")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-pause-retry", Statement: "pause retry before launch claim", Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-pause-initial"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("initial launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item := running.Goal.WorkItems()[0]
	execution := running.Executions[0]
	stop := sqliteLaunchFenceControl(running, "control:sqlite-pause-retry-stop", application.ControlStop,
		application.ControlTargetExecution, item.Ref(), execution.Ref)
	stop.Mode = ports.AgentStopCooperative
	if _, err := orchestrator.Control(ctx, access, stop); err != nil {
		t.Fatal(err)
	}
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-pause-stop"); processErr != nil ||
		!result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("stop: result=%+v err=%v", result, processErr)
	}
	stopped, _ := repository.GetGoal(ctx, running.Goal.Ref())
	item, _ = stopped.Goal.WorkItem(item.Ref())
	retry := sqliteLaunchFenceControl(stopped, "control:sqlite-pause-retry-create", application.ControlRetry,
		application.ControlTargetWorkItem, item.Ref(), execution.Ref)
	if _, err := orchestrator.Control(ctx, access, retry); err != nil {
		t.Fatal(err)
	}
	retried, _ := repository.GetGoal(ctx, running.Goal.Ref())
	replacement := sqliteBoundExecution(t, retried, item.Ref())
	if replacement.State != application.ExecutionQueued {
		t.Fatalf("retry state=%s, want queued", replacement.State)
	}
	pause := sqliteLaunchFenceControl(retried, "control:sqlite-pause-retry-before-claim", application.ControlPause,
		application.ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
	if _, err := orchestrator.Control(ctx, access, pause); err != nil {
		t.Fatal(err)
	}
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-paused-retry"); processErr != nil || result.Processed {
		t.Fatalf("paused retry claimed: result=%+v err=%v", result, processErr)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery rejected paused queued retry: %v", err)
	}
	persisted, _ := restarted.GetGoal(ctx, running.Goal.Ref())
	if current := sqliteBoundExecution(t, persisted, item.Ref()); !persisted.Goal.Paused() || current.State != application.ExecutionQueued {
		t.Fatalf("restart state: paused=%v execution=%s", persisted.Goal.Paused(), current.State)
	}
	restartedOrchestrator := newSQLiteMultiControlOrchestrator(t, restarted, clock, ids, agent)
	resume := sqliteLaunchFenceControl(persisted, "control:sqlite-pause-retry-resume", application.ControlResume,
		application.ControlTargetGoal, goal.WorkItemRef{}, goal.ExecutionRef{})
	if _, err := restartedOrchestrator.Control(ctx, access, resume); err != nil {
		t.Fatal(err)
	}
	if result, processErr := restartedOrchestrator.ProcessNext(ctx, "worker:sqlite-resumed-retry"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("resumed retry: result=%+v err=%v", result, processErr)
	}
	launched, _ := restarted.GetGoal(ctx, running.Goal.Ref())
	if current := sqliteBoundExecution(t, launched, item.Ref()); current.State != application.ExecutionRunning {
		t.Fatalf("resumed retry execution=%s", current.State)
	}
	assertSQLiteLaunchFrontierEvents(t, restarted, replacement.Ref)
}

func TestSQLitePauseGatesAutomaticReplacementBackoffAcrossRestart(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 18, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteLaunchFenceAgent{clock: clock, failLaunches: 1}
	orchestrator := newSQLiteLaunchFenceOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:sqlite-pause-backoff")
	project, _ := goal.NewProjectRef("project:sqlite-pause-backoff")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-pause-backoff", Statement: "pause automatic replacement backoff", Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-backoff-failure"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("create replacement: result=%+v err=%v", result, processErr)
	}
	replaced, _ := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	item := replaced.Goal.WorkItems()[0]
	replacement := sqliteBoundExecution(t, replaced, item.Ref())
	if replacement.State != application.ExecutionQueued || replacement.AttemptNo != 2 {
		t.Fatalf("replacement=%+v", replacement)
	}
	pause := sqliteLaunchFenceControl(replaced, "control:sqlite-pause-backoff-before-claim", application.ControlPause,
		application.ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
	if _, err := orchestrator.Control(ctx, access, pause); err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * time.Second)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-paused-backoff"); processErr != nil || result.Processed {
		t.Fatalf("paused backoff claimed: result=%+v err=%v", result, processErr)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery rejected paused queued replacement: %v cause=%v", err, errors.Unwrap(err))
	}
	persisted, _ := restarted.GetGoal(ctx, replaced.Goal.Ref())
	restartedOrchestrator := newSQLiteLaunchFenceOrchestrator(t, restarted, clock, ids, agent)
	resume := sqliteLaunchFenceControl(persisted, "control:sqlite-pause-backoff-resume", application.ControlResume,
		application.ControlTargetWorkItem, item.Ref(), goal.ExecutionRef{})
	if _, err := restartedOrchestrator.Control(ctx, access, resume); err != nil {
		t.Fatal(err)
	}
	if result, processErr := restartedOrchestrator.ProcessNext(ctx, "worker:sqlite-resumed-backoff"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("resumed backoff: result=%+v err=%v", result, processErr)
	}
	launched, _ := restarted.GetGoal(ctx, replaced.Goal.Ref())
	if current := sqliteBoundExecution(t, launched, item.Ref()); current.State != application.ExecutionRunning {
		t.Fatalf("resumed replacement execution=%s", current.State)
	}
	if agent.launchCount() != 2 {
		t.Fatalf("provider launches=%d, want 2", agent.launchCount())
	}
	assertSQLiteLaunchFrontierEvents(t, restarted, replacement.Ref)
}

func sqliteLaunchFenceControl(
	record application.GoalRecord,
	requestRef string,
	operation application.ControlOperation,
	target application.ControlTarget,
	itemRef goal.WorkItemRef,
	executionRef goal.ExecutionRef,
) application.ControlRequest {
	request := application.ControlRequest{
		RequestRef: requestRef, Operation: operation, Target: target, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		Reason: "verify exact launch preparation fence",
	}
	if itemRef.String() != "" {
		item, _ := record.Goal.WorkItem(itemRef)
		request.WorkItemRef = itemRef
		request.ExpectedWorkItemRevision = item.Revision()
	}
	if executionRef.String() != "" {
		for _, execution := range record.Executions {
			if execution.Ref == executionRef {
				request.ExecutionRef = executionRef
				request.ExpectedExecutionAttempt = execution.AttemptNo
				break
			}
		}
	}
	return request
}

func sqliteBoundExecution(t *testing.T, record application.GoalRecord, itemRef goal.WorkItemRef) application.ExecutionRecord {
	t.Helper()
	item, found := record.Goal.WorkItem(itemRef)
	if !found {
		t.Fatal("WorkItem missing")
	}
	ref, bound := item.Execution()
	if !bound {
		t.Fatal("WorkItem execution missing")
	}
	for _, execution := range record.Executions {
		if execution.Ref == ref {
			return execution
		}
	}
	t.Fatal("bound execution record missing")
	return application.ExecutionRecord{}
}

func assertSQLiteLaunchFrontierEvents(t *testing.T, repository *Repository, executionRef goal.ExecutionRef) {
	t.Helper()
	var queued, dispatching int
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM events WHERE execution_ref = ? AND kind = 'execution.queued'`, executionRef.String()).Scan(&queued); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`
SELECT COUNT(*) FROM events WHERE execution_ref = ? AND kind = 'execution.dispatching'`, executionRef.String()).Scan(&dispatching); err != nil {
		t.Fatal(err)
	}
	if queued != 1 || dispatching != 1 {
		t.Fatalf("launch frontier events queued=%d dispatching=%d", queued, dispatching)
	}
}

type sqliteLaunchFenceAgent struct {
	mu           sync.Mutex
	clock        *restartClock
	failLaunches int
	launches     int
}

type sqliteLaunchFenceDefinitelyUnapplied struct{}

func (sqliteLaunchFenceDefinitelyUnapplied) Error() string {
	return "sqlite launch fence permanent failure"
}
func (sqliteLaunchFenceDefinitelyUnapplied) DefinitelyNotApplied() bool { return true }

func (*sqliteLaunchFenceAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteMultiControlCapabilities(), nil
}

func (agent *sqliteLaunchFenceAgent) Launch(
	_ context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	agent.launches++
	if agent.failLaunches > 0 {
		agent.failLaunches--
		agent.clock.Advance(time.Nanosecond)
		agent.mu.Unlock()
		return ports.AgentLaunchReceipt{}, sqliteLaunchFenceDefinitelyUnapplied{}
	}
	agent.mu.Unlock()
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, LaunchActionFence: request.EffectAuthority.ActionFence,
		SpecHash:    request.SpecHash,
		ProviderRef: "provider:sqlite-multi", ModelRef: "model:sqlite-multi", AgentRef: "agent:sqlite-multi",
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:sqlite-launch:" + request.ExecutionRef.String(), AcceptedAt: agent.clock.Now(),
	}, nil
}

func (*sqliteLaunchFenceAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{}, errors.New("sqlite launch fence observation unused")
}

func (agent *sqliteLaunchFenceAgent) ObserveAgent(
	ctx context.Context,
	request ports.AgentObserveRequest,
) (ports.AgentObservation, error) {
	return agent.Observe(ctx, request.ExecutionRef)
}

func (*sqliteLaunchFenceAgent) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (*sqliteLaunchFenceAgent) Stop(context.Context, ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	return ports.AgentStopReceipt{}, errors.New("sqlite launch fence stop unused")
}

func (agent *sqliteLaunchFenceAgent) launchCount() int {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return agent.launches
}

func newSQLiteLaunchFenceOrchestrator(
	t *testing.T,
	repository *Repository,
	clock *restartClock,
	ids *restartIDs,
	agent *sqliteLaunchFenceAgent,
) *application.Orchestrator {
	t.Helper()
	_, fuentes := prepararCapacidadSQLiteV15(t, repository, clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository, Launcher: agent, Observer: agent, Controller: agent,
		Artifacts: restartArtifacts{}, Clock: clock, IDs: ids, MaxOutputBytes: 4096,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
		AgentCapabilities: sqliteMultiControlCapabilities(), ClaimLease: time.Minute,
		CapacitySources: fuentes, CapacityObservationWait: time.Second,
		DirectorLeaseDuration: time.Minute, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}
