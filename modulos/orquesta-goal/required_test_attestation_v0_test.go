package orquestagoal

import (
	"context"
	"fmt"
	"testing"
)

func TestGoalRequiredTestAttestationV0SelfReportedPassButIndependentFailBlocks(t *testing.T) {
	spec := attestationSpecForTestV0("revision-ref-001", 1)
	store := &attestationStoreForTestV0{}
	attestor := &attestorForTestV0{attestations: []GoalRequiredTestAttestationV0{
		attestationForTestV0(spec, spec.RequiredTests[0], GoalRequiredTestAttestationStatusFailedV0),
	}}
	result := observeAttestedGoalForTestV0(t, spec, store, attestor)
	if result.Accepted || !result.NeedsRework || !hasAttestationIssueForTestV0(result.Closure, ErrGoalRequiredTestAttestationFailedV0) {
		t.Fatalf("closure=%+v", result.Closure)
	}
	// The implementer assertion is preserved in result but is not authority.
	if result.Result.RequiredTestResults[0].Status != "passed" {
		t.Fatalf("diagnostico implementer perdido: %+v", result.Result.RequiredTestResults)
	}
}

func TestGoalRequiredTestAttestationV0SameIdentityRejected(t *testing.T) {
	spec := attestationSpecForTestV0("revision-ref-002", 1)
	attestation := attestationForTestV0(spec, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0)
	attestation.AttestorAgentRef = spec.ImplementerAgentRef
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: attestationReaderForTestV0{items: []GoalRequiredTestAttestationV0{attestation}},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestationMismatchV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0TestHashMutatedBlocks(t *testing.T) {
	spec := attestationSpecForTestV0("revision-ref-003", 1)
	attestation := attestationForTestV0(spec, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0)
	attestation.CommandSHA256 = hashForAttestationTestV0("mutated")
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{
		Reader: attestationReaderForTestV0{items: []GoalRequiredTestAttestationV0{attestation}},
	}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestationMismatchV0) {
		t.Fatalf("closure=%+v err=%v", closure, err)
	}
}

func TestGoalRequiredTestAttestationV0IndependentPassedClosesAndReplayIsIdempotent(t *testing.T) {
	spec := attestationSpecForTestV0("revision-ref-004", 1)
	store := &attestationStoreForTestV0{}
	attestor := &attestorForTestV0{attestations: []GoalRequiredTestAttestationV0{
		attestationForTestV0(spec, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0),
	}}
	first := observeAttestedGoalForTestV0(t, spec, store, attestor)
	second := observeAttestedGoalForTestV0(t, spec, store, attestor)
	if !first.Accepted || !second.Accepted || attestor.calls != 1 || len(store.items) != 1 {
		t.Fatalf("first=%+v second=%+v calls=%d items=%d", first.Closure, second.Closure, attestor.calls, len(store.items))
	}
}

