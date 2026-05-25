package orquestaappdirectorservice

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"strings"
)

func continueAppDirectorResultV0(
	request ContinueAppDirectorRequestV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	closureIssues []orquestacionnucleoapp.ErrorV0,
) ContinueAppDirectorResultV0 {
	status := ContinueAppDirectorStatusPendingV0
	if loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0 ||
		loop.Status == orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		status = ContinueAppDirectorStatusContinuedV0
	}
	return ContinueAppDirectorResultV0{
		SchemaVersion:            ContinueAppDirectorResultSchemaVersionV0,
		Status:                   status,
		CorrelationID:            request.CorrelationID,
		Run:                      loop.Run,
		LoopStatus:               loop.Status,
		StartedAgents:            append([]string(nil), loop.Run.StartedAgents...),
		OperationalClosureIssues: append([]orquestacionnucleoapp.ErrorV0(nil), closureIssues...),
		EvidenceRefs:             []string{"evidence-ref-app-director-continue-v0"},
	}
}

func normalizeContinueAppDirectorRequestV0(
	request ContinueAppDirectorRequestV0,
) ContinueAppDirectorRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.WaitAgentRefs = compactServiceRefsV0(request.WaitAgentRefs)
	request.WaitCohortRef = strings.TrimSpace(request.WaitCohortRef)
	request.WaitWaveRef = strings.TrimSpace(request.WaitWaveRef)
	request.WaitParentTaskRef = strings.TrimSpace(request.WaitParentTaskRef)
	request.OperationalDirectorPlanRef = strings.TrimSpace(request.OperationalDirectorPlanRef)
	request.OperationalDirectorFunctionContractRefs = normalizeServiceWorkflowFunctionContractRefsV0(request.OperationalDirectorFunctionContractRefs)
	request.OperationalDirectorTargetPhaseID = orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(request.OperationalDirectorTargetPhaseID)))
	if request.OperationalDirectorMaxItems < 0 {
		request.OperationalDirectorMaxItems = 0
	}
	if request.CorrelationID == "" {
		request.CorrelationID = "corr-" + request.RunRef
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-director-continue"
	}
	if request.MaxBursts <= 0 {
		request.MaxBursts = defaultStartAppDirectorMaxBurstsV0
	}
	if request.MaxStepsPerBurst <= 0 {
		request.MaxStepsPerBurst = defaultStartAppDirectorMaxStepsPerBurstV0
	}
	if request.MaxDispatchesPerWait <= 0 {
		request.MaxDispatchesPerWait = defaultStartAppDirectorMaxDispatchesPerWaitV0
	}
	if request.MaxCommands <= 0 {
		request.MaxCommands = defaultStartAppDirectorMaxCommandsV0
	}
	request.MaxCommands = boundedStartAppDirectorLimitV0(
		request.MaxCommands,
		orquestadirectorrunner.DirectorCycleMaxCommandsV0,
	)
	if request.MaxOutboxPerCycle <= 0 {
		request.MaxOutboxPerCycle = defaultStartAppDirectorMaxOutboxPerCycleV0
	}
	request.MaxOutboxPerCycle = boundedStartAppDirectorLimitV0(
		request.MaxOutboxPerCycle,
		orquestadirectorrunner.DirectorCycleMaxOutboxV0,
	)
	if request.MaxDecisionCycles <= 0 {
		request.MaxDecisionCycles = defaultStartAppDirectorMaxDecisionCyclesV0
	}
	if request.MaxExternalWaits < 0 {
		request.MaxExternalWaits = 0
	}
	return request
}

func validateContinueAppDirectorRequestV0(
	request ContinueAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
) error {
	if strings.TrimSpace(request.RunRef) == "" {
		return AppDirectorServiceIssueV0{Field: "run_ref"}
	}
	if _, err := startAppDirectorNowV0(request.OccurredAt); err != nil {
		return err
	}
	return validateStartAppDirectorPortsV0(ports)
}
