package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

func TestV17MigrationPreservesLegacyProvenanceWithoutSyntheticPass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-v1", "orquesta.sqlite")
	seedPopulatedV1Database(t, path)
	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if err != nil {
		t.Fatalf("migrate V1 to V17: %v cause=%v", err, errors.Unwrap(err))
	}
	t.Cleanup(func() { _ = repository.Close() })

	var provenance, observed, occurrences, syntheticTests, syntheticOutcomes, syntheticActions int
	queries := []struct {
		destination *int
		query       string
	}{
		{&provenance, `SELECT COUNT(*) FROM attestations WHERE kind='artifact_provenance'`},
		{&observed, `SELECT COUNT(*) FROM attestations WHERE verdict='observed'`},
		{&occurrences, `SELECT COUNT(*) FROM artifact_occurrences`},
		{&syntheticTests, `SELECT COUNT(*) FROM work_item_required_tests`},
		{&syntheticOutcomes, `SELECT COUNT(*) FROM attestation_test_outcomes`},
		{&syntheticActions, `SELECT COUNT(*) FROM outbox WHERE kind='attest_test'`},
	}
	for _, query := range queries {
		if err := repository.db.QueryRow(query.query).Scan(query.destination); err != nil {
			t.Fatal(err)
		}
	}
	if provenance != 1 || observed != 1 || occurrences != 1 ||
		syntheticTests != 0 || syntheticOutcomes != 0 || syntheticActions != 0 {
		t.Fatalf("legacy migration provenance=%d observed=%d occurrences=%d synthetic=%d/%d/%d",
			provenance, observed, occurrences, syntheticTests, syntheticOutcomes, syntheticActions)
	}
	closedRef := mustRef(t, "goal:v1-closed", goal.NewGoalRef)
	record, err := repository.GetGoal(context.Background(), closedRef)
	if err != nil || len(record.Attestations) != 1 ||
		record.Attestations[0].Kind != application.AttestationKindArtifactProvenance ||
		record.Attestations[0].Verdict != application.AttestationVerdictObserved ||
		len(record.Attestations[0].Tests) != 0 {
		t.Fatalf("legacy provenance read model=%+v err=%v", record.Attestations, err)
	}
	if _, _, err := validateRecoveryDatabase(context.Background(), repository.db); err != nil {
		t.Fatalf("migrated V17 recovery: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestV17MigrationRejectsAmbiguousLegacyArtifactProvenance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambiguous-v16", "orquesta.sqlite")
	seedPopulatedV1Database(t, path)
	migrateSQLiteFixtureToV16(t, path)
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	sqliteTestNoError(t, err)
	mustV10Exec(t, database, `
INSERT INTO attestations(ref,goal_ref,work_item_ref,execution_ref,artifact_ref,policy,accepted_at)
SELECT 'attestation:v1-closed:duplicate',goal_ref,work_item_ref,execution_ref,artifact_ref,policy,accepted_at
FROM attestations WHERE ref='attestation:v1-closed'`)
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 8,
	})
	if repository != nil {
		_ = repository.Close()
		t.Fatal("ambiguous V16 provenance migrated")
	}
	if !application.IsStateError(err, application.StateInvalid) ||
		!recoveryErrorContains(err, "sqlite.migration_v17_artifact_provenance_ambiguous") {
		t.Fatalf("ambiguous V16 provenance misclassified: %v cause=%v", err, errors.Unwrap(err))
	}
}

func TestRecoveryV17RejectsPartialAttestationLedger(t *testing.T) {
	t.Run("runtime duplicate occurrence", func(t *testing.T) {
		system := seedSQLiteV16Integrated(t)
		goals, err := system.repository.ListGoals(context.Background(), system.project, 10)
		if err != nil || len(goals) != 1 {
			t.Fatalf("list duplicate fixture goals=%d err=%v", len(goals), err)
		}
		record, err := system.repository.GetGoal(context.Background(), goals[0].Ref)
		sqliteTestNoError(t, err)
		var duplicate application.ArtifactRecord
		for _, artifact := range record.Artifacts {
			if artifact.Kind == application.ArtifactKindTestSubjectManifest {
				duplicate = artifact
				duplicate.OccurrenceRef += ":duplicate"
				break
			}
		}
		transaction, err := system.repository.writer.BeginTx(context.Background(), nil)
		sqliteTestNoError(t, err)
		defer transaction.Rollback()
		if err := insertArtifact(context.Background(), transaction, duplicate); !application.IsStateError(err, application.StateConflict) ||
			!recoveryErrorContains(err, "sqlite.artifact_occurrence_duplicate") {
			t.Fatalf("logical duplicate artifact occurrence runtime error=%v", err)
		}
	})
	cases := []struct {
		name, trigger, statement, code string
	}{
		{"recovery duplicate occurrence", "", `
INSERT INTO artifact_occurrences(
 occurrence_ref,kind,goal_ref,work_item_ref,execution_ref,artifact_ref,
 execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,created_at)
SELECT occurrence_ref || ':duplicate',kind,goal_ref,work_item_ref,execution_ref,artifact_ref,
 execution_attempt,plan_generation,work_item_generation,app_spec_generation,spec_hash,created_at
FROM artifact_occurrences WHERE kind='test_subject_manifest'`, "sqlite.recovery_v17_artifact_occurrence_duplicate"},
		{"non canonical CAS identity", "artifacts_immutable_update",
			`UPDATE artifacts SET digest='` + strings.Repeat("c", 64) + `' WHERE ref LIKE 'artifact:sha256:%'`,
			"sqlite.recovery_v17_artifact_cas_invalid"},
		{"missing outcome", "attestation_test_outcomes_immutable_delete",
			`DELETE FROM attestation_test_outcomes`, "sqlite.recovery_v17_attestation_outcomes_invalid"},
		{"receipt verdict mismatch", "effect_receipts_immutable_update",
			`UPDATE effect_receipts SET status='attested_failed' WHERE status='attested_passed'`,
			"sqlite.recovery_v17_attestation_ledger_invalid"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			system := seedSQLiteV16Integrated(t)
			mutate := func() { mustV10Exec(t, system.repository.db, test.statement) }
			if test.trigger == "" {
				mutate()
			} else {
				rewriteRecoveryTrigger(t, system.repository.db, test.trigger, mutate)
			}
			if _, _, err := validateRecoveryDatabase(context.Background(), system.repository.db); err == nil ||
				!recoveryErrorContains(err, test.code) {
				t.Fatalf("unsafe ledger accepted or misclassified: %s", sqliteTestErrorChain(err))
			}
		})
	}
}

func migrateSQLiteFixtureToV16(t *testing.T, path string) {
	t.Helper()
	database, err := sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
	sqliteTestNoError(t, err)
	database.SetMaxOpenConns(1)
	defer database.Close()
	if _, err := database.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	transaction, err := database.BeginTx(context.Background(), nil)
	sqliteTestNoError(t, err)
	defer transaction.Rollback()
	current, migrated, err := applyMigrationSteps(
		context.Background(), transaction, migrations[:recoverySchemaV16], 1,
	)
	if err != nil || current != recoverySchemaV16 || !migrated {
		t.Fatalf("migrate fixture to V16 current=%d migrated=%v err=%v", current, migrated, err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
}
