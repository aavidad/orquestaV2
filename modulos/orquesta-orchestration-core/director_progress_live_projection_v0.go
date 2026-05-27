package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func liveDirectorTaskAgentsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) map[string]string {
	started := autonomousStringSetV0(run.StartedAgents)
	failed := autonomousStringSetV0(run.FailedAgents)
	lost := autonomousStringSetV0(run.LostAgents)
	stopConfirmed := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	reflected := reflectedDirectorAgentSetV0(run)
	liveByTask := map[string]string{}
	for _, taskRef := range compactStringsV0(run.Tasks) {
		agentRef := WorkflowTaskAgentRequestRefV0(taskRef)
		if started[agentRef] && !failed[agentRef] && !lost[agentRef] &&
			!stopConfirmed[agentRef] && !reflected[agentRef] {
			liveByTask[strings.TrimSpace(taskRef)] = agentRef
		}
	}
	return liveByTask
}

func liveDirectorTaskProgressV0(taskRef string, agentRef string) DirectorTaskProgressV0 {
	return DirectorTaskProgressV0{
		TaskRef:        strings.TrimSpace(taskRef),
		Status:         DirectorTaskProgressInProgressV0,
		AgentRequestID: strings.TrimSpace(agentRef),
		ProgressStatus: string(orquestaruntime.AgentProgressingV0),
	}
}

func directorProgressPercentWithLiveWorkV0(
	total int,
	resolved int,
	tasks []DirectorTaskProgressV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) int {
	percent := directorProgressPercentV0(total, resolved)
	if percent != 0 || total <= 0 || !hasLiveDirectorRunProgressV0(tasks, run) {
		return percent
	}
	return 1
}

func hasLiveDirectorRunProgressV0(
	tasks []DirectorTaskProgressV0,
	run orquestacoreworkflow.OrchestrationRunV0,
) bool {
	return hasLiveDirectorTaskProgressV0(tasks) || hasLiveDirectorAgentProgressV0(run)
}

func hasLiveDirectorTaskProgressV0(tasks []DirectorTaskProgressV0) bool {
	for _, task := range tasks {
		switch strings.TrimSpace(task.Status) {
		case DirectorTaskProgressInProgressV0,
			DirectorTaskProgressStalledV0,
			DirectorTaskProgressLoopDetectedV0:
			return true
		}
	}
	return false
}

func hasLiveDirectorAgentProgressV0(run orquestacoreworkflow.OrchestrationRunV0) bool {
	started := autonomousStringSetV0(run.StartedAgents)
	if len(started) == 0 {
		return false
	}
	failed := autonomousStringSetV0(run.FailedAgents)
	lost := autonomousStringSetV0(run.LostAgents)
	stopConfirmed := autonomousStringSetV0(run.ConfirmedStoppedAgents)
	reflected := reflectedDirectorAgentSetV0(run)
	for agentRef := range started {
		if !failed[agentRef] && !lost[agentRef] && !stopConfirmed[agentRef] &&
			!reflected[agentRef] {
			return true
		}
	}
	return false
}
