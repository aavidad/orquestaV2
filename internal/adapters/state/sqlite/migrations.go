package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const appSpecsBackfillMarker = "-- orquesta:go-backfill app_specs"

type migration struct {
	version  int
	name     string
	checksum string
	sql      string
	preSQL   string
	postSQL  string
	backfill bool
}

func applyMigrations(ctx context.Context, database *sql.DB) error {
	migrations, err := loadMigrations()
	if err != nil {
		return invalid(err)
	}
	connection, err := database.Conn(ctx)
	if err != nil {
		return mapDatabaseError(err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return mapDatabaseError(err)
	}
	var foreignKeysEnabled int
	if err := connection.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeysEnabled); err != nil {
		return mapDatabaseError(err)
	}
	if foreignKeysEnabled != 0 {
		return invalid(fmt.Errorf("sqlite.foreign_keys_disable_failed"))
	}
	foreignKeysDisabled := true
	defer func() {
		if foreignKeysDisabled {
			_, _ = connection.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
		}
	}()
	transaction, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return mapDatabaseError(err)
	}
	defer func() { _ = transaction.Rollback() }()

	var current int
	if err := transaction.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return mapDatabaseError(err)
	}
	if current > migrations[len(migrations)-1].version {
		return invalid(fmt.Errorf("sqlite.schema_newer_than_binary"))
	}
	current, migrated, err := applyMigrationSteps(ctx, transaction, migrations, current)
	if err != nil {
		return err
	}
	// Validate only after the complete schema chain. Domain snapshot readers
	// intentionally understand the latest schema, not transient migration
	// layouts such as V3 before phase contracts are added by V4.
	if migrated {
		if err := validateMigratedGoalRecords(ctx, transaction); err != nil {
			return invalid(fmt.Errorf("sqlite.migrated_goal_invalid: %w", err))
		}
	}
	if err := verifyAppliedMigrations(ctx, transaction, migrations, current); err != nil {
		return err
	}
	if err := verifyForeignKeys(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return mapDatabaseError(err)
	}
	if _, err := connection.ExecContext(context.Background(), "PRAGMA foreign_keys = ON"); err != nil {
		return mapDatabaseError(err)
	}
	if err := connection.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeysEnabled); err != nil {
		return mapDatabaseError(err)
	}
	if foreignKeysEnabled != 1 {
		return invalid(fmt.Errorf("sqlite.foreign_keys_disabled"))
	}
	foreignKeysDisabled = false
	return nil
}

