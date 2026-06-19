package orquestaruntimerequiredtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

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

func tinyGoModuleWithoutTestsForRequiredTestV0(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/orquesta-required-test-no-tests\n\ngo 1.22\n",
		"calc.go": strings.Join([]string{
			"package calc",
			"",
			"func Add(a int, b int) int {",
			"\treturn a + b",
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
