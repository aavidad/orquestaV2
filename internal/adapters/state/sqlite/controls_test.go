package sqlite

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestSQLiteControlsRestartAndConcurrentCAS(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 14, 12, 0, 0, 123456789, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	orchestrator := newRestartOrchestrator(t, repository, clock, ids, &restartAgent{clock: clock})
	actor, _ := goal.NewActorRef("actor:sqlite-control-cas")
	project, _ := goal.NewProjectRef("project:sqlite-control-cas")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-control-cas", Statement: "verify concurrent controls", Confirm: true,
	})
	sqliteTestNoError(t, err)
	record := submitted.Record
	request := func(ref string) application.ControlRequest {
		return application.ControlRequest{
			RequestRef: ref, Operation: application.ControlPause, Target: application.ControlTargetGoal,
			GoalRef: record.Goal.Ref(), ExpectedGoalRevision: record.Goal.Revision(),
			ExpectedPlanGeneration:    record.Goal.PlanGeneration(),
			ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
			Reason: "concurrent CAS proof",
		}
	}
	type outcome struct {
		result application.ControlResult
		err    error
	}
	results := make(chan outcome, 2)
	var start sync.WaitGroup
	start.Add(1)
	for _, ref := range []string{"request:sqlite-control-a", "request:sqlite-control-b"} {
		ref := ref
		go func() {
			start.Wait()
			result, err := orchestrator.Control(ctx, access, request(ref))
			results <- outcome{result: result, err: err}
		}()
	}
	start.Done()
	created, rejected := 0, 0
	for range 2 {
		result := <-results
		if result.err == nil && result.result.Created {
			created++
		} else if result.err != nil {
			rejected++
		}
	}
	if created != 1 || rejected != 1 {
		t.Fatalf("concurrent controls created=%d rejected=%d", created, rejected)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	persisted, err := restarted.GetGoal(ctx, record.Goal.Ref())
	sqliteTestNoError(t, err)
	if !persisted.Goal.Paused() || len(persisted.Controls) != 1 ||
		persisted.Controls[0].Status != application.ControlConfirmed {
		t.Fatalf("restart lost winning control: paused=%v controls=%+v", persisted.Goal.Paused(), persisted.Controls)
	}
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery rejected control state: %v", err)
	}
}

