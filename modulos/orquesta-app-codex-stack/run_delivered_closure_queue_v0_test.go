package orquestaappcodexstack

import (
	"context"
	"testing"
	"time"

	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestStackDrainQueueStatusMantieneEntregadoOperativoParaCierreFormalV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-stack-delivered-operational-closure-001"
	taskRef := "task-ref-stack-delivered-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	task := stackDeliveredOperationalTaskForTestV0(runRef, taskRef)
	task.AcceptanceCriteria = []string{"Director Operativo cierre causal con evidencias"}
	task.FunctionContractRefs = []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
		ContractRef: "contract:function:director-operativo:v0",
	}}
	stack := StackV0{Stores: StoresV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}}
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef),
		},
	}
	status, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil {
		t.Fatalf("stackDrainQueueStatusForCoordinatorV0: %v", err)
	}
	if status != "" {
		t.Fatalf("queue_status=%q, want activo para cierre formal", status)
	}
}

func TestStackDrainQueueStatusMantieneAutoprogrammingActivoParaRevisionYCierreV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-stack-delivered-autoprogramming-001"
	taskRef := "task-autoprogramming-stack-delivered-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	task := stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef)
	stack := StackV0{Stores: StoresV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}}
	run := stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef)
	run.AppSpecRef = "app-spec-ref-autoprogramming-queue-001"
	run.FunctionContracts = []string{"function:BuildAutoprogrammingProgrammableWorkV0"}
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{Run: run},
	}
	status, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil {
		t.Fatalf("stackDrainQueueStatusForCoordinatorV0: %v", err)
	}
	if status != "" {
		t.Fatalf("queue_status=%q, want activo para revision/cierre autoprogramming", status)
	}
}

func TestStackDrainQueueStatusMantieneProgramacionGenericaActivaParaCierreV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-stack-delivered-programming-001"
	taskRef := "task-stack-delivered-programming-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	task := stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef)
	task.TaskID = taskRef
	task.Title = "Programacion generica entregada"
	task.FunctionContractRefs = []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
		FunctionName: "BuildCompleteAppBootstrapV0",
	}}
	stack := StackV0{Stores: StoresV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}}
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef),
		},
	}
	status, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)
	if err != nil {
		t.Fatalf("stackDrainQueueStatusForCoordinatorV0: %v", err)
	}
	if status != "" {
		t.Fatalf("queue_status=%q, want activo para cierre de programacion generica", status)
	}
}

func TestStackDrainQueueStatusNoRetieneDomainWorkPorTaskSourceDirectorV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-stack-delivered-domain-work-001"
	taskRef := "task-ref-app-change-domain-work-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Trabajo externo entregado",
		WriteSet:        []string{"opes-salidas/tema"},
		AcceptanceCriteria: []string{
			"operational_director.task_source: director_decision",
			"entrega externa verificable por dominio",
		},
		RequiredTests: []string{"validar contrato externo de dominio"},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef:  "contract:function:app-change:domain-work:v0",
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	stack := StackV0{Stores: StoresV0{
		TaskStore: orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
	}}
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef),
		},
	}

	status, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, result)

	if err != nil {
		t.Fatalf("stackDrainQueueStatusForCoordinatorV0: %v", err)
	}
	if status != orquestarunqueue.RunStatusDeliveredV0 {
		t.Fatalf("queue_status=%q, want delivered para domain_work entregado", status)
	}
}

