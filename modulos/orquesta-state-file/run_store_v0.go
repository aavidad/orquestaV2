package orquestastatefile

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type runDocumentV0 struct {
	SchemaVersion string                                  `json:"schema_version"`
	RunRef        string                                  `json:"run_ref"`
	Run           orquestacoreworkflow.OrchestrationRunV0 `json:"run"`
}

func (store *StoreV0) LoadRunV0(
	ctx context.Context,
	runRef string,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	runRef = normalizeRefV0(runRef)
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[runDocumentV0](store.runPathV0(runRef))
	if err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if !ok {
		return orquestacoreworkflow.OrchestrationRunV0{},
			orquestacionnucleoapp.RunNotFoundErrorV0{RunRef: runRef}
	}
	if err := validateRunDocumentV0(document, runRef); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	if err := validateLoadedRunProjectionV0(document.Run); err != nil {
		return orquestacoreworkflow.OrchestrationRunV0{}, err
	}
	return document.Run, nil
}

func (store *StoreV0) SaveRunV0(
	ctx context.Context,
	run orquestacoreworkflow.OrchestrationRunV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	runRef := normalizeRefV0(run.RunID)
	if runRef == "" {
		return invalidErrorV0("run_id", "run_id requerido")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return writeJSONAtomicV0(store.runPathV0(runRef), runDocumentV0{
		SchemaVersion: runDocumentSchemaV0,
		RunRef:        runRef,
		Run:           run,
	})
}

func validateRunDocumentV0(document runDocumentV0, expectedRunRef string) error {
	if document.SchemaVersion != runDocumentSchemaV0 {
		return storeErrorV0("run.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef || normalizeRefV0(document.Run.RunID) != expectedRunRef {
		return storeErrorV0("run.ref", "ref inconsistente")
	}
	return nil
}
