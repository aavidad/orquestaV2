package propuestasapp

import "orquesta/db"

type Repository struct{}

func (Repository) ListProposals(estado *db.EstadoPropuesta) ([]*db.Propuesta, error) {
	return db.ListarPropuestas(estado, nil)
}

func (Repository) ListProposalsByProject(estado *db.EstadoPropuesta, proyectoID *int64) ([]*db.Propuesta, error) {
	return db.ListarPropuestas(estado, proyectoID)
}

func (Repository) ListPendingProjectVotes(agente string, proyectoID *int64) ([]*db.Propuesta, error) {
	return db.PropuestasPendientesVotoProyecto(agente, proyectoID)
}

func (Repository) GetProposal(codigo string) (*db.Propuesta, error) {
	return db.GetPropuesta(codigo)
}

func (Repository) CreateProposal(p *db.Propuesta) (int64, error) {
	return db.CrearPropuesta(p)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) CloseProposal(codigo, estado, agente string) error {
	return db.CerrarPropuesta(codigo, estado, agente)
}

func (Repository) ReopenProposal(codigo, agente string) (int, error) {
	return db.ReabrirPropuesta(codigo, agente)
}

func (Repository) RepairPendingVotes(codigo, agente string) (int, error) {
	return db.RepararVotosPendientesPropuesta(codigo, agente)
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
