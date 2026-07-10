package orquestastatefile

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	stateFileMultiprocessActionEnvV0 = "STATE_FILE_MULTIPROCESS_TEST_ACTION"
	stateFileMultiprocessRootEnvV0   = "STATE_FILE_MULTIPROCESS_TEST_ROOT"
	stateFileMultiprocessWriterEnvV0 = "STATE_FILE_MULTIPROCESS_TEST_WRITER"
)

func TestStoreV0GoalRequiredTestAttestationSurvivesRecreateAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	attestation := goalRequiredTestAttestationForStoreTestV0()
	if err := store.SaveGoalRequiredTestAttestationV0(context.Background(), attestation); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.SaveGoalRequiredTestAttestationV0(context.Background(), attestation); err != nil {
		t.Fatalf("Save replay: %v", err)
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStore recovered: %v", err)
	}
	items, err := recovered.ListGoalRequiredTestAttestationsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationQueryV0{RunRef: attestation.RunRef, GoalRef: attestation.GoalRef, RevisionRef: attestation.RevisionRef})
	if err != nil || len(items) != 1 || items[0].AttestationRef != attestation.AttestationRef {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	attestation.ExitCode = 7
	if err := recovered.SaveGoalRequiredTestAttestationV0(context.Background(), attestation); err == nil {
		t.Fatalf("esperaba conflicto de receipt inmutable")
	}
}

func TestStoreV0GoalRequiredTestAttestationConcurrentInstancesKeepImmutableReceipt(t *testing.T) {
	root := t.TempDir()
	first, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	receipt := goalRequiredTestAttestationForStoreTestV0()
	stores := []*StoreV0{first, second}
	errorsByCall := make([]error, 32)
	var wait sync.WaitGroup
	for index := range errorsByCall {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errorsByCall[index] = stores[index%len(stores)].SaveGoalRequiredTestAttestationV0(context.Background(), receipt)
		}(index)
	}
	wait.Wait()
	for _, saveErr := range errorsByCall {
		if saveErr != nil {
			t.Fatalf("save concurrente: %v", saveErr)
		}
	}
	conflict := receipt
	conflict.EvidenceRefs = []string{"evidence-ref-conflicting-payload"}
	if err := second.SaveGoalRequiredTestAttestationV0(context.Background(), conflict); err == nil {
		t.Fatal("payload distinto con mismo attestation_ref debe ser conflicto")
	}
}

func TestStoreV0GoalRequiredTestAttestationMultiprocessKeepsSingleImmutableReceipt(t *testing.T) {
	root := t.TempDir()
	errorsByProcess := make([]error, 8)
	var wait sync.WaitGroup
	for index := range errorsByProcess {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errorsByProcess[index] = runStateFileMultiprocessChildV0(root, "save_attestation", "")
		}(index)
	}
	wait.Wait()
	for _, processErr := range errorsByProcess {
		if processErr != nil {
			t.Fatalf("child save: %v", processErr)
		}
	}
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	receipt := goalRequiredTestAttestationForStoreTestV0()
	items, err := store.ListGoalRequiredTestAttestationsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationQueryV0{
		RunRef: receipt.RunRef, GoalRef: receipt.GoalRef, RevisionRef: receipt.RevisionRef,
	})
	if err != nil || len(items) != 1 || items[0].AttestationRef != receipt.AttestationRef {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	conflict := receipt
	conflict.EvidenceRefs = []string{"evidence-ref-multiprocess-conflict"}
	if err := store.SaveGoalRequiredTestAttestationV0(context.Background(), conflict); err == nil {
		t.Fatal("receipt multiproceso debe permanecer inmutable")
	}
}

