package sqlite

import (
	"context"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteTerminalStopSettlesAfterRestartWithReplacementAgentRouting(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 19, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteTerminalStopAgent{clock: clock}
	originalCapabilities := sqliteMultiControlCapabilities()
	orchestrator := newTerminalStopRoutingOrchestrator(
		t, repository, repository, clock, ids, agent, originalCapabilities,
	)
	actor, _ := goal.NewActorRef("actor:sqlite-terminal-routing")
	project, _ := goal.NewProjectRef("project:sqlite-terminal-routing")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-terminal-routing",
		Statement:  "settle terminal stop after adapter replacement",
		Confirm:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-terminal-routing-launch"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]

	gate := &sqliteGetGoalGate{
		StateRepository: repository, entered: make(chan struct{}), release: make(chan struct{}),
	}
	processor := newTerminalStopRoutingOrchestrator(
		t, gate, repository, clock, ids, agent, originalCapabilities,
	)
	type processOutcome struct {
		result application.ProcessResult
		err    error
	}
	processed := make(chan processOutcome, 1)
	go func() {
		result, processErr := processor.ProcessNext(
			context.WithValue(ctx, sqliteGetGoalGateKey{}, true),
			"worker:sqlite-terminal-routing-observe",
		)
		processed <- processOutcome{result: result, err: processErr}
	}()
	select {
	case <-gate.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("observation was not claimed before stop request")
	}
	requested, err := orchestrator.Control(ctx, access, application.ControlRequest{
		RequestRef: "control:sqlite-terminal-routing", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "completion wins before replacement worker settles stop",
	})
	if err != nil || requested.Control.Status != application.ControlRequested {
		close(gate.release)
		t.Fatalf("request stop: result=%+v err=%v", requested, err)
	}
	close(gate.release)
	select {
	case outcome := <-processed:
		if outcome.err != nil || !outcome.result.Processed || outcome.result.Action != application.ActionObserveAgent {
			t.Fatalf("complete observation: result=%+v err=%v", outcome.result, outcome.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("observation did not reach terminal state")
	}
	terminal, err := repository.GetGoal(ctx, running.Goal.Ref())
	if err != nil || terminal.Goal.State() != goal.GoalStateSucceeded ||
		terminal.Executions[0].State != application.ExecutionSucceeded ||
		terminal.Controls[0].Status != application.ControlRequested {
		t.Fatalf("terminal pending stop: record=%+v err=%v", terminal, err)
	}

	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	replacementCapabilities := ports.AgentCapabilities{
		ProviderRef: "provider:replacement", ModelRef: "model:replacement", AgentRef: "agent:replacement",
	}
	restartedOrchestrator := newTerminalStopRoutingOrchestrator(
		t, restarted, restarted, clock, ids, agent, replacementCapabilities,
	)
	result, processErr := restartedOrchestrator.ProcessNext(ctx, "worker:replacement-terminal-settlement")
	if processErr != nil || !result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("replacement worker terminal stop: result=%+v err=%v", result, processErr)
	}
	settled, err := restarted.GetGoal(ctx, running.Goal.Ref())
	if err != nil || settled.Goal.State() != goal.GoalStateSucceeded ||
		settled.Executions[0].State != application.ExecutionSucceeded ||
		settled.Controls[0].Status != application.ControlConfirmed || agent.stopCount() != 0 {
		t.Fatalf("local terminal settlement: record=%+v physical_stops=%d err=%v", settled, agent.stopCount(), err)
	}
	var receipts int
	if err := restarted.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts
WHERE action_ref = ? AND effect_status = 'already_completed'`,
		"action:stop:"+requested.Control.Ref+":"+execution.Ref.String(),
	).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 {
		t.Fatalf("terminal effect receipts=%d, want 1", receipts)
	}
	if replay, replayErr := restartedOrchestrator.ProcessNext(ctx, "worker:replacement-terminal-replay"); replayErr != nil || replay.Processed {
		t.Fatalf("terminal stop replay: result=%+v err=%v", replay, replayErr)
	}
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery rejected replacement settlement: %v", err)
	}
}

func TestSQLiteRunningStopRetainsExactAgentRouting(t *testing.T) {
	ctx := context.Background()
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 19, 30, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteTerminalStopAgent{clock: clock}
	capabilities := sqliteMultiControlCapabilities()
	orchestrator := newTerminalStopRoutingOrchestrator(
		t, repository, repository, clock, ids, agent, capabilities,
	)
	actor, _ := goal.NewActorRef("actor:sqlite-running-stop-routing")
	project, _ := goal.NewProjectRef("project:sqlite-running-stop-routing")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-running-stop-routing", Statement: "keep live stop on original adapter", Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-running-stop-launch"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil {
		t.Fatal(err)
	}
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]
	if _, err := orchestrator.Control(ctx, access, application.ControlRequest{
		RequestRef: "control:sqlite-running-stop-routing", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "live effect stays on exact provider identity",
	}); err != nil {
		t.Fatal(err)
	}
	replacement := ports.AgentCapabilities{
		ProviderRef: "provider:replacement-live", ModelRef: "model:replacement-live",
		AgentRef: "agent:replacement-live", Unrestricted: true,
	}
	if claim, found, claimErr := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:replacement-live", Token: "claim:replacement-live",
		LeaseDuration: time.Minute, Capabilities: replacement,
	}); claimErr != nil || found {
		t.Fatalf("replacement claimed running stop: found=%v claim=%+v err=%v", found, claim, claimErr)
	}
	claim, found, claimErr := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:original-live", Token: "claim:original-live",
		LeaseDuration: time.Minute, Capabilities: capabilities,
	})
	if claimErr != nil || !found || claim.Action.Kind != application.ActionStopAgent {
		t.Fatalf("original adapter did not claim running stop: found=%v claim=%+v err=%v", found, claim, claimErr)
	}
}

func newTerminalStopRoutingOrchestrator(
	t *testing.T,
	state application.StateRepository,
	access application.AccessRepository,
	clock *restartClock,
	ids *restartIDs,
	agent *sqliteTerminalStopAgent,
	capabilities ports.AgentCapabilities,
) *application.Orchestrator {
	t.Helper()
	orchestrator, err := application.New(application.Dependencies{
		State: state, Access: access, Launcher: agent, Observer: agent, Controller: agent,
		Artifacts: leaseAdvancingArtifacts{clock: clock}, Clock: clock, IDs: ids,
		MaxOutputBytes: 4096, MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
		AgentCapabilities: capabilities, ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
		ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return orchestrator
}
