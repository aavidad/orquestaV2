package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestEliminarColisionWorktreeMicrocicloPurgaRefsLegacyConflictivas(t *testing.T) {
	repo := t.TempDir()
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")
	runGitCmdAPITest(t, repo, "branch", "orq-orquestador-gemini1/t526")

	worktreePath := filepath.Join(repo, ".orquesta-worktrees", "orquestador-gemini1")
	if err := os.MkdirAll(worktreePath, 0o755); err != nil {
		t.Fatalf("mkdir worktree path: %v", err)
	}

	if err := eliminarColisionWorktreeMicrociclo(&db.Proyecto{Slug: "orquestador", RutaAbs: repo}, "Gemini1"); err != nil {
		t.Fatalf("eliminarColisionWorktreeMicrociclo: %v", err)
	}

	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Fatalf("la ruta de worktree deberia eliminarse, err=%v", err)
	}
	refs := gitRefsProyectoMicrocicloTest(t, repo)
	if strings.Contains(refs, "orq-orquestador-gemini1/t526") {
		t.Fatalf("la ref legacy conflictiva deberia haberse purgado: %s", refs)
	}
}

func gitRefsProyectoMicrocicloTest(t *testing.T, repo string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git for-each-ref: %v output=%s", err, string(out))
	}
	return string(out)
}
