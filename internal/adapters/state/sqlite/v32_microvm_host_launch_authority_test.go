package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestV33MicroVMHostLaunchAuthorityMigrationAndStrictShapeSurviveReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "microvm-host-authority-v31.db")
	database := agentCapacityDatabase(t, path, recoverySchemaV38EgressAuthority)
	sqliteTestNoError(t, database.Close())
	sqliteTestNoError(t, os.Chmod(path, 0o600))

	repository, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	assertV32MicroVMHostLaunchAuthorityShape(t, repository.db)
	sqliteTestNoError(t, repository.Close())

	reopened, err := Open(context.Background(), Options{
		Path: path, BusyTimeout: testBusyTimeout, MaxOpenConnections: 4,
	})
	sqliteTestNoError(t, err)
	t.Cleanup(func() { _ = reopened.Close() })
	assertV32MicroVMHostLaunchAuthorityShape(t, reopened.db)
}

func TestV32MicroVMHostLaunchAuthorityRejectsPartialProxy(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	_, attempt := seedV32EffectAttempt(t, system, "proxy")
	base := newV32MicroVMHostLaunchAuthority(t, attempt)
	bindV32MicroVMHostLaunchExecutionSession(t, system, base)
	complete := [4]any{
		"servicio:egress", int64(5002), "identidad-servicio:egress", strings.Repeat("e", 64),
	}
	for index, name := range []string{"service", "port", "identity", "digest"} {
		t.Run("missing "+name, func(t *testing.T) {
			candidate := base
			candidate.proxy = complete
			candidate.proxy[index] = nil
			if err := insertV32MicroVMHostLaunchAuthority(system.repository.db, candidate); err == nil {
				t.Fatalf("partial proxy without %s accepted", name)
			}
		})
	}
	base.proxy = complete
	sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, base))
	assertV32MicroVMHostLaunchAuthorityRoundTrip(t, system.repository.db, base, true, false)
}

func TestV32MicroVMHostLaunchAuthorityRejectsEveryMutationAndDelete(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	_, attempt := seedV32EffectAttempt(t, system, "immutable")
	authority := newV32MicroVMHostLaunchAuthority(t, attempt)
	bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
	authority.proxy = [4]any{
		"servicio:egress", int64(5002), "identidad-servicio:egress", strings.Repeat("e", 64),
	}
	sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, authority))

	mutations := []struct {
		column string
		value  any
	}{
		{"execution_ref", "execution:changed"},
		{"action_fence", int64(attempt.ActionFence + 1)},
		{"effect_attempt_ref", "effect-attempt:changed"},
		{"session_ref", "execution-session:changed"},
		{"plan_sha256", strings.Repeat("c", 64)},
		{"concession_sha256", strings.Repeat("d", 64)},
		{"control_service_ref", "servicio:control_changed"},
		{"control_port", int64(5011)},
		{"control_identity_ref", "identidad-servicio:control_changed"},
		{"control_identity_sha256", strings.Repeat("f", 64)},
		{"proxy_service_ref", "servicio:egress_changed"},
		{"proxy_port", int64(5012)},
		{"proxy_identity_ref", "identidad-servicio:egress_changed"},
		{"proxy_identity_sha256", strings.Repeat("1", 64)},
		{"credential_ref", "credential:provider_other"},
		{"owner_ref", "owner:other"},
		{"scope_ref", "project:other"},
		{"purpose_ref", "provider:other"},
		{"credential_version", int64(4)},
		{"actor_ref", "actor:other"},
		{"request_ref", "request:microvm-host-launch-one-shot:sha256:" + strings.Repeat("2", 64)},
	}
	for _, mutation := range mutations {
		t.Run(mutation.column, func(t *testing.T) {
			_, err := system.repository.db.Exec(
				`UPDATE microvm_host_launch_authorities SET `+mutation.column+`=? WHERE execution_ref=? AND action_fence=?`,
				mutation.value, attempt.Subject.ExecutionRef.String(), attempt.ActionFence,
			)
			if err == nil || !strings.Contains(err.Error(), "sqlite.microvm_host_launch_authority_immutable") {
				t.Fatalf("mutation %s err=%v", mutation.column, err)
			}
		})
	}
	if _, err := system.repository.db.Exec(
		`DELETE FROM microvm_host_launch_authorities WHERE execution_ref=? AND action_fence=?`,
		attempt.Subject.ExecutionRef.String(), attempt.ActionFence,
	); err == nil || !strings.Contains(err.Error(), "sqlite.microvm_host_launch_authority_immutable") {
		t.Fatalf("delete err=%v", err)
	}
}

