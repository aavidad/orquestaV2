package coordinacion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCurrentPathInsideProjectWorktree(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1", "cmd")
	outside := filepath.Join(root, "cmd")

	if !CurrentPathInsideProjectWorktree(worktree, root) {
		t.Fatalf("la ruta bajo .orquesta-worktrees deberia detectarse como worktree del proyecto")
	}
	if CurrentPathInsideProjectWorktree(outside, root) {
		t.Fatalf("la ruta fuera de .orquesta-worktrees no deberia detectarse como worktree")
	}
}

func TestSessionPathInsideActiveWorktree(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1")
	inside := filepath.Join(worktree, "cmd")
	other := filepath.Join(root, ".orquesta-worktrees", "orq-gemini1")

	if err := ensureDir(worktree); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := ensureDir(other); err != nil {
		t.Fatalf("mkdir other worktree: %v", err)
	}

	refs := []WorktreePathRef{
		{Agent: "Codex1", Path: worktree},
		{Agent: "Gemini1", Path: other},
	}
	if !SessionPathInsideActiveWorktree("Codex1", inside, refs) {
		t.Fatalf("la sesion deberia detectarse dentro de la worktree activa del agente")
	}
	if SessionPathInsideActiveWorktree("Gemini1", inside, refs) {
		t.Fatalf("la sesion no deberia emparejar la worktree de otro agente")
	}
}

func TestSelectUsableActiveWorktreePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo", "orquestador")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1")
	legacy := filepath.Join(t.TempDir(), "legacy", "orquestador", ".orquesta-worktrees", "orq-codex1")
	if err := ensureDir(worktree); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	if err := ensureDir(legacy); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}

	got := SelectUsableActiveWorktreePath([]WorktreePathRef{
		{Agent: "Codex1", Path: legacy},
		{Agent: "Codex1", Path: worktree},
	}, root)
	if got != worktree {
		t.Fatalf("ruta de worktree utilizable inesperada: got=%s want=%s", got, worktree)
	}
}

func TestShouldIncludeWorktreeForListing(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1")

	if !ShouldIncludeWorktreeForListing(WorktreeActive, worktree, root) {
		t.Fatalf("la worktree activa coherente deberia incluirse")
	}
	if ShouldIncludeWorktreeForListing(WorktreeActive, filepath.Join("/tmp", "otro", "repo"), root) {
		t.Fatalf("la worktree activa incoherente no deberia incluirse")
	}
	if !ShouldIncludeWorktreeForListing(WorktreeClosed, filepath.Join("/tmp", "otro", "repo"), root) {
		t.Fatalf("las worktrees no activas no deberian filtrarse por coherencia")
	}
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
