package orquestaappdirectorservice

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaoutboxdispatch "orquesta/modulos/orquesta-outbox-dispatch"
)

func TestStartAppDirectorV0StartsDirectorThroughInjectedPorts(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	result, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.DirectorExecutionMode != AppDirectorExecutionModeLegacyDirectorLoopV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("result=%+v", result)
	}
	if result.Run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0 {
		t.Fatalf("phase=%q", result.Run.CurrentPhase)
	}
	if !serviceStringInSetV0(result.StartedAgents, result.DirectorTask.AgentRequestID) {
		t.Fatalf("started=%v task=%+v", result.StartedAgents, result.DirectorTask)
	}
	if result.DirectorTask.TaskRef == "" || len(result.Run.Tasks) != 0 {
		t.Fatalf("director_task no debe materializar microtarea inicial: task=%+v run_tasks=%v", result.DirectorTask, result.Run.Tasks)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventBrainstormRequestedV0) {
		t.Fatalf("eventos sin BrainstormRequested: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0AutonomiaAltaStartsDirectorTeamThroughBatch(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	batchExecutor := &serviceBatchLauncherForTestV0{
		executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T22:34:00Z",
			CorrelationID: "corr-app-director-service-agent-team-001",
			RequestedBy:   "orquesta-app-director-service-test",
		},
	}

	result, err := StartAppDirectorV0(
		context.Background(),
		validHighAutonomyStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
			},
			BatchDispatchers: []orquestacionnucleoapp.OutboxBatchDispatcherBindingV0{{
				TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
				MaxReady:   4,
				Reader:     ledger,
				Claimer:    ledger,
				Executor:   batchExecutor,
				Acker:      ledger,
			}},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.DirectorTasks) != 4 {
		t.Fatalf("director_tasks=%d %+v", len(result.DirectorTasks), result.DirectorTasks)
	}
	for _, task := range result.DirectorTasks {
		if !serviceStringInSetV0(result.StartedAgents, task.AgentRequestID) {
			t.Fatalf("started=%v missing=%s", result.StartedAgents, task.AgentRequestID)
		}
	}
	if batchExecutor.totalBatchSize != len(result.DirectorTasks) {
		t.Fatalf("batch_total=%d tasks=%d", batchExecutor.totalBatchSize, len(result.DirectorTasks))
	}
	if batchExecutor.maxBatchSize > len(result.DirectorTasks) {
		t.Fatalf("batch_max=%d", batchExecutor.maxBatchSize)
	}
}

func TestStartAppDirectorV0GoalFirstLanzaGoalYNoEjecutaLoopLegacy(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			OutboxLedger:         ledger,
			GoalLauncher:         launcher,
			GoalObserver:         serviceGoalObserverForTestV0{},
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:       goalStates,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.DirectorExecutionMode != AppDirectorExecutionModeGoalFirstV0 ||
		result.GoalRef == "" ||
		result.ExternalGoalRef != "thread-ref-service-goal-001" ||
		result.GoalStatus != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("result=%+v", result)
	}
	if len(result.StartedAgents) != 0 || result.LoopStatus != "" {
		t.Fatalf("goal-first no debe arrancar agentes legacy: started=%v loop=%q", result.StartedAgents, result.LoopStatus)
	}
	if launcher.calls != 1 || len(launcher.specs) != 1 {
		t.Fatalf("launcher calls=%d specs=%d", launcher.calls, len(launcher.specs))
	}
	spec := launcher.specs[0]
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		t.Fatalf("goal spec invalido: %+v spec=%+v", issues, spec)
	}
	if spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		spec.RunRef != result.Run.RunID ||
		len(spec.WriteSet) != 1 ||
		spec.WriteSet[0].Path != "generated-apps/agenda" ||
		len(spec.RequiredTests) != 1 ||
		!spec.ClosurePolicy.RequireRequiredTests {
		t.Fatalf("goal spec inesperado: %+v", spec)
	}
	if _, err := store.LoadRunV0(context.Background(), result.Run.RunID); err != nil {
		t.Fatalf("run no persistida: %v", err)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), result.Run.RunID)
	if err != nil {
		t.Fatalf("goal state no persistido: %v", err)
	}
	if state.GoalRef != result.GoalRef ||
		state.ExternalGoalRef != result.ExternalGoalRef ||
		state.Spec.RunRef != result.Run.RunID {
		t.Fatalf("goal state=%+v result=%+v", state, result)
	}
}

