package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
)

func TestGoalFirstRunControlPortV0ForcedStopBloqueaGoalDetieneBackendYCompletaControlV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-goal-first-forced-stop-001"
	goalRef := "goal-ref-goal-first-forced-stop-001"
	externalRef := "thread-ref-goal-first-forced-stop-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state := goalFirstRunControlRunningStateForTestV0(t, runRef, goalRef, externalRef)
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	runControl := orquestarunmemory.NewRunMemoryStoreV0()
	backend := &fakeGoalBackendControlForRunControlTestV0{
		result: GoalBackendControlResultV0{
			Status:          orquestagoal.GoalStatusBlockedV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalRef,
			GoalStatusSet:   true,
			BackendStopped:  true,
			EvidenceRefs:    []string{"evidence-ref-backend-stop-ok"},
		},
	}
	port := goalFirstRunControlPortV0{
		Inner:          runControl,
		GoalStateStore: goalStates,
		BackendControl: backend,
	}

	controlState, err := port.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "operator",
		Reason:       "test forced stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-operator-forced-stop"},
	})
	if err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}
	if controlState.Status != orquestaruncontrol.RunControlStatusStoppedV0 || !controlState.Forced {
		t.Fatalf("controlState=%+v", controlState)
	}
	if backend.calls != 1 ||
		backend.last.Action != "stop" ||
		backend.last.RunRef != runRef ||
		backend.last.GoalRef != goalRef ||
		backend.last.ExternalGoalRef != externalRef ||
		backend.last.WorkspaceAuthoritySchemaVersion != state.LaunchReceipt.WorkspaceAuthoritySchemaVersion ||
		backend.last.WorkspaceRef != state.LaunchReceipt.WorkspaceRef ||
		backend.last.ProviderRef != state.LaunchReceipt.ProviderRef ||
		backend.last.RuntimeGenerationRef != state.LaunchReceipt.RuntimeGenerationRef ||
		!backend.last.Forced {
		t.Fatalf("backend calls=%d last=%+v", backend.calls, backend.last)
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusBlockedV0 ||
		persisted.LastResult == nil ||
		persisted.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		len(persisted.LastResult.Issues) != 1 ||
		persisted.LastResult.Issues[0].Code != "operator_forced_stop_no_artifacts" ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.NeedsRework ||
		!stringInSetV0(persisted.EvidenceRefs, runControlGoalForcedStopTerminalEvidenceV0) {
		t.Fatalf("persisted=%+v", persisted)
	}

	controlState, err = port.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "operator",
		Reason:       "idempotent forced stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-operator-forced-stop-repeat"},
	})
	if err != nil {
		t.Fatalf("StopRunV0 repeat: %v", err)
	}
	if controlState.Status != orquestaruncontrol.RunControlStatusStoppedV0 || backend.calls != 1 {
		t.Fatalf("repeat controlState=%+v backend_calls=%d", controlState, backend.calls)
	}
	persistedAgain, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 repeat: %v", err)
	}
	if persistedAgain.Status != orquestagoal.GoalStatusBlockedV0 ||
		persistedAgain.LastResult == nil ||
		persistedAgain.LastResult.Status != orquestagoal.GoalStatusBlockedV0 {
		t.Fatalf("persistedAgain=%+v", persistedAgain)
	}
}

func TestCodexStackRunControlAPIV0ForcedStopGoalFirstPropagaBackendYBloqueaTerminalV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-goal-first-forced-stop-api-001"
	goalRef := "goal-ref-goal-first-forced-stop-api-001"
	externalRef := "thread-ref-goal-first-forced-stop-api-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	backend := &fakeGoalBackendControlForRunControlTestV0{
		result: GoalBackendControlResultV0{
			Status:          orquestagoal.GoalStatusBlockedV0,
			GoalRef:         goalRef,
			ExternalGoalRef: externalRef,
			GoalStatusSet:   true,
			BackendStopped:  true,
			EvidenceRefs:    []string{"evidence-ref-backend-stop-api-ok"},
		},
	}
	config := codexStackBaseConfigForTestV0(t, newPendingAckCodexStackRuntimeV0(), nil, nil)
	config.Stores.AppGoalStateStore = goalStates
	config.AppGoalBackendControl = backend
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	if err := goalStates.SaveGoalWorkStateV0(
		ctx,
		goalFirstRunControlRunningStateForTestV0(t, runRef, goalRef, externalRef),
	); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}

	control := postRunControlStackV0(t, stack, orquestamcp.MCPRunControlToolInputV0{
		Action:       "stop",
		RunRef:       runRef,
		RequestedBy:  "operator",
		Reason:       "api forced stop",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-operator-forced-stop-api"},
	})
	if control.Status != string(orquestaruncontrol.RunControlStatusStoppedV0) ||
		!stringInSetV0(control.EvidenceRefs, runControlGoalForcedStopTerminalEvidenceV0) ||
		!stringInSetV0(control.EvidenceRefs, runControlGoalForcedStopCompleteEvidenceV0) {
		t.Fatalf("control=%+v", control)
	}
	if backend.calls != 1 ||
		backend.last.Action != "stop" ||
		backend.last.RunRef != runRef ||
		backend.last.ExternalGoalRef != externalRef {
		t.Fatalf("backend calls=%d last=%+v", backend.calls, backend.last)
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusBlockedV0 ||
		persisted.LastResult == nil ||
		persisted.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		!stringInSetV0(persisted.EvidenceRefs, runControlGoalForcedStopTerminalEvidenceV0) {
		t.Fatalf("persisted=%+v", persisted)
	}
}