func TestSQLiteStopActionClaimsAfterCompletionAndConsumesAlreadyCompleted(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 16, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteTerminalStopAgent{clock: clock}
	capabilities := sqliteMultiControlCapabilities()
	_, fuentes := prepararCapacidadSQLiteV15(t, repository, clock, 1_000)
	newOrchestrator := func(state application.StateRepository) *application.Orchestrator {
		orchestrator, err := application.New(application.Dependencies{
			State: state, Access: repository, Launcher: agent, Observer: agent, Controller: agent,
			Artifacts: leaseAdvancingArtifacts{clock: clock}, Clock: clock, IDs: ids,
			MaxOutputBytes: 4096, MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
			MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
			AgentCapabilities: capabilities, ClaimLease: time.Minute, DirectorLeaseDuration: time.Minute,
			CapacitySources: fuentes, CapacityObservationWait: time.Second,
			ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		})
		sqliteTestNoError(t, err)
		return orchestrator
	}
	orchestrator := newOrchestrator(repository)
	actor, _ := goal.NewActorRef("actor:sqlite-terminal-stop")
	project, _ := goal.NewProjectRef("project:sqlite-terminal-stop")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-terminal-stop", Statement: "complete before pending stop is claimed", Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-terminal-launch"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item := running.Goal.WorkItems()[0]
	execution := running.Executions[0]

	gate := &sqliteGetGoalGate{
		StateRepository: repository,
		entered:         make(chan struct{}),
		release:         make(chan struct{}),
	}
	processor := newOrchestrator(gate)
	type processOutcome struct {
		result application.ProcessResult
		err    error
	}
	processed := make(chan processOutcome, 1)
	go func() {
		result, processErr := processor.ProcessNext(
			context.WithValue(ctx, sqliteGetGoalGateKey{}, true),
			"worker:sqlite-terminal-observe",
		)
		processed <- processOutcome{result: result, err: processErr}
	}()
	select {
	case <-gate.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("observation was not claimed before the stop request")
	}
	request := application.ControlRequest{
		RequestRef: "control:sqlite-terminal-stop", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "completion owns the terminal transition",
	}
	requested, err := orchestrator.Control(ctx, access, request)
	if err != nil || requested.Control.Status != application.ControlRequested {
		close(gate.release)
		t.Fatalf("request stop: result=%+v err=%v", requested, err)
	}
	originalRef, refErr := identity.NewPrincipalRef(actor.String())
	if refErr != nil {
		t.Fatal(refErr)
	}
	original, principalErr := identity.NewPrincipal(originalRef, actor, identity.PrincipalKindHuman, "test")
	if principalErr != nil {
		t.Fatal(principalErr)
	}
	second := testPrincipal(t, "principal:sqlite-terminal-second-owner", "actor:sqlite-terminal-second-owner", identity.PrincipalKindHuman)
	grantTestMembership(t, repository, original, second, project, identity.RoleProjectOwner,
		"membership:sqlite-terminal-second-owner", clock.Now())
	authorization := authorizeTest(t, repository, second, project,
		identity.PermissionProjectMembershipManage, original.Ref.String(),
		"authorization:sqlite-terminal-revoke-owner", clock.Now())
	revoke := testRevokeRequest(t, "membership:sqlite-terminal-revoke-owner", second, original.Ref, project, 1, clock.Now())
	if _, _, changed, revokeErr := repository.RevokeMembership(ctx, application.MembershipRevokeState{
		AuthorizationReceipt: authorization, Request: revoke,
	}); revokeErr != nil || !changed {
		close(gate.release)
		t.Fatalf("revoke terminal stop authority changed=%v err=%v", changed, revokeErr)
	}
	close(gate.release)
	select {
	case outcome := <-processed:
		if outcome.err != nil || !outcome.result.Processed || outcome.result.Action != application.ActionObserveAgent {
			t.Fatalf("complete leased observation: result=%+v err=%v", outcome.result, outcome.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("leased completion did not finish")
	}
	completed, err := repository.GetGoal(ctx, running.Goal.Ref())
	if err != nil || completed.Goal.State() != goal.GoalStateSucceeded ||
		completed.Executions[0].State != application.ExecutionSucceeded ||
		completed.Controls[0].Status != application.ControlRequested {
		t.Fatalf("completion did not win: Goal=%s executions=%+v controls=%+v err=%v",
			completed.Goal.State(), completed.Executions, completed.Controls, err)
	}
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-terminal-stop"); processErr != nil ||
		!result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("claim terminal stop: result=%+v err=%v", result, processErr)
	}
	settled, err := repository.GetGoal(ctx, running.Goal.Ref())
	if err != nil || settled.Goal.State() != goal.GoalStateSucceeded ||
		settled.Executions[0].State != application.ExecutionSucceeded ||
		settled.Controls[0].Status != application.ControlConfirmed || agent.stopCount() != 0 {
		t.Fatalf("terminal stop did not settle locally: Goal=%s executions=%+v controls=%+v physical_stops=%d err=%v",
			settled.Goal.State(), settled.Executions, settled.Controls, agent.stopCount(), err)
	}
	var stopReceipt *application.ActionConsumptionReceipt
	for index := range settled.ConsumptionReceipts {
		candidate := &settled.ConsumptionReceipts[index]
		if candidate.Kind == application.ActionStopAgent && candidate.ExecutionRef == execution.Ref {
			stopReceipt = candidate
			break
		}
	}
	if stopReceipt == nil || stopReceipt.Outcome != application.ActionConsumedCompleted ||
		stopReceipt.EffectReceiptRef != "" {
		t.Fatalf("terminal stop consumption receipt=%+v", stopReceipt)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	persisted, err := restarted.GetGoal(ctx, running.Goal.Ref())
	if err != nil || persisted.Controls[0].Status != application.ControlConfirmed ||
		persisted.Executions[0].State != application.ExecutionSucceeded {
		t.Fatalf("terminal stop restart state=%+v err=%v", persisted, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery rejected terminal stop settlement: %v", err)
	}
}

func TestSQLiteBackupExcludesCodexPrivateProcessJournal(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	sentinel := []byte("codex-private-process-journal:must-not-enter-sqlite-backup")
	journal := filepath.Join(filepath.Dir(path), "codex-private-process-journal.json")
	if err := os.WriteFile(journal, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}
	recovery, backupRoot, _ := newV09TestRecovery(t, repository, time.Now().UTC(), nil)
	receipt, err := recovery.CreateBackup(ctx)
	sqliteTestNoError(t, err)
	if _, err := recovery.VerifyBackup(ctx, receipt.Ref); err != nil {
		t.Fatal(err)
	}
	err = filepath.Walk(backupRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if info.Name() == filepath.Base(journal) {
			t.Fatalf("private process journal copied into backup: %s", path)
		}
		content, readErr := os.ReadFile(path)
		if readErr == nil && bytes.Contains(content, sentinel) {
			t.Fatalf("private process journal content leaked into backup: %s", path)
		}
		return readErr
	})
	sqliteTestNoError(t, err)
}

func TestSQLiteGoalCancelPersistsOneStopReceiptPerExecutionAcrossRestart(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 15, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteMultiControlAgent{clock: clock}
	orchestrator := newSQLiteMultiControlOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:sqlite-multi-control")
	project, _ := goal.NewProjectRef("project:sqlite-multi-control")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:sqlite-multi-control", Statement: "cancel every independent execution", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:sqlite-multi-control", Key: "phase:sqlite-multi-control",
				TemplateRef: "phase-template:sqlite-multi-control",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "first", Objective: "first live execution", Phase: "phase:sqlite-multi-control", Role: "role:worker", WriteSet: []string{"internal/first"}, CouncilPolicy: council.PolicyRequired, RequiredTests: sqliteRequiredTestSpecs("required-test:sqlite-control-first"), OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "second", Objective: "second live execution", Phase: "phase:sqlite-multi-control", Role: "role:worker", WriteSet: []string{"internal/second"}, CouncilPolicy: council.PolicyRequired, RequiredTests: sqliteRequiredTestSpecs("required-test:sqlite-control-second"), OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteWorkspaceLaunches(t, orchestrator, "worker:sqlite-multi-launch", 2)
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	if err != nil || len(running.Executions) != 2 {
		t.Fatalf("running executions=%+v err=%v", running.Executions, err)
	}
	request := application.ControlRequest{
		RequestRef: "control:sqlite-multi-cancel", Operation: application.ControlCancel,
		Target: application.ControlTargetGoal, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		Reason: "persist one exact stop receipt per execution",
	}
	requested, err := orchestrator.Control(ctx, access, request)
	if err != nil || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request cancel: result=%+v err=%v", requested, err)
	}
	for index := 0; index < 2; index++ {
		if result, processErr := orchestrator.ProcessNext(ctx, "worker:sqlite-multi-stop"); processErr != nil ||
			!result.Processed || result.Action != application.ActionStopAgent {
			t.Fatalf("stop %d: result=%+v err=%v cause=%v", index+1, result, processErr, errors.Unwrap(processErr))
		}
	}
	closed, err := repository.GetGoal(ctx, running.Goal.Ref())
	if err != nil || closed.Goal.State() != goal.GoalStateCanceled || len(closed.Controls) != 1 ||
		closed.Controls[0].Status != application.ControlConfirmed ||
		closed.Controls[0].ReceiptRef != "receipt:"+closed.Controls[0].Ref {
		t.Fatalf("closed multi cancel: Goal=%s controls=%+v err=%v", closed.Goal.State(), closed.Controls, err)
	}
	assertSQLiteExactStopReceipts(t, closed, running.Executions)

	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	persisted, err := restarted.GetGoal(ctx, running.Goal.Ref())
	if err != nil || persisted.Goal.State() != goal.GoalStateCanceled {
		t.Fatalf("restart Goal=%s err=%v", persisted.Goal.State(), err)
	}
	assertSQLiteExactStopReceipts(t, persisted, running.Executions)
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("recovery rejected exact stop receipts: %v", err)
	}
}