func TestStartAppDirectorV0GoalFirstPorDefectoNoExigeOutboxNiDispatchersLegacy(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = ""

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalLauncher:         launcher,
			GoalObserver:         serviceGoalObserverForTestV0{},
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:       goalStates,
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.DirectorExecutionMode != AppDirectorExecutionModeGoalFirstV0 ||
		result.GoalRef == "" ||
		launcher.calls != 1 ||
		len(result.StartedAgents) != 0 ||
		result.LoopStatus != "" {
		t.Fatalf("result=%+v calls=%d", result, launcher.calls)
	}
}

func TestStartAppDirectorV0GoalFirstEstrictoSinBackendNoCaeALoopLegacy(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:     store,
			EventSink:    sink,
			OutboxLedger: ledger,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	if err == nil || err.Error() != "app_director_service_invalido: goal_backend_unavailable" {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentRequestedV0) ||
		serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStartedV0) {
		t.Fatalf("goal-first estricto no debe caer al loop legacy: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0LegacyDirectorLoopExplicitoNoLanzaGoal(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeLegacyDirectorLoopV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			OutboxLedger:         ledger,
			GoalLauncher:         launcher,
			GoalObserver:         serviceGoalObserverForTestV0{},
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:       goalStates,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.Status != StartAppDirectorStatusStartedV0 ||
		result.DirectorExecutionMode != AppDirectorExecutionModeLegacyDirectorLoopV0 ||
		len(result.StartedAgents) != 1 ||
		result.GoalRef != "" {
		t.Fatalf("result=%+v", result)
	}
	if launcher.calls != 0 || len(goalStates.states) != 0 {
		t.Fatalf("legacy explicito no debe lanzar goal: calls=%d states=%+v", launcher.calls, goalStates.states)
	}
}

func TestStartAppDirectorV0GoalFirstLauncherDegradadoNoCaeALoopLegacy(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{err: errors.New("goal_launcher_unavailable")}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			OutboxLedger:         ledger,
			GoalLauncher:         launcher,
			GoalObserver:         serviceGoalObserverForTestV0{},
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:       goalStates,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	if err == nil || err.Error() != "goal_launcher_unavailable" {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	if launcher.calls != 1 || len(launcher.specs) != 1 {
		t.Fatalf("launcher calls=%d specs=%d", launcher.calls, len(launcher.specs))
	}
	if len(goalStates.states) != 0 {
		t.Fatalf("goal state no debe persistirse si el launcher falla: %+v", goalStates.states)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentRequestedV0) ||
		serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStartedV0) {
		t.Fatalf("goal-first degradado no debe caer al loop legacy: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0GoalFirstBundleIncompletoNoCaeALoopLegacy(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:       store,
			EventSink:      sink,
			OutboxLedger:   ledger,
			GoalLauncher:   launcher,
			GoalStateStore: goalStates,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	if err == nil || err.Error() != "app_director_service_invalido: ports.goal_observer" {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	if launcher.calls != 0 || len(goalStates.states) != 0 {
		t.Fatalf("goal incompleto no debe lanzarse: calls=%d states=%+v", launcher.calls, goalStates.states)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentRequestedV0) ||
		serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStartedV0) {
		t.Fatalf("goal incompleto no debe caer al loop legacy: %+v", sink.EventsV0())
	}
}

func TestObserveAppDirectorGoalV0PersisteResultadoCompletoYClosure(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    serviceRequiredArtifactRefsForGoalSpecV0(spec),
		RequiredTestResults: serviceRequiredTestResultsForGoalSpecV0(
			spec,
			"evidence-ref-service-goal-required-test-passed",
		),
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         observer,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Status != orquestagoal.GoalStatusCompleteV0 ||
		result.DirectorExecutionMode != AppDirectorExecutionModeGoalFirstV0 ||
		!result.Closure.Accepted ||
		result.Closure.Status != orquestagoal.GoalStatusAcceptedV0 ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("result=%+v", result)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0) ||
		!serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0) {
		t.Fatalf("eventos sin validacion/cierre goal-first: %+v", sink.EventsV0())
	}
	loaded, err := store.LoadRunV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if loaded.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run status=%q want cerrada", loaded.Status)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadAppDirectorGoalStateV0: %v", err)
	}
	if state.LastResult == nil || state.LastClosure == nil || !state.LastClosure.Accepted {
		t.Fatalf("state=%+v", state)
	}
}

