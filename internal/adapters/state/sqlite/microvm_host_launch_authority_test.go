package sqlite

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestMicroVMHostLaunchAuthorityPrepareResolveReplayAndReopen(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-reopen")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)

	prepared, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority)
	if err != nil || !reflect.DeepEqual(prepared, authority) {
		t.Fatalf("Prepare()=%+v err=%v", prepared, err)
	}
	prepared.Services[0].ServiceRef = "servicio:mutated"
	resolved, err := system.repository.Resolve(context.Background(), authority.Key)
	if err != nil || !reflect.DeepEqual(resolved, authority) {
		t.Fatalf("Resolve()=%+v err=%v", resolved, err)
	}
	resolved.Services[0].IdentityRef = "identidad-servicio:mutated"

	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	replayed, err := prepareSQLiteMicroVMHostLaunch(reopened, context.Background(), authority)
	if err != nil || !reflect.DeepEqual(replayed, authority) {
		t.Fatalf("reopened Prepare()=%+v err=%v", replayed, err)
	}
	again, err := reopened.Resolve(context.Background(), authority.Key)
	if err != nil || !reflect.DeepEqual(again, authority) {
		t.Fatalf("reopened Resolve()=%+v err=%v", again, err)
	}
}

func TestMicroVMHostLaunchAuthorityPreparedReplayAcceptsBoundAndRejectsDivergence(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-bound-replay")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
		t.Fatal(err)
	}
	bound, err := system.repository.BindExternal(context.Background(), authority.Key, "ejecucion:physical-one")
	if err != nil || bound.ExternalRef != "ejecucion:physical-one" {
		t.Fatalf("BindExternal()=%+v err=%v", bound, err)
	}
	replayed, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority)
	if err != nil || replayed.ExternalRef != bound.ExternalRef ||
		!sameMicroVMHostLaunchAuthorityExceptExternal(replayed, authority) {
		t.Fatalf("bound replay=%+v err=%v", replayed, err)
	}

	divergent := ports.CloneMicroVMHostLaunchAuthorityV1(authority)
	divergent.PlanSHA256 = strings.Repeat("c", 64)
	if got, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), divergent); !reflect.DeepEqual(got, ports.MicroVMHostLaunchAuthorityV1{}) ||
		!application.IsStateError(err, application.StateConflict) {
		t.Fatalf("divergent Prepare()=%+v err=%v", got, err)
	}
}

func TestMicroVMHostLaunchAuthorityPrepareRejectsCrossedAttemptSubject(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-crossed-subject")
	base := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	bindSQLiteMicroVMHostLaunchSession(t, system, base)
	tests := map[string]func(*ports.MicroVMHostLaunchAuthorityV1){
		"actor": func(authority *ports.MicroVMHostLaunchAuthorityV1) {
			authority.OneShotClaim.ActorRef = "actor:crossed"
			authority.OneShotClaim.OwnerRef = "actor:crossed"
		},
		"project": func(authority *ports.MicroVMHostLaunchAuthorityV1) {
			authority.OneShotClaim.ScopeRef = "project:crossed"
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := ports.CloneMicroVMHostLaunchAuthorityV1(base)
			mutate(&candidate)
			if err := ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(candidate); err != nil {
				t.Fatalf("crossed contract fixture invalid before persistence: %v", err)
			}
			if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), candidate); !application.IsStateError(err, application.StateConflict) {
				t.Fatalf("crossed Prepare err=%v", err)
			}
		})
	}
}

func TestMicroVMHostLaunchAuthorityPrepareRequiresExactDurableExecutionSession(t *testing.T) {
	for _, mode := range []string{"empty", "crossed"} {
		t.Run(mode, func(t *testing.T) {
			system, attempt := seedV27AmbiguousLaunch(t, "host-authority-session-"+mode)
			authority := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
			if mode == "crossed" {
				crossed, err := ports.NewExecutionSessionRef(
					"execution-session:sha256:" + strings.Repeat("f", 64),
				)
				if err != nil {
					t.Fatal(err)
				}
				result, err := system.repository.db.Exec(`
UPDATE executions SET execution_session_ref=? WHERE ref=? AND execution_session_ref=''`,
					crossed.String(), authority.Key.RunRef.String(),
				)
				if err != nil {
					t.Fatal(err)
				}
				if rows, err := result.RowsAffected(); err != nil || rows != 1 {
					t.Fatalf("crossed session rows=%d err=%v", rows, err)
				}
			}
			if got, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); !reflect.DeepEqual(got, ports.MicroVMHostLaunchAuthorityV1{}) ||
				!application.IsStateError(err, application.StateConflict) {
				t.Fatalf("%s Prepare()=%+v err=%v", mode, got, err)
			}
			var rows int
			if err := system.repository.db.QueryRow(
				`SELECT COUNT(*) FROM microvm_host_launch_authorities`,
			).Scan(&rows); err != nil || rows != 0 {
				t.Fatalf("%s inserted rows=%d err=%v", mode, rows, err)
			}
		})
	}

	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-session-exact")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	prepared, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority)
	if err != nil || !reflect.DeepEqual(prepared, authority) {
		t.Fatalf("exact Prepare()=%+v err=%v", prepared, err)
	}
}

