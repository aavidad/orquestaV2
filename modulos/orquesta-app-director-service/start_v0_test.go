package orquestaappdirectorservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	orquestaappdirectorintake "orquesta/modulos/orquesta-app-director-intake"
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
		!spec.ClosurePolicy.RequireRequiredTests ||
		spec.Budget.MaxRuntimeSeconds != 600 ||
		spec.Budget.MaxReworkGoals != 2 ||
		spec.ReworkPolicy.MaxReworkGoals != 2 {
		t.Fatalf("goal spec inesperado: %+v", spec)
	}
	if !serviceGoalContextRefInSetForTestV0(spec.ContextRefs, "phase_policy", startAppDirectorGoalNewAppPhasePolicyRefV0) ||
		!serviceStringInSetV0(spec.AcceptanceCriteria, startAppDirectorGoalNewAppPhasePolicyCriterionV0) {
		t.Fatalf("goal spec sin politica de fase Nueva App: refs=%+v criteria=%+v", spec.ContextRefs, spec.AcceptanceCriteria)
	}
	if !serviceGoalContextRefInSetForTestV0(spec.ContextRefs, "technical_stack", "technical-language-go") ||
		!serviceGoalRuleRefInSetForTestV0(spec.RuleRefs, "technical_constraint", "technical_constraint:language=go", orquestagoal.GoalRuleEnforcementHardV0) ||
		!serviceGoalArtifactContractInSetForTestV0(spec.ArtifactContracts, "technical_stack_manifest") ||
		!serviceStringsContainPartV0(spec.AcceptanceCriteria, "Stack tecnico solicitado: lenguaje=go") {
		t.Fatalf("goal spec sin contrato tecnico: refs=%+v rules=%+v artifacts=%+v criteria=%+v", spec.ContextRefs, spec.RuleRefs, spec.ArtifactContracts, spec.AcceptanceCriteria)
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

func TestStartAppDirectorV0GoalFirstAplicaPoliticaAutonomaV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	policy := &serviceAutonomousDirectorPolicyForTestV0{
		decision: orquestacionnucleoapp.AutonomousDirectorDecisionV0{
			TeamSize:          4,
			MaxParallelAgents: 3,
			Summary:           "policy-test team=4 parallel=3",
			EvidenceRefs:      []string{"evidence-ref-policy-test"},
		},
	}
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0

	result, err := StartAppDirectorV0(
		context.Background(),
		request,
		StartAppDirectorPortsV0{
			RunStore:                 store,
			EventSink:                sink,
			GoalLauncher:             launcher,
			GoalObserver:             serviceGoalObserverForTestV0{},
			GoalClosureValidator:     orquestagoal.DefaultGoalWorkClosureValidatorV0{},
			GoalStateStore:           goalStates,
			AutonomousDirectorPolicy: policy,
		},
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.DirectorExecutionMode != AppDirectorExecutionModeGoalFirstV0 || launcher.calls != 1 || policy.calls != 1 {
		t.Fatalf("result=%+v launcher_calls=%d policy_calls=%d", result, launcher.calls, policy.calls)
	}
	if len(policy.inputs) != 1 ||
		policy.inputs[0].Run.RunID != result.Run.RunID ||
		policy.inputs[0].Stats.RunRef != result.Run.RunID ||
		policy.inputs[0].Limits.MaxTeamSize != 6 {
		t.Fatalf("policy input=%+v result=%+v", policy.inputs, result)
	}
	spec := launcher.specs[0]
	if spec.Budget.MaxSubgoals != 4 ||
		!serviceGoalContextRefInSetForTestV0(spec.ContextRefs, "autonomous_director_policy", "autonomous_director_policy:v0") ||
		!serviceStringInSetV0(spec.EvidenceRefs, "evidence-ref-policy-test") {
		t.Fatalf("spec sin decision autonoma aplicada: %+v", spec)
	}
}

func TestStartAppDirectorGoalSpecWithAutonomousPolicyV0StatsDistintosDecisionDistinta(t *testing.T) {
	policy := &serviceAutonomousDirectorPolicyForTestV0{
		decide: func(input orquestacionnucleoapp.AutonomousDirectorDecisionInputV0) orquestacionnucleoapp.AutonomousDirectorDecisionV0 {
			teamSize := 2
			if input.Stats.Counts.TasksOpen >= 4 {
				teamSize = 6
			}
			return orquestacionnucleoapp.AutonomousDirectorDecisionV0{
				TeamSize:     teamSize,
				Summary:      "policy-test tasks_open",
				EvidenceRefs: []string{"evidence-ref-policy-stats"},
			}
		},
	}
	baseSpec := orquestagoal.GoalWorkSpecV0{
		Budget:       orquestagoal.GoalBudgetV0{MaxSubgoals: 10},
		EvidenceRefs: []string{"evidence-ref-base"},
	}
	request := StartAppDirectorRequestV0{CorrelationID: "corr-policy-stats"}
	smallRun := orquestacoreworkflow.OrchestrationRunV0{
		RunID:        "run-policy-small",
		CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Tasks:        []string{"task-1"},
	}
	largeRun := smallRun
	largeRun.RunID = "run-policy-large"
	largeRun.Tasks = []string{"task-1", "task-2", "task-3", "task-4", "task-5"}

	smallSpec, err := startAppDirectorGoalSpecWithAutonomousPolicyV0(
		context.Background(),
		request,
		orquestaappdirectorintake.AppDirectorIntakePreparedV0{Run: smallRun},
		StartAppDirectorPortsV0{AutonomousDirectorPolicy: policy},
		baseSpec,
	)
	if err != nil {
		t.Fatalf("small policy: %v", err)
	}
	largeSpec, err := startAppDirectorGoalSpecWithAutonomousPolicyV0(
		context.Background(),
		request,
		orquestaappdirectorintake.AppDirectorIntakePreparedV0{Run: largeRun},
		StartAppDirectorPortsV0{AutonomousDirectorPolicy: policy},
		baseSpec,
	)
	if err != nil {
		t.Fatalf("large policy: %v", err)
	}
	if smallSpec.Budget.MaxSubgoals != 2 || largeSpec.Budget.MaxSubgoals != 6 {
		t.Fatalf("budgets small=%+v large=%+v", smallSpec.Budget, largeSpec.Budget)
	}
	if len(policy.inputs) != 2 ||
		policy.inputs[0].Stats.Counts.TasksOpen == policy.inputs[1].Stats.Counts.TasksOpen {
		t.Fatalf("policy stats no variaron: %+v", policy.inputs)
	}
}

func TestBuildStartAppDirectorGoalWorkSpecV0TransportaConstraintLenguajeV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeGoalFirstV0
	request.AppSpecRequest.PreferenciasTecnicas.Lenguaje = "Go"
	request.AppSpecRequest.PreferenciasTecnicas.Framework = "net/http"

	if _, err := StartAppDirectorV0(
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
	); err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if len(launcher.specs) != 1 {
		t.Fatalf("launcher specs=%d want 1", len(launcher.specs))
	}
	spec := launcher.specs[0]
	if !serviceGoalRuleRefInSetForTestV0(spec.RuleRefs, "technical_constraint", "technical_constraint:language=go", orquestagoal.GoalRuleEnforcementHardV0) ||
		!serviceGoalRuleRefInSetForTestV0(spec.RuleRefs, "technical_constraint", "technical_constraint:framework=net-http", orquestagoal.GoalRuleEnforcementHardV0) {
		t.Fatalf("spec sin constraints tecnicas hard: %+v", spec.RuleRefs)
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

func TestStartAppDirectorV0LegacyLoopPublicaSunsetNoticeV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	request := validStartAppDirectorRequestForTestV0()
	request.DirectorExecutionMode = AppDirectorExecutionModeLegacyDirectorLoopV0

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
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if result.DirectorExecutionMode != AppDirectorExecutionModeLegacyDirectorLoopV0 ||
		result.LegacySunsetNotice != AppDirectorLegacyLoopSunsetNoticeV0 {
		t.Fatalf("result=%+v", result)
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

func TestStartAppDirectorV0GoalFirstLauncherDegradadoPersisteMarkerV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{err: errors.New("goal_launcher_unavailable")}
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
			Dispatchers: []orquestacionnucleoapp.OutboxDispatcherBindingV0{
				serviceCapacityDispatcherForTestV0(store, sink, ledger),
				serviceAgentLauncherDispatcherForTestV0(store, sink, ledger),
			},
		},
	)

	if err == nil || err.Error() != "goal_launcher_unavailable" {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	marker, err := markers.LoadGoalWorkRunMarkerV0(context.Background(), request.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if marker.RunRef != request.RunRef ||
		marker.Status != orquestagoal.GoalStatusBlockedV0 ||
		!serviceStringInSetV0(marker.EvidenceRefs, "evidence-ref-app-director-goal-first-launch-failed-v0") {
		t.Fatalf("marker=%+v", marker)
	}
	continued, err := ContinueAppDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        request.RunRef,
			OccurredAt:    "2026-05-09T22:33:00Z",
			CorrelationID: "corr-service-goal-first-launch-failed-marker",
		},
		StartAppDirectorPortsV0{
			RunStore:                store,
			GoalStateStore:          goalStates,
			GoalFirstRunMarkerStore: markers,
		},
	)
	if err != nil {
		t.Fatalf("ContinueAppDirectorV0: %v", err)
	}
	if continued.Status != ContinueAppDirectorStatusPendingV0 ||
		continued.LoopStatus != orquestacionnucleoapp.ProgressiveLoopStatusWaitExternalV0 ||
		len(continued.OperationalClosureIssues) != 1 ||
		continued.OperationalClosureIssues[0].Code != continueAppDirectorGoalFirstStateMissingCodeV0 {
		t.Fatalf("continued=%+v", continued)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentRequestedV0) ||
		serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStartedV0) {
		t.Fatalf("goal-first degradado no debe caer al loop legacy: %+v", sink.EventsV0())
	}
}

