package proposalapp

import (
	"fmt"
	"strings"

	"orquesta/db"
)

type Store interface {
	ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error)
	GetProposal(codigo string) (*db.Propuesta, error)
	CreateProposal(p *db.Propuesta) (int64, error)
	CloseProposal(codigo, estado, agente string) error
	Vote(propuestaID int64, agente string, posicion db.PosicionVoto, comentario string) (bool, error)
	CountVotes(propuestaID int64) (int, int, int, int, error)
	ListVotes(propuestaID int64) ([]*db.Voto, error)
	ListAgents() ([]*db.Agente, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type ProposalDetail struct {
	Proposal *db.Propuesta
	Votes    []*db.Voto
}

type VoteResult struct {
	Proposal   *db.Propuesta
	Consenso   bool
	Acuerdo    int
	Desacuerdo int
	Abstencion int
	Pendiente  int
}

type CreateProposalInput struct {
	Codigo       string
	Titulo       string
	Descripcion  string
	Tipo         string
	PropuestoPor string
	Distribuidor string
}

func (s *Service) List(estado *db.EstadoPropuesta) ([]*db.Propuesta, error) {
	return s.store.ListProposals(estado)
}

func (s *Service) GetDetail(codigo string) (*ProposalDetail, error) {
	p, err := s.store.GetProposal(strings.TrimSpace(codigo))
	if err != nil {
		return nil, err
	}
	votos, err := s.store.ListVotes(p.ID)
	if err != nil {
		return nil, err
	}
	return &ProposalDetail{Proposal: p, Votes: votos}, nil
}

func (s *Service) Create(input CreateProposalInput) (int64, *db.Propuesta, error) {
	p := &db.Propuesta{
		Codigo:       strings.TrimSpace(input.Codigo),
		Titulo:       strings.TrimSpace(input.Titulo),
		Descripcion:  strings.TrimSpace(input.Descripcion),
		Tipo:         strings.TrimSpace(input.Tipo),
		PropuestoPor: strings.TrimSpace(input.PropuestoPor),
		Distribuidor: strings.TrimSpace(input.Distribuidor),
	}
	id, err := s.store.CreateProposal(p)
	return id, p, err
}

func (s *Service) Close(codigo, estado, agente string) error {
	return s.store.CloseProposal(strings.TrimSpace(codigo), strings.TrimSpace(estado), strings.TrimSpace(agente))
}

func (s *Service) Vote(codigo, agente string, posicion db.PosicionVoto, comentario string) error {
	_, err := s.VoteDetail(codigo, agente, posicion, comentario)
	return err
}

func (s *Service) VoteDetail(codigo, agente string, posicion db.PosicionVoto, comentario string) (*VoteResult, error) {
	p, err := s.store.GetProposal(strings.TrimSpace(codigo))
	if err != nil {
		return nil, err
	}
	if p.Estado != db.PropuestaAbierta {
		return nil, fmt.Errorf("la propuesta %s ya está cerrada (%s)", p.Codigo, p.Estado)
	}
	consenso, err := s.store.Vote(p.ID, strings.TrimSpace(agente), posicion, strings.TrimSpace(comentario))
	if err != nil {
		return nil, err
	}
	acuerdo, desacuerdo, abstencion, pendiente, err := s.store.CountVotes(p.ID)
	if err != nil {
		return nil, err
	}
	return &VoteResult{
		Proposal:   p,
		Consenso:   consenso,
		Acuerdo:    acuerdo,
		Desacuerdo: desacuerdo,
		Abstencion: abstencion,
		Pendiente:  pendiente,
	}, nil
}

func (s *Service) ListAgents() ([]*db.Agente, error) {
	return s.store.ListAgents()
}

type Repository struct{}

func (Repository) ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error) {
	return db.ListarPropuestas(estado)
}

func (Repository) GetProposal(codigo string) (*db.Propuesta, error) {
	return db.GetPropuesta(codigo)
}

func (Repository) CreateProposal(p *db.Propuesta) (int64, error) {
	return db.CrearPropuesta(p)
}

func (Repository) CloseProposal(codigo, estado, agente string) error {
	return db.CerrarPropuesta(codigo, estado, agente)
}

func (Repository) Vote(propuestaID int64, agente string, posicion db.PosicionVoto, comentario string) (bool, error) {
	return db.Votar(propuestaID, agente, posicion, comentario)
}

func (Repository) ListVotes(propuestaID int64) ([]*db.Voto, error) {
	return db.VotosDePropuesta(propuestaID)
}

func (Repository) CountVotes(propuestaID int64) (int, int, int, int, error) {
	return db.ContarVotos(propuestaID)
}

func (Repository) ListAgents() ([]*db.Agente, error) {
	return db.ListarAgentes()
}
