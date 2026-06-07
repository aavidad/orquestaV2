package main

import (
	"encoding/json"
	"testing"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestMCPRealTransportV0DirectorSupervisorBriefingJSONRPCV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var result mcpToolCallResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/call", map[string]any{
		"name": orquestamcp.MCPDirectorSupervisorBriefingToolNameV0,
		"arguments": orquestamcp.MCPDirectorSupervisorBriefingToolInputV0{
			RequestID:     "request-ref-mcp-real-supervisor-briefing-001",
			CorrelationID: "corr-mcp-real-supervisor-briefing-001",
			BriefingInput: orquestadirectorsupervisor.DirectorSupervisorBriefingInputV0{
				ObjectiveRef: "objective-ref-mcp-real-supervisor-briefing-001",
				ContextRefs:  []string{"context-ref-mcp-real-supervisor-briefing-001"},
				Decision: orquestadirectorsupervisor.DirectorSupervisorDecisionV0{
					RunRef:                   "run-ref-mcp-real-supervisor-briefing-001",
					Action:                   orquestadirectorsupervisor.DirectorSupervisorActionContinueV0,
					ShouldContinue:           true,
					AutonomousRecommendation: orquestadirectorsupervisor.DirectorSupervisorAutonomousContinueV0,
					ReasonCode:               orquestadirectorsupervisor.DirectorSupervisorReasonContinueV0,
					StepNumber:               1,
					MaxSteps:                 5,
				},
			},
		},
	}, &result)
	if result.IsError || len(result.Content) != 1 {
		t.Fatalf("result=%+v", result)
	}
	var toolResult orquestamcp.MCPDirectorSupervisorBriefingToolResultV0
	if err := json.Unmarshal([]byte(result.Content[0].Text), &toolResult); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if toolResult.Estado != orquestamcp.MCPDirectorSupervisorBriefingEstadoOKV0 ||
		toolResult.Briefing == nil ||
		toolResult.Briefing.NextAction == nil ||
		toolResult.Briefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0 {
		t.Fatalf("toolResult=%+v", toolResult)
	}
}

func TestMCPRealTransportV0DirectorSupervisorBriefingSchemaRequeridoV0(t *testing.T) {
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	defer server.Close()

	var tools mcpToolListResultV0
	callMCPJSONRPCTestV0(t, server.URL+mcpRealHTTPPathV0, "tools/list", map[string]any{}, &tools)
	tool := mcpToolByNameTestV0(tools.Tools, orquestamcp.MCPDirectorSupervisorBriefingToolNameV0)
	if tool == nil {
		t.Fatalf("tool no registrado")
	}
	required, _ := tool.InputSchema["required"].([]any)
	if !mcpRealRequiredContainsForTestV0(required, "briefing_input") {
		t.Fatalf("schema sin briefing_input requerido: %+v", tool.InputSchema)
	}
}

func mcpRealRequiredContainsForTestV0(values []any, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
