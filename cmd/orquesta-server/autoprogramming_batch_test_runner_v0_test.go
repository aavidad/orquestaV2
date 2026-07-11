package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestAutoprogrammingBatchTestRunnerV0EjecutaEnWorkdirCanonicoYPersisteReceiptV0(t *testing.T) {
	projectDir := serverAutoprogrammingBatchTinyGoModuleV0(t)
	runner := serverAutoprogrammingBatchTestRunnerForTestV0(t, projectDir)
	request := serverAutoprogrammingBatchTestRequestV0(projectDir)

	result, err := runner.RunAutoprogrammingBatchTestV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunAutoprogrammingBatchTestV0: %v", err)
	}
	if result.Status != orquestaautoprogramming.AutoprogrammingBatchTestReceiptPassedV0 ||
		!strings.HasPrefix(result.ReceiptRef, "autoprogramming-batch-test-receipt-v0/") ||
		len(result.EvidenceRefs) < 2 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(runner.receiptPathV0(result.ReceiptRef)); err != nil {
		t.Fatalf("receipt durable: %v", err)
	}
	if !serverAutoprogrammingBatchHasRefV0(result.EvidenceRefs, "required-test-output-v0/") {
		t.Fatalf("evidencia de executor ausente: %v", result.EvidenceRefs)
	}
}

func TestAutoprogrammingBatchTestRunnerV0ReconcilesOnlyDurableReceiptV0(t *testing.T) {
	projectDir := serverAutoprogrammingBatchTinyGoModuleV0(t)
	runner := serverAutoprogrammingBatchTestRunnerForTestV0(t, projectDir)
	request := serverAutoprogrammingBatchTestRequestV0(projectDir)
	first, err := runner.RunAutoprogrammingBatchTestV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunAutoprogrammingBatchTestV0: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module broken\n\ngo invalid\n"), 0o600); err != nil {
		t.Fatalf("corromper modulo: %v", err)
	}
	retried, err := runner.RunAutoprogrammingBatchTestV0(context.Background(), request)
	if err != nil || retried.ReceiptRef != first.ReceiptRef || retried.Status != first.Status {
		t.Fatalf("retry result=%+v err=%v first=%+v", retried, err, first)
	}
	replayed, found, err := runner.ReconcileAutoprogrammingBatchTestClaimV0(context.Background(), request)
	if err != nil || !found || replayed.ReceiptRef != first.ReceiptRef || replayed.Status != first.Status ||
		strings.Join(replayed.EvidenceRefs, "\x00") != strings.Join(first.EvidenceRefs, "\x00") {
		t.Fatalf("reconcile result=%+v found=%v err=%v first=%+v", replayed, found, err, first)
	}

	missing := request
	missing.ClaimRef = "claim-ref-autoprogramming-batch-missing-001"
	result, found, err := runner.ReconcileAutoprogrammingBatchTestClaimV0(context.Background(), missing)
	if err != nil || found || result.ReceiptRef != "" {
		t.Fatalf("receipt inexistente result=%+v found=%v err=%v", result, found, err)
	}

	divergentGeneration := request
	divergentGeneration.GateGeneration++
	result, found, err = runner.ReconcileAutoprogrammingBatchTestClaimV0(context.Background(), divergentGeneration)
	if err != nil || found || result.ReceiptRef != "" {
		t.Fatalf("replay con generacion divergente result=%+v found=%v err=%v", result, found, err)
	}
}