func TestMicroVMHostLaunchAuthorityPrepareRejectsAttemptWithReceiptWithoutInsert(t *testing.T) {
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:host-authority-receipted")
	claim := claimSQLiteV15(t, system, "claim:host-authority-receipted")
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	if _, created, err := system.repository.RecordEffectAttempt(
		context.Background(), application.RecordEffectAttemptState{
			Claim: claim, Attempt: attempt, OperationAt: system.clock.Now(),
		},
	); err != nil || !created {
		t.Fatalf("RecordEffectAttempt created=%t err=%v", created, err)
	}
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	receipt := sqliteV15EffectReceipt(
		claim, attempt, application.EffectStatusAccepted, system.clock.Now(),
	)
	transaction, err := beginTransaction(context.Background(), system.repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := insertEffectReceipt(context.Background(), transaction, claim, receipt, receipt.ConfirmedAt); err != nil {
		_ = transaction.Rollback()
		t.Fatal(err)
	}
	if err := commit(transaction); err != nil {
		t.Fatal(err)
	}

	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("receipted Prepare err=%v", err)
	}
	var authorities int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM microvm_host_launch_authorities`).Scan(&authorities); err != nil || authorities != 0 {
		t.Fatalf("receipted attempt inserted authorities=%d err=%v", authorities, err)
	}
}

func TestMicroVMHostLaunchAuthorityPrepareRejectsSQLiteIntegerOverflowWithoutMutation(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-integer-overflow")
	base := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	highFence := ports.CloneMicroVMHostLaunchAuthorityV1(base)
	highFence.Key.ActionFence = uint64(math.MaxInt64) + 1
	requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
		highFence.Key, highFence.EffectAttemptRef, highFence.SessionRef,
	)
	if err != nil {
		t.Fatal(err)
	}
	highFence.OneShotClaim.RequestRef = requestRef
	highVersion := ports.CloneMicroVMHostLaunchAuthorityV1(base)
	highVersion.OneShotClaim.Version = credentials.Version(uint64(math.MaxInt64) + 1)
	for name, candidate := range map[string]ports.MicroVMHostLaunchAuthorityV1{
		"action fence":  highFence,
		"claim version": highVersion,
	} {
		t.Run(name, func(t *testing.T) {
			if err := ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(candidate); err != nil {
				t.Fatalf("overflow fixture must pass transport contract: %v", err)
			}
			if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), candidate); !application.IsStateError(err, application.StateInvalid) {
				t.Fatalf("overflow Prepare err=%v", err)
			}
		})
	}
	var authorities int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM microvm_host_launch_authorities`).Scan(&authorities); err != nil || authorities != 0 {
		t.Fatalf("overflow calls inserted authorities=%d err=%v", authorities, err)
	}
}

func TestMicroVMHostLaunchAuthorityBindIsSetOnceAndUnknownFails(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-bind")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
		t.Fatal(err)
	}
	first, err := system.repository.BindExternal(context.Background(), authority.Key, "ejecucion:physical-one")
	if err != nil {
		t.Fatal(err)
	}
	same, err := system.repository.BindExternal(context.Background(), authority.Key, "ejecucion:physical-one")
	if err != nil || !reflect.DeepEqual(same, first) {
		t.Fatalf("same bind=%+v err=%v", same, err)
	}
	if got, err := system.repository.BindExternal(context.Background(), authority.Key, "ejecucion:physical-two"); !reflect.DeepEqual(got, ports.MicroVMHostLaunchAuthorityV1{}) || !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("different bind=%+v err=%v", got, err)
	}
	unknownRun, _ := goal.NewExecutionRef("execution:unknown-host-authority")
	if got, err := system.repository.BindExternal(context.Background(), ports.MicroVMHostLaunchAuthorityKey{
		RunRef: unknownRun, ActionFence: authority.Key.ActionFence,
	}, "ejecucion:physical-unknown"); !reflect.DeepEqual(got, ports.MicroVMHostLaunchAuthorityV1{}) ||
		!application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("unknown bind=%+v err=%v", got, err)
	}
}