func TestObserveAppDirectorGoalV0BloqueaRunSiFaltanRequiredTestsV0(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    serviceRequiredArtifactRefsForGoalSpecV0(spec),
		EvidenceRefs:    spec.ClosurePolicy.RequiredEvidenceRefs,
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         observer,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		len(result.Closure.Issues) == 0 ||
		result.Closure.Issues[0].Field != "required_tests" {
		t.Fatalf("result=%+v", result)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("eventos sin RunBlocked: %+v", sink.EventsV0())
	}
}

func TestObserveAppDirectorGoalV0BloqueaRunSiClosureNoAcepta(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		EvidenceRefs:    spec.ClosurePolicy.RequiredEvidenceRefs,
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         observer,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Closure.Accepted ||
		result.DirectorExecutionMode != AppDirectorExecutionModeGoalFirstV0 ||
		!result.Closure.NeedsRework ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("result=%+v", result)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("eventos sin RunBlocked: %+v", sink.EventsV0())
	}
	loaded, err := store.LoadRunV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if loaded.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		!serviceStringInSetV0(loaded.Blockers, appDirectorGoalBlockerRefV0(resultStateForBlockerTestV0(t, goalStates, spec.RunRef))) {
		t.Fatalf("run no bloqueada por goal-first: %+v", loaded)
	}
}

func TestObserveAppDirectorGoalV0LanzaReworkGoalSiPolicyYPuertoDisponibles(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		EvidenceRefs:    spec.ClosurePolicy.RequiredEvidenceRefs,
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalReworkLauncher:   launcher,
			GoalObserver:         observer,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if launcher.calls != 2 || len(launcher.specs) != 2 {
		t.Fatalf("launcher calls=%d specs=%d", launcher.calls, len(launcher.specs))
	}
	reworkSpec := launcher.specs[1]
	if reworkSpec.GoalRef != spec.GoalRef+"-rework-1" ||
		reworkSpec.RunRef != spec.RunRef ||
		!serviceStringInSetV0(reworkSpec.EvidenceRefs, "evidence-ref-app-director-goal-rework-v0") ||
		!serviceGoalContextRefInSetForTestV0(reworkSpec.ContextRefs, "goal", spec.GoalRef) {
		t.Fatalf("reworkSpec=%+v", reworkSpec)
	}
	if result.Status != orquestagoal.GoalStatusRunningV0 ||
		result.GoalRef != reworkSpec.GoalRef ||
		!result.Closure.NeedsRework ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		t.Fatalf("result=%+v", result)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("rework con launcher no debe bloquear run: %+v", sink.EventsV0())
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.GoalRef != reworkSpec.GoalRef || state.LastResult != nil || state.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestObserveAppDirectorGoalV0BloqueaSiReworkGoalAgotaPresupuesto(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	state.GoalRef = spec.GoalRef + "-rework-1"
	state.Spec.GoalRef = state.GoalRef
	state.LaunchReceipt.GoalRef = state.GoalRef
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         state.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		EvidenceRefs:    spec.ClosurePolicy.RequiredEvidenceRefs,
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalReworkLauncher:   launcher,
			GoalObserver:         observer,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if launcher.calls != 1 || result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("launcher calls=%d result=%+v", launcher.calls, result)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("eventos sin RunBlocked: %+v", sink.EventsV0())
	}
}

