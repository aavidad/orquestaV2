package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPAutoprogrammingStatusDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPAutoprogrammingStatusDescriptorV0()

	if descriptor.Name != MCPAutoprogrammingStatusToolNameV0 ||
		descriptor.ResourceURI != MCPAutoprogrammingStatusResourceURIV0 ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor incompleto: %+v", descriptor)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DelegaEnColaYRun(t *testing.T) {
	queue := &fakeMCPAutoprogrammingQueueStatusV0{}
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: queue,
		Stats: stats,
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RequestID:     "request-ref-autop-status-001",
		CorrelationID: "corr-autop-status-001",
		RunRef:        "run-ref-autop-status-001",
		QueueRef:      "queue-ref-autop-status-001",
		QueueLimit:    3,
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		result.Queue == nil ||
		result.Run == nil ||
		result.Operator == nil ||
		result.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("result=%+v", result)
	}
	if !result.Operator.QueueLive ||
		len(result.Operator.ActiveRuns) != 1 ||
		len(result.Operator.AgentsInFlight) != 1 ||
		len(result.Operator.ClosureBlockers) != 1 ||
		len(result.Operator.SafeActions) < 2 {
		t.Fatalf("operator=%+v", result.Operator)
	}
	if len(result.Projects) != 1 ||
		result.Projects[0].QueueCount != 1 ||
		result.Projects[0].AgentsInFlight != 1 ||
		len(result.Tasks) != 1 ||
		result.Tasks[0].TaskRef != "task-ref-autop-status-001" ||
		result.Tasks[0].DeliveryRef != "delivery-ref-autop-status-001" ||
		!result.Tasks[0].DecisionRequired ||
		len(result.Tasks[0].EvidenceRefs) != 1 ||
		len(result.Agents) != 1 ||
		result.Agents[0].TaskRef != "task-ref-autop-status-001" ||
		result.Agents[0].ProcessRef != "process-ref-autop-status-001" ||
		result.Agents[0].RuntimeKind != "cli" ||
		result.Agents[0].CapacityLevel != "medium" ||
		result.Agents[0].TotalTokens != 123 ||
		len(result.Agents[0].EvidenceRefs) != 3 {
		t.Fatalf("projects=%+v tasks=%+v agents=%+v", result.Projects, result.Tasks, result.Agents)
	}
	if queue.input.Action != MCPRunQueuePriorityActionRankV0 ||
		queue.input.Limit != 3 ||
		stats.input.RunRef != "run-ref-autop-status-001" {
		t.Fatalf("queue=%+v stats=%+v", queue.input, stats.input)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DiagnosticaPuertosNoConfigurados(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{}).Execute(
		context.Background(),
		MCPAutoprogrammingStatusToolInputV0{RunRef: "run-ref-autop-status-missing-001"},
	)

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoErrorV0 ||
		len(result.Diagnostics) < 2 ||
		len(result.Errores) == 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0DeclaraSuperviseColaAunqueFaltenStatsDeRunV0(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	foundSuperviseQueue := false
	for _, action := range result.Operator.SafeActions {
		if action.Action == "supervise" && action.Scope == "queue" {
			foundSuperviseQueue = true
		}
	}
	if !foundSuperviseQueue {
		t.Fatalf("debe permitir supervision de cola con validacion posterior: %+v", result.Operator.SafeActions)
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "run_stats_required_for_safe_supervision") {
		t.Fatalf("el aviso informativo debe conservarse: %+v", result.Diagnostics)
	}
	for _, item := range result.Operator.SupervisorErrors {
		if item.Code == "run_stats_required_for_safe_supervision" {
			t.Fatalf("el aviso informativo no debe bloquear como error de supervisor: %+v", result.Operator.SupervisorErrors)
		}
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoDeclaraSuperviseSeguroConColaVacia(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{empty: true},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	for _, action := range result.Operator.SafeActions {
		if action.Action == "supervise" || action.Action == "retry" {
			t.Fatalf("no debe recomendar supervision sobre cola vacia/no visible: %+v", result.Operator.SafeActions)
		}
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "queue_empty_or_not_visible") {
		t.Fatalf("falta diagnostico de cola vacia/no visible: %+v", result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusExecutorV0NoRecomiendaSupervisarBucleDeReplan(t *testing.T) {
	result, err := (MCPAutoprogrammingStatusToolExecutorV0{
		Queue: &fakeMCPAutoprogrammingQueueStatusV0{},
		Stats: &fakeMCPAutoprogrammingRunStatusV0{
			stats: &orquestacionnucleoapp.DirectorRunStatsV0{
				RunRef:       "run-ref-autop-status-loop-001",
				Status:       "activa",
				CurrentPhase: "programacion",
				Counts: orquestacionnucleoapp.DirectorRunStatsCountsV0{
					AgentsStarted:       107,
					AgentsStopRequested: 106,
					AgentsStopConfirmed: 105,
					AgentsInFlight:      1,
					Deliveries:          0,
					Reviews:             0,
					ReplanDecisions:     106,
					Closures:            0,
				},
			},
		},
	}).Execute(context.Background(), MCPAutoprogrammingStatusToolInputV0{
		RunRef: "run-ref-autop-status-loop-001",
	})

	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Operator == nil {
		t.Fatalf("operator nil: %+v", result)
	}
	for _, action := range result.Operator.SafeActions {
		if action.RequiresPost {
			t.Fatalf("no debe recomendar acciones POST que puedan amplificar supervision en bucle: %+v", result.Operator.SafeActions)
		}
	}
	if !hasMCPAutoprogrammingDiagnosticCodeV0(result.Diagnostics, "supervisor_replan_amplification_blocked") {
		t.Fatalf("falta diagnostico de bucle: %+v", result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusTransportV0RegistradoEInvocable(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPAutoprogrammingStatusToolNameV0,
		MCPAutoprogrammingStatusToolInputV0{RunRef: "run-ref-autop-status-transport-001"},
	)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result MCPAutoprogrammingStatusToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 || result.Run == nil || result.Queue == nil {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPAutoprogrammingStatusTransportV0AceptaOperatorAdviceTexto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    &fakeMCPAutoprogrammingRunStatusV0{},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(context.Background(), MCPAutoprogrammingStatusToolNameV0, map[string]any{
		"queue_limit":     1,
		"telemetry_flags": "summary",
		"operator_advice": "revisar trabajos sin borrar datos validos",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var result mcpAutoprogrammingStatusTransportResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Estado != MCPAutoprogrammingStatusEstadoOKV0 ||
		len(result.OperatorAdvice) != 1 ||
		result.OperatorAdvice[0].Message != "revisar trabajos sin borrar datos validos" ||
		!result.OperatorAdvice[0].NonBlocking ||
		len(result.Diagnostics) == 0 {
		t.Fatalf("result=%+v advice=%+v diagnostics=%+v", result.MCPAutoprogrammingStatusToolResultV0, result.OperatorAdvice, result.Diagnostics)
	}
}

func TestMCPAutoprogrammingStatusTransportV0AceptaIncludesFlexibles(t *testing.T) {
	stats := &fakeMCPAutoprogrammingRunStatusV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunQueuePriority: &fakeMCPAutoprogrammingQueueStatusV0{},
		DirectorStats:    stats,
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	_, err = transport.CallToolV0(context.Background(), MCPAutoprogrammingStatusToolNameV0, map[string]any{
		"run_ref":                "run-ref-autop-status-flex-001",
		"include_process_refs":   []string{"process_refs"},
		"include_agent_progress": "yes",
		"include_agent_usage":    1,
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !stats.input.IncludeProcessRefs ||
		!stats.input.IncludeAgentProgress ||
		!stats.input.IncludeAgentUsage {
		t.Fatalf("stats input no normalizado: %+v", stats.input)
	}
}
