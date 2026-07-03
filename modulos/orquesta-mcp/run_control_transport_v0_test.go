package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestMCPTransportV0RunControlQuedaOptInSinPuerto(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register transport: %v", err)
	}
	output, err := transport.CallToolV0(context.Background(), MCPRunControlToolNameV0, MCPRunControlToolInputV0{})
	if err != nil {
		t.Fatalf("call run control unbound: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode unbound: %v", err)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 ||
		result.Tool != MCPRunControlToolNameV0 {
		t.Fatalf("run control debe ser opt-in: %+v", result)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 300)
}

func TestMCPTransportV0RunControlInvocaExecutor(t *testing.T) {
	port := &fakeMCPRunControlPortV0{}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunControl: NewMCPRunControlToolExecutorV0(port),
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRunControlToolNameV0,
		MCPRunControlToolInputV0{
			RequestID: "request-ref-run-control-transport-001",
			Action:    "pause",
			RunRef:    "run-ref-transport-001",
		},
	)
	if err != nil {
		t.Fatalf("call run control: %v", err)
	}
	var result MCPRunControlToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != "paused" ||
		port.pause.RunRef != "run-ref-transport-001" {
		t.Fatalf("result=%+v port=%+v", result, port)
	}
}

func TestMCPTransportV0RunControlCancelExternalCleanupSinForce(t *testing.T) {
	runRef := "run-ref-control-transport-external-cleanup-cancel-001"
	goalRef := "goal-ref-control-transport-external-cleanup-cancel-001"
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
			Objective:    "Cancelar por transporte MCP tras cleanup externo gobernado.",
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
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunControl: MCPRunControlToolExecutorV0{
			Port:             port,
			GoalBackendState: goalBackend,
			GoalStateStore:   goalStates,
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRunControlToolNameV0,
		MCPRunControlToolInputV0{
			RequestID:    "request-ref-run-control-transport-external-cleanup-cancel-001",
			Action:       "cancel",
			RunRef:       runRef,
			RequestedBy:  "orquesta-director",
			Forced:       false,
			EvidenceRefs: []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
		},
	)
	if err != nil {
		t.Fatalf("call run control: %v", err)
	}
	var result MCPRunControlToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "cancel" ||
		result.Status != string(orquestaruncontrol.RunControlStatusCanceledV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		port.complete.TargetStatus != orquestaruncontrol.RunControlStatusCanceledV0 ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-external-cleanup-reconciled") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v complete=%+v", result, port.complete)
	}
	if strings.Contains(port.complete.Reason, "forced") ||
		!strings.Contains(port.complete.Reason, "external cleanup") ||
		strings.Contains(port.complete.IdempotencyKey, "forced") {
		t.Fatalf("complete transporte cancel degrada cleanup externo a forced: %+v", port.complete)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 1400)
}

func TestMCPTransportV0RunControlStopExternalCleanupSinForce(t *testing.T) {
	runRef := "run-ref-control-transport-external-cleanup-stop-001"
	goalRef := "goal-ref-control-transport-external-cleanup-stop-001"
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
			Objective:    "Parar por transporte MCP tras cleanup externo gobernado.",
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
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunControl: MCPRunControlToolExecutorV0{
			Port:             port,
			GoalBackendState: goalBackend,
			GoalStateStore:   goalStates,
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRunControlToolNameV0,
		MCPRunControlToolInputV0{
			RequestID:    "request-ref-run-control-transport-external-cleanup-stop-001",
			Action:       "stop",
			RunRef:       runRef,
			RequestedBy:  "orquesta-director",
			Forced:       false,
			EvidenceRefs: []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
		},
	)
	if err != nil {
		t.Fatalf("call run control: %v", err)
	}
	var result MCPRunControlToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "stop" ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		port.complete.TargetStatus != orquestaruncontrol.RunControlStatusStoppedV0 ||
		!containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(result.EvidenceRefs, "evidence-ref-run-control-goal-external-cleanup-reconciled") ||
		!containsStringMCPTestV0(result.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v complete=%+v", result, port.complete)
	}
	if strings.Contains(port.complete.Reason, "forced") ||
		!strings.Contains(port.complete.Reason, "external cleanup") ||
		strings.Contains(port.complete.IdempotencyKey, "forced") {
		t.Fatalf("complete transporte stop degrada cleanup externo a forced: %+v", port.complete)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 1400)
}

