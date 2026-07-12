package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestGoalFirstRepairReceiptV0MuerdeMutacionValidadorDirectoSinAtestacion(t *testing.T) {
	ctx := context.Background()
	state := goalMaterializedRefsStateForTestV0(t, "run-goal-repair-lifecycle-attestation-001", "docs")
	test := orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
		TestRef:    "test-ref-goal-repair-lifecycle-attestation-001",
		CommandRef: "command-ref-goal-repair-lifecycle-attestation-001",
		Command:    "go test -count=1 ./modulos/orquesta-app-codex-stack",
	})
	state.Spec.RequiredTests = []orquestagoal.GoalRequiredTestV0{test}
	state.Spec.WriteSetSHA256 = orquestagoal.GoalWriteSetSHA256V0(state.Spec.WriteSet)
	state.Spec.ImplementerAgentRef = "agent-ref-goal-repair-implementer-001"
	state.Spec.ImplementerCredentialRef = "credential-ref-goal-repair-implementer-001"
	state.Spec.ClosurePolicy = orquestagoal.GoalClosurePolicyV0{
		RequireRequiredTests:                      true,
		RequireIndependentRequiredTestAttestation: true,
		RequiredAttestorTrustPolicyRef:            "policy-ref-goal-repair-attestor-001",
	}
	var err error
	state, err = orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		t.Fatalf("NewGoalWorkStateV0: %v", err)
	}
	stateStore := newGoalFirstQueueStateStoreForTestV0()
	if err := stateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	attestationStore := &independentAttestationStoreForStackTestV0{}
	snapshotObserver := &goalFirstRepairSnapshotObserverForTestV0{}
	attestor := &goalFirstRepairAttestorForTestV0{}
	validator := &goalFirstRepairOrderValidatorForTestV0{store: attestationStore}
	result := orquestagoal.NormalizeGoalWorkResultV0(orquestagoal.GoalWorkResultV0{
		Status:  orquestagoal.GoalStatusCompleteV0,
		GoalRef: state.GoalRef,
		Summary: "receipt materializado recuperado por lifecycle",
	})

	repaired, err := repairGoalFirstReceiptFromMaterializedResultV0(
		ctx,
		state,
		result,
		orquestagoal.GoalWorkLifecyclePortsV0{
			StateStore:                   stateStore,
			ClosureValidator:             validator,
			RequiredTestSnapshotObserver: snapshotObserver,
			RequiredTestAttestor:         attestor,
			RequiredTestAttestationStore: attestationStore,
			RequiredTestIdentityVerifier: goalFirstRepairIdentityVerifierForTestV0{},
		},
	)
	if err != nil {
		t.Fatalf("repairGoalFirstReceiptFromMaterializedResultV0: %v", err)
	}
	if !repaired.Repaired || !repaired.Closure.Accepted || snapshotObserver.calls != 1 || attestor.calls != 1 || validator.calls != 1 {
		t.Fatalf("repair=%+v snapshot_calls=%d attestor_calls=%d validator_calls=%d", repaired, snapshotObserver.calls, attestor.calls, validator.calls)
	}
	if attestationStore.snapshot.SnapshotRef == "" || len(attestationStore.items) != 1 {
		t.Fatalf("snapshot=%+v attestations=%+v", attestationStore.snapshot, attestationStore.items)
	}
	persisted, err := stateStore.LoadGoalWorkStateV0(ctx, state.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.LastClosure == nil || !persisted.LastClosure.Accepted || persisted.LastResult == nil ||
		!goalFirstStringSliceContainsV0(persisted.LastResult.EvidenceRefs, goalFirstRepairReceiptAttemptedEvidenceRefV0) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

type goalFirstRepairSnapshotObserverForTestV0 struct {
	calls int
}

func (observer *goalFirstRepairSnapshotObserverForTestV0) CaptureGoalRequiredTestFinalSnapshotV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestFinalSnapshotRequestV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	observer.calls++
	hashes := make([]orquestagoal.GoalAttestedHashV0, 0, len(request.WriteSet))
	for _, scope := range request.WriteSet {
		hashes = append(hashes, orquestagoal.GoalAttestedHashV0{Ref: scope.Path, SHA256: strings.Repeat("a", 64)})
	}
	snapshot := orquestagoal.FreezeGoalRequiredTestFinalSnapshotV0(orquestagoal.GoalRequiredTestFinalSnapshotV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef,
		CheckoutRef: "checkout-ref-goal-repair-001", RevisionRef: "revision-ref-goal-repair-001",
		WriteSetSHA256: request.WriteSetSHA256, Hashes: hashes, ObservedAt: "2026-07-12T12:00:00Z",
	})
	snapshot.EvidenceRefs = []string{"evidence-ref-goal-repair-snapshot-001"}
	return snapshot, nil
}

