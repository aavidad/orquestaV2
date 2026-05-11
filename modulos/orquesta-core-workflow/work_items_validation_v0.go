package orquestacoreworkflow

import (
	"encoding/json"
	"strings"
)

var forbiddenWorkflowTaskWriteSetFragmentsV0 = []string{
	"secret",
	"secreto",
	"token",
	"password",
	"credential",
	"credencial",
	"api_key",
}

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
	if err := validateRequiredWorkflowTaskStringsV0(task.AcceptanceCriteria, "acceptance_criteria"); err != nil {
		return err
	}
	if err := validateOptionalWorkflowTaskStringsV0(task.RequiredTests, "required_tests"); err != nil {
		return err
	}
	if err := validateWorkflowTaskDependsOnV0(task); err != nil {
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
	lower := strings.ToLower(repoPath)
	for _, fragment := range forbiddenWorkflowTaskWriteSetFragmentsV0 {
		if containsForbiddenFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
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
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenWorkflowTaskFragmentsV0 {
		if containsForbiddenFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}

func workflowTaskErrorV0(code string, field string) WorkflowTaskErrorV0 {
	return WorkflowTaskErrorV0{Code: code, Field: field}
}