func applyMigrationSteps(
	ctx context.Context,
	transaction *sql.Tx,
	migrations []migration,
	current int,
) (int, bool, error) {
	migrated := false
	for _, migration := range migrations {
		if migration.version <= current {
			continue
		}
		if migration.version == 5 {
			if err := validateAtomicStateMigrationSource(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if migration.version == recoverySchemaV15 && current == recoverySchemaV14 {
			if err := validateRecoveryV14Controls(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if migration.version == recoverySchemaV16 && current == recoverySchemaV15 {
			if err := validateRecoveryV15Governance(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if migration.version == recoverySchemaV17 && current == recoverySchemaV16 {
			if err := validateRecoveryV16TestAttestorMigrationSource(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if migration.version == recoverySchemaV18 && current == recoverySchemaV17 {
			if err := validateRecoveryV17TestAttestor(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
			if err := validateV18LegacyReviewUpgradeSource(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if migration.version == recoverySchemaV19 && current == recoverySchemaV18 {
			if err := validateV19CouncilUpgradeSource(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if _, err := transaction.ExecContext(ctx, migration.preSQL); err != nil {
			return current, migrated, mapDatabaseError(err)
		}
		if migration.backfill {
			if err := backfillAppSpecs(ctx, transaction); err != nil {
				return current, migrated, invalid(err)
			}
		}
		if strings.TrimSpace(migration.postSQL) != "" {
			if _, err := transaction.ExecContext(ctx, migration.postSQL); err != nil {
				return current, migrated, mapDatabaseError(err)
			}
		}
		if _, err := transaction.ExecContext(
			ctx,
			"INSERT INTO schema_migrations(version, name, checksum) VALUES (?, ?, ?)",
			migration.version,
			migration.name,
			migration.checksum,
		); err != nil {
			return current, migrated, mapDatabaseError(err)
		}
		if _, err := transaction.ExecContext(ctx, "PRAGMA user_version = "+strconv.Itoa(migration.version)); err != nil {
			return current, migrated, mapDatabaseError(err)
		}
		current = migration.version
		migrated = true
	}
	return current, migrated, nil
}

// V18 has no durable Council policy. Any write-scoped item that may still
// progress, or whose external effect frontier is unsettled, must finish under
// V18. V19 never guesses policy or whether an attempted effect happened.
func validateV19CouncilUpgradeSource(ctx context.Context, transaction *sql.Tx) error {
	var live, activeActions, unknownAttempts int
	err := transaction.QueryRowContext(ctx, `
WITH write_scoped AS (
 SELECT DISTINCT goal_ref,work_item_ref FROM work_item_write_scopes
)
SELECT
 (SELECT COUNT(*) FROM work_items item JOIN write_scoped scope
   ON scope.goal_ref=item.goal_ref AND scope.work_item_ref=item.ref
   WHERE item.state NOT IN ('succeeded','failed','skipped','canceled','superseded')),
 (SELECT COUNT(*) FROM outbox action JOIN write_scoped scope
   ON scope.goal_ref=action.goal_ref AND scope.work_item_ref=action.work_item_ref
   WHERE action.completed_at IS NULL AND action.retired_at IS NULL AND action.quarantined_at IS NULL),
 (SELECT COUNT(*) FROM effect_attempts attempt JOIN write_scoped scope
   ON scope.goal_ref=attempt.goal_ref AND scope.work_item_ref=attempt.work_item_ref
   LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
   WHERE receipt.ref IS NULL)`).Scan(&live, &activeActions, &unknownAttempts)
	if err != nil {
		return err
	}
	if live+activeActions+unknownAttempts != 0 {
		return fmt.Errorf("sqlite.v19_upgrade_requires_v18_council_policy:live=%d:active_actions=%d:unknown_attempts=%d",
			live, activeActions, unknownAttempts)
	}
	return nil
}

func validateAtomicStateMigrationSource(ctx context.Context, transaction *sql.Tx) error {
	var incomplete int
	if err := transaction.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM outbox
WHERE completed_at IS NOT NULL
  AND (claim_token IS NULL OR claimed_by IS NULL OR claimed_until IS NULL OR attempt <= 0)`).Scan(&incomplete); err != nil {
		return err
	}
	if incomplete != 0 {
		return fmt.Errorf("sqlite.legacy_completed_action_claim_missing")
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	result := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		separator := strings.IndexByte(entry.Name(), '_')
		if separator <= 0 {
			return nil, fmt.Errorf("sqlite.migration_name_invalid")
		}
		version, err := strconv.Atoi(entry.Name()[:separator])
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("sqlite.migration_version_invalid")
		}
		content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(content)
		loaded := migration{
			version: version, name: entry.Name(),
			checksum: "sha256:" + hex.EncodeToString(digest[:]), sql: string(content),
			preSQL: string(content),
		}
		parts := strings.Split(string(content), appSpecsBackfillMarker)
		if version == 3 {
			if len(parts) != 2 {
				return nil, fmt.Errorf("sqlite.app_specs_backfill_marker_invalid")
			}
			loaded.preSQL, loaded.postSQL, loaded.backfill = parts[0], parts[1], true
		} else if len(parts) != 1 {
			return nil, fmt.Errorf("sqlite.backfill_marker_unexpected")
		}
		result = append(result, loaded)
	}
	sort.Slice(result, func(left, right int) bool { return result[left].version < result[right].version })
	if len(result) == 0 {
		return nil, fmt.Errorf("sqlite.migrations_missing")
	}
	for index, migration := range result {
		if migration.version != index+1 {
			return nil, fmt.Errorf("sqlite.migration_sequence_invalid")
		}
	}
	return result, nil
}

type legacyGoalRow struct {
	ref, requestRef, requestFingerprint string
	intent                              goal.IntentManifestSnapshot
	actorRef, projectRef, state         string
	revision                            int64
	createdAt                           int64
	startedAt, closedAt                 sql.NullInt64
	planGeneration                      int64
}

func backfillAppSpecs(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, `
SELECT g.ref, g.request_ref, g.request_fingerprint,
       i.ref, i.actor_ref, i.project_ref, i.statement, i.submitted_at, i.hash,
       g.actor_ref, g.project_ref, g.state, g.revision,
       g.created_at, g.started_at, g.closed_at, g.plan_generation
FROM goals g
JOIN intents i ON i.ref = g.intent_ref
ORDER BY g.ref`)
	if err != nil {
		return err
	}
	var legacy []legacyGoalRow
	for rows.Next() {
		var row legacyGoalRow
		var submittedAt int64
		if err := rows.Scan(
			&row.ref, &row.requestRef, &row.requestFingerprint,
			&row.intent.Ref, &row.intent.ActorRef, &row.intent.ProjectRef,
			&row.intent.Statement, &submittedAt, &row.intent.Hash,
			&row.actorRef, &row.projectRef, &row.state, &row.revision,
			&row.createdAt, &row.startedAt, &row.closedAt, &row.planGeneration,
		); err != nil {
			_ = rows.Close()
			return err
		}
		row.intent.SubmittedAt = time.Unix(0, submittedAt).UTC()
		legacy = append(legacy, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	for _, row := range legacy {
		manifest, err := goal.RestoreIntentManifest(row.intent)
		if err != nil {
			return err
		}
		if row.actorRef != manifest.Actor().String() || row.projectRef != manifest.Project().String() {
			return fmt.Errorf("sqlite.legacy_goal_scope_invalid")
		}
		if _, err := goal.NewGoalRef(row.ref); err != nil {
			return err
		}
		specRef, err := goal.NewAppSpecRef("app-spec:migrated:" + row.ref)
		if err != nil {
			return err
		}
		spec, err := goal.NewInitialAppSpec(goal.AppSpecInput{
			Ref: specRef, Intent: manifest, Objective: strings.TrimSpace(manifest.Statement()),
			Reason: "migration.v2_to_v3", ConfirmedBy: manifest.Actor(),
			ConfirmedAt: time.Unix(0, row.createdAt).UTC(),
		})
		if err != nil {
			return err
		}
		if err := insertAppSpecSnapshot(ctx, transaction, spec.Snapshot()); err != nil {
			return err
		}
		if _, err := transaction.ExecContext(ctx, `
INSERT INTO goals_v3(
    ref, request_ref, request_fingerprint, app_spec_ref, actor_ref, project_ref,
    state, revision, created_at, started_at, closed_at, plan_generation
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			row.ref, row.requestRef, row.requestFingerprint, spec.Ref().String(),
			row.actorRef, row.projectRef, row.state, row.revision, row.createdAt,
			storedTime(restoredTime(row.startedAt)), storedTime(restoredTime(row.closedAt)),
			row.planGeneration,
		); err != nil {
			return err
		}
	}
	return nil
}

// validateMigratedGoalRecords crosses the durable migration boundary through
// the same canonical reader used at runtime. readGoalRecord rebuilds the full
// AppSpec/Goal snapshot and delegates lifecycle validation to goal.RestoreGoal;
// this function only adds adapter-owned record binding checks.
func validateMigratedGoalRecords(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, "SELECT ref FROM goals ORDER BY ref")
	if err != nil {
		return mapDatabaseError(err)
	}
	var refs []string
	for rows.Next() {
		var ref string
		if err := rows.Scan(&ref); err != nil {
			_ = rows.Close()
			return mapDatabaseError(err)
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return mapDatabaseError(err)
	}
	if err := rows.Close(); err != nil {
		return mapDatabaseError(err)
	}
	for _, ref := range refs {
		record, err := readGoalRecord(ctx, transaction, ref)
		if err != nil {
			return err
		}
		if err := validateGoalRecordConsistency(record, ref); err != nil {
			return err
		}
	}
	return nil
}

func verifyForeignKeys(ctx context.Context, transaction *sql.Tx) error {
	rows, err := transaction.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return mapDatabaseError(err)
	}
	defer rows.Close()
	if rows.Next() {
		var table, parent string
		var rowID sql.NullInt64
		var foreignKeyID int64
		if err := rows.Scan(&table, &rowID, &parent, &foreignKeyID); err != nil {
			return mapDatabaseError(err)
		}
		return invalid(fmt.Errorf("sqlite.foreign_key_check_failed:%s:%s", table, parent))
	}
	if err := rows.Err(); err != nil {
		return mapDatabaseError(err)
	}
	return nil
}

func verifyAppliedMigrations(
	ctx context.Context,
	transaction *sql.Tx,
	migrations []migration,
	current int,
) error {
	if current == 0 {
		return nil
	}
	rows, err := transaction.QueryContext(ctx, "SELECT version, name, checksum FROM schema_migrations ORDER BY version")
	if err != nil {
		return mapDatabaseError(err)
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		var version int
		var name string
		var checksum string
		if err := rows.Scan(&version, &name, &checksum); err != nil {
			return mapDatabaseError(err)
		}
		if index >= len(migrations) || migrations[index].version != version || migrations[index].name != name ||
			migrations[index].checksum != checksum {
			return invalid(fmt.Errorf("sqlite.migration_history_invalid"))
		}
		index++
	}
	if err := rows.Err(); err != nil {
		return mapDatabaseError(err)
	}
	if index != current {
		return invalid(fmt.Errorf("sqlite.migration_history_incomplete"))
	}
	return nil
}
