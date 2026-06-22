package orquestaappcodexstack

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const assessmentReplanMaxTerminalFailedFollowupsV0 = 3

func assessmentReplanFollowupAgentRefsV0(projection string) []string {
	const marker = "#followups:"
	index := strings.Index(projection, marker)
	if index < 0 {
		return nil
	}
	value := projection[index+len(marker):]
	if next := strings.Index(value, "#"); next >= 0 {
		value = value[:next]
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '+' || r == ',' || r == ';' || r == ' '
	})
	refs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "agent-ref-") {
			refs = append(refs, part)
		}
	}
	return refs
}

func assessmentReplanFollowupStillBlocksTaskV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	agentRef = strings.TrimSpace(agentRef)
	if agentRef == "" {
		return false
	}
	if stringInSetV0(run.DeliveredAgents, agentRef) {
		return true
	}
	if stringInSetV0(run.FailedAgents, agentRef) ||
		stringInSetV0(run.LostAgents, agentRef) ||
		stringInSetV0(run.StoppedAgents, agentRef) ||
		stringInSetV0(run.ConfirmedStoppedAgents, agentRef) {
		return false
	}
	return stringInSetV0(run.Agents, agentRef) ||
		stringInSetV0(run.StartedAgents, agentRef)
}

func assessmentReplanTaskTerminalFailedFollowupCountV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	taskRef string,
) int {
	taskRef = strings.TrimSpace(taskRef)
	if taskRef == "" {
		return 0
	}
	needle := "#task:" + taskRef + "#action:" +
		string(orquestacoreworkflow.ReplanDecisionActionReplaceAgentV0)
	count := 0
	for _, projection := range run.ReplanDecisions {
		projection = strings.TrimSpace(projection)
		if !strings.Contains(projection, needle) {
			continue
		}
		for _, followupRef := range assessmentReplanFollowupAgentRefsV0(projection) {
			if assessmentReplanFollowupTerminalFailedV0(run, followupRef) {
				count++
			}
		}
	}
	return count
}

func assessmentReplanFollowupTerminalFailedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	agentRef string,
) bool {
	agentRef = strings.TrimSpace(agentRef)
	if agentRef == "" || stringInSetV0(run.DeliveredAgents, agentRef) {
		return false
	}
	return stringInSetV0(run.FailedAgents, agentRef) ||
		stringInSetV0(run.LostAgents, agentRef) ||
		stringInSetV0(run.StoppedAgents, agentRef) ||
		stringInSetV0(run.ConfirmedStoppedAgents, agentRef)
}
