package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestaapprunner "orquesta/modulos/orquesta-app-runner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMCPEjecutarOrquestacionAppToolExecutorV0ModoVacioNoEjecutaLoopLegacy(t *testing.T) {
	result, err := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{},
	).Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:  "request-ref-mcp-run-app-empty-mode-001",
		OccurredAt: "2026-05-09T23:59:00Z",
		AppSpec:    validMCPPrepareLargeAppSpecForTestV0(t),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPEjecutarOrquestacionAppEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "director_execution_mode" ||
		result.Errores[0].Code != "legacy_director_loop_required" {
		t.Fatalf("result=%+v", result)
	}
	if result.RoutePolicy.PreferredEntrypoint != MCPArrancarDirectorAppToolNameV0 ||
		result.RoutePolicy.LegacyEntrypoint != MCPEjecutarOrquestacionAppToolNameV0 {
		t.Fatalf("route_policy=%+v", result.RoutePolicy)
	}
}

func TestMCPEjecutarOrquestacionAppToolExecutorV0GoalFirstNoEjecutaLoopLegacy(t *testing.T) {
	result, err := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{},
	).Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:             "request-ref-mcp-run-app-goal-first-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		OccurredAt:            "2026-05-09T23:59:00Z",
		AppSpec:               validMCPPrepareLargeAppSpecForTestV0(t),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPEjecutarOrquestacionAppEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "legacy_director_loop_required" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPEjecutarOrquestacionAppToolExecutorV0ArrancaBootstrap(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	executor := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	result, err := executor.Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:             "request-ref-mcp-run-app-001",
		CorrelationID:         "corr-mcp-run-app-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		RunRef:                "run-mcp-run-app-001",
		ProjectRef:            "project-mcp-run-app-001",
		OccurredAt:            "2026-05-09T23:59:00Z",
		AppSpec:               validMCPPrepareLargeAppSpecForTestV0(t),
		MaxBursts:             10,
		MaxStepsPerBurst:      6,
		MaxDispatchesPerWait:  4,
		MaxCommands:           12,
		MaxOutboxPerCycle:     4,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPEjecutarOrquestacionAppEstadoOKV0 ||
		result.Plan.Units != 11 ||
		result.LoopStatus != string(orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0) {
		t.Fatalf("result=%+v", result)
	}
	if !mcpPrepareStringInSetTestV0(result.StartedAgents, "agent-agenda-bootstrap") {
		t.Fatalf("started=%v", result.StartedAgents)
	}
	if result.DirectorStats == nil ||
		result.DirectorStats.RunRef != result.RunRef ||
		result.DirectorStats.Counts.TasksTotal != result.Plan.Units ||
		result.DirectorStats.Counts.AgentsStarted != len(result.StartedAgents) {
		t.Fatalf("director_stats=%+v result=%+v", result.DirectorStats, result)
	}
	assertMCPEjecutarOrquestacionRefsInternasNeutrasV0(t, result, sink.EventsV0())
}

func TestMCPEjecutarOrquestacionAppToolExecutorV0CompletaAppConReceiptsExternos(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	receipts := &mcpReadyStartedAgentReceiptsForTestV0{ReadyAgents: map[string]bool{}}
	executor := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{
			RunStore:       store,
			EventSink:      sink,
			OutboxLedger:   ledger,
			DeliverySource: receipts,
			ExternalWaiter: receipts,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	result, err := executor.Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:             "request-ref-mcp-run-app-complete-001",
		CorrelationID:         "corr-mcp-run-app-complete-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		RunRef:                "run-mcp-run-app-complete-001",
		ProjectRef:            "project-mcp-run-app-complete-001",
		OccurredAt:            "2026-05-09T23:59:00Z",
		AppSpec:               validMCPPrepareLargeAppSpecForTestV0(t),
		MaxBursts:             120,
		MaxStepsPerBurst:      8,
		MaxDispatchesPerWait:  8,
		MaxCommands:           24,
		MaxOutboxPerCycle:     24,
	})
	if err != nil {
		t.Fatalf("Execute complete: %v", err)
	}
	if result.Estado != MCPEjecutarOrquestacionAppEstadoOKV0 ||
		!result.Progress.Complete ||
		result.Progress.DeliveredUnits != result.Plan.Units {
		t.Fatalf("result=%+v", result)
	}
	if result.ExternalWaits == 0 {
		t.Fatalf("external waits no usados: %+v", result)
	}
}