func TestStartAppDirectorV0GoalFirstStateStoreFallaPersisteMarkerConExternalGoalRefV0(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	launcher := &serviceGoalLauncherForTestV0{}
	goalStates := newServiceGoalStateStoreForTestV0()
	goalStates.saveErr = errors.New("goal_state_store_unavailable")
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

	if err == nil || err.Error() != "goal_state_store_unavailable" {
		t.Fatalf("err=%v result=%+v", err, result)
	}
	marker, err := markers.LoadGoalWorkRunMarkerV0(context.Background(), request.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkRunMarkerV0: %v", err)
	}
	if marker.RunRef != request.RunRef ||
		marker.ExternalGoalRef != "thread-ref-service-goal-001" ||
		marker.Status != orquestagoal.GoalStatusBlockedV0 ||
		!serviceStringInSetV0(marker.EvidenceRefs, "evidence-ref-service-goal-launched-001") ||
		!serviceStringInSetV0(marker.EvidenceRefs, "evidence-ref-app-director-goal-first-launch-failed-v0") {
		t.Fatalf("marker=%+v", marker)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentRequestedV0) ||
		serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventAgentStartedV0) {
		t.Fatalf("goal-first con state store degradado no debe caer al loop legacy: %+v", sink.EventsV0())
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
		Summary:         "Entrega Go con API HTTP.",
		ArtifactRefs:    serviceRequiredArtifactRefsForGoalSpecV0(spec),
		ArtifactPaths: []string{
			"generated-apps/agenda/go.mod",
			"generated-apps/agenda/main.go",
		},
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

func TestObserveAppDirectorGoalV0BloqueaRunSiStackTecnicoContradiceAppSpecV0(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		Summary:         "Entrega Python con API HTTP.",
		ArtifactRefs:    serviceRequiredArtifactRefsForGoalSpecV0(spec),
		ArtifactPaths: []string{
			"generated-apps/agenda/pyproject.toml",
			"generated-apps/agenda/run.py",
			"generated-apps/agenda/tests/test_http_api.py",
		},
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
	if result.Closure.Accepted ||
		!result.Closure.NeedsRework ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		len(result.Closure.Issues) == 0 ||
		result.Closure.Issues[0].Code != appDirectorGoalTechnicalLanguageMismatchV0 ||
		result.Closure.Issues[0].Field != "technical.language" {
		t.Fatalf("result=%+v", result)
	}
}

func TestCodexStackV0GoalFirstNoCierraNuevaAppConLenguajeEquivocadoV0(t *testing.T) {
	issues := appDirectorGoalTechnicalClosureIssuesV0(
		orquestagoal.GoalWorkSpecV0{
			ContextRefs: []orquestagoal.GoalContextRefV0{{
				Kind:     "technical_stack",
				Ref:      "technical-language-go",
				Required: true,
			}},
		},
		orquestagoal.GoalWorkResultV0{
			Status: orquestagoal.GoalStatusCompleteV0,
			ArtifactPaths: []string{
				"generated-apps/agenda/pyproject.toml",
				"generated-apps/agenda/src/app.py",
			},
		},
	)
	if len(issues) != 1 ||
		issues[0].Code != appDirectorGoalTechnicalLanguageMismatchV0 ||
		issues[0].Field != "technical.language" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestCodexStackV0GoalFirstCierraNuevaAppSinConstraintDeLenguajeV0(t *testing.T) {
	issues := appDirectorGoalTechnicalClosureIssuesV0(
		orquestagoal.GoalWorkSpecV0{},
		orquestagoal.GoalWorkResultV0{
			Status: orquestagoal.GoalStatusCompleteV0,
			ArtifactPaths: []string{
				"generated-apps/agenda/pyproject.toml",
				"generated-apps/agenda/src/app.py",
			},
		},
	)
	if len(issues) != 0 {
		t.Fatalf("sin constraint no debe bloquear por lenguaje: %+v", issues)
	}
}

func TestObserveAppDirectorGoalV0RecuperaEntregaTardiaTrasTimeoutBloqueadoV0(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	timeoutObserver := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		Summary:         "codex_app_server_goal_active_timeout",
		EvidenceRefs:    []string{"evidence-ref-codex-app-server-goal-active-timeout"},
	}}
	if _, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         timeoutObserver,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	); err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0 timeout: %v", err)
	}
	lateObserver := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:       orquestagoal.GoalWorkResultSchemaV0,
		Status:              orquestagoal.GoalStatusCompleteV0,
		GoalRef:             spec.GoalRef,
		ExternalGoalRef:     started.ExternalGoalRef,
		ArtifactRefs:        serviceRequiredArtifactRefsForGoalSpecV0(spec),
		DomainReceiptRefs:   []string{"domain-receipt-ref-service-late-result"},
		EvidenceRefs:        spec.ClosurePolicy.RequiredEvidenceRefs,
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{},
	}}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         lateObserver,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0 late result: %v", err)
	}
	if result.Status != orquestagoal.GoalStatusCompleteV0 ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		!result.Closure.NeedsRework ||
		len(result.GoalResult.ArtifactRefs) == 0 ||
		len(result.GoalResult.DomainReceiptRefs) == 0 ||
		len(result.Closure.Issues) == 0 ||
		result.Closure.Issues[0].Field != "required_tests" {
		t.Fatalf("result=%+v", result)
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.LastResult == nil ||
		len(state.LastResult.ArtifactRefs) == 0 ||
		len(state.LastResult.DomainReceiptRefs) == 0 ||
		state.LastClosure == nil ||
		state.LastClosure.Issues[0].Field != "required_tests" {
		t.Fatalf("state=%+v", state)
	}
}

