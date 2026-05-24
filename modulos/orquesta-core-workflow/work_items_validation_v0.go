package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

func ValidateWorkflowTaskV0(task WorkflowTaskV0) error {
	if err := validateWorkflowTaskRequiredFieldsV0(task); err != nil {
		return err
	}
	if err := validateWorkflowTaskCollectionsV0(task); err != nil {
		return err
	}
	if err := validateWorkflowTaskForbiddenDetailsV0(task); err != nil {
		return err
	}
	return validateWorkflowTaskCompactPayloadV0(task)
}

func validateWorkflowTaskRequiredFieldsV0(task WorkflowTaskV0) error {
	if strings.TrimSpace(task.SchemaVersion) != WorkflowTaskSchemaVersionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "schema_version")
	}
	if strings.TrimSpace(task.TaskID) == "" {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "task_id")
	}
	if strings.TrimSpace(task.RunID) == "" {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "run_id")
	}
	if strings.TrimSpace(task.Title) == "" {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "title")
	}
	return ValidateOrchestrationPhaseIDV0(task.PhaseID)
}

func validateWorkflowTaskCollectionsV0(task WorkflowTaskV0) error {
	if err := validateWorkflowTaskWriteSetV0(task.WriteSet); err != nil {
		return err
	}
	if strings.TrimSpace(string(task.WorkProfileKind)) != "" {
		if _, ok := LookupWorkProfileDefinitionV0(task.WorkProfileKind); !ok {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "work_profile_kind")
		}
	}
	if err := validateRequiredWorkflowTaskStringsV0(task.AcceptanceCriteria, "acceptance_criteria"); err != nil {
		return err
	}
	if err := validateOptionalWorkflowTaskStringsV0(task.RequiredTests, "required_tests"); err != nil {
		return err
	}
	if err := validateWorkflowTaskDependsOnV0(task); err != nil {
		return err
	}
	if err := validateWorkflowTaskContextRefsV0(task.ContextRefs); err != nil {
		return err
	}
	if err := validateWorkflowTaskLineageV0(task); err != nil {
		return err
	}
	return validateWorkflowFunctionContractRefsV0(task.FunctionContractRefs)
}

func validateRequiredWorkflowTaskStringsV0(values []string, field string) error {
	if len(values) == 0 || len(values) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, field)
	}
	return validateWorkflowTaskStringsContentV0(values, field)
}

func validateOptionalWorkflowTaskStringsV0(values []string, field string) error {
	if len(values) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, field)
	}
	return validateWorkflowTaskStringsContentV0(values, field)
}

func validateWorkflowTaskStringsContentV0(values []string, field string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" || len(value) > maxWorkflowTaskStringV0 {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, field)
		}
	}
	return nil
}

func validateWorkflowTaskDependsOnV0(task WorkflowTaskV0) error {
	if len(task.DependsOn) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "depends_on")
	}
	seen := map[string]bool{}
	for _, dep := range task.DependsOn {
		dep = strings.TrimSpace(dep)
		if dep == "" ||
			dep == strings.TrimSpace(task.TaskID) ||
			seen[dep] ||
			workflowTaskStringHasForbiddenDetailV0(dep) {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "depends_on")
		}
		seen[dep] = true
	}
	return nil
}

func validateWorkflowTaskContextRefsV0(refs []string) error {
	if len(refs) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "context_refs")
	}
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" || !workflowTaskContextRefIsCompactV0(ref) {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "context_refs")
		}
	}
	return nil
}

func workflowTaskContextRefIsCompactV0(value string) bool {
	if len(value) > maxWorkflowTaskStringV0 {
		return false
	}
	return !strings.ContainsAny(value, " /\\\t\r\n")
}

func validateWorkflowTaskLineageV0(task WorkflowTaskV0) error {
	taskID := strings.TrimSpace(task.TaskID)
	if task.DelegationDepth < 0 || task.DelegationDepth > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "delegation_depth")
	}
	if task.MaxDelegationDepth < 0 || task.MaxDelegationDepth > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "max_delegation_depth")
	}
	if task.MaxChildAgents < 0 || task.MaxChildAgents > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "max_child_agents")
	}
	if task.MaxSubagentsPerAgent < 0 || task.MaxSubagentsPerAgent > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "max_subagents_per_agent")
	}
	if task.MaxRecursiveAgents < 0 || task.MaxRecursiveAgents > maxWorkflowTaskRecursiveAgentsV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "max_recursive_agents")
	}
	if err := validateWorkflowTaskOptionalRefV0(task.ParentTaskRef, "parent_task_ref"); err != nil {
		return err
	}
	if strings.TrimSpace(task.ParentTaskRef) != "" && strings.TrimSpace(task.ParentTaskRef) == taskID {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "parent_task_ref")
	}
	if err := validateWorkflowTaskOptionalRefV0(task.CohortRef, "cohort_ref"); err != nil {
		return err
	}
	if err := validateWorkflowTaskOptionalRefV0(task.WaveRef, "wave_ref"); err != nil {
		return err
	}
	if len(task.ChildTaskRefs) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "child_task_refs")
	}
	seen := map[string]bool{}
	for _, child := range task.ChildTaskRefs {
		child = strings.TrimSpace(child)
		if child == "" || child == taskID || child == strings.TrimSpace(task.ParentTaskRef) ||
			seen[child] || !workflowTaskOptionalRefIsCompactV0(child) ||
			workflowTaskStringHasForbiddenDetailV0(child) {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "child_task_refs")
		}
		seen[child] = true
	}
	return nil
}

