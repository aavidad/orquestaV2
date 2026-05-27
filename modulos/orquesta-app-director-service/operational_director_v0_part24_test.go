package orquestaappdirectorservice

import (
	"context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	"testing"
)

func TestContinueAppDirectorV0RecuperaWaitStatePersistidoSinAmpliarCohorte(t *testing.T) {
	runRef := "run-app-director-operational-plan-wait-recovery"
	planRef := "plan-ref-wait-recovery"
	waitRef := "wait-ref-old"
	taskA := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-wait-recovery-a", "wave-01", "cohort-a")
	taskB := serviceWorkflowTaskForWaitRefsTestV0(runRef, "task-wait-recovery-b", "wave-01", "cohort-a")
	agentA := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskA.TaskID)
	agentB := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskB.TaskID)
	state := orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:   orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:        "state-ref-wait-recovery",
		PlanRef:         planRef,
		RequestRef:      "request-ref-wait-recovery",
		RunRef:          runRef,
		ProjectRef:      "orquesta",
		Mode:            orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:          orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:    "step-wait-subagents",
		ActiveWaveRef:   "wave-01",
		ActiveCohortRef: "cohort-a",
		PendingAgentRefs: []string{
			agentA,
			agentB,
		},
		ObservedAt: "2026-05-17T13:59:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{{
			StepID:           "step-wait-subagents",
			Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
			Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
			WaveRef:          "wave-01",
			CohortRef:        "cohort-a",
			TaskRefs:         []string{taskA.TaskID, taskB.TaskID},
			WaitRefs:         []string{waitRef},
			AgentRefs:        []string{agentA, agentB},
			PendingAgentRefs: []string{agentA, agentB},
			BlockerRefs:      []string{"wait-subagents"},
		}},
	}
	waitState := orquestacionnucleoapp.WorkflowTaskWaitStateV0{
		SchemaVersion:    orquestacionnucleoapp.WorkflowTaskWaitStateSchemaVersionV0,
		WaitRef:          waitRef,
		RunRef:           runRef,
		ReasonCode:       orquestacionnucleoapp.WorkflowTaskWaitReasonCohortInProgressV0,
		CohortRef:        "cohort-a",
		WaveRef:          "wave-01",
		TaskRefs:         []string{taskA.TaskID},
		AgentRefs:        []string{agentA},
		PendingAgentRefs: []string{agentA},
		Status:           orquestacionnucleoapp.WorkflowTaskWaitStateStatusWaitingV0,
		Attempt:          1,
		ObservedAt:       "2026-05-17T13:58:00Z",
	}
	run := serviceRunForWaitRefsTestV0(runRef, taskA.TaskID, taskB.TaskID)
	planStateStore := orquestacionnucleoapp.NewInMemoryOperationalDirectorPlanStateStoreV0(state)
	waitStateStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskWaitStateStoreV0(waitState)
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(taskA, taskB)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(run)
	request := ContinueAppDirectorRequestV0{
		RunRef:                     runRef,
		OperationalDirectorPlanRef: planRef,
		OccurredAt:                 "2026-05-17T14:00:00Z",
		CorrelationID:              "corr-wait-recovery",
	}
	ports := StartAppDirectorPortsV0{
		RunStore:                  runStore,
		DirectorTaskStore:         taskStore,
		WaitStateStore:            waitStateStore,
		OperationalPlanStateStore: planStateStore,
	}
	reentered, err := continueRequestWithOperationalDirectorPlanStateV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("continueRequestWithOperationalDirectorPlanStateV0: %v", err)
	}
	if len(reentered.WaitAgentRefs) != 1 ||
		reentered.WaitAgentRefs[0] != agentA ||
		serviceStringInSetV0(reentered.WaitAgentRefs, agentB) ||
		reentered.WaitWaveRef != "" ||
		reentered.WaitCohortRef != "" ||
		reentered.WaitParentTaskRef != "" {
		t.Fatalf("reentered=%+v agentA=%s agentB=%s", reentered, agentA, agentB)
	}
	loopRequest, err := existingDirectorLoopRequestV0(context.Background(), reentered, ports)
	if err != nil {
		t.Fatalf("existingDirectorLoopRequestV0: %v", err)
	}
	if len(loopRequest.WaitAgentRefs) != 1 ||
		loopRequest.WaitAgentRefs[0] != agentA ||
		serviceStringInSetV0(loopRequest.WaitAgentRefs, agentB) {
		t.Fatalf("loop wait refs=%+v agentA=%s agentB=%s", loopRequest.WaitAgentRefs, agentA, agentB)
	}
}