func TestCodexStackServerShutdownAPIForcedStopPropagaSoloGoalActivoV0(t *testing.T) {
	ctx := context.Background()
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	backend := &fakeGoalBackendControlForRunControlTestV0{
		result: GoalBackendControlResultV0{
			Status:         orquestagoal.GoalStatusBlockedV0,
			GoalStatusSet:  true,
			BackendStopped: true,
			EvidenceRefs:   []string{"evidence-ref-forced-shutdown-goal-backend-stopped"},
		},
	}
	config := codexStackBaseConfigForTestV0(t, newPendingAckCodexStackRuntimeV0(), nil, nil)
	config.Stores.AppGoalStateStore = goalStates
	config.AppGoalBackendControl = backend
	stack, err := BuildStackV0(config)
	if err != nil {
		t.Fatalf("BuildStackV0: %v", err)
	}
	target := postDirectorAPIV0(t, stack)
	if err := goalStates.SaveGoalWorkStateV0(ctx, goalFirstRunControlRunningStateForTestV0(
		t, target.RunRef, "goal-ref-forced-shutdown-required-test", "thread-ref-forced-shutdown-required-test",
	)); err != nil {
		t.Fatalf("Save target goal state: %v", err)
	}
	foreignRunRef := "run-ref-foreign-turn-must-stay-alive"
	if err := goalStates.SaveGoalWorkStateV0(ctx, goalFirstRunControlRunningStateForTestV0(
		t, foreignRunRef, "goal-ref-foreign-turn", "thread-ref-foreign-turn",
	)); err != nil {
		t.Fatalf("Save foreign goal state: %v", err)
	}

	shutdown := postServerShutdownStackV0(t, stack, orquestamcp.MCPServerShutdownToolInputV0{
		RequestID:      "request-ref-forced-shutdown-active-required-test",
		CorrelationID:  "corr-forced-shutdown-active-required-test",
		Forced:         true,
		RequestedBy:    "orquesta-director",
		Reason:         "forced shutdown during active required test and turn",
		IdempotencyKey: "idem-forced-shutdown-active-required-test",
		MaxTicks:       2,
		MaxExecutions:  2,
	})
	if shutdown.Estado != orquestamcp.MCPServerShutdownEstadoOKV0 ||
		!shutdown.ShutdownReady || shutdown.Status != "ready" {
		t.Fatalf("shutdown=%+v", shutdown)
	}
	if backend.calls != 1 || backend.last.RunRef != target.RunRef ||
		backend.last.GoalRef != "goal-ref-forced-shutdown-required-test" ||
		backend.last.ExternalGoalRef != "thread-ref-forced-shutdown-required-test" ||
		backend.last.Action != "stop" || !backend.last.Forced {
		t.Fatalf("backend calls=%d last=%+v", backend.calls, backend.last)
	}
	foreign, err := goalStates.LoadGoalWorkStateV0(ctx, foreignRunRef)
	if err != nil {
		t.Fatalf("Load foreign goal state: %v", err)
	}
	if foreign.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("foreign goal must remain running: %+v", foreign)
	}
}