func TestBuildStackFromProjectConfigV0CableaBatchStoreRunnerYIntegradorV0(t *testing.T) {
	canonicalProjectDir := serverAutoprogrammingBatchTinyGoModuleV0(t)
	externalProjectDir := t.TempDir()
	stateDir := t.TempDir()
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go no encontrado: %v", err)
	}
	configPath := filepath.Join(canonicalProjectDir, serverProjectConfigFileNameV0)
	config := `{
		"schema_version":"orquesta_config.v0",
		"required_test_runner":{"enabled":true,"go_command":` + serverAutoprogrammingBatchJSONV0(goCommand) + `},
		"autoprogramming":{"promotion":{"enabled":true,"archive_dir":` + serverAutoprogrammingBatchJSONV0(filepath.Join(stateDir, "archive")) + `}}
	}`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	stack, err := buildStackFromProjectConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:                    externalProjectDir,
		IdleSelfImprovementProjectWorkDir: canonicalProjectDir,
		StateDir:                          stateDir,
		ProjectConfigFilePath:             configPath,
		RuntimeWorkDir:                    filepath.Join(stateDir, "runtime"),
	}, serverCodexGoalBackendV0{}, serverProjectConfigFileV0{})
	if err != nil {
		t.Fatalf("buildStackFromProjectConfigV0: %v", err)
	}
	if stack.Stores.AutoprogrammingBatchStore == nil || stack.AutoprogrammingPromotion.BatchTestRunner == nil ||
		stack.AutoprogrammingPromotion.GoalWorkspaceIntegration == nil || stack.AutoprogrammingPromotion.BatchPromotionFinalizer == nil ||
		stack.AutoprogrammingPromotion.BatchPromotionReconciler == nil || stack.AutoprogrammingPromotion.BatchPromotionReceiptDir != stack.AutoprogrammingPromotion.BatchIntegrationReceiptDir {
		t.Fatalf("batch wiring store=%T runner=%T integration=%T finalizer=%T reconciler=%T promotion_receipts=%q integration_receipts=%q", stack.Stores.AutoprogrammingBatchStore, stack.AutoprogrammingPromotion.BatchTestRunner, stack.AutoprogrammingPromotion.GoalWorkspaceIntegration, stack.AutoprogrammingPromotion.BatchPromotionFinalizer, stack.AutoprogrammingPromotion.BatchPromotionReconciler, stack.AutoprogrammingPromotion.BatchPromotionReceiptDir, stack.AutoprogrammingPromotion.BatchIntegrationReceiptDir)
	}
	if _, ok := stack.AutoprogrammingPromotion.BatchTestRunner.(*serverAutoprogrammingBatchTestRunnerV0); !ok {
		t.Fatalf("runner=%T", stack.AutoprogrammingPromotion.BatchTestRunner)
	}
	runner := stack.AutoprogrammingPromotion.BatchTestRunner.(*serverAutoprogrammingBatchTestRunnerV0)
	finalizer, ok := stack.AutoprogrammingPromotion.BatchPromotionFinalizer.(*serverAutoprogrammingBatchPromotionFinalizerV0)
	if !ok {
		t.Fatalf("finalizer=%T", stack.AutoprogrammingPromotion.BatchPromotionFinalizer)
	}
	if runner.Executor.ProjectWorkDir != canonicalProjectDir || finalizer.CanonicalWorkDir != canonicalProjectDir {
		t.Fatalf("canonical self-programming runner=%q finalizer=%T/%q external=%q", runner.Executor.ProjectWorkDir, stack.AutoprogrammingPromotion.BatchPromotionFinalizer, finalizer.CanonicalWorkDir, externalProjectDir)
	}
	if _, err := finalizer.FinalizeAutoprogrammingBatchPromotionV0(context.Background(), orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0{
		BatchRef: "batch-ref-idle-workdir-negative-001", GateGeneration: 1, ClaimRef: "claim-ref-idle-workdir-negative-001",
		IntegratedRevision: "revision-idle-workdir-negative-001", CanonicalWorkDir: externalProjectDir,
		ReceiptDir: finalizer.ReceiptDir,
	}); err == nil || err.Error() != "autoprogramming_batch_promotion_canonical_work_dir_mismatch" {
		t.Fatalf("external ProjectWorkDir aceptado por finalizer: %v", err)
	}
}

func serverAutoprogrammingBatchTestRunnerForTestV0(t *testing.T, projectDir string) *serverAutoprogrammingBatchTestRunnerV0 {
	t.Helper()
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go no encontrado: %v", err)
	}
	configPath := filepath.Join(projectDir, serverProjectConfigFileNameV0)
	config := `{"schema_version":"orquesta_config.v0","required_test_runner":{"enabled":true,"go_command":` + serverAutoprogrammingBatchJSONV0(goCommand) + `,"output_dir":` + serverAutoprogrammingBatchJSONV0(filepath.Join(t.TempDir(), "required-test-output")) + `}}`
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	runner, err := autoprogrammingBatchTestRunnerFromConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:        projectDir,
		StateDir:              t.TempDir(),
		ProjectConfigFilePath: configPath,
	}, projectDir)
	if err != nil || runner == nil {
		t.Fatalf("autoprogrammingBatchTestRunnerFromConfigV0 runner=%T err=%v", runner, err)
	}
	return runner
}

func serverAutoprogrammingBatchTestRequestV0(projectDir string) orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0 {
	command := "go test ./..."
	sum := sha256.Sum256([]byte(command))
	test := orquestaautoprogramming.AutoprogrammingBatchTestV0{Command: command, SHA256: hex.EncodeToString(sum[:])}
	return orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0{
		BatchRef:         "batch-ref-autoprogramming-server-001",
		Revision:         "revision-autoprogramming-server-001",
		Test:             test,
		TestHash:         orquestaautoprogramming.AutoprogrammingBatchTestHashV0(test),
		ClaimRef:         "claim-ref-autoprogramming-server-001",
		GateGeneration:   1,
		CanonicalWorkDir: projectDir,
	}
}

func serverAutoprogrammingBatchTinyGoModuleV0(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":        "module example.com/orquesta-batch-runner\n\ngo 1.22\n",
		"batch.go":      "package batch\n\nfunc Add(left, right int) int { return left + right }\n",
		"batch_test.go": "package batch\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { if Add(2, 3) != 5 { t.Fatal(\"unexpected\") } }\n",
	} {
		if err := os.WriteFile(filepath.Join(projectDir, name), []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return projectDir
}

func serverAutoprogrammingBatchHasRefV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func serverAutoprogrammingBatchJSONV0(value string) string {
	return `"` + strings.ReplaceAll(value, `\`, `\\`) + `"`
}
