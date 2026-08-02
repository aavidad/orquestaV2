package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"orquesta/internal/ports"
)

func TestCommandAuditPersistsImmutableAdmissionOutcomeAndExactDigests(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 14, 0, 0, 123, time.UTC)
	repository, _ := openCommandAuditTestRepository(t, &now)
	record := commandAuditTestRecord("immutable")

	session, err := repository.Begin(ctx, record)
	if err != nil || !session.AdmissionCreated || session.Terminal != nil ||
		session.OutcomeRef != record.Ref+":outcome" || !session.Record.AdmittedAt.Equal(now) {
		t.Fatalf("admission=%+v err=%v", session, err)
	}
	replayed, err := repository.Begin(ctx, record)
	if err != nil || replayed.AdmissionCreated || replayed.Terminal != nil ||
		replayed.Record != session.Record || replayed.OutcomeRef != session.OutcomeRef {
		t.Fatalf("admission replay=%+v err=%v", replayed, err)
	}
	conflicting := record
	conflicting.InputDigest = strings.Repeat("d", 64)
	if _, err := repository.Begin(ctx, conflicting); !errors.Is(err, ports.ErrCommandAuditConflict) {
		t.Fatalf("changed input replay err=%v", err)
	}
	if _, err := repository.db.Exec(
		`UPDATE command_invocations SET input_digest=? WHERE ref=?`,
		strings.Repeat("e", 64), record.Ref,
	); err == nil || !strings.Contains(err.Error(), "sqlite.command_invocation_immutable") {
		t.Fatalf("admission mutation err=%v", err)
	}

	now = now.Add(time.Second)
	wantTerminal := ports.CommandAuditTerminal{
		OutcomeRef: session.OutcomeRef, Status: "completed",
		OutputDigest: strings.Repeat("f", 64),
	}
	completion, err := repository.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref, Terminal: wantTerminal,
	})
	if err != nil || !completion.Created || !completion.Terminal.CompletedAt.Equal(now) ||
		!sameCommandTerminal(wantTerminal, completion.Terminal) {
		t.Fatalf("completion=%+v err=%v", completion, err)
	}
	replayedCompletion, err := repository.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref, Terminal: wantTerminal,
	})
	if err != nil || replayedCompletion.Created || replayedCompletion.Terminal != completion.Terminal {
		t.Fatalf("completion replay=%+v err=%v", replayedCompletion, err)
	}
	conflictingTerminal := wantTerminal
	conflictingTerminal.OutputDigest = strings.Repeat("1", 64)
	if _, err := repository.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref, Terminal: conflictingTerminal,
	}); !errors.Is(err, ports.ErrCommandAuditConflict) {
		t.Fatalf("changed outcome replay err=%v", err)
	}
	for name, statement := range map[string]string{
		"update": `UPDATE command_outcomes SET output_digest='` + strings.Repeat("2", 64) + `'`,
		"delete": `DELETE FROM command_outcomes`,
	} {
		if _, err := repository.db.Exec(statement); err == nil ||
			!strings.Contains(err.Error(), "sqlite.command_outcome_immutable") {
			t.Fatalf("%s outcome err=%v", name, err)
		}
	}
	terminalReplay, err := repository.Begin(ctx, record)
	if err != nil || terminalReplay.AdmissionCreated || terminalReplay.Terminal == nil ||
		*terminalReplay.Terminal != completion.Terminal {
		t.Fatalf("terminal admission replay=%+v err=%v", terminalReplay, err)
	}
	var admissions, outcomes int
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM command_invocations`).Scan(&admissions))
	sqliteTestNoError(t, repository.db.QueryRow(`SELECT COUNT(*) FROM command_outcomes`).Scan(&outcomes))
	if admissions != 1 || outcomes != 1 {
		t.Fatalf("audit rows admission=%d outcome=%d", admissions, outcomes)
	}
}

func TestCommandAuditCrashFrontiersRecoverWithoutDoubleApplication(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 15, 0, 0, 0, time.UTC)
	repository, path := openCommandAuditTestRepository(t, &now)
	record := commandAuditTestRecord("crash")
	admission, err := repository.Begin(ctx, record)
	sqliteTestNoError(t, err)
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute)
	restarted := openFastTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: func() time.Time { return now },
	})
	pending, err := restarted.Begin(ctx, record)
	if err != nil || pending.AdmissionCreated || pending.Terminal != nil ||
		pending.Record.AdmittedAt != admission.Record.AdmittedAt {
		t.Fatalf("after admission crash=%+v err=%v", pending, err)
	}
	terminal := ports.CommandAuditTerminal{
		OutcomeRef: pending.OutcomeRef, Status: "failed", ErrorCode: "unavailable",
		OutputDigest: strings.Repeat("9", 64),
	}
	completed, err := restarted.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref, Terminal: terminal,
	})
	if err != nil || !completed.Created {
		t.Fatalf("after handler crash completion=%+v err=%v", completed, err)
	}
	if err := restarted.Close(); err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Minute)
	replayedRepository := openFastTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: func() time.Time { return now },
	})
	replayed, err := replayedRepository.Begin(ctx, record)
	if err != nil || replayed.Terminal == nil || *replayed.Terminal != completed.Terminal {
		t.Fatalf("after completion crash=%+v err=%v", replayed, err)
	}
	replayedCompletion, err := replayedRepository.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref, Terminal: terminal,
	})
	if err != nil || replayedCompletion.Created ||
		replayedCompletion.Terminal != completed.Terminal {
		t.Fatalf("lost response replay=%+v err=%v", replayedCompletion, err)
	}
	var admissions, outcomes int
	sqliteTestNoError(t, replayedRepository.db.QueryRow(
		`SELECT COUNT(*) FROM command_invocations`).Scan(&admissions))
	sqliteTestNoError(t, replayedRepository.db.QueryRow(
		`SELECT COUNT(*) FROM command_outcomes`).Scan(&outcomes))
	if admissions != 1 || outcomes != 1 {
		t.Fatalf("restart duplicated audit rows=%d/%d", admissions, outcomes)
	}
}

func TestCommandAuditReplayRejectsChangedPrincipalProjectCommandOrInput(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 16, 0, 0, 0, time.UTC)
	repository, _ := openCommandAuditTestRepository(t, &now)
	record := commandAuditTestRecord("identity")
	_, err := repository.Begin(ctx, record)
	sqliteTestNoError(t, err)

	for name, mutate := range map[string]func(*ports.CommandAuditRecord){
		"principal": func(value *ports.CommandAuditRecord) { value.PrincipalRef = "principal:changed" },
		"project":   func(value *ports.CommandAuditRecord) { value.ProjectRef = "project:changed" },
		"command":   func(value *ports.CommandAuditRecord) { value.CommandID = "orquesta.goals.list" },
		"input":     func(value *ports.CommandAuditRecord) { value.InputDigest = strings.Repeat("d", 64) },
		"schema":    func(value *ports.CommandAuditRecord) { value.SchemaDigest = strings.Repeat("1", 64) },
		"execution": func(value *ports.CommandAuditRecord) { value.AuthenticatedExecutionRef = "execution:changed" },
		"registry":  func(value *ports.CommandAuditRecord) { value.RegistryDigest = "sha256:" + strings.Repeat("2", 64) },
		"ref":       func(value *ports.CommandAuditRecord) { value.Ref = "command-audit:changed" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := record
			mutate(&changed)
			if _, err := repository.Begin(ctx, changed); !errors.Is(err, ports.ErrCommandAuditConflict) {
				t.Fatalf("changed %s replay err=%v", name, err)
			}
		})
	}

	otherScope := record
	otherScope.Ref = "command-audit:" + strings.Repeat("3", 64)
	otherScope.PrincipalRef = "principal:other"
	session, err := repository.Begin(ctx, otherScope)
	if err != nil || !session.AdmissionCreated {
		t.Fatalf("different scope=%+v err=%v", session, err)
	}
}

func TestSQLiteCommandAuditConcurrentExactReplayConverges(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 16, 30, 0, 0, time.UTC)
	repository, _ := openCommandAuditTestRepository(t, &now)
	record := commandAuditTestRecord("concurrent")
	const workers = 16

	begin := make(chan struct{})
	sessions := make(chan ports.CommandAuditSession, workers)
	failures := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			<-begin
			session, err := repository.Begin(ctx, record)
			if err != nil {
				failures <- err
				return
			}
			sessions <- session
		}()
	}
	close(begin)
	wait.Wait()
	close(sessions)
	close(failures)
	for err := range failures {
		t.Errorf("concurrent Begin: %v", err)
	}
	createdAdmissions := 0
	var admittedAt time.Time
	for session := range sessions {
		if session.AdmissionCreated {
			createdAdmissions++
		}
		if admittedAt.IsZero() {
			admittedAt = session.Record.AdmittedAt
		}
		if session.Terminal != nil || session.OutcomeRef != record.Ref+":outcome" ||
			session.Record.AdmittedAt != admittedAt {
			t.Errorf("divergent concurrent admission: %+v", session)
		}
	}
	if createdAdmissions != 1 {
		t.Fatalf("created admissions=%d, want 1", createdAdmissions)
	}

	now = now.Add(time.Second)
	terminal := ports.CommandAuditTerminal{
		OutcomeRef: record.Ref + ":outcome", Status: "completed",
		OutputDigest: strings.Repeat("8", 64),
	}
	complete := make(chan struct{})
	completions := make(chan ports.CommandAuditCompletion, workers)
	failures = make(chan error, workers)
	wait.Add(workers)
	for range workers {
		go func() {
			defer wait.Done()
			<-complete
			completion, err := repository.Complete(ctx, ports.CommandAuditCompletionRequest{
				RecordRef: record.Ref, Terminal: terminal,
			})
			if err != nil {
				failures <- err
				return
			}
			completions <- completion
		}()
	}
	close(complete)
	wait.Wait()
	close(completions)
	close(failures)
	for err := range failures {
		t.Errorf("concurrent Complete: %v", err)
	}
	createdOutcomes := 0
	var completedAt time.Time
	for completion := range completions {
		if completion.Created {
			createdOutcomes++
		}
		if completedAt.IsZero() {
			completedAt = completion.Terminal.CompletedAt
		}
		if !sameCommandTerminal(terminal, completion.Terminal) ||
			completion.Terminal.CompletedAt != completedAt {
			t.Errorf("divergent concurrent outcome: %+v", completion)
		}
	}
	if createdOutcomes != 1 {
		t.Fatalf("created outcomes=%d, want 1", createdOutcomes)
	}
	var admissions, outcomes int
	sqliteTestNoError(t, repository.db.QueryRow(
		`SELECT COUNT(*) FROM command_invocations`).Scan(&admissions))
	sqliteTestNoError(t, repository.db.QueryRow(
		`SELECT COUNT(*) FROM command_outcomes`).Scan(&outcomes))
	if admissions != 1 || outcomes != 1 {
		t.Fatalf("concurrent audit rows=%d/%d", admissions, outcomes)
	}
}

func TestRecoveryRejectsTamperedCommandAuditCrossLinksAndUnsupportedSchema(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 17, 0, 0, 0, time.UTC)
	repository, _ := openCommandAuditTestRepository(t, &now)
	record := commandAuditTestRecord("recovery")
	session, err := repository.Begin(ctx, record)
	sqliteTestNoError(t, err)
	now = now.Add(time.Second)
	_, err = repository.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: record.Ref,
		Terminal: ports.CommandAuditTerminal{
			OutcomeRef: session.OutcomeRef, Status: "rejected", ErrorCode: "forbidden",
			OutputDigest: strings.Repeat("4", 64),
		},
	})
	sqliteTestNoError(t, err)

	rewriteRecoveryTrigger(t, repository.db, "command_invocations_immutable_update", func() {
		mustV10Exec(t, repository.db,
			`UPDATE command_invocations SET admitted_at=admitted_at+? WHERE ref=?`,
			int64(2*time.Second), record.Ref)
	})
	if _, _, err := validateRecoveryDatabase(ctx, repository.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v20_command_outcome_invalid") {
		t.Fatalf("broken command temporal link passed recovery: %v", err)
	}

	missingFact, _ := openCommandAuditTestRepository(t, &now)
	governance := commandAuditTestRecord("missing-governance-fact")
	governance.CommandID = "orquesta.director.plan.propose"
	governance.ReplayMode = ports.CommandReplayApplicationReceipt
	governanceSession, err := missingFact.Begin(ctx, governance)
	sqliteTestNoError(t, err)
	_, err = missingFact.Complete(ctx, ports.CommandAuditCompletionRequest{
		RecordRef: governance.Ref,
		Terminal: ports.CommandAuditTerminal{
			OutcomeRef: governanceSession.OutcomeRef, Status: "completed",
			OutputDigest: strings.Repeat("5", 64),
		},
	})
	sqliteTestNoError(t, err)
	if _, _, err := validateRecoveryDatabase(ctx, missingFact.db); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_v20_command_governance_cross_link_invalid") {
		t.Fatalf("missing governance cross-link passed recovery: %v", err)
	}

	unsupported, path := openCommandAuditTestRepository(t, &now)
	if err := unsupported.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := openCommandAuditRawDatabase(path)
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	_, err = raw.Exec(`PRAGMA user_version=` + strconv.Itoa(recoverySchemaV38Environment+1))
	sqliteTestNoError(t, err)
	if _, _, err := validateRecoveryDatabase(ctx, raw); err == nil ||
		!recoveryErrorContains(err, "sqlite.recovery_schema_version_invalid") {
		t.Fatalf("unsupported command audit schema passed recovery: %v", err)
	}
}

func TestSQLiteCommandRegistryMigrationAndRestartPreserveHistoricalDigest(t *testing.T) {
	path := emptySQLiteV18Database(t)
	database, err := openCommandAuditRawDatabase(path)
	sqliteTestNoError(t, err)
	migrations, err := loadMigrations()
	sqliteTestNoError(t, err)
	sqliteTestNoError(t, migrateSQLiteV19Prefix(
		t, database, migrations, recoverySchemaV18, recoverySchemaV19))
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = repository.Close() })
	var version, v19Receipts, v20Receipts, auditTables int
	sqliteTestNoError(t, repository.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, repository.db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV19).Scan(&v19Receipts))
	sqliteTestNoError(t, repository.db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name='015_command_registry.sql'`,
		recoverySchemaV20).Scan(&v20Receipts))
	sqliteTestNoError(t, repository.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name IN ('command_invocations','command_outcomes')`,
	).Scan(&auditTables))
	if version != recoverySchemaV38Environment || v19Receipts != 1 || v20Receipts != 1 || auditTables != 2 {
		t.Fatalf("V20 migration version=%d receipts=%d/%d tables=%d",
			version, v19Receipts, v20Receipts, auditTables)
	}
	historical := commandAuditTestRecord("migration-history")
	admitted, err := repository.Begin(context.Background(), historical)
	sqliteTestNoError(t, err)
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = restarted.Close() })
	replayed, err := restarted.Begin(context.Background(), historical)
	if err != nil || replayed.AdmissionCreated ||
		replayed.Record.RegistryDigest != admitted.Record.RegistryDigest ||
		replayed.Record.SchemaDigest != admitted.Record.SchemaDigest ||
		replayed.Record.InputDigest != admitted.Record.InputDigest {
		t.Fatalf("historical digest restart=%+v err=%v", replayed, err)
	}
	sqliteTestNoError(t, restarted.db.QueryRow(
		`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV20).Scan(&v20Receipts))
	if v20Receipts != 1 {
		t.Fatalf("V20 restart migration receipts=%d", v20Receipts)
	}
}

func openCommandAuditTestRepository(t *testing.T, now *time.Time) (*Repository, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "orquesta.sqlite")
	repository := openFastTestRepository(t, Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
		Now: func() time.Time { return *now },
	})
	return repository, path
}

func openCommandAuditRawDatabase(path string) (*sql.DB, error) {
	return sql.Open(driverName, buildDSN(path, testBusyTimeout.Milliseconds()))
}

func commandAuditTestRecord(suffix string) ports.CommandAuditRecord {
	return ports.CommandAuditRecord{
		Ref: "command-audit:" + suffix, CommandID: "orquesta.goals.get", CommandVersion: "1",
		RegistryDigest: "sha256:" + strings.Repeat("a", 64),
		SchemaDigest:   strings.Repeat("b", 64), RequestRef: "request:" + suffix,
		InputDigest: strings.Repeat("c", 64), PrincipalRef: "principal:" + suffix,
		ProjectRef: "project:" + suffix, ReplayMode: ports.CommandReplayReadReexecute,
		Status: "admitted",
	}
}