func TestGoalFirstRunControlPortV0ForcedStopCompletaControlSiObserverYaBloqueoAltoConsumoV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-goal-first-forced-stop-already-blocked-001"
	goalRef := "goal-ref-goal-first-forced-stop-already-blocked-001"
	externalRef := "thread-ref-goal-first-forced-stop-already-blocked-001"
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	state := goalFirstRunControlRunningStateForTestV0(t, runRef, goalRef, externalRef)
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         goalRef,
		ExternalGoalRef: externalRef,
		Summary:         "goal_active_no_checkpoint_high_consumption",
		EvidenceRefs: []string{
			"evidence-ref-goal-observer-high-consumption-stop-requested",
			"evidence-ref-goal-cooperative-stop-requested-run-control",
		},
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  "goal_active_no_checkpoint_high_consumption",
			Field: "goal_progress",
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:      orquestagoal.GoalStatusBlockedV0,
		NeedsRework: true,
		EvidenceRefs: []string{
			"evidence-ref-goal-observer-high-consumption-stop-requested",
		},
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  "goal_active_no_checkpoint_high_consumption",
			Field: "goal_progress",
		}},
	}
	if err := goalStates.SaveGoalWorkStateV0(ctx, state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	runControl := orquestarunmemory.NewRunMemoryStoreV0()
	if _, err := runControl.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "goal_observer",
		Reason:       "cooperative high consumption stop",
		EvidenceRefs: []string{"evidence-ref-goal-cooperative-stop-requested-run-control"},
	}); err != nil {
		t.Fatalf("StopRunV0 cooperative: %v", err)
	}
	backend := &fakeGoalBackendControlForRunControlTestV0{
		result: GoalBackendControlResultV0{
			Status:         orquestagoal.GoalStatusBlockedV0,
			BackendStopped: true,
		},
	}
	port := goalFirstRunControlPortV0{
		Inner:          runControl,
		GoalStateStore: goalStates,
		BackendControl: backend,
	}

	controlState, err := port.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "operator",
		Reason:       "forced stop tras alto consumo",
		Forced:       true,
		EvidenceRefs: []string{"evidence-ref-operator-forced-stop-after-high-consumption"},
	})
	if err != nil {
		t.Fatalf("StopRunV0 forced: %v", err)
	}
	if controlState.Status != orquestaruncontrol.RunControlStatusStoppedV0 || !controlState.Forced {
		t.Fatalf("controlState=%+v", controlState)
	}
	if backend.calls != 0 {
		t.Fatalf("backend no debe reabrirse para goal ya terminal: calls=%d last=%+v", backend.calls, backend.last)
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusBlockedV0 ||
		persisted.LastResult == nil ||
		persisted.LastClosure == nil ||
		!persisted.LastClosure.NeedsRework ||
		!stringInSetV0(persisted.EvidenceRefs, runControlGoalForcedStopTerminalEvidenceV0) ||
		!stringInSetV0(persisted.LastResult.EvidenceRefs, runControlGoalForcedStopTerminalEvidenceV0) ||
		!stringInSetV0(persisted.LastClosure.EvidenceRefs, runControlGoalForcedStopTerminalEvidenceV0) ||
		!stringInSetV0(controlState.EvidenceRefs, runControlGoalForcedStopCompleteEvidenceV0) {
		t.Fatalf("persisted=%+v controlState=%+v", persisted, controlState)
	}
}

func goalFirstRunControlRunningStateForTestV0(
	t *testing.T,
	runRef string,
	goalRef string,
	externalRef string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	spec := orquestagoal.GoalWorkSpecV0{
		GoalRef:   goalRef,
		RunRef:    runRef,
		Objective: "probar forced stop goal-first",
		WriteSet:  []orquestagoal.GoalWriteScopeV0{{Path: "docs/forced_stop_goal_first.md"}},
	}
	authority := orquestagoal.GoalExecutionAuthorityForProviderV0(spec, "provider-ref-test")
	receipt := orquestagoal.ApplyGoalExecutionAuthorityToReceiptV0(orquestagoal.GoalLaunchReceiptV0{
		GoalRef: goalRef, ExternalGoalRef: externalRef, Status: orquestagoal.GoalStatusRunningV0,
	}, authority)
	state, err := orquestagoal.NewGoalWorkStateFromLaunchV0(orquestagoal.GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs:  []string{"evidence-ref-goal-first-running"},
	})
	if err != nil {
		t.Fatalf("NewGoalWorkStateFromLaunchV0: %v", err)
	}
	return state
}

type fakeGoalBackendControlForRunControlTestV0 struct {
	calls  int
	last   GoalBackendControlRequestV0
	result GoalBackendControlResultV0
	err    error
}

func (fake *fakeGoalBackendControlForRunControlTestV0) ControlGoalBackendV0(
	_ context.Context,
	request GoalBackendControlRequestV0,
) (GoalBackendControlResultV0, error) {
	fake.calls++
	fake.last = request
	return fake.result, fake.err
}