func TestObserveAppDirectorGoalV0NoReobservaGoalTerminalPorForcedStopV0(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	evidenceRefs := []string{
		"evidence-ref-operator-forced-stop",
		"evidence-ref-run-control-goal-forced-stop-terminal",
	}
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		Summary:         "forced stop accepted; goal backend stopped and marked blocked for rework",
		EvidenceRefs:    evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  "operator_forced_stop_no_artifacts",
			Field: "goal_backend",
		}},
	}
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: evidenceRefs,
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  "operator_forced_stop_no_artifacts",
			Field: "goal_backend",
		}},
	}
	state.EvidenceRefs = compactStartAppDirectorStringsV0(append(state.EvidenceRefs, evidenceRefs...))
	if err := goalStates.SaveGoalWorkStateV0(context.Background(), state); err != nil {
		t.Fatalf("SaveGoalWorkStateV0: %v", err)
	}
	observerCalls := 0
	runningObserver := serviceGoalObserverForTestV0{
		calls: &observerCalls,
		result: orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         spec.GoalRef,
			ExternalGoalRef: started.ExternalGoalRef,
			EvidenceRefs:    []string{"evidence-ref-backend-still-running"},
		},
	}

	result, err := ObserveAppDirectorGoalV0(
		context.Background(),
		ObserveAppDirectorGoalRequestV0{RunRef: spec.RunRef},
		StartAppDirectorPortsV0{
			RunStore:             store,
			EventSink:            sink,
			GoalStateStore:       goalStates,
			GoalObserver:         runningObserver,
			GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if observerCalls != 0 ||
		result.Status != orquestagoal.GoalStatusBlockedV0 ||
		result.GoalResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 {
		t.Fatalf("observerCalls=%d result=%+v", observerCalls, result)
	}
	persisted, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 persisted: %v", err)
	}
	if persisted.Status != orquestagoal.GoalStatusBlockedV0 ||
		persisted.LastResult == nil ||
		persisted.LastResult.Status != orquestagoal.GoalStatusBlockedV0 ||
		serviceStringInSetV0(persisted.EvidenceRefs, "evidence-ref-backend-still-running") {
		t.Fatalf("persisted=%+v", persisted)
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
		result.Closure.Status != "" ||
		result.Closure.NeedsRework ||
		result.GoalResult.GoalRef != reworkSpec.GoalRef ||
		result.GoalResult.Summary != "" ||
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

func TestObserveAppDirectorGoalV0LanzaReworkGoalPorTimeoutActivoV0(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusBlockedV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		Summary:         "codex_app_server_goal_active_timeout",
		EvidenceRefs:    []string{"evidence-ref-codex-app-server-goal-active-timeout"},
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  "codex_app_server_goal_active_timeout",
			Field: "codex_goal_backend",
		}},
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
		!serviceStringInSetV0(reworkSpec.EvidenceRefs, "evidence-ref-codex-app-server-goal-active-timeout") {
		t.Fatalf("reworkSpec=%+v", reworkSpec)
	}
	if result.Status != orquestagoal.GoalStatusRunningV0 ||
		result.GoalRef != reworkSpec.GoalRef ||
		result.Closure.Status != "" ||
		result.Closure.NeedsRework ||
		result.GoalResult.GoalRef != reworkSpec.GoalRef ||
		result.GoalResult.Summary != "" ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		t.Fatalf("result=%+v", result)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("timeout operativo con rework no debe bloquear run: %+v", sink.EventsV0())
	}
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	if state.GoalRef != reworkSpec.GoalRef || state.LastResult != nil || state.Status != orquestagoal.GoalStatusRunningV0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestObserveAppDirectorGoalV0LanzaReworkGoalSiNuevaAppSoloCheckpointInvalidV0(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	observer := serviceGoalObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		Summary:         "resultado durable inicial; pendiente de implementacion, artefactos y pruebas",
		ArtifactPaths: []string{
			spec.WriteSet[0].Path + "/checkpoint_started.txt",
			spec.WriteSet[0].Path + "/docs/orquesta_goal_result_" + spec.GoalRef + ".json",
		},
		Checklist: orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  serviceRequiredArtifactRefsForGoalSpecV0(spec),
			MissingRefs:   []string{"source_tree", "handoff_report", "technical_stack_manifest", "go_app", "tests"},
			EvidenceRefs:  []string{"evidence-ref-app-director-goal-first-v0"},
			CompletedRefs: []string{},
		},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      spec.RequiredTests[0].TestRef,
			Status:       "pending",
			EvidenceRefs: []string{},
		}},
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
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
		result.Closure.NeedsRework ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 {
		t.Fatalf("result=%+v", result)
	}
	if serviceHasEventTypeV0(sink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunBlockedV0) {
		t.Fatalf("invalid checkpoint con launcher debe relanzar rework, no bloquear run: %+v", sink.EventsV0())
	}
}

