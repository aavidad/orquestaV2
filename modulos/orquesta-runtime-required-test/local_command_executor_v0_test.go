package orquestaruntimerequiredtest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const childModeEnvV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_CHILD"
const childInvocationMarkerEnvV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_MARK_INVOCATION"
const childParentEnvLeakMarkerV0 = "ORQUESTA_REQUIRED_TEST_RUNTIME_PARENT_LEAK"
const childInvocationMarkerFileV0 = "required-test-invocations.log"

func TestMain(m *testing.M) {
	switch os.Getenv(childModeEnvV0) {
	case "pass":
		recordRequiredTestChildInvocationV0()
		fmt.Fprintln(os.Stdout, "required test passed")
		os.Exit(0)
	case "fail":
		recordRequiredTestChildInvocationV0()
		fmt.Fprintln(os.Stderr, "required test failed")
		os.Exit(7)
	case "wait":
		recordRequiredTestChildInvocationV0()
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

func TestLocalCommandExecutorV0RechazaHomeExplicitoAunqueSeaRelativo(t *testing.T) {
	for _, item := range []string{
		"HOME=relative-home",
		"USERPROFILE=relative-profile",
		"ORQUESTA_CODEX_HOME=relative-home",
	} {
		t.Run(item, func(t *testing.T) {
			executor := localCommandExecutorForTestV0(t, t.TempDir(), "pass")
			executor.Env = append(executor.Env, item)

			if _, err := executor.RunRequiredTestCommandV0(
				context.Background(),
				commandRequestForTestV0("orquesta-test-bin"),
			); err == nil || !strings.Contains(err.Error(), "required_test_env_invalid") {
				t.Fatalf("err=%v, want env invalid", err)
			}
		})
	}
}

func TestLocalCommandExecutorV0NoHeredaEntornoPadre(t *testing.T) {
	envPath, err := exec.LookPath("env")
	if err != nil {
		t.Fatalf("env no encontrado: %v", err)
	}
	t.Setenv(childParentEnvLeakMarkerV0, "must-not-leak")
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "pass")
	executor.AllowedCommands = map[string]string{"env": envPath}
	executor.Env = nil

	result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("env"))
	if err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
	content := outputArtifactForTestV0(t, outputDir, result.EvidenceRefs[0])
	if strings.Contains(content, "parent-env-leak") || strings.Contains(content, "must-not-leak") {
		t.Fatalf("artifact leaked parent env: %q", content)
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

func TestRequiredTestRunnerV0ConLocalCommandExecutorEjecutaGoTestReal(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go no encontrado: %v", err)
	}
	projectDir := tinyGoModuleForRequiredTestV0(t)
	outputDir := t.TempDir()
	store := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := orquestacionnucleoapp.RequiredTestRunnerV0{
		Executor: LocalCommandExecutorV0{
			ProjectWorkDir: projectDir,
			OutputDir:      outputDir,
			AllowedCommands: map[string]string{
				"go": goPath,
			},
			Env: []string{
				"CGO_ENABLED=0",
				"GOCACHE=" + filepath.Join(t.TempDir(), "go-build-cache"),
				"GOPATH=" + filepath.Join(t.TempDir(), "go-path"),
				"GOMODCACHE=" + filepath.Join(t.TempDir(), "go-mod-cache"),
			},
			MaxOutputBytes: 64 * 1024,
		},
		EvidenceWriter: store,
	}

	request := requiredTestExecutionRequestForRuntimeTestV0()
	result, err := runner.RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0: %v", err)
	}
	if len(result.Issues) != 0 ||
		len(result.PassedEvidenceRefs) != 1 ||
		len(result.FailedEvidenceRefs) != 0 {
		t.Fatalf("result=%+v", result)
	}
	evidence, err := store.LoadRequiredTestEvidenceV0(context.Background(), request.RunRef, result.PassedEvidenceRefs)
	if err != nil {
		t.Fatalf("LoadRequiredTestEvidenceV0: %v", err)
	}
	if len(evidence) != 1 ||
		evidence[0].Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 ||
		evidence[0].TestCommand != "go test ./..." ||
		evidence[0].DeliveryRef != request.DeliveryRef ||
		evidence[0].AcceptedReviewRef != request.AcceptedReviewRef {
		t.Fatalf("evidence=%+v", evidence)
	}
	content := outputArtifactForTestV0(t, outputDir, requiredTestOutputRefForTestV0(t, evidence[0].EvidenceRefs))
	if !strings.Contains(content, "status=passed") ||
		!strings.Contains(content, "test_command=go test ./...") ||
		strings.Contains(content, projectDir) ||
		strings.Contains(content, outputDir) {
		t.Fatalf("artifact content=%q", content)
	}
}

