package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestGoalRequiredTestAttestationConfigV0DisabledLeavesPortsNil(t *testing.T) {
	withoutGoalRequiredTestAttestationEnvV0(t)
	adapter, err := goalRequiredTestAttestationAdapterFromConfigV0(orquestaserver.ConfigV0{ProjectWorkDir: t.TempDir()}, serverProjectConfigFileV0{})
	if err != nil || adapter != nil {
		t.Fatalf("adapter=%T err=%v", adapter, err)
	}
}

func TestGoalRequiredTestAttestationConfigV0PartialJSONFailsStartup(t *testing.T) {
	projectDir := t.TempDir()
	path := filepath.Join(t.TempDir(), "attestation.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":"orquesta_goal_required_test_attestation_config.v0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envGoalRequiredTestAttestationConfigFileV0, path)
	if _, err := goalRequiredTestAttestationAdapterFromConfigV0(orquestaserver.ConfigV0{ProjectWorkDir: projectDir}, serverProjectConfigFileV0{}); err == nil {
		t.Fatal("partial attestation config must fail startup")
	}
}

func TestGoalRequiredTestAttestationConfigV0FailsStartupOnRedPreflight(t *testing.T) {
	projectDir := goalRequiredTestAttestationGitFixtureV0(t)
	path := writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, projectDir, filepath.Join(t.TempDir(), "attestation-runtime"))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document serverGoalRequiredTestAttestationConfigFileV0
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document.PreflightCommands = []string{"test -f toolchain-ready"}
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envGoalRequiredTestAttestationConfigFileV0, path)
	if _, err := goalRequiredTestAttestationAdapterFromConfigV0(orquestaserver.ConfigV0{ProjectWorkDir: projectDir}, serverProjectConfigFileV0{}); err == nil {
		t.Fatal("red attestation preflight must block startup")
	}
}

func TestBuildStackFromEnvV0WiresCompleteGoalRequiredTestAttestationConfig(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "codex-runtime")
	attestationRuntime := filepath.Join(t.TempDir(), "attestation-runtime")
	configPath := writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, projectDir, attestationRuntime)
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envGoalRequiredTestAttestationConfigFileV0, configPath)
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv(envOPESBaseURLLegacyV0, "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	t.Cleanup(func() {
		_ = serverRequiredTestResourceShutdownHookFromStackV0(stack).ShutdownV0(context.Background())
	})
	if stack.Ports.GoalRequiredTestSpecBinder == nil || stack.Ports.GoalRequiredTestSnapshotObserver == nil ||
		stack.Ports.GoalRequiredTestAttestor == nil || stack.Ports.GoalRequiredTestIdentityVerifier == nil {
		t.Fatalf("attestation ports incomplete: %+v", stack.Ports)
	}
}

func TestBuildStackFromEnvV0GoalRequiredTestAttestationExecutesIndependentReceipt(t *testing.T) {
	projectDir := goalRequiredTestAttestationGitFixtureV0(t)
	stateDir := t.TempDir()
	runtimeDir := filepath.Join(t.TempDir(), "codex-runtime")
	attestationRuntime := filepath.Join(t.TempDir(), "attestation-runtime")
	configPath := writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, projectDir, attestationRuntime)
	info, err := os.Stat(configPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("owner-only config info=%v err=%v", info, err)
	}
	t.Setenv(envCodexProjectWorkDirV0, projectDir)
	t.Setenv(envServerStateDirV0, stateDir)
	t.Setenv(envCodexRuntimeWorkDirV0, runtimeDir)
	t.Setenv(envCodexCommandV0, filepath.Join(projectDir, "codex-bin"))
	t.Setenv(envGoalRequiredTestAttestationConfigFileV0, configPath)
	t.Setenv(envOPESBaseURLV0, "")
	t.Setenv(envOPESBaseURLLegacyV0, "")
	t.Setenv(envDomainWorkFileEnabledV0, "")
	t.Setenv(envDomainWorkFileDirV0, "")
	config, err := serverConfigFromEnvV0()
	if err != nil {
		t.Fatal(err)
	}
	stack, err := buildStackFromEnvV0(config)
	if err != nil {
		t.Fatalf("buildStackFromEnvV0: %v", err)
	}
	t.Cleanup(func() {
		_ = serverRequiredTestResourceShutdownHookFromStackV0(stack).ShutdownV0(context.Background())
	})
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "artifact.txt"}}
	spec := orquestagoal.NormalizeGoalWorkSpecV0(orquestagoal.GoalWorkSpecV0{
		RunRef: "run-ref-server-attestation-001", GoalRef: "goal-ref-server-attestation-001",
		Objective: "Verify the final artifact.", DirectorKind: orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: writeSet, WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet),
		RequiredTests: []orquestagoal.GoalRequiredTestV0{orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
			TestRef: "test-ref-server-attestation-001", CommandRef: "command-ref-server-attestation-001", Command: "test -f artifact.txt",
		})},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{RequireRequiredTests: true, RequireIndependentRequiredTestAttestation: true},
	})
	ctx := context.Background()
	bound, err := stack.Ports.GoalRequiredTestSpecBinder.BindGoalRequiredTestSpecV0(ctx, spec)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	snapshot, err := stack.Ports.GoalRequiredTestSnapshotObserver.CaptureGoalRequiredTestFinalSnapshotV0(ctx, orquestagoal.GoalRequiredTestFinalSnapshotRequestV0{
		RunRef: bound.RunRef, GoalRef: bound.GoalRef, WriteSet: bound.WriteSet, WriteSetSHA256: bound.WriteSetSHA256,
	})
	if err != nil || snapshot.RevisionRef == "" || len(snapshot.Hashes) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	receipts, err := stack.Ports.GoalRequiredTestAttestor.AttestGoalRequiredTestsV0(ctx, orquestagoal.GoalRequiredTestAttestationRequestV0{
		RunRef: bound.RunRef, GoalRef: bound.GoalRef, ImplementerAgentRef: bound.ImplementerAgentRef,
		ImplementerCredentialRef: bound.ImplementerCredentialRef, AttestorTrustPolicyRef: bound.ClosurePolicy.RequiredAttestorTrustPolicyRef,
		FinalSnapshot: snapshot, RequiredTests: bound.RequiredTests,
	})
	if err != nil || len(receipts) != 1 || receipts[0].Status != orquestagoal.GoalRequiredTestAttestationStatusPassedV0 || receipts[0].ExitCode != 0 {
		t.Fatalf("receipts=%+v err=%v", receipts, err)
	}
	verification, err := stack.Ports.GoalRequiredTestIdentityVerifier.VerifyGoalRequiredTestIdentityV0(ctx, orquestagoal.GoalRequiredTestIdentityVerificationRequestV0{
		AttestationRef: receipts[0].AttestationRef, TestRef: receipts[0].TestRef, ImplementerAgentRef: bound.ImplementerAgentRef,
		ImplementerCredentialRef: bound.ImplementerCredentialRef, AttestorAgentRef: receipts[0].AttestorAgentRef,
		AttestorCredentialRef: receipts[0].AttestorCredentialRef, RequiredTrustPolicyRef: bound.ClosurePolicy.RequiredAttestorTrustPolicyRef,
	})
	if err != nil || !verification.Verified || !verification.Independent || verification.ImplementerPrincipalRef == verification.AttestorPrincipalRef {
		t.Fatalf("verification=%+v err=%v", verification, err)
	}
}

