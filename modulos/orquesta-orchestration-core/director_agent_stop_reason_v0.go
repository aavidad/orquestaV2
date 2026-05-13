package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type directorAgentStopReasonV0 struct {
	ReasonCode string
	Source     string
	Ref        string
}

func applyDirectorAgentStopReasonsV0(
	stats *DirectorRunStatsV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) {
	if stats == nil || len(stats.Agents) == 0 {
		return
	}
	stopRequests := latestDirectorStopRequestByAgentV0(run.AgentStopRequests)
	assessments := latestDirectorStopAssessmentByAgentV0(run.AgentAssessments)
	for index := range stats.Agents {
		if !stats.Agents[index].StopRequested && !stats.Agents[index].StopConfirmed {
			continue
		}
		agentRef := stats.Agents[index].AgentRequestID
		reason, ok := stopRequests[agentRef]
		if !ok {
			reason, ok = assessments[agentRef]
		}
		if !ok {
			continue
		}
		stats.Agents[index].StopReasonCode = reason.ReasonCode
		stats.Agents[index].StopReasonSource = reason.Source
		stats.Agents[index].StopReasonRef = reason.Ref
	}
}

func latestDirectorStopRequestByAgentV0(
	projections []string,
) map[string]directorAgentStopReasonV0 {
	latest := map[string]directorAgentStopReasonV0{}
	for _, value := range projections {
		projection, ok := orquestacoreworkflow.ParseAgentStopRequestProjectionV0(value)
		if !ok {
			continue
		}
		agentRef := strings.TrimSpace(projection.AgentRequestID)
		reasonCode := strings.TrimSpace(projection.ReasonCode)
		if agentRef == "" || reasonCode == "" {
			continue
		}
		latest[agentRef] = directorAgentStopReasonV0{
			ReasonCode: reasonCode,
			Source:     DirectorAgentStopReasonSourceStopRequestV0,
			Ref:        strings.TrimSpace(value),
		}
	}
	return latest
}

func latestDirectorStopAssessmentByAgentV0(
	projections []string,
) map[string]directorAgentStopReasonV0 {
	latest := map[string]directorAgentStopReasonV0{}
	for _, value := range projections {
		assessment, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(value)
		if !ok || strings.TrimSpace(assessment.Action) != orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
			continue
		}
		agentRef := strings.TrimSpace(assessment.AgentRequestID)
		reasonCode := strings.TrimSpace(assessment.Verdict)
		if agentRef == "" || reasonCode == "" {
			continue
		}
		latest[agentRef] = directorAgentStopReasonV0{
			ReasonCode: reasonCode,
			Source:     DirectorAgentStopReasonSourceAssessmentV0,
			Ref:        strings.TrimSpace(assessment.AssessmentRef),
		}
	}
	return latest
}
