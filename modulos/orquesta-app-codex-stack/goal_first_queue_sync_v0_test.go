package orquestaappcodexstack

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunmemory "orquesta/modulos/orquesta-run-memory"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestObserveAppDirectorGoalV0SincronizaColaClosedConCandidatoPrevio(t *testing.T) {
	ctx := context.Background()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	claim := orquestarunqueue.WorksetClaimV0{
		SchemaVersion: orquestarunqueue.WorksetClaimSchemaVersionV0,
		ClaimRef:      "claim-ref-goal-first-queue-001",
		RunRef:        runRef,
		TaskRef:       "task-ref-goal-first-queue-001",
		WriteSet:      []orquestarunqueue.ScopeRefV0{{Ref: "generated-apps/agenda"}},
		EvidenceRefs:  []string{"evidence-ref-workset-goal-first-001"},
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:           runRef,
		QueueRef:         "goal-first-test",
		AppRef:           "app-ref-previo-goal-first",
		Status:           orquestarunqueue.RunStatusRunningV0,
		PriorityScore:    91,
		UpdatedAt:        time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC),
		FairnessGroupRef: "fairness-goal-first",
		AttemptGroup: orquestarunqueue.RunQueueAttemptGroupV0{
			GroupRef:     "attempt-group-goal-first",
			ConsumerRef:  "consumer-goal-first",
			ObjectiveRef: "objective-goal-first",
			WorkItemRef:  "workitem-goal-first",
			WriteSetRefs: []string{"generated-apps/agenda"},
		},
		ParentRunRef:     "parent-run-goal-first",
		SupersedesRunRef: "superseded-run-goal-first",
		RescueReason:     "goal-first-test-preserve",
		EvidenceRefs:     []string{"evidence-ref-previa-goal-first"},
		WorksetClaims:    []orquestarunqueue.WorksetClaimV0{claim},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}
	observer.result = orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-first-queue-required-test",
		),
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
	}

	result, err := stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		t.Fatalf("run status=%q", result.Run.Status)
	}
	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: "goal-first-test"},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 visible: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("ranking visible=%+v", visible)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, stack)
	if len(all) != 1 {
		t.Fatalf("all=%+v", all)
	}
	candidate := all[0]
	if candidate.Status != orquestarunqueue.RunStatusClosedV0 ||
		candidate.AppRef != "app-ref-previo-goal-first" ||
		candidate.PriorityScore != 91 ||
		candidate.FairnessGroupRef != "fairness-goal-first" ||
		candidate.AttemptGroup.GroupRef != "attempt-group-goal-first" ||
		candidate.ParentRunRef != "parent-run-goal-first" ||
		candidate.SupersedesRunRef != "superseded-run-goal-first" ||
		candidate.RescueReason != "goal-first-test-preserve" ||
		len(candidate.WorksetClaims) != 1 ||
		candidate.WorksetClaims[0].ClaimRef != claim.ClaimRef ||
		!codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-previa-goal-first") ||
		!codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-goal-first-queue-terminal-sync") {
		t.Fatalf("candidate=%+v", candidate)
	}
}

func TestCodexStackObserveAppDirectorGoalExecutorV0UsaWrapperYSincronizaCola(t *testing.T) {
	ctx := context.Background()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	observer.result = orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-first-mcp-required-test",
		),
		EvidenceRefs:      spec.ClosurePolicy.RequiredEvidenceRefs,
		DomainReceiptRefs: []string{"domain-receipt-ref-goal-first-mcp-001"},
	}
	executor := NewCodexStackObserveAppDirectorGoalExecutorV0(&stack)

	result, err := executor.Execute(ctx, orquestamcp.MCPObserveAppDirectorGoalToolInputV0{
		RequestID: "req-goal-first-mcp-observe-001",
		RunRef:    runRef,
	})

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != orquestamcp.MCPObserveAppDirectorGoalEstadoOKV0 ||
		result.GoalRef != spec.GoalRef ||
		result.RunStatus != string(orquestacoreworkflow.OrchestrationRunStatusClosedV0) {
		t.Fatalf("result=%+v", result)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, stack)
	if len(all) != 1 || all[0].RunRef != runRef || all[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue no sincronizada por executor: %+v", all)
	}
}

