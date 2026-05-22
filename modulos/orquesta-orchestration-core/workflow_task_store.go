package orquestacionnucleoapp

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type WorkflowTaskStorePortV0 interface {
	LoadWorkflowTasksV0(
		ctx context.Context,
		runRef string,
		taskRefs []string,
	) ([]orquestacoreworkflow.WorkflowTaskV0, error)
}

type WorkflowTaskByParentStorePortV0 interface {
	LoadWorkflowTasksByParentV0(
		ctx context.Context,
		runRef string,
		parentTaskRef string,
	) ([]orquestacoreworkflow.WorkflowTaskV0, error)
}

type WorkflowTaskWriterPortV0 interface {
	SaveWorkflowTaskV0(ctx context.Context, task orquestacoreworkflow.WorkflowTaskV0) error
}

type InMemoryWorkflowTaskStoreV0 struct {
	mu    sync.Mutex
	tasks map[string]map[string]orquestacoreworkflow.WorkflowTaskV0
	err   error
}

var _ WorkflowTaskStorePortV0 = (*InMemoryWorkflowTaskStoreV0)(nil)
var _ WorkflowTaskByParentStorePortV0 = (*InMemoryWorkflowTaskStoreV0)(nil)
var _ WorkflowTaskWriterPortV0 = (*InMemoryWorkflowTaskStoreV0)(nil)

func NewInMemoryWorkflowTaskStoreV0(
	tasks ...orquestacoreworkflow.WorkflowTaskV0,
) *InMemoryWorkflowTaskStoreV0 {
	store := &InMemoryWorkflowTaskStoreV0{
		tasks: map[string]map[string]orquestacoreworkflow.WorkflowTaskV0{},
	}
	for _, task := range tasks {
		if err := store.saveNormalizedWorkflowTaskV0(task); err != nil && store.err == nil {
			store.err = err
		}
	}
	return store
}

func (store *InMemoryWorkflowTaskStoreV0) SaveWorkflowTaskV0(
	ctx context.Context,
	task orquestacoreworkflow.WorkflowTaskV0,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveNormalizedWorkflowTaskV0(task)
}

func (store *InMemoryWorkflowTaskStoreV0) LoadWorkflowTasksV0(
	ctx context.Context,
	runRef string,
	taskRefs []string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.err != nil {
		return nil, store.err
	}
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return nil, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	tasksByRun := store.tasks[runRef]
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0, len(taskRefs))
	for _, taskRef := range compactStringsV0(taskRefs) {
		task, ok := tasksByRun[taskRef]
		if !ok {
			return nil, errorV0(
				ErrNucleoOrquestacionStoreV0,
				"workflow_tasks",
				fmt.Sprintf("microtarea no encontrada: %s", taskRef),
			)
		}
		out = append(out, task)
	}
	return out, nil
}

func (store *InMemoryWorkflowTaskStoreV0) LoadWorkflowTasksByParentV0(
	ctx context.Context,
	runRef string,
	parentTaskRef string,
) ([]orquestacoreworkflow.WorkflowTaskV0, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.err != nil {
		return nil, store.err
	}
	runRef = strings.TrimSpace(runRef)
	parentTaskRef = strings.TrimSpace(parentTaskRef)
	if runRef == "" {
		return nil, errorV0(ErrNucleoOrquestacionInvalidoV0, "run_ref", "run_ref requerido")
	}
	if parentTaskRef == "" {
		return nil, errorV0(ErrNucleoOrquestacionInvalidoV0, "parent_task_ref", "parent_task_ref requerido")
	}
	tasksByRun := store.tasks[runRef]
	out := make([]orquestacoreworkflow.WorkflowTaskV0, 0)
	for _, task := range tasksByRun {
		if strings.TrimSpace(task.ParentTaskRef) == parentTaskRef {
			out = append(out, task)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TaskID < out[j].TaskID
	})
	return out, nil
}

func (store *InMemoryWorkflowTaskStoreV0) saveNormalizedWorkflowTaskV0(
	task orquestacoreworkflow.WorkflowTaskV0,
) error {
	normalized, err := orquestacoreworkflow.NewWorkflowTaskV0(task)
	if err != nil {
		return errorV0(ErrNucleoOrquestacionInvalidoV0, "workflow_task", err.Error())
	}
	runRef := strings.TrimSpace(normalized.RunID)
	if store.tasks[runRef] == nil {
		store.tasks[runRef] = map[string]orquestacoreworkflow.WorkflowTaskV0{}
	}
	taskRef := strings.TrimSpace(normalized.TaskID)
	if existing, ok := store.tasks[runRef][taskRef]; ok {
		if !reflect.DeepEqual(existing, normalized) {
			return errorV0(ErrNucleoOrquestacionStoreV0, "workflow_task", "microtarea existente con contrato distinto")
		}
		return nil
	}
	store.tasks[runRef][taskRef] = normalized
	return nil
}
