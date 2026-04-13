package cmd

import (
	"testing"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/gitaplicacion"
)

type fakeGitStorePipelineDispatcher struct {
	projectID int64
	worktrees []*db.Worktree
	merges    []*db.GitMergeRequest
	saved     *db.GitMergeRequest
}

func (f *fakeGitStorePipelineDispatcher) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return f.worktrees, nil
}

func (f *fakeGitStorePipelineDispatcher) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return nil, nil
}

func (f *fakeGitStorePipelineDispatcher) ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error) {
	return f.merges, nil
}

func (f *fakeGitStorePipelineDispatcher) SaveMerge(item *db.GitMergeRequest) (int64, error) {
	f.saved = item
	return 77, nil
}

func (f *fakeGitStorePipelineDispatcher) ResolveProjectID(slug string) (*int64, error) {
	return &f.projectID, nil
}

func TestDespachadorPipelineOperativoSolicitaMergeEnIntegracion(t *testing.T) {
	store := &fakeGitStorePipelineDispatcher{
		projectID: 9,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   9,
			ProyectoSlug: "orquestador",
			Agente:       "Codex1",
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Estado:       "activa",
		}},
	}
	anterior := gitService
	gitService = gitaplicacion.NewService(store)
	t.Cleanup(func() { gitService = anterior })

	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "integracion",
			Carril:          "determinista_app",
			TareaObjetivoID: 12,
			TareaObjetivo:   "Integrar parser",
			AgenteTarea:     "Codex1",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "merge_solicitado" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.saved == nil {
		t.Fatalf("merge no guardado")
	}
	if store.saved.SourceBranch != "orq/orquestador/codex1" || store.saved.TargetBranch != "main" {
		t.Fatalf("ramas inesperadas: %+v", store.saved)
	}
}

func TestDespachadorPipelineOperativoReutilizaMergePendienteEnIntegracion(t *testing.T) {
	store := &fakeGitStorePipelineDispatcher{
		projectID: 9,
		worktrees: []*db.Worktree{{
			ID:           81,
			ProyectoID:   9,
			ProyectoSlug: "orquestador",
			Agente:       "Codex1",
			Branch:       "orq/orquestador/codex1",
			BaseRef:      "main",
			Estado:       "activa",
		}},
		merges: []*db.GitMergeRequest{{
			ID:           91,
			ProyectoID:   9,
			ProyectoSlug: "orquestador",
			SourceBranch: "orq/orquestador/codex1",
			TargetBranch: "main",
			Estado:       "pendiente",
		}},
	}
	anterior := gitService
	gitService = gitaplicacion.NewService(store)
	t.Cleanup(func() { gitService = anterior })

	resultado, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: "orquestador",
		Despacho: &capacidadapp.DespachoPipelineLocal{
			Fase:            "integracion",
			Carril:          "determinista_app",
			TareaObjetivoID: 12,
			TareaObjetivo:   "Integrar parser",
			AgenteTarea:     "Codex1",
		},
	})
	if err != nil {
		t.Fatalf("DespacharPipeline: %v", err)
	}
	if resultado == nil || resultado.Estado != "merge_ya_pendiente" {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	if store.saved != nil {
		t.Fatalf("no deberia guardar un merge nuevo: %+v", store.saved)
	}
}
