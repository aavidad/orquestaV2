package orquestagoal

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestGoalRequiredTestAttestationV0CanonicalReceiptCloses(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0)
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{receipt}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: store, SnapshotReader: store, IdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || !closure.Accepted || len(closure.AttestationVerifications) != 1 {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0ContradictoryReceiptsForTestBlock(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	first := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0)
	second := first
	second.EvidenceRefs = append(second.EvidenceRefs, "evidence-ref-contradictory-second")
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{first, second}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: store, SnapshotReader: store, IdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestationMismatchV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0UntrustedIdentityBlocksWithoutStringHeuristic(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0)
	receipt.AttestorAgentRef = spec.ImplementerAgentRef
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{receipt}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: store, SnapshotReader: store, IdentityVerifier: identityVerifierForTestV0{verified: false, independent: false},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestorUntrustedV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0TestMutationAfterSnapshotBlocks(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusFailedV0)
	receipt.HashesAfter[0].SHA256 = hashForAttestationTestV0("mutated-by-test")
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{receipt}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: store, SnapshotReader: store, IdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestSnapshotMismatchV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0InfrastructureFailureBlocksWithoutRework(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusFailedV0)
	receipt.FailureCode = ErrGoalRequiredTestAttestorInfrastructureFailedV0
	receipt.AttestationRef = GoalRequiredTestAttestationCanonicalRefV0(receipt)
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{receipt}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: store, SnapshotReader: store, IdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || closure.NeedsRework || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestorInfrastructureFailedV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0OrdinaryFailureNeedsRework(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusFailedV0)
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{receipt}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: store, SnapshotReader: store, IdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !closure.NeedsRework || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestationFailedV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0OrdinaryFailurePreservesLegacyCanonicalRef(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusFailedV0)
	receipt.FailureCode = ""
	legacyRef := GoalRequiredTestAttestationCanonicalRefV0(receipt)
	receipt.FailureCode = ErrGoalRequiredTestAttestationFailedV0
	if currentRef := GoalRequiredTestAttestationCanonicalRefV0(receipt); currentRef != legacyRef {
		t.Fatalf("ordinary failed receipt changed canonical identity: legacy=%s current=%s", legacyRef, currentRef)
	}
}

func TestGoalRequiredTestAttestationV0LifecycleFailsClosedWithoutSnapshotPort(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	stateStore := &attestationGoalStateStoreForTestV0{state: attestationRunningStateForTestV0(spec)}
	result, err := ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, GoalWorkLifecyclePortsV0{
		Observer:             attestationObserverForTestV0{result: completedAttestedGoalResultForTestV0(spec)},
		RequiredTestAttestor: &attestorForTestV0{}, RequiredTestAttestationStore: &attestationStoreForTestV0{},
		RequiredTestIdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
		ClosureValidator:             acceptingClosureValidatorForTestV0{}, StateStore: stateStore,
	})
	if err != nil || result.Accepted || !hasAttestationIssueForTestV0(result.Closure, ErrGoalRequiredTestAttestationMissingV0) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestGoalRequiredTestAttestationV0PartialReplayClaimsAndRunsOnlyMissingTest(t *testing.T) {
	spec := attestationSpecForTestV0(2)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	first := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0)
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{first}}
	attestor := &attestorForTestV0{}
	result := observeAttestedGoalForTestV0(t, spec, snapshot, store, attestor)
	if !result.Accepted || len(attestor.requests) != 1 || len(attestor.requests[0].RequiredTests) != 1 ||
		attestor.requests[0].RequiredTests[0].TestRef != spec.RequiredTests[1].TestRef {
		t.Fatalf("closure=%+v requests=%+v", result.Closure, attestor.requests)
	}
	if len(store.items) != 2 {
		t.Fatalf("receipts=%d", len(store.items))
	}
}

func TestGoalRequiredTestAttestationV0ConcurrentClaimsRunOneAttestor(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	store := &attestationStoreForTestV0{snapshot: snapshot}
	attestor := &attestorForTestV0{}
	var wait sync.WaitGroup
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			stateStore := &attestationGoalStateStoreForTestV0{state: attestationRunningStateForTestV0(spec)}
			_, _ = ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, attestedLifecyclePortsForTestV0(snapshot, store, attestor, stateStore))
		}()
	}
	wait.Wait()
	if attestor.calls != 1 || len(store.items) != 1 {
		t.Fatalf("calls=%d receipts=%d", attestor.calls, len(store.items))
	}
}