func TestObserveActiveGoalWorksV0UsaWrapperYSincronizaCola(t *testing.T) {
	ctx := context.Background()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	observer.result = orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-first-active-required-test",
		),
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
	}

	result, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{
		List: orquestagoal.GoalWorkStateListRequestV0{MaxItems: 3},
	})

	if err != nil {
		t.Fatalf("ObserveActiveGoalWorksV0: %v", err)
	}
	if len(result.Observations) != 1 ||
		result.Observations[0].State.RunRef != runRef ||
		!result.Observations[0].Terminal ||
		!result.Observations[0].Accepted {
		t.Fatalf("result=%+v", result)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, stack)
	if len(all) != 1 ||
		all[0].RunRef != runRef ||
		all[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue no sincronizada por observacion activa: %+v", all)
	}
}

func TestObserveAppDirectorGoalV0SincronizaColaStoppedSinCandidatoPrevio(t *testing.T) {
	ctx := context.Background()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	observer.result = orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		EvidenceRefs:    spec.ClosurePolicy.RequiredEvidenceRefs,
	}

	result, err := stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework {
		t.Fatalf("result=%+v", result)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, stack)
	if len(all) != 1 {
		t.Fatalf("all=%+v", all)
	}
	candidate := all[0]
	if candidate.Status != orquestarunqueue.RunStatusStoppedV0 ||
		candidate.PriorityScore != 77 ||
		candidate.AppRef == "" ||
		!codexStackStringInSetV0(candidate.EvidenceRefs, "evidence-ref-goal-first-queue-terminal-sync") {
		t.Fatalf("candidate=%+v", candidate)
	}
	visible, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		ctx,
		orquestarunqueue.RunQueueReadRequestV0{QueueRef: "goal-first-test"},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 visible: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("ranking visible=%+v", visible)
	}
}

func TestObserveAppDirectorGoalV0SincronizaColaStoppedSiGoalInvalid(t *testing.T) {
	ctx := context.Background()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	observer.result = orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusInvalidV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		EvidenceRefs:    []string{"evidence-ref-goal-first-invalid-terminal"},
	}

	result, err := stack.ObserveAppDirectorGoalV0(
		ctx,
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: runRef},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
	}
	if result.Status != orquestagoal.GoalStatusInvalidV0 ||
		result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		result.Closure.Accepted ||
		!result.Closure.NeedsRework {
		t.Fatalf("result=%+v", result)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, stack)
	if len(all) != 1 ||
		all[0].RunRef != runRef ||
		all[0].Status != orquestarunqueue.RunStatusStoppedV0 {
		t.Fatalf("queue terminal invalid=%+v", all)
	}
}

func TestObserveAppDirectorGoalV0ReanudaTrasRestartDesdeStateFile(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	launcher := &goalFirstQueueLauncherForTestV0{}
	startStore := goalFirstQueueStateFileStoreForTestV0(t, root)
	startStack := goalFirstQueueStateFileStackForTestV0(
		startStore,
		launcher,
		&goalFirstQueueObserverForTestV0{},
		orquestarunmemory.NewRunMemoryStoreV0(),
	)

	started, err := orquestaappdirectorservice.StartAppDirectorV0(
		ctx,
		goalFirstQueueStartRequestForTestV0(),
		startStack.Ports,
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if started.GoalRef == "" || len(launcher.specs) != 1 {
		t.Fatalf("started=%+v specs=%d", started, len(launcher.specs))
	}

	reopenedStore := goalFirstQueueStateFileStoreForTestV0(t, root)
	spec := launcher.specs[0]
	observer := &goalFirstQueueObserverForTestV0{result: orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-first-restart-required-test",
		),
		EvidenceRefs:      spec.ClosurePolicy.RequiredEvidenceRefs,
		DomainReceiptRefs: []string{"domain-receipt-ref-goal-first-restart-001"},
	}}
	restartedStack := goalFirstQueueStateFileStackForTestV0(
		reopenedStore,
		&goalFirstQueueLauncherForTestV0{},
		observer,
		orquestarunmemory.NewRunMemoryStoreV0(),
	)

	result, err := restartedStack.ObserveAppDirectorGoalV0(
		ctx,
		orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{
			RunRef:        started.Run.RunID,
			CorrelationID: "corr-goal-first-restart-observe-001",
			RequestedBy:   "orquesta-app-codex-stack-goal-first-restart-test",
		},
	)
	if err != nil {
		t.Fatalf("ObserveAppDirectorGoalV0 tras restart: %v", err)
	}
	if result.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!result.Closure.Accepted ||
		result.GoalRef != started.GoalRef {
		t.Fatalf("result=%+v", result)
	}
	persistedRun, err := reopenedStore.LoadRunV0(ctx, started.Run.RunID)
	if err != nil {
		t.Fatalf("LoadRunV0 reopened: %v", err)
	}
	if persistedRun.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(persistedRun.Closures) != 1 ||
		len(persistedRun.Validations) != 1 {
		t.Fatalf("persistedRun=%+v", persistedRun)
	}
	persistedState, err := reopenedStore.LoadGoalWorkStateV0(ctx, started.Run.RunID)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0 reopened: %v", err)
	}
	if persistedState.Status != orquestagoal.GoalStatusCompleteV0 ||
		persistedState.LastResult == nil ||
		persistedState.LastClosure == nil ||
		!persistedState.LastClosure.Accepted {
		t.Fatalf("persistedState=%+v", persistedState)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, restartedStack)
	if len(all) != 1 ||
		all[0].RunRef != started.Run.RunID ||
		all[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue terminal tras restart=%+v", all)
	}
}

