package coordinacion

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildBranchNameMantieneBaseSinTaskID(t *testing.T) {
	got := buildBranchName("orquestador", "Gemini1", nil)
	if got != "orq-orquestador-gemini1" {
		t.Fatalf("branch base inesperada: %q", got)
	}
}

func TestBuildBranchNameUsaSufijoPlanoParaTaskID(t *testing.T) {
	taskID := int64(526)
	got := buildBranchName("orquestador", "Gemini1", &taskID)
	if got != "orq-orquestador-gemini1-t526" {
		t.Fatalf("branch task-specific inesperada: %q", got)
	}
	if containsSlashAfterBase(got) {
		t.Fatalf("la branch task-specific no deberia usar jerarquia git: %q", got)
	}
}

func containsSlashAfterBase(branch string) bool {
	for _, r := range branch {
		if r == '/' {
			return true
		}
	}
	return false
}

type testProjectRepo struct {
	project          *Project
	liteProject      *Project
	liteAgentProject *Project
}

func (r testProjectRepo) GetByRef(ref string) (*Project, error) {
	return r.project, nil
}

func (r testProjectRepo) GetByRefPrepareLite(ref string) (*Project, error) {
	if r.liteProject != nil {
		return r.liteProject, nil
	}
	return r.project, nil
}

func (r testProjectRepo) GetByRefPrepareLiteForAgent(ref, agent string) (*Project, error) {
	if r.liteAgentProject != nil {
		return r.liteAgentProject, nil
	}
	if r.liteProject != nil {
		return r.liteProject, nil
	}
	return r.project, nil
}

type testWorktreeRepo struct {
	created  *Worktree
	byPath   map[string]*Worktree
	createFn func(*Worktree) (*Worktree, error)
}

func (r *testWorktreeRepo) Create(worktree *Worktree) (*Worktree, error) {
	if r.createFn != nil {
		return r.createFn(worktree)
	}
	copia := *worktree
	copia.ID = 1
	r.created = &copia
	return &copia, nil
}

func (r *testWorktreeRepo) GetByID(id int64) (*Worktree, error) { return nil, nil }
func (r *testWorktreeRepo) GetActiveByPath(path string) (*Worktree, error) {
	if r.byPath == nil {
		return nil, nil
	}
	return r.byPath[path], nil
}
func (r *testWorktreeRepo) List(filter WorktreeFilter) ([]*Worktree, error) { return nil, nil }
func (r *testWorktreeRepo) Close(id int64, closedAt time.Time, reason string) (*Worktree, error) {
	return nil, nil
}

type testWorkspaceManager struct {
	repoPath     string
	worktreePath string
	branch       string
	baseRef      string
	err          error
}

func (w *testWorkspaceManager) CreateWorktree(repoPath, worktreePath, branch, baseRef string) error {
	w.repoPath = repoPath
	w.worktreePath = worktreePath
	w.branch = branch
	w.baseRef = baseRef
	return w.err
}

func (w *testWorkspaceManager) RemoveWorktree(repoPath, worktreePath string) error { return nil }

type panicConfigRepo struct{}

func (panicConfigRepo) Get(key string) (string, error) {
	panic("PrepareWorktree no debe consultar config para worktree_root_name")
}

func TestPrepareWorktreeUsaRootPorDefectoSinConsultarConfig(t *testing.T) {
	project := &Project{ID: 3, Slug: "orquestador", RootPath: "/tmp/orquesta"}
	worktrees := &testWorktreeRepo{}
	workspace := &testWorkspaceManager{}
	taskID := int64(559)

	svc := Service{
		Projects:  testProjectRepo{project: project},
		Worktrees: worktrees,
		Config:    panicConfigRepo{},
		Workspace: workspace,
	}

	wt, err := svc.PrepareWorktree(PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "GemmaSmoke5",
		TaskID:     &taskID,
	})
	if err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}
	wantPath := filepath.Join(project.RootPath, defaultWorktreeRootName, "orquestador-gemmasmoke5-t559")
	if workspace.worktreePath != wantPath {
		t.Fatalf("worktree path inesperada: got %q want %q", workspace.worktreePath, wantPath)
	}
	if wt == nil || wt.Path != wantPath {
		t.Fatalf("worktree creada inesperada: %#v", wt)
	}
}