func TestObserveAppDirectorGoalV0BloqueaRunSiGoalTerminaInvalid(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		EvidenceRefs:    []string{"evidence-ref-service-goal-invalid-terminal"},
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         observer,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Status != orquestagoal.GoalStatusInvalidV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		len(result.Closure.Issues) == 0 ||
		result.Closure.Issues[0].Field != "status" {
		t.Fatalf("result=%+v", result)
	}
	if !serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("eventos sin RunBlocked: %+v", sink.EventsV0())
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.Status != orquestagoal.GoalStatusInvalidV0 ||
		state.LastResult == nil ||
		state.LastClosure == nil ||
		!state.LastClosure.NeedsRework {
		t.Fatalf("state=%+v", state)
	}
}

func TestContinueAppDirectorV0GoalFirstContainerNoEjecutaLoopLegacySinPuertosLegacy(t *testing.T) {
	store, _, goalStates, _, started := serviceStartGoalFirstForObserveTestV0(t)

	result, err := ContinueAppDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        started.Run.RunID,
			OccurredAt:    "2026-05-09T22:31:00Z",
			CorrelationID: "corr-service-goal-first-continue",
		},
		StartAppDirectorPortsV0{
			RunStore:       store,
			GoalStateStore: goalStates,
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.Status != ContinueAppDirectorStatusPendingV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		result.Run.RunID != started.Run.RunID ||
		len(result.StartedAgents) != 0 ||
		!serviceStringInSetV0(result.EvidenceRefs, continueAppDirectorGoalFirstObserveEvidenceV0) {
		t.Fatalf("result=%+v", result)
	}
	if len(result.OperationalClosureIssues) != 1 ||
		result.OperationalClosureIssues[0].Code != continueAppDirectorGoalFirstObserveCodeV0 {
		t.Fatalf("operational_closure_issues=%+v", result.OperationalClosureIssues)
	}
}

func TestStartAppDirectorV0GoalFirstPersisteRunMarkerV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	markers := newServiceGoalFirstRunMarkerStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:                store,
			EventSink:               sink,
			OutboxLedger:            ledger,
			GoalLauncher:            launcher,
			GoalObserver:            serviceGoalObserverForTestV0{},
			GoalClosureValidator:    orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:          goalStates,
			GoalFirstRunMarkerStore: markers,
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	marker, err := markers.LoadGoalWorkRunMarkerV0(context.Background(), result.Run.RunID)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if marker.RunRef != result.Run.RunID ||
		marker.GoalRef != result.GoalRef ||
		marker.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		!serviceStringInSetV0(marker.EvidenceRefs, "evidence-ref-app-director-goal-first-run-marker-v0") {
		t.Fatalf("marker=%+v result=%+v", marker, result)
	}
}

func TestContinueAppDirectorV0GoalFirstMarkerSinStateNoEjecutaLoopLegacyV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	markers := newServiceGoalFirstRunMarkerStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	started, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:                store,
			EventSink:               sink,
			OutboxLedger:            ledger,
			GoalLauncher:            launcher,
			GoalObserver:            serviceGoalObserverForTestV0{},
			GoalClosureValidator:    orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:          goalStates,
			GoalFirstRunMarkerStore: markers,
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	missingGoalStateStore := newServiceGoalStateStoreForTestV0()

	result, err := ContinueAppDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        started.Run.RunID,
			OccurredAt:    "2026-05-09T22:33:00Z",
			CorrelationID: "corr-service-goal-first-marker-missing-state",
		},
		StartAppDirectorPortsV0{
			RunStore:                store,
			GoalStateStore:          missingGoalStateStore,
			GoalFirstRunMarkerStore: markers,
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if result.Status != ContinueAppDirectorStatusPendingV0 ||
		result.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		result.Run.RunID != started.Run.RunID ||
		len(result.StartedAgents) != 0 ||
		!serviceStringInSetV0(result.EvidenceRefs, continueAppDirectorGoalFirstStateMissingEvidenceV0) {
		t.Fatalf("result=%+v", result)
	}
	if len(result.OperationalClosureIssues) != 1 ||
		result.OperationalClosureIssues[0].Code != continueAppDirectorGoalFirstStateMissingCodeV0 {
		t.Fatalf("operational_closure_issues=%+v", result.OperationalClosureIssues)
	}
}

