package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsSQLiteMutationInputReplaysExactlyAfterRestart(
	t *testing.T,
) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-input-restart-create")
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-input-restart-mutation",
		created.Record.State.Revision(),
		intake.OriginForm,
	)
	first, err := service.ApplyWizardGaps(ctx, request)
	sqliteTestNoError(t, err)
	if !first.Changed || first.EvaluationReplayExact ||
		first.RequestOutcome.Kind !=
			application.WizardGapsRequestOutcomeIntakeMutation {
		t.Fatalf("first=%+v", first)
	}
	assertSQLiteWizardGapsInputEffects(
		t, system, request.RequestRef, 1, 1, 0,
	)
	questions := first.Evaluation.Questions()
	if len(questions) == 0 {
		t.Fatal("Wizard mutation has no question for later Intake mutation")
	}
	option, found := questions[0].RecommendedOption()
	if !found {
		t.Fatal("Wizard question has no recommended option")
	}
	laterRequestRef := "request:wizard-input-restart-later"
	later, err := system.service.ApplyIntake(
		ctx,
		application.ApplyIntakeRequest{
			RequestRef: laterRequestRef,
			ActorRef:   system.principal.ActorRef,
			ProjectRef: system.project,
			Change: intake.Change{
				StateRef:         system.stateRef,
				ExpectedRevision: first.Record.State.Revision(),
				Origin:           intake.OriginForm,
				Choices: []intake.Choice{{
					QuestionRef: intake.QuestionRef(questions[0].Ref()),
					OptionRef:   intake.OptionRef(option.Ref()),
				}},
			},
			AuthorizationReceipt: system.authorizeIntake(
				t, application.IntakeOperationApply, laterRequestRef,
			),
		},
	)
	sqliteTestNoError(t, err)
	if later.Record.State.Revision() != first.Record.State.Revision()+1 {
		t.Fatalf("later revision=%d", later.Record.State.Revision())
	}

	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	service, err = application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	replayed, err := service.ApplyWizardGaps(ctx, request)
	sqliteTestNoError(t, err)
	if replayed.Changed || replayed.EvaluationReplayExact ||
		replayed.Record.Receipt != first.Record.Receipt ||
		replayed.RequestOutcome != first.RequestOutcome ||
		replayed.InputDurability != first.InputDurability ||
		!reflect.DeepEqual(replayed.Evaluation, first.Evaluation) {
		t.Fatalf("first=%+v replayed=%+v", first, replayed)
	}
}

func TestWizardGapsSQLiteConcurrentExactMutationCommitsOneInput(
	t *testing.T,
) {
	ctx := context.Background()
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-input-race-create")
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-input-race-mutation",
		created.Record.State.Revision(),
		intake.OriginForm,
	)
	type callResult struct {
		result application.ApplyWizardGapsResult
		err    error
	}
	start := make(chan struct{})
	results := make(chan callResult, 2)
	for range 2 {
		go func() {
			<-start
			result, err := service.ApplyWizardGaps(ctx, request)
			results <- callResult{result: result, err: err}
		}()
	}
	close(start)
	var accepted []application.ApplyWizardGapsResult
	for range 2 {
		call := <-results
		if call.err != nil {
			t.Fatalf("concurrent mutation: %v", call.err)
		}
		accepted = append(accepted, call.result)
	}
	if accepted[0].Changed == accepted[1].Changed ||
		accepted[0].Record.Receipt != accepted[1].Record.Receipt ||
		accepted[0].RequestOutcome != accepted[1].RequestOutcome ||
		accepted[0].InputDurability != accepted[1].InputDurability ||
		!reflect.DeepEqual(accepted[0].Evaluation, accepted[1].Evaluation) {
		t.Fatalf("accepted=%+v", accepted)
	}
	assertSQLiteWizardGapsInputEffects(
		t, system, request.RequestRef, 1, 1, 0,
	)
}