func TestGoalRequiredTestAttestationV0ErrorTrasClaimPersisteReworkSinReintento(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	store := &attestationStoreForTestV0{snapshot: snapshot}
	attestor := &attestorForTestV0{err: errors.New("attestor unavailable")}
	stateStore := &attestationGoalStateStoreForTestV0{state: attestationRunningStateForTestV0(spec)}
	ports := attestedLifecyclePortsForTestV0(snapshot, store, attestor, stateStore)
	first, err := ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, ports)
	if err != nil || first.Accepted || !first.NeedsRework || !hasAttestationIssueForTestV0(first.Closure, ErrGoalRequiredTestAttestationFailedV0) {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, ports)
	if err != nil || second.Accepted || !second.NeedsRework || attestor.calls != 1 {
		t.Fatalf("second=%+v calls=%d err=%v", second, attestor.calls, err)
	}
	for _, claim := range store.claims {
		if claim.Status != GoalRequiredTestAttestationClaimStatusFailedV0 || claim.FailureCode != ErrGoalRequiredTestAttestationFailedV0 {
			t.Fatalf("claim=%+v", claim)
		}
	}
}

func TestGoalRequiredTestAttestationV0InfrastructureFailureReplayDoesNotRunAttestor(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	snapshot := attestationSnapshotForTestV0(spec, "revision-ref-current")
	receipt := attestationForTestV0(spec, snapshot, spec.RequiredTests[0], GoalRequiredTestAttestationStatusFailedV0)
	receipt.FailureCode = ErrGoalRequiredTestAttestorInfrastructureFailedV0
	receipt.AttestationRef = GoalRequiredTestAttestationCanonicalRefV0(receipt)
	store := &attestationStoreForTestV0{snapshot: snapshot, items: []GoalRequiredTestAttestationV0{receipt}}
	attestor := &attestorForTestV0{}
	stateStore := &attestationGoalStateStoreForTestV0{state: attestationRunningStateForTestV0(spec)}
	result, err := ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, attestedLifecyclePortsForTestV0(snapshot, store, attestor, stateStore))
	if err != nil || result.Accepted || result.NeedsRework || attestor.calls != 0 ||
		!hasAttestationIssueForTestV0(result.Closure, ErrGoalRequiredTestAttestorInfrastructureFailedV0) {
		t.Fatalf("result=%+v calls=%d err=%v", result, attestor.calls, err)
	}
}

func TestGoalRequiredTestSpecBinderRequiredBeforeLaunch(t *testing.T) {
	spec := attestationSpecForTestV0(1)
	spec.ImplementerAgentRef = ""
	spec.ImplementerCredentialRef = ""
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = ""
	_, err := StartGoalWorkV0(context.Background(), GoalWorkStartRequestV0{RunRef: spec.RunRef, Spec: spec}, GoalWorkLifecyclePortsV0{
		Launcher: launcherForAttestationTestV0{}, StateStore: &attestationGoalStateStoreForTestV0{},
	})
	if err == nil {
		t.Fatal("expected fail-closed binder error")
	}
}

func attestationSpecForTestV0(count int) GoalWorkSpecV0 {
	tests := make([]GoalRequiredTestV0, 0, count)
	for i := 0; i < count; i++ {
		tests = append(tests, FreezeGoalRequiredTestV0(GoalRequiredTestV0{
			TestRef: fmt.Sprintf("test-ref-attestation-%d", i+1), CommandRef: fmt.Sprintf("command-ref-attestation-%d", i+1),
			Command: fmt.Sprintf("go test ./module-%d", i+1),
		}))
	}
	writeSet := []GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}}
	return NormalizeGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef: "goal-ref-attestation-001", RunRef: "run-ref-attestation-001",
		ImplementerAgentRef: "agent-ref-implementer-001", ImplementerCredentialRef: "credential-ref-implementer-001",
		Objective: "Implement independent attestation.", DirectorKind: GoalDirectorKindRuntimeGoalV0,
		WriteSet: writeSet, WriteSetSHA256: GoalWriteSetSHA256V0(writeSet), RequiredTests: tests,
		ClosurePolicy: GoalClosurePolicyV0{
			RequireRequiredTests: true, RequireIndependentRequiredTestAttestation: true,
			RequiredAttestorTrustPolicyRef: "policy-ref-attestor-trust-001",
		},
	})
}

