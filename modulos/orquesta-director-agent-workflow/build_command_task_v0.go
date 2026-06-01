package orquestadirectoragentworkflow

import (
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func directorAgentWorkflowTaskV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) orquestacoreworkflow.WorkflowTaskV0 {
	return orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:        task.SchemaVersion,
		TaskID:               task.TaskID,
		RunID:                task.RunID,
		PhaseID:              orquestacoreworkflow.OrchestrationPhaseIDV0(task.PhaseID),
		WorkProfileKind:      orquestacoreworkflow.WorkProfileKindV0(task.WorkProfileKind),
		Title:                task.Title,
		Summary:              task.Summary,
		WriteSet:             task.WriteSet,
		AcceptanceCriteria:   directorAgentWorkflowTaskAcceptanceCriteriaV0(task),
		RequiredTests:        task.RequiredTests,
		DependsOn:            task.DependsOn,
		ContextRefs:          task.ContextRefs,
		ParentTaskRef:        task.ParentTaskRef,
		CohortRef:            task.CohortRef,
		WaveRef:              task.WaveRef,
		DelegationDepth:      task.DelegationDepth,
		MaxChildAgents:       task.MaxChildAgents,
		ChildTaskRefs:        task.ChildTaskRefs,
		FunctionContractRefs: directorAgentWorkflowFunctionRefsV0(task.FunctionContractRefs),
	}
}

func directorAgentWorkflowTaskAcceptanceCriteriaV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) []string {
	return directorAgentWorkflowOperationalAcceptanceCriteriaV0(task.PhaseID, task.AcceptanceCriteria)
}

func directorAgentWorkflowFunctionRefsV0(
	refs []orquestadirectoragent.DirectorAgentFunctionContractRefV0,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	result := make([]orquestacoreworkflow.WorkflowFunctionContractRefV0, 0, len(refs))
	for _, ref := range refs {
		result = append(result, orquestacoreworkflow.WorkflowFunctionContractRefV0{
			ContractRef:  ref.ContractRef,
			FunctionName: ref.FunctionName,
		})
	}
	return result
}
