package gitaplicacion

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	projectID int64
	merge     *db.GitMergeRequest
	worktrees []*db.Worktree
}

func (f *fakeStore) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return f.worktrees, nil
}

func (f *fakeStore) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return nil, nil
}

func (f *fakeStore) ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error) {
	return []*db.GitMergeRequest{{ProyectoSlug: proyectoSlug}}, nil
}

func (f *fakeStore) SaveMerge(item *db.GitMergeRequest) (int64, error) {
	f.merge = item
	return 99, nil
}

func (f *fakeStore) ResolveProjectID(slug string) (*int64, error) {
	return &f.projectID, nil
}

func TestCreateMergeResuelveProyecto(t *testing.T) {
	t.Parallel()

	store := &fakeStore{projectID: 7}
	svc := NewService(store)
	id, err := svc.CreateMerge(CreateMergeInput{
		ProyectoSlug: "orquestador",
		SourceBranch: "feat-a",
		TargetBranch: "main",
		RequestedBy:  "codex2",
	})
	if err != nil {
		t.Fatalf("CreateMerge: %v", err)
	}
	if id != 99 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.merge == nil || store.merge.ProyectoID != 7 || store.merge.SourceBranch != "feat-a" {
		t.Fatalf("merge inesperado: %+v", store.merge)
	}
}

func TestResolveActiveWorktreeFiltraPorProyecto(t *testing.T) {
	prev := validateActiveWorktree
	validateActiveWorktree = func(item *db.Worktree) bool { return item != nil && item.ID == 2 }
	t.Cleanup(func() { validateActiveWorktree = prev })

	store := &fakeStore{
		projectID: 7,
		worktrees: []*db.Worktree{
			{ID: 1, ProyectoID: 8, ProyectoSlug: "otro", Agente: "Gemma1", Estado: "activa"},
			{ID: 2, ProyectoID: 7, ProyectoSlug: "orquestador", Agente: "Gemma1", Estado: "activa", RutaAbs: "/tmp/wt"},
		},
	}
	svc := NewService(store)
	item, err := svc.ResolveActiveWorktree("orquestador", "Gemma1")
	if err != nil {
		t.Fatalf("ResolveActiveWorktree: %v", err)
	}
	if item == nil || item.ID != 2 {
		t.Fatalf("worktree inesperada: %+v", item)
	}
}

func TestResolveActiveWorktreeOmiteRutaActivaInvalida(t *testing.T) {
	prev := validateActiveWorktree
	validateActiveWorktree = func(item *db.Worktree) bool { return item != nil && item.ID == 2 }
	t.Cleanup(func() { validateActiveWorktree = prev })

	store := &fakeStore{
		projectID: 7,
		worktrees: []*db.Worktree{
			{ID: 1, ProyectoID: 7, ProyectoSlug: "orquestador", Agente: "Codex1", Estado: "activa", RutaAbs: "/tmp/wt-rota"},
			{ID: 2, ProyectoID: 7, ProyectoSlug: "orquestador", Agente: "Codex1", Estado: "activa", RutaAbs: "/tmp/wt-buena"},
		},
	}
	svc := NewService(store)
	item, err := svc.ResolveActiveWorktree("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("ResolveActiveWorktree: %v", err)
	}
	if item == nil || item.ID != 2 {
		t.Fatalf("deberia omitir la worktree rota y devolver la valida: %+v", item)
	}
}

func TestResolveActiveWorktreeAceptaProjectIDSinSlugPersistido(t *testing.T) {
	prev := validateActiveWorktree
	validateActiveWorktree = func(item *db.Worktree) bool { return item != nil && item.ID == 2 }
	t.Cleanup(func() { validateActiveWorktree = prev })

	store := &fakeStore{
		projectID: 7,
		worktrees: []*db.Worktree{
			{ID: 1, ProyectoID: 8, ProyectoSlug: "", Agente: "Codex1", Estado: "activa", RutaAbs: "/tmp/wt-otro"},
			{ID: 2, ProyectoID: 7, ProyectoSlug: "", Agente: "Codex1", Estado: "activa", RutaAbs: "/tmp/wt-ok"},
		},
	}
	svc := NewService(store)
	item, err := svc.ResolveActiveWorktree("orquestador", "Codex1")
	if err != nil {
		t.Fatalf("ResolveActiveWorktree: %v", err)
	}
	if item == nil || item.ID != 2 {
		t.Fatalf("deberia aceptar la worktree por project_id aunque falte el slug: %+v", item)
	}
}

