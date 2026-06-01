package orquestacoreworkflow

import "strings"

func NormalizeWorkflowTaskV0(task WorkflowTaskV0) WorkflowTaskV0 {
	return WorkflowTaskV0{
		SchemaVersion:        normalizeWorkflowTaskSchemaVersionV0(task.SchemaVersion),
		TaskID:               strings.TrimSpace(task.TaskID),
		RunID:                strings.TrimSpace(task.RunID),
		PhaseID:              OrchestrationPhaseIDV0(strings.TrimSpace(string(task.PhaseID))),
		WorkProfileKind:      NormalizeWorkProfileKindV0(task.WorkProfileKind),
		Title:                strings.TrimSpace(task.Title),
		Summary:              strings.TrimSpace(task.Summary),
		WriteSet:             normalizeWorkflowTaskWriteSetV0(task.WriteSet),
		AcceptanceCriteria:   normalizeWorkflowTaskStringsV0(task.AcceptanceCriteria),
		RequiredTests:        normalizeWorkflowTaskStringsV0(task.RequiredTests),
		DependsOn:            normalizeWorkflowTaskStringsV0(task.DependsOn),
		ContextRefs:          normalizeWorkflowTaskContextRefsV0(task.ContextRefs),
		ParentTaskRef:        strings.TrimSpace(task.ParentTaskRef),
		CohortRef:            strings.TrimSpace(task.CohortRef),
		WaveRef:              strings.TrimSpace(task.WaveRef),
		DelegationDepth:      task.DelegationDepth,
		MaxDelegationDepth:   task.MaxDelegationDepth,
		MaxChildAgents:       task.MaxChildAgents,
		MaxSubagentsPerAgent: task.MaxSubagentsPerAgent,
		MaxRecursiveAgents:   task.MaxRecursiveAgents,
		ChildTaskRefs:        normalizeWorkflowTaskStringsV0(task.ChildTaskRefs),
		FunctionContractRefs: normalizeWorkflowFunctionContractRefsV0(task.FunctionContractRefs),
	}
}

func normalizeWorkflowFunctionContractRefsV0(refs []WorkflowFunctionContractRefV0) []WorkflowFunctionContractRefV0 {
	if refs == nil {
		return nil
	}
	normalized := make([]WorkflowFunctionContractRefV0, 0, len(refs))
	for _, ref := range refs {
		normalized = append(normalized, WorkflowFunctionContractRefV0{
			ContractRef:  strings.TrimSpace(ref.ContractRef),
			FunctionName: strings.TrimSpace(ref.FunctionName),
		})
	}
	return normalized
}

func normalizeWorkflowTaskContextRefsV0(values []string) []string {
	if values == nil {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func normalizeWorkflowTaskStringsV0(values []string) []string {
	if values == nil {
		return nil
	}
	normalized := make([]string, len(values))
	for i, value := range values {
		normalized[i] = strings.TrimSpace(value)
	}
	return normalized
}

func normalizeWorkflowTaskSchemaVersionV0(version string) string {
	if strings.TrimSpace(version) == "" {
		return WorkflowTaskSchemaVersionV0
	}
	return strings.TrimSpace(version)
}
