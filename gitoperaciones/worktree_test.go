package gitoperaciones

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalGitRepoPathRejectsNonRepo(t *testing.T) {
	tmp := t.TempDir()
	_, err := canonicalGitRepoPath(tmp)
	if err == nil {
		t.Fatal("deberia fallar para ruta no git")
	}
	if !strings.Contains(err.Error(), "ruta repo inválida") {
		t.Fatalf("error inesperado: %v", err)
	}
}

func TestCanonicalGitRepoPathResolvesTopLevel(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	nested := filepath.Join(repo, "cmd", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, string(out))
	}
	got, err := canonicalGitRepoPath(nested)
	if err != nil {
		t.Fatalf("canonicalGitRepoPath: %v", err)
	}
	if got != repo {
		t.Fatalf("repo root inesperado: got=%s want=%s", got, repo)
	}
}
