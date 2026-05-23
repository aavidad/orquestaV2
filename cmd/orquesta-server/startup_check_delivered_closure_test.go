package main

import (
	"context"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestStartupQueueStatusNoMarcaDeliveredSiQuedaCierreOperativoV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-startup-delivered-operational-closure-001"
	taskRef := "task-ref-startup-delivered-operational-closure-001"
	agentRef := orquestacionnucleoapp.WorkflowTaskAgentRequestRefV0(taskRef)
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreworkflow.OrchestrationRunV0{
		RunID:           runRef,
		Status:          orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		Tasks:           []string{taskRef},
		Agents:          []string{agentRef},
		StartedAgents:   []string{agentRef},
		DeliveredAgents: []string{agentRef},
		Deliveries:      []string{"delivery-ref-" + taskRef},
		DeliveredTasks:  []string{taskRef},
	})
	taskStore := orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(
		startupDeliveredOperationalTaskForTestV0(runRef, taskRef),
	)
	check := serverStartupCheckV0{
		Stack: orquestaappcodexstack.StackV0{
			Stores: orquestaappcodexstack.StoresV0{
				RunStore:  runStore,
				TaskStore: taskStore,
			},
		},
	}
	status, ok, err := check.startupQueueStatusFromRunStoreV0(ctx, runRef)
	if err != nil {
		t.Fatalf("startupQueueStatusFromRunStoreV0: %v", err)
	}
	if ok || status != "" {
		t.Fatalf("status=%q ok=%v, want seguir en cola activa", status, ok)
	}
}

func startupDeliveredOperationalTaskForTestV0(
	runRef string,
	taskRef string,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileImplementationV0,
		Title:              "Cierre formal startup",
		WriteSet:           []string{"cmd/orquesta-server"},
		AcceptanceCriteria: []string{"Director Operativo: cerrar con review y tests"},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "DirectorOperativoClosure",
		}},
	}
}