type goalFirstRepairAttestorForTestV0 struct {
	calls int
}

func (attestor *goalFirstRepairAttestorForTestV0) AttestGoalRequiredTestsV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	attestor.calls++
	if len(request.RequiredTests) != 1 {
		return nil, fmt.Errorf("required test unico esperado")
	}
	test := request.RequiredTests[0]
	receipt := orquestagoal.GoalRequiredTestAttestationV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef,
		FinalSnapshotRef: request.FinalSnapshot.SnapshotRef,
		CheckoutRef:      request.FinalSnapshot.CheckoutRef, RevisionRef: request.FinalSnapshot.RevisionRef,
		WriteSetSHA256: request.FinalSnapshot.WriteSetSHA256,
		TestRef:        test.TestRef, CommandRef: test.CommandRef,
		CommandSHA256: test.CommandSHA256, DefinitionSHA256: test.DefinitionSHA256,
		Status:              orquestagoal.GoalRequiredTestAttestationStatusPassedV0,
		ImplementerAgentRef: request.ImplementerAgentRef,
		AttestorAgentRef:    "agent-ref-goal-repair-attestor-001", AttestorCredentialRef: "credential-ref-goal-repair-attestor-001",
		StartedAt: "2026-07-12T12:00:00Z", FinishedAt: "2026-07-12T12:00:01Z",
		IsolatedEnvironmentRef: "environment-ref-goal-repair-attestor-001",
		HashesBefore:           append([]orquestagoal.GoalAttestedHashV0(nil), request.FinalSnapshot.Hashes...),
		HashesAfter:            append([]orquestagoal.GoalAttestedHashV0(nil), request.FinalSnapshot.Hashes...),
		EvidenceRefs:           []string{"evidence-ref-goal-repair-attestation-001"},
	}
	receipt.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(receipt)
	return []orquestagoal.GoalRequiredTestAttestationV0{orquestagoal.NormalizeGoalRequiredTestAttestationV0(receipt)}, nil
}

type goalFirstRepairOrderValidatorForTestV0 struct {
	store *independentAttestationStoreForStackTestV0
	calls int
}

func (validator *goalFirstRepairOrderValidatorForTestV0) ValidateGoalWorkClosureV0(
	_ context.Context,
	_ orquestagoal.GoalWorkSpecV0,
	_ orquestagoal.GoalWorkResultV0,
) (orquestagoal.GoalClosureValidationV0, error) {
	validator.calls++
	if validator.store.snapshot.SnapshotRef == "" || len(validator.store.items) != 1 {
		return orquestagoal.GoalClosureValidationV0{}, fmt.Errorf("validator ejecutado antes de snapshot y atestacion")
	}
	return orquestagoal.GoalClosureValidationV0{Status: orquestagoal.GoalStatusAcceptedV0, Accepted: true}, nil
}

type goalFirstRepairIdentityVerifierForTestV0 struct{}

func (goalFirstRepairIdentityVerifierForTestV0) VerifyGoalRequiredTestIdentityV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestIdentityVerificationRequestV0,
) (orquestagoal.GoalRequiredTestIdentityVerificationV0, error) {
	return orquestagoal.GoalRequiredTestIdentityVerificationV0{
		AttestationRef: request.AttestationRef, TestRef: request.TestRef,
		Verified: true, Independent: true,
		ImplementerPrincipalRef: "principal-ref-goal-repair-implementer-001",
		AttestorPrincipalRef:    "principal-ref-goal-repair-attestor-001",
		AttestorCredentialRef:   request.AttestorCredentialRef,
		TrustPolicyRef:          request.RequiredTrustPolicyRef,
		EvidenceRefs:            []string{"evidence-ref-goal-repair-identity-001"},
	}, nil
}
