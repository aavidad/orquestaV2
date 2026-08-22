package sqlite

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestV40HistoricalRuntimeAuthorityRequiresV32AndV39AndAcceptedReceipt(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown key", func(t *testing.T) {
		system := newSQLiteV15System(t, 1)
		missing, err := goal.NewExecutionRef("execution:v40-missing")
		if err != nil {
			t.Fatal(err)
		}
		got, err := system.repository.ResolveAgentHistoricalRuntimeAuthority(ctx, ports.AgentHistoricalRuntimeAuthorityKey{
			ExecutionRef: missing, ActionFence: 1,
		})
		if got != (ports.AgentHistoricalRuntimeAuthority{}) || !application.IsStateError(err, application.StateNotFound) {
			t.Fatalf("unknown authority=%+v err=%v", got, err)
		}
	})

	t.Run("v32 without v39 is legacy not found", func(t *testing.T) {
		system, claim, attempt, _, _ := seedV40PreparedLaunch(t, "legacy")
		mustV10Exec(t, system.repository.db, `DROP TRIGGER microvm_host_launch_runtime_digests_immutable_delete`)
		_, err := system.repository.db.Exec(`DELETE FROM microvm_host_launch_runtime_digests WHERE execution_ref=? AND action_fence=?`, attempt.Subject.ExecutionRef.String(), attempt.ActionFence)
		sqliteTestNoError(t, err)
		receiptV40Launch(t, system, claim, attempt)
		got, err := system.repository.ResolveAgentHistoricalRuntimeAuthority(ctx, v40HistoricalKey(attempt))
		if got != (ports.AgentHistoricalRuntimeAuthority{}) || !application.IsStateError(err, application.StateNotFound) {
			t.Fatalf("legacy authority=%+v err=%v", got, err)
		}
	})

	t.Run("v32 and v39 without accepted receipt", func(t *testing.T) {
		system, _, attempt, _, _ := seedV40PreparedLaunch(t, "unreceipted")
		got, err := system.repository.ResolveAgentHistoricalRuntimeAuthority(ctx, v40HistoricalKey(attempt))
		if got != (ports.AgentHistoricalRuntimeAuthority{}) || !application.IsStateError(err, application.StateNotFound) {
			t.Fatalf("unreceipted authority=%+v err=%v", got, err)
		}
	})
}

func TestV40HistoricalRuntimeAuthorityExactProjectionRestartAndConcurrentCAS(t *testing.T) {
	system, claim, attempt, authority, runtime := seedV40PreparedLaunch(t, "exact")
	receipt := receiptV40Launch(t, system, claim, attempt)
	want := v40ExpectedHistoricalAuthority(attempt, receipt, authority, runtime)

	assertV40HistoricalAuthority(t, system.repository, want)
	if err := system.repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openSQLiteV15Repository(t, system.path, system.clock.Now)
	assertV40HistoricalAuthority(t, reopened, want)
	crossedKey := want.Key
	crossedKey.ActionFence++
	if got, err := reopened.ResolveAgentHistoricalRuntimeAuthority(context.Background(), crossedKey); got != (ports.AgentHistoricalRuntimeAuthority{}) || !application.IsStateError(err, application.StateNotFound) {
		t.Fatalf("crossed CAS key authority=%+v err=%v", got, err)
	}

	const readers = 12
	start := make(chan struct{})
	values := make([]ports.AgentHistoricalRuntimeAuthority, readers)
	errs := make([]error, readers)
	var wait sync.WaitGroup
	for index := range values {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			<-start
			values[index], errs[index] = reopened.ResolveAgentHistoricalRuntimeAuthority(context.Background(), want.Key)
		}(index)
	}
	close(start)
	wait.Wait()
	for index := range values {
		if errs[index] != nil || !reflect.DeepEqual(values[index], want) {
			t.Fatalf("concurrent lookup %d=%+v err=%v", index, values[index], errs[index])
		}
	}
}