func startGoalFirstQueueSyncStackForTestV0(
	t *testing.T,
) (StackV0, *goalFirstQueueObserverForTestV0, *goalFirstQueueLauncherForTestV0, orquestaappdirectorservice.StartAppDirectorResultV0) {
	t.Helper()
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	queue := orquestarunmemory.NewRunMemoryStoreV0()
	goalStates := newGoalFirstQueueStateStoreForTestV0()
	launcher := &goalFirstQueueLauncherForTestV0{}
	observer := &goalFirstQueueObserverForTestV0{}
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:             runStore,
		EventSink:            eventSink,
		OutboxLedger:         ledger,
		GoalLauncher:         launcher,
		GoalObserver:         observer,
		GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:       goalStates,
		Dispatchers:          []orquestacionnucleoapp.OutboxDispatcherBindingV0{{}},
	}
	stack := StackV0{
		Ports: ports,
		Stores: StoresV0{
			RunStore:          runStore,
			EventSink:         eventSink,
			OutboxLedger:      ledger,
			RunQueue:          queue,
			AppGoalStateStore: goalStates,
		},
		RunQueue: RunQueueConfigV0{QueueRef: "goal-first-test", DefaultPriorityScore: 77},
		Clock: func() time.Time {
			return time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
		},
	}
	started, err := orquestaappdirectorservice.StartAppDirectorV0(
		context.Background(),
		goalFirstQueueStartRequestForTestV0(),
		stack.Ports,
	)
	if err != nil {
		t.Fatalf("StartAppDirectorV0: %v", err)
	}
	if started.GoalRef == "" || len(launcher.specs) != 1 {
		t.Fatalf("started=%+v specs=%d", started, len(launcher.specs))
	}
	return stack, observer, launcher, started
}

func goalFirstQueueStateFileStoreForTestV0(
	t *testing.T,
	root string,
) *orquestastatefile.StoreV0 {
	t.Helper()
	store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: root})
	if err != nil {
		t.Fatalf("NewStoreV0: %v", err)
	}
	return store
}

func goalFirstQueueStateFileStackForTestV0(
	store *orquestastatefile.StoreV0,
	launcher *goalFirstQueueLauncherForTestV0,
	observer *goalFirstQueueObserverForTestV0,
	queue orquestarunqueue.RunQueuePortV0,
) StackV0 {
	ledger := orquestacionnucleoapp.NewInMemoryOutboxLedgerV0()
	ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
		RunStore:             store,
		EventSink:            store,
		OutboxLedger:         ledger,
		GoalLauncher:         launcher,
		GoalObserver:         observer,
		GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		GoalStateStore:       store,
		Dispatchers:          []orquestacionnucleoapp.OutboxDispatcherBindingV0{{}},
	}
	return StackV0{
		Ports: ports,
		Stores: StoresV0{
			RunStore:          store,
			EventSink:         store,
			OutboxLedger:      ledger,
			RunQueue:          queue,
			AppGoalStateStore: store,
		},
		RunQueue: RunQueueConfigV0{QueueRef: "goal-first-test", DefaultPriorityScore: 77},
		Clock: func() time.Time {
			return time.Date(2026, 6, 25, 13, 0, 0, 0, time.UTC)
		},
	}
}

