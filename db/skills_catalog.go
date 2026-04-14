package db

import (
	"encoding/json"
	"fmt"

	"orquesta/skillspolicy"
)

const defaultSkillPriority = skillspolicy.DefaultPriority

const (
	skillOriginBuiltin    = skillspolicy.OriginBuiltin
	skillOriginLocal      = skillspolicy.OriginLocal
	skillOriginThirdParty = skillspolicy.OriginThirdParty
	skillRiskLow          = skillspolicy.RiskLow
	skillRiskMedium       = skillspolicy.RiskMedium
	skillRiskHigh         = skillspolicy.RiskHigh
)

type SkillEquivalenteError struct {
	Existente *Skill
}

func (e *SkillEquivalenteError) Error() string {
	if e == nil || e.Existente == nil {
		return "ya existe una skill equivalente"
	}
	return fmt.Sprintf("ya existe una skill equivalente: #%d %s", e.Existente.ID, e.Existente.Nombre)
}

func normalizarSkillParaCatalogo(s *Skill) error {
	if s == nil {
		return fmt.Errorf("skill nula")
	}
	snapshot := skillSnapshotFromDB(s)
	if err := skillspolicy.Normalize(snapshot); err != nil {
		return err
	}
	applySkillSnapshotToDB(s, snapshot)
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizarListaJSON(raw string) (string, error) {
	return skillspolicy.NormalizeListJSON(raw)
}

func NormalizarListaJSONPublic(items []string) (string, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return normalizarListaJSON(string(data))
}

func parseListaJSON(raw string) []string {
	return skillspolicy.ParseListJSON(raw)
}

func sameSkillNaturalKey(a, b *Skill) bool {
	return skillspolicy.SameNaturalKey(skillSnapshotFromDB(a), skillSnapshotFromDB(b))
}

func skillsSonEquivalentes(a, b *Skill) bool {
	return skillspolicy.SkillsEquivalent(skillSnapshotFromDB(a), skillSnapshotFromDB(b))
}

func BuscarSkillEquivalente(s *Skill, excludeID int64) (*Skill, error) {
	if err := normalizarSkillParaCatalogo(s); err != nil {
		return nil, err
	}
	items, err := ListarSkills(s.TipoAgente, nil)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item == nil || item.ID == excludeID {
			continue
		}
		if skillsSonEquivalentes(item, s) {
			return item, nil
		}
	}
	return nil, nil
}

func skillEsExterna(s *Skill) bool {
	return skillspolicy.IsExternal(skillSnapshotFromDB(s))
}

func aplicarPoliticaSeguridadSkill(actor string, s *Skill, activacionExplicita bool) error {
	if err := normalizarSkillParaCatalogo(s); err != nil {
		return err
	}
	if !skillEsExterna(s) {
		s.RequiereAprobacion = false
		return nil
	}

	rolActor, habilitado, err := rolAgente(actor)
	if err != nil {
		return err
	}
	if !habilitado {
		return fmt.Errorf("el actor %s está retirado", actor)
	}

	if s.NivelRiesgo == skillRiskLow {
		s.NivelRiesgo = skillRiskMedium
	}
	if !activacionExplicita {
		s.Activa = false
		s.RequiereAprobacion = true
		return nil
	}
	if rolActor != "admin" {
		return fmt.Errorf("solo admin puede activar skills externas")
	}
	s.RequiereAprobacion = false
	return nil
}

func skillSnapshotFromDB(s *Skill) *skillspolicy.SkillSnapshot {
	if s == nil {
		return nil
	}
	return &skillspolicy.SkillSnapshot{
		AgentType:        s.TipoAgente,
		Name:             s.Nombre,
		Description:      s.Descripcion,
		WhenToUse:        s.CuandoUsar,
		Scenario:         s.Escenario,
		Priority:         s.Prioridad,
		AliasesJSON:      s.AliasesJSON,
		ToolsJSON:        s.HerramientasJSON,
		Origin:           s.Origen,
		RiskLevel:        s.NivelRiesgo,
		RequiresApproval: s.RequiereAprobacion,
		Active:           s.Activa,
	}
}

func applySkillSnapshotToDB(dst *Skill, src *skillspolicy.SkillSnapshot) {
	if dst == nil || src == nil {
		return
	}
	dst.TipoAgente = src.AgentType
	dst.Nombre = src.Name
	dst.Descripcion = src.Description
	dst.CuandoUsar = src.WhenToUse
	dst.Escenario = src.Scenario
	dst.Prioridad = src.Priority
	dst.AliasesJSON = src.AliasesJSON
	dst.HerramientasJSON = src.ToolsJSON
	dst.Origen = src.Origin
	dst.NivelRiesgo = src.RiskLevel
	dst.RequiereAprobacion = src.RequiresApproval
	dst.Activa = src.Active
}
