package db

import "fmt"

func EliminarSkill(actor string, id int64) error {
	skill, err := GetSkill(id)
	if err != nil {
		return err
	}
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM skills WHERE id=?`, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	Audit(actor, "eliminar_skill", "skill", id, fmt.Sprintf("%s:%s", skill.TipoAgente, skill.Nombre))
	notificarRefreshSkillCatalogo(actor, skill, "borrar")
	return nil
}