func goalFirstQueueStartRequestForTestV0() orquestaappdirectorservice.StartAppDirectorRequestV0 {
	observability := true
	return orquestaappdirectorservice.StartAppDirectorRequestV0{
		RunRef:        "run-goal-first-queue-sync-001",
		ProjectRef:    "project-goal-first-queue-sync-001",
		OccurredAt:    "2026-06-25T11:00:00Z",
		CorrelationID: "corr-goal-first-queue-sync-001",
		RequestedBy:   "orquesta-app-codex-stack-goal-first-test",
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			RequestID:     "request-ref-goal-first-queue-sync-001",
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

func goalFirstQueueRequiredArtifactRefsV0(spec orquestagoal.GoalWorkSpecV0) []string {
	refs := make([]string, 0, len(spec.ArtifactContracts))
	for _, contract := range spec.ArtifactContracts {
		if contract.Required {
			refs = append(refs, contract.ArtifactRef)
		}
	}
	return refs
}

func goalFirstQueueRequiredTestResultsV0(
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

func listGoalFirstQueueCandidatesForTestV0(
	t *testing.T,
	stack StackV0,
) []orquestarunqueue.RunSchedulingCandidateV0 {
	t.Helper()
	candidates, err := stack.Stores.RunQueue.ListRunSchedulingCandidatesV0(
		context.Background(),
		orquestarunqueue.RunQueueReadRequestV0{
			QueueRef:             "goal-first-test",
			IncludeNonExecutable: true,
		},
	)
	if err != nil {
		t.Fatalf("ListRunSchedulingCandidatesV0 all: %v", err)
	}
	return candidates
}

type goalFirstQueueLauncherForTestV0 struct {
	specs   []orquestagoal.GoalWorkSpecV0
	receipt orquestagoal.GoalLaunchReceiptV0
	err     error
}

func (launcher *goalFirstQueueLauncherForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	launcher.specs = append(launcher.specs, spec)
	if launcher.err != nil || launcher.receipt.Status != "" || launcher.receipt.GoalRef != "" {
		return launcher.receipt, launcher.err
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "thread-ref-goal-first-queue-sync-001",
		EvidenceRefs:    []string{"evidence-ref-goal-first-queue-launch"},
	}, nil
}

type goalFirstQueueObserverForTestV0 struct {
	result orquestagoal.GoalWorkResultV0
}

func (observer *goalFirstQueueObserverForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if observer.result.GoalRef == "" {
		return orquestagoal.GoalWorkResultV0{}, fmt.Errorf("goal result no configurado: %s", request.GoalRef)
	}
	return observer.result, nil
}

type goalFirstQueueStateStoreForTestV0 struct {
	states map[string]orquestagoal.GoalWorkStateV0
}

func newGoalFirstQueueStateStoreForTestV0() *goalFirstQueueStateStoreForTestV0 {
	return &goalFirstQueueStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
}

func (store *goalFirstQueueStateStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *goalFirstQueueStateStoreForTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	state, ok := store.states[runRef]
	if !ok {
		return orquestagoal.GoalWorkStateV0{}, fmt.Errorf("goal state no encontrado: %s", runRef)
	}
	return state, nil
}

func (store *goalFirstQueueStateStoreForTestV0) ListGoalWorkStatesV0(
	_ context.Context,
	request orquestagoal.GoalWorkStateListRequestV0,
) ([]orquestagoal.GoalWorkStateV0, error) {
	request = orquestagoal.NormalizeGoalWorkStateListRequestV0(request)
	refs := make([]string, 0, len(store.states))
	if len(request.RunRefs) > 0 {
		refs = append(refs, request.RunRefs...)
	} else {
		for runRef := range store.states {
			refs = append(refs, runRef)
		}
	}
	sort.Strings(refs)
	out := make([]orquestagoal.GoalWorkStateV0, 0, len(refs))
	for _, ref := range refs {
		state, ok := store.states[strings.TrimSpace(ref)]
		if !ok {
			continue
		}
		normalized, err := orquestagoal.NewGoalWorkStateV0(state)
		if err != nil {
			return nil, err
		}
		if !orquestagoal.GoalWorkStateMatchesListRequestV0(normalized, request) {
			continue
		}
		out = append(out, normalized)
		if request.MaxItems > 0 && len(out) >= request.MaxItems {
			break
		}
	}
	return out, nil
}
