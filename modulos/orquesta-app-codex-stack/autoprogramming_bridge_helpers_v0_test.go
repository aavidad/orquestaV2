package orquestaappcodexstack

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func autoprogrammingBridgeRequestForTestV0() orquestaautoprogramming.AutoprogrammingRequestV0 {
	request := orquestaautoprogramming.AutoprogrammingRequestV0{
		RequestRef:       "run-autoprogramming-bridge-001",
		ProjectRef:       "project-ref-autoprogramming-bridge-001",
		WorktreeRef:      "worktree-ref-autoprogramming-bridge-001",
		WorktreeIsolated: true,
		BranchRef:        "branch-ref-autoprogramming-bridge-001",
		Tasks: []orquestaautoprogramming.AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "source-task-ref-autoprogramming-bridge-001",
			Area:    "app-codex-stack",
		}},
		WriteSet: []string{
			"modulos/orquesta-app-codex-stack/autoprogramming_bridge_v0.go",
		},
		RequiredTests: []string{
			"go test -count=1 ./modulos/orquesta-app-codex-stack -run TestPrepareAutoprogrammingRunV0",
		},
	}
	return request
}

func enableIndependentAttestationForStackTestV0(stack *StackV0) {
	store := &independentAttestationStoreForStackTestV0{}
	stack.Ports.GoalRequiredTestSpecBinder = independentSpecBinderForStackTestV0{}
	stack.Ports.GoalRequiredTestSnapshotObserver = independentSnapshotObserverForStackTestV0{}
	stack.Ports.GoalRequiredTestAttestor = independentAttestorForStackTestV0{}
	stack.Ports.GoalRequiredTestAttestationStore = store
	stack.Ports.GoalRequiredTestIdentityVerifier = independentIdentityVerifierForStackTestV0{}
}

type independentSpecBinderForStackTestV0 struct{}

func (independentSpecBinderForStackTestV0) BindGoalRequiredTestSpecV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalWorkSpecV0, error) {
	spec.ImplementerAgentRef = "agent-ref-autoprogramming-stack-implementer-001"
	spec.ImplementerCredentialRef = "credential-ref-autoprogramming-stack-implementer-001"
	spec.ClosurePolicy.RequiredAttestorTrustPolicyRef = "policy-ref-autoprogramming-stack-attestor-001"
	return orquestagoal.NormalizeGoalWorkSpecV0(spec), nil
}

type independentSnapshotObserverForStackTestV0 struct{}

func (independentSnapshotObserverForStackTestV0) CaptureGoalRequiredTestFinalSnapshotV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestFinalSnapshotRequestV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	hashes := make([]orquestagoal.GoalAttestedHashV0, 0, len(request.WriteSet))
	for _, scope := range request.WriteSet {
		hashes = append(hashes, orquestagoal.GoalAttestedHashV0{Ref: scope.Path, SHA256: strings.Repeat("a", 64)})
	}
	snapshot := orquestagoal.FreezeGoalRequiredTestFinalSnapshotV0(orquestagoal.GoalRequiredTestFinalSnapshotV0{
		RunRef: request.RunRef, GoalRef: request.GoalRef,
		CheckoutRef: "checkout-ref-observed-stack-001", RevisionRef: "revision-ref-observed-stack-001",
		WriteSetSHA256: request.WriteSetSHA256, Hashes: hashes, ObservedAt: "2026-07-10T10:00:00Z",
	})
	snapshot.EvidenceRefs = []string{"evidence-ref-stack-final-snapshot"}
	return snapshot, nil
}

type independentAttestorForStackTestV0 struct{}

func (independentAttestorForStackTestV0) AttestGoalRequiredTestsV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestAttestationRequestV0,
) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	out := make([]orquestagoal.GoalRequiredTestAttestationV0, 0, len(request.RequiredTests))
	for _, test := range request.RequiredTests {
		receipt := orquestagoal.GoalRequiredTestAttestationV0{
			RunRef: request.RunRef, GoalRef: request.GoalRef, FinalSnapshotRef: request.FinalSnapshot.SnapshotRef,
			CheckoutRef: request.FinalSnapshot.CheckoutRef, RevisionRef: request.FinalSnapshot.RevisionRef,
			WriteSetSHA256: request.FinalSnapshot.WriteSetSHA256, TestRef: test.TestRef, CommandRef: test.CommandRef,
			CommandSHA256: test.CommandSHA256, DefinitionSHA256: test.DefinitionSHA256,
			Status: orquestagoal.GoalRequiredTestAttestationStatusPassedV0, ImplementerAgentRef: request.ImplementerAgentRef,
			AttestorAgentRef: "agent-ref-stack-attestor-001", AttestorCredentialRef: "credential-ref-stack-attestor-001",
			StartedAt: "2026-07-10T10:00:00Z", FinishedAt: "2026-07-10T10:00:01Z",
			IsolatedEnvironmentRef: "environment-ref-stack-attestor-001", ExitCode: 0,
			HashesBefore: append([]orquestagoal.GoalAttestedHashV0(nil), request.FinalSnapshot.Hashes...),
			HashesAfter:  append([]orquestagoal.GoalAttestedHashV0(nil), request.FinalSnapshot.Hashes...),
			EvidenceRefs: []string{"evidence-ref-stack-independent-attestation"},
		}
		receipt.AttestationRef = orquestagoal.GoalRequiredTestAttestationCanonicalRefV0(receipt)
		out = append(out, orquestagoal.NormalizeGoalRequiredTestAttestationV0(receipt))
	}
	return out, nil
}