func TestMaterializeContinueOperationalDirectorPlanV0UsaFaseActualSiNoSeIndica(t *testing.T) {
	runRef := "run-app-director-operational-plan-phase-001"
	contractRef := "contract:function:operational-director:v0"
	run := serviceRunForWaitRefsTestV0(runRef)
	run.CurrentPhase = orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0
	run.Phases = []orquestacoreworkflow.OrchestrationPhaseV0{{
		ID:                  orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
		Status:              orquestacoreworkflow.OrchestrationPhaseStatusActiveV0,
		RecommendedCapacity: orquestacoreworkflow.OrchestrationCapacityHighV0,
	}}
	run.FunctionContracts = []string{contractRef}
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0()

	materialized, err := materializeContinueOperationalDirectorPlanV0(context.Background(), ContinueAppDirectorRequestV0{
		RunRef:                  runRef,
		OccurredAt:              "2026-05-17T13:47:00Z",
		CorrelationID:           "corr-app-director-operational-plan-phase-001",
		RequestedBy:             "orquesta-app-director-test",
		OperationalDirectorPlan: serviceOperationalDirectorPlanForContinueTestV0(t, runRef),
	}, StartAppDirectorPortsV0{
		RunStore:          orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
		EventSink:         orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		DirectorTaskStore: taskStore,
	})
	if err != nil {
		t.Fatalf("materializeContinueOperationalDirectorPlanV0: %T %#v", err, err)
	}
	if len(materialized.Tasks) != 1 {
		t.Fatalf("materialized=%+v", materialized)
	}
	if materialized.Tasks[0].PhaseID != orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0 {
		t.Fatalf("phase_id=%s", materialized.Tasks[0].PhaseID)
	}
}

func serviceOperationalDirectorWideWaveWaitStateForTestV0(
	runRef string,
	planRef string,
	parentTaskRef string,
	waveRef string,
	cohortRef string,
	tasks ...serviceOperationalClosureTaskRefsForTestV0,
) orquestacionnucleoapp.OperationalDirectorPlanStateV0 {
	taskRefs := make([]string, 0, len(tasks))
	agentRefs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskRefs = append(taskRefs, task.TaskRef)
		agentRefs = append(agentRefs, orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(task.TaskRef))
	}
	return orquestacionnucleoapp.OperationalDirectorPlanStateV0{
		SchemaVersion:       orquestacionnucleoapp.OperationalDirectorPlanStateSchemaVersionV0,
		StateRef:            "state-ref-" + planRef,
		PlanRef:             planRef,
		RequestRef:          "request-ref-" + planRef,
		RunRef:              runRef,
		ProjectRef:          "orquesta",
		Mode:                orquestadirectoroperativo.OperationalDirectorModeProgrammingV0,
		Status:              orquestacionnucleoapp.OperationalDirectorPlanStateActiveV0,
		ActiveStepID:        "step-wait-subagents",
		ActiveWaveRef:       waveRef,
		ActiveCohortRef:     cohortRef,
		ActiveParentTaskRef: parentTaskRef,
		PendingAgentRefs:    agentRefs,
		RequiredTestRefs:    []string{"go test ./..."},
		ObservedAt:          "2026-05-22T13:59:00Z",
		Steps: []orquestacionnucleoapp.OperationalDirectorPlanStepStateV0{
			{
				StepID:        "step-launch-subagents",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepLaunchSubagentsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepAcceptedV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
				TaskRefs:      taskRefs,
				AgentRefs:     agentRefs,
			},
			{
				StepID:           "step-wait-subagents",
				Kind:             orquestadirectoroperativo.OperationalDirectorStepWaitSubagentsV0,
				Status:           orquestadirectoroperativo.OperationalDirectorStepRunningV0,
				WaveRef:          waveRef,
				CohortRef:        cohortRef,
				ParentTaskRef:    parentTaskRef,
				TaskRefs:         taskRefs,
				AgentRefs:        agentRefs,
				WaitRefs:         []string{"wait-ref-" + planRef},
				PendingAgentRefs: agentRefs,
				BlockerRefs:      []string{"wait-subagents"},
				Reason:           "wait-subagents-running",
			},
			{
				StepID:        "step-review-deliveries",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReviewDeliveriesV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-run-required-tests",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepRunRequiredTestsV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
			{
				StepID:        "step-replan-or-close",
				Kind:          orquestadirectoroperativo.OperationalDirectorStepReplanOrCloseV0,
				Status:        orquestadirectoroperativo.OperationalDirectorStepPendingV0,
				WaveRef:       waveRef,
				CohortRef:     cohortRef,
				ParentTaskRef: parentTaskRef,
			},
		},
	}
}

type serviceOperationalDirectorPlanStateReviewFixtureV0 struct {
	RunRef                  string
	PlanRef                 string
	TaskRef                 string
	AgentRef                string
	WaveRef                 string
	CohortRef               string
	ParentTaskRef           string
	DeliveryRef             string
	ReviewRequestID         string
	ReviewResultRef         string
	AcceptedReviewRef       string
	ReworkRequestRef        string
	ReplanDecisionRef       string
	RequiredTest            string
	RequiredTestEvidenceRef string
	Run                     orquestacoreworkflow.OrchestrationRunV0
	State                   orquestacionnucleoapp.OperationalDirectorPlanStateV0
	Events                  []orquestacoreworkflow.OrchestrationEventV0
}