func TestMicroVMHostLaunchAuthorityConcurrentDifferentBindsHaveOneWinner(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-concurrent")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
		t.Fatal(err)
	}

	externalRefs := []string{"ejecucion:physical-a", "ejecucion:physical-b"}
	errorsByRef := make([]error, len(externalRefs))
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index := range externalRefs {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			_, errorsByRef[index] = system.repository.BindExternal(
				context.Background(), authority.Key, externalRefs[index],
			)
		}(index)
	}
	close(start)
	wait.Wait()
	winners, conflicts := 0, 0
	for _, err := range errorsByRef {
		switch {
		case err == nil:
			winners++
		case application.IsStateError(err, application.StateConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	resolved, err := system.repository.Resolve(context.Background(), authority.Key)
	if err != nil || winners != 1 || conflicts != 1 ||
		(resolved.ExternalRef != externalRefs[0] && resolved.ExternalRef != externalRefs[1]) {
		t.Fatalf("winners=%d conflicts=%d resolved=%+v err=%v", winners, conflicts, resolved, err)
	}
}

func TestMicroVMHostLaunchAuthorityPreservesCanceledContextWithoutMutation(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-cancel")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, ctx, authority); !errors.Is(err, context.Canceled) {
		t.Fatalf("Prepare canceled err=%v", err)
	}
	if _, err := system.repository.Resolve(ctx, authority.Key); !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve canceled err=%v", err)
	}
	if _, err := system.repository.BindExternal(ctx, authority.Key, "ejecucion:physical-cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("BindExternal canceled err=%v", err)
	}
	var rows int
	if err := system.repository.db.QueryRow(`SELECT COUNT(*) FROM microvm_host_launch_authorities`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("canceled calls mutated rows=%d err=%v", rows, err)
	}
}

func TestMicroVMHostLaunchAuthorityRejectsSemanticCorruptionOnRead(t *testing.T) {
	system, attempt := seedV27AmbiguousLaunch(t, "host-authority-corrupt")
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	if _, err := prepareSQLiteMicroVMHostLaunch(system.repository, context.Background(), authority); err != nil {
		t.Fatal(err)
	}
	connection, err := system.repository.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(context.Background(),
		`DROP TRIGGER microvm_host_launch_authorities_bind_once`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `
UPDATE microvm_host_launch_authorities SET control_port=0
WHERE execution_ref=? AND action_fence=?`, authority.Key.RunRef.String(), authority.Key.ActionFence); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := system.repository.Resolve(context.Background(), authority.Key); !application.IsStateError(err, application.StateInvalid) {
		t.Fatalf("corrupt Resolve err=%v", err)
	}
}

func sqliteMicroVMHostLaunchAuthority(
	t *testing.T,
	attempt application.EffectAttempt,
	withProxy bool,
) ports.MicroVMHostLaunchAuthorityV1 {
	t.Helper()
	session := recoveryV32ExecutionSessionRef(t, attempt.Ref)
	key := ports.MicroVMHostLaunchAuthorityKey{
		RunRef: attempt.Subject.ExecutionRef, ActionFence: attempt.ActionFence,
	}
	requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(key, attempt.Ref, session)
	if err != nil {
		t.Fatal(err)
	}
	authority := ports.MicroVMHostLaunchAuthorityV1{
		Key: key, EffectAttemptRef: attempt.Ref, SessionRef: session,
		OneShotClaim: credentials.OneShotUseRequest{
			ActorRef: attempt.Subject.ActorRef.String(), RequestRef: requestRef,
			CredentialRef: "credential:provider_codex_primary",
			OwnerRef:      credentials.OwnerRef(attempt.Subject.ActorRef.String()),
			ScopeRef:      credentials.ScopeRef(attempt.Subject.ProjectRef.String()),
			PurposeRef:    "provider:codex", Version: 3,
		},
		PlanSHA256: strings.Repeat("a", 64), ConcessionSHA256: strings.Repeat("b", 64),
		Services: []ports.MicroVMHostServiceAuthorityV1{{
			Role: ports.MicroVMHostServiceControlBroker, ServiceRef: "servicio:control",
			Port: 10_001, IdentityRef: "identidad-servicio:control",
			IdentitySHA256: strings.Repeat("c", 64),
		}},
	}
	if withProxy {
		authority.Services = append(authority.Services, ports.MicroVMHostServiceAuthorityV1{
			Role: ports.MicroVMHostServiceControlledEgressProxy, ServiceRef: "servicio:proxy",
			Port: 10_002, IdentityRef: "identidad-servicio:proxy",
			IdentitySHA256: strings.Repeat("d", 64),
		})
	}
	if err := ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(authority); err != nil {
		t.Fatalf("fixture authority invalid: %v", err)
	}
	return authority
}

func prepareSQLiteMicroVMHostLaunch(
	repository *Repository,
	ctx context.Context,
	authority ports.MicroVMHostLaunchAuthorityV1,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	return repository.PrepareWithRuntime(ctx, authority, ports.MicroVMHostLaunchRuntimeDigestsV1{
		Key: authority.Key, PlanSHA256: strings.Repeat("4", 64), ConcessionSHA256: strings.Repeat("5", 64), KernelSHA256: strings.Repeat("1", 64),
		InitramfsSHA256: strings.Repeat("2", 64), ProfileSHA256: strings.Repeat("3", 64),
	})
}

func bindSQLiteMicroVMHostLaunchSession(
	t *testing.T,
	system *sqliteV15System,
	authority ports.MicroVMHostLaunchAuthorityV1,
) {
	t.Helper()
	result, err := system.repository.db.Exec(`
UPDATE executions SET execution_session_ref=?
WHERE ref=? AND execution_session_ref=''`,
		authority.SessionRef.String(), authority.Key.RunRef.String(),
	)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		t.Fatalf("bind execution session rows=%d err=%v", rows, err)
	}
}
