package orquestaappcodexstack

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestCodexStackRunSupervisorDrainRequestV0AplicaPresupuestoConservadorPorDefecto(t *testing.T) {
	request := codexStackRunSupervisorDrainRequestV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:   "run-ref-budget-default-001",
		MaxTicks: 1,
	})

	if request.MaxBursts != 1 ||
		request.MaxStepsPerBurst != 1 ||
		request.MaxDispatchesPerWait != 1 ||
		request.MaxCommands != 1 ||
		request.MaxOutboxPerCycle != 1 ||
		request.MaxDecisionCycles != 1 ||
		request.MaxExternalWaits != 1 {
		t.Fatalf("presupuesto por defecto no acotado: %+v", request)
	}
}

func TestCodexStackRunSupervisorDrainRequestV0ConservaPresupuestoExplicito(t *testing.T) {
	request := codexStackRunSupervisorDrainRequestV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RunRef:               "run-ref-budget-explicit-001",
		MaxBursts:            3,
		MaxStepsPerBurst:     4,
		MaxDispatchesPerWait: 5,
		MaxCommands:          6,
		MaxOutboxPerCycle:    7,
		MaxDecisionCycles:    8,
		MaxExternalWaits:     9,
	})

	if request.MaxBursts != 3 ||
		request.MaxStepsPerBurst != 4 ||
		request.MaxDispatchesPerWait != 5 ||
		request.MaxCommands != 6 ||
		request.MaxOutboxPerCycle != 7 ||
		request.MaxDecisionCycles != 8 ||
		request.MaxExternalWaits != 9 {
		t.Fatalf("presupuesto explicito no conservado: %+v", request)
	}
}

func TestCodexStackRunSupervisorCommandV0AplicaPresupuestoConservadorPorDefecto(t *testing.T) {
	command := codexStackRunSupervisorCommandV0(orquestamcp.MCPRunSupervisorToolInputV0{
		QueueRef: "queue-ref-budget-default-001",
	})

	if command.DrainLimits.MaxBursts != 1 ||
		command.DrainLimits.MaxStepsPerBurst != 1 ||
		command.DrainLimits.MaxDispatchesPerWait != 1 ||
		command.DrainLimits.MaxCommands != 1 ||
		command.DrainLimits.MaxOutboxPerCycle != 1 ||
		command.DrainLimits.MaxDecisionCycles != 1 ||
		command.DrainLimits.MaxExternalWaits != 1 {
		t.Fatalf("presupuesto por defecto del coordinador no acotado: %+v", command.DrainLimits)
	}
}

func TestCodexStackRunSupervisorCommandV0RespetaMaxTicksDeEntrada(t *testing.T) {
	command := codexStackRunSupervisorCommandV0(orquestamcp.MCPRunSupervisorToolInputV0{
		RequestID:            "request-ref-run-supervisor-max-ticks-001",
		QueueRef:             DefaultRunQueueRefV0,
		MaxTicks:             3,
		MaxRunsPerTick:       2,
		MaxExecutions:        2,
		MaxDispatchesPerWait: 6,
		MaxOutboxPerCycle:    12,
	})

	if command.MaxTicks != 3 ||
		command.MaxRunsPerTick != 2 ||
		command.MaxExecutions != 2 ||
		command.DrainLimits.MaxDispatchesPerWait != 6 ||
		command.DrainLimits.MaxOutboxPerCycle != 12 {
		t.Fatalf("command=%+v", command)
	}
}

func TestCodexStackRunSupervisorCommandV0ConservaMaxExternalWaitsExplicito(t *testing.T) {
	command := codexStackRunSupervisorCommandV0(orquestamcp.MCPRunSupervisorToolInputV0{
		QueueRef:         "queue-ref-budget-explicit-waits-001",
		MaxExternalWaits: 9,
	})

	if command.DrainLimits.MaxExternalWaits != 9 {
		t.Fatalf("max_external_waits explicito no conservado: %+v", command.DrainLimits)
	}
}

