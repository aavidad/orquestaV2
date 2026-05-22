package orquestacionnucleoapp

import (
	"context"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

func TestReviewReworkReplanCandidateProviderV0ProgressiveLoopSplitCreatesSchedulableTasks(t *testing.T) {
	runRef := "run-nucleo-review-rework-split-001"
	taskA := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-review-split-a", "app/rework_split_a.go")
	taskB := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-review-split-b", "app/rework_split_b.go")
	run := mustReviewGateReadyRunV0(t, runRef)
	run.FunctionContracts = []string{"contract:function:rework-split:v0"}
	taskStore := NewInMemoryWorkflowTaskStoreV0(
		reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-nucleo-001", "app/rework_original.go"),
	)
	store := NewInMemoryRunStoreV0(run)
	ledger := NewInMemoryOutboxLedgerV0()
	sink := NewInMemoryEventSinkV0()
	service := ServiceV0{
		RunStore:  store,
		EventSink: sink,
		CandidateProvider: WorkflowTaskCandidateProviderV0{
			Base: ReviewReworkReplanCandidateProviderV0{
				Base: ReviewGateCandidateProviderV0{
					GateSource:  reviewReworkLoopGateSourceV0{},
					RequestedBy: "orquesta-nucleo-test",
				},
				PlanSource: reviewReworkSplitPlanSourceV0{
					Tasks: []orquestacoreworkflow.WorkflowTaskV0{taskA, taskB},
				},
				TaskWriter:  taskStore,
				RequestedBy: "orquesta-nucleo-test",
			},
			TaskStore:   taskStore,
			RequestedBy: "orquesta-nucleo-test",
		},
		OutboxLedger:      ledger,
		MaxCommands:       12,
		MaxOutboxPerCycle: 4,
	}

	result, err := service.RunProgressiveLoopV0(context.Background(), ProgressiveLoopRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-05-10T10:20:00Z",
		MaxBursts:            4,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 4,
		CorrelationID:        "corr-review-rework-split-001",
		EvidenceRefs:         []string{"evidence-ref-review-rework-split-001"},
		Dispatchers: []OutboxDispatcherBindingV0{
			capacityDecisionDispatcherForTestV0(store, sink, ledger),
		},
	})
	if err != nil {
		t.Fatalf("progressive loop split: %v result=%+v", err, result)
	}
	assertReviewReworkSplitRunV0(t, result.Run, taskA.TaskID, taskB.TaskID)
	if _, err := taskStore.LoadWorkflowTasksV0(context.Background(), runRef, []string{taskA.TaskID, taskB.TaskID}); err != nil {
		t.Fatalf("split tasks no materializadas en store: %v", err)
	}
	assertReviewReworkProgressiveEventsV0(t, sink, []string{
		orquestacoreworkflow.OrchestrationEventReviewRequestedV0,
		orquestacoreworkflow.OrchestrationEventReviewResultRecordedV0,
		orquestacoreworkflow.OrchestrationEventReworkRequestedV0,
		orquestacoreworkflow.OrchestrationEventReplanDecisionRecordedV0,
		orquestacoreworkflow.OrchestrationEventPhaseOpenedV0,
		orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
		orquestacoreworkflow.OrchestrationEventMicrotaskCreatedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityRequestedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventCapacityDecidedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
		orquestacoreworkflow.OrchestrationEventAgentRequestedV0,
	})
}

