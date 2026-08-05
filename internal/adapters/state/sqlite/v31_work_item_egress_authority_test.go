package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
)

type sqliteEgressPolicyResolver struct {
	authority application.EgressPolicyAuthority
}

func (resolver sqliteEgressPolicyResolver) ResolveEgressPolicy(
	context.Context,
	application.EgressPolicyRef,
) (application.EgressPolicyAuthority, error) {
	return resolver.authority, nil
}

func TestV31WorkItemEgressAuthoritySurvivesRestartByteForByte(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	payload := string([]byte{'{', '"', 'v', '"', ':', '1', ',', '"', 'x', '"', ':', '"', 0, 0xff, '"', '}'})
	authority := sqliteEgressAuthority(t, "egress-policy:sqlite-v31", payload)
	system.orchestrator = newSQLiteV31EgressOrchestrator(t, system, sqliteEgressPolicyResolver{authority: authority})
	created := submitSQLiteV31Egress(t, system, "request:v31-egress-restart", authority.PolicyRef.String())
	workItemRef := created.Record.WorkItemAuthorities[0].WorkItemRef
	assertSQLiteV31EgressAuthority(t, system.repository, created.Record.Goal.Ref(), workItemRef, authority)

	sqliteTestNoError(t, system.repository.Close())
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertSQLiteV31EgressAuthority(t, reopened, created.Record.Goal.Ref(), workItemRef, authority)

	var payloadType string
	var payloadBytes []byte
	sqliteTestNoError(t, reopened.db.QueryRow(`
SELECT typeof(egress_policy_canonical_payload),egress_policy_canonical_payload
FROM work_item_authorities WHERE work_item_ref=?`, workItemRef.String()).Scan(&payloadType, &payloadBytes))
	if payloadType != "blob" || string(payloadBytes) != payload {
		t.Fatalf("canonical payload storage type=%s bytes=%x want=%x", payloadType, payloadBytes, []byte(payload))
	}

	system.repository = reopened
	system.orchestrator = newSQLiteV31EgressOrchestrator(t, system, nil)
	replayed := submitSQLiteV31Egress(t, system, "request:v31-egress-restart", authority.PolicyRef.String())
	if replayed.Created || replayed.Record.WorkItemAuthorities[0].EgressPolicy != authority {
		t.Fatalf("restart replay did not preserve historical authority: %+v", replayed)
	}
}

func TestV31WorkItemEgressAuthorityMutationFailsClosed(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	authority := sqliteEgressAuthority(t, "egress-policy:sqlite-v31-mutation", `{"destinations":["example.org"]}`)
	system.orchestrator = newSQLiteV31EgressOrchestrator(t, system, sqliteEgressPolicyResolver{authority: authority})
	created := submitSQLiteV31Egress(t, system, "request:v31-egress-mutation", authority.PolicyRef.String())
	workItemRef := created.Record.WorkItemAuthorities[0].WorkItemRef

	mutations := []struct {
		name      string
		statement string
		value     any
	}{
		{"policy ref", `UPDATE work_item_authorities SET egress_policy_ref=? WHERE work_item_ref=?`, "egress-policy:crossed"},
		{"payload digest", `UPDATE work_item_authorities SET egress_policy_payload_sha256=? WHERE work_item_ref=?`, strings.Repeat("a", 64)},
		{"canonical payload", `UPDATE work_item_authorities SET egress_policy_canonical_payload=? WHERE work_item_ref=?`, []byte(`{"destinations":[]}`)},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			if _, err := system.repository.db.Exec(mutation.statement, mutation.value, workItemRef.String()); err == nil {
				t.Fatalf("immutable egress authority mutation accepted")
			}
			assertSQLiteV31EgressAuthority(t, system.repository, created.Record.Goal.Ref(), workItemRef, authority)
		})
	}
}

