package orquestadirectoragent

import (
	"strings"

	orquestarails "orquesta/modulos/orquesta-rails"
)

const directorAgentDecisionRailBoundaryV0 = "director_agent_decision"

func directorAgentDecisionFieldHasForbiddenDetailV0(field string, values ...string) bool {
	return orquestarails.ValuesContainOperationalSensitiveDetailForFieldV0(
		directorAgentDecisionRailBoundaryV0,
		directorAgentDecisionRailFieldV0(field),
		values,
	)
}

func directorAgentDecisionRailFieldV0(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return "*"
	}
	if index := strings.LastIndex(field, "."); index >= 0 && index+1 < len(field) {
		return field[index+1:]
	}
	return field
}