func TestWizardGapsSQLiteInputFailureRollsBackWholeRequest(t *testing.T) {
	tests := []struct {
		name  string
		setup func(
			*testing.T,
		) (*sqliteIntakeTestSystem, *application.WizardGapsService, application.ApplyWizardGapsRequest, intake.Revision)
	}{
		{
			name: "mutation",
			setup: func(
				t *testing.T,
			) (*sqliteIntakeTestSystem, *application.WizardGapsService, application.ApplyWizardGapsRequest, intake.Revision) {
				system := newSQLiteIntakeTestSystem(t)
				created := system.create(
					t, "request:wizard-input-rollback-mutation-create",
				)
				service, err := application.NewWizardGapsService(
					system.repository,
				)
				sqliteTestNoError(t, err)
				request := sqliteWizardGapsRequest(
					t,
					system,
					"request:wizard-input-rollback-mutation",
					created.Record.State.Revision(),
					intake.OriginForm,
				)
				return system, service, request, created.Record.State.Revision()
			},
		},
		{
			name: "no-op",
			setup: func(
				t *testing.T,
			) (*sqliteIntakeTestSystem, *application.WizardGapsService, application.ApplyWizardGapsRequest, intake.Revision) {
				system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
				request := sqliteWizardGapsRequest(
					t,
					system,
					"request:wizard-input-rollback-noop",
					seed.Record.State.Revision(),
					intake.OriginForm,
				)
				return system, service, request, seed.Record.State.Revision()
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			system, service, request, revision := test.setup(t)
			mustV10Exec(t, system.repository.db, `
CREATE TRIGGER test_wizard_gaps_input_abort
BEFORE INSERT ON wizard_gaps_input_receipts
BEGIN
    SELECT RAISE(ABORT, 'test.wizard_gaps_input_abort');
END`)
			if _, err := service.ApplyWizardGaps(
				context.Background(), request,
			); err == nil {
				t.Fatal("input insertion failure was accepted")
			}
			assertSQLiteWizardGapsInputEffects(
				t, system, request.RequestRef, 0, 0, 0,
			)
			current, err := system.service.GetIntake(
				context.Background(),
				application.GetIntakeRequest{
					ActorRef:   system.principal.ActorRef,
					ProjectRef: system.project, StateRef: system.stateRef,
				},
			)
			sqliteTestNoError(t, err)
			if current.State.Revision() != revision {
				t.Fatalf(
					"revision=%d want=%d",
					current.State.Revision(),
					revision,
				)
			}
			mustV10Exec(
				t,
				system.repository.db,
				`DROP TRIGGER test_wizard_gaps_input_abort`,
			)
		})
	}
}

