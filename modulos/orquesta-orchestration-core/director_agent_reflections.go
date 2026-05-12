package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const directorAgentProjectionSeparatorV0 = "#agent:"

func reflectedDirectorAgentSetV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) map[string]bool {
	started := compactStringsV0(run.StartedAgents)
	out := map[string]bool{}
	for _, projection := range run.PhaseArtifacts {
		markReflectedDirectorAgentV0(out, started, projection)
	}
	for _, deliveryRef := range run.Deliveries {
		markReflectedDirectorAgentV0(out, started, deliveryRef)
	}
	return out
}

func markReflectedDirectorAgentV0(
	out map[string]bool,
	startedAgents []string,
	projection string,
) {
	projection = strings.TrimSpace(projection)
	if projection == "" {
		return
	}
	if agentRef, ok := directorProjectionAgentRefV0(projection); ok {
		out[agentRef] = true
		return
	}
	for _, agentRef := range startedAgents {
		if directorLooseProjectionMentionsAgentV0(projection, agentRef) {
			out[agentRef] = true
		}
	}
}

func directorProjectionAgentRefV0(projection string) (string, bool) {
	_, tail, ok := strings.Cut(strings.TrimSpace(projection), directorAgentProjectionSeparatorV0)
	if !ok {
		return "", false
	}
	if before, _, hasNext := strings.Cut(tail, "#"); hasNext {
		tail = before
	}
	agentRef := strings.TrimSpace(tail)
	return agentRef, agentRef != ""
}

func directorLooseProjectionMentionsAgentV0(projection string, agentRef string) bool {
	projection = strings.TrimSpace(projection)
	agentRef = strings.TrimSpace(agentRef)
	return projection != "" && agentRef != "" &&
		(projection == agentRef || strings.Contains(projection, agentRef))
}
