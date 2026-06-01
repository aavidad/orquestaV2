package orquestacoreworkflow

import "strings"

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
	if err := validateWorkflowTaskParentRefV0(task, taskID); err != nil {
		return err
	}
	if err := validateWorkflowTaskOptionalRefV0(task.CohortRef, "cohort_ref"); err != nil {
		return err
	}
	if err := validateWorkflowTaskOptionalRefV0(task.WaveRef, "wave_ref"); err != nil {
		return err
	}
	return validateWorkflowTaskChildRefsV0(task, taskID)
}

func validateWorkflowTaskParentRefV0(task WorkflowTaskV0, taskID string) error {
	if err := validateWorkflowTaskOptionalRefV0(task.ParentTaskRef, "parent_task_ref"); err != nil {
		return err
	}
	if strings.TrimSpace(task.ParentTaskRef) != "" && strings.TrimSpace(task.ParentTaskRef) == taskID {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "parent_task_ref")
	}
	return nil
}

func validateWorkflowTaskChildRefsV0(task WorkflowTaskV0, taskID string) error {
	if len(task.ChildTaskRefs) > maxWorkflowTaskCollectionV0 {
		return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "child_task_refs")
	}
	seen := map[string]bool{}
	for _, child := range task.ChildTaskRefs {
		child = strings.TrimSpace(child)
		if workflowTaskChildRefInvalidV0(child, task, taskID, seen) {
			return workflowTaskErrorV0(ErrWorkflowTaskInvalidaV0, "child_task_refs")
		}
		seen[child] = true
	}
	return nil
}

func workflowTaskChildRefInvalidV0(child string, task WorkflowTaskV0, taskID string, seen map[string]bool) bool {
	return child == "" ||
		child == taskID ||
		child == strings.TrimSpace(task.ParentTaskRef) ||
		seen[child] ||
		!workflowTaskOptionalRefIsCompactV0(child) ||
		workflowTaskStringHasForbiddenDetailV0(child)
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