func TestV23WizardGapsInputRecoveryRejectsColumnCorruptions(t *testing.T) {
	tests := []struct {
		name     string
		wantCode string
		mutate   func(
			*testing.T,
			*sqliteIntakeTestSystem,
			application.ApplyWizardGapsRequest,
			application.ApplyWizardGapsResult,
			application.IntakeRecord,
		)
	}{
		{
			name:     "facts digest",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				_ application.ApplyWizardGapsResult,
				_ application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts SET facts_digest=? WHERE request_ref=?`,
					strings.Repeat("0", 64), request.RequestRef)
			},
		},
		{
			name:     "pack refs snapshot and digest",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				_ application.ApplyWizardGapsResult,
				_ application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts
SET pack_refs_json='["pack:unknown"]', pack_refs_digest=?
WHERE request_ref=?`, strings.Repeat("0", 64), request.RequestRef)
			},
		},
		{
			name:     "selections snapshot and digest",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				_ application.ApplyWizardGapsResult,
				_ application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts
SET selections_json='[{"dimension":"u1","option":"intake-option:wizard.u1.team"}]',
    selections_digest=?
WHERE request_ref=?`, strings.Repeat("0", 64), request.RequestRef)
			},
		},
		{
			name:     "request fingerprint",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				_ application.ApplyWizardGapsResult,
				_ application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts SET request_fingerprint=? WHERE request_ref=?`,
					strings.Repeat("0", 64), request.RequestRef)
			},
		},
		{
			name:     "evaluator",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				_ application.ApplyWizardGapsResult,
				_ application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts SET evaluator_version='tampered'
WHERE request_ref=?`, request.RequestRef)
			},
		},
		{
			name:     "source receipt",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_binding_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				result application.ApplyWizardGapsResult,
				_ application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts SET source_intake_receipt_ref=?
WHERE request_ref=?`, result.Record.Receipt.Ref, request.RequestRef)
			},
		},
		{
			name:     "outcome receipt",
			wantCode: "sqlite.recovery_v23_wizard_gaps_input_binding_invalid",
			mutate: func(
				t *testing.T,
				system *sqliteIntakeTestSystem,
				request application.ApplyWizardGapsRequest,
				_ application.ApplyWizardGapsResult,
				created application.IntakeRecord,
			) {
				mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_input_receipts SET outcome_receipt_ref=?
WHERE request_ref=?`, created.Receipt.Ref, request.RequestRef)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			suffix := strings.ReplaceAll(test.name, " ", "-")
			system := newSQLiteIntakeTestSystem(t)
			created := system.create(
				t, "request:wizard-input-corrupt-create-"+suffix,
			)
			service, err := application.NewWizardGapsService(
				system.repository,
			)
			sqliteTestNoError(t, err)
			request := sqliteWizardGapsRequest(
				t,
				system,
				"request:wizard-input-corrupt-mutation-"+suffix,
				created.Record.State.Revision(),
				intake.OriginForm,
			)
			result, err := service.ApplyWizardGaps(
				context.Background(), request,
			)
			sqliteTestNoError(t, err)
			rewriteRecoveryTrigger(
				t,
				system.repository.db,
				"wizard_gaps_input_receipts_immutable_update",
				func() {
					test.mutate(
						t, system, request, result, created.Record,
					)
				},
			)
			requireWizardGapsInputRecoveryError(
				t, system, test.wantCode,
			)
		})
	}
}

func TestV23WizardGapsInputRecoveryRejectsCoherentMutationSelectionTamper(
	t *testing.T,
) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-input-coherent-create")
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-input-coherent-mutation",
		created.Record.State.Revision(),
		intake.OriginForm,
	)
	_, err = service.ApplyWizardGaps(context.Background(), request)
	sqliteTestNoError(t, err)
	record := sqliteWizardGapsInputRecord(t, system, request)
	oldInputRef := record.Receipt.Ref
	record.Receipt.Selections = []application.WizardGapsSelectionInput{{
		Dimension: string(gaps.DimensionU1),
		Option:    "intake-option:wizard.u1.team",
	}}
	selectionsJSON, err := json.Marshal(record.Receipt.Selections)
	sqliteTestNoError(t, err)
	record.Receipt.SelectionsDigest = canonicalFingerprint(
		"orquesta.wizard.gaps.selections.v1",
		string(selectionsJSON),
	)
	record.Receipt.Ref = v23TestWizardGapsInputRef(record.Receipt)
	rewriteWizardGapsInputAndSnapshot(
		t,
		system.repository.db,
		nil,
		oldInputRef,
		record.Receipt,
		func(transaction *sql.Tx) {
			_, err := transaction.Exec(`UPDATE wizard_gaps_input_receipts
SET ref=?, selections_json=?, selections_digest=?
WHERE request_ref=?`,
				record.Receipt.Ref,
				string(selectionsJSON),
				record.Receipt.SelectionsDigest,
				request.RequestRef,
			)
			sqliteTestNoError(t, err)
		},
	)
	requireWizardGapsInputRecoveryError(
		t,
		system,
		"sqlite.recovery_v23_wizard_gaps_input_evaluation_invalid",
	)
}

func TestV23WizardGapsInputRejectsCoherentMutationFingerprintTamper(
	t *testing.T,
) {
	system := newSQLiteIntakeTestSystem(t)
	created := system.create(t, "request:wizard-input-fingerprint-create")
	service, err := application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-input-fingerprint-mutation",
		created.Record.State.Revision(),
		intake.OriginForm,
	)
	_, err = service.ApplyWizardGaps(context.Background(), request)
	sqliteTestNoError(t, err)
	record := sqliteWizardGapsInputRecord(t, system, request)
	oldInputRef := record.Receipt.Ref
	oldOutcomeRef := record.OutcomeRecord.Receipt.Ref
	record.OutcomeRecord.Receipt.RequestFingerprint = strings.Repeat("a", 64)
	record.OutcomeRecord.Receipt.Ref = v23TestIntakeReceiptRef(
		record.OutcomeRecord.Receipt,
	)
	record.Receipt.OutcomeReceiptRef = record.OutcomeRecord.Receipt.Ref
	record.Receipt.Ref = v23TestWizardGapsInputRef(record.Receipt)
	rewriteWizardGapsInputAndSnapshot(
		t,
		system.repository.db,
		[]string{
			"intake_receipts_immutable_update",
			"intake_states_revision_guard",
		},
		oldInputRef,
		record.Receipt,
		func(transaction *sql.Tx) {
			_, err = transaction.Exec(`
UPDATE intake_receipts
SET ref=?, request_fingerprint=?
WHERE ref=?`,
				record.OutcomeRecord.Receipt.Ref,
				record.OutcomeRecord.Receipt.RequestFingerprint,
				oldOutcomeRef,
			)
			sqliteTestNoError(t, err)
			_, err = transaction.Exec(`
UPDATE intake_states SET receipt_ref=? WHERE receipt_ref=?`,
				record.OutcomeRecord.Receipt.Ref,
				oldOutcomeRef,
			)
			sqliteTestNoError(t, err)
			_, err = transaction.Exec(`
UPDATE wizard_gaps_input_receipts
SET ref=?, outcome_receipt_ref=?
WHERE request_ref=?`,
				record.Receipt.Ref,
				record.OutcomeRecord.Receipt.Ref,
				request.RequestRef,
			)
			sqliteTestNoError(t, err)
		},
	)
	hydrated := sqliteWizardGapsInputRecord(t, system, request)
	if err := application.ValidateWizardGapsInputRecord(hydrated); err != nil {
		t.Fatalf("coherent structural record: %v", err)
	}
	if err := application.ValidateWizardGapsInputEvaluation(
		hydrated,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("coherent mutation fingerprint err=%v", err)
	}
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err == nil {
		t.Fatal("coherent mutation fingerprint passed recovery")
	}
}

func TestV23WizardGapsInputRecoveryRejectsCoherentNoOpOutcomeTamper(
	t *testing.T,
) {
	system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-input-coherent-noop",
		seed.Record.State.Revision(),
		intake.OriginForm,
	)
	result, err := service.ApplyWizardGaps(context.Background(), request)
	sqliteTestNoError(t, err)
	record := sqliteWizardGapsInputRecord(t, system, request)
	oldInputRef := record.Receipt.Ref
	outcome, found, err := system.repository.ReplayWizardGapsNoOp(
		context.Background(),
		application.WizardGapsNoOpReplayRequest{
			RequestRef:              record.Receipt.RequestRef,
			RequestFingerprint:      record.Receipt.RequestFingerprint,
			ActorRef:                record.Receipt.ActorRef,
			ProjectRef:              record.Receipt.ProjectRef,
			StateRef:                record.Receipt.StateRef,
			ExpectedRevision:        record.Receipt.ExpectedRevision,
			EvaluatorIdentity:       record.Receipt.EvaluatorIdentity,
			AuthorizationReceiptRef: record.Receipt.AuthorizationReceiptRef,
		},
	)
	sqliteTestNoError(t, err)
	if !found {
		t.Fatal("no-op outcome missing")
	}
	oldOutcomeRef := outcome.Ref
	outcome.EvaluationDigest = strings.Repeat("0", 64)
	outcome.Ref = v23TestWizardGapsNoOpRef(outcome)
	if outcome.Ref == oldOutcomeRef {
		t.Fatal("test tamper did not change outcome ref")
	}
	rewriteRecoveryTrigger(
		t,
		system.repository.db,
		"wizard_gaps_outcomes_immutable_update",
		func() {
			mustV10Exec(
				t,
				system.repository.db,
				`UPDATE wizard_gaps_outcomes
SET ref=?, evaluation_digest=?
WHERE ref=?`,
				outcome.Ref,
				outcome.EvaluationDigest,
				oldOutcomeRef,
			)
		},
	)
	record.Receipt.OutcomeReceiptRef = outcome.Ref
	record.Receipt.Ref = v23TestWizardGapsInputRef(record.Receipt)
	rewriteWizardGapsInputAndSnapshot(
		t,
		system.repository.db,
		nil,
		oldInputRef,
		record.Receipt,
		func(transaction *sql.Tx) {
			_, err := transaction.Exec(`UPDATE wizard_gaps_input_receipts
SET ref=?, outcome_receipt_ref=?
WHERE request_ref=?`,
				record.Receipt.Ref,
				outcome.Ref,
				request.RequestRef,
			)
			sqliteTestNoError(t, err)
		},
	)
	if result.RequestOutcome.ReceiptRef != oldOutcomeRef {
		t.Fatalf("result outcome=%s want=%s", result.RequestOutcome.ReceiptRef, oldOutcomeRef)
	}
	requireWizardGapsInputRecoveryError(
		t,
		system,
		"sqlite.recovery_v23_wizard_gaps_input_evaluation_invalid",
	)
}

func TestV23WizardGapsInputUpgradeFromSchema20DoesNotInventLegacyInput(
	t *testing.T,
) {
	system, service, seed := newSQLiteWizardGapsNoOpTestSystem(t)
	request := sqliteWizardGapsRequest(
		t,
		system,
		"request:wizard-input-upgrade-legacy-noop",
		seed.Record.State.Revision(),
		intake.OriginForm,
	)
	legacy, err := service.ApplyWizardGaps(context.Background(), request)
	sqliteTestNoError(t, err)
	if legacy.RequestOutcome.Kind !=
		application.WizardGapsRequestOutcomeNoOp {
		t.Fatalf("legacy=%+v", legacy)
	}
	record := sqliteWizardGapsInputRecord(t, system, request)
	outcome, found, err := system.repository.ReplayWizardGapsNoOp(
		context.Background(),
		application.WizardGapsNoOpReplayRequest{
			RequestRef:              record.Receipt.RequestRef,
			RequestFingerprint:      record.Receipt.RequestFingerprint,
			ActorRef:                record.Receipt.ActorRef,
			ProjectRef:              record.Receipt.ProjectRef,
			StateRef:                record.Receipt.StateRef,
			ExpectedRevision:        record.Receipt.ExpectedRevision,
			EvaluatorIdentity:       record.Receipt.EvaluatorIdentity,
			AuthorizationReceiptRef: record.Receipt.AuthorizationReceiptRef,
		},
	)
	sqliteTestNoError(t, err)
	if !found {
		t.Fatal("seed no-op outcome missing")
	}
	oldOutcomeRef := outcome.Ref
	outcome.RequestFingerprint = v23TestWizardGapsLegacyNoOpFingerprint(
		t, request,
	)
	outcome.Ref = v23TestWizardGapsNoOpRef(outcome)
	rewriteRecoveryTrigger(
		t,
		system.repository.db,
		"wizard_gaps_outcomes_immutable_update",
		func() {
			mustV10Exec(t, system.repository.db, `
UPDATE wizard_gaps_outcomes
SET ref=?, request_fingerprint=?
WHERE ref=?`,
				outcome.Ref,
				outcome.RequestFingerprint,
				oldOutcomeRef,
			)
		},
	)
	downgradeV35AgentEnvironmentLifecycleToCanonicalV34(t, system.repository.db)
	legacyTriggers := canonicalWizardSchemaTriggers(t, recoverySchemaV23WizardGaps,
		"action_consumption_effect_receipt_guard", "effect_receipts_causal_guard")
	mustV10Exec(
		t, system.repository.db,
		`ALTER TABLE agent_environment_receipts DROP COLUMN physical_manifest_digest;
ALTER TABLE agent_environment_receipts DROP COLUMN physical_manifest_ref;
DROP TABLE agent_placement_bindings; DROP TABLE agent_quota_observations;
DROP TABLE agent_capacity_transitions; DROP TABLE agent_capacity_reservations;
DROP TABLE agent_capacity_observations; DROP TABLE agent_environment_receipts;
DROP TABLE microvm_host_launch_authorities;
DROP INDEX effect_attempts_microvm_host_launch_scope_idx;
DROP TRIGGER executions_environment_preservation_write_once;
ALTER TABLE executions DROP COLUMN environment_preservation_required;
DROP TRIGGER outbox_recovery_effect_claim_guard;
DROP TRIGGER action_consumption_effect_receipt_guard;
ALTER TABLE outbox DROP COLUMN recovery_effect_attempt_ref;
DROP TRIGGER effect_receipts_causal_guard;
DROP TRIGGER effect_attempts_claim_lease_guard;
DROP TRIGGER effect_attempts_immutable_update;
ALTER TABLE effect_attempts DROP COLUMN claim_lease_until;
CREATE TRIGGER effect_attempts_immutable_update BEFORE UPDATE ON effect_attempts
BEGIN SELECT RAISE(ABORT, 'sqlite.effect_attempt_immutable'); END;
DROP TRIGGER work_item_authorities_egress_shape_guard;
ALTER TABLE work_item_authorities DROP COLUMN egress_policy_canonical_payload;
ALTER TABLE work_item_authorities DROP COLUMN egress_policy_payload_sha256;
ALTER TABLE work_item_authorities DROP COLUMN egress_policy_ref;
DROP TABLE wizard_gaps_input_receipts`,
	)
	for _, statement := range legacyTriggers {
		mustV10Exec(t, system.repository.db, statement)
	}
	mustV10Exec(
		t,
		system.repository.db,
		`DELETE FROM schema_migrations WHERE version IN (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		recoverySchemaV23, recoverySchemaV38Physical, recoverySchemaV38Capacity, recoverySchemaV38Claim, recoverySchemaV38Environment, recoverySchemaV38EnvironmentGate, recoverySchemaV38AttemptLease, recoverySchemaV38RecoveryClaim, recoverySchemaV38PreservationRatchet, recoverySchemaV38RecoveryRequeue,
		recoverySchemaV38EgressAuthority, recoverySchemaV38MicroVMHostLaunch, recoverySchemaV38MicroVMHostSession, recoverySchemaV38PhysicalManifest,
	)
	mustV10Exec(
		t,
		system.repository.db,
		`PRAGMA user_version=20`,
	)
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err != nil {
		t.Fatalf("schema20 recovery: %s", intakeTestErrorChain(err))
	}
	sqliteTestNoError(t, system.repository.Close())
	system.repository = openSQLiteIntakeTestRepository(t, system.path)
	service, err = application.NewWizardGapsService(system.repository)
	sqliteTestNoError(t, err)
	if _, err = service.ApplyWizardGaps(
		context.Background(), request,
	); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("legacy late attachment err=%v", err)
	}
	assertSQLiteWizardGapsInputEffects(
		t, system, request.RequestRef, 0, 0, 1,
	)
	if _, _, err := validateRecoveryDatabase(
		context.Background(), system.repository.db,
	); err != nil {
		t.Fatalf("schema21 recovery: %s", intakeTestErrorChain(err))
	}
}