func TestV32MicroVMHostLaunchAuthorityBindsExternalRefExactlyOnce(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	_, attempt := seedV32EffectAttempt(t, system, "bind")
	authority := newV32MicroVMHostLaunchAuthority(t, attempt)
	bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
	sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, authority))
	assertV32MicroVMHostLaunchAuthorityRoundTrip(t, system.repository.db, authority, false, false)

	for _, invalidRef := range []string{"ejecucion:bad/path", "ejecucion:"} {
		if _, err := system.repository.db.Exec(`
UPDATE microvm_host_launch_authorities SET external_ref=?
WHERE execution_ref=? AND action_fence=?`, invalidRef, authority.executionRef, authority.actionFence); err == nil {
			t.Fatalf("invalid external ref %q accepted", invalidRef)
		}
	}
	const externalRef = "ejecucion:physical_1"
	_, err := system.repository.db.Exec(`
UPDATE microvm_host_launch_authorities SET external_ref=?
WHERE execution_ref=? AND action_fence=?`, externalRef, authority.executionRef, authority.actionFence)
	sqliteTestNoError(t, err)
	for _, candidate := range []string{externalRef, "ejecucion:physical_2"} {
		_, err = system.repository.db.Exec(`
UPDATE microvm_host_launch_authorities SET external_ref=?
WHERE execution_ref=? AND action_fence=?`, candidate, authority.executionRef, authority.actionFence)
		if err == nil || !strings.Contains(err.Error(), "sqlite.microvm_host_launch_authority_immutable") {
			t.Fatalf("second bind %q err=%v", candidate, err)
		}
	}
	var stored string
	sqliteTestNoError(t, system.repository.db.QueryRow(`
SELECT external_ref FROM microvm_host_launch_authorities
WHERE execution_ref=? AND action_fence=?`, authority.executionRef, authority.actionFence).Scan(&stored))
	if stored != externalRef {
		t.Fatalf("external ref=%q want=%q", stored, externalRef)
	}
	authority.externalRef = externalRef
	assertV32MicroVMHostLaunchAuthorityRoundTrip(t, system.repository.db, authority, false, true)
}

func TestV32MicroVMHostLaunchAuthorityRejectsRawMalformedTextAndPort(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	_, attempt := seedV32EffectAttempt(t, system, "raw-shape")
	base := newV32MicroVMHostLaunchAuthority(t, attempt)
	bindV32MicroVMHostLaunchExecutionSession(t, system, base)
	tests := []struct {
		name   string
		mutate func(*v32MicroVMHostLaunchAuthority)
	}{
		{"nul digest", func(value *v32MicroVMHostLaunchAuthority) {
			value.planSHA256 = strings.Repeat("a", 63) + "\x00"
		}},
		{"request suffix", func(value *v32MicroVMHostLaunchAuthority) { value.requestRef += "a" }},
		{"leading tab", func(value *v32MicroVMHostLaunchAuthority) { value.ownerRef = "\t" + value.ownerRef }},
		{"trailing tab", func(value *v32MicroVMHostLaunchAuthority) { value.sessionRef += "\t" }},
		{"service suffix", func(value *v32MicroVMHostLaunchAuthority) { value.controlServiceRef += "!" }},
		{"credential alphabet", func(value *v32MicroVMHostLaunchAuthority) { value.credentialRef = "credential:_bad" }},
		{"port above uint32", func(value *v32MicroVMHostLaunchAuthority) { value.controlPort = 1 << 32 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := base
			test.mutate(&candidate)
			if err := insertV32MicroVMHostLaunchAuthority(system.repository.db, candidate); err == nil {
				t.Fatal("malformed raw authority accepted")
			}
		})
	}
}

