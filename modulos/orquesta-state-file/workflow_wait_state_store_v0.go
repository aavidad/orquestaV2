package orquestastatefile

import (
	"context"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type workflowWaitStateDocumentV0 struct {
	SchemaVersion string                                        `json:"schema_version"`
	RunRef        string                                        `json:"run_ref"`
	WaitRef       string                                        `json:"wait_ref"`
	State         orquestacionnucleoapp.WorkflowTaskWaitStateV0 `json:"state"`
}

func (store *StoreV0) SaveWorkflowTaskWaitStateV0(
	ctx context.Context,
	state orquestacionnucleoapp.WorkflowTaskWaitStateV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(state)
	if err != nil {
		return invalidErrorV0("workflow_task_wait_state", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.saveWorkflowTaskWaitStateLockedV0(normalized)
}

func (store *StoreV0) LoadWorkflowTaskWaitStateV0(
	ctx context.Context,
	runRef string,
	waitRef string,
) (orquestacionnucleoapp.WorkflowTaskWaitStateV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, err
	}
	runRef = normalizeRefV0(runRef)
	waitRef = normalizeRefV0(waitRef)
	if runRef == "" {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, invalidErrorV0("run_ref", "run_ref requerido")
	}
	if waitRef == "" {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, invalidErrorV0("wait_ref", "wait_ref requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[workflowWaitStateDocumentV0](store.workflowWaitPathV0(runRef, waitRef))
	if err != nil {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, err
	}
	if !ok {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, storeErrorV0("workflow_task_wait_state", "espera no encontrada")
	}
	state, err := validateWorkflowWaitStateDocumentV0(document, runRef, waitRef)
	if err != nil {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, err
	}
	return state, nil
}

func (store *StoreV0) saveWorkflowTaskWaitStateLockedV0(
	state orquestacionnucleoapp.WorkflowTaskWaitStateV0,
) error {
	runRef := normalizeRefV0(state.RunRef)
	waitRef := normalizeRefV0(state.WaitRef)
	return writeJSONAtomicV0(store.workflowWaitPathV0(runRef, waitRef), workflowWaitStateDocumentV0{
		SchemaVersion: workflowWaitDocumentSchemaV0,
		RunRef:        runRef,
		WaitRef:       waitRef,
		State:         state,
	})
}

func validateWorkflowWaitStateDocumentV0(
	document workflowWaitStateDocumentV0,
	expectedRunRef string,
	expectedWaitRef string,
) (orquestacionnucleoapp.WorkflowTaskWaitStateV0, error) {
	if document.SchemaVersion != workflowWaitDocumentSchemaV0 {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, storeErrorV0("workflow_task_wait_state.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef || document.WaitRef != expectedWaitRef {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, storeErrorV0("workflow_task_wait_state.ref", "ref inconsistente")
	}
	state, err := orquestacionnucleoapp.NewWorkflowTaskWaitStateV0(document.State)
	if err != nil {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, err
	}
	if state.RunRef != expectedRunRef || state.WaitRef != expectedWaitRef {
		return orquestacionnucleoapp.WorkflowTaskWaitStateV0{}, storeErrorV0("workflow_task_wait_state.state_ref", "ref interna inconsistente")
	}
	return state, nil
}