func canonicalWizardSchemaTriggers(t *testing.T, version int, names ...string) []string {
	t.Helper()
	database, err := sql.Open(driverName, ":memory:")
	sqliteTestNoError(t, err)
	defer database.Close()
	database.SetMaxOpenConns(1)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	prefix, err := recoveryMigrationPrefix(migrations, version)
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(context.Background(), database, prefix))
	statements := make([]string, 0, len(names))
	for _, name := range names {
		var statement string
		sqliteTestNoError(t, database.QueryRow(`
SELECT sql FROM sqlite_schema WHERE type='trigger' AND name=?`, name).Scan(&statement))
		statements = append(statements, statement)
	}
	return statements
}

func sqliteWizardGapsInputRecord(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	request application.ApplyWizardGapsRequest,
) application.WizardGapsInputRecord {
	t.Helper()
	var fingerprint string
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT request_fingerprint
FROM wizard_gaps_input_receipts
WHERE actor_ref=? AND project_ref=? AND request_ref=?`,
		request.ActorRef.String(),
		request.ProjectRef.String(),
		request.RequestRef,
	).Scan(&fingerprint))
	record, found, err := system.repository.ReplayWizardGapsInput(
		context.Background(),
		application.WizardGapsInputReplayRequest{
			RequestRef: request.RequestRef, RequestFingerprint: fingerprint,
			ActorRef: request.ActorRef, ProjectRef: request.ProjectRef,
			StateRef: request.StateRef, ExpectedRevision: request.ExpectedRevision,
			EvaluatorIdentity:       request.EvaluatorIdentity,
			AuthorizationReceiptRef: request.AuthorizationReceipt.Ref(),
		},
	)
	sqliteTestNoError(t, err)
	if !found {
		t.Fatalf("input %s missing", request.RequestRef)
	}
	return record
}

func assertSQLiteWizardGapsInputEffects(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	requestRef string,
	wantInputs,
	wantMutations,
	wantOutcomes int,
) {
	t.Helper()
	var inputs, mutations, outcomes int
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_input_receipts WHERE request_ref=?`,
		requestRef,
	).Scan(&inputs))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM intake_receipts WHERE request_ref=?`,
		requestRef,
	).Scan(&mutations))
	sqliteTestNoError(t, system.repository.db.QueryRow(
		`SELECT COUNT(*) FROM wizard_gaps_outcomes WHERE request_ref=?`,
		requestRef,
	).Scan(&outcomes))
	if inputs != wantInputs || mutations != wantMutations ||
		outcomes != wantOutcomes {
		t.Fatalf(
			"inputs=%d/%d mutations=%d/%d outcomes=%d/%d",
			inputs,
			wantInputs,
			mutations,
			wantMutations,
			outcomes,
			wantOutcomes,
		)
	}
}

func requireWizardGapsInputRecoveryError(
	t *testing.T,
	system *sqliteIntakeTestSystem,
	code string,
) {
	t.Helper()
	if _, _, err := validateRecoveryDatabase(
		context.Background(),
		system.repository.db,
	); err == nil || !recoveryErrorContains(err, code) {
		t.Fatalf(
			"invalid input passed recovery: want=%s err=%s",
			code,
			intakeTestErrorChain(err),
		)
	}
}

func rewriteRecoveryTriggers(
	t *testing.T,
	database *sql.DB,
	names []string,
	mutate func(),
) {
	t.Helper()
	statements := make([]string, len(names))
	for index, name := range names {
		sqliteTestNoError(t, database.QueryRow(`