func TestV32MicroVMHostLaunchAuthorityRejectsCausalMismatchAndPostReceipt(t *testing.T) {
	t.Run("already bound", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		_, attempt := seedV32EffectAttempt(t, system, "already-bound")
		authority := newV32MicroVMHostLaunchAuthority(t, attempt)
		bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
		authority.externalRef = "ejecucion:premature"
		assertV32CausalInsertRejected(t, system.repository.db, authority)
	})

	t.Run("actor and scope", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		_, attempt := seedV32EffectAttempt(t, system, "causal-subject")
		base := newV32MicroVMHostLaunchAuthority(t, attempt)
		bindV32MicroVMHostLaunchExecutionSession(t, system, base)
		for name, mutate := range map[string]func(*v32MicroVMHostLaunchAuthority){
			"actor": func(value *v32MicroVMHostLaunchAuthority) { value.actorRef = "actor:other" },
			"scope": func(value *v32MicroVMHostLaunchAuthority) { value.scopeRef = "project:other" },
		} {
			t.Run(name, func(t *testing.T) {
				candidate := base
				mutate(&candidate)
				assertV32CausalInsertRejected(t, system.repository.db, candidate)
			})
		}
	})

	t.Run("receipt already exists", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		claim, attempt := seedV32EffectAttempt(t, system, "post-receipt")
		authority := newV32MicroVMHostLaunchAuthority(t, attempt)
		bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
		acceptAgentCapacityLaunch(t, system, claim, attempt)
		assertV32CausalInsertRejected(t, system.repository.db, authority)
	})

	t.Run("non launch intent", func(t *testing.T) {
		system, _ := seedSQLiteV17Committed(t, &sqliteTestAttestor{})
		claim := claimV17WithLeases(t, system.repository, "claim:v32-non-launch", time.Minute, time.Minute, system.policy)
		if claim.Action.EffectIntent.Kind == application.EffectKindAgentLaunch {
			t.Fatal("non-launch fixture selected agent launch")
		}
		attempt := sqliteV15Attempt(claim, system.clock.Now())
		persisted, created, err := system.repository.RecordEffectAttempt(
			context.Background(), application.RecordEffectAttemptState{
				Claim: claim, Attempt: attempt, OperationAt: attempt.StartedAt,
			},
		)
		if err != nil || !created || persisted != attempt {
			t.Fatalf("non-launch attempt=%+v created=%t err=%v", persisted, created, err)
		}
		authority := newV32MicroVMHostLaunchAuthority(t, attempt)
		bindV32MicroVMHostLaunchExecutionSession(t, system, authority)
		assertV32CausalInsertRejected(t, system.repository.db, authority)
	})
}

func TestV32MicroVMHostLaunchAuthorityForeignKeyBindsAttemptExecutionAndFence(t *testing.T) {
	system := newSQLiteV15System(t, 2)
	_, first := seedV32EffectAttempt(t, system, "causal-first")
	_, second := seedV32EffectAttempt(t, system, "causal-second")
	base := newV32MicroVMHostLaunchAuthority(t, first)
	bindV32MicroVMHostLaunchExecutionSession(t, system, base)
	sqliteTestNoError(t, execV32DropCausalInsertGuard(system.repository.db))

	tests := []struct {
		name   string
		mutate func(*v32MicroVMHostLaunchAuthority)
	}{
		{"cross execution", func(value *v32MicroVMHostLaunchAuthority) {
			value.executionRef = second.Subject.ExecutionRef.String()
		}},
		{"cross fence", func(value *v32MicroVMHostLaunchAuthority) {
			value.actionFence = first.ActionFence + 1
		}},
		{"cross attempt", func(value *v32MicroVMHostLaunchAuthority) {
			value.effectAttemptRef = second.Ref
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := base
			test.mutate(&candidate)
			if err := insertV32MicroVMHostLaunchAuthority(system.repository.db, candidate); err == nil ||
				!strings.Contains(strings.ToLower(err.Error()), "foreign key") {
				t.Fatalf("crossed authority err=%v", err)
			}
		})
	}
	sqliteTestNoError(t, insertV32MicroVMHostLaunchAuthority(system.repository.db, base))
	var violations int
	sqliteTestNoError(t, system.repository.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations))
	if violations != 0 {
		t.Fatalf("foreign key violations=%d", violations)
	}
}