func TestV40HistoricalRuntimeAuthorityFailsClosedOnCrossedRows(t *testing.T) {
	mutations := map[string]string{
		"receipt not accepted":   `UPDATE effect_receipts SET status='already_failed' WHERE attempt_ref=?`,
		"receipt crossed fence":  `UPDATE effect_receipts SET action_fence=action_fence+1 WHERE attempt_ref=?`,
		"runtime crossed digest": `UPDATE microvm_host_launch_runtime_digests SET kernel_sha256='FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF' WHERE execution_ref=(SELECT execution_ref FROM effect_attempts WHERE ref=?)`,
		"runtime crossed plan":   `UPDATE microvm_host_launch_runtime_digests SET plan_sha256='bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb' WHERE execution_ref=(SELECT execution_ref FROM effect_attempts WHERE ref=?)`,
		"runtime crossed grant":  `UPDATE microvm_host_launch_runtime_digests SET concession_sha256='cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc' WHERE execution_ref=(SELECT execution_ref FROM effect_attempts WHERE ref=?)`,
	}
	for name, statement := range mutations {
		t.Run(name, func(t *testing.T) {
			system, claim, attempt, _, _ := seedV40PreparedLaunch(t, "crossed-"+strings.ReplaceAll(name, " ", "-"))
			receiptV40Launch(t, system, claim, attempt)
			dropV40ImmutabilityForFault(t, system, name)
			connection, err := system.repository.db.Conn(context.Background())
			sqliteTestNoError(t, err)
			defer connection.Close()
			_, err = connection.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`)
			sqliteTestNoError(t, err)
			_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=ON`)
			sqliteTestNoError(t, err)
			_, err = connection.ExecContext(context.Background(), statement, attempt.Ref)
			sqliteTestNoError(t, err)
			_, err = connection.ExecContext(context.Background(), `PRAGMA ignore_check_constraints=OFF`)
			sqliteTestNoError(t, err)
			got, lookupErr := system.repository.ResolveAgentHistoricalRuntimeAuthority(context.Background(), v40HistoricalKey(attempt))
			if got != (ports.AgentHistoricalRuntimeAuthority{}) || lookupErr == nil {
				t.Fatalf("crossed authority=%+v err=%v", got, lookupErr)
			}
		})
	}
}

func seedV40PreparedLaunch(t *testing.T, suffix string) (*sqliteV15System, application.ActionClaim, application.EffectAttempt, ports.MicroVMHostLaunchAuthorityV1, ports.MicroVMHostLaunchRuntimeDigestsV1) {
	t.Helper()
	system := newSQLiteV15System(t, 1)
	system.submit(t, "request:v40-"+suffix)
	claim := claimSQLiteV15(t, system, "claim:v40-"+suffix)
	prepareSQLiteV15Launch(t, system, claim)
	attempt := sqliteV15Attempt(claim, system.clock.Now())
	stored, created, err := system.repository.RecordEffectAttempt(context.Background(), application.RecordEffectAttemptState{Claim: claim, Attempt: attempt, OperationAt: system.clock.Now()})
	if err != nil || !created || stored != attempt {
		t.Fatalf("attempt=%+v created=%t err=%v", stored, created, err)
	}
	authority := sqliteMicroVMHostLaunchAuthority(t, attempt, true)
	bindSQLiteMicroVMHostLaunchSession(t, system, authority)
	runtime := ports.MicroVMHostLaunchRuntimeDigestsV1{
		Key: authority.Key, PlanSHA256: authority.PlanSHA256, ConcessionSHA256: authority.ConcessionSHA256,
		KernelSHA256:    strings.Repeat("1", 64),
		InitramfsSHA256: strings.Repeat("2", 64), ProfileSHA256: strings.Repeat("3", 64),
	}
	if _, err := system.repository.PrepareWithRuntime(context.Background(), authority, runtime); err != nil {
		t.Fatalf("PrepareWithRuntime err=%s", sqliteTestErrorChain(err))
	}
	return system, claim, attempt, authority, runtime
}

