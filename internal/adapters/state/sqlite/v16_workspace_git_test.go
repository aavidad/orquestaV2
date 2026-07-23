package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

// This is intentionally a repository-level migration/restart gate. The
// end-to-end Git race lives in acceptance; here we prove that V16 keeps the
// four facts in the same SQLite authority and that reopening cannot silently
// fall back to the V15 schema.
func TestSQLiteWorkspaceGitRestartRaceAndReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "orquesta.sqlite")
	options := Options{Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4}
	repository, err := openWithDurability(context.Background(), options, fastSQLiteTestDurability)
	if err != nil {
		t.Fatalf("open v16 repository: %v", err)
	}
	for _, table := range []string{
		"workspace_bindings", "workspace_binding_write_scopes", "change_sets", "change_set_paths",
		"merge_observations", "integration_receipts",
	} {
		var count int
		if err := repository.db.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("v16 table %s missing: count=%d err=%v", table, count, err)
		}
	}
	var version int
	if err := repository.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != recoverySchemaV20 {
		t.Fatalf("v16 version=%d err=%v", version, err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("close first repository: %v", err)
	}
	reopened, err := openWithDurability(context.Background(), options, fastSQLiteTestDurability)
	if err != nil {
		t.Fatalf("reopen v16 repository: %v", err)
	}
	defer reopened.Close()
	if _, _, err := validateRecoveryDatabase(context.Background(), reopened.db); err != nil {
		t.Fatalf("validate v16 recovery: %v", err)
	}
}

type recoveryV16TamperCase struct {
	name, trigger, code string
	mutate              func(*testing.T, *Repository)
}

func TestRecoveryV16RejectsWorkspaceCausalTampering(t *testing.T) {
	assertRecoveryV16Tampering(t, []recoveryV16TamperCase{
		{
			name: "workspace_binding_cross_wired_to_commit_receipt", trigger: "workspace_bindings_immutable_update",
			code: "sqlite.recovery_v16_workspace_binding_invalid",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `UPDATE workspace_bindings SET effect_receipt_ref=(
SELECT receipt.ref FROM effect_receipts receipt JOIN effect_intents intent ON intent.ref=receipt.intent_ref
WHERE intent.kind='commit_change' LIMIT 1)`)
			},
		},
		{
			name: "commit_action_change_ref", trigger: "outbox_integration_admission_immutable",
			code: "sqlite.recovery_v16_change_set_invalid",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `UPDATE outbox SET change_ref='change-set:tampered' WHERE kind='commit_change'`)
			},
		},
		{
			name: "commit_consumption_change_ref", trigger: "action_consumption_receipts_immutable_update",
			code: "sqlite.recovery_v16_change_set_invalid",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `UPDATE action_consumption_receipts SET change_ref='change-set:tampered' WHERE kind='commit_change'`)
			},
		},
		{
			name: "change_set_action_fence", trigger: "change_sets_immutable_update",
			code: "sqlite.recovery_v16_change_set_invalid",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `UPDATE change_sets SET action_fence=action_fence+1`)
			},
		},
		{
			name: "integration_action_expected_target", trigger: "outbox_integration_admission_immutable",
			code: "sqlite.recovery_v16_integration_receipt_invalid",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `UPDATE outbox SET expected_target_oid=? WHERE kind='integrate_change'`, sqliteTestGitOID('2'))
			},
		},
		{
			name: "integration_observation_outcome", trigger: "merge_observations_immutable_update",
			code: "sqlite.recovery_v16_integration_receipt_invalid",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `UPDATE merge_observations
SET outcome='stale',candidate_tree_oid=NULL,conflict_digest=?`, sqliteTestDigest('3'))
			},
		},
	})
}

func TestRecoveryV16RejectsMissingIntegrationFacts(t *testing.T) {
	assertRecoveryV16Tampering(t, []recoveryV16TamperCase{
		{
			name: "integration_receipt_deleted", trigger: "integration_receipts_immutable_delete",
			code: "sqlite.recovery_v16_integration_completion_fact_missing",
			mutate: func(t *testing.T, repository *Repository) {
				mustV10Exec(t, repository.db, `DELETE FROM integration_receipts`)
			},
		},
		{
			name: "integration_receipt_and_observation_deleted", trigger: "integration_receipts_immutable_delete",
			code: "sqlite.recovery_v16_integration_completion_fact_missing",
			mutate: func(t *testing.T, repository *Repository) {
				rewriteRecoveryTrigger(t, repository.db, "merge_observations_immutable_delete", func() {
					mustV10Exec(t, repository.db, `DELETE FROM integration_receipts`)
					mustV10Exec(t, repository.db, `DELETE FROM merge_observations`)
				})
			},
		},
	})
}