func attestationSnapshotForTestV0(spec GoalWorkSpecV0, revision string) GoalRequiredTestFinalSnapshotV0 {
	snapshot := FreezeGoalRequiredTestFinalSnapshotV0(GoalRequiredTestFinalSnapshotV0{
		RunRef: spec.RunRef, GoalRef: spec.GoalRef, CheckoutRef: "checkout-ref-observed-001", RevisionRef: revision,
		WriteSetSHA256: spec.WriteSetSHA256,
		Hashes:         []GoalAttestedHashV0{{Ref: spec.WriteSet[0].Path, SHA256: hashForAttestationTestV0("final-checkout-tree")}},
		ObservedAt:     "2026-07-10T10:00:00Z",
	})
	snapshot.EvidenceRefs = []string{"evidence-ref-final-snapshot-001"}
	return snapshot
}

func attestationForTestV0(spec GoalWorkSpecV0, snapshot GoalRequiredTestFinalSnapshotV0, test GoalRequiredTestV0, status string) GoalRequiredTestAttestationV0 {
	receipt := GoalRequiredTestAttestationV0{
		RunRef: spec.RunRef, GoalRef: spec.GoalRef, FinalSnapshotRef: snapshot.SnapshotRef,
		CheckoutRef: snapshot.CheckoutRef, RevisionRef: snapshot.RevisionRef, WriteSetSHA256: snapshot.WriteSetSHA256,
		TestRef: test.TestRef, CommandRef: test.CommandRef, CommandSHA256: test.CommandSHA256, DefinitionSHA256: test.DefinitionSHA256,
		Status: status, ImplementerAgentRef: spec.ImplementerAgentRef,
		FailureCode:      map[string]string{GoalRequiredTestAttestationStatusFailedV0: ErrGoalRequiredTestAttestationFailedV0}[status],
		AttestorAgentRef: "agent-ref-attestor-001", AttestorCredentialRef: "credential-ref-attestor-001",
		StartedAt: "2026-07-10T10:00:00Z", FinishedAt: "2026-07-10T10:00:01Z",
		IsolatedEnvironmentRef: "environment-ref-isolated-001",
		ExitCode:               map[string]int{GoalRequiredTestAttestationStatusPassedV0: 0, GoalRequiredTestAttestationStatusFailedV0: 1}[status],
		HashesBefore:           append([]GoalAttestedHashV0(nil), snapshot.Hashes...), HashesAfter: append([]GoalAttestedHashV0(nil), snapshot.Hashes...),
		EvidenceRefs: []string{"evidence-ref-attestation-" + test.TestRef},
	}
	receipt.AttestationRef = GoalRequiredTestAttestationCanonicalRefV0(receipt)
	return NormalizeGoalRequiredTestAttestationV0(receipt)
}

func completedAttestedGoalResultForTestV0(spec GoalWorkSpecV0) GoalWorkResultV0 {
	results := make([]GoalRequiredTestResultV0, 0, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		results = append(results, GoalRequiredTestResultV0{TestRef: test.TestRef, Status: "passed", EvidenceRefs: []string{"evidence-ref-implementer-diagnostic"}})
	}
	return NormalizeGoalWorkResultV0(GoalWorkResultV0{GoalRef: spec.GoalRef, Status: GoalStatusCompleteV0, RequiredTestResults: results})
}

func attestationRunningStateForTestV0(spec GoalWorkSpecV0) GoalWorkStateV0 {
	return GoalWorkStateV0{
		RunRef: spec.RunRef, GoalRef: spec.GoalRef, ExternalGoalRef: spec.GoalRef, Status: GoalStatusRunningV0, Spec: spec,
		LaunchReceipt: GoalLaunchReceiptV0{GoalRef: spec.GoalRef, ExternalGoalRef: spec.GoalRef, Status: GoalStatusRunningV0},
	}
}

func observeAttestedGoalForTestV0(t *testing.T, spec GoalWorkSpecV0, snapshot GoalRequiredTestFinalSnapshotV0, store *attestationStoreForTestV0, attestor *attestorForTestV0) GoalWorkObserveResultV0 {
	t.Helper()
	stateStore := &attestationGoalStateStoreForTestV0{state: attestationRunningStateForTestV0(spec)}
	result, err := ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, attestedLifecyclePortsForTestV0(snapshot, store, attestor, stateStore))
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	return result
}