func TestMCPEjecutarOrquestacionAppToolExecutorV0ActivaDirectorAutonomo(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	policy := &mcpAutonomousDirectorPolicyForTestV0{}
	executor := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{
			RunStore:                 store,
			EventSink:                sink,
			OutboxLedger:             ledger,
			AutonomousDirectorPolicy: policy,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				mcpDirectorCapacityDispatcherForTestV0(store, sink, ledger),
				mcpDirectorAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	result, err := executor.Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:                 "request-ref-mcp-run-app-autonomous-001",
		CorrelationID:             "corr-mcp-run-app-autonomous-001",
		DirectorExecutionMode:     orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		RunRef:                    "run-mcp-run-app-autonomous-001",
		ProjectRef:                "project-mcp-run-app-autonomous-001",
		OccurredAt:                "2026-05-10T10:00:00Z",
		AppSpec:                   validMCPPrepareLargeAppSpecForTestV0(t),
		MaxBursts:                 10,
		MaxStepsPerBurst:          6,
		MaxDispatchesPerWait:      4,
		MaxCommands:               12,
		MaxOutboxPerCycle:         4,
		UseAutonomousDirectorLoop: true,
	})
	if err != nil {
		t.Fatalf("Execute autonomous: %v", err)
	}
	if !policy.Called ||
		result.DirectorLoopStats == nil ||
		result.DirectorLoopStats.Decision.Summary != mcpAutonomousDirectorSummaryForTestV0 {
		t.Fatalf("policy=%+v loop_stats=%+v result=%+v", policy, result.DirectorLoopStats, result)
	}
	if result.DirectorLoopStats.Run.RunRef != result.RunRef ||
		result.DirectorStats == nil ||
		result.DirectorStats.Progress.TasksTotal != result.Plan.Units {
		t.Fatalf("stats=%+v director_stats=%+v", result.DirectorLoopStats, result.DirectorStats)
	}
}

func TestMCPEjecutarOrquestacionAppToolExecutorV0DevuelveErrorPublico(t *testing.T) {
	result, err := NewMCPEjecutarOrquestacionAppToolExecutorV0(
		orquestaapprunner.RunPreparedAppOrchestrationPortsV0{},
	).Execute(context.Background(), MCPEjecutarOrquestacionAppToolInputV0{
		RequestID:             "request-ref-mcp-run-app-invalid-001",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeLegacyDirectorLoopV0,
		AppSpec:               validMCPPrepareLargeAppSpecForTestV0(t),
	})
	if err != nil {
		t.Fatalf("Execute invalid: %v", err)
	}
	if result.Estado != MCPEjecutarOrquestacionAppEstadoErrorV0 || len(result.Errores) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if result.Errores[0].Field != "occurred_at" {
		t.Fatalf("errores=%+v", result.Errores)
	}
}

const mcpAutonomousDirectorSummaryForTestV0 = "mcp autonomous director policy"

type mcpAutonomousDirectorPolicyForTestV0 struct {
	Called bool
}

func (policy *mcpAutonomousDirectorPolicyForTestV0) DecideAutonomousDirectorV0(
	_ context.Context,
	input orquestacionnucleoapp.AutonomousDirectorDecisionInputV0,
) (orquestacionnucleoapp.AutonomousDirectorDecisionV0, error) {
	policy.Called = true
	return orquestacionnucleoapp.AutonomousDirectorDecisionV0{
		TeamSize:             1,
		MaxParallelAgents:    1,
		MaxBursts:            input.Limits.MaxBursts,
		MaxStepsPerBurst:     input.Limits.MaxStepsPerBurst,
		MaxDispatchesPerWait: input.Limits.MaxDispatchesPerWait,
		MaxCommandsPerCycle:  input.Limits.MaxCommandsPerCycle,
		MaxOutboxPerCycle:    input.Limits.MaxOutboxPerCycle,
		Summary:              mcpAutonomousDirectorSummaryForTestV0,
		EvidenceRefs:         []string{"evidence-ref-mcp-autonomous-policy-001"},
	}, nil
}

type mcpReadyStartedAgentReceiptsForTestV0 struct {
	ReadyAgents map[string]bool
}

func (source *mcpReadyStartedAgentReceiptsForTestV0) BuildAgentDeliveryObservationsV0(
	_ context.Context,
	request orquestacionnucleoapp.AgentDeliveryObservationRequestV0,
) ([]orquestacionnucleoapp.AgentDeliveryObservationV0, error) {
	observations := make([]orquestacionnucleoapp.AgentDeliveryObservationV0, 0)
	for _, agentRef := range request.Run.StartedAgents {
		if !source.ReadyAgents[agentRef] {
			continue
		}
		taskRef := strings.Replace(agentRef, "agent-", "task-", 1)
		deliveryRef := strings.Replace(agentRef, "agent-", "ack-", 1)
		if mcpPrepareStringInSetTestV0(request.Run.Deliveries, deliveryRef) {
			continue
		}
		observations = append(observations, orquestacionnucleoapp.AgentDeliveryObservationV0{
			CandidateRef: "delivery-candidate-ref-" + deliveryRef,
			DeliveryRef:  deliveryRef,
			TaskID:       taskRef,
			AgentRef:     agentRef,
			Summary:      "Receipt compacto de prueba MCP.",
			EvidenceRefs: []string{"evidence-ref-mcp-run-app-receipt-001"},
		})
	}
	return observations, nil
}

func (source *mcpReadyStartedAgentReceiptsForTestV0) WaitExternalProgressV0(
	_ context.Context,
	request orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	for _, agentRef := range request.LastResult.Run.StartedAgents {
		source.ReadyAgents[agentRef] = true
	}
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-mcp-run-app-wait-001"},
	}, nil
}

func assertMCPEjecutarOrquestacionRefsInternasNeutrasV0(
	t *testing.T,
	result MCPEjecutarOrquestacionAppToolResultV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) {
	t.Helper()
	refs := []string{result.RunRef, result.Plan.RunRef}
	for _, event := range events {
		refs = append(refs, event.RunID, event.CorrelationID)
	}
	for _, ref := range refs {
		if strings.Contains(strings.ToLower(ref), "mcp") {
			t.Fatalf("ref interna filtra adaptador MCP: %q", ref)
		}
	}
}
