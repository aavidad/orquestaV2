package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/runtimesapp"
)

type fakeGitStorePremium struct {
	worktrees []*db.Worktree
	merges    []*db.GitMergeRequest
	saved     *db.GitMergeRequest
	resolveID *int64
}

func (f *fakeGitStorePremium) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return f.worktrees, nil
}

func (f *fakeGitStorePremium) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return nil, nil
}

func (f *fakeGitStorePremium) ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error) {
	return f.merges, nil
}

func (f *fakeGitStorePremium) SaveMerge(item *db.GitMergeRequest) (int64, error) {
	f.saved = item
	if item == nil {
		return 0, nil
	}
	return 91, nil
}

func (f *fakeGitStorePremium) ResolveProjectID(slug string) (*int64, error) {
	return f.resolveID, nil
}

func TestPremiumRuntimeGitServiceRegistraMergeDesdeWorktreeActiva(t *testing.T) {
	repo := prepararRepoGitPremium(t)
	if err := os.WriteFile(filepath.Join(repo, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumTest() {}\n"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
	projectID := int64(11)
	store := &fakeGitStorePremium{
		resolveID: &projectID,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   projectID,
			ProyectoSlug: "orquestador",
			RutaAbs:      repo,
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Agente:       "Codex1",
			Estado:       "activa",
		}},
	}
	service := premiumRuntimeGitService{git: gitaplicacion.NewService(store)}

	resultado, err := service.RegistrarEntregaGitPremium(runtimesapp.EntradaRegistrarEntregaGitPremium{
		Agente:          "Codex1",
		ProyectoID:      &projectID,
		ProyectoSlug:    "orquestador",
		SolicitadoPor:   "orquesta",
		Evidencia:       "diff listo",
		Carril:          "premium_worktree",
		TareaObjetivoID: 530,
		WriteSet:        []string{"cmd/controlplane_support.go"},
	})
	if err != nil {
		t.Fatalf("RegistrarEntregaGitPremium: %v", err)
	}
	if resultado == nil || resultado.GitMergeID != 91 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.saved == nil {
		t.Fatalf("deberia guardar merge")
	}
	if store.saved.SourceBranch != "orq/orquestador/codex1" || store.saved.TargetBranch != "main" {
		t.Fatalf("ramas inesperadas: %+v", store.saved)
	}
	if !strings.Contains(store.saved.MetadataJSON, `"source":"premium_runtime"`) || !strings.Contains(store.saved.MetadataJSON, `"tarea_objetivo_id":530`) {
		t.Fatalf("metadata inesperada: %s", store.saved.MetadataJSON)
	}
}

func TestPremiumRuntimeGitServiceReutilizaMergePendiente(t *testing.T) {
	repo := prepararRepoGitPremium(t)
	projectID := int64(11)
	store := &fakeGitStorePremium{
		resolveID: &projectID,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   projectID,
			ProyectoSlug: "orquestador",
			RutaAbs:      repo,
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Agente:       "Codex1",
			Estado:       "activa",
		}},
		merges: []*db.GitMergeRequest{{
			ID:           77,
			ProyectoID:   projectID,
			ProyectoSlug: "orquestador",
			SourceBranch: "orq/orquestador/codex1",
			TargetBranch: "main",
			Estado:       "pendiente",
		}},
	}
	service := premiumRuntimeGitService{git: gitaplicacion.NewService(store)}

	resultado, err := service.RegistrarEntregaGitPremium(runtimesapp.EntradaRegistrarEntregaGitPremium{
		Agente:       "Codex1",
		ProyectoID:   &projectID,
		ProyectoSlug: "orquestador",
		Carril:       "premium_worktree",
		WriteSet:     []string{"cmd/controlplane_support.go"},
	})
	if err != nil {
		t.Fatalf("RegistrarEntregaGitPremium reutiliza merge: %v", err)
	}
	if resultado == nil || resultado.GitMergeID != 77 {
		t.Fatalf("merge reutilizado inesperado: %+v", resultado)
	}
	if store.saved != nil {
		t.Fatalf("no deberia crear merge nuevo: %+v", store.saved)
	}
}

func TestPremiumRuntimeGitServiceFallaSinWriteSet(t *testing.T) {
	repo := prepararRepoGitPremium(t)
	if err := os.WriteFile(filepath.Join(repo, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumTest() {}\n"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
	projectID := int64(11)
	store := &fakeGitStorePremium{
		resolveID: &projectID,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   projectID,
			ProyectoSlug: "orquestador",
			RutaAbs:      repo,
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Agente:       "Codex1",
			Estado:       "activa",
		}},
	}
	service := premiumRuntimeGitService{git: gitaplicacion.NewService(store)}

	_, err := service.RegistrarEntregaGitPremium(runtimesapp.EntradaRegistrarEntregaGitPremium{
		Agente:       "Codex1",
		ProyectoID:   &projectID,
		ProyectoSlug: "orquestador",
		Carril:       "premium_worktree",
	})
	if err == nil || !strings.Contains(err.Error(), "write_set premium vacio") {
		t.Fatalf("deberia fallar por write_set vacio, got=%v", err)
	}
	if store.saved != nil {
		t.Fatalf("no deberia crear merge con write_set vacio: %+v", store.saved)
	}
}

func TestPremiumRuntimeGitServiceFallaSiDiffSaleDelWriteSet(t *testing.T) {
	repo := prepararRepoGitPremium(t)
	if err := os.WriteFile(filepath.Join(repo, "cmd", "controlplane_support.go"), []byte("package cmd\n\nfunc premiumTest() {}\n"), 0o644); err != nil {
		t.Fatalf("write changed file: %v", err)
	}
	projectID := int64(11)
	store := &fakeGitStorePremium{
		resolveID: &projectID,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   projectID,
			ProyectoSlug: "orquestador",
			RutaAbs:      repo,
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Agente:       "Codex1",
			Estado:       "activa",
		}},
	}
	service := premiumRuntimeGitService{git: gitaplicacion.NewService(store)}

	_, err := service.RegistrarEntregaGitPremium(runtimesapp.EntradaRegistrarEntregaGitPremium{
		Agente:       "Codex1",
		ProyectoID:   &projectID,
		ProyectoSlug: "orquestador",
		Carril:       "premium_worktree",
		WriteSet:     []string{"db/controlplane_entities.go"},
	})
	if err == nil || !strings.Contains(err.Error(), "fuera del write_set") {
		t.Fatalf("deberia fallar por diff fuera del write_set, got=%v", err)
	}
	if store.saved != nil {
		t.Fatalf("no deberia crear merge fuera del write_set: %+v", store.saved)
	}
}

func prepararRepoGitPremium(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitPremium(t, repo, "init", "-b", "main")
	runGitPremium(t, repo, "config", "user.email", "test@example.com")
	runGitPremium(t, repo, "config", "user.name", "Test")
	if err := os.MkdirAll(filepath.Join(repo, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "cmd", "controlplane_support.go"), []byte("package cmd\n"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}
	runGitPremium(t, repo, "add", ".")
	runGitPremium(t, repo, "commit", "-m", "init")
	return repo
}

func runGitPremium(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
}
