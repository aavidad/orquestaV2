package orquestastatefile

import "encoding/json"

const (
	defaultWorkflowTaskParentIndexEntriesV0        = 20000
	defaultWorkflowTaskParentIndexBytesV0          = stateFileJSONMaxBytesV0
	defaultWorkflowTaskParentIndexRebuildEntriesV0 = 20000
	defaultWorkflowTaskParentIndexRebuildBytesV0   = stateFileJSONMaxBytesV0
)

type workflowTaskStoreBudgetV0 struct {
	MaxParentIndexEntries        int
	MaxParentIndexBytes          int64
	MaxParentIndexRebuildEntries int
	MaxParentIndexRebuildBytes   int64
}

func normalizeWorkflowTaskStoreBudgetV0(config ConfigV0) workflowTaskStoreBudgetV0 {
	budget := workflowTaskStoreBudgetV0{
		MaxParentIndexEntries:        config.MaxWorkflowTaskParentIndexEntries,
		MaxParentIndexBytes:          config.MaxWorkflowTaskParentIndexBytes,
		MaxParentIndexRebuildEntries: config.MaxWorkflowTaskParentIndexRebuildEntries,
		MaxParentIndexRebuildBytes:   config.MaxWorkflowTaskParentIndexRebuildBytes,
	}
	if budget.MaxParentIndexEntries <= 0 {
		budget.MaxParentIndexEntries = defaultWorkflowTaskParentIndexEntriesV0
	}
	if budget.MaxParentIndexBytes <= 0 {
		budget.MaxParentIndexBytes = defaultWorkflowTaskParentIndexBytesV0
	}
	if budget.MaxParentIndexRebuildEntries <= 0 {
		budget.MaxParentIndexRebuildEntries = defaultWorkflowTaskParentIndexRebuildEntriesV0
	}
	if budget.MaxParentIndexRebuildBytes <= 0 {
		budget.MaxParentIndexRebuildBytes = defaultWorkflowTaskParentIndexRebuildBytesV0
	}
	return budget
}

func (budget workflowTaskStoreBudgetV0) validateParentIndexV0(index workflowTaskParentIndexDocumentV0) error {
	if len(index.Tasks) > budget.MaxParentIndexEntries {
		return storeErrorV0("workflow_task_parent_index.budget", "workflow_task_parent_index_budget_exhausted")
	}
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	if int64(len(data)) > budget.MaxParentIndexBytes {
		return storeErrorV0("workflow_task_parent_index.budget", "workflow_task_parent_index_budget_exhausted")
	}
	return nil
}
