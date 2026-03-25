package operacionesapp

import "orquesta/db"

type Store interface {
	ListAgents() ([]*db.Agente, error)
	ListConnectors() ([]*db.Conector, error)
	ListAssignments(estado, agente string) ([]*db.Asignacion, error)
	ListActiveSessions() ([]*db.SesionActiva, error)
	GetProject(ref string) (*db.Proyecto, error)
	ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error)
	GetInspectionSession(id int64) (*db.Sesion, error)
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

type ListInspectionSessionsInput struct {
	Agente      string
	ProyectoRef string
	Estado      string
	Activa      *bool
}

func (s *Service) ListInspectionSessions(input ListInspectionSessionsInput) ([]*db.Sesion, error) {
	filtro := db.FiltroSesionesInspeccion{}
	if input.Agente != "" {
		filtro.Agente = &input.Agente
	}
	if input.ProyectoRef != "" {
		proyecto, err := s.store.GetProject(input.ProyectoRef)
		if err != nil {
			return nil, err
		}
		filtro.ProyectoID = &proyecto.ID
	}
	if input.Estado != "" {
		filtro.Estado = &input.Estado
	}
	if input.Activa != nil {
		filtro.Activa = input.Activa
	}
	return s.store.ListInspectionSessions(filtro)
}

func (s *Service) GetInspectionSession(id int64) (*db.Sesion, error) {
	return s.store.GetInspectionSession(id)
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

type Repository struct{}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}

func (Repository) ListConnectors() ([]*db.Conector, error) {
	return db.ListarConectores()
}

func (Repository) ListAssignments(estado, agente string) ([]*db.Asignacion, error) {
	return db.ListarAsignacionesOpsView(estado, agente)
}

func (Repository) ListActiveSessions() ([]*db.SesionActiva, error) {
	return db.ListarSesionesActivasOpsView()
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) ListInspectionSessions(filtro db.FiltroSesionesInspeccion) ([]*db.Sesion, error) {
	return db.ListarSesionesInspeccion(filtro)
}

func (Repository) GetInspectionSession(id int64) (*db.Sesion, error) {
	return db.GetSesionInspeccionByID(id)
}

func (Repository) AuditLog(limit int) ([]db.AuditEntry, error) {
	return db.AuditLog(limit)
}

func (Repository) RegisterAgent(nombre, rol string) error {
	return db.RegistrarAgente(nombre, rol)
}

func (Repository) RetireAgent(nombre string) error {
	return db.RetirarAgente(nombre)
}

func (Repository) RehabilitateAgent(nombre string) error {
	return db.RehabilitarAgente(nombre)
}
