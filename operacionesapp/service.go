package operacionesapp

import "orquesta/db"

type Store interface {
	ListAgents() ([]*db.Agente, error)
	ListConnectors() ([]*db.Conector, error)
	ListAssignments(estado, agente string) ([]*db.Asignacion, error)
	ListActiveSessions() ([]*db.SesionActiva, error)
	AuditLog(limit int) ([]db.AuditEntry, error)
	RegisterAgent(nombre, rol string) error
	RetireAgent(nombre string) error
	RehabilitateAgent(nombre string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListAgents() ([]*db.Agente, error) {
	return s.store.ListAgents()
}

func (s *Service) ListConnectors() ([]*db.Conector, error) {
	return s.store.ListConnectors()
}

func (s *Service) ListAssignments(estado, agente string) ([]*db.Asignacion, error) {
	return s.store.ListAssignments(estado, agente)
}

func (s *Service) ListActiveSessions() ([]*db.SesionActiva, error) {
	return s.store.ListActiveSessions()
}

func (s *Service) AuditLog(limit int) ([]db.AuditEntry, error) {
	return s.store.AuditLog(limit)
}

func (s *Service) RegisterAgent(nombre, rol string) error {
	return s.store.RegisterAgent(nombre, rol)
}

func (s *Service) RetireAgent(nombre string) error {
	return s.store.RetireAgent(nombre)
}

func (s *Service) RehabilitateAgent(nombre string) error {
	return s.store.RehabilitateAgent(nombre)
}
