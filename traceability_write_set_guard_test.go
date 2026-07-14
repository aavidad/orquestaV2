package orquesta_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTraceabilityRebuildWriteSetGuardCoversCurrentSurfaces(t *testing.T) {
	script, err := os.ReadFile("scripts/check_rebuild_write_set.sh")
	if err != nil {
		t.Fatal(err)
	}
	repository := t.TempDir()
	writeTraceGuardFile(t, repository, "scripts/check_rebuild_write_set.sh", script, 0o700)
	writeTraceGuardFile(t, repository, "product/seed.json", []byte("{}\n"), 0o600)
	traceRunGuardCommand(t, repository, "git", "init", "-q")
	traceRunGuardCommand(t, repository, "git", "add", ".")
	traceRunGuardCommand(t, repository, "git", "-c", "user.name=Orquesta Test", "-c", "user.email=orquesta@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "base")
	base := strings.TrimSpace(traceRunGuardCommand(t, repository, "git", "rev-parse", "HEAD"))

	allowed := []string{
		"acceptance/README.md",
		"acceptance/fixtures/v03_canonical_ledgers.json",
		"docs/reconstruccion/estado_y_handoff_rebuild.md",
		"product/traceability/task_entries.jsonl",
		"traceability_task_candidate_review_test.go",
	}
	for _, path := range allowed {
		writeTraceGuardFile(t, repository, path, []byte("fixture\n"), 0o600)
	}
	command := exec.Command("bash", "scripts/check_rebuild_write_set.sh", base)
	command.Dir = repository
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("current rebuild surfaces rejected: %v\n%s", err, output)
	}

	writeTraceGuardFile(t, repository, "modulos/legacy.go", []byte("package legacy\n"), 0o600)
	command = exec.Command("bash", "scripts/check_rebuild_write_set.sh", base)
	command.Dir = repository
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "rebuild_write_set_violation") || !strings.Contains(string(output), "modulos/legacy.go") {
		t.Fatalf("frozen legacy surface not rejected: err=%v output=%s", err, output)
	}
}

func writeTraceGuardFile(t *testing.T, root, relative string, content []byte, mode os.FileMode) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, content, mode); err != nil {
		t.Fatal(err)
	}
}

func traceRunGuardCommand(t *testing.T, directory, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = directory
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return string(output)
}
