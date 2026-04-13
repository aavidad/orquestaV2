package gitaplicacion

import (
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
	t.Parallel()

	store := &fakeStore{
		worktrees: []*db.Worktree{
			{ID: 1, ProyectoSlug: "otro", Agente: "Gemma1", Estado: "activa"},
			{ID: 2, ProyectoSlug: "orquestador", Agente: "Gemma1", Estado: "activa", RutaAbs: "/tmp/wt"},
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
