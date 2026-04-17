package gitgobernanza

import (
	"testing"

	"orquesta/coordinacion"
	"orquesta/db"
)

type fakeStore struct {
	projectID int64
	saved     *GitMerge
	listed    []*GitMerge
	worktrees []*db.Worktree
	locks     []*db.Lock
}

func (f *fakeStore) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return f.worktrees, nil
}

func (f *fakeStore) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return f.locks, nil
}

func (f *fakeStore) GetWorktree(id int64) (*coordinacion.Worktree, error) {
	return &coordinacion.Worktree{ID: id, Name: "wt-a"}, nil
}

func (f *fakeStore) GetLock(id int64) (*coordinacion.Lock, error) {
	return &coordinacion.Lock{ID: id, ScopeKey: "branch:a"}, nil
}

func (f *fakeStore) ResolveProyectoIDBySlug(slug string) (*int64, error) {
	return &f.projectID, nil
}

func (f *fakeStore) GetGitMerge(id int64) (*GitMerge, error) {
	if f.saved != nil && f.saved.ID == id {
		return f.saved, nil
	}
	for _, item := range f.listed {
		if item != nil && item.ID == id {
			return item, nil
		}
	}
	return nil, nil
}

func (f *fakeStore) SaveGitMerge(m *GitMerge) (int64, error) {
	f.saved = m
	if m.ID != 0 {
		return m.ID, nil
	}
	return 44, nil
}

func (f *fakeStore) ListGitMerges(proyectoID *int64, estado string) ([]*GitMerge, error) {
	return f.listed, nil
}

func TestSaveRequestResuelveProyectoYNormalizaCampos(t *testing.T) {
	t.Parallel()

	store := &fakeStore{projectID: 9}
	svc := NewService(store)
	id, err := svc.SaveRequest(SaveMergeRequestInput{
		ProyectoSlug: " orquestador ",
		SourceBranch: " feature/mcp ",
		TargetBranch: " main ",
		RequestedBy:  " codex2 ",
		Estado:       " pendiente ",
		Notas:        " listo para revisar ",
	})
	if err != nil {
		t.Fatalf("SaveRequest: %v", err)
	}
	if id != 44 {
		t.Fatalf("id inesperado: %d", id)
	}
	if store.saved == nil {
		t.Fatalf("no se guardo el merge")
	}
	if store.saved.ProyectoID != 9 || store.saved.SourceBranch != "feature/mcp" || store.saved.TargetBranch != "main" {
		t.Fatalf("payload inesperado: %+v", store.saved)
	}
	if store.saved.RequestedBy != "codex2" || store.saved.Estado != "pendiente" {
		t.Fatalf("normalizacion inesperada: %+v", store.saved)
	}
}

func TestListRequestsResuelveProyectoYDelega(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		projectID: 3,
		listed: []*GitMerge{
			{ID: 1, ProyectoID: 3, SourceBranch: "feat/a", TargetBranch: "main"},
		},
	}
	svc := NewService(store)
	items, err := svc.ListRequests("orquestador", "pendiente")
	if err != nil {
		t.Fatalf("ListRequests: %v", err)
	}
	if len(items) != 1 || items[0].ProyectoID != 3 {
		t.Fatalf("resultado inesperado: %+v", items)
	}
}

func TestListRequestsSinProyectoPermiteListadoGlobal(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		listed: []*GitMerge{
			{ID: 1, ProyectoID: 3, SourceBranch: "feat/a", TargetBranch: "main"},
		},
	}
	svc := NewService(store)
	items, err := svc.ListRequests("", "pendiente")
	if err != nil {
		t.Fatalf("ListRequests: %v", err)
	}
	if len(items) != 1 || items[0].ID != 1 {
		t.Fatalf("resultado inesperado: %+v", items)
	}
}

func TestListWorktreesAndLocksDelegan(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		worktrees: []*db.Worktree{{ID: 1, Nombre: "wt-a"}},
		locks:     []*db.Lock{{ID: 2, ScopeKey: "branch:a"}},
	}
	svc := NewService(store)
	worktrees, err := svc.ListWorktrees("activa", "codex2")
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	locks, err := svc.ListLocks("activa", "codex2")
	if err != nil {
		t.Fatalf("ListLocks: %v", err)
	}
	if len(worktrees) != 1 || worktrees[0].Nombre != "wt-a" {
		t.Fatalf("worktrees inesperadas: %+v", worktrees)
	}
	if len(locks) != 1 || locks[0].ScopeKey != "branch:a" {
		t.Fatalf("locks inesperados: %+v", locks)
	}
}
