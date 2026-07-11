package orquestamcp

import (
	"context"
	"errors"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

type fakeMCPRunControlBackendStopEscalatorV0 struct {
	requests []MCPRunControlBackendStopEscalationRequestV0
	result   MCPRunControlBackendStopEscalationResultV0
	err      error
}

type failingMCPGoalStateStoreV0 struct {
	delegate  *mcpGoalStateStoreForTestV0
	failSave  bool
	saveCalls int
}

func (store *failingMCPGoalStateStoreV0) SaveGoalWorkStateV0(
	ctx context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	store.saveCalls++
	if store.failSave {
		return errors.New("goal state save failed")
	}
	return store.delegate.SaveGoalWorkStateV0(ctx, state)
}

func (store *failingMCPGoalStateStoreV0) LoadGoalWorkStateV0(
	ctx context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	return store.delegate.LoadGoalWorkStateV0(ctx, runRef)
}

type failingMCPRunControlCompletePortV0 struct {
	*fakeMCPRunControlPortV0
	failComplete  bool
	completeCalls int
}

func (port *failingMCPRunControlCompletePortV0) CompleteRunControlV0(
	ctx context.Context,
	command orquestaruncontrol.CompleteRunControlCommandV0,
) (orquestaruncontrol.RunControlStateV0, error) {
	port.completeCalls++
	if port.failComplete {
		return orquestaruncontrol.RunControlStateV0{}, errors.New("run control complete failed")
	}
	return port.fakeMCPRunControlPortV0.CompleteRunControlV0(ctx, command)
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

func TestMCPRunControlBackendStopEscaladorPrimerControlForzadoCierraGoalYRunControlSinReobserveV0(t *testing.T) {
	runRef := "run-ref-control-escalation-first-terminal-001"
	goalRef := "goal-ref-" + runRef
	escalationEvidenceRef := "evidence-ref-backend-stop-real-first-control-001"
	goalStates := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-" + runRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Cerrar el primer control forzado con evidencia del escalador.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-mcp"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + runRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	staleRunning := mcpRunControlGoalBackendStatsForTestV0(runRef, goalRef, "running")
	goalBackend := &fakeMCPRunControlGoalBackendStateV0{
		results: []MCPDirectorStatsToolResultV0{staleRunning, staleRunning},
	}
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		result: MCPRunControlBackendStopEscalationResultV0{
			Stopped:      true,
			EvidenceRefs: []string{escalationEvidenceRef},
		},
	}
	port := &fakeMCPRunControlPortV0{}
	executor := MCPRunControlToolExecutorV0{
		Port:                 port,
		GoalBackendState:     goalBackend,
		GoalStateStore:       goalStates,
		BackendStopEscalator: escalator,
	}

	result, err := executor.Execute(context.Background(), MCPRunControlToolInputV0{
		RequestID:   "request-ref-control-escalation-first-terminal-001",
		Action:      "stop",
		RunRef:      runRef,
		RequestedBy: "orquesta-director",
		Forced:      true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if result.Estado != MCPRunControlEstadoOKV0 ||
		result.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.FinalStatus != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		result.RecommendedAction != "replan_narrow_context" ||
		port.complete.TargetStatus != orquestaruncontrol.RunControlStatusStoppedV0 ||
		state.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.LastResult == nil ||
		len(state.LastResult.Issues) != 1 ||
		state.LastResult.Issues[0].Code != mcpRunControlReasonBackendStopEscalatedV0 ||
		state.LastClosure == nil ||
		!state.LastClosure.NeedsRework ||
		!containsStringMCPTestV0(state.EvidenceRefs, escalationEvidenceRef) ||
		!containsStringMCPTestV0(port.complete.EvidenceRefs, escalationEvidenceRef) {
		t.Fatalf("primer control no cierra goal/run-control: result=%+v state=%+v complete=%+v", result, state, port.complete)
	}
	if len(goalBackend.inputs) != 2 || len(escalator.requests) != 1 {
		t.Fatalf("la primera llamada requirio reobserve/segundo control: observes=%d escalations=%d", len(goalBackend.inputs), len(escalator.requests))
	}
}

func TestMCPRunControlBackendStopEscaladorFalloSaveNoPublicaStoppedYReplayCompletaV0(t *testing.T) {
	runRef := "run-ref-control-escalation-save-replay-001"
	goalRef := "goal-ref-" + runRef
	delegate := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	seedMCPRunControlGoalStateForTestV0(t, delegate, runRef, goalRef)
	goalStates := &failingMCPGoalStateStoreV0{delegate: delegate, failSave: true}
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		result: MCPRunControlBackendStopEscalationResultV0{
			Stopped:      true,
			EvidenceRefs: []string{"evidence-ref-backend-stop-save-replay-001"},
		},
	}
	port := &failingMCPRunControlCompletePortV0{fakeMCPRunControlPortV0: &fakeMCPRunControlPortV0{}}
	executor := MCPRunControlToolExecutorV0{
		Port:                 port,
		GoalBackendState:     mcpRunControlEscalationActiveBackendForTestV0(runRef),
		GoalStateStore:       goalStates,
		BackendStopEscalator: escalator,
	}
	input := MCPRunControlToolInputV0{
		RequestID:      "request-ref-control-escalation-save-replay-001",
		Action:         "stop",
		RunRef:         runRef,
		RequestedBy:    "orquesta-director",
		Forced:         true,
		IdempotencyKey: "idem-control-escalation-save-replay-001",
	}

	failed, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed save: %v", err)
	}
	state, _ := delegate.LoadGoalWorkStateV0(context.Background(), runRef)
	if failed.Estado != MCPRunControlEstadoErrorV0 ||
		failed.Status == string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		failed.FinalStatus == string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		failed.RecommendedAction != "retry_same_run_control_order" ||
		!containsMCPRunControlDiagnosticForTestV0(failed.Diagnostics, "goal_state_save_failed") ||
		state.Status != orquestagoal.GoalStatusRunningV0 ||
		port.completeCalls != 0 {
		t.Fatalf("fallo SaveGoalWorkState publico terminal: result=%+v state=%+v complete_calls=%d", failed, state, port.completeCalls)
	}

	goalStates.failSave = false
	replayed, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute replay: %v", err)
	}
	state, _ = delegate.LoadGoalWorkStateV0(context.Background(), runRef)
	if replayed.Estado != MCPRunControlEstadoOKV0 ||
		replayed.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		state.Status != orquestagoal.GoalStatusBlockedV0 ||
		port.completeCalls != 1 ||
		goalStates.saveCalls != 2 {
		t.Fatalf("replay tras fallo save no completo: result=%+v state=%+v saves=%d completes=%d", replayed, state, goalStates.saveCalls, port.completeCalls)
	}
}