func TestCodexStackRunSupervisorErrorResultMCPV0ExponeDiagnosticoPublicoDelDrain(t *testing.T) {
	partial := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopRuntimeErrorV0,
		Ticks:      1,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:       CodexSupervisorRuntimeFailedV0,
			SessionRef:   "run-ref-diagnostics-001",
			EvidenceRefs: []string{"evidence-ref-drain"},
			Diagnostics: []orquestaruncoordinator.RunDrainDiagnosticV0{{
				Kind:        "outbox_batch_dispatch_error",
				Status:      "error",
				RunRef:      "run-ref-diagnostics-001",
				Error:       "codex launch failed: /home/alberto/secreto/token.txt",
				MessageType: "LaunchRuntimeAgent",
				MessageID:   "message-ref-001",
				Issues:      1,
			}},
		},
	}

	result := codexStackRunSupervisorErrorResultMCPV0(
		orquestamcp.MCPRunSupervisorToolInputV0{},
		partial,
		errors.New("runtime error"),
	)

	if len(result.Diagnostics) == 0 {
		t.Fatalf("diagnostics vacios: %+v", result)
	}
	if result.Diagnostics[0].Code != "outbox_batch_dispatch_error" {
		t.Fatalf("diagnostic code=%q", result.Diagnostics[0].Code)
	}
	if !strings.Contains(result.Diagnostics[0].Message, "LaunchRuntimeAgent") ||
		!strings.Contains(result.Diagnostics[0].Message, "target=unknown") {
		t.Fatalf("diagnostic message sin contexto publico: %q", result.Diagnostics[0].Message)
	}
	if strings.Contains(result.Diagnostics[0].Message, "capacity-missing") {
		t.Fatalf("diagnostic inventa target de capacidad: %q", result.Diagnostics[0].Message)
	}
	if strings.Contains(result.Diagnostics[0].Message, "/home/alberto") ||
		strings.Contains(result.Diagnostics[0].Message, "token.txt") {
		t.Fatalf("diagnostic filtra path local sensible: %q", result.Diagnostics[0].Message)
	}
}

func TestCodexStackRunSupervisorErrorResultMCPV0DistingueErrorConAgenteVivo(t *testing.T) {
	partial := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopRuntimeErrorV0,
		Ticks:      1,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeRunningLiveV0,
			SessionRef: "run-ref-live-error-001",
			AgentRef:   "agent-ref-live-error-001",
			ProcessRef: "process-ref-live-error-001",
			EvidenceRefs: []string{
				"evidence-ref-codex-supervisor-process-live",
			},
		},
	}

	result := codexStackRunSupervisorErrorResultMCPV0(
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: "run-ref-live-error-001"},
		partial,
		errors.New("transicion_invalida: idempotency_key"),
	)

	if result.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		result.Last.Status != string(CodexSupervisorRuntimeRunningLiveV0) ||
		result.StopReason != string(CodexSupervisorStopRuntimeErrorV0) {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "supervisor_transition_error_but_agents_live" ||
		!strings.Contains(result.Diagnostics[0].Message, "action=wait_agents_or_retry_supervise") {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
	if !codexStackStringInSetForTestV0(result.NextActions, "wait_agents") ||
		!codexStackStringInSetForTestV0(result.NextActions, "retry_supervise") ||
		!codexStackStringInSetForTestV0(result.NextActions, "do_not_relaunch_same_run_ref_while_process_live") {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
}

func TestCodexStackRunSupervisorErrorResultMCPV0ExponeReviewPayloadInvalidoTrasEntrega(t *testing.T) {
	partial := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopRuntimeErrorV0,
		Ticks:      1,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeFailedV0,
			SessionRef: "run-ref-review-payload-invalid-001",
			AgentRef:   "agent-ref-review-payload-invalid-001",
			EvidenceRefs: []string{
				"delivery-ref-review-payload-invalid-001",
				"evidence-ref-review-gate-payload-compacted",
			},
		},
	}

	result := codexStackRunSupervisorErrorResultMCPV0(
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: "run-ref-review-payload-invalid-001"},
		partial,
		orquestacoreworkflow.ReviewResultErrorV0{
			Code:  orquestacoreworkflow.ErrReviewResultPayloadInvalidoV0,
			Field: "review_result_ref",
		},
	)

	if result.Estado != orquestamcp.MCPRunSupervisorEstadoErrorV0 ||
		result.RunRef != "run-ref-review-payload-invalid-001" ||
		len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "review_result_payload_invalid_after_delivery" ||
		!strings.Contains(result.Diagnostics[0].Message, "retry_review_with_compact_payload") ||
		!codexStackStringInSetForTestV0(result.Diagnostics[0].EvidenceRefs, "delivery-ref-review-payload-invalid-001") ||
		!codexStackStringInSetForTestV0(result.Diagnostics[0].EvidenceRefs, "evidence-ref-review-result-payload-invalid") {
		t.Fatalf("result=%+v", result)
	}
	if !codexStackStringInSetForTestV0(result.NextActions, "preserve_delivery_evidence") ||
		!codexStackStringInSetForTestV0(result.NextActions, "retry_review_with_compact_payload") ||
		!codexStackStringInSetForTestV0(result.NextActions, "do_not_relaunch_agent_for_review_payload_error") {
		t.Fatalf("next_actions=%+v", result.NextActions)
	}
}