func TestReviewReworkReplanSplitTaskV0ValidatesRecursiveParentLimits(t *testing.T) {
	runRef := "run-nucleo-review-rework-recursive-split-001"

	t.Run("rechaza_depth_imposible", func(t *testing.T) {
		parent := reviewReworkRecursiveSplitParentForTestV0(runRef)
		child := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-depth", "app/rework_child_depth.go")
		child.DelegationDepth = parent.DelegationDepth + 2

		_, _, err := reviewReworkRecursiveSplitCandidatesForTestV0(t, parent, child)
		assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "split_task.delegation_depth")
	})

	t.Run("rechaza_fanout_imposible", func(t *testing.T) {
		parent := reviewReworkRecursiveSplitParentForTestV0(runRef)
		parent.MaxChildAgents = 1
		parent.ChildTaskRefs = nil
		childA := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-a", "app/rework_child_a.go")
		childB := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-b", "app/rework_child_b.go")

		_, store, err := reviewReworkRecursiveSplitCandidatesForTestV0(t, parent, childA, childB)
		assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "split_task.max_child_agents")
		if _, loadErr := store.LoadWorkflowTasksV0(context.Background(), runRef, []string{childA.TaskID}); loadErr == nil {
			t.Fatalf("split_task invalida no debe guardarse: %s", childA.TaskID)
		}
	})

	t.Run("acepta_parent_child_valido", func(t *testing.T) {
		parent := reviewReworkRecursiveSplitParentForTestV0(runRef)
		child := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-valid", "app/rework_child_valid.go")
		parent.ChildTaskRefs = []string{child.TaskID}

		candidates, store, err := reviewReworkRecursiveSplitCandidatesForTestV0(t, parent, child)
		if err != nil {
			t.Fatalf("split_task recursiva valida: %v", err)
		}
		if len(candidates) != 1 || candidates[0].Payload.Task.TaskID != child.TaskID {
			t.Fatalf("candidates=%+v", candidates)
		}
		stored, err := store.LoadWorkflowTasksV0(context.Background(), runRef, []string{child.TaskID})
		if err != nil {
			t.Fatalf("child no guardado: %v", err)
		}
		got := stored[0]
		if got.ParentTaskRef != parent.TaskID ||
			got.DelegationDepth != parent.DelegationDepth+1 ||
			got.WaveRef != parent.WaveRef ||
			got.CohortRef != parent.CohortRef {
			t.Fatalf("linaje recursivo no conservado: got=%+v parent=%+v", got, parent)
		}
	})

	t.Run("rechaza_fanout_con_hijos_ya_persistidos_por_parent_ref", func(t *testing.T) {
		parent := reviewReworkRecursiveSplitParentForTestV0(runRef)
		parent.MaxChildAgents = 2
		parent.ChildTaskRefs = nil
		childA := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-persisted-a", "app/rework_persisted_a.go")
		childB := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-persisted-b", "app/rework_persisted_b.go")
		childC := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-candidate-c", "app/rework_candidate_c.go")

		_, store, err := reviewReworkRecursiveSplitCandidatesWithStoredTasksForTestV0(
			t,
			parent,
			[]string{parent.TaskID},
			[]orquestacoreworkflow.WorkflowTaskV0{childA, childB},
			childC,
		)
		assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "split_task.max_child_agents")
		if _, loadErr := store.LoadWorkflowTasksV0(context.Background(), runRef, []string{childC.TaskID}); loadErr == nil {
			t.Fatalf("split_task invalida no debe guardarse: %s", childC.TaskID)
		}
	})

	t.Run("acepta_fanout_con_hijos_persistidos_dentro_limite", func(t *testing.T) {
		parent := reviewReworkRecursiveSplitParentForTestV0(runRef)
		parent.MaxChildAgents = 3
		parent.ChildTaskRefs = nil
		persisted := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-persisted-ok", "app/rework_persisted_ok.go")
		childA := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-ok-a", "app/rework_child_ok_a.go")
		childB := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-ok-b", "app/rework_child_ok_b.go")

		candidates, _, err := reviewReworkRecursiveSplitCandidatesWithStoredTasksForTestV0(
			t,
			parent,
			[]string{parent.TaskID},
			[]orquestacoreworkflow.WorkflowTaskV0{persisted},
			childA,
			childB,
		)
		if err != nil {
			t.Fatalf("split_task recursiva con hijos persistidos dentro de limite: %v", err)
		}
		if len(candidates) != 2 {
			t.Fatalf("candidates=%+v", candidates)
		}
	})

	t.Run("acepta_task_ajena_del_run_sin_exigirla_en_store", func(t *testing.T) {
		parent := reviewReworkRecursiveSplitParentForTestV0(runRef)
		child := reviewReworkRecursiveSplitChildForTestV0(parent, "task-ref-recursive-child-legacy-run", "app/rework_child_legacy.go")

		candidates, _, err := reviewReworkRecursiveSplitCandidatesWithRunTasksForTestV0(
			t,
			parent,
			[]string{parent.TaskID, "task-ref-legacy-ajena-solo-evento"},
			child,
		)
		if err != nil {
			t.Fatalf("split_task recursiva no debe exigir task ajena en store: %v", err)
		}
		if len(candidates) != 1 || candidates[0].Payload.Task.TaskID != child.TaskID {
			t.Fatalf("candidates=%+v", candidates)
		}
	})
}