func TestV31RecoveryRejectsDigestOrPayloadCorruption(t *testing.T) {
	tests := map[string]struct {
		column string
		value  any
	}{
		"digest":  {column: "egress_policy_payload_sha256", value: strings.Repeat("a", 64)},
		"payload": {column: "egress_policy_canonical_payload", value: []byte(`{"destinations":[]}`)},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			system := newSQLiteV15System(t, 1)
			authority := sqliteEgressAuthority(t, "egress-policy:sqlite-v31-corrupt-"+name, `{"destinations":["example.org"]}`)
			system.orchestrator = newSQLiteV31EgressOrchestrator(t, system, sqliteEgressPolicyResolver{authority: authority})
			created := submitSQLiteV31Egress(t, system, "request:v31-egress-corrupt-"+name, authority.PolicyRef.String())
			workItemRef := created.Record.WorkItemAuthorities[0].WorkItemRef
			sqliteTestNoError(t, execSQLiteV31IgnoringAuthorityImmutability(
				system.repository, `UPDATE work_item_authorities SET `+test.column+`=? WHERE work_item_ref=?`,
				test.value, workItemRef.String(),
			))
			if _, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref()); err == nil ||
				!strings.Contains(sqliteTestErrorChain(err), "sqlite.work_item_egress_authority_invalid") {
				t.Fatalf("corrupted egress authority recovered: %s", sqliteTestErrorChain(err))
			}
		})
	}
}

func TestV31RecoveryRejectsPartialNullEgressAuthority(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	authority := sqliteEgressAuthority(t, "egress-policy:sqlite-v31-partial-null", `{"destinations":["example.org"]}`)
	system.orchestrator = newSQLiteV31EgressOrchestrator(t, system, sqliteEgressPolicyResolver{authority: authority})
	created := submitSQLiteV31Egress(t, system, "request:v31-egress-partial-null", authority.PolicyRef.String())
	workItemRef := created.Record.WorkItemAuthorities[0].WorkItemRef
	sqliteTestNoError(t, execSQLiteV31IgnoringAuthorityImmutability(
		system.repository,
		`UPDATE work_item_authorities SET egress_policy_payload_sha256=NULL WHERE work_item_ref=?`,
		workItemRef.String(),
	))
	if _, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref()); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "sqlite.work_item_egress_authority_invalid") {
		t.Fatalf("partial-null egress authority recovered: %s", sqliteTestErrorChain(err))
	}
}

func TestV31RecoveryRejectsCorruptEgressPolicyRef(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	authority := sqliteEgressAuthority(t, "egress-policy:sqlite-v31-corrupt-ref", `{"destinations":["example.org"]}`)
	system.orchestrator = newSQLiteV31EgressOrchestrator(t, system, sqliteEgressPolicyResolver{authority: authority})
	created := submitSQLiteV31Egress(t, system, "request:v31-egress-corrupt-ref", authority.PolicyRef.String())
	workItemRef := created.Record.WorkItemAuthorities[0].WorkItemRef
	connection, err := system.repository.db.Conn(context.Background())
	sqliteTestNoError(t, err)
	defer connection.Close()
	_, err = connection.ExecContext(context.Background(), `DROP TRIGGER work_item_authorities_immutable_update`)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=ON`)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(context.Background(), `
UPDATE work_item_authorities SET egress_policy_ref=' egress-policy:corrupt'
WHERE work_item_ref=?`, workItemRef.String())
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=OFF`)
	sqliteTestNoError(t, err)
	if _, err := system.repository.GetGoal(context.Background(), created.Record.Goal.Ref()); err == nil ||
		!strings.Contains(sqliteTestErrorChain(err), "application.egress_policy_ref_invalid") {
		t.Fatalf("corrupt egress policy ref recovered: %s", sqliteTestErrorChain(err))
	}
}

