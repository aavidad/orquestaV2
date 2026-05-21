package orquestaruntimerequiredtest

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const childModeEnvV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_CHILD"

func TestMain(m *testing.M) {
	switch os.Getenv(childModeEnvV0) {
	case "pass":
		fmt.Fprintln(os.Stdout, "required test passed")
		os.Exit(0)
	case "fail":
		fmt.Fprintln(os.Stderr, "required test failed")
		os.Exit(7)
	case "wait":
		requiredTestChildWaitV0()
	}
	os.Exit(m.Run())
}

func TestLocalCommandExecutorV0EjecutaProcesoRealYGuardaArtefacto(t *testing.T) {
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "pass")

	result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin"))
	if err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
	if result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 || len(result.EvidenceRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	content := outputArtifactForTestV0(t, outputDir, result.EvidenceRefs[0])
	if !strings.Contains(content, "required test passed") ||
		!strings.Contains(content, "status=passed") ||
		strings.Contains(content, outputDir) ||
		strings.Contains(content, executor.ProjectWorkDir) {
		t.Fatalf("artifact content=%q", content)
	}
}

func TestLocalCommandExecutorV0NonZeroEsEvidenciaFailed(t *testing.T) {
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "fail")

	result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin"))
	if err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
	if result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 || len(result.EvidenceRefs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	content := outputArtifactForTestV0(t, outputDir, result.EvidenceRefs[0])
	if !strings.Contains(content, "required test failed") || !strings.Contains(content, "status=failed") {
		t.Fatalf("artifact content=%q", content)
	}
}

func TestLocalCommandExecutorV0RechazaShellYSintaxisDeShell(t *testing.T) {
	executor := localCommandExecutorForTestV0(t, t.TempDir(), "pass")
	executor.AllowedCommands["sh"] = "/bin/sh"

	if _, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("sh -c echo")); err == nil ||
		!strings.Contains(err.Error(), "shell") {
		t.Fatalf("err=%v, want shell prohibited", err)
	}
	delete(executor.AllowedCommands, "sh")
	if _, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin; echo nope")); err == nil ||
		!strings.Contains(err.Error(), "shell_syntax") {
		t.Fatalf("err=%v, want shell syntax prohibited", err)
	}
}

func TestLocalCommandExecutorV0RequiereComandoPermitido(t *testing.T) {
	executor := localCommandExecutorForTestV0(t, t.TempDir(), "pass")
	if _, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("go test ./...")); err == nil ||
		!strings.Contains(err.Error(), "not_allowed") {
		t.Fatalf("err=%v, want command not allowed", err)
	}
}

func TestLocalCommandExecutorV0AceptaCacheGoExplicitaSinHome(t *testing.T) {
	executor := localCommandExecutorForTestV0(t, t.TempDir(), "pass")
	executor.Env = append(executor.Env, "GOCACHE="+filepath.Join(t.TempDir(), "go-build-cache"))

	if _, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin")); err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
}

func TestLocalCommandExecutorV0AceptaEntornoGoExplicitoSinHome(t *testing.T) {
	executor := localCommandExecutorForTestV0(t, t.TempDir(), "pass")
	executor.Env = append(
		executor.Env,
		"GOCACHE="+filepath.Join(t.TempDir(), "go-build-cache"),
		"GOPATH="+filepath.Join(t.TempDir(), "go-path"),
		"GOMODCACHE="+filepath.Join(t.TempDir(), "go-mod-cache"),
	)

	if _, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin")); err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
}

func TestLocalCommandExecutorV0RespetaContexto(t *testing.T) {
	executor := localCommandExecutorForTestV0(t, t.TempDir(), "wait")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, err := executor.RunRequiredTestCommandV0(ctx, commandRequestForTestV0("orquesta-test-bin")); err == nil {
		t.Fatal("err=nil, want context deadline")
	}
}

func localCommandExecutorForTestV0(t *testing.T, outputDir string, childMode string) LocalCommandExecutorV0 {
	t.Helper()
	if !filepath.IsAbs(os.Args[0]) {
		t.Fatalf("test binary path no es absoluto: %q", os.Args[0])
	}
	return LocalCommandExecutorV0{
		ProjectWorkDir: t.TempDir(),
		OutputDir:      outputDir,
		AllowedCommands: map[string]string{
			"orquesta-test-bin": os.Args[0],
		},
		Env:            []string{childModeEnvV0 + "=" + childMode},
		MaxOutputBytes: 4096,
	}
}

func commandRequestForTestV0(command string) orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0 {
	return orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0{
		RunRef:        "run-runtime-required-test-001",
		TaskRef:       "task-runtime-required-test-001",
		TestCommand:   command,
		CorrelationID: "correlation-runtime-required-test-001",
		EvidenceRefs:  []string{"review-evidence-ref-runtime-required-test-001"},
	}
}

func outputArtifactForTestV0(t *testing.T, outputDir string, evidenceRef string) string {
	t.Helper()
	const prefix = "required-test-output-v0/"
	if !strings.HasPrefix(evidenceRef, prefix) {
		t.Fatalf("evidence_ref=%q", evidenceRef)
	}
	data, err := os.ReadFile(filepath.Join(outputDir, strings.TrimPrefix(evidenceRef, prefix)))
	if err != nil {
		t.Fatalf("leer artifact: %v", err)
	}
	return string(data)
}

func requiredTestChildWaitV0() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	defer signal.Stop(signals)

	select {
	case <-signals:
		os.Exit(0)
	case <-time.After(30 * time.Second):
		os.Exit(3)
	}
}
