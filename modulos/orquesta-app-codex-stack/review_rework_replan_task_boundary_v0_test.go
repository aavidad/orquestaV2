package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func TestReviewReworkReplanSourceV0CreaTareaCorreccionSiYaHayPadreV0(t *testing.T) {
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.Spec.AgentPacket.Task.WriteSet = []string{"modulos/orquesta-app-codex-stack"}
	descriptor.Spec.AgentPacket.Task.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"}
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.FunctionContracts = []string{"BuildAutoprogrammingProgrammableWorkV0"}
	request.Run.Agents = []string{descriptor.AgentRef}
	request.Run.StartedAgents = []string{descriptor.AgentRef}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	plan := plans[0]
	if plan.RequestedAction != orquestacorereplanner.ReplanActionSplitTaskV0 ||
		plan.AgentRequestID != "" ||
		len(plan.SplitTasks) != 1 {
		t.Fatalf("plan no crea split_task de correccion: %+v", plan)
	}
	task := plan.SplitTasks[0]
	if task.TaskID == "task-ref-target" ||
		task.RunID != request.Run.RunID ||
		task.PhaseID != orquestacoreworkflow.OrchestrationPhaseProgramacionV0 ||
		task.WorkProfileKind != orquestacoreworkflow.WorkProfileImplementationV0 {
		t.Fatalf("split task invalida: %+v", task)
	}
	if !reviewReworkPlanHasEvidenceForTestV0(plan.EvidenceRefs, "evidence-ref-review-rework-task-boundary") ||
		!reviewReworkPlanHasEvidenceForTestV0(task.DependsOn, "task-ref-target") ||
		!reviewReworkPlanHasEvidenceForTestV0(task.WriteSet, "modulos/orquesta-app-codex-stack") ||
		len(task.FunctionContractRefs) != 1 ||
		task.FunctionContractRefs[0].ContractRef != "BuildAutoprogrammingProgrammableWorkV0" {
		t.Fatalf("refs/evidencia incompletas plan=%v task=%+v", plan.EvidenceRefs, task)
	}
}

func TestReviewReworkReplanSourceV0CreaTareaCorreccionSinFunctionContractsV0(t *testing.T) {
	descriptor := reviewReworkDescriptorForTestV0("delivery-ref-target", "task-ref-target")
	descriptor.Spec.AgentPacket.Task.WriteSet = []string{"modulos/orquesta-app-codex-stack"}
	descriptor.Spec.AgentPacket.Task.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-app-codex-stack"}
	source := ReviewReworkReplanSourceV0{
		Store: orquestaruntimecodexdelivery.NewInMemoryCodexReceiptDescriptorStoreV0(descriptor),
	}
	request := reviewReworkPlanRequestForTestV0(false)
	request.Run.FunctionContracts = nil
	request.Run.Agents = []string{descriptor.AgentRef}
	request.Run.StartedAgents = []string{descriptor.AgentRef}

	plans, err := source.BuildReviewReworkReplanPlansV0(context.Background(), request)
	if err != nil {
		t.Fatalf("BuildReviewReworkReplanPlansV0: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("plans=%+v", plans)
	}
	task := plans[0].SplitTasks[0]
	if task.TaskID == "" ||
		!reviewReworkPlanHasEvidenceForTestV0(task.DependsOn, "task-ref-target") ||
		len(task.FunctionContractRefs) != 0 {
		t.Fatalf("tarea de correccion sin contratos opcionales invalida: plan=%+v task=%+v", plans[0], task)
	}
}
