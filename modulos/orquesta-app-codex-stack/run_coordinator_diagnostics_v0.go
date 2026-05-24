package orquestaappcodexstack

import (
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func stackDrainDiagnosticsV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) []orquestaruncoordinator.RunDrainDiagnosticV0 {
	diagnostics := make([]orquestaruncoordinator.RunDrainDiagnosticV0, 0, len(result.Attempts)+len(result.ExternalWaits)+1)
	for _, attempt := range result.Attempts {
		diagnostics = append(diagnostics, stackDrainLoopDiagnosticsV0(attempt.AttemptNumber, attempt.Result)...)
	}
	for _, wait := range result.ExternalWaits {
		diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
			Kind:          "external_wait",
			Status:        stackDrainBoolStatusV0(wait.Continue),
			AttemptNumber: wait.WaitNumber,
			EvidenceRefs:  compactCodexStackStringsV0(wait.EvidenceRefs),
		})
	}
	diagnostics = append(diagnostics, stackDrainFinalDiagnosticV0(result))
	return diagnostics
}

func stackDrainLoopDiagnosticsV0(
	attemptNumber int,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
) []orquestaruncoordinator.RunDrainDiagnosticV0 {
	diagnostics := []orquestaruncoordinator.RunDrainDiagnosticV0{
		{
			Kind:               "drain_attempt",
			Status:             string(loop.Status),
			RunRef:             strings.TrimSpace(loop.Run.RunID),
			AttemptNumber:      attemptNumber,
			ExecutedSteps:      loop.TotalExecutedSteps,
			FinalAction:        string(loop.FinalAction),
			FirstPendingCount:  loop.FirstPendingCount,
			FirstPendingRefs:   compactCodexStackStringsV0(loop.FirstPendingRefs),
			PendingOutboxCount: loop.PendingOutboxCount,
			PendingOutboxRefs:  compactCodexStackStringsV0(loop.PendingOutboxRefs),
		},
	}
	for _, burst := range loop.Bursts {
		diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
			Kind:          "director_burst",
			Status:        string(burst.FinalAction),
			RunRef:        strings.TrimSpace(loop.Run.RunID),
			AttemptNumber: attemptNumber,
			BurstNumber:   burst.BurstNumber,
			ExecutedSteps: burst.Executed,
			FinalAction:   string(burst.FinalAction),
			EvidenceRefs:  compactCodexStackStringsV0(burst.EvidenceRefs),
		})
	}
	for _, dispatch := range loop.BatchDispatches {
		diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
			Kind:          "outbox_batch_dispatch",
			Status:        dispatch.Status,
			RunRef:        strings.TrimSpace(dispatch.RunRef),
			AttemptNumber: attemptNumber,
			TargetPort:    strings.TrimSpace(dispatch.TargetPort),
			MessageType:   strings.TrimSpace(dispatch.MessageType),
			PlannedCount:  dispatch.PlannedCount,
			AckedCount:    dispatch.AckedCount,
			PendingCount:  dispatch.PendingCount,
			FailedCount:   dispatch.FailedCount,
			Issues:        dispatch.Issues,
			ClaimedMessages: compactCodexStackStringsV0(
				dispatch.ClaimedMessageIDs,
			),
			AckedMessages: compactCodexStackStringsV0(dispatch.AckedMessages),
		})
	}
	for _, dispatch := range loop.Dispatches {
		diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
			Kind:          "outbox_dispatch",
			Status:        dispatch.Status,
			RunRef:        strings.TrimSpace(dispatch.RunRef),
			AttemptNumber: attemptNumber,
			TargetPort:    strings.TrimSpace(dispatch.TargetPort),
			MessageType:   strings.TrimSpace(dispatch.MessageType),
			MessageID:     strings.TrimSpace(dispatch.MessageID),
			DispatchRef:   strings.TrimSpace(dispatch.DispatchRef),
			Issues:        dispatch.Issues,
			EvidenceRefs:  compactCodexStackStringsV0(dispatch.EvidenceRefs),
		})
	}
	return diagnostics
}

