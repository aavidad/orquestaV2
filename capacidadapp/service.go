package capacidadapp

import "orquesta/db"

type Store interface {
	ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error)
	GetPool(slug string) (*db.PoolCapacidad, error)
	ListPoolModels(slug string) ([]*db.PoolModelo, error)
	SavePool(pool *db.PoolCapacidad) (int64, error)
	SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error)
	SeedInitialModels() error
	ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error)
	SaveModelPolicy(policy *db.PoliticaModelo) (int64, error)
	SeedInitialModelPolicies() error
	ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error)
}

type PhaseProvider interface {
	GetActivePhase(proyecto string) (string, error)
}

type Service struct {
	store         Store
	phaseProvider PhaseProvider
}

type PoolDetail struct {
	Pool                *db.PoolCapacidad
	SesionesActivas     int
	CapacidadDisponible int
	Modelos             []*db.PoolModelo
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) SetPhaseProvider(provider PhaseProvider) {
	s.phaseProvider = provider
}

func (s *Service) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return s.store.ListPoolsSummary(activo)
}

func (s *Service) GetPoolDetail(slug string) (*PoolDetail, error) {
	pool, err := s.store.GetPool(slug)
	if err != nil {
		return nil, err
	}
	modelos, err := s.store.ListPoolModels(slug)
	if err != nil {
		return nil, err
	}
	resumen, err := s.store.ListPoolsSummary(nil)
	if err != nil {
		return nil, err
	}
	detail := &PoolDetail{
		Pool:                pool,
		SesionesActivas:     0,
		CapacidadDisponible: pool.CapacidadTotal - pool.CapacidadReservada,
		Modelos:             modelos,
	}
	for _, item := range resumen {
		if item.Pool.ID == pool.ID {
			detail.SesionesActivas = item.SesionesActivas
			detail.CapacidadDisponible = item.CapacidadDisponible
			break
		}
	}
	return detail, nil
}

func (s *Service) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return s.store.ListPoolModels(slug)
}

func (s *Service) SavePool(pool *db.PoolCapacidad) (int64, error) {
	return s.store.SavePool(pool)
}

func (s *Service) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return s.store.SavePoolModel(poolSlug, modelo)
}

func (s *Service) SeedInitialModels() error {
	return s.store.SeedInitialModels()
}

func (s *Service) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return s.store.ListModelPolicies(scopeTipo, scopeRef, activa)
}

func (s *Service) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	return s.store.SaveModelPolicy(policy)
}

func (s *Service) SeedInitialModelPolicies() error {
	return s.store.SeedInitialModelPolicies()
}

func (s *Service) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	if input.ProyectoSlug != "" && input.Fase == "" && s.phaseProvider != nil {
		fase, err := s.phaseProvider.GetActivePhase(input.ProyectoSlug)
		if err == nil && fase != "" {
			input.Fase = fase
		}
	}
	return s.store.ResolveModelPolicy(input)
}

type Repository struct{}

func (Repository) ListPoolsSummary(activo *bool) ([]*db.PoolCapacidadResumen, error) {
	return db.ListarPoolsResumen(activo)
}

func (Repository) GetPool(slug string) (*db.PoolCapacidad, error) {
	return db.GetPool(slug)
}

func (Repository) ListPoolModels(slug string) ([]*db.PoolModelo, error) {
	return db.ListarModelosPool(slug)
}

func (Repository) SavePool(pool *db.PoolCapacidad) (int64, error) {
	return db.GuardarPool(pool)
}

func (Repository) SavePoolModel(poolSlug string, modelo *db.PoolModelo) (int64, error) {
	return db.GuardarPoolModelo(poolSlug, modelo)
}

func (Repository) SeedInitialModels() error {
	return db.SeedModelosIniciales()
}

func (Repository) ListModelPolicies(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, error) {
	return db.ListarPoliticasModelo(scopeTipo, scopeRef, activa)
}

func (Repository) SaveModelPolicy(policy *db.PoliticaModelo) (int64, error) {
	return db.GuardarPoliticaModelo(policy)
}

func (Repository) SeedInitialModelPolicies() error {
	return db.SeedPoliticasModeloIniciales()
}

func (Repository) ResolveModelPolicy(input db.ResolverPoliticaInput) (*db.ResolucionModelo, error) {
	return db.ResolverPoliticaModelo(input)
}