SELECT sql FROM sqlite_schema WHERE type='trigger' AND name=?`,
			name,
		).Scan(&statements[index]))
		mustV10Exec(
			t, database, "DROP TRIGGER "+quoteSQLiteIdentifier(name),
		)
	}
	mutate()
	for _, statement := range statements {
		mustV10Exec(t, database, statement)
	}
}

func rewriteWizardGapsInputAndSnapshot(
	t *testing.T,
	database *sql.DB,
	extraTriggers []string,
	oldInputRef string,
	receipt application.WizardGapsInputReceipt,
	mutate func(*sql.Tx),
) {
	t.Helper()
	triggers := append([]string(nil), extraTriggers...)
	triggers = append(triggers,
		"wizard_gaps_input_receipts_immutable_update",
		"wizard_gaps_result_snapshots_immutable_update",
	)
	rewriteRecoveryTriggers(t, database, triggers, func() {
		transaction, err := database.Begin()
		sqliteTestNoError(t, err)
		defer transaction.Rollback()
		_, err = transaction.Exec(`PRAGMA defer_foreign_keys=ON`)
		sqliteTestNoError(t, err)
		mutate(transaction)
		snapshotRef := "wizard-gaps-result-snapshot:" + canonicalFingerprint(
			"orquesta.wizard.gaps.result-snapshot-binding.v1",
			receipt.Ref,
			receipt.ResultSnapshot.Digest,
		)
		_, err = transaction.Exec(`
