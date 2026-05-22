package orquestastatefile

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type workflowTaskDocumentV0 struct {
	SchemaVersion string                              `json:"schema_version"`
	RunRef        string                              `json:"run_ref"`
	TaskRef       string                              `json:"task_ref"`
	Task          orquestacoreworkflow.WorkflowTaskV0 `json:"task"`
}

func (store *StoreV0) SaveWorkflowTaskV0(
	ctx context.Context,
	task orquestacoreworkflow.WorkflowTaskV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return invalidErrorV0("workflow_task", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveWorkflowTaskLockedV0(normalized)
}

func (store *StoreV0) LoadWorkflowTasksV0(
	ctx context.Context,
	runRef string,
	taskRefs []string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = normalizeRefV0(runRef)
	if runRef == "" {
		return nil, invalidErrorV0("run_ref", "run_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadWorkflowTasksLockedV0(runRef, compactStringsV0(taskRefs))
}

func (store *StoreV0) LoadWorkflowTasksByParentV0(
	ctx context.Context,
	runRef string,
	parentTaskRef string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	runRef = normalizeRefV0(runRef)
	parentTaskRef = normalizeRefV0(parentTaskRef)
	if runRef == "" {
		return nil, invalidErrorV0("run_ref", "run_ref requerido")
	}
	if parentTaskRef == "" {
		return nil, invalidErrorV0("parent_task_ref", "parent_task_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.loadWorkflowTasksByParentLockedV0(runRef, parentTaskRef)
}

func (store *StoreV0) saveWorkflowTaskLockedV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) error {
	runRef := normalizeRefV0(task.RunID)
	taskRef := normalizeRefV0(task.TaskID)
	path := store.workflowTaskPathV0(runRef, taskRef)
	existing, ok, err := readJSONFileV0[workflowTaskDocumentV0](path)
	if err != nil {
		return err
	}
	if ok {
		existingTask, err := validateWorkflowTaskDocumentV0(existing, runRef, taskRef)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(existingTask, task) {
			return storeErrorV0("workflow_task", "microtarea existente con contrato distinto")
		}
		return nil
	}
	return writeJSONAtomicV0(path, workflowTaskDocumentV0{
		SchemaVersion: workflowTaskDocumentSchemaV0,
		RunRef:        runRef,
		TaskRef:       taskRef,
		Task:          task,
	})
}

func (store *StoreV0) loadWorkflowTasksLockedV0(
	runRef string,
	taskRefs []string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(taskRefs))
	for _, taskRef := range taskRefs {
		document, ok, err := readJSONFileV0[workflowTaskDocumentV0](store.workflowTaskPathV0(runRef, taskRef))
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, storeErrorV0("workflow_tasks", fmt.Sprintf("microtarea no encontrada: %s", taskRef))
		}
		task, err := validateWorkflowTaskDocumentV0(document, runRef, taskRef)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, nil
}

func (store *StoreV0) loadWorkflowTasksByParentLockedV0(
	runRef string,
	parentTaskRef string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	dir := filepath.Join(store.rootDir, workflowTasksDirV0, hashRefsV0(runRef))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		document, ok, err := readJSONFileV0[workflowTaskDocumentV0](filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		taskRef := normalizeRefV0(document.TaskRef)
		task, err := validateWorkflowTaskDocumentV0(document, runRef, taskRef)
		if err != nil {
			return nil, err
		}
		if normalizeRefV0(task.ParentTaskRef) == parentTaskRef {
			out = append(out, task)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TaskID < out[j].TaskID
	})
	return out, nil
}

func validateWorkflowTaskDocumentV0(
	document workflowTaskDocumentV0,
	expectedRunRef string,
	expectedTaskRef string,
) (orquestacoreworkflow.WorkflowTaskV0, error) {
	if document.SchemaVersion != workflowTaskDocumentSchemaV0 {
		return orquestacoreworkflow.WorkflowTaskV0{}, storeErrorV0("workflow_task.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef || document.TaskRef != expectedTaskRef {
		return orquestacoreworkflow.WorkflowTaskV0{}, storeErrorV0("workflow_task.ref", "ref inconsistente")
	}
	task, err := orquestacoreworkflow.NewWorkflowTaskV0(document.Task)
	if err != nil {
		return orquestacoreworkflow.WorkflowTaskV0{}, err
	}
	if task.RunID != expectedRunRef || task.TaskID != expectedTaskRef {
		return orquestacoreworkflow.WorkflowTaskV0{}, storeErrorV0("workflow_task.state_ref", "ref interna inconsistente")
	}
	return task, nil
}
