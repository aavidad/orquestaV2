package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func noSignalDirectorAgentRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	byAgent map[string]AgentProgressObservationV0,
	byAssessmentAgent map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) []string {
	missing := make([]string, 0)
	failed := autonomousStringSetV0(run.FailedAgents)
	lost := autonomousStringSetV0(run.LostAgents)
	stopRequested := autonomousStringSetV0(run.StoppedAgents)
	stopConfirmed := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	reflected := reflectedDirectorAgentSetV0(run)
	for _, agentRef := range compactStringsV0(run.StartedAgents) {
		if failed[agentRef] || lost[agentRef] || stopRequested[agentRef] ||
			stopConfirmed[agentRef] || reflected[agentRef] {
			continue
		}
		if _, ok := byAssessmentAgent[agentRef]; ok {
			continue
		}
		if _, ok := byAgent[agentRef]; !ok {
			missing = append(missing, agentRef)
		}
	}
	return missing
}

func latestDirectorAssessmentByAgentV0(
	assessments []string,
) map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0 {
	latest := map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0{}
	for _, value := range assessments {
		assessment, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(value)
		if !ok {
			continue
		}
		agentRef := strings.TrimSpace(assessment.AgentRequestID)
		if agentRef != "" {
			latest[agentRef] = assessment
		}
	}
	return latest
}

func latestDirectorAssessmentByTaskV0(
	assessments []string,
) map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0 {
	latest := map[string]orquestacoreworkflow.AgentWorkAssessmentProjectionV0{}
	for _, value := range assessments {
		assessment, ok := orquestacoreworkflow.ParseAgentAssessmentProjectionV0(value)
		if !ok {
			continue
		}
		taskRef := strings.TrimSpace(assessment.TaskRef)
		if taskRef != "" {
			latest[taskRef] = assessment
		}
	}
	return latest
}

func directorAssessmentProgressStatusV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) string {
	switch strings.TrimSpace(assessment.Verdict) {
	case orquestacoreworkflow.AgentAssessmentVerdictLoopDetectedV0:
		return string(orquestaruntime.AgentLoopDetectedV0)
	case orquestacoreworkflow.AgentAssessmentVerdictGarbageV0,
		orquestacoreworkflow.AgentAssessmentVerdictNeedsRevisionV0,
		orquestacoreworkflow.AgentAssessmentVerdictCapacityLimitedV0:
		return string(orquestaruntime.AgentStalledV0)
	}
	if strings.TrimSpace(assessment.Action) == orquestacoreworkflow.AgentAssessmentActionStopAgentV0 {
		return string(orquestaruntime.AgentStoppedV0)
	}
	return string(orquestaruntime.AgentProgressingV0)
}

func directorTaskStatusFromAssessmentV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) string {
	switch directorAssessmentProgressStatusV0(assessment) {
	case string(orquestaruntime.AgentLoopDetectedV0):
		return DirectorTaskProgressLoopDetectedV0
	case string(orquestaruntime.AgentStoppedV0):
		return DirectorTaskProgressStoppedV0
	case string(orquestaruntime.AgentStalledV0):
		return DirectorTaskProgressStalledV0
	default:
		return DirectorTaskProgressInProgressV0
	}
}

func directorAssessmentDecisionRequiredV0(
	assessment orquestacoreworkflow.AgentWorkAssessmentProjectionV0,
) bool {
	switch strings.TrimSpace(assessment.Action) {
	case orquestacoreworkflow.AgentAssessmentActionAskDirectorV0,
		orquestacoreworkflow.AgentAssessmentActionRequestRevisionV0,
		orquestacoreworkflow.AgentAssessmentActionStopAgentV0:
		return true
	default:
		return false
	}
}