func attestedLifecyclePortsForTestV0(snapshot GoalRequiredTestFinalSnapshotV0, store *attestationStoreForTestV0, attestor *attestorForTestV0, stateStore *attestationGoalStateStoreForTestV0) GoalWorkLifecyclePortsV0 {
	return GoalWorkLifecyclePortsV0{
		Observer:                     attestationObserverForTestV0{result: completedAttestedGoalResultForTestV0(stateStore.state.Spec)},
		RequiredTestSnapshotObserver: snapshotObserverForTestV0{snapshot: snapshot},
		RequiredTestAttestor:         attestor, RequiredTestAttestationStore: store,
		RequiredTestIdentityVerifier: identityVerifierForTestV0{verified: true, independent: true},
		ClosureValidator:             DefaultGoalWorkClosureValidatorV0{}, StateStore: stateStore,
	}
}

type snapshotObserverForTestV0 struct {
	snapshot GoalRequiredTestFinalSnapshotV0
}

func (observer snapshotObserverForTestV0) CaptureGoalRequiredTestFinalSnapshotV0(context.Context, GoalRequiredTestFinalSnapshotRequestV0) (GoalRequiredTestFinalSnapshotV0, error) {
	return observer.snapshot, nil
}

type attestationObserverForTestV0 struct{ result GoalWorkResultV0 }

func (observer attestationObserverForTestV0) ObserveGoalWorkV0(context.Context, GoalObservationRequestV0) (GoalWorkResultV0, error) {
	return observer.result, nil
}

type attestorForTestV0 struct {
	mu       sync.Mutex
	calls    int
	requests []GoalRequiredTestAttestationRequestV0
	err      error
}

func (attestor *attestorForTestV0) AttestGoalRequiredTestsV0(_ context.Context, request GoalRequiredTestAttestationRequestV0) ([]GoalRequiredTestAttestationV0, error) {
	attestor.mu.Lock()
	defer attestor.mu.Unlock()
	attestor.calls++
	attestor.requests = append(attestor.requests, request)
	if attestor.err != nil {
		return nil, attestor.err
	}
	test := request.RequiredTests[0]
	spec := attestationSpecForTestV0(1)
	spec.RunRef, spec.GoalRef = request.RunRef, request.GoalRef
	spec.ImplementerAgentRef = request.ImplementerAgentRef
	return []GoalRequiredTestAttestationV0{attestationForTestV0(spec, request.FinalSnapshot, test, GoalRequiredTestAttestationStatusPassedV0)}, nil
}

type identityVerifierForTestV0 struct{ verified, independent bool }

func (verifier identityVerifierForTestV0) VerifyGoalRequiredTestIdentityV0(_ context.Context, request GoalRequiredTestIdentityVerificationRequestV0) (GoalRequiredTestIdentityVerificationV0, error) {
	return GoalRequiredTestIdentityVerificationV0{
		AttestationRef: request.AttestationRef, TestRef: request.TestRef, Verified: verifier.verified, Independent: verifier.independent,
		ImplementerPrincipalRef: "principal-ref-implementer-001", AttestorPrincipalRef: "principal-ref-attestor-001",
		AttestorCredentialRef: request.AttestorCredentialRef, TrustPolicyRef: request.RequiredTrustPolicyRef,
		EvidenceRefs: []string{"evidence-ref-identity-verification-001"},
	}, nil
}

type acceptingClosureValidatorForTestV0 struct{}

func (acceptingClosureValidatorForTestV0) ValidateGoalWorkClosureV0(context.Context, GoalWorkSpecV0, GoalWorkResultV0) (GoalClosureValidationV0, error) {
	return GoalClosureValidationV0{Status: GoalStatusAcceptedV0, Accepted: true}, nil
}

type attestationStoreForTestV0 struct {
	mu       sync.Mutex
	snapshot GoalRequiredTestFinalSnapshotV0
	items    []GoalRequiredTestAttestationV0
	claims   map[string]GoalRequiredTestAttestationClaimV0
}

func (store *attestationStoreForTestV0) SaveGoalRequiredTestAttestationV0(_ context.Context, item GoalRequiredTestAttestationV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.items = append(store.items, item)
	return nil
}