func TestRunGlobalTickV0ReabreDeliveredOPESConACKIngeridoYCierraTaskV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-stack-opes-delivered-open-ack-001"
	changeRef := "opes-job-delivered-open-ack-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-opes-delivered-open-ack-001"
	requiredTest := "opes-domain-test-delivered-open-ack-001"
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Resolver job OPES entregado y pendiente de cierre",
		WriteSet:        []string{"external/opes/draft_content_block/delivered-open-ack"},
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto por conector y el cierre usa refs causales",
			"operational_director.task_source: director_decision",
		},
		RequiredTests: []string{requiredTest},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	run := stackOperationalClosureRunForTestV0(runRef, taskRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseRevisionV0
	run.Phases = stackDeliveredClosurePhaseCatalogForTestV0(orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.Reviews = []string{"review-request-ref-" + deliveryRef}
	run.ReviewResults = []string{
		"review-result-ref-" + deliveryRef +
			"#review_result:accepted#review_request:review-request-ref-" + deliveryRef +
			"#delivery:" + deliveryRef,
	}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	if err := stack.Stores.AppChangeStore.SaveAppChangeRequestV0(ctx, opesDomainWorkAppChangeRecordForTestV0(runRef, changeRef)); err != nil {
		t.Fatalf("SaveAppChangeRequestV0: %v", err)
	}
	evidence := opesDomainRequiredTestEvidenceForTestV0(runRef, taskRef, deliveryRef, requiredTest)
	evidenceStore := orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(evidence)
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, opesAcceptedSubmissionRecordForFixtureV0(opesDomainWorkClosureFixtureForTestV0{
		Run:         run,
		Task:        task,
		DeliveryRef: deliveryRef,
	}, "delivered-open-ack")); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	stack.Stores.RequiredTestEvidenceStore = evidenceStore
	stack.DomainDelivery.Ledger = ledger
	stack.Ports.RequiredTestEvidenceStore = evidenceStore
	stack.Ports.OperationalClosureSource = nil
	planRef := "operational-director-plan-director-decisions-" + runRef
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0()
	stack.Stores.OperationalPlanStateWriter = planStateStore
	stack.Stores.OperationalPlanStateStore = planStateStore
	stack.Ports.OperationalPlanStateWriter = planStateStore
	stack.Ports.OperationalPlanStateStore = planStateStore
	if err := planStateStore.SaveOperationalDirectorPlanStateV0(
		ctx,
		stackDeliveredOPESPlanStateReadyForCloseTestV0(runRef, planRef, taskRef, agentRef, deliveryRef, evidence.EvidenceRef),
	); err != nil {
		t.Fatalf("SaveOperationalDirectorPlanStateV0: %v", err)
	}
	if _, err := stack.Stores.RunQueue.SetRunPriorityV0(ctx, orquestarunqueue.RunQueuePriorityCommandV0{
		RunRef:        runRef,
		QueueRef:      DefaultRunQueueRefV0,
		AppRef:        "opes",
		Status:        orquestarunqueue.RunStatusDeliveredV0,
		PriorityScore: 90,
		UpdatedAt:     time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
		EvidenceRefs:  []string{"evidence-ref-test-delivered-open-ack"},
	}); err != nil {
		t.Fatalf("SetRunPriorityV0: %v", err)
	}

	result, err := stack.RunGlobalTickV0(ctx, orquestaruncoordinator.RunCoordinatorTickCommandV0{
		QueueRef:         DefaultRunQueueRefV0,
		MaxRuns:          1,
		AllowLegacyDrain: true,
		DrainLimits: orquestaruncoordinator.RunDrainLimitsV0{
			MaxBursts:            4,
			MaxStepsPerBurst:     6,
			MaxDispatchesPerWait: 4,
			MaxCommands:          16,
			MaxOutboxPerCycle:    8,
			MaxDecisionCycles:    2,
			MaxExternalWaits:     1,
		},
		OccurredAt: time.Date(2026, 6, 12, 10, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RunGlobalTickV0: %v result=%+v", err, result)
	}
	closed := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	if !codexStackStringInSetV0(closed.ClosedTasks, taskRef) ||
		closed.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 {
		state, _ := planStateStore.LoadOperationalDirectorPlanStateV0(ctx, runRef, planRef)
		t.Fatalf("run delivered/open no cerro: status=%s tasks=%v delivered=%v closed=%v closures=%v state=%+v result=%+v", closed.Status, closed.Tasks, closed.DeliveredTasks, closed.ClosedTasks, closed.Closures, state, result)
	}
}

func TestEnrichQueuedOperationalDirectorDrainRequestV0RecuperaDomainWorkOpenReviewBloqueadoConEntregaAceptadaV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-stack-opes-blocked-open-review-001"
	changeRef := "opes-job-blocked-open-review-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-opes-blocked-open-review-001"
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Resolver job OPES entregado y bloqueado al abrir revision",
		WriteSet:        []string{"external/opes/draft_content_block/blocked-open-review"},
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto por conector y el supervisor reabre el cierre causal",
			"operational_director.task_source: director_decision",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	run := stackOperationalClosureRunForTestV0(runRef, taskRef)
	run.Status = orquestacoreworkflow.OrchestrationRunStatusBlockedV0
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.Phases = stackDeliveredClosurePhaseCatalogForTestV0(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.Blockers = []string{
		"app-director-decision-director-decision-apply-error-director-decision-director-decision-app-change-open-review-" + changeRef,
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, opesAcceptedSubmissionRecordForFixtureV0(opesDomainWorkClosureFixtureForTestV0{
		Run:         run,
		Task:        task,
		DeliveryRef: deliveryRef,
	}, "blocked-open-review")); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	stack.DomainDelivery.Ledger = ledger

	_, err := stack.enrichQueuedOperationalDirectorDrainRequestV0(ctx, DrainRunRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-06-12T10:15:00Z",
		CorrelationID:              "corr-blocked-open-review",
		WaitAgentRefs:              []string{agentRef},
		OperationalDirectorPlanRef: "plan-ref-existing",
	})
	if err != nil {
		t.Fatalf("enrichQueuedOperationalDirectorDrainRequestV0: %v", err)
	}
	recovered := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	if recovered.Status == orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		len(recovered.Blockers) != 0 {
		t.Fatalf("run no recuperado: status=%s blockers=%v", recovered.Status, recovered.Blockers)
	}
}

func TestEnrichQueuedOperationalDirectorDrainRequestV0RecuperaAutoprogrammingOpenReviewBloqueadoConACKV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-stack-autoprogramming-blocked-open-review-001"
	taskRef := "task-autoprogramming-blocked-open-review-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	task := stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef)
	run := stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef)
	run.SchemaVersion = orquestacoreworkflow.OrchestrationRunSchemaVersionV0
	run.ProjectRef = "project-autoprogramming-open-review-001"
	run.AppSpecRef = "app-spec-ref-autoprogramming-blocked-open-review-001"
	run.Status = orquestacoreworkflow.OrchestrationRunStatusBlockedV0
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.Phases = stackDeliveredClosurePhaseCatalogForTestV0(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.Blockers = []string{
		"app-director-decision-director-decision-apply-error-director-decision-director-decision-autoprogramming-open-review-" + taskRef,
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}

	_, err := stack.enrichQueuedOperationalDirectorDrainRequestV0(ctx, DrainRunRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-06-22T12:10:00Z",
		CorrelationID:              "corr-autoprogramming-open-review",
		WaitAgentRefs:              []string{agentRef},
		OperationalDirectorPlanRef: "plan-ref-existing",
	})
	if err != nil {
		t.Fatalf("enrichQueuedOperationalDirectorDrainRequestV0: %v", err)
	}
	recovered := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	if recovered.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 ||
		len(recovered.Blockers) != 0 ||
		recovered.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		t.Fatalf("run no recuperado: status=%s current_phase=%s blockers=%v", recovered.Status, recovered.CurrentPhase, recovered.Blockers)
	}
}

