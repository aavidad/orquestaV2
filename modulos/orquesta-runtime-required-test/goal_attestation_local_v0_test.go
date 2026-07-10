package orquestaruntimerequiredtest

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestLocalGoalRequiredTestAttestationAdapterV0RealCheckoutAndIsolatedProcess(t *testing.T) {
	project, runtimeRoot, gitPath := localGoalAttestationGitRepoForTestV0(t)
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	adapter := localGoalAttestationAdapterForTestV0(t, project, runtimeRoot, gitPath, map[string]string{"go": goPath})
	spec := localGoalAttestationSpecForTestV0("go test -count=1 ./...")
	bound, err := adapter.BindGoalRequiredTestSpecV0(context.Background(), spec)
	if err != nil || bound.ImplementerCredentialRef != "credential-ref-implementer-local-001" {
		t.Fatalf("bound=%+v err=%v", bound, err)
	}
	snapshot, err := adapter.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: bound.RunRef, GoalRef: bound.GoalRef, WriteSet: bound.WriteSet, WriteSetSHA256: bound.WriteSetSHA256,
	})
	if err != nil || len(snapshot.Hashes) != 1 || snapshot.RevisionRef == "" || strings.Contains(snapshot.CheckoutRef, project) {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	request := orquestagoal.GoalRequiredTestAttestationRequestV0{
		RunRef: bound.RunRef, GoalRef: bound.GoalRef,
		ImplementerAgentRef: bound.ImplementerAgentRef, ImplementerCredentialRef: bound.ImplementerCredentialRef,
		AttestorTrustPolicyRef: bound.ClosurePolicy.RequiredAttestorTrustPolicyRef,
		FinalSnapshot:          snapshot, RequiredTests: bound.RequiredTests,
	}
	receipts, err := adapter.AttestGoalRequiredTestsV0(context.Background(), request)
	if err != nil || len(receipts) != 1 || receipts[0].Status != orquestagoal.GoalRequiredTestAttestationStatusPassedV0 ||
		!reflect.DeepEqual(receipts[0].HashesBefore, receipts[0].HashesAfter) {
		t.Fatalf("receipts=%+v err=%v", receipts, err)
	}
	if len(receipts[0].EvidenceRefs) < 3 {
		t.Fatalf("evidencia durable incompleta: %+v", receipts[0].EvidenceRefs)
	}
	verification, err := adapter.VerifyGoalRequiredTestIdentityV0(context.Background(), orquestagoal.GoalRequiredTestIdentityVerificationRequestV0{
		AttestationRef: receipts[0].AttestationRef, TestRef: receipts[0].TestRef,
		ImplementerAgentRef: bound.ImplementerAgentRef, ImplementerCredentialRef: bound.ImplementerCredentialRef,
		AttestorAgentRef: receipts[0].AttestorAgentRef, AttestorCredentialRef: receipts[0].AttestorCredentialRef,
		RequiredTrustPolicyRef: bound.ClosurePolicy.RequiredAttestorTrustPolicyRef,
	})
	if err != nil || !verification.Verified || !verification.Independent || len(verification.EvidenceRefs) != 1 {
		t.Fatalf("verification=%+v err=%v", verification, err)
	}
	if entries, err := os.ReadDir(filepath.Join(runtimeRoot, "evidence", "commands")); err != nil || len(entries) != 1 {
		t.Fatalf("command evidence entries=%d err=%v", len(entries), err)
	}
}