func TestReviewReworkReplanSplitTaskV0AcceptsNeutralCurrentRunPhase(t *testing.T) {
	runRef := "run-nucleo-review-rework-split-doc-001"
	task := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-review-split-doc", "docs/tema.md")
	task.PhaseID = orquestacoreworkflow.OrchestrationPhaseDocumentacionV0
	task.WorkProfileKind = orquestacoreworkflow.WorkProfileDomainWorkV0

	got, err := reviewReworkSplitTaskV0(
		SchedulerCandidateRequestV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				RunID:        runRef,
				CurrentPhase: orquestacoreworkflow.OrchestrationPhaseDocumentacionV0,
			},
		},
		task,
	)
	if err != nil {
		t.Fatalf("split_task documentacion/domain_work valida: %v", err)
	}
	if got.PhaseID != orquestacoreworkflow.OrchestrationPhaseDocumentacionV0 ||
		got.WorkProfileKind != orquestacoreworkflow.WorkProfileDomainWorkV0 {
		t.Fatalf("task neutral no conservada: %+v", got)
	}
}

func TestReviewReworkReplanSplitTaskV0RejectsPhaseOutsideCurrentRun(t *testing.T) {
	runRef := "run-nucleo-review-rework-split-phase-guard-001"
	task := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-review-split-doc-forbidden", "docs/tema.md")
	task.PhaseID = orquestacoreworkflow.OrchestrationPhaseDocumentacionV0
	task.WorkProfileKind = orquestacoreworkflow.WorkProfileDomainWorkV0

	_, err := reviewReworkSplitTaskV0(
		SchedulerCandidateRequestV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				RunID:        runRef,
				CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			},
		},
		task,
	)
	assertNucleoErrorV0(t, err, ErrNucleoOrquestacionInvalidoV0, "split_task.phase_id")
}

type reviewReworkSplitPlanSourceV0 struct {
	Tasks []orquestacoreworkflow.WorkflowTaskV0
}

