package orquestaappcodexstack

import (
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

func reviewReworkPlanWithTaskBoundaryV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	plan orquestacionnucleoapp.ReviewReworkReplanPlanV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.ReviewReworkReplanPlanV0 {
	if !reviewReworkProgrammingAgentAlreadyAssignedToTaskV0(request.Run, plan.TaskRef, descriptors) {
		if reviewReworkFallbackPlanWouldDuplicateParentV0(request.Run, plan) {
			return orquestacionnucleoapp.ReviewReworkReplanPlanV0{}
		}
		return plan
	}
	task, ok := reviewReworkCorrectionTaskV0(request, plan, descriptors)
	if !ok {
		return orquestacionnucleoapp.ReviewReworkReplanPlanV0{}
	}
	plan.RequestedAction = orquestacorereplanner.ReplanActionSplitTaskV0
	plan.CapacityRequestRef = ""
	plan.AgentRequestID = ""
	plan.AgentRole = ""
	plan.SplitTasks = []orquestacoreworkflow.WorkflowTaskV0{task}
	plan.Summary = "Crear tarea de correccion tras revision; no relanzar segundo padre sobre la misma tarea."
	plan.EvidenceRefs = compactStringsV0(append(
		plan.EvidenceRefs,
		"evidence-ref-review-rework-task-boundary",
		task.TaskID,
	))
	plan.ReviewResult.Summary = plan.Summary
	plan.ReviewResult.EvidenceRefs = plan.EvidenceRefs
	return plan
}

func reviewReworkFallbackPlanWouldDuplicateParentV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	plan orquestacionnucleoapp.ReviewReworkReplanPlanV0,
) bool {
	if !reviewReworkPlanUsesTaskFallbackV0(plan) {
		return false
	}
	prefix := reviewReworkReplanTaskRetryAgentPrefixV0(plan.TaskRef)
	legacyPrefix := "agent-ref-" + reviewReworkReplanAgentStemV0(plan.TaskRef)
	for _, agentRef := range reviewReworkRunAgentRefsV0(run) {
		agentRef = strings.TrimSpace(agentRef)
		if agentRef == legacyPrefix || strings.HasPrefix(agentRef, prefix) {
			return true
		}
	}
	return false
}

func reviewReworkPlanUsesTaskFallbackV0(
	plan orquestacionnucleoapp.ReviewReworkReplanPlanV0,
) bool {
	return reviewReworkReplanStringInSetV0(
		plan.EvidenceRefs,
		"evidence-ref-review-rework-task-fallback",
	)
}

func reviewReworkRunAgentRefsV0(run orquestacoreworkflow.OrchestrationRunV0) []string {
	refs := []string{}
	refs = append(refs, run.Agents...)
	refs = append(refs, run.StartedAgents...)
	refs = append(refs, run.DeliveredAgents...)
	refs = append(refs, run.FailedAgents...)
	refs = append(refs, run.LostAgents...)
	refs = append(refs, run.StoppedAgents...)
	refs = append(refs, run.ConfirmedStoppedAgents...)
	return compactStringsV0(refs)
}

func reviewReworkCorrectionTaskV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	plan orquestacionnucleoapp.ReviewReworkReplanPlanV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (orquestacoreworkflow.WorkflowTaskV0, bool) {
	descriptor, ok := codexStackReviewGateDescriptorForDeliveryV0(descriptors, plan.ReviewResult.DeliveryRef)
	if !ok {
		return orquestacoreworkflow.WorkflowTaskV0{}, false
	}
	writeSet := compactStringsV0(descriptor.Spec.AgentPacket.Task.WriteSet)
	if len(writeSet) == 0 {
		return orquestacoreworkflow.WorkflowTaskV0{}, false
	}
	functionContracts := reviewReworkFunctionContractRefsFromRunV0(request.Run)
	taskID := reviewReworkCorrectionTaskIDV0(plan)
	task, err := orquestacoreworkflow.NewWorkflowTaskV0(orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:               taskID,
		RunID:                request.Run.RunID,
		PhaseID:              orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:      orquestacoreworkflow.WorkProfileImplementationV0,
		Title:                "Correccion de entrega tras revision",
		Summary:              "Completar correcciones en una tarea nueva conservando entrega valida.",
		WriteSet:             writeSet,
		AcceptanceCriteria:   []string{"Corregir la entrega rechazada sin relanzar otro agente padre sobre la tarea original."},
		RequiredTests:        compactStringsV0(descriptor.Spec.AgentPacket.Task.RequiredTests),
		DependsOn:            compactStringsV0([]string{plan.TaskRef}),
		ContextRefs:          []string{"context-ref-" + taskID},
		FunctionContractRefs: functionContracts,
	})
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, false
	}
	return task, true
}

func reviewReworkFunctionContractRefsFromRunV0(
	run orquestacoreworkflow.OrchestrationRunV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	contracts := compactStringsV0(run.FunctionContracts)
	if len(contracts) == 0 {
		return nil
	}
	refs := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(contracts))
	for _, contract := range contracts {
		refs = append(refs, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef: contract,
		})
	}
	return refs
}

func reviewReworkCorrectionTaskIDV0(
	plan orquestacionnucleoapp.ReviewReworkReplanPlanV0,
) string {
	stem := reviewReworkReplanAgentStemV0(plan.TaskRef)
	digest := codexStackDeterministicDigestV0(
		plan.ReworkRequestRef,
		plan.ReplanRef,
		plan.TaskRef,
		plan.ReviewResult.DeliveryRef,
	)
	if len(digest) > 32 {
		digest = digest[:32]
	}
	return "task-ref-review-rework-" + stem + "-" + digest
}
