package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"orquesta/db"
)

func TestHigienizarInboxesWorktreeEliminaHuerfanasYDuplicadasPorTarea(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)
	repo := filepath.Join(tmp, "repo-inbox-hygiene")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	runGitCmdAPITest(t, repo, "init")
	runGitCmdAPITest(t, repo, "config", "user.name", "Orquesta Test")
	runGitCmdAPITest(t, repo, "config", "user.email", "orquesta@example.com")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCmdAPITest(t, repo, "add", "README.md")
	runGitCmdAPITest(t, repo, "commit", "-m", "base")

	root := filepath.Join(repo, ".orquesta-worktrees")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	target := filepath.Join(root, "wt-a")
	duplicate := filepath.Join(root, "wt-b")
	orphan := filepath.Join(root, "wt-stale")
	runGitCmdAPITest(t, repo, "worktree", "add", target, "-b", "orq/test/a", "HEAD")
	runGitCmdAPITest(t, repo, "worktree", "add", duplicate, "-b", "orq/test/b", "HEAD")
	if err := os.MkdirAll(orphan, 0o755); err != nil {
		t.Fatalf("mkdir orphan: %v", err)
	}

	targetInbox := "# Microtarea Activa de Orquesta\n\n- Tarea: `#41 Frente A`\n"
	duplicateInbox := "# Microtarea Activa de Orquesta\n\n- Tarea: `#41 Frente B duplicado`\n"
	orphanInbox := "# Microtarea Activa de Orquesta\n\n- Tarea: `#99 Frente stale`\n"
	if err := os.WriteFile(filepath.Join(target, ".orquesta-inbox.md"), []byte(targetInbox), 0o644); err != nil {
		t.Fatalf("write target inbox: %v", err)
	}
	if err := os.WriteFile(filepath.Join(duplicate, ".orquesta-inbox.md"), []byte(duplicateInbox), 0o644); err != nil {
		t.Fatalf("write duplicate inbox: %v", err)
	}
	if err := os.WriteFile(filepath.Join(orphan, ".orquesta-inbox.md"), []byte(orphanInbox), 0o644); err != nil {
		t.Fatalf("write orphan inbox: %v", err)
	}
	rootInbox := "# Microtarea Activa de Orquesta\n\n- Tarea: `#41 Frente root duplicado`\n"
	if err := os.WriteFile(filepath.Join(repo, ".orquesta-inbox.md"), []byte(rootInbox), 0o644); err != nil {
		t.Fatalf("write root inbox: %v", err)
	}

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "repo-inbox-hygiene",
		Nombre:  "Repo Inbox Hygiene",
		RutaAbs: repo,
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := higienizarInboxesWorktree(&db.Proyecto{ID: proyectoID, RutaAbs: repo}, target, 41); err != nil {
		t.Fatalf("higienizarInboxesWorktree: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, ".orquesta-inbox.md")); err != nil {
		t.Fatalf("la inbox target deberia permanecer: %v", err)
	}
	if _, err := os.Stat(filepath.Join(duplicate, ".orquesta-inbox.md")); !os.IsNotExist(err) {
		t.Fatalf("la inbox duplicada deberia eliminarse: %v", err)
	}
	if _, err := os.Stat(filepath.Join(orphan, ".orquesta-inbox.md")); !os.IsNotExist(err) {
		t.Fatalf("la inbox huerfana deberia eliminarse: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".orquesta-inbox.md")); !os.IsNotExist(err) {
		t.Fatalf("la inbox root duplicada deberia eliminarse: %v", err)
	}
}
