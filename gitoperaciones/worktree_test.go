package gitoperaciones

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeleteBranchDescendantsEliminaRefsLegacyAnidadas(t *testing.T) {
	repo := t.TempDir()
	runGitCmdWorktreeTest(t, repo, "init")
	runGitCmdWorktreeTest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdWorktreeTest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitCmdWorktreeTest(t, repo, "add", "README.md")
	runGitCmdWorktreeTest(t, repo, "commit", "-m", "base")
	runGitCmdWorktreeTest(t, repo, "branch", "orq-orquestador-gemini1/t526")
	runGitCmdWorktreeTest(t, repo, "branch", "feature/keep")

	if err := (WorktreeManager{}).DeleteBranchDescendants(repo, "orq-orquestador-gemini1"); err != nil {
		t.Fatalf("DeleteBranchDescendants: %v", err)
	}

	refs := gitRefsHeadsWorktreeTest(t, repo)
	if strings.Contains(refs, "orq-orquestador-gemini1/t526") {
		t.Fatalf("la ref legacy descendiente deberia haberse eliminado: %s", refs)
	}
	if !strings.Contains(refs, "feature/keep") {
		t.Fatalf("la ref no relacionada deberia mantenerse: %s", refs)
	}
}

func runGitCmdWorktreeTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
	return string(out)
}

func gitRefsHeadsWorktreeTest(t *testing.T, repo string) string {
	t.Helper()
	return runGitCmdWorktreeTest(t, repo, "for-each-ref", "--format=%(refname:short)", "refs/heads")
}
