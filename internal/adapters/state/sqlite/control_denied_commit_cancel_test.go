package sqlite

import (
	"context"
	"errors"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestSQLiteCancelAwaitingCommitWithDeniedEffectRetiresAction(t *testing.T) {
	for _, target := range []application.ControlTarget{
		application.ControlTargetWorkItem,
		application.ControlTargetGoal,
	} {
		t.Run(string(target), func(t *testing.T) {
			testSQLiteCancelAwaitingCommitWithDeniedEffectRetiresAction(t, target)
		})
	}
}

func testSQLiteCancelAwaitingCommitWithDeniedEffectRetiresAction(
	t *testing.T,
	target application.ControlTarget,
) {
	system := newSQLiteV15System(t, 2)
	system.orchestrator = newSQLiteV16OrchestratorWithSessions(
		t, system, &sqliteExecutionSessionBroker{at: system.clock.Now()},
	)
	submitted, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:sqlite-denied-commit", Statement: "produce a candidate that can be denied", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:sqlite-denied-commit", Key: "phase:sqlite-denied-commit",
				TemplateRef: "phase-template:sqlite-denied-commit",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "write isolated evidence", Phase: "phase:sqlite-denied-commit",
				Role: "role:writer", WriteSet: []string{"internal/denied-commit"},
				CouncilPolicy:  council.PolicyAuto,
				RequiredTests:  sqliteRequiredTestSpecs("required-test:sqlite-denied-commit"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace,
		application.ActionLaunchAgent,
		application.ActionObserveAgent,
	)

	before, err := system.repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item := before.Goal.WorkItems()[0]
	execution := before.Executions[0]
	if execution.State != application.ExecutionAwaitingCommit {
		t.Fatalf("execution state=%s want=%s", execution.State, application.ExecutionAwaitingCommit)
	}
	var commitIntent application.EffectIntent
	for _, candidate := range before.EffectIntents {
		if candidate.ActionKind == application.ActionCommitChange &&
			candidate.Subject.ExecutionRef == execution.Ref {
			commitIntent = candidate
			break
		}
	}
	if commitIntent.Ref == "" {
		t.Fatal("commit effect intent missing")
	}
	denied, err := system.orchestrator.DecideEffect(
		context.Background(), system.access, application.DecideEffectRequest{
			RequestRef: "request:sqlite-deny-empty-commit", GoalRef: before.Goal.Ref(),
			IntentRef: commitIntent.Ref, ExpectedIntentDigest: commitIntent.Digest,
			Decision: application.EffectDenied, Reason: "workspace is clean and delivery is blocked",
		},
	)
	if err != nil || !denied.Created {
		t.Fatalf("deny commit effect: result=%+v err=%v", denied, err)
	}

	request := application.ControlRequest{
		RequestRef: "request:sqlite-cancel-denied-commit", Operation: application.ControlCancel,
		Target: target, GoalRef: before.Goal.Ref(),
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: before.Goal.AppSpec().Generation(), ExpectedSpecHash: before.Goal.SpecHash(),
		Reason: "retire blocked commit without false evidence",
	}
	if target == application.ControlTargetWorkItem {
		request.WorkItemRef = item.Ref()
		request.ExpectedWorkItemRevision = item.Revision()
	}
	result, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || !result.Created || result.Control.Status != application.ControlConfirmed {
		t.Fatalf("cancel awaiting commit: result=%+v err=%v cause=%v", result, err, errors.Unwrap(err))
	}

	after, err := system.repository.GetGoal(context.Background(), before.Goal.Ref())
	if err != nil {
		t.Fatalf("read canceled goal: err=%v cause=%v", err, errors.Unwrap(err))
	}
	if after.Executions[0].State != application.ExecutionCanceled {
		t.Fatalf("execution after cancel=%s", after.Executions[0].State)
	}
	wantGoalState := goal.GoalStateFailed
	if target == application.ControlTargetGoal {
		wantGoalState = goal.GoalStateCanceled
	}
	if after.Goal.State() != wantGoalState {
		t.Fatalf("goal after cancel=%s want=%s", after.Goal.State(), wantGoalState)
	}
	retirementReceipts := 0
	for _, receipt := range after.ConsumptionReceipts {
		if receipt.ActionRef != "action:commit-change:"+execution.Ref.String() {
			continue
		}
		retirementReceipts++
		if receipt.Kind != application.ActionCommitChange || receipt.ChangeRef.String() == "" ||
			receipt.Outcome != application.ActionConsumedCompleted ||
			receipt.ErrorCode != "application.action_retired" || receipt.EffectReceiptRef != "" {
			t.Fatalf("commit retirement receipt=%+v", receipt)
		}
	}
	if retirementReceipts != 1 {
		t.Fatalf("commit retirement receipts=%d", retirementReceipts)
	}
	var pending int
	err = system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE ref = ? AND completed_at IS NULL AND retired_at IS NULL AND quarantined_at IS NULL`,
		"action:commit-change:"+execution.Ref.String(),
	).Scan(&pending)
	sqliteTestNoError(t, err)
	if pending != 0 {
		t.Fatalf("denied commit action remains pending=%d", pending)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected canceled denied commit: %s", sqliteTestErrorChain(err))
	}
	var revocations int
	err = system.repository.db.QueryRow(`
SELECT COUNT(*) FROM outbox
WHERE kind='revoke_execution_session' AND execution_ref=?`,
		execution.Ref.String(),
	).Scan(&revocations)
	sqliteTestNoError(t, err)
	if revocations != 1 {
		t.Fatalf("execution session revocations=%d want=1", revocations)
	}
	processSQLiteV16Actions(t, system, application.ActionRevokeSession)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected consumed session revocation: %s", sqliteTestErrorChain(err))
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	restarted, err := system.repository.GetGoal(context.Background(), before.Goal.Ref())
	if err != nil || restarted.Executions[0].State != application.ExecutionCanceled {
		t.Fatalf("restart canceled denied commit: state=%+v err=%v", restarted.Executions, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("restart recovery rejected canceled denied commit: %s", sqliteTestErrorChain(err))
	}
}

func TestSQLiteCancelMixedReviewCohortRemainsReadableUntilStopSettlement(t *testing.T) {
	system, goalRef := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest,
		application.ActionLaunchAgent,
	)

	before, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := before.Goal.WorkItems()[0]
	request := application.ControlRequest{
		RequestRef: "request:sqlite-cancel-mixed-review-cohort", Operation: application.ControlCancel,
		Target: application.ControlTargetWorkItem, GoalRef: goalRef,
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: before.Goal.AppSpec().Generation(), ExpectedSpecHash: before.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		Reason: "cancel author and every reviewer without an unreadable intermediate state",
	}
	requested, err := system.orchestrator.Control(context.Background(), system.access, request)
	if err != nil || !requested.Created || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request mixed review cancel: result=%+v err=%v cause=%v",
			requested, err, errors.Unwrap(err))
	}

	pending, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("read pending mixed review cancel: err=%v cause=%v", err, errors.Unwrap(err))
	}
	states := map[application.ExecutionState]int{}
	for _, execution := range pending.Executions {
		states[execution.State]++
	}
	if states[application.ExecutionAwaitingIntegration] != 1 ||
		states[application.ExecutionCanceled] != 1 ||
		states[application.ExecutionRunning] != 1 {
		t.Fatalf("pending mixed review states=%+v", states)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected pending mixed review cancel: %s", sqliteTestErrorChain(err))
	}

	processSQLiteV16Actions(t, system, application.ActionStopAgent)
	closed, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("read closed mixed review cancel: err=%v cause=%v", err, errors.Unwrap(err))
	}
	item, _ = closed.Goal.WorkItem(item.Ref())
	states = map[application.ExecutionState]int{}
	for _, execution := range closed.Executions {
		states[execution.State]++
	}
	if item.State() != goal.WorkItemStateCanceled ||
		requested.Control.Ref == "" || closed.Controls[len(closed.Controls)-1].Status != application.ControlConfirmed ||
		states[application.ExecutionCanceled] != 2 || states[application.ExecutionStopped] != 1 {
		t.Fatalf("closed mixed review item=%s states=%+v controls=%+v",
			item.State(), states, closed.Controls)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected closed mixed review cancel: %s", sqliteTestErrorChain(err))
	}
}

func TestSQLiteCancelTwoRunningReviewersRecoversBetweenStopSettlements(t *testing.T) {
	system, goalRef := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest,
		application.ActionLaunchAgent,
		application.ActionLaunchAgent,
	)

	before, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := before.Goal.WorkItems()[0]
	requested, err := system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: "request:sqlite-cancel-running-reviewers", Operation: application.ControlCancel,
		Target: application.ControlTargetWorkItem, GoalRef: goalRef,
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: before.Goal.AppSpec().Generation(), ExpectedSpecHash: before.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		Reason: "settle every running reviewer before closing the staged author",
	})
	if err != nil || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request running reviewers cancel: result=%+v err=%v", requested, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected requested reviewer stops: %s", sqliteTestErrorChain(err))
	}

	processSQLiteV16Actions(t, system, application.ActionStopAgent)
	mid, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("read first reviewer stop: err=%v cause=%v", err, errors.Unwrap(err))
	}
	states := map[application.ExecutionState]int{}
	for _, execution := range mid.Executions {
		states[execution.State]++
	}
	control, found := controlByRefForSQLiteTest(mid.Controls, requested.Control.Ref)
	if !found || control.Status != application.ControlRequested ||
		states[application.ExecutionAwaitingIntegration] != 1 ||
		states[application.ExecutionRunning] != 1 || states[application.ExecutionStopped] != 1 {
		t.Fatalf("first reviewer stop states=%+v control=%+v", states, control)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected first reviewer stop: %s", sqliteTestErrorChain(err))
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteV15Repository(t, system.path, system.clock.Now)
	system.orchestrator = newSQLiteV16Orchestrator(t, system)
	processSQLiteV16Actions(t, system, application.ActionStopAgent)

	closed, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("read completed reviewer stops: err=%v cause=%v", err, errors.Unwrap(err))
	}
	item, _ = closed.Goal.WorkItem(item.Ref())
	states = map[application.ExecutionState]int{}
	for _, execution := range closed.Executions {
		states[execution.State]++
	}
	control, found = controlByRefForSQLiteTest(closed.Controls, requested.Control.Ref)
	if !found || item.State() != goal.WorkItemStateCanceled ||
		control.Status != application.ControlConfirmed ||
		states[application.ExecutionCanceled] != 1 || states[application.ExecutionStopped] != 2 {
		t.Fatalf("completed reviewer stops item=%s states=%+v control=%+v",
			item.State(), states, control)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected completed reviewer stops: %s", sqliteTestErrorChain(err))
	}
}

func TestSQLiteCancelReviewerAlreadyCompletedSettlesThroughObservation(t *testing.T) {
	system, goalRef := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
	processSQLiteV16Actions(t, system,
		application.ActionAttestTest,
		application.ActionLaunchAgent,
	)
	system.external.mu.Lock()
	system.external.stopStatus = ports.AgentStopAlreadyCompleted
	system.external.mu.Unlock()

	before, err := system.repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	item := before.Goal.WorkItems()[0]
	requested, err := system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: "request:sqlite-cancel-reviewer-already-completed", Operation: application.ControlCancel,
		Target: application.ControlTargetWorkItem, GoalRef: goalRef,
		ExpectedGoalRevision: before.Goal.Revision(), ExpectedPlanGeneration: before.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: before.Goal.AppSpec().Generation(), ExpectedSpecHash: before.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		Reason: "preserve provider completion without manufacturing a review",
	})
	if err != nil || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request already-completed cancel: result=%+v err=%v", requested, err)
	}
	processSQLiteV16Actions(t, system, application.ActionStopAgent)
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected already-completed stop receipt: %s", sqliteTestErrorChain(err))
	}

	processSQLiteV16Actions(t, system, application.ActionObserveAgent)
	closed, err := system.repository.GetGoal(context.Background(), goalRef)
	if err != nil {
		t.Fatalf("read observed cancel completion: err=%v cause=%v", err, errors.Unwrap(err))
	}
	item, _ = closed.Goal.WorkItem(item.Ref())
	states := map[application.ExecutionState]int{}
	for _, execution := range closed.Executions {
		states[execution.State]++
	}
	control, found := controlByRefForSQLiteTest(closed.Controls, requested.Control.Ref)
	if !found || item.State() != goal.WorkItemStateCanceled ||
		control.Status != application.ControlConfirmed || len(closed.Reviews) != 0 ||
		states[application.ExecutionCanceled] != 2 || states[application.ExecutionSucceeded] != 1 {
		t.Fatalf("observed cancel item=%s states=%+v reviews=%d control=%+v",
			item.State(), states, len(closed.Reviews), control)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("recovery rejected observed cancel completion: %s", sqliteTestErrorChain(err))
	}
	mustV10Exec(t, system.repository.db, `
UPDATE executions SET finished_at=finished_at+1
WHERE state='succeeded' AND purpose IN ('primary_review','adversarial_review')`)
	if _, err := system.repository.GetGoal(context.Background(), goalRef); err == nil ||
		!application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("tampered cancel settlement accepted: %v", err)
	}
}

func TestSQLiteRecoveryDoesNotTreatOrdinaryStopAsCancelGenerationProof(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	submitted := system.submit(t, "request:sqlite-ordinary-stop-generation")
	processSQLiteV16Actions(t, system, application.ActionLaunchAgent)
	running, err := system.repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item := running.Goal.WorkItems()[0]
	execution := running.Executions[0]
	requested, err := system.orchestrator.Control(context.Background(), system.access, application.ControlRequest{
		RequestRef: "request:sqlite-ordinary-stop-generation", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "ordinary stop must not authorize stale observation generation",
	})
	if err != nil || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request ordinary stop: result=%+v err=%v", requested, err)
	}
	rewriteRecoveryTrigger(t, system.repository.db, "outbox_identity_immutable", func() {
		mustV10Exec(t, system.repository.db, `
UPDATE outbox SET work_item_generation=work_item_generation-1
WHERE kind='observe_agent' AND execution_ref=?`, execution.Ref.String())
	})
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_outbox_active_state_invalid") {
		t.Fatalf("ordinary stop accepted stale observation generation: %s", sqliteTestErrorChain(err))
	}
}

func controlByRefForSQLiteTest(
	controls []application.ControlRecord,
	ref string,
) (application.ControlRecord, bool) {
	for _, control := range controls {
		if control.Ref == ref {
			return control, true
		}
	}
	return application.ControlRecord{}, false
}
