package orquestastatefile

import (
	"io"
	"os"
	"path/filepath"
	"sort"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const workflowTaskParentIndexRebuildRequiredV0 = "workflow_task_parent_index_rebuild_required"

type workflowTaskParentIndexDocumentV0 struct {
	SchemaVersion     string                                    `json:"schema_version"`
	RunRef            string                                    `json:"run_ref"`
	LastRebuildReason string                                    `json:"last_rebuild_reason,omitempty"`
	Parents           map[string][]string                       `json:"parents,omitempty"`
	Tasks             map[string]workflowTaskParentIndexEntryV0 `json:"tasks,omitempty"`
}

type workflowTaskParentIndexEntryV0 struct {
	RunRef          string   `json:"run_ref"`
	TaskRef         string   `json:"task_ref"`
	ParentTaskRef   string   `json:"parent_task_ref,omitempty"`
	WaveRef         string   `json:"wave_ref,omitempty"`
	CohortRef       string   `json:"cohort_ref,omitempty"`
	DelegationDepth int      `json:"delegation_depth,omitempty"`
	MaxChildAgents  int      `json:"max_child_agents,omitempty"`
	ChildTaskRefs   []string `json:"child_task_refs,omitempty"`
}

func (store *StoreV0) loadWorkflowTaskParentIndexV0(runRef string) (workflowTaskParentIndexDocumentV0, error) {
	index, ok, err := readJSONFileV0[workflowTaskParentIndexDocumentV0](store.workflowTaskParentIndexPathV0(runRef))
	if err == nil && ok {
		if valid, err := store.validateWorkflowTaskParentIndexDocumentV0(index, runRef); err == nil {
			return valid, nil
		}
	}
	return store.rebuildWorkflowTaskParentIndexV0(runRef, workflowTaskParentIndexRebuildRequiredV0)
}

func (store *StoreV0) rebuildWorkflowTaskParentIndexV0(
	runRef string,
	reason string,
) (workflowTaskParentIndexDocumentV0, error) {
	index := newWorkflowTaskParentIndexDocumentV0(runRef)
	index.LastRebuildReason = reason
	tasks, err := store.loadWorkflowTasksForParentIndexRebuildV0(runRef)
	if err != nil {
		return workflowTaskParentIndexDocumentV0{}, err
	}
	for _, task := range tasks {
		index = workflowTaskParentIndexWithTaskV0(index, task)
	}
	if err := store.workflowTasks.validateParentIndexV0(index); err != nil {
		return workflowTaskParentIndexDocumentV0{}, err
	}
	if err := writeJSONAtomicV0(store.workflowTaskParentIndexPathV0(runRef), index); err != nil {
		return workflowTaskParentIndexDocumentV0{}, err
	}
	return index, nil
}

func (store *StoreV0) loadWorkflowTasksForParentIndexRebuildV0(
	runRef string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	dir := filepath.Join(store.rootDir, workflowTasksDirV0, hashRefsV0(runRef))
	handle, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer handle.Close()
	var tasks []orquestacoreworkflow.WorkflowTaskV0
	var totalBytes int64
	for {
		entries, err := handle.ReadDir(64)
		if err != nil && err != io.EOF {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if len(tasks)+1 > store.workflowTasks.MaxParentIndexRebuildEntries {
				return nil, storeErrorV0("workflow_task_parent_index.budget", "workflow_task_parent_index_budget_exhausted")
			}
			info, err := entry.Info()
			if err != nil {
				return nil, err
			}
			totalBytes += info.Size()
			if totalBytes > store.workflowTasks.MaxParentIndexRebuildBytes {
				return nil, storeErrorV0("workflow_task_parent_index.budget", "workflow_task_parent_index_budget_exhausted")
			}
			task, err := loadWorkflowTaskDocumentForParentIndexV0(filepath.Join(dir, entry.Name()), runRef)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, task)
		}
		if err == io.EOF {
			break
		}
	}
	return tasks, nil
}

func loadWorkflowTaskDocumentForParentIndexV0(
	path string,
	runRef string,
) (orquestacoreworkflow.WorkflowTaskV0, error) {
	document, ok, err := readJSONFileV0[workflowTaskDocumentV0](path)
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, err
	}
	if !ok {
		return orquestacoreworkflow.WorkflowTaskV0{}, storeErrorV0("workflow_task_parent_index.rebuild", "workflow_task_missing")
	}
	return validateWorkflowTaskDocumentV0(document, runRef, normalizeRefV0(document.TaskRef))
}

