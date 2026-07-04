package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestMCPAutoprogrammingPrepareRunDescriptorV0EsAdaptadorOptIn(t *testing.T) {
	descriptor := MCPAutoprogrammingPrepareRunDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingPrepareRunToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingPrepareRunResourceURIV0 ||
		descriptor.InputSchema == "" ||
		descriptor.Output == "" {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if len(descriptor.Invariantes) == 0 {
		t.Fatalf("invariantes vacias")
	}
	if !strings.Contains(descriptor.InputSchema, "autoprogramming_request:AutoprogrammingRequestV0") {
		t.Fatalf("descriptor debe publicar autoprogramming_request como objeto tipado, no string generico: %q", descriptor.InputSchema)
	}
	if !strings.Contains(descriptor.InputSchema, "required_settings?[]{key,value}") {
		t.Fatalf("descriptor debe publicar required_settings: %q", descriptor.InputSchema)
	}
	if !strings.Contains(descriptor.Output, "goal_spec_summaries?") ||
		strings.Contains(descriptor.Output, "goal_specs?") {
		t.Fatalf("descriptor debe publicar solo resumenes de specs en prepare-run: %q", descriptor.Output)
	}
	for _, want := range []string{
		"goal_spec_summaries?[]{schema_version,goal_ref?,run_ref?,director_kind?,spec_hash?",
		"context_refs?",
		"rule_refs?",
		"required_test_refs?",
		"artifact_types?",
		"write_set_count?",
		"required_test_count?",
		"closure_requires_artifact_paths?",
	} {
		if !strings.Contains(descriptor.Output, want) {
			t.Fatalf("descriptor debe tipar goal_spec_summaries publicos: falta %q en %q", want, descriptor.Output)
		}
	}
	if !strings.Contains(descriptor.Output, "goals?[]") ||
		!strings.Contains(descriptor.Output, "external_goal_ref?") {
		t.Fatalf("descriptor debe publicar goals[] tipado en prepare-run: %q", descriptor.Output)
	}
	if !strings.Contains(descriptor.Output, "run_ref?") || !strings.Contains(descriptor.Output, "continue?") {
		t.Fatalf("descriptor debe publicar run_ref y continue como opcionales en goal-first: %q", descriptor.Output)
	}
	if !containsMCPStringPartForTestV0(descriptor.Invariantes, "goal-first es la ruta preferente") ||
		!containsMCPStringPartForTestV0(descriptor.Invariantes, "compatibilidad legacy") ||
		!containsMCPStringPartForTestV0(descriptor.Invariantes, "goals[] es canonico") {
		t.Fatalf("descriptor debe demotar continue legacy frente a goal-first: %+v", descriptor.Invariantes)
	}
}

func TestValidateMCPRequiredSettingsProjectionV0DetectaMismatchSinValoresSensibles(t *testing.T) {
	issues := ValidateMCPRequiredSettingsProjectionV0(
		[]MCPRequiredSettingV0{{
			Key:   "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS",
			Value: "450000",
		}},
		[]MCPConfigProjectionSettingV0{{
			Key:       "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS",
			Value:     "123",
			Sensitive: true,
		}},
	)

	if len(issues) != 1 ||
		issues[0].Code != MCPConfigProjectionMismatchV0 ||
		!strings.Contains(issues[0].Field, "ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS") ||
		!strings.Contains(issues[0].Message, "expected_configured=true") ||
		!strings.Contains(issues[0].Message, "actual_configured=true") ||
		strings.Contains(issues[0].Message, "450000") ||
		strings.Contains(issues[0].Message, "123") {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestNewMCPAutoprogrammingPrepareRunErrorResultV0NormalizaErrorPublico(t *testing.T) {
	result := NewMCPAutoprogrammingPrepareRunErrorResultV0(
		MCPAutoprogrammingPrepareRunToolInputV0{
			RequestID:     "request-prepare-run-001",
			CorrelationID: "corr-prepare-run-001",
		},
		"",
		"executor",
		"",
	)

	if result.Estado != MCPAutoprogrammingPrepareRunEstadoErrorV0 ||
		result.Accepted ||
		result.RequestID != "request-prepare-run-001" ||
		result.CorrelationID != "corr-prepare-run-001" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code == "" ||
		result.Errores[0].Message == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingPrepareRunTransportV0BoundInvocaExecutor(t *testing.T) {
	executor := &fakeMCPAutoprogrammingPrepareRunTransportExecutorV0{
		result: MCPAutoprogrammingPrepareRunToolResultV0{
			Estado:           MCPAutoprogrammingPrepareRunEstadoOKV0,
			RequestID:        "request-ref-prepare-run-bound-001",
			CorrelationID:    "corr-prepare-run-bound-001",
			Accepted:         true,
			RunRef:           "run-ref-prepare-run-bound-001",
			WorkflowTaskRefs: []string{"workflow-task-ref-001"},
			WaitAgentRefs:    []string{"agent-request-ref-001"},
			GoalSpecs: []orquestagoal.GoalWorkSpecV0{{
				SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
				GoalRef:       "goal-ref-prepare-run-bound-001",
				RunRef:        "run-ref-prepare-run-bound-001",
				Objective:     "validar transporte MCP de specs internas",
				DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
				WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-mcp"}},
			}},
		},
	}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		AutoprogrammingPrepareRun: executor,
	}); err != nil {
		t.Fatalf("register transport: %v", err)
	}

	tool, ok := transport.tools[MCPAutoprogrammingPrepareRunToolNameV0]
	if !ok {
		t.Fatalf("tool no registrado: %s", MCPAutoprogrammingPrepareRunToolNameV0)
	}
	descriptor := MCPAutoprogrammingPrepareRunDescriptorV0()
	if tool.ResourceURI != descriptor.ResourceURI ||
		tool.InputShape != descriptor.InputSchema ||
		tool.OutputShape != descriptor.Output ||
		tool.Mode != MCPTransportModeOptInV0 {
		t.Fatalf("envelope inesperado: %+v", tool)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingPrepareRunToolNameV0,
		MCPAutoprogrammingPrepareRunToolInputV0{
			RequestID:              "request-ref-prepare-run-bound-001",
			CorrelationID:          "corr-prepare-run-bound-001",
			AutoprogrammingRequest: validMCPAutoprogrammingRequestV0(),
		},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if executor.called != 1 ||
		executor.input.RequestID != "request-ref-prepare-run-bound-001" ||
		executor.input.CorrelationID != "corr-prepare-run-bound-001" {
		t.Fatalf("executor no invocado correctamente: called=%d input=%+v", executor.called, executor.input)
	}
	if strings.Contains(string(output), `"goal_specs"`) ||
		strings.Contains(string(output), `"objective"`) ||
		strings.Contains(string(output), `"write_set"`) ||
		strings.Contains(string(output), "validar transporte MCP de specs internas") ||
		strings.Contains(string(output), "modulos/orquesta-mcp") {
		t.Fatalf("payload MCP filtra GoalWorkSpec completo: %s", string(output))
	}
	var result MCPAutoprogrammingPrepareRunToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingPrepareRunEstadoOKV0 ||
		!result.Accepted ||
		result.RunRef != "run-ref-prepare-run-bound-001" ||
		len(result.WaitAgentRefs) != 1 ||
		len(result.GoalSpecSummaries) != 1 ||
		result.GoalSpecSummaries[0].RunRef != "run-ref-prepare-run-bound-001" ||
		result.GoalSpecSummaries[0].DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		result.GoalSpecSummaries[0].SpecHash == "" {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, output, 1400)
}

type fakeMCPAutoprogrammingPrepareRunTransportExecutorV0 struct {
	called int
	input  MCPAutoprogrammingPrepareRunToolInputV0
	result MCPAutoprogrammingPrepareRunToolResultV0
}

func (executor *fakeMCPAutoprogrammingPrepareRunTransportExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingPrepareRunToolInputV0,
) (MCPAutoprogrammingPrepareRunToolResultV0, error) {
	_ = ctx
	executor.called++
	executor.input = input
	return executor.result, nil
}