func TestDrainRunV0RecuperaAutoprogrammingOpenReviewBloqueadoConACKV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-stack-autoprogramming-blocked-open-review-drain-001"
	taskRef := "task-autoprogramming-blocked-open-review-drain-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	task := stackDeliveredAutoprogrammingTaskForTestV0(runRef, taskRef)
	run := stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef)
	run.SchemaVersion = orquestacoreworkflow.OrchestrationRunSchemaVersionV0
	run.ProjectRef = "project-autoprogramming-open-review-drain-001"
	run.AppSpecRef = "app-spec-ref-autoprogramming-blocked-open-review-drain-001"
	run.Status = orquestacoreworkflow.OrchestrationRunStatusBlockedV0
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.Phases = stackDeliveredClosurePhaseCatalogForTestV0(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	run.Blockers = []string{
		"app-director-decision-director-decision-apply-error-director-decision-director-decision-autoprogramming-open-review-" + taskRef,
	}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}

	_, err := stack.DrainRunV0(ctx, DrainRunRequestV0{
		RunRef:               runRef,
		OccurredAt:           "2026-06-22T12:15:00Z",
		CorrelationID:        "corr-autoprogramming-open-review-drain",
		WaitAgentRefs:        []string{agentRef},
		MaxExternalWaits:     0,
		MaxBursts:            1,
		MaxStepsPerBurst:     1,
		MaxCommands:          2,
		MaxDecisionCycles:    1,
		MaxDispatchesPerWait: 1,
		MaxOutboxPerCycle:    1,
	})
	if err != nil {
		t.Fatalf("DrainRunV0: %v", err)
	}
	recovered := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	if recovered.Status == orquestacoreworkflow.OrchestrationRunStatusBlockedV0 ||
		len(recovered.Blockers) != 0 {
		t.Fatalf("run sigue bloqueado: status=%s current_phase=%s blockers=%v", recovered.Status, recovered.CurrentPhase, recovered.Blockers)
	}
}