func validateWorkflowTaskOptionalRefV0(value string, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if !workflowTaskOptionalRefIsCompactV0(value) || workflowTaskStringHasForbiddenDetailV0(value) {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, field)
	}
	return nil
}

func workflowTaskOptionalRefIsCompactV0(value string) bool {
	if len(value) > maxWorkflowTaskStringV0 {
		return false
	}
	return !strings.ContainsAny(value, " \t\r\n")
}

func validateWorkflowTaskDependencyRefsExistV0(current OrchestrationRunV0, task WorkflowTaskV0) error {
	for _, dep := range task.DependsOn {
		if !microtaskAlreadyReflectedV0(current, dep) {
			return commandErrorV0(ErrTransicionInvalidaV0, "payload.task.depends_on")
		}
	}
	return nil
}

func validateWorkflowTaskWriteSetV0(paths []string) error {
	if err := validateRequiredWorkflowTaskStringsV0(paths, "write_set"); err != nil {
		return err
	}
	for _, repoPath := range paths {
		if !workflowTaskWriteSetPathIsSafeV0(repoPath) {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "write_set")
		}
		if workflowTaskWriteSetPathHasForbiddenDetailV0(repoPath) {
			return workflowTaskErrorV0(ErrDetalleProhibidoV0, "write_set")
		}
	}
	return nil
}

func workflowTaskWriteSetPathIsSafeV0(repoPath string) bool {
	if strings.Contains(repoPath, "\x00") || strings.Contains(repoPath, "\\") {
		return false
	}
	if strings.HasPrefix(repoPath, "/") || strings.HasPrefix(repoPath, "~") {
		return false
	}
	if strings.Contains(repoPath, "://") || strings.Contains(repoPath, ":") || strings.Contains(repoPath, "$") {
		return false
	}
	for _, segment := range strings.Split(repoPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func workflowTaskWriteSetPathHasForbiddenDetailV0(repoPath string) bool {
	return textContainsForbiddenOperationalSensitiveDetailV0(repoPath)
}

func validateWorkflowFunctionContractRefsV0(refs []WorkflowFunctionContractRefV0) error {
	if len(refs) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "function_contract_refs")
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref.ContractRef) == "" && strings.TrimSpace(ref.FunctionName) == "" {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "function_contract_refs")
		}
	}
	return nil
}

func validateWorkflowTaskForbiddenDetailsV0(task WorkflowTaskV0) error {
	for field, values := range workflowTaskTextFieldsV0(task) {
		if workflowTaskStringsHaveForbiddenDetailV0(values) {
			return workflowTaskErrorV0(ErrDetalleProhibidoV0, field)
		}
	}
	return nil
}

func validateWorkflowTaskCompactPayloadV0(task WorkflowTaskV0) error {
	data, err := json.Marshal(task)
	if err != nil {
		return workflowTaskErrorV0(ErrWorkflowTaskPayloadInvalidoV0, "payload")
	}
	if len(data) == 0 || len(data) > maxWorkflowTaskPayloadBytesV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskPayloadInvalidoV0, "payload")
	}
	return nil
}

func workflowTaskTextFieldsV0(task WorkflowTaskV0) map[string][]string {
	return map[string][]string{
		"task_id":                []string{task.TaskID},
		"run_id":                 []string{task.RunID},
		"title":                  []string{task.Title},
		"summary":                []string{task.Summary},
		"acceptance_criteria":    task.AcceptanceCriteria,
		"depends_on":             task.DependsOn,
		"context_refs":           task.ContextRefs,
		"parent_task_ref":        []string{task.ParentTaskRef},
		"cohort_ref":             []string{task.CohortRef},
		"wave_ref":               []string{task.WaveRef},
		"child_task_refs":        task.ChildTaskRefs,
		"function_contract_refs": workflowFunctionContractRefStringsV0(task.FunctionContractRefs),
	}
}

func workflowFunctionContractRefStringsV0(refs []WorkflowFunctionContractRefV0) []string {
	values := make([]string, 0, len(refs)*2)
	for _, ref := range refs {
		values = append(values, ref.ContractRef, ref.FunctionName)
	}
	return values
}

func workflowTaskStringsHaveForbiddenDetailV0(values []string) bool {
	for _, value := range values {
		if workflowTaskStringHasForbiddenDetailV0(value) {
			return true
		}
	}
	return false
}

func workflowTaskStringHasForbiddenDetailV0(value string) bool {
	return textContainsForbiddenOperationalSensitiveDetailV0(value)
}

func workflowTaskErrorV0(code string, field string) WorkflowTaskErrorV0 {
	return WorkflowTaskErrorV0{Code: code, Field: field}
}
