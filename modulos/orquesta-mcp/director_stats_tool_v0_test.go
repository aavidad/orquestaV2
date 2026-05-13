package orquestamcp

import (
	"context"
	"encoding/json"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaobservability "orquesta/modulos/orquesta-observability"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

func TestMCPDirectorStatsToolDescriptorV0ExponeContratoCompacto(t *testing.T) {
	descriptor := MCPDirectorStatsDescriptorV0()

	if descriptor.Name != MCPDirectorStatsToolNameV0 ||
		descriptor.Version != MCPDirectorStatsToolVersionV0 ||
		descriptor.ResourceURI != MCPDirectorStatsResourceURIV0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	if descriptor.InputSchema == "" ||
		descriptor.Output == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor incompleto=%+v", descriptor)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, descriptor, 1400)
}

func TestMCPDirectorStatsToolExecutorV0DevuelveStatsDeRunStore(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-001")
	registry := orquestacionnucleoapp.NewInMemoryAgentProcessRegistryV0()
	if err := registry.RecordAgentProcessV0(context.Background(), orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          run.RunID,
		AgentRequestID: "agent-ref-stats-001",
		ProcessRef:     "process-ref-mcp-stats-001",
		SessionRef:     "session-ref-mcp-stats-001",
		LaunchRef:      "launch-ref-mcp-stats-001",
		ReadinessRef:   "readiness-ref-mcp-stats-001",
		EvidenceRefs:   []string{"evidence-ref-mcp-stats-process-001"},
	}); err != nil {
		t.Fatalf("record process: %v", err)
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:        orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ProcessRegistry: registry,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:          "request-ref-mcp-director-stats-001",
		CorrelationID:      "corr-mcp-director-stats-001",
		RunRef:             run.RunID,
		IncludeProcessRefs: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != run.RunID ||
		result.Stats == nil ||
		result.DecisionContext == nil {
		t.Fatalf("result=%+v", result)
	}
	if result.Stats.SchemaVersion != orquestacionnucleoapp.DirectorRunStatsSchemaVersionV0 ||
		result.Stats.Counts.TasksTotal != 3 ||
		result.Stats.Counts.AgentsStarted != 1 ||
		result.Stats.Counts.AgentsControlRegistered != 1 ||
		result.Stats.Closure.Status != orquestacionnucleoapp.DirectorClosureStatusBlockedV0 {
		t.Fatalf("stats=%+v", result.Stats)
	}
	if !containsStringMCPTestV0(result.Stats.Closure.BlockedBy, orquestacionnucleoapp.DirectorClosureBlockedByContratosV0) ||
		!containsStringMCPTestV0(result.Stats.Closure.BlockedBy, orquestacionnucleoapp.DirectorClosureBlockedByValidacionFinalV0) {
		t.Fatalf("closure=%+v", result.Stats.Closure)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-001")
	if agent.Process == nil || agent.Process.SessionRef != "session-ref-mcp-stats-001" {
		t.Fatalf("agent=%+v", agent)
	}
	if err := orquestaobservability.ValidateDirectorDecisionContextV0(*result.DecisionContext); err != nil {
		t.Fatalf("decision context invalido: %v\n%+v", err, result.DecisionContext)
	}
	if result.DecisionContext.Lifecycle.AgentsRunning != 1 ||
		result.DecisionContext.Lifecycle.AgentsFailed != 0 ||
		len(result.DecisionContext.Agents) != 2 ||
		result.DecisionContext.Agents[0].SessionRef == "" ||
		len(result.DecisionContext.Blockers) == 0 ||
		len(result.DecisionContext.Activity) == 0 {
		t.Fatalf("decision context incompleto=%+v", result.DecisionContext)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, result, 12000)
}

func TestMCPDirectorStatsToolExecutorV0DevuelveIssuesPublicos(t *testing.T) {
	result, err := (MCPDirectorStatsToolExecutorV0{}).Execute(
		context.Background(),
		MCPDirectorStatsToolInputV0{RequestID: "request-ref-mcp-director-stats-invalid-001"},
	)
	if err != nil {
		t.Fatalf("Execute invalid: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Field != "run_ref" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0ResuelveRunPorJobExterno(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-job-001")
	source := mcpDirectorExternalJobStatsSourceForTestV0{
		Stats: MCPDirectorExternalJobStatsV0{
			AppRef:       "opes",
			JobRef:       "job-ref-opes-001",
			WorkKind:     "draft_content_block",
			ChangeRef:    "opes-job-job-ref-opes-001",
			RunRef:       run.RunID,
			TaskRef:      "task-ref-opes-001",
			AgentRef:     "agent-ref-opes-001",
			Status:       "running",
			DeliveryRefs: []string{"delivery-ref-opes-001"},
		},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ExternalJobSource: source,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:      "request-ref-mcp-director-stats-job-001",
		CorrelationID:  "corr-mcp-director-stats-job-001",
		AppRef:         "opes",
		ExternalJobRef: "job-ref-opes-001",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != run.RunID ||
		result.ExternalJob == nil ||
		result.ExternalJob.JobRef != "job-ref-opes-001" ||
		result.ExternalJob.TaskRef != "task-ref-opes-001" ||
		result.Stats == nil {
		t.Fatalf("result=%+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, result, 12000)
}

func TestMCPDirectorStatsToolExecutorV0RecuperaRunCanonicoSiRunRefObsoleto(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-job-canonico-001")
	source := mcpDirectorExternalJobStatsSourceForTestV0{
		Stats: MCPDirectorExternalJobStatsV0{
			AppRef:   "opes",
			JobRef:   "job-ref-opes-canonico-001",
			WorkKind: "draft_content_block",
			RunRef:   run.RunID,
			TaskRef:  "task-ref-opes-canonico-001",
			AgentRef: "agent-ref-opes-canonico-001",
			Status:   "running",
		},
	}

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ExternalJobSource: source,
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:      "request-ref-mcp-director-stats-job-canonico-001",
		CorrelationID:  "corr-mcp-director-stats-job-canonico-001",
		RunRef:         "run-ref-obsoleto",
		AppRef:         "opes",
		ExternalJobRef: "job-ref-opes-canonico-001",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPDirectorStatsEstadoOKV0 ||
		result.RunRef != run.RunID ||
		result.ExternalJob == nil ||
		result.ExternalJob.RunRef != run.RunID {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPDirectorStatsToolExecutorV0IncluyeUsoDeAgentesOptIn(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-usage-001")

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		AgentUsageSource: mcpDirectorStatsUsageSourceForTestV0{
			Observations: []orquestacionnucleoapp.AgentUsageStatsObservationV0{{
				AgentRequestID: "agent-ref-stats-001",
				RuntimeKind:    "codex",
				ConnectorRef:   "connector-ref-codex-001",
				ProfileRef:     "profile-ref-codex-001",
				ModelAlias:     "gpt-5.5",
				CapacityLevel:  "xhigh",
				QuotaStatus:    orquestacionnucleoapp.DirectorAgentUsageQuotaAvailableV0,
				QuotaRemaining: 90,
				QuotaLimit:     100,
				TotalTokens:    1500,
			}},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:         "request-ref-mcp-director-stats-usage-001",
		CorrelationID:     "corr-mcp-director-stats-usage-001",
		RunRef:            run.RunID,
		IncludeAgentUsage: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-001")
	if agent.Usage == nil ||
		agent.Usage.ModelAlias != "gpt-5.5" ||
		agent.Usage.QuotaRemaining != 90 ||
		agent.Usage.TotalTokens != 1500 {
		t.Fatalf("usage=%+v", agent.Usage)
	}
}

func TestMCPDirectorStatsToolExecutorV0DecisionContextCompletoParaDirector(t *testing.T) {
	run := mcpDirectorStatsRunForTestV0(t, "run-mcp-director-stats-context-001")
	run.Agents = []string{"agent-ref-stats-001", "agent-ref-stats-002", "agent-ref-stats-003"}
	run.StartedAgents = []string{"agent-ref-stats-001", "agent-ref-stats-002", "agent-ref-stats-003"}
	run.FailedAgents = []string{"agent-ref-stats-002"}
	run.StoppedAgents = []string{"agent-ref-stats-003"}
	run.AgentStopRequests = []string{
		orquestacoreworkflow.AgentStopRequestProjectionRefV0(orquestacoreworkflow.AgentStopRequestedPayloadV0{
			AgentRequestID: "agent-ref-stats-003",
			ReasonCode:     "run_stop_requested",
			Summary:        "Parada solicitada por control de run.",
		}),
	}
	run.ConfirmedStoppedAgents = []string{"agent-ref-stats-003"}
	run.ReworkRequests = []string{"rework-ref-mcp-director-stats-001"}
	run.ReplanDecisions = []string{"replan-ref-mcp-director-stats-001"}
	for index := range run.Phases {
		if run.Phases[index].ID == orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
			run.Phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			run.Phases[index].OpenedAt = "2026-05-10T08:00:00Z"
		}
	}
	observation := mcpDirectorStatsProgressObservationForTestV0(run.RunID)
	observation.Report.Status = orquestaruntime.AgentStalledV0
	observation.Report.NoProgressTicks = 4

	result, err := (MCPDirectorStatsToolExecutorV0{
		RunStore: orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		ProgressSource: mcpDirectorStatsProgressSourceForTestV0{
			Observations: []orquestacionnucleoapp.AgentProgressObservationV0{observation},
		},
	}).Execute(context.Background(), MCPDirectorStatsToolInputV0{
		RequestID:            "request-ref-mcp-director-stats-context-001",
		CorrelationID:        "corr-mcp-director-stats-context-001",
		RunRef:               run.RunID,
		OccurredAt:           "2026-05-10T09:00:00Z",
		IncludeAgentProgress: true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	context := result.DecisionContext
	if context == nil {
		t.Fatalf("sin decision_context: %+v", result)
	}
	if err := orquestaobservability.ValidateDirectorDecisionContextV0(*context); err != nil {
		t.Fatalf("decision context invalido: %v\n%+v", err, context)
	}
	if context.Lifecycle.AgentsRunning != 1 ||
		context.Lifecycle.AgentsFailed != 1 ||
		context.Lifecycle.AgentsStopped != 1 ||
		context.Quietness.MaxNoProgressTicks != 4 ||
		context.ReworkReplan.ReworkRequests != 1 ||
		context.ReworkReplan.ReplanDecisions != 1 ||
		!context.Closure.Blocked {
		t.Fatalf("decision context incompleto=%+v", context)
	}
	if !mcpDirectorContextHasCurrentPhaseDurationV0(context, "programacion", 3600) {
		t.Fatalf("phases sin duracion actual: %+v", context.Phases)
	}
	agent := mcpDirectorStatsAgentForTestV0(t, *result.Stats, "agent-ref-stats-003")
	if agent.StopReasonCode != "run_stop_requested" ||
		agent.StopReasonSource != orquestacionnucleoapp.DirectorAgentStopReasonSourceStopRequestV0 {
		t.Fatalf("agent stop reason=%+v", agent)
	}
}

func TestMCPTransportV0DirectorStatsQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(
		context.Background(),
		MCPDirectorStatsToolNameV0,
		MCPDirectorStatsToolInputV0{RunRef: "run-ref-mcp-director-stats-unbound-001"},
	)
	if err != nil {
		t.Fatalf("call director stats unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 {
		t.Fatalf("director stats debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}