func TestLocalGoalRequiredTestAttestationAdapterV0MutationDuringTestProducesFailedReceipt(t *testing.T) {
	project, runtimeRoot, gitPath := localGoalAttestationGitRepoForTestV0(t)
	truncatePath, err := exec.LookPath("truncate")
	if err != nil {
		t.Skip("truncate unavailable")
	}
	adapter := localGoalAttestationAdapterForTestV0(t, project, runtimeRoot, gitPath, map[string]string{"truncate": truncatePath})
	spec := localGoalAttestationSpecForTestV0("truncate -s 0 artifact.txt")
	bound, err := adapter.BindGoalRequiredTestSpecV0(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := adapter.CaptureGoalRequiredTestFinalSnapshotV0(context.Background(), orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: bound.RunRef, GoalRef: bound.GoalRef, WriteSet: bound.WriteSet, WriteSetSHA256: bound.WriteSetSHA256,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipts, err := adapter.AttestGoalRequiredTestsV0(context.Background(), orquestagoal.GoalRequiredTestAttestationRequestV0{
		RunRef: bound.RunRef, GoalRef: bound.GoalRef,
		ImplementerAgentRef: bound.ImplementerAgentRef, ImplementerCredentialRef: bound.ImplementerCredentialRef,
		AttestorTrustPolicyRef: bound.ClosurePolicy.RequiredAttestorTrustPolicyRef,
		FinalSnapshot:          snapshot, RequiredTests: bound.RequiredTests,
	})
	if err != nil || len(receipts) != 1 || receipts[0].Status != orquestagoal.GoalRequiredTestAttestationStatusFailedV0 ||
		reflect.DeepEqual(receipts[0].HashesBefore, receipts[0].HashesAfter) {
		t.Fatalf("receipts=%+v err=%v", receipts, err)
	}
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationV0(receipts[0]); len(issues) > 0 {
		t.Fatalf("failed mutation receipt must remain durable evidence: %+v", issues)
	}
}

func TestLocalGoalRequiredTestAttestationAdapterV0RejectsNonIndependentPolicy(t *testing.T) {
	project, runtimeRoot, gitPath := localGoalAttestationGitRepoForTestV0(t)
	testPath, err := exec.LookPath("test")
	if err != nil {
		t.Fatal(err)
	}
	config := localGoalAttestationConfigForTestV0(project, runtimeRoot, gitPath, map[string]string{"test": testPath})
	config.Identity.AttestorPrincipalRef = config.Identity.ImplementerPrincipalRef
	if _, err := NewLocalGoalRequiredTestAttestationAdapterV0(config); err == nil {
		t.Fatal("trusted policy with one principal must fail")
	}
}

func localGoalAttestationGitRepoForTestV0(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "project")
	runtimeRoot := filepath.Join(root, "runtime")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "--quiet", project},
		{"-C", project, "config", "user.name", "Orquesta Test"},
		{"-C", project, "config", "user.email", "orquesta-test@example.invalid"},
	} {
		if output, err := exec.Command(gitPath, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	files := map[string]string{
		"go.mod":       "module example.com/orquesta-goal-attestation-fixture\n\ngo 1.22\n",
		"artifact.txt": "final artifact\n",
		"artifact_test.go": strings.Join([]string{
			"package fixture",
			"",
			"import (",
			"\t\"os\"",
			"\t\"testing\"",
			")",
			"",
			"func TestArtifactIsFinal(t *testing.T) {",
			"\tdata, err := os.ReadFile(\"artifact.txt\")",
			"\tif err != nil || string(data) != \"final artifact\\n\" {",
			"\t\tt.Fatalf(\"artifact=%q err=%v\", data, err)",
			"\t}",
			"}",
			"",
		}, "\n"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	for _, args := range [][]string{{"-C", project, "add", "go.mod", "artifact.txt", "artifact_test.go"}, {"-C", project, "commit", "--quiet", "-m", "fixture"}} {
		if output, err := exec.Command(gitPath, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	return project, runtimeRoot, gitPath
}

func localGoalAttestationAdapterForTestV0(t *testing.T, project, runtimeRoot, gitPath string, commands map[string]string) *LocalGoalRequiredTestAttestationAdapterV0 {
	t.Helper()
	adapter, err := NewLocalGoalRequiredTestAttestationAdapterV0(localGoalAttestationConfigForTestV0(project, runtimeRoot, gitPath, commands))
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func localGoalAttestationConfigForTestV0(project, runtimeRoot, gitPath string, commands map[string]string) LocalGoalRequiredTestAttestationConfigV0 {
	return LocalGoalRequiredTestAttestationConfigV0{
		ProjectWorkDir: project, RuntimeRoot: runtimeRoot, GitCommandPath: gitPath,
		AllowedCommands: commands, MaxRuntime: 10 * time.Second, MaxOutputBytes: 64 * 1024, MaxArtifacts: 20,
		Identity: LocalTrustedGoalRequiredTestIdentityPolicyV0{
			TrustPolicyRef: "policy-ref-local-attestation-001", PolicyEvidenceRef: "evidence-ref-owner-only-policy-001",
			ImplementerAgentRef: "agent-ref-implementer-local-001", ImplementerCredentialRef: "credential-ref-implementer-local-001",
			ImplementerPrincipalRef: "principal-ref-implementer-local-001",
			AttestorAgentRef:        "agent-ref-attestor-local-001", AttestorCredentialRef: "credential-ref-attestor-local-001",
			AttestorPrincipalRef: "principal-ref-attestor-local-001",
		},
	}
}

func localGoalAttestationSpecForTestV0(command string) orquestagoal.GoalWorkSpecV0 {
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "artifact.txt"}}
	return orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		RunRef: "run-ref-local-attestation-001", GoalRef: "goal-ref-local-attestation-001",
		Objective: "Verify final artifact.", DirectorKind: orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: writeSet, WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet),
		RequiredTests: []orquestagoal.GoalRequiredTestV0{orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
			TestRef: "test-ref-local-attestation-001", CommandRef: "command-ref-local-attestation-001", Command: command,
		})},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests: true, RequireIndependentRequiredTestAttestation: true,
		},
	})
}
