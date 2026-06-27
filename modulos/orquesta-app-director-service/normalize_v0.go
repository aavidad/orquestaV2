package orquestaappdirectorservice

import (
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
)

const (
	defaultStartAppDirectorMaxBurstsV0            = 16
	defaultStartAppDirectorMaxStepsPerBurstV0     = 12
	defaultStartAppDirectorMaxDispatchesPerWaitV0 = 8
	defaultStartAppDirectorMaxCommandsV0          = 20
	defaultStartAppDirectorMaxOutboxPerCycleV0    = 8
	defaultStartAppDirectorMaxDecisionCyclesV0    = 4
	defaultStartAppDirectorMaxExternalWaitsV0     = 120
)

func normalizeStartAppDirectorRequestV0(
	request StartAppDirectorRequestV0,
) StartAppDirectorRequestV0 {
	request.RunRef = strings.TrimSpace(request.RunRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.DirectorExecutionMode = normalizeStartAppDirectorExecutionModeV0(request.DirectorExecutionMode)
	request.WaitAgentRefs = compactStartAppDirectorStringsV0(request.WaitAgentRefs)
	request.WaitCohortRef = strings.TrimSpace(request.WaitCohortRef)
	request.WaitWaveRef = strings.TrimSpace(request.WaitWaveRef)
	request.WaitParentTaskRef = strings.TrimSpace(request.WaitParentTaskRef)
	request.OperationalDirectorPlanRef = strings.TrimSpace(request.OperationalDirectorPlanRef)
	request.OperationalDirectorFunctionContractRefs = normalizeServiceWorkflowFunctionContractRefsV0(request.OperationalDirectorFunctionContractRefs)
	request.OperationalDirectorTargetPhaseID = orquestacoreworkflow.OrchestrationPhaseIDV0(strings.TrimSpace(string(request.OperationalDirectorTargetPhaseID)))
	if request.OperationalDirectorMaxItems < 0 {
		request.OperationalDirectorMaxItems = 0
	}
	request.AppSpecRequest.RequestID = strings.TrimSpace(request.AppSpecRequest.RequestID)
	request.AppSpecRequest.Source = strings.TrimSpace(request.AppSpecRequest.Source)
	if request.CorrelationID == "" {
		request.CorrelationID = strings.TrimSpace(request.AppSpecRequest.RequestID)
	}
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-director-service"
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
	if request.MaxExternalWaits <= 0 {
		request.MaxExternalWaits = defaultStartAppDirectorMaxExternalWaitsV0
	}
	return request
}

func normalizeStartAppDirectorExecutionModeV0(value string) string {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "":
		return AppDirectorExecutionModeGoalFirstV0
	case AppDirectorExecutionModeGoalFirstV0:
		return AppDirectorExecutionModeGoalFirstV0
	case AppDirectorExecutionModeLegacyDirectorLoopV0:
		return AppDirectorExecutionModeLegacyDirectorLoopV0
	default:
		return trimmed
	}
}

func boundedStartAppDirectorLimitV0(value int, max int) int {
	if value > max {
		return max
	}
	return value
}

func startAppDirectorNowV0(occurredAt string) (time.Time, error) {
	occurredAt = strings.TrimSpace(occurredAt)
	if occurredAt == "" {
		return time.Now().UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		return time.Time{}, AppDirectorServiceIssueV0{Field: "occurred_at"}
	}
	return parsed.UTC(), nil
}

func compactStartAppDirectorStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}
