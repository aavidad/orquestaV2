package gobernanzaapp

import "orquesta/db"

type Repository struct{}

func (Repository) ListRules(tipoAgente string, activa *bool) ([]*db.Regla, error) {
	return db.ListarReglas(tipoAgente, activa)
}

func (Repository) SaveRule(r *db.Regla) (int64, error) {
	return db.CrearRegla("", r)
}

func (Repository) ListSkills(tipoAgente string, activa *bool) ([]*db.Skill, error) {
	return db.ListarSkills(tipoAgente, activa)
}

func (Repository) SaveSkill(s *db.Skill) (int64, error) {
	return db.CrearSkill("", s)
}

func (Repository) ListWorkflows(tipoAgente string, activo *bool) ([]*db.Workflow, error) {
	return db.ListarWorkflows(tipoAgente, activo)
}

func (Repository) SaveWorkflow(w *db.Workflow) (int64, error) {
	return db.CrearWorkflow("", w)
}

func (Repository) ListOverrides(scopeTipo, scopeRef, tipoAgente, entidad string) ([]*db.GovernanceOverride, error) {
	return db.ListarGovernanceOverrides(scopeTipo, scopeRef, tipoAgente, entidad)
}

func (Repository) SaveOverride(actor string, item *db.GovernanceOverride) (int64, error) {
	return db.GuardarGovernanceOverride(actor, item)
}

func (Repository) ResolveCatalogForContext(tipoAgente string, proyectoID *int64, agente string) (*db.GovernanceCatalog, error) {
	return db.ResolveGovernanceCatalogForContext(tipoAgente, proyectoID, agente)
}

func (Repository) GetProject(ref string) (*db.Proyecto, error) {
	return db.GetProyecto(ref)
}

func (Repository) GetAgent(nombre string) (*db.Agente, error) {
	return db.GetAgente(nombre)
}