func TestGoalRequiredTestAttestationV0SeveralTestsAndStaleRevisionDoesNotClose(t *testing.T) {
	spec := attestationSpecForTestV0("revision-ref-current", 2)
	stale := attestationSpecForTestV0("revision-ref-stale", 2)
	store := &attestationStoreForTestV0{items: []GoalRequiredTestAttestationV0{
		attestationForTestV0(stale, stale.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0),
		attestationForTestV0(stale, stale.RequiredTests[1], GoalRequiredTestAttestationStatusPassedV0),
	}}
	closure, err := (IndependentGoalRequiredTestAttestationClosureValidatorV0{Reader: store}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || closure.Accepted || !hasAttestationIssueForTestV0(closure, ErrGoalRequiredTestAttestationMissingV0) {
		t.Fatalf("stale closure=%+v err=%v", closure, err)
	}
	store.items = []GoalRequiredTestAttestationV0{
		attestationForTestV0(spec, spec.RequiredTests[0], GoalRequiredTestAttestationStatusPassedV0),
		attestationForTestV0(spec, spec.RequiredTests[1], GoalRequiredTestAttestationStatusPassedV0),
	}
	closure, err = (IndependentGoalRequiredTestAttestationClosureValidatorV0{Reader: store}).ValidateGoalWorkClosureV0(context.Background(), spec, completedAttestedGoalResultForTestV0(spec))
	if err != nil || !closure.Accepted {
		t.Fatalf("multiple closure=%+v err=%v", closure, err)
	}
}

func attestationSpecForTestV0(revision string, count int) GoalWorkSpecV0 {
	tests := make([]GoalRequiredTestV0, 0, count)
	for i := 0; i < count; i++ {
		command := fmt.Sprintf("go test ./module-%d", i+1)
		tests = append(tests, GoalRequiredTestV0{
			TestRef:          fmt.Sprintf("test-ref-attestation-%d", i+1),
			CommandRef:       fmt.Sprintf("command-ref-attestation-%d", i+1),
			Command:          command,
			CommandSHA256:    goalRequiredTestCommandSHA256V0(command),
			DefinitionSHA256: hashForAttestationTestV0(fmt.Sprintf("definition-%d", i+1)),
		})
	}
	return NormalizeGoalWorkSpecV0(GoalWorkSpecV0{
		GoalRef:             "goal-ref-attestation-001",
		RunRef:              "run-ref-attestation-001",
		RevisionRef:         revision,
		ImplementerAgentRef: "agent-ref-implementer-001",
		Objective:           "Implement independent attestation.",
		DirectorKind:        GoalDirectorKindRuntimeGoalV0,
		WriteSet:            []GoalWriteScopeV0{{Path: "modulos/orquesta-goal"}},
		RequiredTests:       tests,
		ClosurePolicy: GoalClosurePolicyV0{
			RequireRequiredTests:                      true,
			RequireIndependentRequiredTestAttestation: true,
		},
	})
}

func completedAttestedGoalResultForTestV0(spec GoalWorkSpecV0) GoalWorkResultV0 {
	results := make([]GoalRequiredTestResultV0, 0, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		results = append(results, GoalRequiredTestResultV0{TestRef: test.TestRef, Status: "passed", EvidenceRefs: []string{"evidence-ref-implementer-diagnostic"}})
	}
	return NormalizeGoalWorkResultV0(GoalWorkResultV0{GoalRef: spec.GoalRef, Status: GoalStatusCompleteV0, RequiredTestResults: results})
}

func attestationForTestV0(spec GoalWorkSpecV0, test GoalRequiredTestV0, status string) GoalRequiredTestAttestationV0 {
	return NormalizeGoalRequiredTestAttestationV0(GoalRequiredTestAttestationV0{
		AttestationRef:         "attestation-ref-" + spec.RevisionRef + "-" + test.TestRef,
		RunRef:                 spec.RunRef,
		GoalRef:                spec.GoalRef,
		RevisionRef:            spec.RevisionRef,
		TestRef:                test.TestRef,
		CommandRef:             test.CommandRef,
		CommandSHA256:          test.CommandSHA256,
		DefinitionSHA256:       test.DefinitionSHA256,
		Status:                 status,
		ImplementerAgentRef:    spec.ImplementerAgentRef,
		AttestorAgentRef:       "agent-ref-attestor-001",
		AttestorCredentialRef:  "credential-ref-attestor-001",
		StartedAt:              "2026-07-10T10:00:00Z",
		FinishedAt:             "2026-07-10T10:00:01Z",
		IsolatedEnvironmentRef: "environment-ref-isolated-001",
		ExitCode:               map[string]int{GoalRequiredTestAttestationStatusPassedV0: 0, GoalRequiredTestAttestationStatusFailedV0: 1}[status],
		HashesBefore:           []GoalAttestedHashV0{{Ref: "hash-ref-before", SHA256: hashForAttestationTestV0("before")}},
		HashesAfter:            []GoalAttestedHashV0{{Ref: "hash-ref-after", SHA256: hashForAttestationTestV0("after")}},
		EvidenceRefs:           []string{"evidence-ref-attestation-" + test.TestRef},
	})
}

func observeAttestedGoalForTestV0(t *testing.T, spec GoalWorkSpecV0, store *attestationStoreForTestV0, attestor *attestorForTestV0) GoalWorkObserveResultV0 {
	t.Helper()
	stateStore := &attestationGoalStateStoreForTestV0{state: GoalWorkStateV0{
		RunRef: spec.RunRef, GoalRef: spec.GoalRef, ExternalGoalRef: spec.GoalRef, Status: GoalStatusRunningV0, Spec: spec,
		LaunchReceipt: GoalLaunchReceiptV0{GoalRef: spec.GoalRef, ExternalGoalRef: spec.GoalRef, Status: GoalStatusRunningV0},
	}}
	result, err := ObserveGoalWorkV0(context.Background(), GoalWorkObserveRequestV0{RunRef: spec.RunRef}, GoalWorkLifecyclePortsV0{
		Observer:                     attestationObserverForTestV0{result: completedAttestedGoalResultForTestV0(spec)},
		RequiredTestAttestor:         attestor,
		RequiredTestAttestationStore: store,
		ClosureValidator:             IndependentGoalRequiredTestAttestationClosureValidatorV0{Reader: store},
		StateStore:                   stateStore,
	})
	if err != nil {
		t.Fatalf("ObserveGoalWorkV0: %v", err)
	}
	return result
}

type attestationObserverForTestV0 struct{ result GoalWorkResultV0 }

func (observer attestationObserverForTestV0) ObserveGoalWorkV0(context.Context, GoalObservationRequestV0) (GoalWorkResultV0, error) {
	return observer.result, nil
}

type attestorForTestV0 struct {
	attestations []GoalRequiredTestAttestationV0
	calls        int
}

func (attestor *attestorForTestV0) AttestGoalRequiredTestsV0(context.Context, GoalRequiredTestAttestationRequestV0) ([]GoalRequiredTestAttestationV0, error) {
	attestor.calls++
	return append([]GoalRequiredTestAttestationV0(nil), attestor.attestations...), nil
}

type attestationReaderForTestV0 struct {
	items []GoalRequiredTestAttestationV0
}

func (reader attestationReaderForTestV0) ListGoalRequiredTestAttestationsV0(_ context.Context, query GoalRequiredTestAttestationQueryV0) ([]GoalRequiredTestAttestationV0, error) {
	return filterAttestationsForTestV0(reader.items, query), nil
}

type attestationStoreForTestV0 struct {
	items []GoalRequiredTestAttestationV0
}

func (store *attestationStoreForTestV0) SaveGoalRequiredTestAttestationV0(_ context.Context, item GoalRequiredTestAttestationV0) error {
	for _, old := range store.items {
		if old.AttestationRef == item.AttestationRef {
			return nil
		}
	}
	store.items = append(store.items, item)
	return nil
}
func (store *attestationStoreForTestV0) ListGoalRequiredTestAttestationsV0(_ context.Context, query GoalRequiredTestAttestationQueryV0) ([]GoalRequiredTestAttestationV0, error) {
	return filterAttestationsForTestV0(store.items, query), nil
}
func filterAttestationsForTestV0(items []GoalRequiredTestAttestationV0, query GoalRequiredTestAttestationQueryV0) []GoalRequiredTestAttestationV0 {
	out := []GoalRequiredTestAttestationV0{}
	for _, item := range items {
		if item.RunRef == query.RunRef && item.GoalRef == query.GoalRef && item.RevisionRef == query.RevisionRef {
			out = append(out, item)
		}
	}
	return out
}

type attestationGoalStateStoreForTestV0 struct{ state GoalWorkStateV0 }

func (store *attestationGoalStateStoreForTestV0) SaveGoalWorkStateV0(_ context.Context, state GoalWorkStateV0) error {
	store.state = state
	return nil
}
func (store *attestationGoalStateStoreForTestV0) LoadGoalWorkStateV0(context.Context, string) (GoalWorkStateV0, error) {
	return store.state, nil
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