func TestCodexStackRunSupervisorEvidenceDiagnosticsMCPV0ExponeStreamFDWarning(t *testing.T) {
	diagnostics := codexStackRunSupervisorEvidenceDiagnosticsMCPV0(
		CodexSupervisorRuntimeSnapshotV0{
			SessionRef: "run-ref-stream-fd-warning-001",
			AgentRef:   "agent-ref-stream-fd-warning-001",
			ProcessRef: "process-ref-stream-fd-warning-001",
			EvidenceRefs: []string{
				"evidence-ref-warning-stream-fd",
			},
		},
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != codexStackCodexRuntimeStreamFDWarningDiagnosticV0 ||
		!strings.Contains(diagnostics[0].Scope, "run:run-ref-stream-fd-warning-001") ||
		!strings.Contains(diagnostics[0].Message, "continue_observing_do_not_relaunch_by_itself") ||
		!codexStackStringInSetForTestV0(diagnostics[0].EvidenceRefs, "evidence-ref-warning-stream-fd") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestCodexStackRunSupervisorEvidenceDiagnosticsMCPV0ExponeQuotaYCapacidad(t *testing.T) {
	diagnostics := codexStackRunSupervisorEvidenceDiagnosticsMCPV0(
		CodexSupervisorRuntimeSnapshotV0{
			SessionRef: "run-ref-quota-001",
			EvidenceRefs: []string{
				"evidence-ref-provider-usage-limit-retry-after",
				"evidence-ref-codex-usage-quota-exhausted",
				"evidence-ref-capacity-limited",
			},
		},
	)

	if !codexStackDiagnosticsContainCodeForTestV0(diagnostics, codexStackCodexProviderQuotaExhaustedDiagnosticV0) ||
		!codexStackDiagnosticsContainCodeForTestV0(diagnostics, codexStackCodexProviderCapacityLimitedDiagnosticV0) {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	quota := codexStackDiagnosticByCodeForTestV0(diagnostics, codexStackCodexProviderQuotaExhaustedDiagnosticV0)
	if !strings.Contains(quota.Message, "wait_for_quota_or_requeue_capacity_limited") ||
		!codexStackStringInSetForTestV0(quota.EvidenceRefs, "evidence-ref-provider-usage-limit-retry-after") ||
		!codexStackStringInSetForTestV0(quota.EvidenceRefs, "evidence-ref-codex-usage-quota-exhausted") {
		t.Fatalf("quota=%+v", quota)
	}
	capacity := codexStackDiagnosticByCodeForTestV0(diagnostics, codexStackCodexProviderCapacityLimitedDiagnosticV0)
	if !strings.Contains(capacity.Message, "wait_for_capacity_or_requeue_capacity_limited") ||
		!codexStackStringInSetForTestV0(capacity.EvidenceRefs, "evidence-ref-capacity-limited") {
		t.Fatalf("capacity=%+v", capacity)
	}
}

func TestCodexStackRunSupervisorErrorResultMCPV0IncluyeDiagnosticosPorEvidencia(t *testing.T) {
	partial := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopRuntimeErrorV0,
		Ticks:      1,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeFailedV0,
			SessionRef: "run-ref-error-evidence-001",
			EvidenceRefs: []string{
				"evidence-ref-warning-stream-fd",
				"evidence-ref-provider-quota-exhausted",
			},
		},
	}

	result := codexStackRunSupervisorErrorResultMCPV0(
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: "run-ref-error-evidence-001"},
		partial,
		errors.New("runtime error"),
	)

	if !codexStackDiagnosticsContainCodeForTestV0(result.Diagnostics, codexStackCodexRuntimeStreamFDWarningDiagnosticV0) ||
		!codexStackDiagnosticsContainCodeForTestV0(result.Diagnostics, codexStackCodexProviderQuotaExhaustedDiagnosticV0) {
		t.Fatalf("diagnostics=%+v", result.Diagnostics)
	}
}

