package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
	"orquesta/internal/wizard/gaps"
)

func TestWizardGapsSnapshotSchemaMigrationIsLossless(t *testing.T) {
	ctx := context.Background()
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	database, err := sql.Open(driverName, ":memory:")
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	sqliteTestNoError(t, applyRecoveryMigrationPrefix(ctx, database, migrations[:recoverySchemaV23]))
	_, err = database.Exec(`PRAGMA foreign_keys=ON`)
	sqliteTestNoError(t, err)
	_, err = database.Exec(migrations[recoverySchemaV23WizardGapsSnapshot-1].sql)
	sqliteTestNoError(t, err)

	now := time.Date(2026, 8, 11, 8, 0, 0, 0, time.UTC)
	repository := &Repository{db: database, writer: database, now: func() time.Time { return now }}
	principal := testPrincipal(t, "principal:v23-snapshot", "actor:v23-snapshot", identity.PrincipalKindHuman)
	project := mustRef(t, "project:v23-snapshot", goal.NewProjectRef)
	provisionTestAccess(t, repository, principal, project, identity.RoleProjectOwner, now)
	intakeService, err := application.NewIntakeService(repository)
	sqliteTestNoError(t, err)
	system := &sqliteIntakeTestSystem{repository: repository, service: intakeService,
		principal: principal, project: project, stateRef: intake.Ref("intake:v23-snapshot"), now: now}
	created := system.create(t, "request:v23-snapshot-source")
	wizard, err := application.NewWizardGapsService(repository)
	sqliteTestNoError(t, err)
	request := sqliteWizardGapsRequest(t, system, "request:v23-snapshot", created.Record.State.Revision(), intake.OriginForm)
	result, err := wizard.ApplyWizardGaps(ctx, request)
	sqliteTestNoError(t, err)
	snapshotBytes, snapshotDigest, err := gaps.MarshalResultSnapshot(result.Evaluation)
	sqliteTestNoError(t, err)
	snapshotRef := "wizard-gaps-result-snapshot:" + strings.Repeat("a", 64)
	before := sqliteWizardGapsInputRecord(t, system, request)
	_, err = database.Exec(`DROP TABLE wizard_gaps_result_snapshots`)
	sqliteTestNoError(t, err)
	before.Receipt.ResultSnapshot = application.WizardGapsResultSnapshot{}

	sqliteTestNoError(t, applyMigrations(ctx, database))
	after := sqliteWizardGapsInputRecord(t, system, request)
	if !reflect.DeepEqual(before, after) || after.Receipt.ResultSnapshot.Ref != "" {
		t.Fatalf("schema 21 receipt changed: before=%+v after=%+v", before, after)
	}
	assertWizardGapsSnapshotSchemaRejects(t, database, `
INSERT INTO wizard_gaps_result_snapshots(
 ref,input_receipt_ref,evaluator_schema,evaluator_version,evaluator_semantic_digest,
 snapshot_schema,snapshot_digest,snapshot_bytes)
VALUES(?,?,?,?,?,?,?,?)`, snapshotRef, before.Receipt.Ref,
		before.Receipt.EvaluatorIdentity.Schema, before.Receipt.EvaluatorIdentity.Version,
		before.Receipt.EvaluatorIdentity.SemanticDigest, gaps.ResultSnapshotSchema,
		snapshotDigest, snapshotBytes)
	assertWizardGapsSnapshotSchemaRejects(t, database, `
INSERT INTO wizard_gaps_result_snapshots
SELECT ?,ref,expected_revision+1,evaluator_schema,evaluator_version,evaluator_semantic_digest,?,?,?
FROM wizard_gaps_input_receipts WHERE ref=?`, snapshotRef, gaps.ResultSnapshotSchema,
		snapshotDigest, snapshotBytes, before.Receipt.Ref)
	assertWizardGapsSnapshotSchemaRejects(t, database, `
INSERT INTO wizard_gaps_result_snapshots
SELECT ?,ref,expected_revision,evaluator_schema,evaluator_version,evaluator_semantic_digest,?,?,zeroblob(0)
FROM wizard_gaps_input_receipts WHERE ref=?`, snapshotRef, gaps.ResultSnapshotSchema,
		snapshotDigest, before.Receipt.Ref)
	assertWizardGapsSnapshotSchemaRejects(t, database, `
INSERT INTO wizard_gaps_result_snapshots
SELECT ?,ref,expected_revision,evaluator_schema,evaluator_version,evaluator_semantic_digest,?,?,zeroblob(1048577)
FROM wizard_gaps_input_receipts WHERE ref=?`, snapshotRef, gaps.ResultSnapshotSchema,
		snapshotDigest, before.Receipt.Ref)

	_, err = database.Exec(`INSERT INTO wizard_gaps_result_snapshots
SELECT ?,ref,expected_revision,evaluator_schema,evaluator_version,evaluator_semantic_digest,?,?,?
FROM wizard_gaps_input_receipts WHERE ref=?`, snapshotRef, gaps.ResultSnapshotSchema,
		snapshotDigest, snapshotBytes, before.Receipt.Ref)
	sqliteTestNoError(t, err)
	assertWizardGapsSnapshotSchemaRejects(t, database,
		`UPDATE wizard_gaps_result_snapshots SET snapshot_digest=?`, strings.Repeat("f", 64))
	sqliteTestNoError(t, applyMigrations(ctx, database))
	var version, snapshots int
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM wizard_gaps_result_snapshots`).Scan(&snapshots))
	if version != recoverySchemaLatest || snapshots != 1 {
		t.Fatalf("repeated migration version=%d snapshots=%d", version, snapshots)
	}
}

func assertWizardGapsSnapshotSchemaRejects(t *testing.T, database *sql.DB, statement string, arguments ...any) {
	t.Helper()
	if _, err := database.Exec(statement, arguments...); err == nil {
		t.Fatal("invalid Wizard gaps result snapshot accepted")
	}
}
