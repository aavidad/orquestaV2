package orquestaruncoordinator

import (
	"strings"

	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func executionSummaryV0(
	candidate orquestarunqueue.RankedRunCandidateV0,
	result RunDrainResultV0,
) RunExecutionSummaryV0 {
	queueStatus := effectiveExecutedRunQueueStatusV0(candidate, result)
	return RunExecutionSummaryV0{
		RunRef:       candidate.RunRef,
		AppRef:       candidate.AppRef,
		Rank:         candidate.Rank,
		Outcome:      result.Outcome,
		QueueStatus:  queueStatus,
		EvidenceRefs: append([]string(nil), result.EvidenceRefs...),
		Diagnostics:  append([]RunDrainDiagnosticV0(nil), result.Diagnostics...),
	}
}

func runDrainResultWithErrorDiagnosticV0(
	candidate orquestarunqueue.RankedRunCandidateV0,
	result RunDrainResultV0,
	err error,
) RunDrainResultV0 {
	if strings.TrimSpace(result.RunRef) == "" {
		result.RunRef = candidate.RunRef
	}
	if strings.TrimSpace(result.AppRef) == "" {
		result.AppRef = candidate.AppRef
	}
	if strings.TrimSpace(result.Outcome) == "" {
		result.Outcome = "error"
	}
	message := ""
	if err != nil {
		message = strings.TrimSpace(err.Error())
	}
	if message == "" {
		return result
	}
	for _, diagnostic := range result.Diagnostics {
		if strings.TrimSpace(diagnostic.Error) != "" {
			return result
		}
	}
	result.Diagnostics = append(result.Diagnostics, RunDrainDiagnosticV0{
		Kind:   "drain_error",
		Status: "error",
		RunRef: strings.TrimSpace(candidate.RunRef),
		Error:  message,
	})
	return result
}
