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

func TestSQLiteForcedStopSupersessionIsAtomicConcurrentAndRestartSafe(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 18, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteEscalationAgent{clock: clock}
	_, fuentes := prepararCapacidadSQLiteV15(t, repository, clock, 1_000)
	newOrchestrator := func(state application.StateRepository, access application.AccessRepository) *application.Orchestrator {
		orchestrator, err := application.New(application.Dependencies{
			State: state, Access: access, Launcher: agent, Observer: agent, Controller: agent,
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
	orchestrator := newOrchestrator(repository, repository)
	actor, _ := goal.NewActorRef("actor:sqlite-forced-escalation")
	project, _ := goal.NewProjectRef("project:sqlite-forced-escalation")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-forced-escalation", Statement: "escalate one pending cooperative stop", Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-escalation-launch"); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]
	cooperative := sqliteExactStopRequest(
		running, item, execution, "control:sqlite-cooperative-owner", ports.AgentStopCooperative,
	)
	first, err := orchestrator.Control(ctx, access, cooperative)
	if err != nil || first.Control.Status != application.ControlRequested {
		t.Fatalf("cooperative request: result=%+v err=%v", first, err)
	}
	current, err := repository.GetGoal(ctx, running.Goal.Ref())
	sqliteTestNoError(t, err)
	item, _ = current.Goal.WorkItem(item.Ref())
	forced := sqliteExactStopRequest(
		current, item, execution, "control:sqlite-forced-owner", ports.AgentStopForced,
	)
	type outcome struct {
		result application.ControlResult
		err    error
	}
	results := make(chan outcome, 2)
	start := make(chan struct{})
	for range 2 {
		go func() {
			<-start
			result, controlErr := orchestrator.Control(ctx, access, forced)
			results <- outcome{result: result, err: controlErr}
		}()
	}
	close(start)
	created, replayed := 0, 0
	var forcedRef string
	for range 2 {
		outcome := <-results
		if outcome.err != nil {
			t.Fatalf("concurrent forced control: %v", outcome.err)
		}
		if outcome.result.Created {
			created++
		} else {
			replayed++
		}
		if forcedRef == "" {
			forcedRef = outcome.result.Control.Ref
		} else if forcedRef != outcome.result.Control.Ref {
			t.Fatalf("concurrent refs=%q/%q", forcedRef, outcome.result.Control.Ref)
		}
	}
	if created != 1 || replayed != 1 {
		t.Fatalf("concurrent forced created/replayed=%d/%d", created, replayed)
	}

	transferred, err := repository.GetGoal(ctx, running.Goal.Ref())
	if err != nil || len(transferred.Controls) != 2 {
		t.Fatalf("transferred controls=%+v err=%v", transferred.Controls, err)
	}
	old, next := sqliteControlsByRequest(t, transferred, cooperative.RequestRef, forced.RequestRef)
	if old.Status != application.ControlSuperseded || old.SupersededByControlRef != next.Ref ||
		next.Status != application.ControlRequested || next.SupersedesControlRef != old.Ref {
		t.Fatalf("lineage old=%+v next=%+v", old, next)
	}
	var forcedIntent application.EffectIntent
	for _, intent := range transferred.EffectIntents {
		if intent.ActionRef == "action:stop:"+next.Ref+":"+execution.Ref.String() {
			forcedIntent = intent
		}
	}
	approved, err := orchestrator.DecideEffect(ctx, access, application.DecideEffectRequest{
		RequestRef: "approval:sqlite-forced-owner", GoalRef: transferred.Goal.Ref(),
		IntentRef: forcedIntent.Ref, ExpectedIntentDigest: forcedIntent.Digest,
		Decision: application.EffectApproved, Reason: "owner forced stop approval",
	})
	if err != nil || !approved.Created {
		t.Fatalf("forced stop approval=%+v intent=%+v err=%v", approved, forcedIntent, err)
	}
	var active, oldRetirements int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM outbox
WHERE kind = 'stop_agent' AND completed_at IS NULL`).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts
WHERE action_ref = ? AND error_code = 'application.action_retired' AND effect_receipt_ref IS NULL`,
		"action:stop:"+old.Ref+":"+execution.Ref.String(),
	).Scan(&oldRetirements); err != nil {
		t.Fatal(err)
	}
	if active != 1 || oldRetirements != 1 {
		t.Fatalf("atomic outbox active=%d old_retirements=%d", active, oldRetirements)
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("recovery before restart: %v cause=%v", err, errors.Unwrap(err))
	}

	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	restartedOrchestrator := newOrchestrator(restarted, restarted)
	if result, processErr := restartedOrchestrator.ProcessNext(ctx, "worker:sqlite-forced-after-restart"); processErr != nil || !result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("forced after restart: result=%+v err=%v", result, processErr)
	}
	settled, err := restarted.GetGoal(ctx, running.Goal.Ref())
	sqliteTestNoError(t, err)
	old, next = sqliteControlsByRequest(t, settled, cooperative.RequestRef, forced.RequestRef)
	stopped, _ := sqliteExecutionByRef(settled.Executions, execution.Ref)
	requests, physical := agent.snapshot()
	if old.Status != application.ControlSuperseded || next.Status != application.ControlConfirmed ||
		stopped.State != application.ExecutionStopped || len(requests) != 1 || physical != 1 {
		t.Fatalf("settled old=%s next=%s execution=%s requests=%d physical=%d",
			old.Status, next.Status, stopped.State, len(requests), physical)
	}
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery after settlement: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestSQLiteForcedStopRejectsQuarantinedCooperativeOwnerWithoutPartialWrite(t *testing.T) {
	ctx := context.Background()
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 19, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteEscalationAgent{clock: clock}
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
	actor, _ := goal.NewActorRef("actor:sqlite-inactive-supersession")
	project, _ := goal.NewProjectRef("project:sqlite-inactive-supersession")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-inactive-supersession",
		Statement:  "reject escalation after cooperative action is quarantined", Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-inactive-launch"); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]
	cooperative := sqliteExactStopRequest(
		running, item, execution, "control:sqlite-inactive-cooperative", ports.AgentStopCooperative,
	)
	old, err := orchestrator.Control(ctx, access, cooperative)
	if err != nil || old.Control.Status != application.ControlRequested {
		t.Fatalf("cooperative request: result=%+v err=%v", old, err)
	}
	claim, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:sqlite-quarantine-cooperative", Token: "claim:sqlite-quarantine-cooperative",
		LeaseDuration: time.Minute, Capabilities: sqliteMultiControlCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found || claim.Action.Kind != application.ActionStopAgent ||
		claim.Action.ControlRef != old.Control.Ref {
		t.Fatalf("claim cooperative stop: claim=%+v found=%v err=%v", claim, found, err)
	}
	quarantinedAt := clock.Now()
	if err := repository.QuarantineAction(ctx, application.ActionQuarantinedState{
		Claim: claim, ErrorCode: "test.cooperative_stop_quarantined", OperationAt: quarantinedAt,
		Event: application.EventRecord{
			Ref: "event:action-quarantined:" + claim.Action.Ref, Kind: "action.quarantined",
			GoalRef: claim.Action.GoalRef, WorkItemRef: claim.Action.WorkItemRef,
			ExecutionRef: claim.Action.ExecutionRef, OccurredAt: quarantinedAt,
		},
	}); err != nil {
		t.Fatalf("quarantine cooperative stop: %v", err)
	}

	current, err := repository.GetGoal(ctx, running.Goal.Ref())
	sqliteTestNoError(t, err)
	item, _ = current.Goal.WorkItem(item.Ref())
	forced := sqliteExactStopRequest(
		current, item, execution, "control:sqlite-inactive-forced", ports.AgentStopForced,
	)
	var controlsBefore, actionsBefore, eventsBefore int
	for query, target := range map[string]*int{
		`SELECT COUNT(*) FROM controls`: &controlsBefore,
		`SELECT COUNT(*) FROM outbox`:   &actionsBefore,
		`SELECT COUNT(*) FROM events`:   &eventsBefore,
	} {
		if err := repository.db.QueryRow(query).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := orchestrator.Control(ctx, access, forced); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("forced escalation over quarantined owner error=%v", err)
	}
	var controlsAfter, actionsAfter, eventsAfter, forcedRows, quarantinedRows int
	for query, target := range map[string]*int{
		`SELECT COUNT(*) FROM controls`: &controlsAfter,
		`SELECT COUNT(*) FROM outbox`:   &actionsAfter,
		`SELECT COUNT(*) FROM events`:   &eventsAfter,
	} {
		if err := repository.db.QueryRow(query).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM controls WHERE request_ref = ?`, forced.RequestRef).Scan(&forcedRows); err != nil {
		t.Fatal(err)
	}
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM outbox
WHERE ref = ? AND completed_at IS NOT NULL AND quarantined_at IS NOT NULL`, claim.Action.Ref).Scan(&quarantinedRows); err != nil {
		t.Fatal(err)
	}
	if controlsAfter != controlsBefore || actionsAfter != actionsBefore || eventsAfter != eventsBefore ||
		forcedRows != 0 || quarantinedRows != 1 {
		t.Fatalf("partial supersession controls=%d/%d actions=%d/%d events=%d/%d forced=%d quarantined=%d",
			controlsBefore, controlsAfter, actionsBefore, actionsAfter, eventsBefore, eventsAfter,
			forcedRows, quarantinedRows)
	}
	persisted, err := repository.GetGoal(ctx, running.Goal.Ref())
	if err != nil || len(persisted.Controls) != 1 || persisted.Controls[0].Status != application.ControlRequested {
		t.Fatalf("old owner changed: controls=%+v err=%v", persisted.Controls, err)
	}
}