func assertRecoveryV16Tampering(t *testing.T, tests []recoveryV16TamperCase) {
	t.Helper()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system := seedSQLiteV16Integrated(t)
			rewriteRecoveryTrigger(t, system.repository.db, test.trigger, func() {
				test.mutate(t, system.repository)
			})
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
				!recoveryErrorContains(err, test.code) {
				t.Fatalf("V16 causal tamper accepted or misclassified: err=%v want=%s", err, test.code)
			}
		})
	}
}

func seedSQLiteV16Integrated(t *testing.T) *sqliteV15System {
	t.Helper()
	system := newSQLiteV15System(t, 4)
	system.orchestrator = newSQLiteV16Orchestrator(t, system)
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: "request:v16-recovery-tamper", Statement: "persist exact workspace causal facts", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:v16-recovery", Key: "phase:v16-recovery", TemplateRef: "phase-template:v16-recovery",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "write isolated evidence", Phase: "phase:v16-recovery", Role: "role:writer",
				WriteSet:       []string{"internal/workspace"},
				CouncilPolicy:  council.PolicyAuto,
				RequiredTests:  sqliteRequiredTestSpecs("required-test:v17-sqlite"),
				OutputContract: goal.OutputContractEvidenceBundle,
			}},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, system,
		application.ActionPrepareWorkspace,
		application.ActionLaunchAgent,
		application.ActionObserveAgent,
		application.ActionCommitChange,
		application.ActionAttestTest,
		application.ActionLaunchAgent,
		application.ActionLaunchAgent,
		application.ActionObserveAgent,
		application.ActionObserveAgent,
		application.ActionLaunchAgent,
		application.ActionLaunchAgent,
		application.ActionLaunchAgent,
		application.ActionObserveAgent,
		application.ActionObserveAgent,
		application.ActionObserveAgent,
	)
	record, err := system.repository.GetGoal(context.Background(), result.Record.Goal.Ref())
	if err != nil || len(record.ChangeSets) != 1 || len(record.WorkspaceBindings) != 1 {
		t.Fatalf("V16 committed seed incomplete: record=%+v err=%v", record, err)
	}
	if _, err := system.orchestrator.IntegrateChange(context.Background(), system.access, application.IntegrateChangeRequest{
		RequestRef: "request:v16-recovery-integrate", GoalRef: record.Goal.Ref(),
		ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
	}); err != nil {
		t.Fatal(err)
	}
	processSQLiteV16Actions(t, system, application.ActionIntegrateChange)
	record, err = system.repository.GetGoal(context.Background(), record.Goal.Ref())
	if err != nil || len(record.IntegrationReceipts) != 1 ||
		record.IntegrationReceipts[0].Status != ports.IntegrationStatusIntegrated {
		t.Fatalf("V16 integrated seed incomplete: record=%+v err=%v cause=%v", record, err, errors.Unwrap(err))
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err != nil {
		t.Fatalf("valid V16 recovery seed rejected: %v cause=%v", err, errors.Unwrap(err))
	}
	return system
}

func processSQLiteV16Actions(t *testing.T, system *sqliteV15System, expected ...application.ActionKind) {
	t.Helper()
	for _, want := range expected {
		var result application.ProcessResult
		var err error
		for retries := 0; retries < 4; retries++ {
			result, err = system.orchestrator.ProcessNext(context.Background(), "worker:v16-recovery")
			if err != nil || result.Processed {
				break
			}
			system.clock.Advance(time.Second)
		}
		if err != nil || !result.Processed || result.Action != want {
			t.Fatalf("process V16 action=%s result=%+v err=%v cause=%v", want, result, err, errors.Unwrap(err))
		}
	}
}

func sqliteTestDigest(value byte) string {
	return strings.Repeat(string([]byte{value}), 64)
}
