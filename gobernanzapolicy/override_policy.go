package gobernanzapolicy

import (
	"fmt"
	"strings"
)

const (
	ScopeProject = "proyecto"
	ScopeAgent   = "agente"

	EntityRule     = "regla"
	EntitySkill    = "skill"
	EntityWorkflow = "workflow"

	ActionEnable  = "enable"
	ActionDisable = "disable"
)

type OverrideSpec struct {
	ScopeType string
	Entity    string
	Action    string
}

func NormalizeOverrideSpec(scopeType, entity, action string) (OverrideSpec, error) {
	spec := OverrideSpec{
		ScopeType: strings.TrimSpace(scopeType),
		Entity:    strings.TrimSpace(entity),
		Action:    strings.TrimSpace(action),
	}
	if !validScope(spec.ScopeType) {
		return OverrideSpec{}, fmt.Errorf("scope_tipo invalido: %s", spec.ScopeType)
	}
	if !validEntity(spec.Entity) {
		return OverrideSpec{}, fmt.Errorf("entidad invalida: %s", spec.Entity)
	}
	if !validAction(spec.Action) {
		return OverrideSpec{}, fmt.Errorf("accion invalida: %s", spec.Action)
	}
	return spec, nil
}

func validScope(scope string) bool {
	switch scope {
	case ScopeProject, ScopeAgent:
		return true
	default:
		return false
	}
}

func validEntity(entity string) bool {
	switch entity {
	case EntityRule, EntitySkill, EntityWorkflow:
		return true
	default:
		return false
	}
}

func validAction(action string) bool {
	switch action {
	case ActionEnable, ActionDisable:
		return true
	default:
		return false
	}
}
