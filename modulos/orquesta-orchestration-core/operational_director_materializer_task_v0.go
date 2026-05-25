package orquestacionnucleoapp

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoroperativo "orquesta/modulos/orquesta-director-operativo"
)

func operationalDirectorWorkflowTaskFromItemV0(
	request OperationalDirectorPlanMaterializeRequestV0,
	item orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	itemTaskRefs map[string]string,
	itemWaveRefs map[string]string,
) (orquestacoreworkflow.WorkflowTaskV0, error) {
	criteria := operationalDirectorWorkflowTaskCriteriaV0(request, item, itemTaskRefs, itemWaveRefs)
	waveRef := strings.TrimSpace(itemWaveRefs[item.ItemID])
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:               itemTaskRefs[item.ItemID],
		RunID:                request.Plan.RunRef,
		PhaseID:              request.TargetPhaseID,
		WorkProfileKind:      orquestacoreworkflow.WorkProfileKindV0(item.WorkProfileKind),
		Title:                item.Title,
		Summary:              "Materializar item del Director Operativo.",
		WriteSet:             append([]string(nil), item.WriteSet...),
		AcceptanceCriteria:   criteria,
		RequiredTests:        append([]string(nil), item.RequiredTests...),
		DependsOn:            operationalDirectorWorkflowTaskDependsOnV0(item.DependsOn, itemTaskRefs),
		ParentTaskRef:        strings.TrimSpace(itemTaskRefs[item.ParentItemID]),
		CohortRef:            operationalDirectorCohortRefV0(request.Plan, waveRef),
		WaveRef:              waveRef,
		DelegationDepth:      item.DelegationDepth,
		MaxDelegationDepth:   request.Plan.MaxDelegationDepth,
		MaxChildAgents:       item.MaxChildItems,
		MaxSubagentsPerAgent: request.Plan.MaxSubagentsPerAgent,
		MaxRecursiveAgents:   request.Plan.MaxRecursiveAgents,
		ChildTaskRefs:        operationalDirectorWorkflowTaskChildRefsV0(item.ChildItemIDs, itemTaskRefs),
		FunctionContractRefs: append([]orquestacoreworkflow.WorkflowFunctionContractRefV0(nil), request.FunctionContractRefs...),
	}
	return orquestacoreworkflow.NewWorkflowTaskV0(task)
}

func operationalDirectorWorkflowTaskCriteriaV0(
	request OperationalDirectorPlanMaterializeRequestV0,
	item orquestadirectoroperativo.OperationalDirectorWorkItemV0,
	itemTaskRefs map[string]string,
	itemWaveRefs map[string]string,
) []string {
	criteria := []string{"objetivo_actual: " + request.Plan.Objective}
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.plan_ref", request.Plan.PlanRef)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.request_ref", request.Plan.RequestRef)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.wave_ref", itemWaveRefs[item.ItemID])
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.source_step_ref", item.SourceStepID)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.source_item_ref", item.ItemID)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.parent_item_ref", item.ParentItemID)
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.parent_task_ref", itemTaskRefs[item.ParentItemID])
	criteria = operationalDirectorAppendCriterionV0(criteria, "operational_director.child_item_refs", strings.Join(item.ChildItemIDs, ","))
	criteria = append(criteria, item.AcceptanceCriteria...)
	return criteria
}

func operationalDirectorCohortRefV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	waveRef string,
) string {
	waveRef = strings.TrimSpace(waveRef)
	if waveRef == "" {
		return ""
	}
	return "cohort-operational-director-" +
		operationalDirectorSafeRefPartV0(plan.RequestRef) +
		"-" +
		operationalDirectorSafeRefPartV0(waveRef)
}

func operationalDirectorAppendCriterionV0(criteria []string, key string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return criteria
	}
	return append(criteria, key+": "+value)
}

func operationalDirectorWorkflowTaskDependsOnV0(
	itemDependsOn []string,
	itemTaskRefs map[string]string,
) []string {
	out := make([]string, 0, len(itemDependsOn))
	for _, itemRef := range itemDependsOn {
		if taskRef := strings.TrimSpace(itemTaskRefs[itemRef]); taskRef != "" {
			out = append(out, taskRef)
		}
	}
	return out
}

func operationalDirectorWorkflowTaskChildRefsV0(
	itemChildIDs []string,
	itemTaskRefs map[string]string,
) []string {
	out := make([]string, 0, len(itemChildIDs))
	for _, itemRef := range itemChildIDs {
		if taskRef := strings.TrimSpace(itemTaskRefs[itemRef]); taskRef != "" {
			out = append(out, taskRef)
		}
	}
	return out
}

func operationalDirectorCreateMicrotaskCommandV0(
	materializer OperationalDirectorPlanMaterializerV0,
	request OperationalDirectorPlanMaterializeRequestV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	suffix := operationalDirectorSafeRefPartV0(task.TaskID)
	return orquestacoreworkflow.NewCreateMicrotaskCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-operational-director-create-" + suffix,
			RunID:          request.Plan.RunRef,
			IdempotencyKey: "idem-operational-director-create-" + suffix,
			CorrelationID:  request.CorrelationID,
			RequestedBy:    operationalDirectorRequestedByV0(materializer.RequestedBy),
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.CreateMicrotaskCommandPayloadV0{Task: task},
	)
}

func operationalDirectorTaskRefV0(
	plan orquestadirectoroperativo.OperationalDirectorPlanV0,
	item orquestadirectoroperativo.OperationalDirectorWorkItemV0,
) string {
	return "task-operational-director-" +
		operationalDirectorSafeRefPartV0(plan.RequestRef) +
		"-" +
		operationalDirectorSafeRefPartV0(item.SourceStepID)
}
