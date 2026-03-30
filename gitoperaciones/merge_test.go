package gitoperaciones

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeBranchIsolatedFusionaYActualizaRamaObjetivo(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	mustRunGit(t, "", "init", "-b", "master", repo)
	mustRunGit(t, repo, "config", "user.name", "Orquesta Test")
	mustRunGit(t, repo, "config", "user.email", "orquesta@example.test")

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	mustRunGit(t, repo, "add", "README.md")
	mustRunGit(t, repo, "commit", "-m", "base")

	mustRunGit(t, repo, "checkout", "-b", "feature/autonomia")
	if err := os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	mustRunGit(t, repo, "add", "feature.txt")
	mustRunGit(t, repo, "commit", "-m", "feature")
	sourceCommit := mustGitOutput(t, repo, "rev-parse", "feature/autonomia")
	mustRunGit(t, repo, "checkout", "master")

	result, err := MergeBranchIsolated(repo, "feature/autonomia", "master")
	if err != nil {
		t.Fatalf("MergeBranchIsolated: %v", err)
	}
	if result.SourceCommit != sourceCommit {
		t.Fatalf("source commit inesperado: got=%s want=%s", result.SourceCommit, sourceCommit)
	}
	masterHead := mustGitOutput(t, repo, "rev-parse", "master")
	if result.MergeCommit == "" || result.MergeCommit != masterHead {
		t.Fatalf("merge commit inesperado: result=%+v master=%s", result, masterHead)
	}
	if _, err := os.Stat(filepath.Join(repo, "feature.txt")); err != nil {
		t.Fatalf("feature.txt debería quedar fusionado en master: %v", err)
	}
	if ramas := mustGitOutput(t, repo, "branch", "--list", "orquesta/merge-tmp/*"); strings.TrimSpace(ramas) != "" {
		t.Fatalf("ramas temporales de merge no limpiadas: %s", ramas)
	}
	if entries, err := os.ReadDir(filepath.Join(repo, ".orquesta-worktrees", ".merge-tmp")); err == nil && len(entries) != 0 {
		t.Fatalf("worktrees temporales de merge no limpiados: %d", len(entries))
	}
}

func TestMergeBranchIsolatedFallaSiRepoDestinoTieneNoTrackeados(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	mustRunGit(t, "", "init", "-b", "master", repo)
	mustRunGit(t, repo, "config", "user.name", "Orquesta Test")
	mustRunGit(t, repo, "config", "user.email", "orquesta@example.test")

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	mustRunGit(t, repo, "add", "README.md")
	mustRunGit(t, repo, "commit", "-m", "base")

	mustRunGit(t, repo, "checkout", "-b", "feature/autonomia")
	if err := os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	mustRunGit(t, repo, "add", "feature.txt")
	mustRunGit(t, repo, "commit", "-m", "feature")
	mustRunGit(t, repo, "checkout", "master")

	if err := os.WriteFile(filepath.Join(repo, "local.tmp"), []byte("no trackeado\n"), 0o644); err != nil {
		t.Fatalf("write untracked: %v", err)
	}

	_, err := MergeBranchIsolated(repo, "feature/autonomia", "master")
	if err == nil || !strings.Contains(err.Error(), "cambios locales") {
		t.Fatalf("deberia rechazar repo destino sucio por no trackeados, got=%v", err)
	}
}

func mustRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func mustGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}