func TestEnrichQueuedOperationalDirectorDrainRequestV0ReparaRevisionParcialDomainWorkAceptadoV0(t *testing.T) {
	ctx := context.Background()
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	runRef := "run-stack-opes-partial-review-phase-001"
	changeRef := "opes-job-partial-review-phase-001"
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	deliveryRef := "delivery-ref-opes-partial-review-phase-001"
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Resolver job OPES con revision parcialmente abierta",
		WriteSet:        []string{"external/opes/draft_content_block/partial-review-phase"},
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto por conector y el supervisor repara la fase de revision",
			"operational_director.task_source: director_decision",
		},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	run := stackOperationalClosureRunForTestV0(runRef, taskRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhaseProgramacionV0
	run.Phases = stackDeliveredClosurePhaseCatalogForTestV0(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)
	for index := range run.Phases {
		if run.Phases[index].ID == orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
			run.Phases[index].OpenedAt = "2026-06-12T10:25:00Z"
		}
	}
	run.StartedAgents = []string{agentRef}
	run.DeliveredAgents = []string{agentRef}
	run.DeliveredTasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.Reviews = []string{"review-request-ref-" + deliveryRef}
	if err := stack.Stores.RunStore.SaveRunV0(ctx, run); err != nil {
		t.Fatalf("SaveRunV0: %v", err)
	}
	if err := stack.Stores.TaskStore.SaveWorkflowTaskV0(ctx, task); err != nil {
		t.Fatalf("SaveWorkflowTaskV0: %v", err)
	}
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, opesAcceptedSubmissionRecordForFixtureV0(opesDomainWorkClosureFixtureForTestV0{
		Run:         run,
		Task:        task,
		DeliveryRef: deliveryRef,
	}, "partial-review-phase")); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	stack.DomainDelivery.Ledger = ledger

	_, err := stack.enrichQueuedOperationalDirectorDrainRequestV0(ctx, DrainRunRequestV0{
		RunRef:                     runRef,
		OccurredAt:                 "2026-06-12T10:26:00Z",
		CorrelationID:              "corr-partial-review-phase",
		WaitAgentRefs:              []string{agentRef},
		OperationalDirectorPlanRef: "plan-ref-existing",
	})
	if err != nil {
		t.Fatalf("enrichQueuedOperationalDirectorDrainRequestV0: %v", err)
	}
	recovered := mustLoadCodexStackRunForTestV0(t, stack, runRef)
	if recovered.CurrentPhase != orquestacoreworkflow.OrchestrationPhaseRevisionV0 {
		t.Fatalf("current_phase=%s, want revision phases=%+v", recovered.CurrentPhase, recovered.Phases)
	}
}

func stackDeliveredOPESPlanStateReadyForCloseTestV0(
	runRef string,
	planRef string,
	taskRef string,
	agentRef string,
	deliveryRef string,
	requiredTestEvidenceRef string,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-" + planRef,
		PlanRef:             planRef,
		RequestRef:          "request-ref-" + planRef,
		RunRef:              runRef,
		ProjectRef:          "opes",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-replan-or-close",
		ActiveWaveRef:       "wave-ref-" + planRef,
		ActiveCohortRef:     "cohort-ref-" + planRef,
		ActiveParentTaskRef: taskRef,
		RequiredTestRefs:    []string{"validar contrato externo de dominio"},
		ObservedAt:          "2026-06-12T10:00:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:    "step-launch-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-ref-" + planRef,
				CohortRef: "cohort-ref-" + planRef,
				TaskRefs:  []string{taskRef},
				AgentRefs: []string{agentRef},
			},
			{
				StepID:    "step-wait-subagents",
				Kind:      orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:    orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:   "wave-ref-" + planRef,
				CohortRef: "cohort-ref-" + planRef,
				TaskRefs:  []string{taskRef},
				AgentRefs: []string{agentRef},
			},
			{
				StepID:             "step-review-deliveries",
				Kind:               orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:             orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:            "wave-ref-" + planRef,
				CohortRef:          "cohort-ref-" + planRef,
				TaskRefs:           []string{taskRef},
				AgentRefs:          []string{agentRef},
				DeliveryRefs:       []string{deliveryRef},
				ReviewResultRefs:   []string{"review-result-ref-" + deliveryRef},
				AcceptedReviewRefs: []string{"accepted-review-ref-" + deliveryRef},
			},
			{
				StepID:                   "step-run-required-tests",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:                  "wave-ref-" + planRef,
				CohortRef:                "cohort-ref-" + planRef,
				TaskRefs:                 []string{taskRef},
				AgentRefs:                []string{agentRef},
				DeliveryRefs:             []string{deliveryRef},
				ReviewResultRefs:         []string{"review-result-ref-" + deliveryRef},
				RequiredTestEvidenceRefs: []string{requiredTestEvidenceRef},
			},
			{
				StepID:                   "step-replan-or-close",
				Kind:                     orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:                   orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:                  "wave-ref-" + planRef,
				CohortRef:                "cohort-ref-" + planRef,
				ParentTaskRef:            taskRef,
				TaskRefs:                 []string{taskRef},
				AgentRefs:                []string{agentRef},
				DeliveryRefs:             []string{deliveryRef},
				ReviewResultRefs:         []string{"review-result-ref-" + deliveryRef},
				RequiredTestEvidenceRefs: []string{requiredTestEvidenceRef},
			},
		},
	}
}

