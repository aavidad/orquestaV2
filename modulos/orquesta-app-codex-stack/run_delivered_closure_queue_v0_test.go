package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
