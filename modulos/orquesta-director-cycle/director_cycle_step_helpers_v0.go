package orquestadirectorcycle

import (
	"strings"

	orquestadirectorrunner "orquesta/modulos/orquesta-director-runner"
)

func normalizeDirectorCycleStepInputV0(input DirectorCycleStepInputV0) DirectorCycleStepInputV0 {
	input.CycleRef = strings.TrimSpace(input.CycleRef)
	input.TickRef = strings.TrimSpace(input.TickRef)
	input.RunRef = strings.TrimSpace(input.RunRef)
	input.OccurredAt = strings.TrimSpace(input.OccurredAt)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.EvidenceRefs = compactDirectorCycleStepStringsV0(input.EvidenceRefs)
	return input
}

func newDirectorCycleStepResultV0(input DirectorCycleStepInputV0) DirectorCycleStepResultV0 {
	return DirectorCycleStepResultV0{
		CycleRef: input.CycleRef,
		TickRef:  input.TickRef,
		RunRef:   input.RunRef,
	}
}

func copyRunnerResultToStepV0(
	result *DirectorCycleStepResultV0,
	runnerResult orquestadirectorrunner.DirectorCycleResultV0,
) {
	result.Status = runnerResult.Status
	result.SchedulerStatus = runnerResult.SchedulerStatus
	result.StopReason = runnerResult.StopReason
	result.AppliedCommands = append([]orquestadirectorrunner.DirectorCycleAppliedCommandV0(nil), runnerResult.AppliedCommands...)
	result.WaitingReasons = append(result.WaitingReasons[:0:0], runnerResult.WaitingReasons...)
	result.BlockedRefs = compactDirectorCycleStepStringsV0(runnerResult.BlockedRefs)
	result.EventsCount = runnerResult.EventsCount
}

func compactDirectorCycleStepStringsV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