func newWorkflowTaskParentIndexDocumentV0(runRef string) workflowTaskParentIndexDocumentV0 {
	return workflowTaskParentIndexDocumentV0{
		SchemaVersion: workflowTaskParentIndexSchemaV0,
		RunRef:        runRef,
		Parents:       map[string][]string{},
		Tasks:         map[string]workflowTaskParentIndexEntryV0{},
	}
}

func workflowTaskParentIndexWithTaskV0(
	index workflowTaskParentIndexDocumentV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) workflowTaskParentIndexDocumentV0 {
	index = cloneWorkflowTaskParentIndexDocumentV0(index)
	taskRef := normalizeRefV0(task.TaskID)
	parentRef := normalizeRefV0(task.ParentTaskRef)
	if existing, ok := index.Tasks[taskRef]; ok {
		index.Parents[existing.ParentTaskRef] = removeStringV0(index.Parents[existing.ParentTaskRef], taskRef)
	}
	index.Tasks[taskRef] = workflowTaskParentIndexEntryFromTaskV0(task)
	if parentRef != "" {
		index.Parents[parentRef] = compactStringsV0(append(index.Parents[parentRef], taskRef))
		sort.Strings(index.Parents[parentRef])
	}
	return index
}

func workflowTaskParentIndexEntryFromTaskV0(task orquestacoreworkflow.WorkflowTaskV0) workflowTaskParentIndexEntryV0 {
	return workflowTaskParentIndexEntryV0{
		RunRef:          normalizeRefV0(task.RunID),
		TaskRef:         normalizeRefV0(task.TaskID),
		ParentTaskRef:   normalizeRefV0(task.ParentTaskRef),
		WaveRef:         normalizeRefV0(task.WaveRef),
		CohortRef:       normalizeRefV0(task.CohortRef),
		DelegationDepth: task.DelegationDepth,
		MaxChildAgents:  task.MaxChildAgents,
		ChildTaskRefs:   compactStringsV0(task.ChildTaskRefs),
	}
}

func (store *StoreV0) validateWorkflowTaskParentIndexDocumentV0(
	index workflowTaskParentIndexDocumentV0,
	runRef string,
) (workflowTaskParentIndexDocumentV0, error) {
	if index.SchemaVersion != workflowTaskParentIndexSchemaV0 {
		return workflowTaskParentIndexDocumentV0{}, storeErrorV0("workflow_task_parent_index.schema_version", "schema_version invalida")
	}
	if index.RunRef != runRef {
		return workflowTaskParentIndexDocumentV0{}, storeErrorV0("workflow_task_parent_index.ref", "ref inconsistente")
	}
	if index.Parents == nil {
		index.Parents = map[string][]string{}
	}
	if index.Tasks == nil {
		index.Tasks = map[string]workflowTaskParentIndexEntryV0{}
	}
	rebuiltParents := map[string][]string{}
	for taskRef, entry := range index.Tasks {
		if entry.RunRef != runRef || entry.TaskRef != taskRef {
			return workflowTaskParentIndexDocumentV0{}, storeErrorV0("workflow_task_parent_index.ref", "ref inconsistente")
		}
		if entry.ParentTaskRef != "" {
			rebuiltParents[entry.ParentTaskRef] = append(rebuiltParents[entry.ParentTaskRef], taskRef)
		}
	}
	index.Parents = normalizeWorkflowTaskParentMapV0(rebuiltParents)
	if err := store.workflowTasks.validateParentIndexV0(index); err != nil {
		return workflowTaskParentIndexDocumentV0{}, err
	}
	return index, nil
}

func normalizeWorkflowTaskParentMapV0(parents map[string][]string) map[string][]string {
	out := make(map[string][]string, len(parents))
	for parentRef, taskRefs := range parents {
		parentRef = normalizeRefV0(parentRef)
		if parentRef == "" {
			continue
		}
		out[parentRef] = compactStringsV0(taskRefs)
		sort.Strings(out[parentRef])
	}
	return out
}

func cloneWorkflowTaskParentIndexDocumentV0(
	index workflowTaskParentIndexDocumentV0,
) workflowTaskParentIndexDocumentV0 {
	out := index
	out.Parents = make(map[string][]string, len(index.Parents))
	for parentRef, taskRefs := range index.Parents {
		out.Parents[parentRef] = append([]string(nil), taskRefs...)
	}
	out.Tasks = make(map[string]workflowTaskParentIndexEntryV0, len(index.Tasks))
	for taskRef, entry := range index.Tasks {
		entry.ChildTaskRefs = append([]string(nil), entry.ChildTaskRefs...)
		out.Tasks[taskRef] = entry
	}
	return out
}

func removeStringV0(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalizeRefV0(value) != target {
			out = append(out, value)
		}
	}
	return compactStringsV0(out)
}