func assertSQLiteExactStopReceipts(
	t *testing.T,
	record application.GoalRecord,
	executions []application.ExecutionRecord,
) {
	t.Helper()
	effects := make(map[goal.ExecutionRef]application.EffectReceipt)
	for _, receipt := range record.EffectReceipts {
		if receipt.Status == application.EffectStatusStopped {
			effects[receipt.Subject.ExecutionRef] = receipt
		}
	}
	if len(effects) != len(executions) {
		t.Fatalf("exact stop receipts=%d want=%d: %+v", len(effects), len(executions), effects)
	}
	for _, execution := range executions {
		receipt, found := effects[execution.Ref]
		if !found || receipt.ExternalRef != "receipt:sqlite-stop:"+execution.Ref.String() ||
			receipt.ConfirmedAt.IsZero() {
			t.Fatalf("receipt for %s=%+v found=%v", execution.Ref, receipt, found)
		}
	}
}

type sqliteMultiControlAgent struct {
	clock *restartClock
}

type sqliteGetGoalGateKey struct{}

type sqliteGetGoalGate struct {
	application.StateRepository
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (state *sqliteGetGoalGate) GetGoal(
	ctx context.Context,
	ref goal.GoalRef,
) (application.GoalRecord, error) {
	if gated, _ := ctx.Value(sqliteGetGoalGateKey{}).(bool); gated {
		state.once.Do(func() { close(state.entered) })
		select {
		case <-state.release:
		case <-ctx.Done():
			return application.GoalRecord{}, ctx.Err()
		}
	}
	return state.StateRepository.GetGoal(ctx, ref)
}

type sqliteTerminalStopAgent struct {
	mu       sync.Mutex
	clock    *restartClock
	specHash string
	stops    int
}

func (*sqliteTerminalStopAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteMultiControlCapabilities(), nil
}

func (agent *sqliteTerminalStopAgent) Launch(
	_ context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	agent.mu.Lock()
	agent.specHash = request.SpecHash
	agent.mu.Unlock()
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:sqlite-multi", ModelRef: "model:sqlite-multi", AgentRef: "agent:sqlite-multi",
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:sqlite-launch:" + request.ExecutionRef.String(), AcceptedAt: agent.clock.Now(),
	}, nil
}

