package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const defaultSkillPriority = 100
const (
	skillOriginBuiltin    = "builtin"
	skillOriginLocal      = "local"
	skillOriginThirdParty = "third_party"
	skillRiskLow          = "bajo"
	skillRiskMedium       = "medio"
	skillRiskHigh         = "alto"
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
	s.TipoAgente = strings.TrimSpace(s.TipoAgente)
	s.Nombre = strings.TrimSpace(s.Nombre)
	s.Descripcion = strings.TrimSpace(s.Descripcion)
	s.CuandoUsar = strings.TrimSpace(s.CuandoUsar)
	s.Escenario = strings.ToLower(strings.TrimSpace(s.Escenario))
	s.Origen = strings.ToLower(strings.TrimSpace(s.Origen))
	s.NivelRiesgo = strings.ToLower(strings.TrimSpace(s.NivelRiesgo))
	if s.Prioridad <= 0 {
		s.Prioridad = defaultSkillPriority
	}
	if s.Origen == "" {
		s.Origen = skillOriginBuiltin
	}
	switch s.Origen {
	case skillOriginBuiltin, skillOriginLocal, skillOriginThirdParty:
	default:
		return fmt.Errorf("origen invalido: %s", s.Origen)
	}
	if s.NivelRiesgo == "" {
		if s.Origen == skillOriginBuiltin {
			s.NivelRiesgo = skillRiskLow
		} else {
			s.NivelRiesgo = skillRiskMedium
		}
	}
	switch s.NivelRiesgo {
	case skillRiskLow, skillRiskMedium, skillRiskHigh:
	default:
		return fmt.Errorf("nivel_riesgo invalido: %s", s.NivelRiesgo)
	}

	var err error
	s.AliasesJSON, err = normalizarListaJSON(s.AliasesJSON)
	if err != nil {
		return fmt.Errorf("aliases_json invalido: %w", err)
	}
	s.HerramientasJSON, err = normalizarListaJSON(s.HerramientasJSON)
	if err != nil {
		return fmt.Errorf("herramientas_json invalido: %w", err)
	}
	if s.TipoAgente == "" || s.Nombre == "" {
		return fmt.Errorf("tipo_agente y nombre son obligatorios")
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizarListaJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return "", err
	}
	normalizados := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		canonical := strings.ToLower(trimmed)
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		normalizados = append(normalizados, canonical)
	}
	sort.Strings(normalizados)
	data, err := json.Marshal(normalizados)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseListaJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, strings.ToLower(trimmed))
		}
	}
	sort.Strings(out)
	return out
}

func skillIdentityKeys(s *Skill) map[string]struct{} {
	keys := map[string]struct{}{}
	if s == nil {
		return keys
	}
	for _, raw := range append([]string{s.Nombre}, parseListaJSON(s.AliasesJSON)...) {
		key := normalizarClaveSkill(raw)
		if key == "" {
			continue
		}
		keys[key] = struct{}{}
	}
	return keys
}

func normalizarClaveSkill(v string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(v)), func(r rune) bool {
		switch r {
		case ' ', '-', '_', '/', '\\', '.':
			return true
		default:
			return false
		}
	})
	return strings.Join(parts, "")
}

func normalizarTextoComparacion(v string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(v))), " ")
}

func listasIguales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sameSkillNaturalKey(a, b *Skill) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.TipoAgente), strings.TrimSpace(b.TipoAgente)) &&
		strings.EqualFold(strings.TrimSpace(a.Nombre), strings.TrimSpace(b.Nombre))
}

func skillsSonEquivalentes(a, b *Skill) bool {
	if a == nil || b == nil {
		return false
	}
	keysA := skillIdentityKeys(a)
	for key := range skillIdentityKeys(b) {
		if _, ok := keysA[key]; ok && key != "" {
			return true
		}
	}
	if strings.TrimSpace(a.Escenario) != "" &&
		strings.EqualFold(strings.TrimSpace(a.Escenario), strings.TrimSpace(b.Escenario)) &&
		listasIguales(parseListaJSON(a.HerramientasJSON), parseListaJSON(b.HerramientasJSON)) &&
		normalizarTextoComparacion(a.CuandoUsar) != "" &&
		normalizarTextoComparacion(a.CuandoUsar) == normalizarTextoComparacion(b.CuandoUsar) {
		return true
	}
	return false
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
	if s == nil {
		return false
	}
	return s.Origen == skillOriginLocal || s.Origen == skillOriginThirdParty
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