func TestStoreV0GoalRequiredTestFinalSnapshotIsImmutableAcrossProcesses(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := goalRequiredTestFinalSnapshotForStoreTestV0()
	if _, err := store.FreezeGoalRequiredTestFinalSnapshotV0(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	replay := snapshot
	replay.ObservedAt = "2026-07-10T10:00:02Z"
	replay.EvidenceRefs = []string{"evidence-ref-state-file-snapshot-replay"}
	if got, err := store.FreezeGoalRequiredTestFinalSnapshotV0(context.Background(), replay); err != nil || got.ObservedAt != snapshot.ObservedAt {
		t.Fatalf("idempotent snapshot=%+v err=%v", got, err)
	}
	conflict := snapshot
	conflict.Hashes = append([]orquestagoal.GoalAttestedHashV0(nil), snapshot.Hashes...)
	conflict.Hashes[0].SHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	conflict = orquestagoal.FreezeGoalRequiredTestFinalSnapshotV0(conflict)
	if _, err := store.FreezeGoalRequiredTestFinalSnapshotV0(context.Background(), conflict); err == nil {
		t.Fatal("snapshot final contradictorio debe bloquear")
	}
}

func TestStoreV0GoalRequiredTestAttestationClaimMultiprocessHasSingleOwner(t *testing.T) {
	root := t.TempDir()
	const processes = 8
	errorsByProcess := make([]error, processes)
	var wait sync.WaitGroup
	for index := range errorsByProcess {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errorsByProcess[index] = runStateFileMultiprocessChildV0(root, "claim_attestation", fmt.Sprintf("%d", index))
		}(index)
	}
	wait.Wait()
	acquired := 0
	for index, processErr := range errorsByProcess {
		if processErr != nil {
			t.Fatalf("child claim %d: %v", index, processErr)
		}
		result, err := os.ReadFile(filepath.Join(root, fmt.Sprintf("multiprocess-result-%d", index)))
		if err != nil {
			t.Fatal(err)
		}
		if string(result) == "acquired" {
			acquired++
		}
	}
	if acquired != 1 {
		t.Fatalf("claim owners=%d", acquired)
	}
}