func TestBuildContinueAppDirectorLoopRuntimeV0GoalFirstContainerCierraSinLoopLegacy(t *testing.T) {
	store, _, goalStates, _, started := serviceStartGoalFirstForObserveTestV0(t)

	runtime, err := BuildContinueAppDirectorLoopRuntimeV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        started.Run.RunID,
			OccurredAt:    "2026-05-09T22:32:00Z",
			CorrelationID: "corr-service-goal-first-runtime",
		},
		StartAppDirectorPortsV0{
			RunStore:       store,
			GoalStateStore: goalStates,
		},
	)
	if err != nil {
		t.Fatalf("BuildContinueAppDirectorLoopRuntimeV0: %v", err)
	}
	if !runtime.Closed ||
		runtime.ClosedResult.Status != ContinueAppDirectorStatusPendingV0 ||
		runtime.ClosedResult.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		runtime.ClosedResult.Run.RunID != started.Run.RunID ||
		!serviceStringInSetV0(runtime.ClosedResult.EvidenceRefs, continueAppDirectorGoalFirstContainerEvidenceV0) {
		t.Fatalf("runtime=%+v", runtime)
	}
}

func serviceStartGoalFirstForObserveTestV0(t *testing.T) (
	*orquestacionnucleoapp.InMemoryRunStoreV0,
	*orquestacionnucleoapp.InMemoryEventSinkV0,
	*serviceGoalStateStoreForTestV0,
	*serviceGoalLauncherForTestV0,
	StartAppDirectorResultV0,
) {
	t.Helper()
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0
	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			OutboxLedger:         ledger,
			GoalLauncher:         launcher,
			GoalObserver:         serviceGoalObserverForTestV0{},
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:       goalStates,
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(launcher.specs) != 1 {
		t.Fatalf("launcher specs=%d want 1", len(launcher.specs))
	}
	return store, sink, goalStates, launcher, result
}

func serviceRequiredArtifactRefsForGoalSpecV0(spec orquestagoal.GoalWorkSpecV0) []string {
	refs := make([]string, 0, len(spec.ArtifactContracts))
	for _, contract := range spec.ArtifactContracts {
		if contract.Required {
			refs = append(refs, contract.ArtifactRef)
		}
	}
	return refs
}

func serviceRequiredTestResultsForGoalSpecV0(
	spec orquestagoal.GoalWorkSpecV0,
	evidenceRefs ...string,
) []orquestagoal.GoalRequiredTestResultV0 {
	results := make([]orquestagoal.GoalRequiredTestResultV0, 0, len(spec.RequiredTests))
	for _, test := range spec.RequiredTests {
		results = append(results, orquestagoal.GoalRequiredTestResultV0{
			TestRef:      test.TestRef,
			Status:       "passed",
			EvidenceRefs: append([]string(nil), evidenceRefs...),
		})
	}
	return results
}

func resultStateForBlockerTestV0(
	t *testing.T,
	store *serviceGoalStateStoreForTestV0,
	runRef string,
) AppDirectorGoalStateV0 {
	t.Helper()
	state, err := store.LoadGoalWorkStateV0(context.Background(), runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	return state
}

func serviceGoalContextRefInSetForTestV0(
	refs []orquestagoal.GoalContextRefV0,
	kind string,
	ref string,
) bool {
	for _, candidate := range refs {
		if candidate.Kind == kind && candidate.Ref == ref {
			return true
		}
	}
	return false
}

func TestStartAppDirectorV0ReturnsFactoryValidationIssues(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.AppSpecRequest.Locale = "locale invalido"

	result, err := StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err != nil {
		t.Fatalf("request invalida no debe ser error de transporte: %v", err)
	}
	if result.Status != StartAppDirectorStatusInvalidV0 || len(result.ValidationIssues) == 0 {
		t.Fatalf("result=%+v", result)
	}
	if result.Run.RunID != "" || len(result.StartedAgents) != 0 {
		t.Fatalf("request invalida no debe crear run: %+v", result)
	}
}

func TestStartAppDirectorV0RejectsOperationalDirectorPlanInicialSinContratoFuncional(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.OperationalDirectorPlan = serviceOperationalDirectorPlanForContinueTestV0(t, request.RunRef)

	_, err := StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{})
	if err == nil || err.Error() != "app_director_service_invalido: operational_director_function_contract_refs" {
		t.Fatalf("err=%v", err)
	}
}