UPDATE wizard_gaps_result_snapshots SET ref=?,input_receipt_ref=?
WHERE input_receipt_ref=?`, snapshotRef, receipt.Ref, oldInputRef)
		sqliteTestNoError(t, err)
		sqliteTestNoError(t, transaction.Commit())
	})
}

func v23TestWizardGapsNoOpRef(
	outcome application.WizardGapsNoOpOutcome,
) string {
	return "wizard-gaps-outcome:" + canonicalFingerprint(
		"orquesta.wizard.gaps.noop.outcome.v1",
		outcome.RequestRef,
		outcome.RequestFingerprint,
		outcome.ActorRef.String(),
		outcome.ProjectRef.String(),
		string(outcome.StateRef),
		strconv.FormatUint(uint64(outcome.ExpectedRevision), 10),
		outcome.SourceIntakeReceiptRef,
		outcome.EvaluatorIdentity.Schema,
		outcome.EvaluatorIdentity.Version,
		outcome.EvaluatorIdentity.SemanticDigest,
		outcome.EvaluationDigest,
		outcome.AuthorizationReceiptRef,
	)
}

func v23TestWizardGapsInputRef(
	receipt application.WizardGapsInputReceipt,
) string {
	return "wizard-gaps-input:" + canonicalFingerprint(
		"orquesta.wizard.gaps.input-receipt.v1",
		receipt.RequestRef,
		receipt.RequestFingerprint,
		receipt.ActorRef.String(),
		receipt.ProjectRef.String(),
		string(receipt.StateRef),
		strconv.FormatUint(uint64(receipt.ExpectedRevision), 10),
		string(receipt.Origin),
		receipt.SourceIntakeReceiptRef,
		string(receipt.OutcomeKind),
		receipt.OutcomeReceiptRef,
		receipt.FactsDigest,
		receipt.PackRefsDigest,
		receipt.SelectionsDigest,
		receipt.EvaluatorIdentity.Schema,
		receipt.EvaluatorIdentity.Version,
		receipt.EvaluatorIdentity.SemanticDigest,
		receipt.AuthorizationReceiptRef,
	)
}

func v23TestWizardGapsLegacyNoOpFingerprint(
	t *testing.T,
	request application.ApplyWizardGapsRequest,
) string {
	t.Helper()
	factsJSON, err := json.Marshal(request.Facts)
	sqliteTestNoError(t, err)
	packRefs := make([]string, len(request.PackRefs))
	for index, ref := range request.PackRefs {
		packRefs[index] = ref.String()
	}
	packRefsJSON, err := json.Marshal(packRefs)
	sqliteTestNoError(t, err)
	return canonicalFingerprint(
		"orquesta.wizard.gaps.noop.request.v1",
		request.ActorRef.String(),
		request.ProjectRef.String(),
		string(request.StateRef),
		strconv.FormatUint(uint64(request.ExpectedRevision), 10),
		string(request.Origin),
		string(factsJSON),
		string(packRefsJSON),
		request.EvaluatorIdentity.Schema,
		request.EvaluatorIdentity.Version,
		request.EvaluatorIdentity.SemanticDigest,
		request.AuthorizationReceipt.Ref(),
	)
}
