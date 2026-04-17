package skillsapp

import "orquesta/db"

type Repository struct{}

func (Repository) FindEquivalentSkill(s *db.Skill) (*db.Skill, error) {
	return db.BuscarSkillEquivalente(s, 0)
}

func (Repository) CreateSkill(actor string, s *db.Skill) (int64, error) {
	return db.CrearSkill(actor, s)
}

func (Repository) Audit(actor, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(actor, accion, entidad, entidadID, detalle)
}