func TestStartAppDirectorV0RequiresInjectedPorts(t *testing.T) {
	_, err := StartAppDirectorV0(
		context.Background(),
		validStartAppDirectorRequestForTestV0(),
		StartAppDirectorPortsV0{},
	)
	if err == nil {
		t.Fatalf("esperaba error por puertos ausentes")
	}
}

func TestStartAppDirectorV0RejectsDirectorExecutionModeInvalido(t *testing.T) {
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = "director-antiguo"

	_, err := StartAppDirectorV0(context.Background(), request, StartAppDirectorPortsV0{})

	if err == nil || err.Error() != "app_director_service_invalido: director_execution_mode" {
		t.Fatalf("err=%v", err)
	}
}

type serviceGoalLauncherForTestV0 struct {
	calls int
	specs []orquestagoal.GoalWorkSpecV0
	err   error
}

func (launcher *serviceGoalLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.calls++
	launcher.specs = append(launcher.specs, spec)
	if launcher.err != nil {
		return orquestagoal.GoalLaunchReceiptV0{}, launcher.err
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-service-goal-001",
		EvidenceRefs:    []string{"evidence-ref-service-goal-launched-001"},
	}, nil
}

type serviceGoalObserverForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

func (observer serviceGoalObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	return observer.result, nil
}

type serviceGoalStateStoreForTestV0 struct {
	states map[string]AppDirectorGoalStateV0
}

func newServiceGoalStateStoreForTestV0() *serviceGoalStateStoreForTestV0 {
	return &serviceGoalStateStoreForTestV0{states: map[string]AppDirectorGoalStateV0{}}
}

func (store *serviceGoalStateStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state AppDirectorGoalStateV0,
) error {
	normalized, err := NewAppDirectorGoalStateV0(state)
	if err != nil {
		return err
	}
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *serviceGoalStateStoreForTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (AppDirectorGoalStateV0, error) {
	state, ok := store.states[runRef]
	if !ok {
		return AppDirectorGoalStateV0{}, AppDirectorServiceIssueV0{Field: "goal_state"}
	}
	return state, nil
}

type serviceGoalFirstRunMarkerStoreForTestV0 struct {
	markers map[string]AppDirectorGoalFirstRunMarkerV0
}

func newServiceGoalFirstRunMarkerStoreForTestV0() *serviceGoalFirstRunMarkerStoreForTestV0 {
	return &serviceGoalFirstRunMarkerStoreForTestV0{markers: map[string]AppDirectorGoalFirstRunMarkerV0{}}
}

func (store *serviceGoalFirstRunMarkerStoreForTestV0) SaveGoalWorkRunMarkerV0(
	_ context.Context,
	marker AppDirectorGoalFirstRunMarkerV0,
) error {
	normalized, err := NewAppDirectorGoalFirstRunMarkerV0(marker)
	if err != nil {
		return err
	}
	store.markers[normalized.RunRef] = normalized
	return nil
}

func (store *serviceGoalFirstRunMarkerStoreForTestV0) LoadGoalWorkRunMarkerV0(
	_ context.Context,
	runRef string,
) (AppDirectorGoalFirstRunMarkerV0, error) {
	marker, ok := store.markers[runRef]
	if !ok {
		return AppDirectorGoalFirstRunMarkerV0{}, AppDirectorServiceIssueV0{Field: "goal_first_run_marker"}
	}
	return marker, nil
}

func validStartAppDirectorRequestForTestV0() StartAppDirectorRequestV0 {
	observability := true
	return StartAppDirectorRequestV0{
		RunRef:                "run-app-director-service-001",
		ProjectRef:            "project-app-director-service-001",
		OccurredAt:            "2026-05-09T22:30:00Z",
		CorrelationID:         "corr-app-director-service-001",
		RequestedBy:           "orquesta-app-director-service-test",
		DirectorExecutionMode: AppDirectorExecutionModeLegacyDirectorLoopV0,
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			RequestID:     "request-ref-app-director-service-001",
			Source:        "orquesta-web",
			Locale:        "es-ES",
			Nombre:        "Agenda",
			Objetivo:      "Gestionar contactos y citas desde una API y una web.",
			TipoApp:       "mixed",
			PreferenciasTecnicas: orquestafactory.PreferenciasTecnicasV0{
				Lenguaje:     "go",
				Arquitectura: "hexagonal",
			},
			Calidad: orquestafactory.CalidadRequestV0{
				Pruebas:        "media",
				Accesibilidad:  "basica",
				Observabilidad: &observability,
			},
		},
	}
}