func stackDeliveredClosurePhaseCatalogForTestV0(
	active orquestacoreworkflow.OrchestrationPhaseIDV0,
) []orquestacoreworkflow.OrchestrationPhaseV0 {
	phases := orquestacoreworkflow.OrchestrationPhaseCatalogV0()
	activeSeen := false
	for index := range phases {
		if phases[index].ID == active {
			phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusActiveV0
			activeSeen = true
			continue
		}
		if !activeSeen {
			phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusClosedV0
			continue
		}
		phases[index].Status = orquestacoreworkflow.OrchestrationPhaseStatusPendingV0
	}
	return phases
}

func TestStackDrainQueueStatusConservaDeliveredLegacySinTaskStoreV0(t *testing.T) {
	runRef := "run-stack-delivered-legacy-001"
	taskRef := "task-ref-stack-delivered-legacy-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Run: stackDeliveredRunForQueueTestV0(runRef, taskRef, agentRef),
		},
	}
	status, err := (StackV0{}).stackDrainQueueStatusForCoordinatorV0(context.Background(), result)
	if err != nil {
		t.Fatalf("stackDrainQueueStatusForCoordinatorV0: %v", err)
	}
	if status != orquestarunqueue.RunStatusDeliveredV0 {
		t.Fatalf("queue_status=%q", status)
	}
}

func TestStackDrainQueueStatusMantieneActivoSiQuedanTareasAbiertasV0(t *testing.T) {
	run := stackDeliveredRunForQueueTestV0(
		"run-stack-open-rework-001",
		"task-original-001",
		orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0("task-original-001"),
	)
	run.Tasks = append(run.Tasks, "task-rework-001")
	result := orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
	}

	status := stackDrainQueueStatusV0(result)

	if status != "" {
		t.Fatalf("queue_status=%q, want activo por tarea abierta", status)
	}
}

func stackDeliveredRunForQueueTestV0(
	runRef string,
	taskRef string,
	agentRef string,
) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		Tasks:           []string{taskRef},
		Agents:          []string{agentRef},
		StartedAgents:   []string{agentRef},
		DeliveredAgents: []string{agentRef},
		Deliveries:      []string{"delivery-ref-" + taskRef},
		DeliveredTasks:  []string{taskRef},
	}
}

func stackDeliveredAutoprogrammingTaskForTestV0(
	runRef string,
	taskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileImplementationV0,
		Title:           "Autoprogramacion entregada",
		WriteSet:        []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{
			"autoprogramacion revisada con evidencias causales",
		},
		RequiredTests: []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "BuildAutoprogrammingProgrammableWorkV0",
		}},
	}
}

func stackDeliveredOperationalTaskForTestV0(
	runRef string,
	taskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Cierre formal de run entregado",
		WriteSet:           []string{"modulos/orquesta-app-codex-stack"},
		AcceptanceCriteria: []string{"operational_director.plan_ref: plan-ref"},
		CohortRef:          "cohort-ref-director-operativo-001",
		WaveRef:            "wave-ref-cierre-001",
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			ContractRef: "contract:function:operational-director:v0",
		}},
	}
}
