package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const flakyHarnessChildEnvV0 = "ORQUESTA_CMD_SERVER_FLAKY_HARNESS_CHILD"
const flakyHarnessAttemptTimeoutV0 = 90 * time.Second
const flakyHarnessChildTestTimeoutV0 = 60 * time.Second
const flakyHarnessAttemptsV0 = 3

func TestFlakyHarnessV0DocumentaCasoT10V0(t *testing.T) {
	runbook := filepath.Join(projectRootForFlakyHarnessV0(t), "docs", "runbooks", "flaky_tests_observability_2026-05-24.md")
	data, err := os.ReadFile(runbook)
	if err != nil {
		t.Fatalf("leer runbook T10: %v", err)
	}
	text := string(data)
	for _, needle := range []string{
		"Fecha observada: 2026-05-23",
		"go test -count=1 ./...",
		"TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje",
		"stdout de un agente fake no contenia el prompt esperado",
		"go test -count=1 ./cmd/orquesta-server -run TestFlakyHarnessV0RepiteCasoDirectorRecursiveFakeRuntimeV0",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("runbook T10 no contiene %q", needle)
		}
	}
}

func TestFlakyHarnessV0RepiteCasoDirectorRecursiveFakeRuntimeV0(t *testing.T) {
	if strings.TrimSpace(os.Getenv(flakyHarnessChildEnvV0)) == "1" {
		t.Skip("child run")
	}
	childGoCache := strings.TrimSpace(os.Getenv("GOCACHE"))
	if childGoCache == "" {
		childGoCache = filepath.Join(os.TempDir(), "orquesta-go-build-cache")
	}
	if err := os.MkdirAll(childGoCache, 0o700); err != nil {
		t.Fatalf("preparar GOCACHE para child: %v", err)
	}
	for attempt := 1; attempt <= flakyHarnessAttemptsV0; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), flakyHarnessAttemptTimeoutV0)
		cmd := exec.CommandContext(
			ctx,
			"go",
			"test",
			"-count=1",
			"./cmd/orquesta-server",
			"-run",
			"^TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje$",
			"-timeout",
			flakyHarnessChildTestTimeoutV0.String(),
		)
		cmd.Dir = projectRootForFlakyHarnessV0(t)
		cmd.Env = append(os.Environ(), flakyHarnessChildEnvV0+"=1", "GOCACHE="+childGoCache)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			cancel()
			if ctx.Err() == context.DeadlineExceeded {
				t.Fatalf("attempt=%d timeout=%s stdout=%s stderr=%s",
					attempt,
					flakyHarnessAttemptTimeoutV0,
					goSmokeDiagnosticForTestV0(stdout.String()),
					goSmokeDiagnosticForTestV0(stderr.String()),
				)
			}
			t.Fatalf("attempt=%d err=%v stdout=%s stderr=%s",
				attempt,
				err,
				goSmokeDiagnosticForTestV0(stdout.String()),
				goSmokeDiagnosticForTestV0(stderr.String()),
			)
		}
		cancel()
	}
}

func projectRootForFlakyHarnessV0(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if strings.HasSuffix(filepath.ToSlash(wd), "/cmd/orquesta-server") {
		return filepath.Dir(filepath.Dir(wd))
	}
	return wd
}

func waitForCodexDirectorTestFileContainsV0(t *testing.T, path string, needles ...string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			last = string(data)
			for _, needle := range needles {
				if strings.Contains(last, needle) {
					return last
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("archivo sin contenido esperado: %s\n%s",
		goSmokeDiagnosticForTestV0(path),
		goSmokeDiagnosticForTestV0(last),
	)
	return ""
}
