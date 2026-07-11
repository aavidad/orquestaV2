package orquestastatefile

import (
	"context"
	"reflect"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const autoprogrammingBatchDocumentSchemaV0 = "orquesta_state_file.autoprogramming_batch.v0"

type autoprogrammingBatchDocumentV0 struct {
	SchemaVersion string                                         `json:"schema_version"`
	BatchRef      string                                         `json:"batch_ref"`
	Batch         orquestaautoprogramming.AutoprogrammingBatchV0 `json:"batch"`
}

func (store *StoreV0) LoadAutoprogrammingBatchV0(
	ctx context.Context,
	batchRef string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	batchRef = normalizeRefV0(batchRef)
	if batchRef == "" {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch.batch_ref", "batch_ref requerido")
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.autoprogrammingBatchPathV0(batchRef)
	var document autoprogrammingBatchDocumentV0
	var found bool
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		var readErr error
		document, found, readErr = readJSONFileV0[autoprogrammingBatchDocumentV0](path)
		return readErr
	})
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	if !found {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, storeErrorV0("autoprogramming_batch", "batch no encontrado")
	}
	return validateAutoprogrammingBatchDocumentV0(document, batchRef)
}

func (store *StoreV0) CompareAndSwapAutoprogrammingBatchV0(
	ctx context.Context,
	expectedVersion uint64,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	batch = orquestaautoprogramming.NormalizeAutoprogrammingBatchV0(batch)
	if batch.BatchRef == "" {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch.batch_ref", "batch_ref requerido")
	}
	batch.StoreVersion = expectedVersion + 1
	validation := orquestaautoprogramming.ValidateAutoprogrammingBatchV0(batch)
	if !validation.Accepted {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, invalidErrorV0("autoprogramming_batch", "batch invalido")
	}
	batch = validation.Batch

	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.autoprogrammingBatchPathV0(batch.BatchRef)
	var saved orquestaautoprogramming.AutoprogrammingBatchV0
	err := withProcessFileLockV0(ctx, path+".lock", func() error {
		document, found, readErr := readJSONFileV0[autoprogrammingBatchDocumentV0](path)
		if readErr != nil {
			return readErr
		}
		if !found {
			if expectedVersion != 0 {
				return autoprogrammingBatchCASConflictV0(batch, expectedVersion, 0)
			}
			saved = batch
			return store.writeAutoprogrammingBatchV0(path, saved)
		}

		existing, validateErr := validateAutoprogrammingBatchDocumentV0(document, batch.BatchRef)
		if validateErr != nil {
			return validateErr
		}
		if reflect.DeepEqual(existing, batch) {
			saved = existing
			return nil
		}
		if existing.PlanHash != batch.PlanHash || existing.StoreVersion != expectedVersion {
			return autoprogrammingBatchCASConflictV0(batch, expectedVersion, existing.StoreVersion)
		}
		saved = batch
		return store.writeAutoprogrammingBatchV0(path, saved)
	})
	if err != nil {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, err
	}
	return saved, nil
}

func (store *StoreV0) writeAutoprogrammingBatchV0(
	path string,
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
) error {
	return writeJSONAtomicV0(path, autoprogrammingBatchDocumentV0{
		SchemaVersion: autoprogrammingBatchDocumentSchemaV0,
		BatchRef:      batch.BatchRef,
		Batch:         batch,
	})
}

func validateAutoprogrammingBatchDocumentV0(
	document autoprogrammingBatchDocumentV0,
	batchRef string,
) (orquestaautoprogramming.AutoprogrammingBatchV0, error) {
	if document.SchemaVersion != autoprogrammingBatchDocumentSchemaV0 || normalizeRefV0(document.BatchRef) != batchRef {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, storeErrorV0("autoprogramming_batch", "documento persistido inconsistente")
	}
	validation := orquestaautoprogramming.ValidateAutoprogrammingBatchV0(document.Batch)
	if !validation.Accepted || validation.Batch.BatchRef != batchRef {
		return orquestaautoprogramming.AutoprogrammingBatchV0{}, storeErrorV0("autoprogramming_batch", "batch persistido invalido")
	}
	return validation.Batch, nil
}

func autoprogrammingBatchCASConflictV0(
	batch orquestaautoprogramming.AutoprogrammingBatchV0,
	expectedVersion uint64,
	actualVersion uint64,
) error {
	return storeErrorV0("autoprogramming_batch.cas", "batch divergente o store_version no coincide")
}