func TestGoalRequiredTestAttestationConfigV0ProjectJSONPathIsCanonicalFallback(t *testing.T) {
	withoutGoalRequiredTestAttestationEnvV0(t)
	projectDir := t.TempDir()
	path := writeCompleteGoalRequiredTestAttestationConfigForTestV0(t, projectDir, filepath.Join(t.TempDir(), "attestation-runtime"))
	relative, err := filepath.Rel(projectDir, path)
	if err != nil {
		t.Fatal(err)
	}
	projectConfig := serverProjectConfigFileV0{Autoprogramming: serverProjectConfigAutoprogrammingV0{RequiredTestAttestationConfigFile: &relative}}
	adapter, err := goalRequiredTestAttestationAdapterFromConfigV0(orquestaserver.ConfigV0{ProjectWorkDir: projectDir}, projectConfig)
	if err != nil || adapter == nil {
		t.Fatalf("adapter=%T err=%v", adapter, err)
	}
}

func writeCompleteGoalRequiredTestAttestationConfigForTestV0(t *testing.T, projectDir, runtimeRoot string) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	testPath, err := exec.LookPath("test")
	if err != nil {
		t.Fatal(err)
	}
	document := serverGoalRequiredTestAttestationConfigFileV0{
		SchemaVersion:  goalRequiredTestAttestationConfigSchemaV0,
		ProjectWorkDir: projectDir, RuntimeRoot: runtimeRoot, GitCommandPath: gitPath,
		AllowedCommands:   map[string]string{"test": testPath},
		PreflightCommands: []string{"test -d ."},
		MaxRuntimeSeconds: 15, MaxOutputBytes: 64 * 1024, MaxArtifacts: 20,
		Identity: serverGoalRequiredTestAttestationIdentityFileV0{
			TrustPolicyRef:      "policy-ref-server-attestation-001",
			ImplementerAgentRef: "agent-ref-server-implementer-001", ImplementerCredentialRef: "credential-ref-server-implementer-001",
			ImplementerPrincipalRef: "principal-ref-server-implementer-001",
			AttestorAgentRef:        "agent-ref-server-attestor-001", AttestorCredentialRef: "credential-ref-server-attestor-001",
			AttestorPrincipalRef: "principal-ref-server-attestor-001",
		},
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projectDir, "attestation-config.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func goalRequiredTestAttestationGitFixtureV0(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "--quiet", projectDir},
		{"-C", projectDir, "config", "user.name", "Orquesta Test"},
		{"-C", projectDir, "config", "user.email", "orquesta-test@example.invalid"},
	} {
		if output, err := exec.Command(gitPath, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	if err := os.WriteFile(filepath.Join(projectDir, "artifact.txt"), []byte("final artifact\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-C", projectDir, "add", "artifact.txt"}, {"-C", projectDir, "commit", "--quiet", "-m", "fixture"}} {
		if output, err := exec.Command(gitPath, args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	return projectDir
}

func withoutGoalRequiredTestAttestationEnvV0(t *testing.T) {
	t.Helper()
	value, present := os.LookupEnv(envGoalRequiredTestAttestationConfigFileV0)
	if err := os.Unsetenv(envGoalRequiredTestAttestationConfigFileV0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(envGoalRequiredTestAttestationConfigFileV0, value)
		} else {
			_ = os.Unsetenv(envGoalRequiredTestAttestationConfigFileV0)
		}
	})
}
