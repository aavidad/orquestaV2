package db

import (
	"encoding/json"
)

const MailboxKindSkillsRefresh = "skills_refresh"

func notificarRefreshSkillCatalogo(actor string, skill *Skill, motivo string) {
	if skill == nil {
		return
	}
	sesiones, err := ListarSesionesActivasOperativas()
	if err != nil {
		Audit(actor, "skill_refresh_error", "skill", skill.ID, err.Error())
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"skill_id":            skill.ID,
		"tipo_agente":         skill.TipoAgente,
		"nombre":              skill.Nombre,
		"motivo":              motivo,
		"origen":              skill.Origen,
		"nivel_riesgo":        skill.NivelRiesgo,
		"requiere_aprobacion": skill.RequiereAprobacion,
		"activa":              skill.Activa,
	})

	seen := map[string]struct{}{}
	for _, sesion := range sesiones {
		if sesion == nil {
			continue
		}
		agente, err := GetAgente(sesion.Agente)
		if err != nil || agente == nil || agente.Rol != skill.TipoAgente {
			continue
		}
		if _, ok := seen[sesion.Agente]; ok {
			continue
		}
		seen[sesion.Agente] = struct{}{}
		if _, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
			FromAgente:  actor,
			ToAgente:    sesion.Agente,
			ProyectoID:  sesion.ProyectoID,
			Kind:        MailboxKindSkillsRefresh,
			PayloadJSON: string(payload),
		}); err != nil {
			Audit(actor, "skill_refresh_error", "skill", skill.ID, err.Error())
		}
	}
}

func NotificarRefreshSkillCatalogo(actor string, skill *Skill, motivo string) {
	notificarRefreshSkillCatalogo(actor, skill, motivo)
}
