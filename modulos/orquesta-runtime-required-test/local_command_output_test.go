package orquestaruntimerequiredtest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestLocalCommandExecutorV0RedactaSalidaAntesDePersistir(t *testing.T) {
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "sensitive")

	result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin"))
	if err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
	content := outputArtifactForTestV0(t, outputDir, result.EvidenceRefs[0])
	for _, forbidden := range []string{"token-real", "/home/alberto/proyecto"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("artifact no redactado: %q", content)
		}
	}
	for _, want := range []string{"output_redacted=true", "redaction_applied=true", "<home-path-redacted>"} {
		if !strings.Contains(content, want) {
			t.Fatalf("artifact sin %q: %q", want, content)
		}
	}
}

func TestLocalCommandExecutorV0BloqueaSalidaNoRedactableSinPersistirCrudo(t *testing.T) {
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "binary")

	result, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0("orquesta-test-bin"))
	if err != nil {
		t.Fatalf("RunRequiredTestCommandV0: %v", err)
	}
	content := outputArtifactForTestV0(t, outputDir, result.EvidenceRefs[0])
	if result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0 ||
		!strings.Contains(content, "required_test_output_blocked_sensitive_unredactable") ||
		strings.Contains(content, "\xff") {
		t.Fatalf("result=%+v content=%q", result, content)
	}
}

func TestLocalCommandExecutorV0RetieneSoloArtefactosRecientes(t *testing.T) {
	outputDir := t.TempDir()
	executor := localCommandExecutorForTestV0(t, outputDir, "pass")
	executor.MaxArtifacts = 2
	commands := []string{
		"orquesta-test-bin one",
		"orquesta-test-bin two",
		"orquesta-test-bin three",
	}
	for _, command := range commands {
		if _, err := executor.RunRequiredTestCommandV0(context.Background(), commandRequestForTestV0(command)); err != nil {
			t.Fatalf("RunRequiredTestCommandV0 %q: %v", command, err)
		}
		time.Sleep(time.Millisecond)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("artefactos retenidos=%d, want 2", count)
	}
}
