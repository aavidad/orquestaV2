package orquestastatefile

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
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

func goalRequiredTestAttestationForStoreTestV0() orquestagoal.GoalRequiredTestAttestationV0 {
	hash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	return orquestagoal.GoalRequiredTestAttestationV0{
		AttestationRef: "attestation-ref-state-file-001", RunRef: "run-ref-state-file-001", GoalRef: "goal-ref-state-file-001", RevisionRef: "revision-ref-state-file-001", TestRef: "test-ref-state-file-001", CommandRef: "command-ref-state-file-001", CommandSHA256: hash, DefinitionSHA256: hash, Status: orquestagoal.GoalRequiredTestAttestationStatusPassedV0, ImplementerAgentRef: "agent-ref-implementer-state-file", AttestorAgentRef: "agent-ref-attestor-state-file", AttestorCredentialRef: "credential-ref-attestor-state-file", StartedAt: "2026-07-10T10:00:00Z", FinishedAt: "2026-07-10T10:00:01Z", IsolatedEnvironmentRef: "environment-ref-state-file-001", HashesBefore: []orquestagoal.GoalAttestedHashV0{{Ref: "hash-ref-before-state-file", SHA256: hash}}, HashesAfter: []orquestagoal.GoalAttestedHashV0{{Ref: "hash-ref-after-state-file", SHA256: hash}}, EvidenceRefs: []string{"evidence-ref-state-file-attestation"},
	}
}