func TestObserveAppDirectorGoalV0BloqueaSiReworkGoalAgotaPresupuesto(t *testing.T) {
	store, sink, goalStates, launcher, started := serviceStartGoalFirstForObserveTestV0(t)
	spec := launcher.specs[0]
	state, err := goalStates.LoadGoalWorkStateV0(context.Background(), spec.RunRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	state.GoalRef = spec.GoalRef + "-rework-2"
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
		!serviceStringInSetV0(result.EvidenceRefs, "evidence-ref-app-director-goal-state-repaired-from-marker-v0") {
		t.Fatalf("result=%+v", result)
	}
	if len(result.OperationalClosureIssues) != 1 ||
		result.OperationalClosureIssues[0].Code != continueAppDirectorGoalFirstObserveCodeV0 {
		t.Fatalf("operational_closure_issues=%+v", result.OperationalClosureIssues)
	}
	repaired, err := missingGoalStateStore.LoadGoalWorkStateV0(context.Background(), started.Run.RunID)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 repaired: %v", err)
	}
	if repaired.GoalRef != started.GoalRef ||
		repaired.ExternalGoalRef != "thread-ref-service-goal-001" ||
		repaired.Spec.GoalRef != started.GoalRef ||
		!serviceStringInSetV0(repaired.EvidenceRefs, "evidence-ref-app-director-goal-state-repaired-from-marker-v0") {
		t.Fatalf("repaired=%+v started=%+v", repaired, started)
	}
}