func TestSyncDirtyWorkspaceToWorktreeCopiaCambiosLocales(t *testing.T) {
	t.Parallel()

	repo := initGitRepoForSyncTest(t)
	writeFileSyncTest(t, filepath.Join(repo, "uno.txt"), "uno\n")
	gitCommitSyncTest(t, repo, "uno.txt")

	if err := os.WriteFile(filepath.Join(repo, "uno.txt"), []byte("uno-mod\n"), 0o644); err != nil {
		t.Fatalf("modificar tracked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "nuevo.txt"), []byte("nuevo\n"), 0o644); err != nil {
		t.Fatalf("crear untracked: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".orquesta-inbox.md"), []byte("no-sync\n"), 0o644); err != nil {
		t.Fatalf("crear inbox root: %v", err)
	}
	worktree := t.TempDir()
	writeFileSyncTest(t, filepath.Join(worktree, "uno.txt"), "uno-viejo\n")

	svc := NewService(&fakeStore{})
	result, err := svc.SyncDirtyWorkspaceToWorktree(repo, worktree)
	if err != nil {
		t.Fatalf("SyncDirtyWorkspaceToWorktree: %v", err)
	}
	if !slices.Contains(result.SyncedPaths, "uno.txt") || !slices.Contains(result.SyncedPaths, "nuevo.txt") {
		t.Fatalf("paths sincronizadas inesperadas: %+v", result)
	}
	if !slices.Contains(result.SkippedPaths, ".orquesta-inbox.md") {
		t.Fatalf("deberia omitir inbox local: %+v", result)
	}
	rawUno, err := os.ReadFile(filepath.Join(worktree, "uno.txt"))
	if err != nil {
		t.Fatalf("leer uno.txt sync: %v", err)
	}
	if string(rawUno) != "uno-mod\n" {
		t.Fatalf("contenido uno.txt inesperado: %q", string(rawUno))
	}
	rawNuevo, err := os.ReadFile(filepath.Join(worktree, "nuevo.txt"))
	if err != nil {
		t.Fatalf("leer nuevo.txt sync: %v", err)
	}
	if string(rawNuevo) != "nuevo\n" {
		t.Fatalf("contenido nuevo.txt inesperado: %q", string(rawNuevo))
	}
	if _, err := os.Stat(filepath.Join(worktree, ".orquesta-inbox.md")); !os.IsNotExist(err) {
		t.Fatalf("la inbox local no deberia sincronizarse a la worktree")
	}
}

func TestSyncDirtyWorkspaceToWorktreeEliminaBorradosLocales(t *testing.T) {
	t.Parallel()

	repo := initGitRepoForSyncTest(t)
	writeFileSyncTest(t, filepath.Join(repo, "borrar.txt"), "adios\n")
	gitCommitSyncTest(t, repo, "borrar.txt")
	if err := os.Remove(filepath.Join(repo, "borrar.txt")); err != nil {
		t.Fatalf("borrar archivo en repo: %v", err)
	}
	worktree := t.TempDir()
	writeFileSyncTest(t, filepath.Join(worktree, "borrar.txt"), "viejo\n")

	svc := NewService(&fakeStore{})
	result, err := svc.SyncDirtyWorkspaceToWorktree(repo, worktree)
	if err != nil {
		t.Fatalf("SyncDirtyWorkspaceToWorktree: %v", err)
	}
	if !slices.Contains(result.RemovedPaths, "borrar.txt") {
		t.Fatalf("deberia registrar la ruta borrada: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(worktree, "borrar.txt")); !os.IsNotExist(err) {
		t.Fatalf("borrar.txt deberia desaparecer de la worktree")
	}
}

func initGitRepoForSyncTest(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitSyncTest(t, repo, "init")
	runGitSyncTest(t, repo, "config", "user.name", "Orquesta Test")
	runGitSyncTest(t, repo, "config", "user.email", "orquesta@example.com")
	return repo
}

func gitCommitSyncTest(t *testing.T, repo string, paths ...string) {
	t.Helper()
	args := append([]string{"add"}, paths...)
	runGitSyncTest(t, repo, args...)
	runGitSyncTest(t, repo, "commit", "-m", "sync test")
}

func writeFileSyncTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGitSyncTest(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v output=%s", args, err, string(out))
	}
}