func sqliteExactStopRequest(
	record application.GoalRecord,
	item goal.WorkItem,
	execution application.ExecutionRecord,
	requestRef string,
	mode ports.AgentStopMode,
) application.ControlRequest {
	return application.ControlRequest{
		RequestRef: requestRef, Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: mode, Reason: "exact SQLite forced escalation",
	}
}

func sqliteControlsByRequest(
	t *testing.T,
	record application.GoalRecord,
	oldRequestRef, nextRequestRef string,
) (application.ControlRecord, application.ControlRecord) {
	t.Helper()
	var old, next application.ControlRecord
	for _, control := range record.Controls {
		switch control.RequestRef {
		case oldRequestRef:
			old = control
		case nextRequestRef:
			next = control
		}
	}
	if old.Ref == "" || next.Ref == "" {
		t.Fatalf("control lineage missing: %+v", record.Controls)
	}
	return old, next
}

type sqliteEscalationAgent struct {
	mu       sync.Mutex
	clock    *restartClock
	requests []ports.AgentStopRequest
	physical int
}

func (*sqliteEscalationAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteMultiControlCapabilities(), nil
}

func (agent *sqliteEscalationAgent) Launch(
	_ context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
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

func (*sqliteEscalationAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{}, nil
}

func (agent *sqliteEscalationAgent) ObserveAgent(
	ctx context.Context,
	request ports.AgentObserveRequest,
) (ports.AgentObservation, error) {
	return agent.Observe(ctx, request.ExecutionRef)
}

func (*sqliteEscalationAgent) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (agent *sqliteEscalationAgent) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	agent.requests = append(agent.requests, request)
	status := ports.AgentStopPending
	if request.Mode == ports.AgentStopForced {
		status = ports.AgentStopped
		agent.physical++
	}
	receipt := ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, StopEffectAttemptRef: request.StopEffectAttemptRef,
		StopActionFence: request.StopActionFence, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: status,
	}
	if status == ports.AgentStopped {
		receipt.ReceiptRef = "receipt:sqlite-forced:" + request.ExecutionRef.String()
		receipt.ConfirmedAt = agent.clock.Now()
	}
	return receipt, nil
}

func (agent *sqliteEscalationAgent) snapshot() ([]ports.AgentStopRequest, int) {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return append([]ports.AgentStopRequest(nil), agent.requests...), agent.physical
}
