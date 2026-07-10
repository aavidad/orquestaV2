package orquestamcp

import (
	"context"
	"strings"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

const (
	mcpRunControlIssueControlNotPropagatedV0 = "control_not_propagated_to_goal_backend"

	mcpRunControlEvidenceBackendStopEscalatedV0 = "evidence-ref-run-control-backend-stop-escalated"
	mcpRunControlEvidenceBackendStopResidualV0  = "evidence-ref-run-control-backend-stop-residual"
	mcpRunControlEvidenceBackendStopErrorV0     = "evidence-ref-run-control-backend-stop-escalation-error"
)

type MCPRunControlBackendStopEscalationRequestV0 struct {
	RunRef          string
	GoalRef         string
	ExternalGoalRef string
	Action          string
	Reason          string
	RequestedBy     string
	EvidenceRefs    []string
}

type MCPRunControlBackendStopEscalationResultV0 struct {
	Stopped      bool
	ResidualRefs []string
	EvidenceRefs []string
}

// MCPRunControlBackendStopEscalatorPortV0 permite a la composicion inyectar
// una parada real del backend goal (p.ej. shutdown work cleaner del backend
// tmux) cuando el control local no propaga stop/cancel.
type MCPRunControlBackendStopEscalatorPortV0 interface {
	EscalateBackendStopV0(
		ctx context.Context,
		request MCPRunControlBackendStopEscalationRequestV0,
	) (MCPRunControlBackendStopEscalationResultV0, error)
}

func (executor MCPRunControlToolExecutorV0) escalateBackendStopIfRequestedV0(
	ctx context.Context,
	input MCPRunControlToolInputV0,
	result MCPRunControlToolResultV0,
) MCPRunControlToolResultV0 {
	if executor.BackendStopEscalator == nil || !input.Forced {
		return result
	}
	if !mcpRunControlResultHasIssueCodeV0(result, mcpRunControlIssueControlNotPropagatedV0) {
		return result
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runRef := strings.TrimSpace(result.RunRef)
	if runRef == "" {
		runRef = strings.TrimSpace(input.RunRef)
	}
	escalated, err := executor.BackendStopEscalator.EscalateBackendStopV0(ctx, MCPRunControlBackendStopEscalationRequestV0{
		RunRef:          runRef,
		GoalRef:         strings.TrimSpace(result.GoalRef),
		ExternalGoalRef: strings.TrimSpace(result.ExternalGoalRef),
		Action:          normalizeMCPRunControlActionV0(input.Action),
		Reason:          strings.TrimSpace(input.Reason),
		RequestedBy:     firstNonEmptyMCPV0(input.RequestedBy, "orquesta-mcp-run-control"),
		EvidenceRefs:    compactStringsMCPV0(input.EvidenceRefs),
	})
	if err != nil {
		result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, mcpRunControlEvidenceBackendStopErrorV0))
		result.Diagnostics = append(result.Diagnostics, MCPRunControlDiagnosticV0{
			Code:         "goal_backend_stop_escalation_failed",
			Scope:        "run:" + runRef,
			Message:      "el escalador de parada del backend goal fallo; se conserva el error de no propagacion",
			EvidenceRefs: []string{mcpRunControlEvidenceBackendStopErrorV0},
		})
		return result
	}
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, escalated.EvidenceRefs...))
	if !escalated.Stopped {
		residualRefs := compactStringsMCPV0(escalated.ResidualRefs)
		result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, mcpRunControlEvidenceBackendStopResidualV0))
		result.Diagnostics = append(result.Diagnostics, MCPRunControlDiagnosticV0{
			Code:         "goal_backend_stop_escalation_residual",
			Scope:        "run:" + runRef,
			Message:      "el escalador no confirmo la parada del backend goal; quedan residuos y se conserva el error",
			EvidenceRefs: compactStringsMCPV0(append([]string{mcpRunControlEvidenceBackendStopResidualV0}, residualRefs...)),
		})
		return result
	}
	target := orquestaruncontrol.RunControlStatusStoppedV0
	if normalizeMCPRunControlActionV0(input.Action) == "cancel" {
		target = orquestaruncontrol.RunControlStatusCanceledV0
	}
	result.Estado = MCPRunControlEstadoOKV0
	result.Status = string(target)
	result.FinalStatus = result.Status
	result.GoalControlSignalSent = true
	result.GoalControlSignalConfirmed = true
	result.RecommendedAction = "observe_later"
	result.Errores = mcpRunControlIssuesWithoutCodeV0(result.Errores, mcpRunControlIssueControlNotPropagatedV0)
	result.EvidenceRefs = compactStringsMCPV0(append(result.EvidenceRefs, mcpRunControlEvidenceBackendStopEscalatedV0))
	result.Diagnostics = append(result.Diagnostics, MCPRunControlDiagnosticV0{
		Code:         "goal_backend_stop_escalated",
		Scope:        "run:" + runRef,
		Message:      "el escalador confirmo la parada real del backend goal tras control forzado",
		EvidenceRefs: compactStringsMCPV0(append([]string{mcpRunControlEvidenceBackendStopEscalatedV0}, escalated.EvidenceRefs...)),
	})
	return result
}

func mcpRunControlResultHasIssueCodeV0(result MCPRunControlToolResultV0, code string) bool {
	code = strings.TrimSpace(code)
	for _, issue := range result.Errores {
		if strings.TrimSpace(issue.Code) == code {
			return true
		}
	}
	return false
}

func mcpRunControlIssuesWithoutCodeV0(
	issues []MCPValidationIssueV0,
	code string,
) []MCPValidationIssueV0 {
	code = strings.TrimSpace(code)
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		if strings.TrimSpace(issue.Code) == code {
			continue
		}
		out = append(out, issue)
	}
	return out
}
