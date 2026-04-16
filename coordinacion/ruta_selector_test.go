package coordinacion

import (
	"path/filepath"
	"testing"
)

func marcadorRepoStubSoloRaiz(path string) (bool, error) {
	abs := filepath.Clean(path)
	root := filepath.Clean(filepath.Join(string(filepath.Separator), "tmp", "repo", "orquestador"))
	return abs == root, nil
}

func TestPreferredWorkPathPrefersActiveWorktreeOverProjectRoot(t *testing.T) {
	root := filepath.Join("/tmp", "orquesta")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-gemini1")

	got := PreferredWorkPath(root, worktree, root, false)
	if got != worktree {
		t.Fatalf("ruta preferida inesperada: got=%s want=%s", got, worktree)
	}
}

func TestPreferredWorkPathKeepsCurrentPathInsideActiveWorktree(t *testing.T) {
	root := filepath.Join("/tmp", "orquesta")
	worktree := filepath.Join(root, ".orquesta-worktrees", "orq-codex1")
	current := filepath.Join(worktree, "cmd")

	got := PreferredWorkPath(current, worktree, root, true)
	if got != current {
		t.Fatalf("deberia conservar la ruta actual dentro de la worktree: got=%s want=%s", got, current)
	}
}

func TestPreferredWorkPathFallsBackToEffectiveProjectPath(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	old := filepath.Join(root, "claw-code-dev-rust", "rust", "crates", "api")

	got := PreferredWorkPath(old, "", root, false)
	if got != root {
		t.Fatalf("deberia volver a la ruta efectiva del proyecto: got=%s want=%s", got, root)
	}
}

func TestActiveWorktreePathCoherent(t *testing.T) {
	root := filepath.Join("/tmp", "repo", "orquestador")
	inside := filepath.Join(root, ".orquesta-worktrees", "orq-codex1")
	outside := filepath.Join("/tmp", "historico", "orquestador", ".orquesta-worktrees", "orq-codex1")

	if !ActiveWorktreePathCoherent(inside, root) {
		t.Fatalf("la worktree bajo la ruta efectiva deberia ser coherente")
	}
	if ActiveWorktreePathCoherent(outside, root) {
		t.Fatalf("la worktree fuera de la ruta efectiva no deberia ser coherente")
	}
}

func TestResolveProjectEffectivePathPrefersSessionOverActiveWorktreeHint(t *testing.T) {
	fallback := filepath.Join("/tmp", "historico", "orquestador")
	cwdHint := filepath.Join("/tmp", "repo", "orquestador", "cmd")
	session := filepath.Join("/tmp", "repo", "orquestador", "src")
	got := ResolveProjectEffectivePath(fallback, cwdHint, true, session, false, marcadorRepoStubSoloRaiz)
	if got != filepath.Join("/tmp", "repo", "orquestador") {
		t.Fatalf("ruta efectiva inesperada: got=%s want=%s", got, filepath.Join("/tmp", "repo", "orquestador"))
	}
}

func TestResolveProjectEffectivePathFallbackWhenNoCandidates(t *testing.T) {
	fallback := filepath.Join("/tmp", "fallback", "orquestador")
	got := ResolveProjectEffectivePath(
		fallback,
		filepath.Join("/tmp", "otro", "path"),
		false,
		filepath.Join("/tmp", "otro2", "path"),
		false,
		func(path string) (bool, error) {
			return false, nil
		},
	)
	if got != fallback {
		t.Fatalf("deberia caer al fallback cuando no hay candidatos: got=%s want=%s", got, fallback)
	}
}