func validHighAutonomyStartAppDirectorRequestForTestV0() StartAppDirectorRequestV0 {
	request := validStartAppDirectorRequestForTestV0()
	request.RunRef = "run-app-director-service-team-001"
	request.ProjectRef = "project-app-director-service-team-001"
	request.CorrelationID = "corr-app-director-service-team-001"
	request.AppSpecRequest.RequestID = "request-ref-app-director-service-team-001"
	request.AppSpecRequest.Datos = orquestafactory.DatosRequestV0{
		DBRequired:         true,
		NecesidadFuncional: "Guardar contactos y citas mediante un puerto de persistencia.",
	}
	request.AppSpecRequest.Calidad.Pruebas = "alta"
	request.AppSpecRequest.Agentes = orquestafactory.AgentesRequestV0{Autonomia: "alta"}
	return request
}

func serviceCapacityDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetCapacityV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.CapacityDecisionExecutorV0{
			RunStore:        store,
			EventSink:       sink,
			Tier:            orquestacoreworkflow.OrchestrationCapacityHighV0,
			ReasoningEffort: orquestacoreworkflow.OrchestrationCapacityHighV0,
			OccurredAt:      "2026-05-09T22:31:00Z",
			CorrelationID:   "corr-app-director-service-capacity-001",
			RequestedBy:     "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

func serviceAgentLauncherDispatcherForTestV0(
	store *orquestacionnucleoapp.InMemoryRunStoreV0,
	sink *orquestacionnucleoapp.InMemoryEventSinkV0,
	ledger *orquestacionnucleoapp.InMemoryOutboxLedgerV0,
) orquestacionnucleoapp.OutboxDispatcherBindingV0 {
	return orquestacionnucleoapp.OutboxDispatcherBindingV0{
		TargetPort: orquestacoreworkflow.OutboxTargetAgentLauncherV0,
		Reader:     ledger,
		Claimer:    ledger,
		Executor: orquestacionnucleoapp.AgentLauncherExecutorV0{
			RunStore:      store,
			EventSink:     sink,
			Launcher:      orquestacionnucleoapp.NewFakeLifecycleAgentLauncherV0(),
			OccurredAt:    "2026-05-09T22:32:00Z",
			CorrelationID: "corr-app-director-service-agent-001",
			RequestedBy:   "orquesta-app-director-service-test",
		},
		Acker: ledger,
	}
}

type serviceBatchLauncherForTestV0 struct {
	executor       orquestacionnucleoapp.AgentLauncherExecutorV0
	totalBatchSize int
	maxBatchSize   int
}

func (executor *serviceBatchLauncherForTestV0) ExecuteOutboxDispatchBatchV0(
	ctx context.Context,
	intents []orquestaoutboxdispatch.DispatchIntentV0,
) ([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, error) {
	executor.totalBatchSize += len(intents)
	if len(intents) > executor.maxBatchSize {
		executor.maxBatchSize = len(intents)
	}
	acks := make([]orquestaoutboxdispatch.OutboxDispatchAckObservationV0, 0, len(intents))
	for _, intent := range intents {
		execution, err := executor.executor.ExecuteOutboxDispatchV0(intent)
		if err != nil {
			return nil, err
		}
		acks = append(acks, orquestaoutboxdispatch.OutboxDispatchAckObservationV0{
			MessageID:    intent.MessageID,
			RunID:        intent.RunID,
			TargetPort:   intent.TargetPort,
			Status:       orquestaoutboxdispatch.OutboxDispatchAckObservationSuccessV0,
			DispatchRef:  execution.DispatchRef,
			EvidenceRefs: execution.EvidenceRefs,
		})
	}
	return acks, nil
}

func serviceStringInSetV0(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func serviceHasEventTypeV0(
	events []orquestacoreworkflow.OrchestrationEventV0,
	eventType string,
) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
