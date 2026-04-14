package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestSincronizarWorkspaceProyectoEnWorktreeNoCopiaWorkspaceSucioPorDefecto(t *testing.T) {
	t.Setenv("ORQUESTA_SYNC_DIRTY_WORKSPACE_TO_WORKTREE", "")

	repo := t.TempDir()
	worktree := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "archivo.txt"), []byte("repo-sucio\n"), 0o644); err != nil {
		t.Fatalf("write repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(worktree, "archivo.txt"), []byte("worktree-limpia\n"), 0o644); err != nil {
		t.Fatalf("write worktree: %v", err)
	}

	if err := sincronizarWorkspaceProyectoEnWorktree(&db.Proyecto{RutaAbs: repo}, &db.Worktree{RutaAbs: worktree}); err != nil {
		t.Fatalf("sincronizarWorkspaceProyectoEnWorktree: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(worktree, "archivo.txt"))
	if err != nil {
		t.Fatalf("read worktree: %v", err)
	}
	if string(got) != "worktree-limpia\n" {
		t.Fatalf("la worktree no deberia contaminarse por defecto, got=%q", string(got))
	}
}

func TestWorktreeSyncDirtyWorkspaceEnabledAceptaTrue(t *testing.T) {
	t.Setenv("ORQUESTA_SYNC_DIRTY_WORKSPACE_TO_WORKTREE", "true")
	if !worktreeSyncDirtyWorkspaceEnabled() {
		t.Fatal("el flag explicito deberia habilitar la sincronizacion legacy")
	}
}

func TestWorktreeActivaReutilizableParaPremiumRechazaWorktreeSuciaPorDefecto(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	worktree := filepath.Join(tmp, "worktree")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write repo README: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")
	runGitCmdAPITest(t, repo, "worktree", "add", worktree, "HEAD")
	if err := os.WriteFile(filepath.Join(worktree, "README.md"), []byte("dirty-worktree\n"), 0o644); err != nil {
		t.Fatalf("write dirty worktree: %v", err)
	}

	ok, err := worktreeActivaReutilizableParaPremium(
		&db.Proyecto{RutaAbs: repo, Tipo: db.ProyectoRepo},
		&db.Worktree{RutaAbs: worktree},
	)
	if err != nil {
		t.Fatalf("worktreeActivaReutilizableParaPremium: %v", err)
	}
	if ok {
		t.Fatal("una worktree premium sucia no deberia reutilizarse por defecto")
	}
}

func TestWorktreeActivaReutilizableParaPremiumAceptaWorktreeSuciaConFlagLegacy(t *testing.T) {
	t.Setenv("ORQUESTA_SYNC_DIRTY_WORKSPACE_TO_WORKTREE", "true")

	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	worktree := filepath.Join(tmp, "worktree")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write repo README: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")
	runGitCmdAPITest(t, repo, "worktree", "add", worktree, "HEAD")
	if err := os.WriteFile(filepath.Join(worktree, "README.md"), []byte("dirty-worktree\n"), 0o644); err != nil {
		t.Fatalf("write dirty worktree: %v", err)
	}

	ok, err := worktreeActivaReutilizableParaPremium(
		&db.Proyecto{RutaAbs: repo, Tipo: db.ProyectoRepo},
		&db.Worktree{RutaAbs: worktree},
	)
	if err != nil {
		t.Fatalf("worktreeActivaReutilizableParaPremium: %v", err)
	}
	if !ok {
		t.Fatal("el flag legacy deberia permitir reutilizar worktree sucia")
	}
}