func (store *attestationStoreForTestV0) ListGoalRequiredTestAttestationsV0(_ context.Context, query GoalRequiredTestAttestationQueryV0) ([]GoalRequiredTestAttestationV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	out := []GoalRequiredTestAttestationV0{}
	for _, item := range store.items {
		if item.RunRef == query.RunRef && item.GoalRef == query.GoalRef && (query.RevisionRef == "" || item.RevisionRef == query.RevisionRef) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (store *attestationStoreForTestV0) FreezeGoalRequiredTestFinalSnapshotV0(_ context.Context, snapshot GoalRequiredTestFinalSnapshotV0) (GoalRequiredTestFinalSnapshotV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.snapshot.SnapshotRef != "" && !GoalRequiredTestFinalSnapshotIdentityEqualV0(store.snapshot, snapshot) {
		return GoalRequiredTestFinalSnapshotV0{}, fmt.Errorf("snapshot conflict")
	}
	if store.snapshot.SnapshotRef == "" {
		store.snapshot = snapshot
	}
	return store.snapshot, nil
}

func (store *attestationStoreForTestV0) LoadGoalRequiredTestFinalSnapshotV0(context.Context, string, string) (GoalRequiredTestFinalSnapshotV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.snapshot.SnapshotRef == "" {
		return GoalRequiredTestFinalSnapshotV0{}, fmt.Errorf("snapshot missing")
	}
	return store.snapshot, nil
}

func (store *attestationStoreForTestV0) AcquireGoalRequiredTestAttestationClaimV0(_ context.Context, request GoalRequiredTestAttestationClaimRequestV0) (GoalRequiredTestAttestationClaimResultV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.claims == nil {
		store.claims = map[string]GoalRequiredTestAttestationClaimV0{}
	}
	ref := GoalRequiredTestAttestationClaimRefV0(request)
	if claim, ok := store.claims[ref]; ok {
		return GoalRequiredTestAttestationClaimResultV0{Claim: claim}, nil
	}
	claim := NormalizeGoalRequiredTestAttestationClaimV0(GoalRequiredTestAttestationClaimV0{
		ClaimRef: ref, RunRef: request.RunRef, GoalRef: request.GoalRef, RevisionRef: request.RevisionRef,
		TestRef: request.TestRef, DefinitionSHA256: request.DefinitionSHA256,
		Status: GoalRequiredTestAttestationClaimStatusPendingV0, ClaimedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	store.claims[ref] = claim
	return GoalRequiredTestAttestationClaimResultV0{Claim: claim, Acquired: true}, nil
}

func (store *attestationStoreForTestV0) CompleteGoalRequiredTestAttestationClaimV0(_ context.Context, claim GoalRequiredTestAttestationClaimV0, receipt GoalRequiredTestAttestationV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, current := range store.items {
		if current.GoalRef == receipt.GoalRef && current.RevisionRef == receipt.RevisionRef && current.TestRef == receipt.TestRef {
			if !reflect.DeepEqual(current, receipt) {
				return fmt.Errorf("receipt conflict")
			}
			return nil
		}
	}
	store.items = append(store.items, receipt)
	claim.Status = GoalRequiredTestAttestationClaimStatusCompletedV0
	claim.AttestationRef = receipt.AttestationRef
	claim.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	store.claims[claim.ClaimRef] = claim
	return nil
}

func (store *attestationStoreForTestV0) FailGoalRequiredTestAttestationClaimV0(_ context.Context, claim GoalRequiredTestAttestationClaimV0, code string) (GoalRequiredTestAttestationClaimV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	current := store.claims[claim.ClaimRef]
	current.Status = GoalRequiredTestAttestationClaimStatusFailedV0
	current.FailureCode = code
	current.FailedAt = time.Now().UTC().Format(time.RFC3339Nano)
	store.claims[claim.ClaimRef] = current
	return current, nil
}

type attestationGoalStateStoreForTestV0 struct {
	mu    sync.Mutex
	state GoalWorkStateV0
}

func (store *attestationGoalStateStoreForTestV0) SaveGoalWorkStateV0(_ context.Context, state GoalWorkStateV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.state = state
	return nil
}

func (store *attestationGoalStateStoreForTestV0) LoadGoalWorkStateV0(context.Context, string) (GoalWorkStateV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.state, nil
}

type launcherForAttestationTestV0 struct{}

func (launcherForAttestationTestV0) LaunchGoalWorkV0(context.Context, GoalWorkSpecV0) (GoalLaunchReceiptV0, error) {
	return GoalLaunchReceiptV0{Status: GoalStatusRunningV0, GoalRef: "goal-ref-attestation-001", ExternalGoalRef: "goal-ref-attestation-001"}, nil
}

func hasAttestationIssueForTestV0(closure GoalClosureValidationV0, code string) bool {
	for _, issue := range closure.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func hashForAttestationTestV0(value string) string { return goalRequiredTestCommandSHA256V0(value) }
