package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

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
	if stack.Ports.GoalRequiredTestSpecBinder == nil || stack.Ports.GoalRequiredTestSnapshotObserver == nil ||
		stack.Ports.GoalRequiredTestAttestor == nil || stack.Ports.GoalRequiredTestIdentityVerifier == nil {
		t.Fatalf("attestation ports incomplete: %+v", stack.Ports)
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
