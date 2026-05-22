package orquestacoreworkflow

func WorkflowTaskFromWorkProfileV0(profile WorkProfileV0) (WorkflowTaskV0, error) {
	profile = NormalizeWorkProfileV0(profile)
	if err := validateWorkProfileFieldsV0(profile); err != nil {
		return WorkflowTaskV0{}, err
	}
	return workflowTaskFromWorkProfileUncheckedV0(profile)
}

func workflowTaskFromWorkProfileUncheckedV0(profile WorkProfileV0) (WorkflowTaskV0, error) {
	definition, ok := LookupWorkProfileDefinitionV0(profile.ProfileKind)
	if !ok {
		return WorkflowTaskV0{}, workProfileErrorV0("profile_kind")
	}
	task := WorkflowTaskV0{
		SchemaVersion:        WorkflowTaskSchemaVersionV0,
		TaskID:               profile.TaskRef,
		RunID:                profile.RunRef,
		PhaseID:              profile.PhaseID,
		WorkProfileKind:      profile.ProfileKind,
		Title:                profile.Title,
		Summary:              workProfileTaskSummaryV0(profile),
		WriteSet:             append([]string(nil), profile.ScopeRefs...),
		AcceptanceCriteria:   workProfileTaskCriteriaV0(definition, profile),
		RequiredTests:        append([]string(nil), profile.RequiredTests...),
		DependsOn:            append([]string(nil), profile.DependsOn...),
		ParentTaskRef:        profile.ParentTaskRef,
		CohortRef:            profile.CohortRef,
		WaveRef:              profile.WaveRef,
		DelegationDepth:      profile.DelegationDepth,
		MaxDelegationDepth:   profile.MaxDelegationDepth,
		MaxChildAgents:       profile.MaxChildAgents,
		MaxSubagentsPerAgent: profile.MaxSubagentsPerAgent,
		MaxRecursiveAgents:   profile.MaxRecursiveAgents,
		ChildTaskRefs:        append([]string(nil), profile.ChildTaskRefs...),
		FunctionContractRefs: append([]WorkflowFunctionContractRefV0(nil), profile.FunctionContractRefs...),
	}
	normalized, err := NewWorkflowTaskV0(task)
	if err != nil {
		return WorkflowTaskV0{}, workProfileTaskErrorV0("workflow_task")
	}
	return normalized, nil
}

func workProfileTaskSummaryV0(profile WorkProfileV0) string {
	if profile.Summary != "" {
		return profile.Summary
	}
	return profile.Objective
}

func workProfileTaskCriteriaV0(
	definition WorkProfileDefinitionV0,
	profile WorkProfileV0,
) []string {
	return compactStringsV0(append(
		append([]string(nil), definition.DefaultCriteria...),
		profile.AcceptanceCriteria...,
	))
}