func (source reviewReworkSplitPlanSourceV0) BuildReviewReworkReplanPlansV0(
	_ context.Context,
	request ReviewReworkReplanPlanRequestV0,
) ([]ReviewReworkReplanPlanV0, error) {
	if !reviewReworkRunHasRefV0(request.Run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") ||
		reviewReworkRunContainsRefV0(request.Run.Agents, WorkflowTaskAgentRequestRefV0("task-ref-review-split-a")) {
		return nil, nil
	}
	return []ReviewReworkReplanPlanV0{reviewReworkSplitPlanV0(request.Run.RunID, source.Tasks)}, nil
}

func reviewReworkSplitPlanV0(
	runRef string,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) ReviewReworkReplanPlanV0 {
	plan := reviewReworkRetryPlanV0(runRef)
	plan.CandidateRef = "review-rework-split-candidate-ref-nucleo-review-001"
	plan.ReplanRef = "replan-ref-nucleo-review-split-001"
	plan.SignalRef = "review-rework-split-signal-ref-nucleo-review-001"
	plan.RequestedAction = orquestacorereplanner.ReplanActionSplitTaskV0
	plan.CapacityRequestRef = ""
	plan.AgentRequestID = ""
	plan.Summary = "Dividir tarea tras revision no aceptada."
	plan.SplitTasks = tasks
	return plan
}

func reviewReworkSplitWorkflowTaskV0(
	runRef string,
	taskRef string,
	writeSet string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion: orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:        taskRef,
		RunID:         runRef,
		PhaseID:       orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Title:         "Microtarea de retrabajo",
		Summary:       "Trabajo compacto tras revision no aceptada.",
		WriteSet:      []string{writeSet},
		AcceptanceCriteria: []string{
			"Solo se modifica el write-set declarado.",
			"La entrega queda lista para revision.",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{
			{ContractRef: "contract:function:rework-split:v0", FunctionName: "NewWorkflowTaskV0"},
		},
	}
}

func reviewReworkRecursiveSplitParentForTestV0(
	runRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	parent := reviewReworkSplitWorkflowTaskV0(runRef, "task-ref-recursive-parent", "app/rework_parent.go")
	parent.WaveRef = "wave-ref-recursive-001"
	parent.CohortRef = "cohort-ref-recursive-001"
	parent.DelegationDepth = 1
	parent.MaxChildAgents = 2
	return parent
}

func reviewReworkRecursiveSplitChildForTestV0(
	parent orquestacoreworkflow.WorkflowTaskV0,
	taskRef string,
	writeSet string,
) orquestacoreworkflow.WorkflowTaskV0 {
	child := reviewReworkSplitWorkflowTaskV0(parent.RunID, taskRef, writeSet)
	child.ParentTaskRef = parent.TaskID
	child.WaveRef = parent.WaveRef
	child.CohortRef = parent.CohortRef
	child.DelegationDepth = parent.DelegationDepth + 1
	return child
}

func reviewReworkRecursiveSplitCandidatesForTestV0(
	t *testing.T,
	parent orquestacoreworkflow.WorkflowTaskV0,
	tasks ...orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestadirector.ReplanMicrotaskCandidateV0, *InMemoryWorkflowTaskStoreV0, error) {
	t.Helper()
	return reviewReworkRecursiveSplitCandidatesWithRunTasksForTestV0(
		t,
		parent,
		[]string{parent.TaskID},
		tasks...,
	)
}

func reviewReworkRecursiveSplitCandidatesWithRunTasksForTestV0(
	t *testing.T,
	parent orquestacoreworkflow.WorkflowTaskV0,
	runTaskRefs []string,
	tasks ...orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestadirector.ReplanMicrotaskCandidateV0, *InMemoryWorkflowTaskStoreV0, error) {
	t.Helper()
	return reviewReworkRecursiveSplitCandidatesWithStoredTasksForTestV0(
		t,
		parent,
		runTaskRefs,
		nil,
		tasks...,
	)
}

func reviewReworkRecursiveSplitCandidatesWithStoredTasksForTestV0(
	t *testing.T,
	parent orquestacoreworkflow.WorkflowTaskV0,
	runTaskRefs []string,
	storedTasks []orquestacoreworkflow.WorkflowTaskV0,
	tasks ...orquestacoreworkflow.WorkflowTaskV0,
) ([]orquestadirector.ReplanMicrotaskCandidateV0, *InMemoryWorkflowTaskStoreV0, error) {
	t.Helper()
	initialTasks := append([]orquestacoreworkflow.WorkflowTaskV0{parent}, storedTasks...)
	store := NewInMemoryWorkflowTaskStoreV0(initialTasks...)
	provider := ReviewReworkReplanCandidateProviderV0{
		TaskWriter:  store,
		RequestedBy: "orquesta-nucleo-test",
	}
	candidates, err := provider.reviewReworkMicrotaskCandidatesV0(
		context.Background(),
		SchedulerCandidateRequestV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				RunID:        parent.RunID,
				CurrentPhase: orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
				Tasks:        runTaskRefs,
			},
			OccurredAt:    "2026-05-10T10:30:00Z",
			CorrelationID: "corr-review-rework-recursive-split-001",
		},
		ReviewReworkReplanPlanV0{SplitTasks: tasks},
		orquestacoreworkflow.ReplanDecisionActionSplitTaskV0,
	)
	return candidates, store, err
}

func assertReviewReworkSplitRunV0(t *testing.T, run orquestacoreworkflow.OrchestrationRunV0, taskA string, taskB string) {
	t.Helper()
	if run.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 {
		t.Fatalf("current_phase=%s run=%+v", run.CurrentPhase, run)
	}
	for _, taskRef := range []string{taskA, taskB} {
		if !reviewReworkRunContainsRefV0(run.Tasks, taskRef) {
			t.Fatalf("task %s no autorizada: %v", taskRef, run.Tasks)
		}
		if !reviewReworkRunContainsRefV0(run.CapacityRequests, WorkflowTaskCapacityRequestRefV0(taskRef)) ||
			!reviewReworkRunHasRefV0(run.CapacityDecisions, WorkflowTaskCapacityRequestRefV0(taskRef), "#capacity_decision:") {
			t.Fatalf("capacity para %s incompleta: requests=%v decisions=%v", taskRef, run.CapacityRequests, run.CapacityDecisions)
		}
		if !reviewReworkRunContainsRefV0(run.Agents, WorkflowTaskAgentRequestRefV0(taskRef)) {
			t.Fatalf("agent para %s no solicitado: %v", taskRef, run.Agents)
		}
	}
	if !reviewReworkRunHasRefV0(run.ReworkRequests, "rework-request-ref-nucleo-review-001", "#review_result:") ||
		!reviewReworkRunHasRefV0(run.ReplanDecisions, "replan-ref-nucleo-review-split-001", "#source:") {
		t.Fatalf("rework/replan incompleto: reworks=%v replans=%v", run.ReworkRequests, run.ReplanDecisions)
	}
}