func TestRequiredTestRunnerV0ConStoreReaderNoReejecutaExternoEnReplay(t *testing.T) {
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "pass")
	executor.Env = append(executor.Env, childInvocationMarkerEnvV0+"=1")
	store := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0()
	runner := orquestacionnucleoapp.RequiredTestRunnerV0{
		Executor:       executor,
		EvidenceWriter: store,
	}
	request := requiredTestExecutionRequestForRuntimeTestV0()
	request.TestCommands = []string{"orquesta-test-bin"}

	first, err := runner.RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0 first: %v", err)
	}
	second, err := runner.RunRequiredTestsV0(context.Background(), request)
	if err != nil {
		t.Fatalf("RunRequiredTestsV0 second: %v", err)
	}
	if strings.Join(first.EvidenceRefs, "\n") != strings.Join(second.EvidenceRefs, "\n") ||
		len(second.Issues) != 0 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	markerPath := filepath.Join(executor.ProjectWorkDir, childInvocationMarkerFileV0)
	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("leer marker de invocacion: %v", err)
	}
	if got := strings.Count(string(data), "run\n"); got != 1 {
		t.Fatalf("invocaciones externas=%d, want 1; marker=%q", got, string(data))
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

func tinyGoModuleForRequiredTestV0(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/orquesta-required-test-smoke\n\ngo 1.22\n",
		"calc.go": strings.Join([]string{
			"package calc",
			"",
			"func Add(a int, b int) int {",
			"\treturn a + b",
			"}",
			"",
		}, "\n"),
		"calc_test.go": strings.Join([]string{
			"package calc",
			"",
			"import \"testing\"",
			"",
			"func TestAdd(t *testing.T) {",
			"\tif Add(2, 3) != 5 {",
			"\t\tt.Fatalf(\"Add fallo\")",
			"\t}",
			"}",
			"",
		}, "\n"),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(projectDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return projectDir
}

func requiredTestExecutionRequestForRuntimeTestV0() orquestacionnucleoapp.RequiredTestExecutionRequestV0 {
	return orquestacionnucleoapp.RequiredTestExecutionRequestV0{
		RunRef:            "run-runtime-required-test-real-001",
		TaskRef:           "task-runtime-required-test-real-001",
		TestCommands:      []string{"go test ./..."},
		DeliveryRef:       "delivery-ref-runtime-required-test-real-001",
		ReviewRequestID:   "review-request-ref-runtime-required-test-real-001",
		ReviewResultRef:   "review-result-ref-runtime-required-test-real-001",
		AcceptedReviewRef: "accepted-review-ref-runtime-required-test-real-001",
		OccurredAt:        "2026-05-22T10:00:00Z",
		CorrelationID:     "correlation-runtime-required-test-real-001",
		EvidenceRefs:      []string{"review-evidence-ref-runtime-required-test-real-001"},
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

func requiredTestOutputRefForTestV0(t *testing.T, refs []string) string {
	t.Helper()
	const prefix = "required-test-output-v0/"
	for _, ref := range refs {
		if strings.HasPrefix(ref, prefix) {
			return ref
		}
	}
	t.Fatalf("sin ref de salida en evidence_refs=%v", refs)
	return ""
}

func recordRequiredTestChildInvocationV0() {
	if os.Getenv(childInvocationMarkerEnvV0) == "" {
		return
	}
	file, err := os.OpenFile(childInvocationMarkerFileV0, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "required test marker failed: %v\n", err)
		os.Exit(98)
	}
	defer file.Close()
	if _, err := file.WriteString("run\n"); err != nil {
		fmt.Fprintf(os.Stderr, "required test marker failed: %v\n", err)
		os.Exit(98)
	}
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
