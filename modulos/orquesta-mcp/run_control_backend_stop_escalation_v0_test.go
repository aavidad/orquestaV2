package orquestamcp

import (
	"context"
	"errors"
	"testing"

	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type fakeMCPRunControlBackendStopEscalatorV0 struct {
	requests []MCPRunControlBackendStopEscalationRequestV0
	result   MCPRunControlBackendStopEscalationResultV0
	err      error
}

func (fake *fakeMCPRunControlBackendStopEscalatorV0) EscalateBackendStopV0(
	_ context.Context,
	request MCPRunControlBackendStopEscalationRequestV0,
) (MCPRunControlBackendStopEscalationResultV0, error) {
	fake.requests = append(fake.requests, request)
	return fake.result, fake.err
}

func mcpRunControlEscalationActiveBackendForTestV0(runRef string) *fakeMCPRunControlGoalBackendStateV0 {
	goal := MCPDirectorGoalStatsV0{
		GoalRef:         "goal-ref-" + runRef,
		ExternalGoalRef: "thread-ref-" + runRef,
		Status:          "running",
	}
	return &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, Goal: &goal},
			{Estado: MCPDirectorStatsEstadoOKV0, Goal: &goal},
		},
	}
}

func TestMCPRunControlBackendStopEscaladorConfirmaPublicaStoppedV0(t *testing.T) {
	runRef := "run-ref-control-escalation-confirm-001"
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		result: MCPRunControlBackendStopEscalationResultV0{
			Stopped:      true,
			EvidenceRefs: []string{"evidence-ref-backend-stop-real-001"},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:                 &fakeMCPRunControlPortV0{},
		GoalBackendState:     mcpRunControlEscalationActiveBackendForTestV0(runRef),
		BackendStopEscalator: escalator,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID: "request-ref-control-escalation-confirm-001",
		Action:    "stop",
		RunRef:    runRef,
		Forced:    true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.FinalStatus != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		!result.GoalControlSignalConfirmed ||
		mcpRunControlResultHasIssueCodeV0(result, mcpRunControlIssueControlNotPropagatedV0) ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-backend-stop-real-001") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpRunControlEvidenceBackendStopEscalatedV0) {
		t.Fatalf("escalada confirmada no publica stopped: %+v", result)
	}
	if len(escalator.requests) != 1 ||
		escalator.requests[0].RunRef != runRef ||
		escalator.requests[0].GoalRef != "goal-ref-"+runRef ||
		escalator.requests[0].Action != "stop" {
		t.Fatalf("request de escalada invalida: %+v", escalator.requests)
	}
}

func TestMCPRunControlBackendStopEscaladorResidualConservaErrorV0(t *testing.T) {
	runRef := "run-ref-control-escalation-residual-001"
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		result: MCPRunControlBackendStopEscalationResultV0{
			Stopped:      false,
			ResidualRefs: []string{"process-ref-residual-001"},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:                 &fakeMCPRunControlPortV0{},
		GoalBackendState:     mcpRunControlEscalationActiveBackendForTestV0(runRef),
		BackendStopEscalator: escalator,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID: "request-ref-control-escalation-residual-001",
		Action:    "stop",
		RunRef:    runRef,
		Forced:    true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		!mcpRunControlResultHasIssueCodeV0(result, mcpRunControlIssueControlNotPropagatedV0) ||
		result.GoalControlSignalConfirmed ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpRunControlEvidenceBackendStopResidualV0) ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_backend_stop_escalation_residual") {
		t.Fatalf("residual no conserva error: %+v", result)
	}
}

func TestMCPRunControlBackendStopEscaladorErrorConservaErrorV0(t *testing.T) {
	runRef := "run-ref-control-escalation-error-001"
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		err: errors.New("tmux kill failed"),
	}
	executor := MCPRunControlToolExecutorV0{
		Port:                 &fakeMCPRunControlPortV0{},
		GoalBackendState:     mcpRunControlEscalationActiveBackendForTestV0(runRef),
		BackendStopEscalator: escalator,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID: "request-ref-control-escalation-error-001",
		Action:    "stop",
		RunRef:    runRef,
		Forced:    true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		!mcpRunControlResultHasIssueCodeV0(result, mcpRunControlIssueControlNotPropagatedV0) ||
		result.GoalControlSignalConfirmed ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_backend_stop_escalation_failed") {
		t.Fatalf("error de escalada no conserva error: %+v", result)
	}
}

func TestMCPRunControlBackendStopSinForcedNoEscalaV0(t *testing.T) {
	runRef := "run-ref-control-escalation-noforce-001"
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		result: MCPRunControlBackendStopEscalationResultV0{Stopped: true},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:                 &fakeMCPRunControlPortV0{},
		GoalBackendState:     mcpRunControlEscalationActiveBackendForTestV0(runRef),
		BackendStopEscalator: escalator,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID: "request-ref-control-escalation-noforce-001",
		Action:    "stop",
		RunRef:    runRef,
		Forced:    false,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(escalator.requests) != 0 ||
		!mcpRunControlResultHasIssueCodeV0(result, mcpRunControlIssueControlNotPropagatedV0) {
		t.Fatalf("escalada sin forced: requests=%+v result=%+v", escalator.requests, result)
	}
}
