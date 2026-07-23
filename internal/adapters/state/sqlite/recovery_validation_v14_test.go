package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestV14RecoveryAndBackupRejectControlSemanticTamperingWithValidForeignKeys(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, recoveryV14ControlSeed)
		want   string
	}{
		{
			name: "reason whitespace is not canonical",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				mutateRecoveryControlIgnoringChecks(t, seed.repository.db,
					`UPDATE controls SET reason = ' ' || reason || ' ' WHERE ref = ?`, seed.control.Ref,
				)
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "operation target matrix",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET operation = 'retry' WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "mode belongs only to stop",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET mode = 'cooperative' WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "target carries exact work item fence",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				var itemRef string
				var revision int64
				if err := seed.repository.db.QueryRow(
					`SELECT ref, revision FROM work_items WHERE goal_ref = ? LIMIT 1`, seed.control.GoalRef.String(),
				).Scan(&itemRef, &revision); err != nil {
					t.Fatal(err)
				}
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls
SET target = 'work_item', work_item_ref = ?, work_item_revision = ? WHERE ref = ?`,
						itemRef, revision, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "request fingerprint",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET request_fingerprint = ? WHERE ref = ?`,
						"tampered-control-fingerprint", seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "Goal revision",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET goal_revision = goal_revision + 1 WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "plan generation",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET plan_generation = plan_generation + 1 WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "app spec generation",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET app_spec_generation = app_spec_generation + 1 WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "spec hash",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET spec_hash = ? WHERE ref = ?`,
						fmt.Sprintf("%064x", 1), seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "authorization receipt",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET authorization_receipt_ref = ? WHERE ref = ?`,
						seed.alternateAuthorizationRef, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_record_invalid",
		},
		{
			name: "requested time must equal causal event",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET requested_at = requested_at + 1,
confirmed_at = confirmed_at + 1 WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_event_invalid",
		},
		{
			name: "confirmed receipt is causal",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls SET receipt_ref = 'receipt:tampered' WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_local_invalid",
		},
		{
			name: "local control cannot regress to requested",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls
SET status = 'requested', confirmed_at = NULL, receipt_ref = NULL WHERE ref = ?`, seed.control.Ref)
				})
			},
			want: "sqlite.recovery_control_local_invalid",
		},
		{
			name: "causal event cannot disappear",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				mustV10Exec(t, seed.repository.db, `DELETE FROM events WHERE ref = ?`, "event:control:"+seed.control.Ref)
			},
			want: "sqlite.recovery_control_event_invalid",
		},
		{
			name: "causal event kind cannot change",
			mutate: func(t *testing.T, seed recoveryV14ControlSeed) {
				mustV10Exec(t, seed.repository.db, `UPDATE events SET kind = 'control.cancel' WHERE ref = ?`, "event:control:"+seed.control.Ref)
			},
			want: "sqlite.recovery_control_event_invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			seed := seedRecoveryV14PauseControl(t, test.name)
			test.mutate(t, seed)
			if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
				(!recoveryErrorContains(err, test.want) &&
					!application.IsStateError(err, application.StateInvalid)) {
				t.Fatalf("semantic tamper accepted or wrong error: want=%s err=%v", test.want, err)
			}
			if test.name == "request fingerprint" || test.name == "reason whitespace is not canonical" {
				recovery, _, _ := newV09TestRecovery(t, seed.repository, seed.clock.Now(), nil)
				if _, err := recovery.CreateBackup(context.Background()); err == nil ||
					(!recoveryErrorContains(err, test.want) &&
						!application.IsStateError(err, application.StateInvalid)) {
					t.Fatalf("backup accepted semantic tamper: %v", err)
				}
				if err := recovery.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestV14RecoveryAndBackupRejectStopSupersessionTampering(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testing.T, recoveryV14SupersessionSeed)
		want   string
	}{
		{
			name: "old reverse link",
			mutate: func(t *testing.T, seed recoveryV14SupersessionSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls
SET superseded_by_control_ref = ? WHERE ref = ?`, seed.old.Ref, seed.old.Ref)
				})
			},
			want: "sqlite.recovery_control_supersession_lineage_invalid",
		},
		{
			name: "new forward link",
			mutate: func(t *testing.T, seed recoveryV14SupersessionSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls
SET supersedes_control_ref = ? WHERE ref = ?`, seed.next.Ref, seed.next.Ref)
				})
			},
			want: "sqlite.recovery_control_supersession_lineage_invalid",
		},
		{
			name: "superseded time",
			mutate: func(t *testing.T, seed recoveryV14SupersessionSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, `UPDATE controls
SET superseded_at = superseded_at + 1 WHERE ref = ?`, seed.old.Ref)
				})
			},
			want: "sqlite.recovery_control_supersession_lineage_invalid",
		},
		{
			name: "lineage event",
			mutate: func(t *testing.T, seed recoveryV14SupersessionSeed) {
				mustV10Exec(t, seed.repository.db, `DELETE FROM events WHERE ref = ?`,
					"event:control-superseded:"+seed.old.Ref)
			},
			want: "sqlite.recovery_control_supersession_event_invalid",
		},
		{
			name: "retired cooperative action steals forced provider effect",
			mutate: func(t *testing.T, seed recoveryV14SupersessionSeed) {
				rewriteRecoveryTrigger(t, seed.repository.db, "action_consumption_receipts_immutable_update", func() {
					oldActionRef := "action:stop:" + seed.old.Ref + ":" + seed.old.ExecutionRef.String()
					forcedActionRef := "action:stop:" + seed.next.Ref + ":" + seed.next.ExecutionRef.String()
					mustV10Exec(t, seed.repository.db, `UPDATE outbox SET last_error_code = '' WHERE ref = ?`, oldActionRef)
					mustV10Exec(t, seed.repository.db, `UPDATE action_consumption_receipts
SET error_code = '', effect_receipt_ref = (
    SELECT effect_receipt_ref FROM action_consumption_receipts WHERE action_ref = ?
) WHERE action_ref = ?`, forcedActionRef, oldActionRef)
				})
			},
			want: "sqlite.recovery_stop_effect_control_invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			seed := seedRecoveryV14Supersession(t, test.name)
			test.mutate(t, seed)
			if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
				!recoveryErrorContains(err, test.want) {
				t.Fatalf("supersession tamper accepted or wrong error: want=%s err=%v", test.want, err)
			}
			recovery, _, _ := newV09TestRecovery(t, seed.repository, seed.clock.Now(), nil)
			if _, err := recovery.CreateBackup(context.Background()); err == nil ||
				!recoveryErrorContains(err, test.want) {
				t.Fatalf("backup accepted supersession tamper: want=%s err=%v", test.want, err)
			}
			if err := recovery.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type recoveryV14SupersessionSeed struct {
	repository *Repository
	clock      *restartClock
	old        application.ControlRecord
	next       application.ControlRecord
}

func seedRecoveryV14Supersession(t *testing.T, suffix string) recoveryV14SupersessionSeed {
	t.Helper()
	suffix = strings.ReplaceAll(suffix, " ", "-")
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 18, 30, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteEscalationAgent{clock: clock}
	orchestrator, err := application.New(application.Dependencies{
		State: repository, Access: repository, Launcher: agent, Observer: agent, Controller: agent,
		Artifacts: restartArtifacts{}, Clock: clock, IDs: ids, MaxOutputBytes: 4096,
		MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
		AgentCapabilities: sqliteMultiControlCapabilities(), ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	sqliteTestNoError(t, err)
	actor, _ := goal.NewActorRef("actor:recovery-v14-supersession-" + suffix)
	project, _ := goal.NewProjectRef("project:recovery-v14-supersession-" + suffix)
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:recovery-v14-supersession-" + suffix,
		Statement:  "recovery validates exact stop supersession " + suffix, Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-supersession-launch"); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch seed: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]
	cooperative := sqliteExactStopRequest(
		running, item, execution, "control:recovery-cooperative-"+suffix, ports.AgentStopCooperative,
	)
	if _, err := orchestrator.Control(context.Background(), access, cooperative); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []application.ControlOperation{application.ControlPause, application.ControlResume} {
		intermediate, getErr := repository.GetGoal(context.Background(), running.Goal.Ref())
		if getErr != nil {
			t.Fatal(getErr)
		}
		intermediateItem, _ := intermediate.Goal.WorkItem(item.Ref())
		_, controlErr := orchestrator.Control(context.Background(), access, application.ControlRequest{
			RequestRef: "control:recovery-" + string(operation) + "-" + suffix,
			Operation:  operation, Target: application.ControlTargetWorkItem,
			GoalRef: intermediate.Goal.Ref(), ExpectedGoalRevision: intermediate.Goal.Revision(),
			ExpectedPlanGeneration:    intermediate.Goal.PlanGeneration(),
			ExpectedAppSpecGeneration: intermediate.Goal.AppSpec().Generation(),
			ExpectedSpecHash:          intermediate.Goal.SpecHash(), WorkItemRef: intermediateItem.Ref(),
			ExpectedWorkItemRevision: intermediateItem.Revision(), Reason: "intermediate pause fence",
		})
		if controlErr != nil {
			t.Fatalf("intermediate %s: %v", operation, controlErr)
		}
	}
	current, err := repository.GetGoal(context.Background(), running.Goal.Ref())
	sqliteTestNoError(t, err)
	item, _ = current.Goal.WorkItem(item.Ref())
	forced := sqliteExactStopRequest(
		current, item, execution, "control:recovery-forced-"+suffix, ports.AgentStopForced,
	)
	if _, err := orchestrator.Control(context.Background(), access, forced); err != nil {
		t.Fatal(err)
	}
	pendingForced, err := repository.GetGoal(context.Background(), running.Goal.Ref())
	sqliteTestNoError(t, err)
	_, forcedControl := sqliteControlsByRequest(t, pendingForced, cooperative.RequestRef, forced.RequestRef)
	var forcedIntent application.EffectIntent
	for _, intent := range pendingForced.EffectIntents {
		if intent.ActionRef == "action:stop:"+forcedControl.Ref+":"+execution.Ref.String() {
			forcedIntent = intent
			break
		}
	}
	approved, err := orchestrator.DecideEffect(context.Background(), access, application.DecideEffectRequest{
		RequestRef: "approval:recovery-supersession-forced-" + suffix, GoalRef: pendingForced.Goal.Ref(),
		IntentRef: forcedIntent.Ref, ExpectedIntentDigest: forcedIntent.Digest,
		Decision: application.EffectApproved, Reason: "owner approves forced supersession seed",
	})
	if err != nil || !approved.Created {
		t.Fatalf("forced seed approval=%+v intent=%+v err=%v", approved, forcedIntent, err)
	}
	if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-supersession-forced"); processErr != nil || !result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("forced seed: result=%+v err=%v", result, processErr)
	}
	transferred, err := repository.GetGoal(context.Background(), running.Goal.Ref())
	sqliteTestNoError(t, err)
	old, next := sqliteControlsByRequest(t, transferred, cooperative.RequestRef, forced.RequestRef)
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("valid supersession seed rejected: %v cause=%v", err, errors.Unwrap(err))
	}
	return recoveryV14SupersessionSeed{repository: repository, clock: clock, old: old, next: next}
}

func mutateRecoveryControlIgnoringChecks(t *testing.T, database *sql.DB, statement string, args ...any) {
	t.Helper()
	connection, err := database.Conn(context.Background())
	sqliteTestNoError(t, err)
	defer connection.Close()
	var triggerSQL string
	if err := connection.QueryRowContext(context.Background(), `
SELECT sql FROM sqlite_schema WHERE type = 'trigger' AND name = 'controls_update_guard'`).Scan(&triggerSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `DROP TRIGGER controls_update_guard`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints = ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), statement, args...); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), triggerSQL); err != nil {
		t.Fatal(err)
	}
}

func TestV14RecoveryRejectsStopActionAndEffectReceiptCausalTampering(t *testing.T) {
	t.Run("multi cancel cannot lose one pending execution stop", func(t *testing.T) {
		seed := seedRecoveryV14PendingMultiCancel(t)
		var actionRef string
		if err := seed.repository.db.QueryRow(`
SELECT ref FROM outbox WHERE kind = 'stop_agent' AND control_ref = ? ORDER BY ref LIMIT 1`,
			seed.control.Ref,
		).Scan(&actionRef); err != nil {
			t.Fatal(err)
		}
		mustV10Exec(t, seed.repository.db, `DELETE FROM outbox WHERE ref = ?`, actionRef)
		var foreignKeyErrors int
		if err := seed.repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&foreignKeyErrors); err != nil {
			t.Fatal(err)
		}
		if foreignKeyErrors != 0 {
			t.Fatalf("tamper fixture broke foreign keys: %d", foreignKeyErrors)
		}
		if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_control_cancel_coverage_invalid") {
			t.Fatalf("multi cancel missing stop accepted: %v", err)
		}
	})

	t.Run("stop action keeps exact control WorkItem generation", func(t *testing.T) {
		seed := seedRecoveryV14StoppedControl(t, "action-generation")
		rewriteRecoveryTrigger(t, seed.repository.db, "outbox_identity_immutable", func() {
			rewriteRecoveryTrigger(t, seed.repository.db, "action_consumption_receipts_immutable_update", func() {
				mustV10Exec(t, seed.repository.db, `UPDATE outbox
SET work_item_generation = work_item_generation + 1 WHERE kind = 'stop_agent'`)
				mustV10Exec(t, seed.repository.db, `UPDATE action_consumption_receipts
SET work_item_generation = work_item_generation + 1 WHERE kind = 'stop_agent'`)
			})
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_stop_action_target_invalid") {
			t.Fatalf("tampered stop generation accepted: %v", err)
		}
	})

	t.Run("stop control receipt equals exact effect receipt", func(t *testing.T) {
		seed := seedRecoveryV14StoppedControl(t, "effect-control")
		rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
			mustV10Exec(t, seed.repository.db, `UPDATE controls
SET receipt_ref = 'receipt:other-valid-provider-proof' WHERE ref = ?`, seed.control.Ref)
		})
		if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
			!recoveryErrorContains(err, "sqlite.recovery_stop_effect_control_invalid") {
			t.Fatalf("misbound stop effect accepted: %v", err)
		}
	})

	t.Run("WorkItem cancel receipt and time bind exact stopped effect", func(t *testing.T) {
		for _, mutation := range []struct {
			name  string
			query string
			want  string
		}{
			{"receipt", `UPDATE controls SET receipt_ref = 'receipt:other-cancel-proof' WHERE ref = ?`, "sqlite.recovery_stop_effect_control_invalid"},
			{"time", `UPDATE controls SET confirmed_at = confirmed_at + 1 WHERE ref = ?`, "sqlite.recovery_stop_effect_control_invalid"},
		} {
			t.Run(mutation.name, func(t *testing.T) {
				seed := seedRecoveryV14ConfirmedWorkItemCancel(t, mutation.name)
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, mutation.query, seed.control.Ref)
				})
				if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
					!recoveryErrorContains(err, mutation.want) {
					t.Fatalf("misbound WorkItem cancel %s accepted: %v", mutation.name, err)
				}
			})
		}
	})

	t.Run("Goal cancel receipt and time bind exact closure", func(t *testing.T) {
		for _, mutation := range []struct {
			name  string
			query string
			want  string
		}{
			{"receipt", `UPDATE controls SET receipt_ref = 'receipt:other-goal-cancel' WHERE ref = ?`, "sqlite.recovery_control_cancel_receipt_invalid"},
			{"time", `UPDATE controls SET confirmed_at = confirmed_at + 1 WHERE ref = ?`, "sqlite.recovery_control_cancel_confirmation_invalid"},
		} {
			t.Run(mutation.name, func(t *testing.T) {
				seed := seedRecoveryV14ConfirmedGoalCancel(t)
				rewriteRecoveryTrigger(t, seed.repository.db, "controls_update_guard", func() {
					mustV10Exec(t, seed.repository.db, mutation.query, seed.control.Ref)
				})
				if _, _, err := validateRecoveryDatabase(context.Background(), seed.repository.db); err == nil ||
					!recoveryErrorContains(err, mutation.want) {
					t.Fatalf("misbound Goal cancel %s accepted: %v", mutation.name, err)
				}
			})
		}
	})
}

func seedRecoveryV14PendingMultiCancel(t *testing.T) recoveryV14ControlSeed {
	t.Helper()
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 17, 15, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteMultiControlAgent{clock: clock}
	orchestrator := newSQLiteMultiControlOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:recovery-v14-multi-cancel")
	project, _ := goal.NewProjectRef("project:recovery-v14-multi-cancel")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:recovery-v14-multi-cancel", Statement: "cancel two live executions", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:recovery-v14-multi-cancel", Key: "phase:recovery-v14-multi-cancel",
				TemplateRef: "phase-template:recovery-v14-multi-cancel",
			}},
			WorkItems: []application.WorkItemSpec{
				{Key: "first", Objective: "first live cancellation target", Phase: "phase:recovery-v14-multi-cancel", Role: "role:worker", WriteSet: []string{"first"}, CouncilPolicy: council.PolicyRequired, RequiredTests: sqliteRequiredTestSpecs("required-test:sqlite-recovery-first"), OutputContract: goal.OutputContractEvidenceBundle},
				{Key: "second", Objective: "second live cancellation target", Phase: "phase:recovery-v14-multi-cancel", Role: "role:worker", WriteSet: []string{"second"}, CouncilPolicy: council.PolicyRequired, RequiredTests: sqliteRequiredTestSpecs("required-test:sqlite-recovery-second"), OutputContract: goal.OutputContractEvidenceBundle},
			},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteWorkspaceLaunches(t, orchestrator, "worker:recovery-v14-multi-launch", 2)
	running, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	requested, err := orchestrator.Control(context.Background(), access, application.ControlRequest{
		RequestRef: "control:recovery-v14-multi-cancel",
		Operation:  application.ControlCancel, Target: application.ControlTargetGoal,
		GoalRef: running.Goal.Ref(), ExpectedGoalRevision: running.Goal.Revision(),
		ExpectedPlanGeneration:    running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		Reason: "cancel both exact running executions",
	})
	if err != nil || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request multi cancel: result=%+v err=%v", requested, err)
	}
	var actions int
	if err := repository.db.QueryRow(`SELECT COUNT(*) FROM outbox
WHERE kind = 'stop_agent' AND control_ref = ? AND completed_at IS NULL`, requested.Control.Ref).Scan(&actions); err != nil {
		t.Fatal(err)
	}
	if actions != 2 {
		t.Fatalf("pending multi cancel actions=%d", actions)
	}
	return recoveryV14ControlSeed{repository: repository, clock: clock, control: requested.Control}
}

func seedRecoveryV14ConfirmedGoalCancel(t *testing.T) recoveryV14ControlSeed {
	t.Helper()
	seed := seedRecoveryV14PendingMultiCancel(t)
	agent := &sqliteMultiControlAgent{clock: seed.clock}
	orchestrator := newSQLiteMultiControlOrchestrator(
		t, seed.repository, seed.clock, &restartIDs{}, agent,
	)
	for range 2 {
		if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-v14-goal-cancel"); processErr != nil || !result.Processed || result.Action != application.ActionStopAgent {
			t.Fatalf("settle Goal cancel: result=%+v err=%v", result, processErr)
		}
	}
	settled, err := seed.repository.GetGoal(context.Background(), seed.control.GoalRef)
	if err != nil || len(settled.Controls) != 1 || settled.Controls[0].Status != application.ControlConfirmed {
		t.Fatalf("confirmed Goal cancel: controls=%+v err=%v", settled.Controls, err)
	}
	seed.control = settled.Controls[0]
	return seed
}

func seedRecoveryV14ConfirmedWorkItemCancel(t *testing.T, suffix string) recoveryV14ControlSeed {
	t.Helper()
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 17, 25, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteMultiControlAgent{clock: clock}
	orchestrator := newSQLiteMultiControlOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:recovery-v14-item-cancel-" + suffix)
	project, _ := goal.NewProjectRef("project:recovery-v14-item-cancel-" + suffix)
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:recovery-v14-item-cancel-" + suffix,
		Statement:  "cancel one exact WorkItem " + suffix, Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-v14-item-launch"); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch WorkItem cancel: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item := running.Goal.WorkItems()[0]
	requested, err := orchestrator.Control(context.Background(), access, application.ControlRequest{
		RequestRef: "control:recovery-v14-item-cancel-" + suffix,
		Operation:  application.ControlCancel, Target: application.ControlTargetWorkItem,
		GoalRef: running.Goal.Ref(), ExpectedGoalRevision: running.Goal.Revision(),
		ExpectedPlanGeneration:    running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		Reason: "bind WorkItem cancellation receipt",
	})
	if err != nil || requested.Control.Status != application.ControlRequested {
		t.Fatalf("request WorkItem cancel: result=%+v err=%v", requested, err)
	}
	if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-v14-item-stop"); processErr != nil || !result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("settle WorkItem cancel: result=%+v err=%v", result, processErr)
	}
	settled, err := repository.GetGoal(context.Background(), running.Goal.Ref())
	if err != nil || len(settled.Controls) != 1 || settled.Controls[0].Status != application.ControlConfirmed {
		t.Fatalf("confirmed WorkItem cancel: controls=%+v err=%v", settled.Controls, err)
	}
	return recoveryV14ControlSeed{repository: repository, clock: clock, control: settled.Controls[0]}
}

func TestV14RecoveryAcceptsClaimedTerminalStopThenReclaimsAndSettlesOnce(t *testing.T) {
	ctx := context.Background()
	repository, path := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 18, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteTerminalStopAgent{clock: clock}
	orchestrator := newRecoveryV14ControlOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:recovery-v14-terminal-claim")
	project, _ := goal.NewProjectRef("project:recovery-v14-terminal-claim")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(ctx, access, application.SubmitRequest{
		RequestRef: "request:recovery-v14-terminal-claim", Statement: "finish before stop settlement", Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(ctx, "worker:recovery-v14-launch"); processErr != nil ||
		!result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(ctx, submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]
	gate := &sqliteGetGoalGate{
		StateRepository: repository, entered: make(chan struct{}), release: make(chan struct{}),
	}
	processor := newRecoveryV14ControlOrchestrator(t, gate, clock, ids, agent)
	processed := make(chan error, 1)
	go func() {
		result, processErr := processor.ProcessNext(
			context.WithValue(ctx, sqliteGetGoalGateKey{}, true), "worker:recovery-v14-observe",
		)
		if processErr == nil && (!result.Processed || result.Action != application.ActionObserveAgent) {
			processErr = fmt.Errorf("unexpected observe result: %+v", result)
		}
		processed <- processErr
	}()
	select {
	case <-gate.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("observe claim did not reach crash gate")
	}
	request := application.ControlRequest{
		RequestRef: "control:recovery-v14-terminal-claim", Operation: application.ControlStop,
		Target: application.ControlTargetExecution, GoalRef: running.Goal.Ref(),
		ExpectedGoalRevision: running.Goal.Revision(), ExpectedPlanGeneration: running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "prove terminal claimed stop restart frontier",
	}
	if result, controlErr := orchestrator.Control(ctx, access, request); controlErr != nil ||
		result.Control.Status != application.ControlRequested {
		close(gate.release)
		t.Fatalf("request stop: result=%+v err=%v", result, controlErr)
	}
	close(gate.release)
	if processErr := <-processed; processErr != nil {
		t.Fatal(processErr)
	}
	claim, found, err := repository.ClaimNextAction(ctx, application.ClaimRequest{
		WorkerRef: "worker:recovery-v14-stop-crashed", Token: "claim:recovery-v14-stop-crashed",
		LeaseDuration: time.Minute, Capabilities: sqliteMultiControlCapabilities(), BudgetPolicy: sqliteRuntimeTestPolicy(),
	})
	if err != nil || !found || claim.Action.Kind != application.ActionStopAgent {
		t.Fatalf("claim terminal stop: found=%v claim=%+v err=%v", found, claim, err)
	}
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err != nil {
		t.Fatalf("live claimed terminal stop rejected: %v", err)
	}
	recovery, _, _ := newV09TestRecovery(t, repository, clock.Now(), nil)
	if _, err := recovery.CreateBackup(ctx); err != nil {
		t.Fatalf("backup live claimed terminal stop: %v", err)
	}
	if err := recovery.Close(); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4, Now: clock.Now,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("restarted live claim rejected: %v", err)
	}
	clock.Advance(2 * time.Minute)
	restartedOrchestrator := newRecoveryV14ControlOrchestrator(t, restarted, clock, ids, agent)
	if result, processErr := restartedOrchestrator.ProcessNext(ctx, "worker:recovery-v14-stop-reclaim"); processErr != nil ||
		!result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("reclaim terminal stop: result=%+v err=%v", result, processErr)
	}
	var attempts, receipts int
	if err := restarted.db.QueryRow(`SELECT delivery_attempt FROM outbox WHERE ref = ?`, claim.Action.Ref).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if err := restarted.db.QueryRow(`SELECT COUNT(*) FROM action_consumption_receipts WHERE action_ref = ?`, claim.Action.Ref).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || receipts != 1 || agent.stopCount() != 0 {
		t.Fatalf("settlement attempts=%d receipts=%d physical_stops=%d", attempts, receipts, agent.stopCount())
	}
	if _, _, err := validateRecoveryDatabase(ctx, restarted.db); err != nil {
		t.Fatalf("settled recovery invalid: %v", err)
	}
}

type recoveryV14ControlSeed struct {
	repository                *Repository
	clock                     *restartClock
	control                   application.ControlRecord
	alternateAuthorizationRef string
}

func seedRecoveryV14PauseControl(t *testing.T, suffix string) recoveryV14ControlSeed {
	t.Helper()
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 17, 0, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	orchestrator := newRestartOrchestrator(t, repository, clock, ids, &restartAgent{clock: clock})
	actor, _ := goal.NewActorRef("actor:recovery-v14-pause")
	project, _ := goal.NewProjectRef("project:recovery-v14-pause")
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:recovery-v14-pause", Statement: "validate recovery " + suffix, Confirm: true,
	})
	sqliteTestNoError(t, err)
	record, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	result, err := orchestrator.Control(context.Background(), access, application.ControlRequest{
		RequestRef: "control:recovery-v14-pause",
		Operation:  application.ControlPause, Target: application.ControlTargetGoal,
		GoalRef: record.Goal.Ref(), ExpectedGoalRevision: record.Goal.Revision(),
		ExpectedPlanGeneration:    record.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: record.Goal.AppSpec().Generation(), ExpectedSpecHash: record.Goal.SpecHash(),
		Reason: "recovery semantic seed",
	})
	if err != nil {
		t.Fatalf("pause seed: %v", err)
	}
	principalRef, _ := identity.NewPrincipalRef(actor.String())
	principal, _ := identity.NewPrincipal(principalRef, actor, identity.PrincipalKindHuman, "test")
	authorizationRequest, err := identity.NewAuthorizationRequest(identity.AuthorizationRequestInput{
		RequestRef: "authorization-request:recovery-v14-alternate",
		Principal:  principal, ProjectRef: project, Permission: identity.PermissionGoalsDirect,
		ResourceRef: record.Goal.Ref().String(), RequestedAt: clock.Now(),
	})
	sqliteTestNoError(t, err)
	alternate, err := repository.Authorize(context.Background(), authorizationRequest)
	sqliteTestNoError(t, err)
	return recoveryV14ControlSeed{
		repository: repository, clock: clock, control: result.Control,
		alternateAuthorizationRef: alternate.Ref(),
	}
}

func seedRecoveryV14StoppedControl(t *testing.T, suffix string) recoveryV14ControlSeed {
	t.Helper()
	repository, _ := openTestRepository(t)
	clock := &restartClock{now: time.Date(2026, 7, 16, 17, 30, 0, 0, time.UTC)}
	repository.now = clock.Now
	ids := &restartIDs{}
	agent := &sqliteMultiControlAgent{clock: clock}
	orchestrator := newSQLiteMultiControlOrchestrator(t, repository, clock, ids, agent)
	actor, _ := goal.NewActorRef("actor:recovery-v14-stop-" + suffix)
	project, _ := goal.NewProjectRef("project:recovery-v14-stop-" + suffix)
	access := newRestartAccess(t, repository, actor, project, clock.Now())
	submitted, err := orchestrator.Submit(context.Background(), access, application.SubmitRequest{
		RequestRef: "request:recovery-v14-stop-" + suffix, Statement: "stop recovery " + suffix, Confirm: true,
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-v14-stop-launch"); processErr != nil || !result.Processed || result.Action != application.ActionLaunchAgent {
		t.Fatalf("launch stop seed: result=%+v err=%v", result, processErr)
	}
	running, err := repository.GetGoal(context.Background(), submitted.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	item, execution := running.Goal.WorkItems()[0], running.Executions[0]
	requested, err := orchestrator.Control(context.Background(), access, application.ControlRequest{
		RequestRef: "control:recovery-v14-stop-" + suffix,
		Operation:  application.ControlStop, Target: application.ControlTargetExecution,
		GoalRef: running.Goal.Ref(), ExpectedGoalRevision: running.Goal.Revision(),
		ExpectedPlanGeneration:    running.Goal.PlanGeneration(),
		ExpectedAppSpecGeneration: running.Goal.AppSpec().Generation(), ExpectedSpecHash: running.Goal.SpecHash(),
		WorkItemRef: item.Ref(), ExpectedWorkItemRevision: item.Revision(),
		ExecutionRef: execution.Ref, ExpectedExecutionAttempt: execution.AttemptNo,
		Mode: ports.AgentStopCooperative, Reason: "stop recovery semantic seed",
	})
	sqliteTestNoError(t, err)
	if result, processErr := orchestrator.ProcessNext(context.Background(), "worker:recovery-v14-stop"); processErr != nil || !result.Processed || result.Action != application.ActionStopAgent {
		t.Fatalf("stop seed: result=%+v err=%v", result, processErr)
	}
	settled, err := repository.GetGoal(context.Background(), running.Goal.Ref())
	if err != nil || len(settled.Controls) != 1 {
		t.Fatalf("read settled seed: controls=%d err=%v", len(settled.Controls), err)
	}
	return recoveryV14ControlSeed{repository: repository, clock: clock, control: requested.Control}
}

func newRecoveryV14ControlOrchestrator(
	t *testing.T,
	state application.StateRepository,
	clock *restartClock,
	ids *restartIDs,
	agent *sqliteTerminalStopAgent,
) *application.Orchestrator {
	t.Helper()
	repository, ok := state.(*Repository)
	if !ok {
		if gate, gateOK := state.(*sqliteGetGoalGate); gateOK {
			repository, _ = gate.StateRepository.(*Repository)
		}
	}
	if repository == nil {
		t.Fatal("recovery V14 repository unavailable")
	}
	orchestrator, err := application.New(application.Dependencies{
		State: state, Access: repository, Launcher: agent, Observer: agent, Controller: agent,
		Artifacts: leaseAdvancingArtifacts{clock: clock}, Clock: clock, IDs: ids,
		MaxOutputBytes: 4096, MaxMailboxEnvelopeBytes: 64 << 10, MaxExecutionAttempts: 3,
		MaxChildrenPerParent: 6, EffectApprovalTTL: time.Hour, BudgetPolicy: sqliteTestBudgetPolicy(clock.Now()),
		AgentCapabilities: sqliteMultiControlCapabilities(), ClaimLease: time.Minute,
		DirectorLeaseDuration: time.Minute, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}