func TestPrepareWorktreeReconcilesExistingPhysicalWorktree(t *testing.T) {
	projectRoot := t.TempDir()
	project := &Project{ID: 3, Slug: "orquestador", RootPath: projectRoot}
	worktrees := &testWorktreeRepo{}
	taskID := int64(559)
	wantPath := filepath.Join(project.RootPath, defaultWorktreeRootName, "orquestador-gemmasmoke5-t559")
	if err := os.MkdirAll(wantPath, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	workspace := &testWorkspaceManager{err: fmt.Errorf("git worktree add: exit status 128: fatal: 'orq-orquestador-gemmasmoke5-t559' is already used by worktree at '%s'", wantPath)}

	svc := Service{
		Projects:  testProjectRepo{project: project},
		Worktrees: worktrees,
		Workspace: workspace,
	}

	wt, err := svc.PrepareWorktree(PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "GemmaSmoke5",
		TaskID:     &taskID,
	})
	if err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}
	if wt == nil || wt.Path != wantPath || wt.ID <= 0 {
		t.Fatalf("worktree reconciliada inesperada: %#v", wt)
	}
	if worktrees.created == nil || worktrees.created.Path != wantPath {
		t.Fatalf("deberia persistir la worktree existente: %#v", worktrees.created)
	}
}

func TestPrepareWorktreeDevuelveWorktreeSinteticaSiPersistenciaSeBloquea(t *testing.T) {
	projectRoot := t.TempDir()
	project := &Project{ID: 3, Slug: "orquestador", RootPath: projectRoot}
	taskID := int64(559)
	wantPath := filepath.Join(project.RootPath, defaultWorktreeRootName, "orquestador-gemmasmoke5-t559")
	if err := os.MkdirAll(wantPath, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}
	workspace := &testWorkspaceManager{}
	worktrees := &testWorktreeRepo{
		createFn: func(worktree *Worktree) (*Worktree, error) {
			time.Sleep(50 * time.Millisecond)
			return nil, nil
		},
	}
	prev := prepareWorktreeStoreTimeout
	prepareWorktreeStoreTimeout = 10 * time.Millisecond
	t.Cleanup(func() { prepareWorktreeStoreTimeout = prev })

	svc := Service{
		Projects:  testProjectRepo{project: project},
		Worktrees: worktrees,
		Workspace: workspace,
	}

	wt, err := svc.PrepareWorktree(PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "GemmaSmoke5",
		TaskID:     &taskID,
	})
	if err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}
	if wt == nil || wt.ID != 0 || wt.Path != wantPath {
		t.Fatalf("worktree sintetica inesperada: %#v", wt)
	}
	if wt.CreatedAt.IsZero() || wt.UpdatedAt.IsZero() {
		t.Fatalf("timestamps sinteticos no inicializados: %#v", wt)
	}
}

func TestPrepareWorktreeUsaGetterLitePorAgenteCuandoExiste(t *testing.T) {
	project := &Project{ID: 3, Slug: "orquestador", RootPath: "/tmp/base"}
	liteProject := &Project{ID: 3, Slug: "orquestador", RootPath: "/tmp/incorrecta"}
	liteAgentProject := &Project{ID: 3, Slug: "orquestador", RootPath: "/tmp/correcta"}
	repo := testProjectRepo{project: project, liteProject: liteProject, liteAgentProject: liteAgentProject}
	worktrees := &testWorktreeRepo{}
	workspace := &testWorkspaceManager{}

	svc := Service{
		Projects:  repo,
		Worktrees: worktrees,
		Workspace: workspace,
	}

	wt, err := svc.PrepareWorktree(PrepareWorktreeInput{
		ProjectRef: "orquestador",
		Agent:      "QwenCoder1",
	})
	if err != nil {
		t.Fatalf("PrepareWorktree: %v", err)
	}
	wantPath := filepath.Join(liteAgentProject.RootPath, defaultWorktreeRootName, "orquestador-qwencoder1")
	if workspace.repoPath != liteAgentProject.RootPath {
		t.Fatalf("repoPath inesperada: got %q want %q", workspace.repoPath, liteAgentProject.RootPath)
	}
	if workspace.worktreePath != wantPath {
		t.Fatalf("worktree path inesperada: got %q want %q", workspace.worktreePath, wantPath)
	}
	if wt == nil || wt.Path != wantPath {
		t.Fatalf("worktree inesperada: %#v", wt)
	}
}