func TestMCPTransportV0RunControlResumeNoReconciliaExternalCleanupSinForce(t *testing.T) {
	runRef := "run-ref-control-transport-external-cleanup-resume-001"
	goalRef := "goal-ref-control-transport-external-cleanup-resume-001"
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
			Objective:    "Reanudar por transporte MCP sin reconciliar cleanup externo.",
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
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusPausedV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunControl: MCPRunControlToolExecutorV0{
			Port:             port,
			GoalBackendState: goalBackend,
			GoalStateStore:   goalStates,
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRunControlToolNameV0,
		MCPRunControlToolInputV0{
			RequestID:    "request-ref-run-control-transport-external-cleanup-resume-001",
			Action:       "resume",
			RunRef:       runRef,
			RequestedBy:  "orquesta-director",
			Reason:       "reanudar sin reconciliar cleanup externo por transporte",
			Forced:       false,
			EvidenceRefs: []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
		},
	)
	if err != nil {
		t.Fatalf("call run control: %v", err)
	}
	var result MCPRunControlToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "resume" ||
		result.Status != string(orquestaruncontrol.RunControlStatusRunningV0) ||
		result.RecommendedAction != "" ||
		port.complete.RunRef != "" ||
		containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(port.resume.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v resume=%+v complete=%+v", result, port.resume, port.complete)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusRunningV0 ||
		state.LastResult != nil ||
		state.LastClosure != nil {
		t.Fatalf("state mutado por resume transporte con evidencia cleanup externo: %+v", state)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 1400)
}

func TestMCPTransportV0RunControlPauseNoReconciliaExternalCleanupSinForce(t *testing.T) {
	runRef := "run-ref-control-transport-external-cleanup-pause-001"
	goalRef := "goal-ref-control-transport-external-cleanup-pause-001"
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
			Objective:    "Pausar por transporte MCP sin reconciliar cleanup externo.",
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
		readState: runControlStateForMCPTestV0(runRef, orquestaruncontrol.RunControlStatusRunningV0, false),
	}
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
			{Estado: MCPDirectorStatsEstadoOKV0, RunRef: runRef},
		},
	}
	transport := newFakeMCPTransportV0()
	err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		RunControl: MCPRunControlToolExecutorV0{
			Port:             port,
			GoalBackendState: goalBackend,
			GoalStateStore:   goalStates,
		},
	})
	if err != nil {
		t.Fatalf("register transport: %v", err)
	}

	output, err := transport.CallToolV0(
		context.Background(),
		MCPRunControlToolNameV0,
		MCPRunControlToolInputV0{
			RequestID:    "request-ref-run-control-transport-external-cleanup-pause-001",
			Action:       "pause",
			RunRef:       runRef,
			RequestedBy:  "orquesta-director",
			Reason:       "pausar sin reconciliar cleanup externo por transporte",
			Forced:       false,
			EvidenceRefs: []string{mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0},
		},
	)
	if err != nil {
		t.Fatalf("call run control: %v", err)
	}
	var result MCPRunControlToolResultV0
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Action != "pause" ||
		result.Status != string(orquestaruncontrol.RunControlStatusPausedV0) ||
		result.RecommendedAction != "" ||
		port.complete.RunRef != "" ||
		containsMCPRunControlDiagnosticForTestV0(result.Diagnostics, "goal_state_terminal_reconciled_after_external_cleanup") ||
		!containsStringMCPTestV0(port.pause.EvidenceRefs, mcpAutoprogrammingEvidenceGoalBackendMissingAfterExternalCleanupV0) {
		t.Fatalf("result=%+v pause=%+v complete=%+v", result, port.pause, port.complete)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusRunningV0 ||
		state.LastResult != nil ||
		state.LastClosure != nil {
		t.Fatalf("state mutado por pause transporte con evidencia cleanup externo: %+v", state)
	}
	assertTransportPayloadSaneadoMCPTestV0(t, json.RawMessage(output), 1400)
}