func (agent *sqliteTerminalStopAgent) Observe(
	_ context.Context,
	executionRef goal.ExecutionRef,
) (ports.AgentObservation, error) {
	agent.mu.Lock()
	specHash := agent.specHash
	agent.mu.Unlock()
	return ports.AgentObservation{
		ExecutionRef: executionRef, SpecHash: specHash, Status: ports.AgentCompleted,
		MediaType: "text/plain", Content: []byte("completion wins before stop claim"),
		Usage:      governance.ResourceUsage{Quality: governance.UsageQualityUnknown},
		ObservedAt: agent.clock.Now(),
	}, nil
}

func (*sqliteTerminalStopAgent) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true}, nil
}

func (agent *sqliteTerminalStopAgent) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	agent.mu.Lock()
	agent.stops++
	agent.mu.Unlock()
	return ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: ports.AgentStopped, ReceiptRef: "receipt:unexpected-physical-stop", ConfirmedAt: agent.clock.Now(),
	}, nil
}

func (agent *sqliteTerminalStopAgent) stopCount() int {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return agent.stops
}

func (*sqliteMultiControlAgent) Capabilities(context.Context) (ports.AgentCapabilities, error) {
	return sqliteMultiControlCapabilities(), nil
}

func (agent *sqliteMultiControlAgent) Launch(
	_ context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	return ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: "provider:sqlite-multi", ModelRef: "model:sqlite-multi", AgentRef: "agent:sqlite-multi",
		ExternalRef: "external:" + request.ExecutionRef.String(), IdempotencyKey: request.IdempotencyKey,
		ReceiptRef: "receipt:sqlite-launch:" + request.ExecutionRef.String(), AcceptedAt: agent.clock.Now(),
	}, nil
}

func (*sqliteMultiControlAgent) Observe(context.Context, goal.ExecutionRef) (ports.AgentObservation, error) {
	return ports.AgentObservation{}, nil
}

func (*sqliteMultiControlAgent) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true}, nil
}

func (agent *sqliteMultiControlAgent) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	return ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: ports.AgentStopped, ReceiptRef: "receipt:sqlite-stop:" + request.ExecutionRef.String(),
		ConfirmedAt: agent.clock.Now(),
	}, nil
}

func newSQLiteMultiControlOrchestrator(
	t *testing.T,
	repository *Repository,
	clock *restartClock,
	ids *restartIDs,
	agent *sqliteMultiControlAgent,
) *application.Orchestrator {
	t.Helper()
	_, fuentes := prepararCapacidadSQLiteV15(t, repository, clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository, Launcher: agent, Observer: agent, Controller: agent,
		WorkspaceManager: &sqliteTestWorkspaceManager{},
		Artifacts:        restartArtifacts{}, Clock: clock, IDs: ids, MaxOutputBytes: 4096,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
		AgentCapabilities: sqliteMultiControlCapabilities(), ClaimLease: time.Minute,
		CapacitySources: fuentes, CapacityObservationWait: time.Second,
		DirectorLeaseDuration: time.Minute, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func sqliteMultiControlCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:sqlite-multi", ModelRef: "model:sqlite-multi",
		AgentRef: "agent:sqlite-multi", Unrestricted: true,
	}
}
