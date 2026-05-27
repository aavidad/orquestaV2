package orquestaappcodexstack

import (
	"errors"
	"fmt"
	"strings"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestarails "orquesta/modulos/orquesta-rails"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
)

func codexStackRunSupervisorErrorResultMCPV0(
	input orquestamcp.MCPRunSupervisorToolInputV0,
	partial CodexSupervisorResultV0,
	err error,
) orquestamcp.MCPRunSupervisorToolResultV0 {
	var drainErr DrainObservationApplyErrorV0
	if errors.As(err, &drainErr) {
		result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
			input,
			"delivery_ack_ingestion_failed",
			firstNonEmptyQueuedSourceV0(drainErr.Field, "drain_observation"),
			codexStackDrainObservationPublicMessageV0(drainErr),
		)
		result.RunRef = firstNonEmptyQueuedSourceV0(drainErr.RunRef, input.RunRef)
		result.Diagnostics = codexStackDrainObservationDiagnosticsMCPV0(drainErr)
		result.EvidenceRefs = compactStringsV0(append(
			codexStackDrainObservationEvidenceRefsV0(drainErr),
			partial.Last.EvidenceRefs...,
		))
		result.NextActions = compactStringsV0([]string{drainErr.NextActionV0()})
		return result
	}
	result := orquestamcp.NewMCPRunSupervisorErrorResultV0(
		input,
		"run_supervisor_execute_error",
		"executor",
		"run_supervisor_execute_error",
	)
	result.RunRef = firstNonEmptyQueuedSourceV0(partial.Last.SessionRef, input.RunRef)
	result.StopReason = string(partial.StopReason)
	result.Ticks = partial.Ticks
	result.Last = codexStackRunSupervisorSnapshotMCPV0(partial.Last)
	result.History = codexStackRunSupervisorHistoryMCPV0(partial.History)
	result.EvidenceRefs = compactStringsV0(partial.Last.EvidenceRefs)
	if diagnostics := codexStackRunSupervisorDiagnosticsMCPV0(partial.Last.Diagnostics); len(diagnostics) > 0 {
		diagnostics = codexStackRunSupervisorDiagnosticsWithFallbackErrorV0(diagnostics, err)
		result.Diagnostics = diagnostics
		return result
	}
	if len(result.EvidenceRefs) > 0 {
		result.Diagnostics = []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
			Code:         "run_supervisor_partial_snapshot",
			Scope:        "run:" + result.RunRef,
			Message:      "executor fallo con snapshot parcial disponible error=" + codexStackRunSupervisorPublicDiagnosticErrorV0(err.Error()),
			EvidenceRefs: result.EvidenceRefs,
		}}
	}
	return result
}

func codexStackRunSupervisorDiagnosticsWithFallbackErrorV0(
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
	err error,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	if err == nil || len(diagnostics) == 0 {
		return diagnostics
	}
	publicError := codexStackRunSupervisorPublicDiagnosticErrorV0(err.Error())
	if publicError == "" {
		return diagnostics
	}
	out := append([]orquestamcp.MCPAutoprogrammingDiagnosticV0(nil), diagnostics...)
	hasError := false
	for index := range out {
		if strings.Contains(out[index].Message, "error=") && !strings.Contains(out[index].Message, "error= issues=") {
			hasError = true
			break
		}
	}
	if !hasError {
		out[0].Message = strings.TrimSpace(out[0].Message + " error=" + publicError)
	}
	return out
}

func codexStackRunSupervisorDiagnosticsMCPV0(
	diagnostics []orquestaruncoordinator.RunDrainDiagnosticV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	out := make([]orquestamcp.MCPAutoprogrammingDiagnosticV0, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		code := strings.TrimSpace(diagnostic.Kind)
		if code == "" {
			code = "run_supervisor_diagnostic"
		}
		out = append(out, orquestamcp.MCPAutoprogrammingDiagnosticV0{
			Code:         code,
			Scope:        codexStackRunSupervisorDiagnosticScopeMCPV0(diagnostic),
			Message:      codexStackRunSupervisorDiagnosticMessageMCPV0(diagnostic),
			EvidenceRefs: compactStringsV0(diagnostic.EvidenceRefs),
		})
	}
	return out
}

func codexStackRunSupervisorDiagnosticScopeMCPV0(
	diagnostic orquestaruncoordinator.RunDrainDiagnosticV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"run:" + strings.TrimSpace(diagnostic.RunRef),
		"message:" + strings.TrimSpace(diagnostic.MessageID),
	}), " ")
}

func codexStackRunSupervisorDiagnosticMessageMCPV0(
	diagnostic orquestaruncoordinator.RunDrainDiagnosticV0,
) string {
	return strings.Join(compactStringsV0([]string{
		"status=" + strings.TrimSpace(diagnostic.Status),
		"message_type=" + strings.TrimSpace(diagnostic.MessageType),
		"target=" + firstNonEmptyQueuedSourceV0(diagnostic.TargetPort, "capacity-missing"),
		"error=" + codexStackRunSupervisorPublicDiagnosticErrorV0(diagnostic.Error),
		fmt.Sprintf("issues=%d", diagnostic.Issues),
	}), " ")
}

func codexStackRunSupervisorPublicDiagnosticErrorV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	redacted, _ := orquestarails.RedactOperationalTextForFieldV0(
		"codex_stack_run_supervisor",
		"diagnostic_error",
		value,
	)
	return strings.TrimSpace(redacted)
}

func codexStackDrainObservationPublicMessageV0(err DrainObservationApplyErrorV0) string {
	parts := compactStringsV0([]string{
		"delivery_ack_ingestion_failed",
		"field=" + err.Field,
		"code=" + err.CommandCode,
		"cause=" + err.Cause,
		"next_action=" + err.NextActionV0(),
	})
	return strings.Join(parts, " ")
}

func codexStackDrainObservationDiagnosticsMCPV0(
	err DrainObservationApplyErrorV0,
) []orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	message := strings.Join(compactStringsV0([]string{
		"field=" + err.Field,
		"code=" + err.CommandCode,
		"cause=" + err.Cause,
		"next_action=" + err.NextActionV0(),
		fmt.Sprintf("agents=%d started=%d failed=%d lost=%d stopped=%d confirmed_stopped=%d deliveries=%d",
			len(compactStringsV0(err.Agents)),
			len(compactStringsV0(err.StartedAgents)),
			len(compactStringsV0(err.FailedAgents)),
			len(compactStringsV0(err.LostAgents)),
			len(compactStringsV0(err.StoppedAgents)),
			len(compactStringsV0(err.ConfirmedStoppedAgents)),
			len(compactStringsV0(err.Deliveries)),
		),
	}), " ")
	return []orquestamcp.MCPAutoprogrammingDiagnosticV0{{
		Code:         "drain_observation_apply_failed",
		Scope:        codexStackDrainObservationScopeMCPV0(err),
		Message:      message,
		EvidenceRefs: codexStackDrainObservationEvidenceRefsV0(err),
	}}
}

func codexStackDrainObservationScopeMCPV0(err DrainObservationApplyErrorV0) string {
	parts := compactStringsV0([]string{
		"run:" + err.RunRef,
		"task:" + err.TaskRef,
		"agent:" + err.AgentRef,
	})
	return strings.Join(parts, "/")
}

func codexStackDrainObservationEvidenceRefsV0(err DrainObservationApplyErrorV0) []string {
	return compactStringsV0([]string{
		err.ArtifactRef,
		err.DeliveryRef,
		err.TaskRef,
		err.AgentRef,
		err.PhaseID,
	})
}