func TestContinueAppDirectorV0GoalFirstMarkerLegacySinStateNoReparaNiEjecutaLoopLegacyV0(t *testing.T) {
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
	if err := markers.SaveGoalWorkRunMarkerV0(context.Background(), AppDirectorGoalFirstRunMarkerV0{
		SchemaVersion:   AppDirectorGoalFirstRunMarkerSchemaV0,
		RunRef:          started.Run.RunID,
		GoalRef:         started.GoalRef,
		ExternalGoalRef: "thread-ref-service-goal-001",
		DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		EvidenceRefs:    []string{"evidence-ref-legacy-marker"},
	}); err != nil {
		t.Fatalf("SaveGoalWorkRunMarkerV0 legacy: %v", err)
	}
	missingGoalStateStore := newServiceGoalStateStoreForTestV0()

	result, err := ContinueAppDirectorV0(
		context.Background(),
		ContinueAppDirectorRequestV0{
			RunRef:        started.Run.RunID,
			OccurredAt:    "2026-05-09T22:33:00Z",
			CorrelationID: "corr-service-goal-first-legacy-marker-missing-state",
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
	if _, err := missingGoalStateStore.LoadGoalWorkStateV0(context.Background(), started.Run.RunID); err == nil {
		t.Fatalf("marker legacy no debe reparar GoalWorkStateV0")
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

func serviceGoalRuleRefInSetForTestV0(
	refs []orquestagoal.GoalRuleRefV0,
	kind string,
	ref string,
	enforcement string,
) bool {
	for _, candidate := range refs {
		if candidate.Kind == kind && candidate.Ref == ref && candidate.Enforcement == enforcement {
			return true
		}
	}
	return false
}

func serviceGoalArtifactContractInSetForTestV0(
	contracts []orquestagoal.GoalArtifactContractV0,
	artifactType string,
) bool {
	for _, candidate := range contracts {
		if candidate.ArtifactType == artifactType {
			return true
		}
	}
	return false
}

func serviceStringsContainPartV0(values []string, part string) bool {
	for _, value := range values {
		if strings.Contains(value, part) {
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

type serviceAutonomousDirectorPolicyForTestV0 struct {
	calls    int
	inputs   []orquestacionnucleoapp.AutonomousDirectorDecisionInputV0
	decision orquestacionnucleoapp.AutonomousDirectorDecisionV0
	decide   func(orquestacionnucleoapp.AutonomousDirectorDecisionInputV0) orquestacionnucleoapp.AutonomousDirectorDecisionV0
	err      error
}

func (policy *serviceAutonomousDirectorPolicyForTestV0) DecideAutonomousDirectorV0(
	_ context.Context,
	input orquestacionnucleoapp.AutonomousDirectorDecisionInputV0,
) (orquestacionnucleoapp.AutonomousDirectorDecisionV0, error) {
	policy.calls++
	policy.inputs = append(policy.inputs, input)
	if policy.err != nil {
		return orquestacionnucleoapp.AutonomousDirectorDecisionV0{}, policy.err
	}
	if policy.decide != nil {
		return policy.decide(input), nil
	}
	return policy.decision, nil
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
	calls  *int
	result orquestagoal.GoalWorkResultV0
}

func (observer serviceGoalObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	_ orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if observer.calls != nil {
		*observer.calls = *observer.calls + 1
	}
	return observer.result, nil
}

type serviceGoalStateStoreForTestV0 struct {
	states  map[string]AppDirectorGoalStateV0
	saveErr error
}

func newServiceGoalStateStoreForTestV0() *serviceGoalStateStoreForTestV0 {
	return &serviceGoalStateStoreForTestV0{states: map[string]AppDirectorGoalStateV0{}}
}

func (store *serviceGoalStateStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state AppDirectorGoalStateV0,
) error {
	if store.saveErr != nil {
		return store.saveErr
	}
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