func receiptV40Launch(t *testing.T, system *sqliteV15System, claim application.ActionClaim, attempt application.EffectAttempt) application.EffectReceipt {
	t.Helper()
	receipt := sqliteV15EffectReceipt(claim, attempt, application.EffectStatusAccepted, system.clock.Now())
	receipt.ExternalRef = "ejecucion:v40-physical"
	if _, err := system.repository.BindExternal(context.Background(), ports.MicroVMHostLaunchAuthorityKey{
		RunRef: attempt.Subject.ExecutionRef, ActionFence: attempt.ActionFence,
	}, receipt.ExternalRef); err != nil {
		t.Fatalf("BindExternal err=%s", sqliteTestErrorChain(err))
	}
	record, err := system.repository.GetGoal(context.Background(), attempt.Subject.GoalRef)
	sqliteTestNoError(t, err)
	execution := record.Executions[0]
	execution.State = application.ExecutionRunning
	capabilities := sqliteTestCapabilities()
	execution.ProviderRef, execution.ModelRef, execution.AgentRef = capabilities.ProviderRef, capabilities.ModelRef, capabilities.AgentRef
	execution.ExternalRef, execution.LaunchReceiptRef = receipt.ExternalRef, receipt.Ref
	execution.StartedAt, execution.DeadlineAt, execution.ProviderAcceptedAt = system.clock.Now(), claim.LeaseUntil, system.clock.Now()
	item, found := record.Goal.WorkItem(execution.WorkItemRef)
	if !found {
		t.Fatal("launch work item not found")
	}
	err = system.repository.RecordLaunchAccepted(context.Background(), application.LaunchAcceptedState{
		Claim: claim, Execution: execution, EffectReceipt: receipt, OperationAt: system.clock.Now(),
		NextAction: application.ActionRecord{
			Ref: "action:observe:" + execution.Ref.String(), Kind: application.ActionObserveAgent,
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref,
			PlanGeneration: execution.PlanGeneration, WorkItemGeneration: item.Revision(), AvailableAt: system.clock.Now(),
		},
		Event: application.EventRecord{
			Ref: "event:v40-accepted:" + execution.Ref.String(), Kind: "execution.accepted",
			GoalRef: execution.GoalRef, WorkItemRef: execution.WorkItemRef, ExecutionRef: execution.Ref, OccurredAt: system.clock.Now(),
		},
	})
	if err != nil {
		t.Fatalf("RecordLaunchAccepted err=%s", sqliteTestErrorChain(err))
	}
	return receipt
}

func v40HistoricalKey(attempt application.EffectAttempt) ports.AgentHistoricalRuntimeAuthorityKey {
	return ports.AgentHistoricalRuntimeAuthorityKey{ExecutionRef: attempt.Subject.ExecutionRef, ActionFence: attempt.ActionFence}
}

func v40ExpectedHistoricalAuthority(attempt application.EffectAttempt, receipt application.EffectReceipt, authority ports.MicroVMHostLaunchAuthorityV1, runtime ports.MicroVMHostLaunchRuntimeDigestsV1) ports.AgentHistoricalRuntimeAuthority {
	capabilities := sqliteTestCapabilities()
	return ports.AgentHistoricalRuntimeAuthority{
		Key: v40HistoricalKey(attempt),
		Subject: ports.AgentEnvironmentLifecycleSubject{
			ExecutionRef: attempt.Subject.ExecutionRef, GoalRef: attempt.Subject.GoalRef, WorkItemRef: attempt.Subject.WorkItemRef,
			PlanGeneration: attempt.Subject.PlanGeneration, AppSpecGeneration: attempt.Subject.AppSpecGeneration,
			ExecutionAttempt: 1, SpecHash: attempt.Subject.SpecHash,
			ProviderRef: capabilities.ProviderRef, ModelRef: capabilities.ModelRef, AgentRef: capabilities.AgentRef, ExternalRef: receipt.ExternalRef,
		},
		Digests: ports.AgentHistoricalRuntimeDigests{
			PlanSHA256: runtime.PlanSHA256, GrantSHA256: runtime.ConcessionSHA256,
			KernelSHA256: runtime.KernelSHA256, InitramfsSHA256: runtime.InitramfsSHA256, ProfileSHA256: runtime.ProfileSHA256,
		},
	}
}

func assertV40HistoricalAuthority(t *testing.T, repository *Repository, want ports.AgentHistoricalRuntimeAuthority) {
	t.Helper()
	got, err := repository.ResolveAgentHistoricalRuntimeAuthority(context.Background(), want.Key)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("historical authority=%+v want=%+v err=%v", got, want, err)
	}
}

func dropV40ImmutabilityForFault(t *testing.T, system *sqliteV15System, name string) {
	t.Helper()
	switch name {
	case "receipt not accepted", "receipt crossed fence":
		mustV10Exec(t, system.repository.db, `DROP TRIGGER effect_receipts_immutable_update`)
	case "runtime crossed digest", "runtime crossed plan", "runtime crossed grant":
		mustV10Exec(t, system.repository.db, `DROP TRIGGER microvm_host_launch_runtime_digests_immutable_update`)
	}
}