func TestCodexStackRunSupervisorRequestedNotStartedDiagnosticsMCPV0ExponeExternalWorkQAVisual(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-opes-qa-visual-remota-tcae-psicologo-asg-operario-20260626"
	if err := stack.Stores.RunStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID:      runRef,
		ProjectRef: "opes",
		AppSpecRef: "app-spec-external-work-opes-qa-visual",
		Status:     orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		Tasks:      []string{"task-ref-qa-visual-remota-tcae-001"},
		Agents:     []string{"agent-ref-qa-visual-remota-tcae-001"},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	result := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopStoppedV0,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeStoppedV0,
			SessionRef: runRef,
			EvidenceRefs: []string{
				"operational-director-plan-state:blocked",
				"wait-subagents-terminal-without-delivery",
			},
		},
	}

	diagnostics := stack.codexStackRunSupervisorRequestedNotStartedDiagnosticsMCPV0(
		ctx,
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
		result,
	)
	output := codexStackRunSupervisorWithRequestedNotStartedActionsMCPV0(
		orquestamcp.NewMCPRunSupervisorOKResultV0(
			orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
			runRef,
			string(CodexSupervisorStopStoppedV0),
			1,
			codexStackRunSupervisorSnapshotMCPV0(result.Last),
			nil,
		),
		diagnostics,
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != codexStackExternalWorkAgentRequestedNotStartedDiagnosticV0 ||
		!strings.Contains(diagnostics[0].Message, "requested_agents=1") ||
		!strings.Contains(diagnostics[0].Message, "started_agents=0") ||
		!strings.Contains(diagnostics[0].Message, "retry_materialization_or_check_capacity_auth_runtime_queue_outbox_policy") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if !codexStackStringInSetForTestV0(output.NextActions, "retry_materialization") ||
		!codexStackStringInSetForTestV0(output.NextActions, "check_capacity_auth_runtime_queue_outbox_policy") ||
		!codexStackStringInSetForTestV0(output.NextActions, "do_not_mark_completed_without_agent_start_or_delivery") {
		t.Fatalf("next_actions=%+v", output.NextActions)
	}
}

func TestCodexStackRunSupervisorStoppedNoDeliveryDiagnosticsMCPV0ExponeExternalWorkVisualSinEntrega(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-opes-psicologo-rework-visual-019-030-20260626"
	if err := stack.Stores.RunStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID:      runRef,
		ProjectRef: "opes",
		AppSpecRef: "app-spec-external-work-opes-rework-visual",
		Status:     orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		Tasks:      []string{"task-ref-rework-visual-psicologo-019-030-001"},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	result := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopStoppedV0,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeStoppedV0,
			SessionRef: runRef,
			EvidenceRefs: []string{
				"evidence-ref-external-work-run-started",
				"evidence-ref-external-work-run-queued",
				"evidence-ref-run-coordinator-executed",
			},
		},
	}

	diagnostics := stack.codexStackRunSupervisorStoppedNoDeliveryDiagnosticsMCPV0(
		ctx,
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
		result,
	)
	output := codexStackRunSupervisorWithStoppedNoDeliveryActionsMCPV0(
		orquestamcp.NewMCPRunSupervisorOKResultV0(
			orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
			runRef,
			string(CodexSupervisorStopStoppedV0),
			1,
			codexStackRunSupervisorSnapshotMCPV0(result.Last),
			nil,
		),
		diagnostics,
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != codexStackExternalWorkStoppedNoDeliveryDiagnosticV0 ||
		!strings.Contains(diagnostics[0].Message, "agents_requested=0") ||
		!strings.Contains(diagnostics[0].Message, "deliveries=0") ||
		!strings.Contains(diagnostics[0].Message, "relaunch_or_replan_external_work_with_causal_error") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if !codexStackStringInSetForTestV0(output.NextActions, "relaunch_or_replan_external_work_with_causal_error") ||
		!codexStackStringInSetForTestV0(output.NextActions, "inspect_external_work_payload_and_runtime_binding") ||
		!codexStackStringInSetForTestV0(output.NextActions, "do_not_mark_completed_without_agent_start_or_delivery") {
		t.Fatalf("next_actions=%+v", output.NextActions)
	}
}

