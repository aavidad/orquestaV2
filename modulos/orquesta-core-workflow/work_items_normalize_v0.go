package orquestacoreworkflow

import "strings"

func NormalizeWorkflowTaskV0(task WorkflowTaskV0) WorkflowTaskV0 {
	return WorkflowTaskV0{
		SchemaVersion:        normalizeWorkflowTaskSchemaVersionV0(task.SchemaVersion),
		TaskID:               strings.TrimSpace(task.TaskID),
		RunID:                strings.TrimSpace(task.RunID),
		PhaseID:              OrchestrationPhaseIDV0(strings.TrimSpace(string(task.PhaseID))),
		Title:                strings.TrimSpace(task.Title),
		Summary:              strings.TrimSpace(task.Summary),
		WriteSet:             normalizeWorkflowTaskStringsV0(task.WriteSet),
		AcceptanceCriteria:   normalizeWorkflowTaskStringsV0(task.AcceptanceCriteria),
		RequiredTests:        normalizeWorkflowTaskStringsV0(task.RequiredTests),
		DependsOn:            normalizeWorkflowTaskStringsV0(task.DependsOn),
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