func TestStoreV0GoalRequiredTestAttestationClaimFailedSurvivesRecreateWithoutReacquire(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	request := goalRequiredTestAttestationClaimRequestForStoreTestV0()
	acquired, err := store.AcquireGoalRequiredTestAttestationClaimV0(context.Background(), request)
	if err != nil || !acquired.Acquired {
		t.Fatalf("acquired=%+v err=%v", acquired, err)
	}
	failed, err := store.FailGoalRequiredTestAttestationClaimV0(context.Background(), acquired.Claim, orquestagoal.ErrGoalRequiredTestAttestationFailedV0)
	if err != nil || failed.Status != orquestagoal.GoalRequiredTestAttestationClaimStatusFailedV0 {
		t.Fatalf("failed=%+v err=%v", failed, err)
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := recovered.AcquireGoalRequiredTestAttestationClaimV0(context.Background(), request)
	if err != nil || replayed.Acquired || replayed.Claim.Status != orquestagoal.GoalRequiredTestAttestationClaimStatusFailedV0 || replayed.Claim.FailureCode != orquestagoal.ErrGoalRequiredTestAttestationFailedV0 {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
}

func TestStoreV0GoalStateCASAcrossInstancesAllowsSingleWriterAndPersistsOperatorEvidence(t *testing.T) {
	root := t.TempDir()
	first, _ := NewStoreV0(ConfigV0{RootDir: root})
	second, _ := NewStoreV0(ConfigV0{RootDir: root})
	state, err := first.CompareAndSwapGoalWorkStateV0(context.Background(), 0, appDirectorGoalStateForTestV0())
	if err != nil || state.StoreVersion != 1 {
		t.Fatalf("initial state=%+v err=%v", state, err)
	}
	left, _ := first.LoadGoalWorkStateV0(context.Background(), state.RunRef)
	right, _ := second.LoadGoalWorkStateV0(context.Background(), state.RunRef)
	left.EvidenceRefs = append(left.EvidenceRefs, "evidence-ref-cas-left")
	right.EvidenceRefs = append(right.EvidenceRefs, "evidence-ref-cas-right")
	left.LastClosure = operatorAttestationClosureForStoreTestV0()
	right.LastClosure = operatorAttestationClosureForStoreTestV0()
	results := make(chan error, 2)
	go func() {
		_, saveErr := first.CompareAndSwapGoalWorkStateV0(context.Background(), left.StoreVersion, left)
		results <- saveErr
	}()
	go func() {
		_, saveErr := second.CompareAndSwapGoalWorkStateV0(context.Background(), right.StoreVersion, right)
		results <- saveErr
	}()
	successes := 0
	for range 2 {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("CAS successes=%d", successes)
	}
	recovered, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := recovered.LoadGoalWorkStateV0(context.Background(), state.RunRef)
	if err != nil || loaded.StoreVersion != 2 || loaded.LastClosure == nil ||
		len(loaded.LastClosure.AttestationVerifications) != 1 || !loaded.LastClosure.AttestationVerifications[0].Verified {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
}

func TestStoreV0GoalStateCASMultiprocessAllowsSingleWriter(t *testing.T) {
	root := t.TempDir()
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := store.CompareAndSwapGoalWorkStateV0(context.Background(), 0, appDirectorGoalStateForTestV0())
	if err != nil || initial.StoreVersion != 1 {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	writers := []string{"left", "right"}
	errorsByProcess := make([]error, len(writers))
	var wait sync.WaitGroup
	for index, writer := range writers {
		wait.Add(1)
		go func(index int, writer string) {
			defer wait.Done()
			errorsByProcess[index] = runStateFileMultiprocessChildV0(root, "cas_goal_state", writer)
		}(index, writer)
	}
	wait.Wait()
	for _, processErr := range errorsByProcess {
		if processErr != nil {
			t.Fatalf("child CAS: %v", processErr)
		}
	}
	saved := 0
	for _, writer := range writers {
		result, err := os.ReadFile(filepath.Join(root, "multiprocess-result-"+writer))
		if err != nil {
			t.Fatalf("result %s: %v", writer, err)
		}
		if string(result) == "saved" {
			saved++
		}
	}
	loaded, err := store.LoadGoalWorkStateV0(context.Background(), initial.RunRef)
	if err != nil || saved != 1 || loaded.StoreVersion != 2 {
		t.Fatalf("saved=%d loaded=%+v err=%v", saved, loaded, err)
	}
}

func TestStoreV0MultiprocessChildV0(t *testing.T) {
	action := os.Getenv(stateFileMultiprocessActionEnvV0)
	if action == "" {
		t.Skip("subprocess helper")
	}
	root := os.Getenv(stateFileMultiprocessRootEnvV0)
	store, err := NewStoreV0(ConfigV0{RootDir: root})
	if err != nil {
		t.Fatal(err)
	}
	switch action {
	case "save_attestation":
		if err := store.SaveGoalRequiredTestAttestationV0(context.Background(), goalRequiredTestAttestationForStoreTestV0()); err != nil {
			t.Fatal(err)
		}
	case "cas_goal_state":
		writer := os.Getenv(stateFileMultiprocessWriterEnvV0)
		state := appDirectorGoalStateForTestV0()
		state.StoreVersion = 1
		state.EvidenceRefs = append(state.EvidenceRefs, "evidence-ref-multiprocess-"+writer)
		_, err := store.CompareAndSwapGoalWorkStateV0(context.Background(), state.StoreVersion, state)
		result := "saved"
		if err != nil {
			result = "conflict"
		}
		if err := os.WriteFile(filepath.Join(root, "multiprocess-result-"+writer), []byte(result), 0o600); err != nil {
			t.Fatal(err)
		}
	case "claim_attestation":
		writer := os.Getenv(stateFileMultiprocessWriterEnvV0)
		claim, err := store.AcquireGoalRequiredTestAttestationClaimV0(context.Background(), goalRequiredTestAttestationClaimRequestForStoreTestV0())
		if err != nil {
			t.Fatal(err)
		}
		result := "pending"
		if claim.Acquired {
			result = "acquired"
		}
		if err := os.WriteFile(filepath.Join(root, "multiprocess-result-"+writer), []byte(result), 0o600); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown action %q", action)
	}
}

func runStateFileMultiprocessChildV0(root, action, writer string) error {
	command := exec.Command(os.Args[0], "-test.run=^TestStoreV0MultiprocessChildV0$")
	command.Env = append(os.Environ(),
		stateFileMultiprocessActionEnvV0+"="+action,
		stateFileMultiprocessRootEnvV0+"="+root,
		stateFileMultiprocessWriterEnvV0+"="+writer,
	)
	return command.Run()
}

func operatorAttestationClosureForStoreTestV0() *orquestagoal.GoalClosureValidationV0 {
	return &orquestagoal.GoalClosureValidationV0{
		Status: orquestagoal.GoalStatusAcceptedV0, Accepted: true,
		EvidenceRefs: []string{"evidence-ref-operator-attestation-001"},
		AttestationVerifications: []orquestagoal.GoalRequiredTestIdentityVerificationV0{{
			AttestationRef: "attestation-ref-state-file-001", TestRef: "test-ref-state-file-001",
			Verified: true, Independent: true, ImplementerPrincipalRef: "principal-ref-implementer-state-file",
			AttestorPrincipalRef: "principal-ref-attestor-state-file", AttestorCredentialRef: "credential-ref-attestor-state-file",
			TrustPolicyRef: "policy-ref-state-file", EvidenceRefs: []string{"evidence-ref-identity-state-file"},
		}},
	}
}

func goalRequiredTestAttestationForStoreTestV0() orquestagoal.GoalRequiredTestAttestationV0 {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}}
	expected := []orquestagoal.GoalAttestedHashV0{{Ref: writeSet[0].Path, SHA256: hash}}
	receipt := orquestagoal.GoalRequiredTestAttestationV0{
		RunRef: "run-ref-state-file-001", GoalRef: "goal-ref-state-file-001",
		FinalSnapshotRef: goalRequiredTestFinalSnapshotForStoreTestV0().SnapshotRef,
		CheckoutRef:      "checkout-ref-state-file-001", RevisionRef: "revision-ref-state-file-001",
		WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet), TestRef: "test-ref-state-file-001",
		CommandRef: "command-ref-state-file-001", CommandSHA256: hash, DefinitionSHA256: hash,
		Status: orquestagoal.GoalRequiredTestAttestationStatusPassedV0, ImplementerAgentRef: "agent-ref-implementer-state-file",
		AttestorAgentRef: "agent-ref-attestor-state-file", AttestorCredentialRef: "credential-ref-attestor-state-file",
		StartedAt: "2026-07-10T10:00:00Z", FinishedAt: "2026-07-10T10:00:01Z", IsolatedEnvironmentRef: "environment-ref-state-file-001",
		HashesBefore: expected, HashesAfter: expected, EvidenceRefs: []string{"evidence-ref-state-file-attestation"},
	}
	receipt.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(receipt)
	return orquestagoal.NormalizeGoalRequiredTestAttestationV0(receipt)
}

func goalRequiredTestFinalSnapshotForStoreTestV0() orquestagoal.GoalRequiredTestFinalSnapshotV0 {
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}}
	snapshot := orquestagoal.FreezeGoalRequiredTestFinalSnapshotV0(orquestagoal.GoalRequiredTestFinalSnapshotV0{
		RunRef: "run-ref-state-file-001", GoalRef: "goal-ref-state-file-001",
		CheckoutRef: "checkout-ref-state-file-001", RevisionRef: "revision-ref-state-file-001",
		WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet),
		Hashes:         []orquestagoal.GoalAttestedHashV0{{Ref: writeSet[0].Path, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		ObservedAt:     "2026-07-10T10:00:00Z",
	})
	snapshot.EvidenceRefs = []string{"evidence-ref-state-file-snapshot"}
	return snapshot
}

func goalRequiredTestAttestationClaimRequestForStoreTestV0() orquestagoal.GoalRequiredTestAttestationClaimRequestV0 {
	receipt := goalRequiredTestAttestationForStoreTestV0()
	return orquestagoal.GoalRequiredTestAttestationClaimRequestV0{
		RunRef: receipt.RunRef, GoalRef: receipt.GoalRef, RevisionRef: receipt.RevisionRef,
		TestRef: receipt.TestRef, DefinitionSHA256: receipt.DefinitionSHA256,
	}
}