type independentAttestationStoreForStackTestV0 struct {
	mu       sync.Mutex
	items    []orquestagoal.GoalRequiredTestAttestationV0
	snapshot orquestagoal.GoalRequiredTestFinalSnapshotV0
	claims   map[string]orquestagoal.GoalRequiredTestAttestationClaimV0
}

func (store *independentAttestationStoreForStackTestV0) SaveGoalRequiredTestAttestationV0(_ context.Context, item orquestagoal.GoalRequiredTestAttestationV0) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.items = append(store.items, item)
	return nil
}

func (store *independentAttestationStoreForStackTestV0) ListGoalRequiredTestAttestationsV0(_ context.Context, query orquestagoal.GoalRequiredTestAttestationQueryV0) ([]orquestagoal.GoalRequiredTestAttestationV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	out := []orquestagoal.GoalRequiredTestAttestationV0{}
	for _, item := range store.items {
		if item.RunRef == query.RunRef && item.GoalRef == query.GoalRef && (query.RevisionRef == "" || item.RevisionRef == query.RevisionRef) {
			out = append(out, item)
		}
	}
	return out, nil
}

func (store *independentAttestationStoreForStackTestV0) FreezeGoalRequiredTestFinalSnapshotV0(
	_ context.Context,
	snapshot orquestagoal.GoalRequiredTestFinalSnapshotV0,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.snapshot.SnapshotRef != "" && !orquestagoal.GoalRequiredTestFinalSnapshotIdentityEqualV0(store.snapshot, snapshot) {
		return orquestagoal.GoalRequiredTestFinalSnapshotV0{}, fmt.Errorf("snapshot conflict")
	}
	if store.snapshot.SnapshotRef == "" {
		store.snapshot = snapshot
	}
	return store.snapshot, nil
}

func (store *independentAttestationStoreForStackTestV0) LoadGoalRequiredTestFinalSnapshotV0(
	context.Context,
	string,
	string,
) (orquestagoal.GoalRequiredTestFinalSnapshotV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.snapshot, nil
}

func (store *independentAttestationStoreForStackTestV0) AcquireGoalRequiredTestAttestationClaimV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestAttestationClaimRequestV0,
) (orquestagoal.GoalRequiredTestAttestationClaimResultV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.claims == nil {
		store.claims = map[string]orquestagoal.GoalRequiredTestAttestationClaimV0{}
	}
	ref := orquestagoal.GoalRequiredTestAttestationClaimRefV0(request)
	if claim, ok := store.claims[ref]; ok {
		return orquestagoal.GoalRequiredTestAttestationClaimResultV0{Claim: claim}, nil
	}
	claim := orquestagoal.NormalizeGoalRequiredTestAttestationClaimV0(orquestagoal.GoalRequiredTestAttestationClaimV0{
		ClaimRef: ref, RunRef: request.RunRef, GoalRef: request.GoalRef, RevisionRef: request.RevisionRef,
		TestRef: request.TestRef, DefinitionSHA256: request.DefinitionSHA256,
		Status: orquestagoal.GoalRequiredTestAttestationClaimStatusPendingV0, ClaimedAt: time.Now().UTC().Format(time.RFC3339Nano),
	})
	store.claims[ref] = claim
	return orquestagoal.GoalRequiredTestAttestationClaimResultV0{Claim: claim, Acquired: true}, nil
}

func (store *independentAttestationStoreForStackTestV0) CompleteGoalRequiredTestAttestationClaimV0(
	_ context.Context,
	claim orquestagoal.GoalRequiredTestAttestationClaimV0,
	receipt orquestagoal.GoalRequiredTestAttestationV0,
) error {
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
	claim.Status = orquestagoal.GoalRequiredTestAttestationClaimStatusCompletedV0
	claim.AttestationRef = receipt.AttestationRef
	claim.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	store.claims[claim.ClaimRef] = claim
	return nil
}

type independentIdentityVerifierForStackTestV0 struct{}

func (independentIdentityVerifierForStackTestV0) VerifyGoalRequiredTestIdentityV0(
	_ context.Context,
	request orquestagoal.GoalRequiredTestIdentityVerificationRequestV0,
) (orquestagoal.GoalRequiredTestIdentityVerificationV0, error) {
	return orquestagoal.GoalRequiredTestIdentityVerificationV0{
		AttestationRef: request.AttestationRef, Verified: true, Independent: true,
		ImplementerPrincipalRef: "principal-ref-stack-implementer-001", AttestorPrincipalRef: "principal-ref-stack-attestor-001",
		AttestorCredentialRef: request.AttestorCredentialRef, TrustPolicyRef: request.RequiredTrustPolicyRef,
		EvidenceRefs: []string{"evidence-ref-stack-identity-verification"},
	}, nil
}

func withAutoprogrammingAttestationForTestV0(
	request orquestaautoprogramming.AutoprogrammingRequestV0,
) orquestaautoprogramming.AutoprogrammingRequestV0 {
	return request
}

func autoprogrammingBridgeCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			OccurredAt: "2026-05-22T10:01:00Z",
		},
		Acker: ledger,
	}
}

func autoprogrammingBridgeAgentDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:   store,
			EventSink:  sink,
			Launcher:   orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt: "2026-05-22T10:02:00Z",
		},
		Acker: ledger,
	}
}

func autoprogrammingBridgeStringInSetForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func autoprogrammingBridgeStringContainsForTestV0(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func codexStackPlanStateStepForTestV0(
	t *testing.T,
	state orquestacionnucleoapp.OperationalDirectorPlanStateV0,
	stepID string,
) orquestacionnucleoapp.OperationalDirectorPlanStepStateV0 {
	t.Helper()
	for _, step := range state.Steps {
		if step.StepID == stepID {
			return step
		}
	}
	t.Fatalf("step %s no existe en %+v", stepID, state)
	return orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{}
}