func stackDrainFinalDiagnosticV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
) orquestaruncoordinator.RunDrainDiagnosticV0 {
	return orquestaruncoordinator.RunDrainDiagnosticV0{
		Kind:               "drain_final",
		Status:             string(result.Status),
		RunRef:             strings.TrimSpace(result.Final.Run.RunID),
		ExecutedSteps:      result.Final.TotalExecutedSteps,
		FinalAction:        string(result.Final.FinalAction),
		FirstPendingCount:  result.Final.FirstPendingCount,
		FirstPendingRefs:   compactCodexStackStringsV0(result.Final.FirstPendingRefs),
		PendingOutboxCount: result.Final.PendingOutboxCount,
		PendingOutboxRefs:  compactCodexStackStringsV0(result.Final.PendingOutboxRefs),
	}
}

func stackDrainBoolStatusV0(value bool) string {
	if value {
		return "continue"
	}
	return "stop"
}

func stackDrainDiagnosticsWithErrorV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
	errorMessage string,
) []orquestaruncoordinator.RunDrainDiagnosticV0 {
	errorMessage = strings.TrimSpace(errorMessage)
	if errorMessage == "" {
		return diagnostics
	}
	diagnostics = append(diagnostics, stackDrainErrorDiagnosticV0(result, errorMessage))
	for _, attempt := range result.Attempts {
		diagnostics = append(diagnostics, stackDrainDispatchErrorDiagnosticsV0(
			attempt.AttemptNumber,
			attempt.Result,
			errorMessage,
		)...)
	}
	return diagnostics
}

func stackDrainErrorDiagnosticV0(
	result orquestacionnucleoapp.ManagedProgressiveLoopResultV0,
	errorMessage string,
) orquestaruncoordinator.RunDrainDiagnosticV0 {
	return orquestaruncoordinator.RunDrainDiagnosticV0{
		Kind:   "drain_error",
		Status: "error",
		RunRef: strings.TrimSpace(result.Final.Run.RunID),
		Error:  errorMessage,
	}
}

func stackDrainDispatchErrorDiagnosticsV0(
	attemptNumber int,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	errorMessage string,
) []orquestaruncoordinator.RunDrainDiagnosticV0 {
	diagnostics := []orquestaruncoordinator.RunDrainDiagnosticV0{}
	for _, dispatch := range loop.BatchDispatches {
		if !stackDrainBatchDispatchErroredV0(dispatch.Status) {
			continue
		}
		for _, messageID := range compactCodexStackStringsV0(dispatch.ClaimedMessageIDs) {
			diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
				Kind:          "outbox_batch_dispatch_error",
				Status:        dispatch.Status,
				RunRef:        strings.TrimSpace(dispatch.RunRef),
				Error:         errorMessage,
				AttemptNumber: attemptNumber,
				TargetPort:    strings.TrimSpace(dispatch.TargetPort),
				MessageType:   strings.TrimSpace(dispatch.MessageType),
				MessageID:     messageID,
				Issues:        dispatch.Issues,
			})
		}
	}
	for _, dispatch := range loop.Dispatches {
		if !stackDrainOnceDispatchErroredV0(dispatch.Status) {
			continue
		}
		diagnostics = append(diagnostics, orquestaruncoordinator.RunDrainDiagnosticV0{
			Kind:          "outbox_dispatch_error",
			Status:        dispatch.Status,
			RunRef:        strings.TrimSpace(dispatch.RunRef),
			Error:         errorMessage,
			AttemptNumber: attemptNumber,
			TargetPort:    strings.TrimSpace(dispatch.TargetPort),
			MessageType:   strings.TrimSpace(dispatch.MessageType),
			MessageID:     strings.TrimSpace(dispatch.MessageID),
			Issues:        dispatch.Issues,
		})
	}
	return diagnostics
}

func stackDrainBatchDispatchErroredV0(status string) bool {
	switch strings.TrimSpace(status) {
	case orquestacionnucleoapp.OutboxDispatchBatchRunExecutionFailedV0,
		orquestacionnucleoapp.OutboxDispatchBatchRunAckFailedV0:
		return true
	default:
		return false
	}
}

func stackDrainOnceDispatchErroredV0(status string) bool {
	switch strings.TrimSpace(status) {
	case "dispatch_failed", "ack_failed":
		return true
	default:
		return false
	}
}
