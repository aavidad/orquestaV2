package orquestaautoprogramming

import (
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func autoprogrammingWorkflowTaskForGroupV1(
	request AutoprogrammingRequestV0,
	group AutoprogrammingTaskGroupV0,
	writeSet []string,
	dependsOn []string,
	requiredTests []string,
	groupIndex int,
	selected autoprogrammingSelectedWorkProfileV1,
) (orquestacoreworkflow.WorkProfileV0, orquestacoreworkflow.WorkflowTaskV0, AutoprogrammingRequestIssueV0) {
	taskRef := autoprogrammingProgrammableTaskRefV0(request.RequestRef, groupIndex)
	limits := autoprogrammingDelegationLimitsForRequestV0(request)
	profile, err := orquestacoreworkflow.NewWorkProfileV0(orquestacoreworkflow.WorkProfileV0{
		SchemaVersion:        orquestacoreworkflow.WorkProfileSchemaVersionV0,
		ProfileRef:           "profile-" + taskRef,
		ProfileKind:          selected.kind,
		TaskRef:              taskRef,
		RunRef:               strings.TrimSpace(request.RequestRef),
		Title:                autoprogrammingTitleForGroupV0(group, groupIndex),
		Objective:            autoprogrammingObjectiveForGroupV0(group),
		Summary:              autoprogrammingSummaryForGroupV0(group),
		ScopeRefs:            append([]string(nil), writeSet...),
		RequiredTests:        autoprogrammingRequiredTestsForGroupV0(requiredTests, group),
		AcceptanceCriteria:   autoprogrammingAcceptanceCriteriaForGroupV0(group),
		MaxDelegationDepth:   limits.maxDelegationDepth,
		MaxChildAgents:       limits.maxSubagentsPerAgent,
		MaxSubagentsPerAgent: limits.maxSubagentsPerAgent,
		MaxRecursiveAgents:   limits.maxRecursiveAgents,
		FunctionContractRefs: autoprogrammingFunctionContractRefsV1(selected),
		DependsOn:            dependsOn,
	})
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingWorkProfileIssueV0(err)
	}
	task, err := orquestacoreworkflow.WorkflowTaskFromWorkProfileV0(profile)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingWorkflowTaskIssueV0(err)
	}
	task.ContextRefs = autoprogrammingContextRefsForGroupV0(request, group)
	task, issue := EnsureAutoprogrammingWorkflowTaskAcceptedByCoreV0(task)
	if issue.Code != "" {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{}, issue
	}
	task, err = orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return orquestacoreworkflow.WorkProfileV0{}, orquestacoreworkflow.WorkflowTaskV0{},
			autoprogrammingWorkflowTaskIssueV0(err)
	}
	return profile, task, AutoprogrammingRequestIssueV0{}
}

func autoprogrammingFunctionContractRefsV1(
	selected autoprogrammingSelectedWorkProfileV1,
) []orquestacoreworkflow.WorkflowFunctionContractRefV0 {
	refs := []orquestacoreworkflow.WorkflowFunctionContractRefV0{
		{FunctionName: "ValidateAutoprogrammingRequestV1"},
		{FunctionName: "BuildAutoprogrammingProgrammableWorkV1"},
	}
	return append(refs, selected.functionContractRefs...)
}