func TestCodexStackRunSupervisorNoAgentMaterializedDiagnosticsMCPV0ExponeExternalWorkDoneSinAgentes(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-opes-tractorista-tema-001-done-sin-agentes"
	if err := stack.Stores.RunStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID:      runRef,
		ProjectRef: "opes",
		AppSpecRef: "app-spec-external-work-opes-tractorista",
		Status:     "done",
		Tasks:      []string{"task-ref-tractorista-001"},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	result := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopDoneV0,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeDoneV0,
			SessionRef: runRef,
			EvidenceRefs: []string{
				"evidence-ref-external-work-run-started",
				"evidence-ref-external-work-run-queued",
				"evidence-ref-run-coordinator-executed",
			},
		},
	}

	diagnostics := stack.codexStackRunSupervisorNoAgentMaterializedDiagnosticsMCPV0(
		ctx,
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
		result,
	)
	output := codexStackRunSupervisorWithNoAgentMaterializedActionsMCPV0(
		orquestamcp.NewMCPRunSupervisorOKResultV0(
			orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
			runRef,
			string(CodexSupervisorStopDoneV0),
			1,
			codexStackRunSupervisorSnapshotMCPV0(result.Last),
			nil,
		),
		diagnostics,
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != codexStackExternalWorkNoAgentMaterializedDiagnosticV0 ||
		!strings.Contains(diagnostics[0].Message, "terminal_status=done") ||
		!strings.Contains(diagnostics[0].Message, "deliveries=0") ||
		!strings.Contains(diagnostics[0].Message, "relaunch_or_replan_external_work_with_causal_error") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if !codexStackStringInSetForTestV0(output.NextActions, "relaunch_or_replan_external_work_with_causal_error") ||
		!codexStackStringInSetForTestV0(output.NextActions, "inspect_external_work_payload_and_runtime_binding") ||
		!codexStackStringInSetForTestV0(output.NextActions, "do_not_mark_completed_without_agent_start_or_delivery") {
		t.Fatalf("next_actions=%+v", output.NextActions)
	}
}

func TestCodexStackRunSupervisorStopPendingDispatchDiagnosticsMCPV0ExponeMaterializacionTardia(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-opes-integracion-social-b-t014-rework-productivo-20260627-19023i"
	agentRef := "agent-ref-integracion-social-t014-padre"
	if err := stack.Stores.RunStore.SaveRunV0(ctx, orquestacoreworkflow.OrchestrationRunV0{
		RunID:         runRef,
		ProjectRef:    "opes",
		AppSpecRef:    "app-spec-external-work-opes-integracion-social",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		Tasks:         []string{"task-ref-integracion-social-t014"},
		Agents:        []string{agentRef},
		StartedAgents: []string{agentRef},
	}); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if _, err := stack.Stores.RunControl.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "opes",
		Reason:       "descartar ola contaminada tras dispatch tardio",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-mcp-run-control-checkpoint-recorded"},
	}); err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	result := CodexSupervisorResultV0{
		StopReason: CodexSupervisorStopPendingV0,
		Last: CodexSupervisorRuntimeSnapshotV0{
			Status:     CodexSupervisorRuntimeRunningLiveV0,
			SessionRef: runRef,
			AgentRef:   agentRef,
			ProcessRef: "process-ref-integracion-social-t014-padre",
			EvidenceRefs: []string{
				"evidence-ref-codex-supervisor-process-live",
			},
		},
	}

	diagnostics := stack.codexStackRunSupervisorStopPendingDispatchDiagnosticsMCPV0(
		ctx,
		orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
		result,
	)
	output := codexStackRunSupervisorWithStopPendingDispatchActionsMCPV0(
		orquestamcp.NewMCPRunSupervisorOKResultV0(
			orquestamcp.MCPRunSupervisorToolInputV0{RunRef: runRef},
			runRef,
			string(CodexSupervisorStopPendingV0),
			1,
			codexStackRunSupervisorSnapshotMCPV0(result.Last),
			nil,
		),
		diagnostics,
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != codexStackStopPendingDispatchInProgressDiagnosticV0 ||
		!strings.Contains(diagnostics[0].Message, "control_status=stop_requested") ||
		!strings.Contains(diagnostics[0].Message, "started_agents=1") ||
		!strings.Contains(diagnostics[0].Message, "wait_agents_or_confirm_stop_do_not_kill_useful_delivery") ||
		!codexStackStringInSetForTestV0(diagnostics[0].EvidenceRefs, "evidence-ref-stop-pending-dispatch-in-progress") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
	if !codexStackStringInSetForTestV0(output.NextActions, "wait_agents_or_confirm_stop") ||
		!codexStackStringInSetForTestV0(output.NextActions, "observe_run_control_and_director_stats") ||
		!codexStackStringInSetForTestV0(output.NextActions, "do_not_kill_live_agents_with_useful_output") {
		t.Fatalf("next_actions=%+v", output.NextActions)
	}
}

func TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponePresionWaitingOutbox(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-ref-queue-pressure-ready-001",
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "app-ref-queue-pressure",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 80,
		RequestedBy:   "stack-test",
	}); err != nil {
		t.Fatalf("SetRunPriority ready: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-ref-queue-pressure-running-001",
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "app-ref-queue-pressure",
		Status:        orquestarunqueue.RunStatusRunningV0,
		PriorityScore: 70,
		RequestedBy:   "stack-test",
	}); err != nil {
		t.Fatalf("SetRunPriority running: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-ref-queue-pressure-stopped-001",
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "app-ref-queue-pressure",
		Status:        orquestarunqueue.RunStatusStoppedV0,
		PriorityScore: 10,
		RequestedBy:   "stack-test",
	}); err != nil {
		t.Fatalf("SetRunPriority stopped: %v", err)
	}

	diagnostics := stack.codexStackRunSupervisorQueueDiagnosticsMCPV0(
		ctx,
		orquestamcp.MCPRunSupervisorToolInputV0{QueueRef: DefaultRunQueueRefV0},
		CodexSupervisorResultV0{
			Last: CodexSupervisorRuntimeSnapshotV0{Status: CodexSupervisorRuntimeWaitingOutboxV0},
		},
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != "run_supervisor_queue_pressure" ||
		!strings.Contains(diagnostics[0].Message, "total=3") ||
		!strings.Contains(diagnostics[0].Message, "executable=2") ||
		!strings.Contains(diagnostics[0].Message, "ready=1") ||
		!strings.Contains(diagnostics[0].Message, "running=1") ||
		!strings.Contains(diagnostics[0].Message, "stopped=1") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestCodexStackRunSupervisorQueueDiagnosticsMCPV0ExponeNoExecutionConReady(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        "run-ref-queue-no-execution-ready-001",
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "app-ref-queue-no-execution",
		Status:        orquestarunqueue.RunStatusReadyV0,
		PriorityScore: 90,
		RequestedBy:   "stack-test",
	}); err != nil {
		t.Fatalf("SetRunPriority ready: %v", err)
	}

	diagnostics := stack.codexStackRunSupervisorQueueDiagnosticsMCPV0(
		ctx,
		orquestamcp.MCPRunSupervisorToolInputV0{
			QueueRef:             DefaultRunQueueRefV0,
			MaxTicks:             1,
			MaxRunsPerTick:       6,
			MaxExecutions:        6,
			MaxDispatchesPerWait: 6,
			MaxOutboxPerCycle:    12,
		},
		CodexSupervisorResultV0{
			StopReason: CodexSupervisorStopDoneV0,
			Last: CodexSupervisorRuntimeSnapshotV0{
				Status: CodexSupervisorRuntimeDoneV0,
				EvidenceRefs: []string{
					"evidence-ref-codex-supervisor-stack-global",
					orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
				},
			},
		},
	)

	if len(diagnostics) != 1 ||
		diagnostics[0].Code != "run_supervisor_queue_no_execution_with_ready_candidates" ||
		!strings.Contains(diagnostics[0].Message, "total=1") ||
		!strings.Contains(diagnostics[0].Message, "executable=1") ||
		!strings.Contains(diagnostics[0].Message, "ready=1") ||
		!strings.Contains(diagnostics[0].Message, "max_runs_per_tick=6") ||
		!strings.Contains(diagnostics[0].Message, "max_executions=6") ||
		!strings.Contains(diagnostics[0].Message, "max_dispatches_per_wait=6") ||
		!strings.Contains(diagnostics[0].Message, "max_outbox_per_cycle=12") ||
		!strings.Contains(diagnostics[0].Message, "executions=0") ||
		!strings.Contains(diagnostics[0].Message, "top_candidates=run-ref-queue-no-execution-ready-001:ready:90") ||
		!strings.Contains(diagnostics[0].Message, "action=supervise_with_resident_mode_or_run_ref") {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func codexStackDiagnosticByCodeForTestV0(
	diagnostics []orquestamcp.MCPAutoprogrammingDiagnosticV0,
	code string,
) orquestamcp.MCPAutoprogrammingDiagnosticV0 {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return diagnostic
		}
	}
	return orquestamcp.MCPAutoprogrammingDiagnosticV0{}
}
