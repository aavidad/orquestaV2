package gitapp

import "orquesta/db"

type Store interface {
	ListWorktrees(estado, agente string) ([]*db.Worktree, error)
	ListLocks(estado, agente string) ([]*db.Lock, error)
	ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error)
	SaveMerge(item *db.GitMergeRequest) (int64, error)
	ResolveProjectID(slug string) (*int64, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type CreateMergeInput struct {
	ProyectoSlug string
	SourceBranch string
	TargetBranch string
	RequestedBy  string
	Estado       string
	SourceCommit string
	MergeCommit  string
	Notas        string
	MetadataJSON string
}

func (s *Service) ListWorktrees(estado, agente string) ([]*db.Worktree, error) {
	return s.store.ListWorktrees(estado, agente)
}

func (s *Service) ListLocks(estado, agente string) ([]*db.Lock, error) {
	return s.store.ListLocks(estado, agente)
}

func (s *Service) ListMerges(proyectoSlug, estado string) ([]*db.GitMergeRequest, error) {
	return s.store.ListMerges(proyectoSlug, estado)
}

func (s *Service) CreateMerge(input CreateMergeInput) (int64, error) {
	proyectoID, err := s.store.ResolveProjectID(input.ProyectoSlug)
	if err != nil {
		return 0, err
	}
	return s.store.SaveMerge(&db.GitMergeRequest{
		ProyectoID:   *proyectoID,
		SourceBranch: input.SourceBranch,
		TargetBranch: input.TargetBranch,
		RequestedBy:  input.RequestedBy,
		Estado:       input.Estado,
		SourceCommit: input.SourceCommit,
		MergeCommit:  input.MergeCommit,
		Notas:        input.Notas,
		MetadataJSON: input.MetadataJSON,
	})
}
