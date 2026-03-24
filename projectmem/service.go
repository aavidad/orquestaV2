package projectmem

import (
	"strings"

	"orquesta/db"
)

type Store interface {
	ResolveProyectoIDBySlug(slug string) (*int64, error)
	ListProjects(activo *bool) ([]*db.Proyecto, error)
	GetProjectBySlug(slug string) (*db.Proyecto, error)
	ListVoteHistoryByProjectID(proyectoID int64) ([]*db.HistorialVotacionProyecto, error)
	SaveDecision(d *db.DecisionProyecto) (int64, error)
	ListDecisionsByProjectID(proyectoID int64) ([]*db.DecisionProyecto, error)
	SaveExternalDoc(doc *db.DocumentoExterno) (int64, error)
	ListExternalDocsByProjectID(proyectoID int64) ([]*db.DocumentoExterno, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type CreateDecisionInput struct {
	ProyectoSlug string
	Categoria    string
	Titulo       string
	Solucion     string
	Motivo       string
	Alternativas string
	Impacto      string
	Estado       string
	PropuestaID  *int64
	TareaID      *int64
	MetadataJSON string
}

type CreateExternalDocInput struct {
	ProyectoSlug  string
	TipoDocumento string
	Titulo        string
	RutaRef       string
	Resumen       string
	Estado        string
	Fuente        string
	PropuestaID   *int64
	TareaID       *int64
	MetadataJSON  string
}

type ProjectOverview struct {
	Proyecto   *db.Proyecto                    `json:"proyecto"`
	Votaciones []*db.HistorialVotacionProyecto `json:"votaciones"`
	Decisiones []*db.DecisionProyecto          `json:"decisiones"`
	Documentos []*db.DocumentoExterno          `json:"documentos"`
}

func (s *Service) ListProjects(activo *bool) ([]*db.Proyecto, error) {
	return s.store.ListProjects(activo)
}

func (s *Service) GetProject(slug string) (*db.Proyecto, error) {
	return s.store.GetProjectBySlug(strings.TrimSpace(slug))
}

func (s *Service) ListVoteHistory(slug string) ([]*db.HistorialVotacionProyecto, error) {
	proyectoID, err := s.requireProjectID(slug)
	if err != nil {
		return nil, err
	}
	return s.store.ListVoteHistoryByProjectID(*proyectoID)
}

func (s *Service) CreateDecision(input CreateDecisionInput) (int64, error) {
	proyectoID, err := s.requireProjectID(input.ProyectoSlug)
	if err != nil {
		return 0, err
	}
	return s.store.SaveDecision(&db.DecisionProyecto{
		ProyectoID:   *proyectoID,
		Categoria:    input.Categoria,
		Titulo:       input.Titulo,
		Solucion:     input.Solucion,
		Motivo:       input.Motivo,
		Alternativas: input.Alternativas,
		Impacto:      input.Impacto,
		Estado:       input.Estado,
		PropuestaID:  input.PropuestaID,
		TareaID:      input.TareaID,
		MetadataJSON: input.MetadataJSON,
	})
}

func (s *Service) ListDecisions(slug string) ([]*db.DecisionProyecto, error) {
	proyectoID, err := s.requireProjectID(slug)
	if err != nil {
		return nil, err
	}
	return s.store.ListDecisionsByProjectID(*proyectoID)
}

func (s *Service) CreateExternalDoc(input CreateExternalDocInput) (int64, error) {
	proyectoID, err := s.requireProjectID(input.ProyectoSlug)
	if err != nil {
		return 0, err
	}
	return s.store.SaveExternalDoc(&db.DocumentoExterno{
		ProyectoID:    *proyectoID,
		TipoDocumento: input.TipoDocumento,
		Titulo:        input.Titulo,
		RutaRef:       input.RutaRef,
		Resumen:       input.Resumen,
		Estado:        input.Estado,
		Fuente:        input.Fuente,
		PropuestaID:   input.PropuestaID,
		TareaID:       input.TareaID,
		MetadataJSON:  input.MetadataJSON,
	})
}

func (s *Service) ListExternalDocs(slug string) ([]*db.DocumentoExterno, error) {
	proyectoID, err := s.requireProjectID(slug)
	if err != nil {
		return nil, err
	}
	return s.store.ListExternalDocsByProjectID(*proyectoID)
}

func (s *Service) Overview(slug string) (*ProjectOverview, error) {
	proyecto, err := s.GetProject(slug)
	if err != nil {
		return nil, err
	}
	votaciones, err := s.ListVoteHistory(slug)
	if err != nil {
		return nil, err
	}
	decisiones, err := s.ListDecisions(slug)
	if err != nil {
		return nil, err
	}
	documentos, err := s.ListExternalDocs(slug)
	if err != nil {
		return nil, err
	}
	return &ProjectOverview{
		Proyecto:   proyecto,
		Votaciones: votaciones,
		Decisiones: decisiones,
		Documentos: documentos,
	}, nil
}

func (s *Service) requireProjectID(slug string) (*int64, error) {
	return s.store.ResolveProyectoIDBySlug(strings.TrimSpace(slug))
}
