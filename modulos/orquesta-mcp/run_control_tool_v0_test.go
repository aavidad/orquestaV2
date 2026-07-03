package orquestamcp

import (
	"context"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestMCPRunControlDescriptorV0EsAdaptadorFino(t *testing.T) {
	descriptor := MCPRunControlDescriptorV0()
	if descriptor.Name != MCPRunControlToolNameV0 ||
		descriptor.ResourceURI != MCPRunControlResourceURIV0 ||
		descriptor.InputSchema == "" ||
		len(descriptor.Invariantes) == 0 {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	for _, action := range []string{"pause", "resume", "stop", "cancel"} {
		if !strings.Contains(descriptor.InputSchema, action) {
			t.Fatalf("descriptor no declara action %q: %+v", action, descriptor)
		}
	}
	for _, field := range []string{"forced?", "evidence_refs?", "diagnostics?"} {
		if !strings.Contains(descriptor.Output, field) {
			t.Fatalf("descriptor no declara salida operacional %q: %+v", field, descriptor)
		}
	}
}

func TestMCPRunControlExecutorV0DelegaEnPuertoInyectado(t *testing.T) {
	port := &fakeMCPRunControlPortV0{}
	executor := NewMCPRunControlToolExecutorV0(port)

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:      "req-run-control-001",
		CorrelationID:  "corr-run-control-001",
		Action:         " stop ",
		RunRef:         " run-ref-001 ",
		RequestedBy:    " director ",
		Reason:         " cierre operativo ",
		Forced:         true,
		IdempotencyKey: " idem-001 ",
		EvidenceRefs:   []string{" evidence-1 ", "evidence-1", ""},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "stop" ||
		result.Status != string(orquestaruncontrol.RunControlStatusStopRequestedV0) ||
		result.RunRef != "run-ref-001" ||
		!result.CheckpointRecorded ||
		!result.Forced {
		t.Fatalf("result=%+v", result)
	}
	if port.stop.RunRef != "run-ref-001" ||
		port.stop.RequestedBy != "director" ||
		port.stop.IdempotencyKey != "idem-001" ||
		len(port.stop.EvidenceRefs) != 1 {
		t.Fatalf("command=%+v", port.stop)
	}
	if port.checkpoint.RunRef != "run-ref-001" ||
		port.checkpoint.RequestedBy != "director" ||
		port.checkpoint.IdempotencyKey != "idem-001" ||
		!containsStringMCPTestV0(port.checkpoint.EvidenceRefs, "evidence-ref-mcp-run-control-checkpoint-recorded") {
		t.Fatalf("checkpoint=%+v", port.checkpoint)
	}
}

func TestMCPRunControlExecutorV0ErrorConservaEvidenciaV0(t *testing.T) {
	executor := NewMCPRunControlToolExecutorV0(&fakeMCPRunControlPortV0{})

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID: "req-run-control-error-evidence-001",
		Action:    "restart",
		RunRef:    "run-ref-control-error-evidence-001",
		Forced:    true,
		EvidenceRefs: []string{
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
			"",
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		result.Action != "restart" ||
		result.RunRef != "run-ref-control-error-evidence-001" ||
		!result.Forced ||
		len(result.EvidenceRefs) != 1 ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) ||
		len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "action_no_soportada" ||
		result.Diagnostics[0].Scope != "run:run-ref-control-error-evidence-001" ||
		!containsStringMCPTestV0(
			result.Diagnostics[0].EvidenceRefs,
			mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0,
		) ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "action_no_soportada" {
		t.Fatalf("error run-control debe conservar evidencia compacta: %+v", result)
	}
}

func TestMCPRunControlExecutorV0StopForcedNoPublicaStoppedSiGoalBackendSigueActive(t *testing.T) {
	runRef := "run-ref-run-control-goal-active-001"
	goalRef := "goal-ref-run-control-goal-active-001"
	port := &fakeMCPRunControlPortV0{
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			mcpRunControlGoalBackendStatsForTestV0(runRef, goalRef, "active"),
			mcpRunControlGoalBackendStatsForTestV0(runRef, goalRef, "active"),
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "req-run-control-goal-active-001",
		Action:      "stop",
		RunRef:      runRef,
		RequestedBy: "opes",
		Reason:      "parada forzada de ola OPES",
		Forced:      true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStopRequestedV0) ||
		result.FinalStatus != string(orquestaruncontrol.RunControlStatusStopRequestedV0) ||
		result.PreviousStatus != string(orquestaruncontrol.RunControlStatusRunningV0) ||
		result.GoalRef != goalRef ||
		result.GoalStatusBefore != "active" ||
		result.GoalStatusAfter != "active" ||
		result.GoalControlSignalSent ||
		result.GoalControlSignalConfirmed ||
		result.RecommendedAction != "observe_goal_backend_before_declaring_stopped" {
		t.Fatalf("result=%+v", result)
	}
	if len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != "control_not_propagated_to_goal_backend" ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "control_not_propagated_to_goal_backend" {
		t.Fatalf("diagnostics=%+v errores=%+v", result.Diagnostics, result.Errores)
	}
	if !containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-backend-active") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-goal-backend-active") {
		t.Fatalf("evidence_refs=%+v", result.EvidenceRefs)
	}
	if len(goalBackend.inputs) != 2 ||
		goalBackend.inputs[0].RunRef != runRef ||
		!goalBackend.inputs[1].IncludeProcessRefs ||
		!goalBackend.inputs[1].IncludeAgentUsage {
		t.Fatalf("goal backend inputs=%+v", goalBackend.inputs)
	}
}

func TestMCPRunControlExecutorV0StopForcedPermiteTerminalSiGoalBackendYaComplete(t *testing.T) {
	runRef := "run-ref-run-control-goal-complete-001"
	goalRef := "goal-ref-run-control-goal-complete-001"
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			mcpRunControlGoalBackendStatsForTestV0(runRef, goalRef, "active"),
			mcpRunControlGoalBackendStatsForTestV0(runRef, goalRef, "complete"),
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID: "req-run-control-goal-complete-001",
		Action:    "stop",
		RunRef:    runRef,
		Forced:    true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.GoalStatusAfter != "complete" ||
		!result.GoalControlSignalConfirmed ||
		len(result.Diagnostics) != 0 ||
		len(result.Errores) != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumptionCheckpointOnly(t *testing.T) {
	runRef := "run-ref-run-control-goal-checkpoint-only-stop-001"
	goalRef := "goal-ref-run-control-goal-checkpoint-only-stop-001"
	checkpointRef := "artifact-ref-checkpoint:run-control:checkpoint-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-" + goalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cerrar forced stop checkpoint-only como replanificable.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + goalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-run-control-goal-state-seeded"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			mcpRunControlGoalBackendStatsWithUsageForTestV0(runRef, goalRef, "active", 150000, []string{checkpointRef}, nil),
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "req-run-control-goal-checkpoint-only-stop-001",
		Action:      "stop",
		RunRef:      runRef,
		RequestedBy: "orquesta-director",
		Reason:      "forced stop tras checkpoint_only_high_consumption",
		Forced:      true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		port.complete.TargetStatus != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!containsStringMCPTestV0(port.complete.EvidenceRefs, "evidence-ref-run-control-terminal-after-goal-reconcile") ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_forced_stop") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-forced-terminal-reconciled") {
		t.Fatalf("result=%+v", result)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult == nil ||
		state.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		!containsStringMCPTestV0(state.LastResult.ArtifactRefs, checkpointRef) ||
		state.LastClosure == nil ||
		state.LastClosure.Status != orquestagoal.GoalStatusBlockedV0 ||
		!state.LastClosure.NeedsRework ||
		!containsStringMCPTestV0(state.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0) {
		t.Fatalf("state=%+v", state)
	}
}

func TestMCPRunControlExecutorV0StopForcedRespetaUmbralConfiguradoV0(t *testing.T) {
	runRef := "run-ref-run-control-goal-checkpoint-policy-stop-001"
	goalRef := "goal-ref-run-control-goal-checkpoint-policy-stop-001"
	checkpointRef := "artifact-ref-checkpoint:run-control:checkpoint-policy-started-txt"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-" + goalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cerrar forced stop checkpoint-only con politica configurable.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + goalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			mcpRunControlGoalBackendStatsWithUsageForTestV0(runRef, goalRef, "active", 23000, []string{checkpointRef}, nil),
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
		GoalProgressPolicy: MCPAutoprogrammingGoalProgressPolicyV0{
			CheckpointOnlyHighConsumptionTokens: 20000,
		},
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "req-run-control-goal-checkpoint-policy-stop-001",
		Action:      "stop",
		RunRef:      runRef,
		RequestedBy: "orquesta-director",
		Reason:      "forced stop tras checkpoint_only_high_consumption configurado",
		Forced:      true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceCheckpointOnlyHighConsumptionV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunControlExecutorV0StopForcedReconcilesGoalHighConsumptionSinCheckpoint(t *testing.T) {
	runRef := "run-ref-run-control-goal-no-checkpoint-stop-001"
	goalRef := "goal-ref-run-control-goal-no-checkpoint-stop-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-" + goalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cerrar forced stop sin checkpoint como replanificable.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + goalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-run-control-goal-state-seeded"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			mcpRunControlGoalBackendStatsWithUsageForTestV0(runRef, goalRef, "active", 175000, nil, nil),
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "req-run-control-goal-no-checkpoint-stop-001",
		Action:      "stop",
		RunRef:      runRef,
		RequestedBy: "orquesta-director",
		Reason:      "forced stop tras goal_active_no_checkpoint_high_consumption",
		Forced:      true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_forced_stop") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-forced-terminal-reconciled") {
		t.Fatalf("result=%+v", result)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult == nil ||
		state.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		len(state.LastResult.ArtifactRefs) != 0 ||
		state.LastClosure == nil ||
		state.LastClosure.Status != orquestagoal.GoalStatusBlockedV0 ||
		!state.LastClosure.NeedsRework ||
		!containsStringMCPTestV0(state.EvidenceRefs, mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0) {
		t.Fatalf("state=%+v", state)
	}
}

func TestMCPRunControlExecutorV0CancelForcedCompletaRunControlTrasReconciliarGoalV0(t *testing.T) {
	runRef := "run-ref-run-control-goal-cancel-reconcile-001"
	goalRef := "goal-ref-run-control-goal-cancel-reconcile-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-" + goalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cancelar forced stop sin checkpoint como replanificable.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + goalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-run-control-goal-state-cancel-seeded"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			mcpRunControlGoalBackendStatsWithUsageForTestV0(runRef, goalRef, "active", 175000, nil, nil),
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "req-run-control-goal-cancel-reconcile-001",
		Action:      "cancel",
		RunRef:      runRef,
		RequestedBy: "orquesta-director",
		Reason:      "forced cancel tras goal_active_no_checkpoint_high_consumption",
		Forced:      true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusCanceledV0) ||
		port.complete.TargetStatus != orquestaruncontrol.RunControlStatusCanceledV0 ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-terminal-after-goal-reconcile") {
		t.Fatalf("result=%+v complete=%+v", result, port.complete)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult == nil ||
		!strings.Contains(state.LastResult.Summary, "forced cancel reconciled") ||
		strings.Contains(state.LastResult.Summary, "forced stop reconciled") ||
		state.LastClosure == nil ||
		!state.LastClosure.NeedsRework ||
		!containsStringMCPTestV0(state.EvidenceRefs, mcpAutoprogrammingEvidenceNoCheckpointHighConsumptionV0) {
		t.Fatalf("state=%+v", state)
	}
}

func TestMCPRunControlExecutorV0StopForcedReconcilesGoalBackendMissingAfterExternalCleanup(t *testing.T) {
	runRef := "run-ref-run-control-goal-backend-missing-stop-001"
	goalRef := "goal-ref-run-control-goal-backend-missing-stop-001"
	externalGoalRef := "thread-ref-" + goalRef
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Reconciliar estado running tras cleanup externo.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
		EvidenceRefs: []string{"evidence-ref-run-control-goal-state-external-cleanup-seeded"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "req-run-control-goal-backend-missing-stop-001",
		Action:      "stop",
		RunRef:      runRef,
		RequestedBy: "orquesta-director",
		Reason:      "reconciliar cleanup externo de backend goal",
		Forced:      true,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_forced_stop") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v", result)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult == nil ||
		state.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult.GoalRef != goalRef ||
		state.LastResult.ExternalGoalRef != externalGoalRef ||
		state.LastClosure == nil ||
		!state.LastClosure.NeedsRework ||
		!containsStringMCPTestV0(state.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("state=%+v", state)
	}
}

func TestMCPRunControlExecutorV0StopReconcilesExternalCleanupConEvidenciaSinForce(t *testing.T) {
	runRef := "run-ref-run-control-goal-backend-missing-soft-stop-001"
	goalRef := "goal-ref-run-control-goal-backend-missing-soft-stop-001"
	externalGoalRef := "thread-ref-" + goalRef
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: externalGoalRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Reconciliar cleanup externo desde la accion recomendada por status.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "docs"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: externalGoalRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	port := &fakeMCPRunControlPortV0{
		readState:  runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
		stopStatus: orquestaruncontrol.RunControlStatusStoppedV0,
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	executor := MCPRunControlToolExecutorV0{
		Port:             port,
		GoalBackendState: goalBackend,
		GoalStateStore:   goalStates,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:    "req-run-control-goal-backend-missing-soft-stop-001",
		Action:       "stop",
		RunRef:       runRef,
		RequestedBy:  "orquesta-director",
		Reason:       "seguir run_control_reconcile_external_cleanup",
		Forced:       false,
		EvidenceRefs: []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		port.complete.TargetStatus != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-forced-terminal-reconciled") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-external-cleanup-reconciled") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v complete=%+v", result, port.complete)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult == nil ||
		state.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult.GoalRef != goalRef ||
		state.LastResult.ExternalGoalRef != externalGoalRef ||
		strings.Contains(state.LastResult.Summary, "forced ") ||
		!strings.Contains(state.LastResult.Summary, "external cleanup") ||
		state.LastClosure == nil ||
		!state.LastClosure.NeedsRework {
		t.Fatalf("state=%+v", state)
	}
}

func TestMCPRunControlExecutorV0ValidaAction(t *testing.T) {
	result, err := NewMCPRunControlToolExecutorV0(&fakeMCPRunControlPortV0{}).Execute(
		context.Background(),
		MCPRunControlToolInputV0{RequestID: "request-ref-run-control-action-001", Action: "restart", RunRef: "run-ref-001"},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != "action_no_soportada" {
		t.Fatalf("result=%+v", result)
	}
}

func TestMCPRunControlExecutorV0ResuelveRunPorJobExterno(t *testing.T) {
	port := &fakeMCPRunControlPortV0{}
	executor := MCPRunControlToolExecutorV0{
		Port: port,
		ExternalJobSource: mcpDirectorExternalJobStatsSourceForTestV0{
			Stats: MCPDirectorExternalJobStatsV0{
				AppRef:   "opes",
				JobRef:   "job-ref-opes-001",
				RunRef:   "run-ref-opes-001",
				TaskRef:  "task-ref-opes-001",
				AgentRef: "agent-ref-opes-001",
				Status:   "running",
			},
		},
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:      "req-run-control-opes-001",
		CorrelationID:  "corr-run-control-opes-001",
		Action:         "pause",
		AppRef:         "opes",
		ExternalJobRef: "job-ref-opes-001",
		RequestedBy:    "opes",
		Reason:         "pausa solicitada desde job OPES",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.RunRef != "run-ref-opes-001" ||
		result.Status != string(orquestaruncontrol.RunControlStatusPausedV0) ||
		port.pause.RunRef != "run-ref-opes-001" ||
		!containsStringMCPTestV0(port.pause.EvidenceRefs, "job-ref-opes-001") {
		t.Fatalf("result=%+v pause=%+v", result, port.pause)
	}
}

type fakeMCPRunControlPortV0 struct {
	pause      orquestaruncontrol.PauseRunCommandV0
	resume     orquestaruncontrol.ResumeRunCommandV0
	stop       orquestaruncontrol.StopRunCommandV0
	cancel     orquestaruncontrol.CancelRunCommandV0
	checkpoint orquestaruncontrol.RecordRunCheckpointCommandV0
	complete   orquestaruncontrol.CompleteRunControlCommandV0
	readState  orquestaruncontrol.RunControlStateV0
	stopStatus orquestaruncontrol.RunControlStatusV0
}

func (fake *fakeMCPRunControlPortV0) ReadRunControlStateV0(
	_ context.Context,
	request orquestaruncontrol.RunControlReadRequestV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	if fake.readState.RunRef != "" {
		return fake.readState, nil
	}
	return runControlStateForMCPTestV0(request.RunRef, orquestaruncontrol.RunControlStatusRunningV0, false), nil
}

func (fake *fakeMCPRunControlPortV0) PauseRunV0(
	_ context.Context,
	command orquestaruncontrol.PauseRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.pause = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusPausedV0, false), nil
}

func (fake *fakeMCPRunControlPortV0) ResumeRunV0(
	_ context.Context,
	command orquestaruncontrol.ResumeRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.resume = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusRunningV0, false), nil
}

func (fake *fakeMCPRunControlPortV0) StopRunV0(
	_ context.Context,
	command orquestaruncontrol.StopRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.stop = command
	status := fake.stopStatus
	if status == "" {
		status = orquestaruncontrol.RunControlStatusStopRequestedV0
	}
	return runControlStateForMCPTestV0(command.RunRef, status, command.Forced), nil
}

func (fake *fakeMCPRunControlPortV0) CancelRunV0(
	_ context.Context,
	command orquestaruncontrol.CancelRunCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.cancel = command
	return runControlStateForMCPTestV0(command.RunRef, orquestaruncontrol.RunControlStatusCancelRequestedV0, command.Forced), nil
}

type fakeMCPRunControlGoalBackendStateV0 struct {
	inputs  []MCPDirectorStatsToolInputV0
	results []MCPDirectorStatsToolResultV0
}

func (fake *fakeMCPRunControlGoalBackendStateV0) Execute(
	_ context.Context,
	input MCPDirectorStatsToolInputV0,
) (MCPDirectorStatsToolResultV0, error) {
	fake.inputs = append(fake.inputs, input)
	index := len(fake.inputs) - 1
	if index >= len(fake.results) {
		index = len(fake.results) - 1
	}
	if index < 0 {
		return MCPDirectorStatsToolResultV0{Estado: MCPDirectorStatsEstadoErrorV0}, nil
	}
	result := fake.results[index]
	result.RunRef = input.RunRef
	return result, nil
}

func mcpRunControlGoalBackendStatsForTestV0(
	runRef string,
	goalRef string,
	status string,
) MCPDirectorStatsToolResultV0 {
	return MCPDirectorStatsToolResultV0{
		Estado: MCPDirectorStatsEstadoOKV0,
		RunRef: runRef,
		Goal: &MCPDirectorGoalStatsV0{
			RunRef:          runRef,
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + goalRef,
			Status:          status,
			EvidenceRefs:    []string{"evidence-ref-goal-backend-" + status},
		},
	}
}

func mcpRunControlGoalBackendStatsWithUsageForTestV0(
	runRef string,
	goalRef string,
	status string,
	tokens int64,
	artifactRefs []string,
	domainReceiptRefs []string,
) MCPDirectorStatsToolResultV0 {
	result := mcpRunControlGoalBackendStatsForTestV0(runRef, goalRef, status)
	result.Goal.ArtifactRefs = compactStringsMCPV0(artifactRefs)
	result.Goal.DomainReceiptRefs = compactStringsMCPV0(domainReceiptRefs)
	result.Stats = &orquestacionnucleoapp.DirectorRunStatsV0{
		RunRef: runRef,
		UsageSummary: &orquestacionnucleoapp.DirectorRunUsageStatsV0{
			TotalTokens: tokens,
		},
	}
	return result
}

func containsMCPRunControlDiagnosticForTestV0(
	diagnostics []MCPRunControlDiagnosticV0,
	code string,
) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func (fake *fakeMCPRunControlPortV0) RecordRunCheckpointV0(
	_ context.Context,
	command orquestaruncontrol.RecordRunCheckpointCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.checkpoint = command
	status := orquestaruncontrol.RunControlStatusStopRequestedV0
	forced := fake.stop.Forced
	if fake.stopStatus != "" {
		status = fake.stopStatus
	}
	if fake.cancel.RunRef == command.RunRef && fake.stop.RunRef == "" {
		status = orquestaruncontrol.RunControlStatusCancelRequestedV0
		forced = fake.cancel.Forced
	}
	state := runControlStateForMCPTestV0(command.RunRef, status, forced)
	state.CheckpointRecorded = true
	state.EvidenceRefs = command.EvidenceRefs
	return state, nil
}

func (fake *fakeMCPRunControlPortV0) CompleteRunControlV0(
	_ context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	fake.complete = command
	state := runControlStateForMCPTestV0(command.RunRef, command.TargetStatus, false)
	state.EvidenceRefs = command.EvidenceRefs
	return state, nil
}

func runControlStateForMCPTestV0(
	runRef string,
	status orquestaruncontrol.RunControlStatusV0,
	forced bool,
) orquestaruncontrol.RunControlStateV0 {
	return orquestaruncontrol.RunControlStateV0{
		RunRef:             runRef,
		Status:             status,
		CheckpointRecorded: false,
		Forced:             forced,
		EvidenceRefs:       []string{"evidence-1"},
	}
}
