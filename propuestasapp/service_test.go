package propuestasapp

import (
	"testing"

	"orquesta/db"
)

type fakeStore struct {
	proposal *db.Propuesta
	consenso bool
	counts   struct {
		acuerdo    int
		desacuerdo int
		abstencion int
		pendiente  int
	}
	voted struct {
		id         int64
		agente     string
		posicion   db.PosicionVoto
		comentario string
	}
}

func (f *fakeStore) ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error) {
	return []*db.Propuesta{f.proposal}, nil
}

func (f *fakeStore) GetProposal(codigo string) (*db.Propuesta, error) {
	return f.proposal, nil
}

func (f *fakeStore) CreateProposal(p *db.Propuesta) (int64, error) {
	f.proposal = p
	return 22, nil
}

func (f *fakeStore) CloseProposal(codigo, estado, agente string) error {
	return nil
}

func (f *fakeStore) ReopenProposal(codigo, agente string) (int, error) {
	return 0, nil
}

func (f *fakeStore) RepairPendingVotes(codigo, agente string) (int, error) {
	return 0, nil
}

func (f *fakeStore) Vote(propuestaID int64, agente string, posicion db.PosicionVoto, comentario string) (bool, error) {
	f.voted.id = propuestaID
	f.voted.agente = agente
	f.voted.posicion = posicion
	f.voted.comentario = comentario
	return false, nil
}

func (f *fakeStore) ListVotes(propuestaID int64) ([]*db.Voto, error) {
	return []*db.Voto{{PropuestaID: propuestaID, Agente: "codex2"}}, nil
}

func (f *fakeStore) CountVotes(propuestaID int64) (int, int, int, int, error) {
	return f.counts.acuerdo, f.counts.desacuerdo, f.counts.abstencion, f.counts.pendiente, nil
}

func (f *fakeStore) ListAgents() ([]*db.Agente, error) {
	return []*db.Agente{{Nombre: "codex2"}}, nil
}

func TestCreateProposal(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)
	id, p, err := svc.Create(CreateProposalInput{
		Codigo:       "OP-999",
		Titulo:       "Nueva",
		Descripcion:  "Desc",
		Tipo:         "arquitectura",
		PropuestoPor: "codex2",
		Distribuidor: "claude",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id != 22 || p.Codigo != "OP-999" {
		t.Fatalf("resultado inesperado: id=%d proposal=%+v", id, p)
	}
}

func TestVoteResuelvePropuestaYRegistraComentario(t *testing.T) {
	t.Parallel()

	store := &fakeStore{proposal: &db.Propuesta{ID: 7, Codigo: "OP-999", Estado: db.PropuestaAbierta}}
	store.counts.acuerdo = 3
	store.counts.pendiente = 1
	svc := NewService(store)
	result, err := svc.VoteDetail("OP-999", "codex2", db.VotoAcuerdo, "ok")
	if err != nil {
		t.Fatalf("VoteDetail: %v", err)
	}
	if store.voted.id != 7 || store.voted.agente != "codex2" || store.voted.posicion != db.VotoAcuerdo {
		t.Fatalf("voto inesperado: %+v", store.voted)
	}
	if result.Acuerdo != 3 || result.Pendiente != 1 || result.Proposal.Codigo != "OP-999" {
		t.Fatalf("resultado inesperado: %+v", result)
	}
}
