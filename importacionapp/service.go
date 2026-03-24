package importacionapp

import "orquesta/db"

type Store interface {
	ImportLegacyProposalHistory() error
	ImportWave2Tasks() error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ImportHistory() error {
	if err := s.store.ImportLegacyProposalHistory(); err != nil {
		return err
	}
	return s.store.ImportWave2Tasks()
}

type Repository struct{}

func (Repository) ImportLegacyProposalHistory() error {
	return db.ImportarHistorialOPs()
}

func (Repository) ImportWave2Tasks() error {
	return db.ImportarTareasOla2()
}
