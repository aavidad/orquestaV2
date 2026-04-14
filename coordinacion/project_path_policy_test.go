package coordinacion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCandidateEffectiveProjectPathIgnoresActiveWorktreeSession(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1", "cmd")

	got := CandidateEffectiveProjectPath(worktree, true, func(path string) (bool, error) {
		return path == root, nil
	})
	if got != "" {
		t.Fatalf("no deberia devolver ruta efectiva para una sesion ya dentro de worktree activa: %s", got)
	}
}

func TestResolveWorkRootFindsNearestRepoRoot(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	nested := filepath.Join(root, "cmd", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, ok := ResolveWorkRoot(nested, func(path string) (bool, error) {
		return path == root, nil
	})
	if !ok || got != root {
		t.Fatalf("raiz inesperada: got=%s ok=%v want=%s", got, ok, root)
	}
}

func TestWorkPathBelongsToProject(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1")

	if !WorkPathBelongsToProject(filepath.Join(root, "cmd"), root, "") {
		t.Fatalf("la ruta bajo la raiz del proyecto deberia pertenecer al proyecto")
	}
	if !WorkPathBelongsToProject(filepath.Join(worktree, "pkg"), root, worktree) {
		t.Fatalf("la ruta bajo la worktree activa deberia pertenecer al proyecto")
	}
	if WorkPathBelongsToProject(filepath.Join("/tmp", "otro", "repo"), root, worktree) {
		t.Fatalf("la ruta ajena no deberia pertenecer al proyecto")
	}
}