type v32MicroVMHostLaunchAuthority struct {
	executionRef, effectAttemptRef, sessionRef string
	actionFence                                uint64
	planSHA256, concessionSHA256               string
	controlServiceRef                          string
	controlPort                                uint64
	controlIdentityRef, controlIdentitySHA256  string
	proxy                                      [4]any
	externalRef                                string
	credentialRef, ownerRef, scopeRef          string
	purposeRef                                 string
	credentialVersion                          uint64
	actorRef, requestRef                       string
}

func newV32MicroVMHostLaunchAuthority(t *testing.T, attempt application.EffectAttempt) v32MicroVMHostLaunchAuthority {
	t.Helper()
	digest := sha256.Sum256([]byte(attempt.Ref))
	hexDigest := hex.EncodeToString(digest[:])
	sessionRef, err := ports.NewExecutionSessionRef("execution-session:sha256:" + hexDigest)
	sqliteTestNoError(t, err)
	requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
		ports.MicroVMHostLaunchAuthorityKey{
			RunRef: attempt.Subject.ExecutionRef, ActionFence: attempt.ActionFence,
		},
		attempt.Ref,
		sessionRef,
	)
	sqliteTestNoError(t, err)
	actorRef := attempt.Subject.ActorRef.String()
	return v32MicroVMHostLaunchAuthority{
		executionRef: attempt.Subject.ExecutionRef.String(), effectAttemptRef: attempt.Ref,
		actionFence: attempt.ActionFence, sessionRef: sessionRef.String(),
		planSHA256: strings.Repeat("a", 64), concessionSHA256: strings.Repeat("b", 64),
		controlServiceRef: "servicio:control", controlPort: 5001,
		controlIdentityRef: "identidad-servicio:control", controlIdentitySHA256: strings.Repeat("c", 64),
		credentialRef: "credential:provider_codex_primary", ownerRef: actorRef,
		scopeRef: attempt.Subject.ProjectRef.String(), purposeRef: "provider:codex", credentialVersion: 3,
		actorRef: actorRef, requestRef: requestRef,
	}
}