func TestMCPRunControlBackendStopEscaladorFalloCompleteNoPublicaStoppedYReplayNoReescalaV0(t *testing.T) {
	runRef := "run-ref-control-escalation-complete-replay-001"
	goalRef := "goal-ref-" + runRef
	delegate := &mcpGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
	seedMCPRunControlGoalStateForTestV0(t, delegate, runRef, goalRef)
	goalStates := &failingMCPGoalStateStoreV0{delegate: delegate}
	escalator := &fakeMCPRunControlBackendStopEscalatorV0{
		result: MCPRunControlBackendStopEscalationResultV0{
			Stopped:      true,
			EvidenceRefs: []string{"evidence-ref-backend-stop-complete-replay-001"},
		},
	}
	port := &failingMCPRunControlCompletePortV0{
		fakeMCPRunControlPortV0: &fakeMCPRunControlPortV0{},
		failComplete:            true,
	}
	executor := MCPRunControlToolExecutorV0{
		Port:                 port,
		GoalBackendState:     mcpRunControlEscalationActiveBackendForTestV0(runRef),
		GoalStateStore:       goalStates,
		BackendStopEscalator: escalator,
	}
	input := MCPRunControlToolInputV0{
		RequestID:      "request-ref-control-escalation-complete-replay-001",
		Action:         "stop",
		RunRef:         runRef,
		RequestedBy:    "orquesta-director",
		Forced:         true,
		IdempotencyKey: "idem-control-escalation-complete-replay-001",
	}

	failed, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed complete: %v", err)
	}
	state, _ := delegate.LoadGoalWorkStateV0(context.Background(), runRef)
	if failed.Estado != MCPRunControlEstadoErrorV0 ||
		failed.Status == string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		failed.FinalStatus == string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		!containsMCPRunControlDiagnosticForTestV0(failed.Diagnostics, "run_control_complete_failed") ||
		state.Status != orquestagoal.GoalStatusBlockedV0 ||
		port.completeCalls != 1 ||
		len(escalator.requests) != 1 {
		t.Fatalf("fallo CompleteRunControl publico terminal o perdio parcial: result=%+v state=%+v completes=%d escalations=%d", failed, state, port.completeCalls, len(escalator.requests))
	}

	port.failComplete = false
	executor.BackendStopEscalator = nil
	replayed, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute replay: %v", err)
	}
	if replayed.Estado != MCPRunControlEstadoOKV0 ||
		replayed.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		!containsMCPRunControlDiagnosticForTestV0(replayed.Diagnostics, "goal_state_terminal_reconcile_replayed") ||
		port.completeCalls != 2 ||
		goalStates.saveCalls != 1 ||
		len(escalator.requests) != 1 {
		t.Fatalf("replay tras fallo complete no fue idempotente: result=%+v saves=%d completes=%d escalations=%d", replayed, goalStates.saveCalls, port.completeCalls, len(escalator.requests))
	}
}

func seedMCPRunControlGoalStateForTestV0(
	t *testing.T,
	store *mcpGoalStateStoreForTestV0,
	runRef string,
	goalRef string,
) {
	t.Helper()
	if err := store.SaveGoalWorkStateV0(context.Background(), orquestagoal.GoalWorkStateV0{
		RunRef:          runRef,
		GoalRef:         goalRef,
		ExternalGoalRef: "thread-ref-" + runRef,
		Status:          orquestagoal.GoalStatusRunningV0,
		Spec: orquestagoal.GoalWorkSpecV0{
			RunRef:       runRef,
			GoalRef:      goalRef,
			Objective:    "Probar replay idempotente del reconcile de run control.",
			DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:     []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-mcp"}},
		},
		LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
			GoalRef:         goalRef,
			ExternalGoalRef: "thread-ref-" + runRef,
			Status:          orquestagoal.GoalStatusRunningV0,
		},
	}); err != nil {
		t.Fatalf("SaveGoalWorkStateV0 seed: %v", err)
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
