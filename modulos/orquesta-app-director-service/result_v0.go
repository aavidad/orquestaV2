package orquestaappdirectorservice

import (
	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func invalidStartAppDirectorResultV0(
	request StartAppDirectorRequestV0,
	issues []orquestafactory.ValidationIssue,
) StartAppDirectorResultV0 {
	return StartAppDirectorResultV0{
		SchemaVersion:    StartAppDirectorResultSchemaVersionV0,
		Status:           StartAppDirectorStatusInvalidV0,
		CorrelationID:    request.CorrelationID,
		ValidationIssues: append([]orquestafactory.ValidationIssue(nil), issues...),
		EvidenceRefs:     []string{"evidence-ref-app-director-service-invalid-v0"},
	}
}

func startAppDirectorResultV0(
	request StartAppDirectorRequestV0,
	spec orquestafactory.AppSpecV0,
	prepared orquestaappdirectorintake.AppDirectorIntakePreparedV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) StartAppDirectorResultV0 {
	status := StartAppDirectorStatusPendingV0
	if startAppDirectorLoopStartedV0(request, loop, prepared.DirectorTask.AgentRequestID) {
		status = StartAppDirectorStatusStartedV0
	}
	return StartAppDirectorResultV0{
		SchemaVersion:         StartAppDirectorResultSchemaVersionV0,
		Status:                status,
		DirectorExecutionMode: AppDirectorExecutionModeLegacyDirectorLoopV0,
		CorrelationID:         request.CorrelationID,
		AppSpec:               spec,
		Run:                   loop.Run,
		DirectorTask:          prepared.DirectorTask,
		DirectorTasks:         append([]orquestaappdirectorintake.AppDirectorTaskV0(nil), prepared.DirectorTasks...),
		LoopStatus:            loop.Status,
		StartedAgents:         append([]string(nil), loop.Run.StartedAgents...),
		EvidenceRefs: compactStartAppDirectorStringsV0(append(
			[]string{"evidence-ref-app-director-service-v0"},
			prepared.EvidenceRefs...,
		)),
	}
}

func startAppDirectorLoopStartedV0(
	request StartAppDirectorRequestV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	agentRequestID string,
) bool {
	for _, started := range loop.Run.StartedAgents {
		if started == agentRequestID {
			return true
		}
	}
	return continueHasOperationalDirectorPlanV0(request.OperationalDirectorPlan) &&
		len(loop.Run.StartedAgents) > 0
}