func insertV32MicroVMHostLaunchAuthority(database *sql.DB, authority v32MicroVMHostLaunchAuthority) error {
	_, err := database.Exec(`
INSERT INTO microvm_host_launch_authorities(
 execution_ref,action_fence,effect_attempt_ref,session_ref,plan_sha256,concession_sha256,
 control_service_ref,control_port,control_identity_ref,control_identity_sha256,
 proxy_service_ref,proxy_port,proxy_identity_ref,proxy_identity_sha256,external_ref,
 credential_ref,owner_ref,scope_ref,purpose_ref,credential_version,actor_ref,request_ref
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		authority.executionRef, authority.actionFence, authority.effectAttemptRef, authority.sessionRef,
		authority.planSHA256, authority.concessionSHA256,
		authority.controlServiceRef, authority.controlPort, authority.controlIdentityRef, authority.controlIdentitySHA256,
		authority.proxy[0], authority.proxy[1], authority.proxy[2], authority.proxy[3], nullableV32String(authority.externalRef),
		authority.credentialRef, authority.ownerRef, authority.scopeRef, authority.purposeRef, authority.credentialVersion,
		authority.actorRef, authority.requestRef,
	)
	return err
}

func assertV32MicroVMHostLaunchAuthorityRoundTrip(
	t *testing.T,
	database *sql.DB,
	want v32MicroVMHostLaunchAuthority,
	wantProxy, wantBound bool,
) {
	t.Helper()
	var executionRef, effectAttemptRef, sessionRef, planSHA256, concessionSHA256 string
	var controlServiceRef, controlIdentityRef, controlIdentitySHA256 string
	var proxyServiceRef, proxyIdentityRef, proxyIdentitySHA256, externalRef sql.NullString
	var actionFence, controlPort, credentialVersion uint64
	var proxyPort sql.NullInt64
	var credentialRef, ownerRef, scopeRef, purposeRef, actorRef, requestRef string
	sqliteTestNoError(t, database.QueryRow(`
SELECT execution_ref,action_fence,effect_attempt_ref,session_ref,plan_sha256,concession_sha256,
 control_service_ref,control_port,control_identity_ref,control_identity_sha256,
 proxy_service_ref,proxy_port,proxy_identity_ref,proxy_identity_sha256,external_ref,
 credential_ref,owner_ref,scope_ref,purpose_ref,credential_version,actor_ref,request_ref
FROM microvm_host_launch_authorities WHERE execution_ref=? AND action_fence=?`,
		want.executionRef, want.actionFence,
	).Scan(
		&executionRef, &actionFence, &effectAttemptRef, &sessionRef, &planSHA256, &concessionSHA256,
		&controlServiceRef, &controlPort, &controlIdentityRef, &controlIdentitySHA256,
		&proxyServiceRef, &proxyPort, &proxyIdentityRef, &proxyIdentitySHA256, &externalRef,
		&credentialRef, &ownerRef, &scopeRef, &purposeRef, &credentialVersion, &actorRef, &requestRef,
	))
	runRef, err := goal.NewExecutionRef(executionRef)
	sqliteTestNoError(t, err)
	typedSessionRef, err := ports.NewExecutionSessionRef(sessionRef)
	sqliteTestNoError(t, err)
	authority := ports.MicroVMHostLaunchAuthorityV1{
		Key:              ports.MicroVMHostLaunchAuthorityKey{RunRef: runRef, ActionFence: actionFence},
		EffectAttemptRef: effectAttemptRef, SessionRef: typedSessionRef,
		OneShotClaim: credentials.OneShotUseRequest{
			ActorRef: actorRef, RequestRef: requestRef, CredentialRef: credentials.CredentialRef(credentialRef),
			OwnerRef: credentials.OwnerRef(ownerRef), ScopeRef: credentials.ScopeRef(scopeRef),
			PurposeRef: credentials.PurposeRef(purposeRef), Version: credentials.Version(credentialVersion),
		},
		PlanSHA256: planSHA256, ConcessionSHA256: concessionSHA256,
		Services: []ports.MicroVMHostServiceAuthorityV1{{
			Role: ports.MicroVMHostServiceControlBroker, ServiceRef: controlServiceRef, Port: uint32(controlPort),
			IdentityRef: controlIdentityRef, IdentitySHA256: controlIdentitySHA256,
		}},
	}
	if wantProxy {
		if !proxyServiceRef.Valid || !proxyPort.Valid || !proxyIdentityRef.Valid || !proxyIdentitySHA256.Valid {
			t.Fatal("complete proxy did not round-trip")
		}
		authority.Services = append(authority.Services, ports.MicroVMHostServiceAuthorityV1{
			Role: ports.MicroVMHostServiceControlledEgressProxy, ServiceRef: proxyServiceRef.String,
			Port: uint32(proxyPort.Int64), IdentityRef: proxyIdentityRef.String, IdentitySHA256: proxyIdentitySHA256.String,
		})
	}
	if externalRef.Valid {
		authority.ExternalRef = externalRef.String
	}
	if wantBound {
		err = ports.ValidateMicroVMHostLaunchAuthorityBoundV1(authority)
	} else {
		err = ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(authority)
	}
	if err != nil || authority.OneShotClaim.RequestRef != want.requestRef || authority.OneShotClaim.ActorRef != want.actorRef ||
		string(authority.OneShotClaim.OwnerRef) != want.ownerRef || string(authority.OneShotClaim.ScopeRef) != want.scopeRef {
		t.Fatalf("round-trip authority=%+v err=%v", authority, err)
	}
}

func assertV32CausalInsertRejected(t *testing.T, database *sql.DB, authority v32MicroVMHostLaunchAuthority) {
	t.Helper()
	err := insertV32MicroVMHostLaunchAuthority(database, authority)
	if err == nil || !strings.Contains(err.Error(), "sqlite.microvm_host_launch_authority_causal_invalid") {
		t.Fatalf("causal insert err=%v", err)
	}
}

func execV32DropCausalInsertGuard(database *sql.DB) error {
	_, err := database.Exec(`DROP TRIGGER microvm_host_launch_authorities_causal_insert`)
	return err
}

func nullableV32String(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func seedV32EffectAttempt(
	t *testing.T,
	system *sqliteV15System,
	suffix string,
) (application.ActionClaim, application.EffectAttempt) {
	t.Helper()
	system.submit(t, "request:v32-"+suffix)
	claim := claimSQLiteV15(t, system, "claim:v32-"+suffix)
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	persisted, created, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: attempt.StartedAt,
		},
	)
	if err != nil || !created || persisted != attempt {
		t.Fatalf("seed attempt=%+v created=%t err=%v", persisted, created, err)
	}
	session := recoveryV32ExecutionSessionRef(t, attempt.Ref)
	mustV10Exec(t, system.repository.db, `
UPDATE executions SET execution_session_ref=? WHERE ref=? AND execution_session_ref=''`,
		session.String(), attempt.Subject.ExecutionRef.String())
	return claim, attempt
}

func bindV32MicroVMHostLaunchExecutionSession(
	t *testing.T,
	system *sqliteV15System,
	authority v32MicroVMHostLaunchAuthority,
) {
	t.Helper()
	result, err := system.repository.db.Exec(`
UPDATE executions SET execution_session_ref=?
WHERE ref=? AND (execution_session_ref='' OR execution_session_ref=?)`,
		authority.sessionRef, authority.executionRef, authority.sessionRef)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		t.Fatalf("bind V32 execution session rows=%d err=%v", rows, err)
	}
}

func assertV32MicroVMHostLaunchAuthorityShape(t *testing.T, database *sql.DB) {
	t.Helper()
	var version, receiptV32, receiptV33, strict, rows, foreignKeyViolations int
	var nameV32, nameV33, tableSQL, causalTriggerSQL string
	sqliteTestNoError(t, database.QueryRow(`PRAGMA user_version`).Scan(&version))
	sqliteTestNoError(t, database.QueryRow(`
SELECT name FROM schema_migrations WHERE version=?`, recoverySchemaV38MicroVMHostLaunch).Scan(&nameV32))
	sqliteTestNoError(t, database.QueryRow(`
SELECT name FROM schema_migrations WHERE version=?`, recoverySchemaV38MicroVMHostSession).Scan(&nameV33))
	sqliteTestNoError(t, database.QueryRow(`
SELECT strict FROM pragma_table_list WHERE name='microvm_host_launch_authorities'`).Scan(&strict))
	sqliteTestNoError(t, database.QueryRow(`
SELECT sql FROM sqlite_schema WHERE type='table' AND name='microvm_host_launch_authorities'`).Scan(&tableSQL))
	sqliteTestNoError(t, database.QueryRow(`
SELECT sql FROM sqlite_schema
WHERE type='trigger' AND name='microvm_host_launch_authorities_causal_insert'`).Scan(&causalTriggerSQL))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM microvm_host_launch_authorities`).Scan(&rows))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV38MicroVMHostLaunch).Scan(&receiptV32))
	sqliteTestNoError(t, database.QueryRow(`
SELECT COUNT(*) FROM schema_migrations WHERE version=?`, recoverySchemaV38MicroVMHostSession).Scan(&receiptV33))
	sqliteTestNoError(t, database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&foreignKeyViolations))
	if version != recoverySchemaLatest || nameV32 != "032_microvm_host_launch_authority.sql" ||
		nameV33 != "033_microvm_host_launch_session.sql" || receiptV32 != 1 || receiptV33 != 1 ||
		strict != 1 || rows != 0 || foreignKeyViolations != 0 {
		t.Fatalf("V33 shape version=%d names=%q/%q receipts=%d/%d strict=%d rows=%d fk=%d",
			version, nameV32, nameV33, receiptV32, receiptV33, strict, rows, foreignKeyViolations)
	}
	if !strings.Contains(causalTriggerSQL, "execution.execution_session_ref<>''") ||
		!strings.Contains(causalTriggerSQL, "execution.execution_session_ref=NEW.session_ref") {
		t.Fatalf("V33 causal trigger lacks exact durable session: %s", causalTriggerSQL)
	}
	for _, column := range []string{
		"execution_ref", "effect_attempt_ref", "session_ref", "plan_sha256", "concession_sha256",
		"control_service_ref", "control_identity_ref", "control_identity_sha256", "proxy_service_ref",
		"proxy_identity_ref", "proxy_identity_sha256", "external_ref", "credential_ref", "owner_ref",
		"scope_ref", "purpose_ref", "actor_ref", "request_ref",
	} {
		if !strings.Contains(tableSQL, "instr("+column+",char(0))=0") {
			t.Fatalf("TEXT column %s lacks explicit NUL guard", column)
		}
	}
}