func TestV31MigrationKeepsV30AuthorityAsExplicitNoEgress(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	created := system.submit(t, "request:v31-migrate-no-egress")
	sqliteTestNoError(t, system.repository.Close())
	database := openFastV18MigrationFixture(t, system.path)
	mustV10Exec(t, database, `
DROP TRIGGER work_item_authorities_egress_shape_guard;
ALTER TABLE work_item_authorities DROP COLUMN egress_policy_canonical_payload;
ALTER TABLE work_item_authorities DROP COLUMN egress_policy_payload_sha256;
ALTER TABLE work_item_authorities DROP COLUMN egress_policy_ref;
DELETE FROM schema_migrations WHERE version=31;
PRAGMA user_version=30`)
	sqliteTestNoError(t, database.Close())

	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	record, err := reopened.GetGoal(context.Background(), created.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	if len(record.WorkItemAuthorities) != 1 ||
		record.WorkItemAuthorities[0].EgressPolicy != (application.EgressPolicyAuthority{}) {
		t.Fatalf("V30 no-egress authority changed on migration: %+v", record.WorkItemAuthorities)
	}
	var version, columns int
	sqliteTestNoError(t, reopened.db.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, reopened.db.QueryRow(`
SELECT COUNT(*) FROM pragma_table_info('work_item_authorities')
WHERE name LIKE 'egress_policy_%'`).Scan(&columns))
	if version != recoverySchemaV38EgressAuthority || columns != 3 {
		t.Fatalf("V31 migration version=%d columns=%d", version, columns)
	}
}

func newSQLiteV31EgressOrchestrator(
	t *testing.T,
	system *sqliteV15System,
	resolver application.EgressPolicyResolver,
) *application.Orchestrator {
	t.Helper()
	_, sources := prepararCapacidadSQLiteV15(t, system.repository, system.clock, 1_000)
	orchestrator, err := application.New(application.Dependencies{
		State: system.repository, Access: system.repository, Launcher: system.external, Observer: system.external,
		Controller: system.external, Artifacts: system.external, Clock: system.clock, IDs: system.ids,
		MaxOutputBytes: 1024, MaxMailboxEnvelopeBytes: 64 << 10,
		MaxExecutionAttempts: 3, MaxChildrenPerParent: 6, ClaimLease: time.Minute,
		DirectorLeaseDuration: 30 * time.Second, EffectApprovalTTL: system.policy.EffectApprovalTTL,
		BudgetPolicy: system.policy, ObservationDelay: time.Second, ExecutionTimeout: time.Hour,
		AgentCapabilities: sqliteTestCapabilities(), CapacitySources: sources, CapacityObservationWait: time.Second,
		EgressPolicies: resolver,
	})
	sqliteTestNoError(t, err)
	return orchestrator
}

func submitSQLiteV31Egress(
	t *testing.T,
	system *sqliteV15System,
	requestRef, policyRef string,
) application.SubmitResult {
	t.Helper()
	result, err := system.orchestrator.Submit(context.Background(), system.access, application.SubmitRequest{
		RequestRef: requestRef, Statement: "produce governed egress evidence", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:" + requestRef, Key: goal.DefaultPhaseKey().String(), TemplateRef: "phase-template:egress",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "work", Objective: "produce governed egress evidence", Phase: goal.DefaultPhaseKey().String(),
				Role: goal.DefaultRoleKey().String(), OutputContract: goal.OutputContractEvidenceBundle,
				EgressPolicyRef: policyRef,
			}},
		},
	})
	sqliteTestNoError(t, err)
	return result
}

func assertSQLiteV31EgressAuthority(
	t *testing.T,
	repository *Repository,
	goalRef goal.GoalRef,
	workItemRef goal.WorkItemRef,
	want application.EgressPolicyAuthority,
) {
	t.Helper()
	record, err := repository.GetGoal(context.Background(), goalRef)
	sqliteTestNoError(t, err)
	for _, authority := range record.WorkItemAuthorities {
		if authority.WorkItemRef == workItemRef {
			if authority.EgressPolicy != want {
				t.Fatalf("egress authority=%+v want=%+v", authority.EgressPolicy, want)
			}
			return
		}
	}
	t.Fatalf("work item authority %s missing", workItemRef.String())
}

func sqliteEgressAuthority(t *testing.T, rawRef, payload string) application.EgressPolicyAuthority {
	t.Helper()
	ref, err := application.NewEgressPolicyRef(rawRef)
	sqliteTestNoError(t, err)
	digest := sha256.Sum256([]byte(payload))
	return application.EgressPolicyAuthority{
		PolicyRef: ref, PayloadSHA256: hex.EncodeToString(digest[:]), CanonicalPayload: payload,
	}
}

func execSQLiteV31IgnoringAuthorityImmutability(
	repository *Repository,
	statement string,
	arguments ...any,
) error {
	if _, err := repository.db.Exec(`DROP TRIGGER work_item_authorities_immutable_update`); err != nil {
		return err
	}
	_, err := repository.db.Exec(statement, arguments...)
	return err
}
