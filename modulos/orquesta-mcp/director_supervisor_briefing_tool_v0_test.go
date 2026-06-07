package orquestamcp

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	orquestadirectorsupervisor "orquesta/modulos/orquesta-director-supervisor"
)

func TestMCPDirectorSupervisorBriefingToolExecutorV0ProyectaBriefingPuro(t *testing.T) {
	result, err := (MCPDirectorSupervisorBriefingToolExecutorV0{}).Execute(
		context.Background(),
		validMCPDirectorSupervisorBriefingInputForTestV0(),
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorSupervisorBriefingEstadoOKV0 ||
		result.RunRef != "run-mcp-supervisor-briefing-001" ||
		result.Briefing == nil ||
		result.Briefing.NextAction == nil ||
		result.Briefing.NextAction.Kind != orquestadirectorsupervisor.DirectorSupervisorActionKindRunStepV0 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Briefing.ActionQueue) != 1 || len(result.Briefing.Timeline) != 1 {
		t.Fatalf("briefing=%+v", result.Briefing)
	}
}

func TestMCPDirectorSupervisorBriefingToolExecutorV0DevuelveErrorPublico(t *testing.T) {
	input := validMCPDirectorSupervisorBriefingInputForTestV0()
	input.BriefingInput.Decision.RunRef = ""

	result, err := (MCPDirectorSupervisorBriefingToolExecutorV0{}).Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorSupervisorBriefingEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "decision.run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorSupervisorBriefingTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(
		context.Background(),
		MCPDirectorSupervisorBriefingToolNameV0,
		validMCPDirectorSupervisorBriefingInputForTestV0(),
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPDirectorSupervisorBriefingToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPDirectorSupervisorBriefingEstadoOKV0 ||
		result.Briefing == nil ||
		result.Briefing.SchemaVersion != orquestadirectorsupervisor.DirectorSupervisorBriefingSchemaV0 {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 2500)
}

func TestMCPDirectorSupervisorBriefingDescriptorV0ExponeContratoCompacto(t *testing.T) {
	descriptor := MCPDirectorSupervisorBriefingDescriptorV0()
	if descriptor.Name != MCPDirectorSupervisorBriefingToolNameV0 ||
		descriptor.Version != MCPDirectorSupervisorBriefingToolVersionV0 ||
		descriptor.ResourceURI != MCPDirectorSupervisorBriefingResourceURIV0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	fields, ok := MCPTransportToolInputFieldsV0(MCPDirectorSupervisorBriefingToolNameV0)
	if !ok || len(fields) == 0 {
		t.Fatalf("fields=%+v ok=%v", fields, ok)
	}
	if !mcpDirectorSupervisorBriefingFieldRequiredForTestV0(fields, "briefing_input") {
		t.Fatalf("briefing_input debe ser requerido: %+v", fields)
	}
}

func TestMCPDirectorSupervisorBriefingTransportV0NilExecutorConcurrente(t *testing.T) {
	handler := mcpDirectorSupervisorBriefingTransportHandlerV0(nil)
	raw, err := json.Marshal(validMCPDirectorSupervisorBriefingInputForTestV0())
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	var wg sync.WaitGroup
	for idx := 0; idx < 12; idx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			output, err := handler(context.Background(), raw)
			if err != nil {
				t.Errorf("handler: %v", err)
				return
			}
			var result MCPDirectorSupervisorBriefingToolResultV0
			if err := json.Unmarshal(output, &result); err != nil {
				t.Errorf("decode: %v", err)
				return
			}
			if result.Estado != MCPDirectorSupervisorBriefingEstadoOKV0 || result.Briefing == nil {
				t.Errorf("result=%+v", result)
			}
		}()
	}
	wg.Wait()
}

func validMCPDirectorSupervisorBriefingInputForTestV0() MCPDirectorSupervisorBriefingToolInputV0 {
	return MCPDirectorSupervisorBriefingToolInputV0{
		RequestID:     "request-ref-mcp-supervisor-briefing-001",
		CorrelationID: "correlation-ref-mcp-supervisor-briefing-001",
		BriefingInput: orquestadirectorsupervisor.DirectorSupervisorBriefingInputV0{
			ObjectiveRef: "objective-ref-mcp-supervisor-briefing-001",
			ContextRefs:  []string{"context-ref-mcp-supervisor-briefing-001"},
			Decision: orquestadirectorsupervisor.DirectorSupervisorDecisionV0{
				RunRef:                   "run-mcp-supervisor-briefing-001",
				Action:                   orquestadirectorsupervisor.DirectorSupervisorActionContinueV0,
				ShouldContinue:           true,
				AutonomousRecommendation: orquestadirectorsupervisor.DirectorSupervisorAutonomousContinueV0,
				ReasonCode:               orquestadirectorsupervisor.DirectorSupervisorReasonContinueV0,
				StepNumber:               2,
				MaxSteps:                 10,
				EvidenceRefs:             []string{"evidence-ref-mcp-supervisor-briefing-001"},
			},
		},
	}
}

func mcpDirectorSupervisorBriefingFieldRequiredForTestV0(
	fields []MCPTransportToolInputFieldV0,
	name string,
) bool {
	for _, field := range fields {
		if field.Name == name {
			return field.Required
		}
	}
	return false
}
